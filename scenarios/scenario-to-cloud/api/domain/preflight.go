// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

// PreflightCheckStatus represents the outcome of a preflight check.
type PreflightCheckStatus string

const (
	PreflightPass PreflightCheckStatus = "pass"
	PreflightWarn PreflightCheckStatus = "warn"
	PreflightFail PreflightCheckStatus = "fail"
)

// PreflightCheck represents a single preflight validation result.
type PreflightCheck struct {
	ID      string               `json:"id"`
	Title   string               `json:"title"`
	Status  PreflightCheckStatus `json:"status"`
	Details string               `json:"details,omitempty"`
	Hint    string               `json:"hint,omitempty"`
	Data    map[string]string    `json:"data,omitempty"`
}

// PreflightResponse contains the results of all preflight checks.
type PreflightResponse struct {
	OK        bool              `json:"ok"`
	Checks    []PreflightCheck  `json:"checks"`
	Issues    []ValidationIssue `json:"issues,omitempty"`
	Timestamp string            `json:"timestamp"`
}
