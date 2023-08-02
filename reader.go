package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/exp/slices"
)

var path = flag.String("p", ".", "path")

var r = regexp.MustCompile(`\[(.*)\]`)

func main() {
	flag.Parse()

	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)

	data, err := loadFiles(path)
	if err != nil {
		return
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	t := S{data: data, limit: 10, files: keys}

	go t.Console()

	// TODO:
	sig := <-cancelChan
	log.Printf("Caught signal %v\n", sig)
}

func loadFiles(path *string) (files map[string]STGPayload, err error) {
	dir, err := os.ReadDir(filepath.Clean(*path))
	if err != nil {
		return nil, err
	}

	files = make(map[string]STGPayload)
	for _, entry := range dir {
		name := entry.Name()
		ext := filepath.Ext(name)
		if entry.Type().IsRegular() && strings.HasPrefix(name, "manual-stg-") && ext == ".json" {
			path := filepath.Clean(*path + "/" + name)
			content, err := ioutil.ReadFile(path)
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
				var allowedGroups []int
				for _, x := range strings.Split(match[1], " ") {
					id, err := strconv.ParseInt(x, 10, 0)
					if err == nil {
						allowedGroups = append(allowedGroups, int(id))
					}
				}

				g := &payload.Groups
				for i := len(*g) - 1; i >= 0; i-- {
					if !slices.Contains(allowedGroups, (*g)[i].ID) {
						(*g) = append((*g)[:i], (*g)[i+1:]...)
					}
				}
			}

			files[path] = payload
		}
	}

	return
}

func save(path string, payload STGPayload) {
	file, _ := json.MarshalIndent(payload, "", "    ")
	_ = ioutil.WriteFile(path, file, 0o644)
}

type S struct {
	data      map[string]STGPayload
	files     []string
	lastCmd   string
	limit     int
	prevFound []Tab
}

func (s *S) Console() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
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
	s.Console()
}

func (s *S) Process(cmd string) (err error) {
	tokens := strings.SplitN(cmd, " ", 2)
	fn := tokens[0]
	limit := int(math.Min(float64(len(s.prevFound)), float64(s.limit)))

	switch fn {
	case "set":
		tokens = strings.SplitN(tokens[1], " ", 2)
		subfn := tokens[0]
		switch subfn {
		case "limit":
			limit, _ := strconv.ParseInt(tokens[1], 10, 0)
			s.limit = int(limit)
		}

	case "find":
		if len(tokens) == 1 {
			return
		}
		search := tokens[1]
		var found []Tab
		for path, payload := range s.data {
			full := false
			for j, g := range payload.Groups {
				var skipped []Tab
				count := 0
				for i, t := range g.Tabs {
					if strings.Contains(t.URL, search) || strings.Contains(t.Title, search) {
						found = append(found, t)
						count++
					} else {
						skipped = append(skipped, t)
					}
					if len(found) >= s.limit {
						full = true
						skipped = append(skipped, g.Tabs[i+1:]...)
						break
					}
				}
				if count > 0 {
					fmt.Printf("Found `%d` tabs in group `%s` [total=%d, skipped=%d]\n", count, g.Title, len(g.Tabs), len(skipped))
					s.data[path].Groups[j].Tabs = skipped
				}
				if full {
					break
				}
			}
			if full {
				break
			}
		}
		s.prevFound = found

	case "open":
		if len(tokens) == 2 {
			if l, err := strconv.ParseInt(tokens[1], 10, 0); err == nil {
				if l > 0 {
					limit = int(l)
				}
			}
		}

		fmt.Printf("cmd=%s size=%d limit=%d\n", cmd, len(s.prevFound), limit)

		i := int(math.Min(float64(len(s.prevFound)), float64(limit))) - 1
		for ; i >= 0; i-- {
			fmt.Println(s.prevFound[i].URL)
			open(s.prevFound[i].URL)
			// TODO: filter after opening
			s.prevFound = append(s.prevFound[:i], s.prevFound[i+1:]...)
		}

	case "list":
		fmt.Printf("cmd=%s size=%d\n", cmd, len(s.prevFound))
		var subfn string
		if len(tokens) > 1 {
			subfn = tokens[1]
		}
		switch subfn {
		case "files":
			for _, x := range s.files {
				fmt.Println(x)
			}
		default:
			for _, x := range s.prevFound {
				fmt.Println(x.URL, x.Title)
			}
		}
	case "save":
		for path, payload := range s.data {
			save(path, payload)
		}
		os.Exit(0)
	case "exit":
		fmt.Println("Exiting program")
		os.Exit(0)
	case "clear":
		fmt.Print("\033[H\033[2J")
	}

	return
}

func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	case "darwin":
		cmd = "open"
	default: // linux freebsd openbsd netbsd
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
