package main

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
