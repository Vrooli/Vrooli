package testrules

type Violation struct {
	RuleID         string `json:"rule_id"`
	Type           string `json:"type"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	Description    string `json:"description"`
	File           string `json:"file"`
	FilePath       string `json:"file_path"`
	Line           int    `json:"line"`
	LineNumber     int    `json:"line_number"`
	CodeSnippet    string `json:"code_snippet,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
	Standard       string `json:"standard,omitempty"`
}
