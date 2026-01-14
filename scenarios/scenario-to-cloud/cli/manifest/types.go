// Package manifest provides manifest validation commands for the CLI.
package manifest

// ValidateResponse represents the response from manifest validation.
type ValidateResponse struct {
	Valid     bool                   `json:"valid"`
	Issues    []string               `json:"issues"`
	Manifest  map[string]interface{} `json:"manifest"`
	Timestamp string                 `json:"timestamp"`
}
