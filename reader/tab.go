package reader

import "strings"

type Tab struct {
	Url   string `json:"url"`
	Title string `json:"title"`
	Id    int    `json:"id"`
}

func (t *Tab) Contains(query *SearchQuery, insensitive bool) bool {
	pattern := query.pattern
	if strings.Contains(t.Url[7:], pattern) { // http://
		return true
	}

	if query.priority == PriorityUrl {
		// skip title search if explictly searched for url
		return false
	}

	title := t.Title
	if insensitive {
		pattern = strings.ToLower(pattern)
		title = strings.ToLower(title)
	}

	return strings.Contains(title, pattern)
}

func (t Tab) ToString() string {
	return t.Url + " " + t.Title
}
