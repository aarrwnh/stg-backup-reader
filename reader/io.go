package reader

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

const filenamePrefix = "manual-stg-"

var groupId = regexp.MustCompile(`\[(.*)\]`)

func LoadFiles(path *string) (files map[Path]Data, count int, err error) {
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
			match := groupId.FindStringSubmatch(name)
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
					if !slices.Contains(allowedGroups, g[i].Id) {
						g.Remove(i)
					}
				}
				*payload.Groups = g
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

type Tab struct {
	Url   string `json:"url"`
	Title string `json:"title"`
	Id    int    `json:"id"`
}

func (t *Tab) Contains(pattern string) bool {
	size := len(pattern)
	pattern = strings.ToLower(pattern)
	url := t.Url[7:] // http://
	if size <= len(url) && strings.Contains(strings.ToLower(url), pattern) {
		return true
	}
	if size <= len(t.Title) && strings.Contains(strings.ToLower(t.Title), pattern) {
		return true
	}
	return false
}

func (t Tab) ToString() string {
	return t.Url + " " + t.Title
}

type STGPayload struct {
	Version string `json:"version"`
	Groups  *Arr[struct {
		Id    int       `json:"id"`
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
