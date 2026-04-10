// Package manifest provides manifest commands for the CLI.
package manifest

// ValidationIssue represents a manifest validation issue.
type ValidationIssue struct {
	Path     string `json:"path"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	Severity string `json:"severity"`
}

// ValidateResponse represents the response from manifest validation.
type ValidateResponse struct {
	Valid      bool                   `json:"valid"`
	Issues     []ValidationIssue      `json:"issues"`
	Manifest   map[string]interface{} `json:"manifest"`
	Timestamp  string                 `json:"timestamp"`
	SchemaHint string                 `json:"schema_hint,omitempty"`
}

// SchemaResponse represents the response from manifest schema endpoint.
type SchemaResponse struct {
	Schema    map[string]interface{} `json:"schema"`
	Timestamp string                 `json:"timestamp"`
}

// InitRequest is the request for manifest init.
type InitRequest struct {
	ScenarioID string `json:"scenario_id,omitempty"`
	Host       string `json:"host,omitempty"`
	Domain     string `json:"domain,omitempty"`
	User       string `json:"user,omitempty"`
	Port       int    `json:"port,omitempty"`
	KeyPath    string `json:"key_path,omitempty"`
	Workdir    string `json:"workdir,omitempty"`
	CaddyEmail string `json:"caddy_email,omitempty"`
}

// InitResponse represents the response from manifest init.
type InitResponse struct {
	Manifest  map[string]interface{} `json:"manifest"`
	Issues    []ValidationIssue      `json:"issues"`
	Source    string                 `json:"source"`
	Timestamp string                 `json:"timestamp"`
}

// TemplateResponse represents the response from manifest template.
type TemplateResponse struct {
	Variant   string                 `json:"variant"`
	Manifest  map[string]interface{} `json:"manifest"`
	Timestamp string                 `json:"timestamp"`
}

// DoctorRequest wraps a manifest for doctor/fix operations.
type DoctorRequest struct {
	Manifest map[string]interface{} `json:"manifest"`
}

// DoctorResponse is the response for doctor/fix operations.
type DoctorResponse struct {
	Valid      bool                   `json:"valid"`
	Issues     []ValidationIssue      `json:"issues"`
	Manifest   map[string]interface{} `json:"manifest"`
	CanFix     bool                   `json:"can_fix,omitempty"`
	Applied    bool                   `json:"applied,omitempty"`
	Timestamp  string                 `json:"timestamp"`
	SchemaHint string                 `json:"schema_hint,omitempty"`
}
