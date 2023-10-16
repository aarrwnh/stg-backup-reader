package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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

const filenamePrefix = "manual-stg-"

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

	t := S{data: data, limit: 10}

	f := func() {
		select {
		case sig := <-cancelChan:
			log.Printf("Caught signal %v\n", sig)
			cancelFunc()
		case <-ctx.Done():
		}
	}

	go t.Console(f)

	// TODO:
	sig := <-cancelChan
	log.Printf("Caught signal %v\n", sig)
}

func loadFiles(path *string) (files map[string]Data, err error) {
	dir, err := os.ReadDir(filepath.Clean(*path))
	if err != nil {
		return nil, err
	}

	files = make(map[string]Data)
	// keys = make(map[string]bool, len(files))
	for _, entry := range dir {
		name := entry.Name()
		ext := filepath.Ext(name)
		if entry.Type().IsRegular() && strings.HasPrefix(name, filenamePrefix) && ext == ".json" {
			path := filepath.Clean(*path + "/" + name)
			content, err := os.ReadFile(path)
			if err != nil {
				log.Fatal("can't read file", err)
			}

			var payload STGPayload
			err = json.Unmarshal(content, &payload)
			if err != nil {
				log.Fatal("Error during Unmarshal()", err)
			}

			// simple filter by group id found inside brackets []
			match := r.FindStringSubmatch(name)
			if len(match) == 2 {
				var allowedGroups Arr[int]
				for _, x := range strings.Split(match[1], " ") {
					id, err := strconv.ParseInt(x, 10, 0)
					if err == nil {
						allowedGroups.Append(int(id))
					}
				}

				g := &payload.Groups
				for i := len(*g) - 1; i >= 0; i-- {
					if !slices.Contains(allowedGroups, (*g)[i].ID) {
						(*g) = append((*g)[:i], (*g)[i+1:]...)
					}
				}
			}

			var flag bool
			files[path] = Data{payload, &flag}
		}
	}

	return
}

func saveFiles(path string, payload STGPayload) error {
	file, _ := json.MarshalIndent(payload, "", "    ")
	return os.WriteFile(path, file, 0o644)
}

func (s *S) Console(f func()) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n> ")
	cmd, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
		return
	}
	err = s.Process(strings.Trim(cmd, "\n\r"))
	if err != nil {
		log.Fatal(err)
		return
	}
	s.Console(f)
}

func (s *S) Process(cmd string) (err error) {
	tok0, tok1, tok2 := tokenize(cmd)
	limit := min(s.size, s.limit)

	var consumed Arr[string]

	switch tok0 {
	case "set":
		switch tok1 {
		case "limit":
			limit, err := strconv.ParseInt(tok2, 10, 0)
			if err == nil {
				s.limit = int(limit)
			}
		}

	case "f":
		fallthrough
	case "find":
		if tok1 == "" {
			return
		}
		var found Arr[Tab]
		search := strings.ToLower(strings.SplitN(cmd, " ", 2)[1])
		for path, data := range s.data {
			for _, g := range data.payload.Groups {
				count := 0
				for _, t := range g.Tabs {
					if strings.Contains(strings.ToLower(t.URL+t.Title), search) {
						found.Append(t)
						fmt.Println(t.URL, t.Title)
						count++
					}
				}
				if count > 0 {
					fmt.Printf(
						"\033[33m[Found `%d` tabs in group `%s` [total=%d path=%s]\033[0m\n",
						count,
						g.Title,
						len(g.Tabs),
						path,
					)
				}
			}
		}
		s.found = found
		s.size = len(found)

	case "o":
		fallthrough
	case "open":
		if s.size == 0 {
			return
		}

		if l, err := strconv.ParseInt(tok1, 10, 0); err == nil {
			limit = int(l)
		}

		max := min(s.size, limit)
		found := &s.found
		for i := 0; i < max; i++ {
			u := (*found)[i].URL
			fmt.Println(u)
			xdg.Open(u)
			consumed.Append(u)
		}
		*found = (*found)[max:]
		s.size = len(*found)
		defer s.RemoveTabs(consumed)

	case "remove":
		if s.size == 0 {
			return
		}

		for _, x := range s.found {
			consumed.Append(x.URL)
		}

		s.found = nil
		s.size = 0
		fmt.Println("Cleaned search list")
		defer s.RemoveTabs(consumed)

	case "s":
		fallthrough
	case "show":
		fallthrough
	case "list":
		switch tok1 {
		case "files":
			for path := range s.data {
				fmt.Println(path)
			}
		default:
			for _, x := range s.found {
				fmt.Println(x.URL, x.Title)
			}
		}
	case "save":
		for path, data := range s.data {
			if *data.modified {
				if err := saveFiles(path, data.payload); err != nil {
					panic(err)
				}
				fmt.Printf("saved: %s\n", path)
				*data.modified = false
			}
		}
	case "exit":
		fmt.Println("Exiting program")
		os.Exit(0)
	case "clear":
		fmt.Print("\033[H\033[2J")
	}

	return
}

func (s *S) RemoveTabs(o []string) {
	if len(o) == 0 {
		return
	}
	for path, data := range s.data {
		for j, group := range data.payload.Groups {
			Tabs := group.Tabs
			for i := len(Tabs) - 1; i >= 0; i-- {
				tab := Tabs[i]
				if slices.Contains(o, tab.URL) {
					popitem(&Tabs, i)
					if !*data.modified {
						*data.modified = true
					}
				}
			}
			data.payload.Groups[j].Tabs = Tabs
		}
		s.data[path] = data
	}
}

type Tab struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	ID    int    `json:"id"`
}

type STGPayload struct {
	Version string `json:"version"`
	Groups  []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Tabs  []Tab  `json:"tabs"`
	} `json:"groups"`
}

type Data struct {
	payload  STGPayload
	modified *bool
}

type S struct {
	data  map[string]Data
	limit int
	found []Tab
	size  int
}
