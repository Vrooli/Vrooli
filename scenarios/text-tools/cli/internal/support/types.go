package support

// Change is one entry in a diff result.
type Change struct {
	Type      string `json:"type"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Content   string `json:"content"`
}

// DiffResponse is the shape of /api/v1/text/diff.
type DiffResponse struct {
	Changes         []Change `json:"changes"`
	SimilarityScore float64  `json:"similarity_score"`
	Summary         string   `json:"summary"`
}

// Match is one hit in a search result.
type Match struct {
	Line    int     `json:"line"`
	Column  int     `json:"column"`
	Length  int     `json:"length"`
	Context string  `json:"context"`
	Score   float64 `json:"score,omitempty"`
}

// SearchResponse is the shape of /api/v1/text/search.
type SearchResponse struct {
	Matches      []Match `json:"matches"`
	TotalMatches int     `json:"total_matches"`
}

// TransformResponse is the shape of /api/v1/text/transform.
type TransformResponse struct {
	Result                 string   `json:"result"`
	TransformationsApplied []string `json:"transformations_applied"`
	Warnings               []string `json:"warnings,omitempty"`
}

// ExtractResponse is the shape of /api/v1/text/extract.
type ExtractResponse struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
}

// Entity is one named entity produced by analyze.
type Entity struct {
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

// Sentiment is the sentiment sub-result of analyze.
type Sentiment struct {
	Score float64 `json:"score"`
	Label string  `json:"label"`
}

// Keyword is one keyword produced by analyze.
type Keyword struct {
	Word  string  `json:"word"`
	Score float64 `json:"score"`
}

// Language is the language detection sub-result of analyze.
type Language struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

// AnalyzeResponse is the shape of /api/v1/text/analyze.
type AnalyzeResponse struct {
	Entities  []Entity  `json:"entities,omitempty"`
	Sentiment Sentiment `json:"sentiment,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	Keywords  []Keyword `json:"keywords,omitempty"`
	Language  Language  `json:"language,omitempty"`
}

// ResourcesStatus is the shape of the root /resources endpoint.
type ResourcesStatus struct {
	Timestamp int64                  `json:"timestamp"`
	Resources map[string]interface{} `json:"resources"`
	Metrics   map[string]interface{} `json:"metrics"`
}
