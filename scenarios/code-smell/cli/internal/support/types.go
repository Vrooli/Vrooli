package support

import (
	"encoding/json"
	"time"
)

// Violation mirrors the Violation struct emitted by /api/v1/code-smell/analyze
// and /api/v1/code-smell/queue.
type Violation struct {
	ID           string    `json:"id"`
	RuleID       string    `json:"rule_id"`
	RuleName     string    `json:"rule_name"`
	FilePath     string    `json:"file_path"`
	LineNumber   int       `json:"line_number"`
	ColumnNumber int       `json:"column_number"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	SuggestedFix string    `json:"suggested_fix"`
	AutoFixable  bool      `json:"auto_fixable"`
	Status       string    `json:"status"`
	DetectedAt   time.Time `json:"detected_at"`
}

// AnalyzeResponse is the shape returned by POST /api/v1/code-smell/analyze.
type AnalyzeResponse struct {
	Violations  []Violation `json:"violations"`
	AutoFixed   int         `json:"auto_fixed"`
	NeedsReview int         `json:"needs_review"`
	TotalFiles  int         `json:"total_files"`
	DurationMs  int64       `json:"duration_ms"`
}

// Rule mirrors the Rule struct returned by /api/v1/code-smell/rules.
type Rule struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Category       string          `json:"category"`
	RiskLevel      string          `json:"risk_level"`
	VrooliSpecific bool            `json:"vrooli_specific"`
	Pattern        json.RawMessage `json:"pattern,omitempty"`
	FixTemplate    string          `json:"fix_template,omitempty"`
	Enabled        bool            `json:"enabled"`
}

// RulesResponse wraps the rules listing response.
type RulesResponse struct {
	Rules               []Rule   `json:"rules"`
	Categories          []string `json:"categories"`
	VrooliSpecificCount int      `json:"vrooli_specific_count"`
}

// QueueResponse is the shape returned by GET /api/v1/code-smell/queue.
type QueueResponse struct {
	Violations []Violation    `json:"violations"`
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"by_severity"`
}
