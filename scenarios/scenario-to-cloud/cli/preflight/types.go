// Package preflight provides VPS preflight check commands for the CLI.
package preflight

// Response represents the response from preflight checks.
type Response struct {
	OK        bool     `json:"ok"`
	Checks    []Check  `json:"checks"`
	Issues    []string `json:"issues,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// Check represents a single preflight check result.
type Check struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}
