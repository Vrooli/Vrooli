package jobs

type Opening struct {
	ID          string `json:"id"`
	Company     string `json:"company"`
	Title       string `json:"title"`
	Location    string `json:"location,omitempty"`
	URL         string `json:"url,omitempty"`
	Category    string `json:"category,omitempty"`
	Status      string `json:"status"`
	Source      string `json:"source,omitempty"`
	SourceID    string `json:"source_id,omitempty"`
	Description string `json:"description,omitempty"`
}

type Fact struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Content string   `json:"content"`
	Tags    []string `json:"tags,omitempty"`
}

type Resume struct {
	ID             string   `json:"id"`
	OpeningID      string   `json:"opening_id,omitempty"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	FactIDs        []string `json:"fact_ids,omitempty"`
	ReviewFindings []string `json:"review_findings,omitempty"`
}

type CoverLetter struct {
	ID        string   `json:"id"`
	OpeningID string   `json:"opening_id,omitempty"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	FactIDs   []string `json:"fact_ids,omitempty"`
}

type Source struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URLTemplate string `json:"url_template"`
	Configured  bool   `json:"configured"`
}
type SearchResult struct {
	Source   string    `json:"source"`
	Query    string    `json:"query"`
	URL      string    `json:"url,omitempty"`
	Status   string    `json:"status"`
	Error    string    `json:"error,omitempty"`
	Openings []Opening `json:"openings,omitempty"`
}
type DiscoveryResult struct {
	Results []SearchResult `json:"results"`
	Partial bool           `json:"partial"`
}
