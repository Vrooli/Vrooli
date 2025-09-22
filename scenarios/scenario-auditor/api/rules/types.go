package rules

import "time"

// Violation represents a rule violation found during scanning.
type Violation struct {
	ID             string    `json:"id,omitempty"`
	RuleID         string    `json:"rule_id,omitempty"`
	Type           string    `json:"type,omitempty"`
	Severity       string    `json:"severity"`
	Title          string    `json:"title,omitempty"`
	Message        string    `json:"message,omitempty"`
	Description    string    `json:"description,omitempty"`
	File           string    `json:"file,omitempty"`
	FilePath       string    `json:"file_path,omitempty"`
	Line           int       `json:"line,omitempty"`
	LineNumber     int       `json:"line_number,omitempty"`
	CodeSnippet    string    `json:"code_snippet,omitempty"`
	Recommendation string    `json:"recommendation,omitempty"`
	Standard       string    `json:"standard,omitempty"`
	Category       string    `json:"category,omitempty"`
	DiscoveredAt   time.Time `json:"discovered_at,omitempty"`
}

// Rule represents a compliance rule's metadata.
type Rule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Severity    string   `json:"severity"`
	Enabled     bool     `json:"enabled"`
	Standard    string   `json:"standard"`
	Tags        []string `json:"tags,omitempty"`
}

// RuleMetadata is extracted from rule file documentation.
type RuleMetadata struct {
	Rule        string
	Description string
	Reason      string
	Category    string
	Severity    string
	Standard    string
}
