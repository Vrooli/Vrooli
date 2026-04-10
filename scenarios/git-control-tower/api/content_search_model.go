package main

import "time"

// Content search constants for safety limits
const (
	ContentSearchMinQueryLen     = 2
	ContentSearchMaxQueryLen     = 500
	ContentSearchDefaultLimit    = 100
	ContentSearchMaxLimit        = 500
	ContentSearchMaxPerFile      = 10
	ContentSearchDefaultTimeout  = 10000 // ms
	ContentSearchMaxTimeout      = 30000 // ms
	ContentSearchMaxContextLines = 5
	ContentSearchMaxLineLen      = 500
)

// ContentSearchRequest holds the parsed query parameters for content search.
type ContentSearchRequest struct {
	Query         string
	CaseSensitive bool
	WholeWord     bool
	Regex         bool
	Include       string // Comma-separated globs
	Exclude       string // Comma-separated globs
	ContextLines  int
	Limit         int
	Timeout       int // milliseconds
}

// ContentSearchMatch represents a single match in a file.
type ContentSearchMatch struct {
	Path          string `json:"path"`
	LineNumber    int    `json:"line_number"`
	Content       string `json:"content"`
	ContextBefore string `json:"context_before,omitempty"`
	ContextAfter  string `json:"context_after,omitempty"`
}

// ContentSearchResponse is the API response for content search.
type ContentSearchResponse struct {
	Matches   []ContentSearchMatch `json:"matches"`
	Total     int                  `json:"total"`
	Truncated bool                 `json:"truncated"`
	Cancelled bool                 `json:"cancelled"`
	Query     string               `json:"query"`
	Timestamp time.Time            `json:"timestamp"`
}
