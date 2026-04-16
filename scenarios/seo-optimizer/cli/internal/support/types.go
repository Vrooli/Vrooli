package support

// AuditRequest mirrors the SEOAuditRequest shape accepted by POST /api/seo-audit.
type AuditRequest struct {
	URL   string `json:"url"`
	Depth int    `json:"depth,omitempty"`
}

// KeywordRequest mirrors POST /api/keyword-research.
type KeywordRequest struct {
	SeedKeyword    string `json:"seed_keyword"`
	TargetLocation string `json:"target_location,omitempty"`
	Language       string `json:"language,omitempty"`
}

// ContentRequest mirrors POST /api/content-optimize.
type ContentRequest struct {
	Content        string `json:"content"`
	TargetKeywords string `json:"target_keywords,omitempty"`
	ContentType    string `json:"content_type,omitempty"`
}

// CompetitorRequest mirrors POST /api/competitor-analysis.
type CompetitorRequest struct {
	YourURL       string `json:"your_url"`
	CompetitorURL string `json:"competitor_url"`
	AnalysisType  string `json:"analysis_type,omitempty"`
}
