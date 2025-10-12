package reader

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/exp/slices"
)

var (
	cmdPrefix = regexp.MustCompile("^[;:]")
	xdg       = NewOpener()
)

type App struct {
	cancel context.CancelFunc

	data        map[Path]Data
	found       []Tab
	size        int
	prevQuery   string
	saved       bool
	wsConnected bool

	// Remember opened tabs for batch removal on save
	consumed Arr[string]
	// How many tabs to open in browser at one time
	limit uint8
	// Tab count removed from payload but yet to be saved
	removePending int

	// TODO: temp?
	debugLevel uint8
}

func NewApp(data map[Path]Data, limit uint8, cancel context.CancelFunc) App {
	return App{
		data:   data,
		limit:  limit,
		cancel: cancel,
	}
}

var path = flag.String("p", ".", "path")

func Start() {
	flag.Parse()

	data, count, err := LoadFiles(path)
	if err != nil {
		return
	}
	printInfo("loaded %d tabs", count)

	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(interrupt)

	app := NewApp(data, 10, cancel)

	go StartWebsocket(&app)
	go app.run()

	select {
	case <-ctx.Done():
		printInfo("Exiting program")
	case sig := <-interrupt:
		printInfo("Caught signal: %v", sig)
	}

	time.Sleep(time.Millisecond * 100)
}

func (s *App) run() {
	reader := bufio.NewReader(os.Stdin)
	for {
		s.UpdateTitle()
		fmt.Print("\n> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println()
			printInfo(fmt.Sprint(err))
			s.Quit()
			return
		}

		if err := s.Process(strings.Trim(input, "\n\r")); err != nil {
			return
		}
	}
}

func (s *App) Quit() error {
	s.cancel()
	return errors.New("Exiting program")
}

func (s *App) Process(input string) (err error) {
	cmd, subcmd, rest := commandParse(input)
	query := subcmd + " " + rest

	if cmdPrefix.MatchString(cmd) {
		switch string(cmdPrefix.ReplaceAll([]byte(cmd), []byte(""))) {
		case "set":
			s.Set(subcmd, rest)
		case "f", "find":
			s.FindTabs(query, true)
		case "filter":
			s.filterTabs(query, true)
		case "o", "open":
			s.OpenTabs(subcmd)
		case "remove", "rm", "rem":
			s.ForceRemove()
		case "show", "list", "ls":
			s.ShowCurrent(subcmd)
		case "s", "save":
			s.writeTabs()
		case "q", "quit", "exit":
			err = s.Quit()
		case "c", "clear":
			fmt.Print("\033[H\033[2J")
		}
	} else {
		s.FindTabs(input, true)
	}

	return
}

func (s *App) Set(token1, token2 string) {
	switch token1 {
	case "limit":
		if num, err := strconv.ParseInt(token2, 10, 8); err == nil {
			printInfo("OpenLimit %d => %d", s.limit, num)
			s.limit = uint8(num)
		}
	case "debug":
		if num, err := strconv.ParseInt(token2, 10, 8); err == nil {
			printInfo("DebugLevel %d => %d", s.debugLevel, num)
			s.debugLevel = uint8(num)
		}
	}
}

func (s *App) FindTabs(query string, printLines bool) {
	if len(query) <= 1 {
		return
	}

	defer timeTrack(time.Now())

	s.prevQuery = query

	var found Arr[Tab]
	query = strings.ToLower(query)
	for path, data := range s.data {
		for _, g := range *data.payload.Groups {
			count := 0
			for _, t := range *g.Tabs {
				if t.Contains(query) {
					found.Append(t)
					if printLines {
						fmt.Println(highlightWord(query, t.ToString()))
					}
					count++
				}
			}
			if count > 0 && s.debugLevel != 0 {
				printInfo(
					"found `%d` tabs in group `%s` | %d | %s",
					count,
					g.Title,
					len(*g.Tabs),
					path.name,
				)
			}
		}
	}
	s.found = found
	s.size = found.Length()
	printInfo("found %d tabs", s.size)
}

// Perform further search on found query
func (s *App) filterTabs(query string, printLines bool) {
	if s.size == 0 {
		return
	}
	var found Arr[Tab]
	query = strings.ToLower(query)
	for _, t := range s.found {
		if t.Contains(query) {
			found.Append(t)
			if printLines {
				fmt.Println(highlightWord(query, t.ToString()))
			}
		}
	}
	s.found = found
	s.size = found.Length()
	printInfo("found %d tabs", s.size)
}

func (s *App) OpenTabs(token string) {
	if s.size == 0 {
		return
	}

	limit := min(s.size, int(s.limit))
	if l, err := strconv.ParseInt(token, 10, 0); err == nil {
		limit = int(l)
	}

	if limit > 40 {
		// limit just in case
		return
	}

	_max := min(s.size, limit)
	for i := 0; i < _max; i++ {
		u := s.found[i]
		fmt.Println(u.ToString())
		xdg.Open(u.URL)
		s.consumed.Append(u.URL)
	}

	s.found = s.found[_max:]
	s.size = len(s.found)
	s.prevQuery = ""
	s.RemoveTabs()
}

// Use when removing without opening
func (s *App) ForceRemove() {
	if s.size == 0 {
		return
	}

	for _, x := range s.found {
		s.consumed.Append(x.URL)
	}

	s.found = nil
	s.size = 0
	s.prevQuery = ""
	s.RemoveTabs()
}

func (s *App) ShowCurrent(cmd string) {
	switch cmd {
	case "files":
		var total int
		for path := range s.data {
			var entries int
			g := *s.data[path].payload.Groups
			for i := range g {
				entries += len(*g[i].Tabs)
			}
			fmt.Printf("%7d  %s\n", entries, path.name)
			total += entries
		}
		fmt.Printf("%7d\n", total)
	default:
		for _, t := range s.found {
			fmt.Println(highlightWord(s.prevQuery, t.ToString()))
		}
		printInfo("found %d tabs", s.size)
	}
}

// Write changes into files
func (s *App) writeTabs() {
	for path, data := range s.data {
		if *data.modified {
			if err := saveFiles(path.path, data.payload); err != nil {
				panic(err)
			}
			fmt.Printf("done: %s\n", path.name)
			*data.modified = false
		}
	}
	s.saved = true
	s.removePending = 0
}

// Remove currently found tabs from groups.
func (s *App) RemoveTabs() {
	if len(s.consumed) == 0 {
		return
	}
	defer timeTrack(time.Now())

	removed := 0
	for _, data := range s.data {
		for _, group := range *data.payload.Groups {
			tabs := group.Tabs
			for idx := len(*tabs) - 1; idx >= 0; idx-- {
				if slices.Contains(s.consumed, (*tabs)[idx].URL) {
					tabs.Remove(idx)
					removed += 1
					if !*data.modified {
						*data.modified = true
					}
				}
			}
		}
	}

	s.removePending += removed
	s.consumed.Clear()
	s.saved = false

	printInfo("removed %d item/s", removed)
}

func (s *App) UpdateTitle() {
	var a string
	if s.wsConnected {
		a = " | *"
	}
	setTitle(fmt.Sprintf("f:%d | rem:%d%s", s.size, s.removePending, a))
}
