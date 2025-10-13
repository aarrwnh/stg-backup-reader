package reader

import (
	"encoding/json"
	"io/fs"
	"iter"
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

func filterFiles[T fs.DirEntry](dir []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, entry := range dir {
			if !entry.Type().IsRegular() {
				continue
			}

			name := entry.Name()
			ext := filepath.Ext(name)
			if ext[1:] != "json" {
				continue
			}

			if !strings.HasPrefix(name, filenamePrefix) {
				continue
			}

			if !yield(entry) {
				return
			}
		}
	}
}

func loadFiles(path *string) (files map[Path]Data, count int, err error) {
	dir, err := os.ReadDir(filepath.Clean(*path))
	if err != nil {
		return nil, count, err
	}

	files = make(map[Path]Data)

	for entry := range filterFiles(dir) {
		name := entry.Name()
		path := filepath.Clean(*path + "/" + name)
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("can't read file", err)
		}

		var payload STGPayload
		if err := json.Unmarshal(content, &payload); err != nil {
			log.Fatal(err)
		}

		filterGroups(&name, &payload)

		files[Path{path, name}] = Data{payload, new(bool)}

		for _, v := range *payload.Groups {
			count += len(*v.Tabs)
		}
	}

	return
}

// Simple filter by group id found inside brackets in the filename:
// manual-stg-backup-2025-08-19@drive4ik[417].json
func filterGroups(name *string, payload *STGPayload) {
	match := groupId.FindStringSubmatch(*name)
	if len(match) != 2 {
		return
	}

	var allowedGroups Arr[int]
	for _, x := range strings.Split(match[1], " ") {
		if id, err := strconv.ParseInt(x, 10, 0); err == nil {
			allowedGroups.Append(int(id))
		}
	}

	payload.Groups.Filter(func(item Groups) bool {
		return !slices.Contains(allowedGroups, item.Id)
	})
}

func saveFiles(path string, payload STGPayload) error {
	file, _ := json.MarshalIndent(payload, "", "    ")
	return os.WriteFile(path, file, 0o644)
}

type Groups struct {
	Id    int       `json:"id"`
	Title string    `json:"title"`
	Tabs  *Arr[Tab] `json:"tabs"`
}

type STGPayload struct {
	Version string       `json:"version"`
	Groups  *Arr[Groups] `json:"groups"`
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
