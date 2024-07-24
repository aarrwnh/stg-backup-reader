package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

const filenamePrefix = "manual-stg-"

func loadFiles(path *string) (files map[Path]Data, count int, err error) {
	dir, err := os.ReadDir(filepath.Clean(*path))
	if err != nil {
		return nil, 0, err
	}

	files = make(map[Path]Data)
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
				log.Fatal(err)
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

				g := *payload.Groups
				for i := len(g) - 1; i >= 0; i-- {
					if !slices.Contains(allowedGroups, g[i].ID) {
						g.Remove(i)
					}
				}
			}

			var flag bool
			files[Path{path, name}] = Data{payload, &flag}

			for _, v := range *payload.Groups {
				count += len(*v.Tabs)
			}
		}
	}

	return
}

func saveFiles(path string, payload STGPayload) error {
	file, _ := json.MarshalIndent(payload, "", "    ")
	return os.WriteFile(path, file, 0o644)
}

type Url string

type Tab struct {
	URL   Url    `json:"url"`
	Title string `json:"title"`
	ID    int    `json:"id"`
}

func (t *Tab) Contains(query string) bool {
	return strings.Contains(strings.ToLower(string(t.URL)+t.Title), query)
}

type STGPayload struct {
	Version string `json:"version"`
	Groups  *Arr[struct {
		ID    int       `json:"id"`
		Title string    `json:"title"`
		Tabs  *Arr[Tab] `json:"tabs"`
	}] `json:"groups"`
}

type Data struct {
	payload STGPayload
	// marker indicating if a file needs to be flushed to disk
	modified *bool
}

type Path struct {
	path string
	name string
}

type App struct {
	data  map[Path]Data
	limit int
	found []Tab
	size  int

	totalRemoved int

	consumed Arr[Url]
}
