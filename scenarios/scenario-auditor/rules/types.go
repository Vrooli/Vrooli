package rules

import "time"

// Violation represents a rule violation found during scanning
type Violation struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	Severity       string    `json:"severity"` // critical, high, medium, low
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	FilePath       string    `json:"file_path"`
	LineNumber     int       `json:"line_number"`
	CodeSnippet    string    `json:"code_snippet,omitempty"`
	Recommendation string    `json:"recommendation"`
	Standard       string    `json:"standard"`
	Category       string    `json:"category"`
	DiscoveredAt   time.Time `json:"discovered_at"`
}

// Rule represents a compliance rule
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

// RuleMetadata is extracted from rule file documentation
type RuleMetadata struct {
	Rule        string
	Description string
	Reason      string
	Category    string
	Severity    string
	Standard    string
}