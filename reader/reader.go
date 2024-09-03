package reader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

var (
	cmdPrefix = regexp.MustCompile("^[;:]")
	xdg       = NewOpener()
)

type App struct {
	data         map[Path]Data
	limit        int
	found        []Tab
	size         int
	totalRemoved int
	consumed     Arr[string]
	cancel       context.CancelFunc
	wsConnected  bool
}

func NewApp(data map[Path]Data, limit int, cancel context.CancelFunc) App {
	return App{
		data:   data,
		limit:  limit,
		cancel: cancel,
	}
}

func (s *App) Start() {
	s.ConsoleTick()
}

func (s *App) ConsoleTick() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n> ")

	s.UpdateTitle()

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println()
		log.Println(err)
		s.Quit()
		return
	}

	if err := s.Process(strings.Trim(input, "\n\r")); err != nil {
		return
	}

	s.ConsoleTick()
}

func (s *App) Quit() error {
	log.Printf("Removed %d tabs during session", s.totalRemoved)
	s.cancel()
	return errors.New("Exiting program")
}

func (s *App) Process(input string) (err error) {
	cmd, subcmd, rest := commandParse(input)

	if cmdPrefix.MatchString(cmd) {
		switch string(cmdPrefix.ReplaceAll([]byte(cmd), []byte(""))) {
		case "set":
			s.Set(subcmd, rest)
		case "f", "find":
			s.FindTabs(strings.SplitN(input, " ", 2)[1], true)
		case "o", "open":
			s.OpenTabs(subcmd)
		case "remove", "rm", "rem":
			s.ForceRemove()
		case "show", "list", "ls":
			s.ShowCurrent(subcmd)
		case "s", "save":
			s.SaveTabs()
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
		if limit, err := strconv.ParseInt(token2, 10, 0); err == nil {
			s.limit = int(limit)
		}
	}
}

func (s *App) FindTabs(query string, printLines bool) {
	if len(query) <= 1 {
		return
	}

	defer timeTrack(time.Now())

	var found Arr[Tab]
	query = strings.ToLower(query)
	for path, data := range s.data {
		for _, g := range *data.payload.Groups {
			count := 0
			for _, t := range *g.Tabs {
				if t.Contains(query) {
					found.Append(t)
					if printLines {
						line := highlightWord(query, string(t.URL)+" "+t.Title)
						fmt.Println(line)
					}
					count++
				}
			}
			if count > 0 {
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
}

func (s *App) OpenTabs(token string) {
	if s.size == 0 {
		return
	}

	limit := min(s.size, s.limit)
	if l, err := strconv.ParseInt(token, 10, 0); err == nil {
		limit = int(l)
	}

	_max := min(s.size, limit)
	for i := 0; i < _max; i++ {
		u := s.found[i].URL
		fmt.Println(u)
		xdg.Open(u)
		s.consumed.Append(u)
	}

	s.found = s.found[_max:]
	s.size = len(s.found)
	s.RemoveTabs()
}

func (s *App) ForceRemove() {
	if s.size == 0 {
		return
	}

	for _, x := range s.found {
		s.consumed.Append(x.URL)
	}

	s.found = nil
	s.size = 0
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
		for _, x := range s.found {
			fmt.Println(x.URL, x.Title)
		}
	}
}

func (s *App) SaveTabs() {
	for path, data := range s.data {
		if *data.modified {
			if err := saveFiles(path.path, data.payload); err != nil {
				panic(err)
			}
			fmt.Printf("done: %s\n", path.name)
			*data.modified = false
		}
	}
	s.totalRemoved = 0
}

// Remove currently found tabs from groups.
func (s *App) RemoveTabs() {
	if len(s.consumed) == 0 {
		return
	}
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

	s.totalRemoved += removed
	s.consumed.Clear()

	printInfo("removed %d item/s", removed)
}

func (s *App) UpdateTitle() {
	var a string
	if s.wsConnected {
		a = " | *"
	}
	setTitle(fmt.Sprintf("f:%d | rem:%d%s", s.size, s.totalRemoved, a))
}
