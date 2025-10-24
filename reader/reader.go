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
	cmdPrefix  = regexp.MustCompile("^[;:]")
	httpPrefix = regexp.MustCompile("^https?://")
	xdg        = newOpener()
)

type App struct {
	cancel        context.CancelFunc
	data          map[Path]Data
	found         Arr[Tab]
	size          int
	prevQuery     string
	saved         bool
	wsConnected   bool
	insensitive   bool        // Regex "i" flag. Perform case-insensitive search.
	consumed      Arr[string] // Cache tabs for batch removal on save.
	limit         uint8       // How many tabs to open in browser at one time
	removePending int         // Tab count removed from payload but yet to be saved
	debugLevel    uint8       // TODO: temp?
}

func newApp(data map[Path]Data, limit uint8, cancel context.CancelFunc) App {
	return App{
		data:        data,
		limit:       limit,
		cancel:      cancel,
		insensitive: true,
	}
}

var path = flag.String("p", ".", "path")

func Start() {
	flag.Parse()

	data, count, err := loadFiles(path)
	if err != nil {
		return
	}
	printInfo("loaded %d tabs", count)

	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(interrupt)

	app := newApp(data, 10, cancel)

	go startWebsocket(&app)
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
			s.quit()
			return
		}

		if err := s.Process(strings.Trim(input, "\n\r")); err != nil {
			return
		}
	}
}

func (s *App) quit() error {
	s.cancel()
	return errors.New("Exiting program")
}

func (s *App) Process(input string) (err error) {
	cmd, subcmd, rest := commandParse(input)
	query := strings.TrimSpace(subcmd + " " + rest)

	if cmdPrefix.MatchString(cmd) {
		switch string(cmdPrefix.ReplaceAll([]byte(cmd), []byte(""))) {
		case "set":
			s.set(subcmd, rest)
		case "f", "find":
			s.findTabs(newQuery(query))
		case "findurl":
			s.findTabs(newQuery(query).withPriorityUrl())
		case "filter":
			s.filterTabs(newQuery(query))
		case "o", "open":
			s.openTabs(subcmd)
		case "remove", "rm", "rem":
			s.forceRemove()
		case "show", "list", "ls":
			s.showCurrent(subcmd)
		case "s", "save":
			s.writeTabs()
		case "q", "quit", "exit":
			err = s.quit()
		case "c", "clear":
			fmt.Print("\033[H\033[2J")
		}
	} else {
		s.findTabs(newQuery(input))
	}
	return
}

func (s *App) set(token1, token2 string) {
	switch token1 {
	case "insensitive", "i":
		if b, err := strconv.ParseBool(token2); err == nil {
			printInfo("SearchInsensitive %v => %v", s.insensitive, b)
			s.insensitive = b
		}

	case "limit", "l":
		if num, err := strconv.ParseInt(token2, 10, 8); err == nil {
			printInfo("OpenLimit %d => %d", s.limit, num)
			s.limit = uint8(num)
		}
	case "debug", "d":
		if num, err := strconv.ParseInt(token2, 10, 8); err == nil {
			printInfo("DebugLevel %d => %d", s.debugLevel, num)
			s.debugLevel = uint8(num)
		}
	}
}

func stripProtocol(s string) string {
	return string(httpPrefix.ReplaceAll([]byte(s), []byte("")))
}

func (s *App) findTabs(query *SearchQuery) {
	if len(query.pattern) <= 1 {
		return
	}

	defer timeTrack(time.Now())

	s.prevQuery = stripProtocol(query.pattern)

	var found Arr[Tab]
	for path, data := range s.data {
		for _, g := range *data.payload.Groups {
			count, found0 := search(g.Tabs, query, s.insensitive)
			found.Append(found0...)
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
	if s.size > 0 {
		printInfo("found %d tabs", s.size)
	}
}

// Perform further search on found query
func (s *App) filterTabs(query *SearchQuery) {
	if s.size == 0 {
		return
	}
	defer timeTrack(time.Now())
	size, found := search(&s.found, query, true)
	if size > 0 {
		s.found = found
		s.size = size
	}
	printInfo("found %d tabs", size)
}

func (s *App) openTabs(token string) {
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
		xdg.Open(u.Url)
		s.consumed.Append(u.Url)
	}

	s.found = s.found[_max:]
	s.size = len(s.found)
	s.prevQuery = ""
	s.RemoveTabs()
}

// Use when removing without opening
func (s *App) forceRemove() {
	if s.size == 0 {
		return
	}

	for _, x := range s.found {
		s.consumed.Append(x.Url)
	}

	s.found = nil
	s.size = 0
	s.prevQuery = ""
	s.RemoveTabs()
}

func (s *App) showCurrent(cmd string) {
	switch cmd {
	case "files":
		var total int
		for path := range s.data {
			entries := 0
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
		j := 0
		for _, group := range *data.payload.Groups {
			group.Tabs.Filter(func(item Tab) bool {
				if slices.Contains(s.consumed, item.Url) {
					j += 1
					return true
				}
				return false
			})
		}
		if j > 0 {
			*data.modified = true
			removed += j
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
