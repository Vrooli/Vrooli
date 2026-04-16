package support

import "time"

// Scan mirrors one entry returned by GET /api/scans.
type Scan struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Status     string    `json:"status"`
	Created    time.Time `json:"created"`
	Violations int       `json:"violations"`
}

// Violation mirrors one entry returned by GET /api/violations and as nested
// items in report responses.
type Violation struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Element     string `json:"element"`
}

// Report mirrors one entry returned by GET /api/reports.
type Report struct {
	ID     string      `json:"id"`
	ScanID string      `json:"scan_id"`
	Title  string      `json:"title"`
	Score  float64     `json:"score"`
	Issues []Violation `json:"issues"`
	Date   time.Time   `json:"date"`
}
