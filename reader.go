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

	data, count, err := loadFiles(path)
	if err != nil {
		return
	}

	log.Printf("\033[30mloaded %d tabs\033[0m", count)

	t := App{data: data, limit: 10}

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

	sig := <-cancelChan
	log.Printf("Caught signal: %v\n", sig)
}

func (s *App) Console(f func()) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n> ")

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println()
		s.Quit()
		log.Fatal(err)
	}

	err = s.Process(strings.Trim(input, "\n\r"))
	if err != nil {
		log.Fatal(err)
	}

	f()
	go s.Console(f)
}

func (s *App) Quit() {
	log.Printf("removed %d tabs", s.totalRemoved)
	log.Println("Exiting program")
}

func (s *App) Process(input string) (err error) {
	cmd, subcmd, rest := tokenize(input)

	if strings.HasPrefix(cmd, ":") {
		switch strings.Replace(cmd, ":", "", 1) {
		case "set":
			s.Set(subcmd, rest)
		case "f", "find":
			s.FindTabs(strings.SplitN(input, " ", 2)[1])
		case "o", "open":
			s.OpenTabs(subcmd)
		case "remove", "rm":
			s.ForceRemove()
		case "show", "list":
			s.ShowCurrent(subcmd)
		case "s", "save":
			s.SaveTabs()
		case "q", "quit", "exit":
			s.Quit()
			os.Exit(0)
		case "c", "clear":
			fmt.Print("\033[H\033[2J")
		}
	} else {
		s.FindTabs(input)
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

func highlightWord(pattern, line string) string {
	start := strings.Index(strings.ToLower(line), strings.ToLower(pattern))
	pattern = line[start : start+len(pattern)]
	parts := strings.Split(line, pattern)
	return strings.Join(parts, "\033[34m"+pattern+"\033[0m")
}

func (s *App) FindTabs(query string) {
	if len(query) <= 1 {
		return
	}

	var found Arr[Tab]
	query = strings.ToLower(query)
	for path, data := range s.data {
		for _, g := range *data.payload.Groups {
			count := 0
			for _, t := range *g.Tabs {
				if t.Contains(query) {
					found.Append(t)
					line := highlightWord(query, string(t.URL)+" "+t.Title)
					fmt.Println(line)
					count++
				}
			}
			if count > 0 {
				fmt.Printf(
					"\033[33m[Found `%d` tabs in group `%s` [total=%d file=%s]\033[0m\n",
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
	defer s.RemoveTabs()
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
	fmt.Println("Cleaned search list")
	defer s.RemoveTabs()
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
			fmt.Printf("saved: %s\n", path.name)
			*data.modified = false
		}
	}
}

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
}
