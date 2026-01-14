// Package scenario provides scenario discovery commands for the CLI.
package scenario

// ListResponse represents the response from listing scenarios.
type ListResponse struct {
	Scenarios []ScenarioInfo `json:"scenarios"`
	Timestamp string         `json:"timestamp"`
}

// ScenarioInfo represents information about a scenario.
type ScenarioInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Path        string   `json:"path,omitempty"`
	Resources   []string `json:"resources,omitempty"`
	HasAPI      bool     `json:"has_api"`
	HasUI       bool     `json:"has_ui"`
	HasCLI      bool     `json:"has_cli"`
}

// PortsResponse represents the response from listing scenario ports.
type PortsResponse struct {
	ScenarioID string     `json:"scenario_id"`
	Ports      []PortInfo `json:"ports"`
	Timestamp  string     `json:"timestamp"`
}

// PortInfo represents a port allocation.
type PortInfo struct {
	Service  string `json:"service"`  // api, ui, cli, resource name
	Port     int    `json:"port"`
	Protocol string `json:"protocol,omitempty"` // tcp, udp
	Public   bool   `json:"public"`             // Exposed externally
	Path     string `json:"path,omitempty"`     // URL path if proxied
}

// DepsResponse represents the response from listing scenario dependencies.
type DepsResponse struct {
	ScenarioID   string           `json:"scenario_id"`
	Dependencies []DependencyInfo `json:"dependencies"`
	Timestamp    string           `json:"timestamp"`
}

// DependencyInfo represents a scenario dependency.
type DependencyInfo struct {
	Type     string `json:"type"`     // resource, scenario
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Required bool   `json:"required"`
	Status   string `json:"status,omitempty"` // available, missing, version_mismatch
	Message  string `json:"message,omitempty"`
}
