// Package status provides the health check command for the CLI.
package status

// Response represents the health check response from the API.
type Response struct {
	Status    string                 `json:"status"`
	Readiness bool                   `json:"readiness"`
	Service   string                 `json:"service,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}
