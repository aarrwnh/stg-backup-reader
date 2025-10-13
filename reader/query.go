package reader

import (
	"cmp"
	"fmt"
)

type PriorityKind int

const (
	PriorityNone PriorityKind = iota
	PriorityUrl
)

type SearchQuery struct {
	pattern  string
	print    bool // whether to print found items in the console
	priority PriorityKind
}

func newQuery(pat string) *SearchQuery {
	return &SearchQuery{pattern: pat, print: true, priority: PriorityNone}
}

func (p *SearchQuery) setPrint(v bool) *SearchQuery {
	p.print = v
	return p
}

func (p *SearchQuery) withPriorityUrl() *SearchQuery {
	p.priority = PriorityUrl
	return p
}

func search(tabs *Arr[Tab], query *SearchQuery, insensitive bool) (int, Arr[Tab]) {
	var found Arr[Tab]
	for _, t := range *tabs {
		if t.Contains(query, insensitive) {
			found.Append(t)
		}
	}

	if query.print && len(found) > 0 {
		// TODO there's probably no need to sort because we're just printing for each file
		found.Sort(func(a, b Tab) int {
			return -cmp.Compare(a.Url, b.Url)
		})
		for _, t := range found {
			fmt.Println(highlightWord(query.pattern, t.ToString()))
		}
	}
	return len(found), found
}
