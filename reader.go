package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/exp/slices"
)

var (
	path = flag.String("p", ".", "path")

	r   = regexp.MustCompile(`\[(.*)\]`)
	xdg = NewOpener()
)

func init() {
	flag.Parse()
}

func main() {
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)

	data, err := loadFiles(path)
	if err != nil {
		return
	}

	t := Files{data: data, limit: 10}

	f := func() {
		select {
		case sig := <-cancelChan:
			log.Printf("\nCaught signal %v\n", sig)
			cancelFunc()
		case <-ctx.Done():
		default:
		}
	}

	go t.Console(f)

	// TODO:
	sig := <-cancelChan
	log.Printf("                       Caught signal %v\n", sig)
}

func (s *Files) Console(f func()) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n> ")
	cmd, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("no command: ", err)
		return
	}
	err = s.Process(strings.Trim(cmd, "\n\r"))
	if err != nil {
		log.Fatal(err)
		return
	}
	f()
	s.Console(f)
}

func (s *Files) Process(cmd string) (err error) {
	tok0, tok1, tok2 := tokenize(cmd)
	limit := min(s.size, s.limit)

	switch tok0 {
	case "set":
		s.Set(&tok1, &tok2)
	case "f", "find":
		s.FindTabs(&tok1, cmd)
	case "o", "open":
		s.OpenTabs(&tok1, limit)
	case "remove":
		s.ForceRemove()
	case "s", "show", "list":
		s.ShowCurrent(&tok1)
	case "save":
		s.SaveTabs()
	case "x", "exit":
		fmt.Println("Exiting program")
		os.Exit(0)
	case "c", "clear":
		fmt.Print("\033[H\033[2J")
	}

	return
}

func (s *Files) Set(token1, token2 *string) {
	switch *token1 {
	case "limit":
		limit, err := strconv.ParseInt(*token2, 10, 0)
		if err == nil {
			s.limit = int(limit)
		}
	}
}

func (s *Files) FindTabs(token *string, cmd string) {
	if *token == "" {
		return
	}
	var found Arr[Tab]
	query := strings.ToLower(strings.SplitN(cmd, " ", 2)[1])
	for path, data := range s.data {
		for _, g := range data.payload.Groups {
			count := 0
			for _, t := range g.Tabs {
				if t.Contains(query) {
					found.Append(t)
					fmt.Println(t.URL, t.Title)
					count++
				}
			}
			if count > 0 {
				fmt.Printf(
					"\033[33m[Found `%d` tabs in group `%s` [total=%d file=%s]\033[0m\n",
					count,
					g.Title,
					len(g.Tabs),
					path.name,
				)
			}
		}
	}
	s.found = found
	s.size = len(found)
}

func (s *Files) OpenTabs(token *string, limit int) {
	if s.size == 0 {
		return
	}

	if l, err := strconv.ParseInt(*token, 10, 0); err == nil {
		limit = int(l)
	}

	_max := min(s.size, limit)
	found := &s.found
	for i := 0; i < _max; i++ {
		u := (*found)[i].URL
		fmt.Println(u)
		xdg.Open(u)
		s.consumed.Append(u)
	}
	*found = (*found)[_max:]
	s.size = len(*found)
	defer s.RemoveTabs()
}

func (s *Files) ForceRemove() {
	if s.size == 0 {
		return
	}

	for _, x := range s.found {
		s.consumed.Append(x.URL)
	}

	s.found = nil
	s.size = 0
	fmt.Println("Cleaned search list")
	defer s.RemoveTabs()
}

func (s *Files) ShowCurrent(token *string) {
	switch *token {
	case "files":
		var total int
		for path := range s.data {
			var entries int
			g := s.data[path].payload.Groups
			for i := range g {
				entries += len(g[i].Tabs)
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

func (s *Files) SaveTabs() {
	for path, data := range s.data {
		if *data.modified {
			if err := saveFiles(path.path, data.payload); err != nil {
				panic(err)
			}
			fmt.Printf("saved: %s\n", path.name)
			*data.modified = false
		}
	}
}

func (s *Files) RemoveTabs() {
	if len(s.consumed) == 0 {
		return
	}
	for _, data := range s.data {
		for j, group := range data.payload.Groups {
			tabs := group.Tabs
			for idx := len(tabs) - 1; idx >= 0; idx-- {
				tab := tabs[idx]
				if slices.Contains(s.consumed, tab.URL) {
					tabs.Remove(idx)
					if !*data.modified {
						*data.modified = true
					}
				}
			}
			data.payload.Groups[j].Tabs = tabs
		}
	}
	s.consumed.Clear()
}
