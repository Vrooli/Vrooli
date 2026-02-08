// Package scenario provides scenario discovery commands for the CLI.
package scenario

import "time"

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
	Service  string `json:"service"` // api, ui, cli, resource name
	Port     int    `json:"port"`
	Protocol string `json:"protocol,omitempty"` // tcp, udp
	Public   bool   `json:"public"`             // Exposed externally
	Path     string `json:"path,omitempty"`     // URL path if proxied
}

// DepsResponse represents the response from listing scenario dependencies.
type DepsResponse struct {
	ScenarioID        string   `json:"scenario_id"`
	Resources         []string `json:"resources"`
	Scenarios         []string `json:"scenarios"`
	AnalyzerAvailable bool     `json:"analyzer_available"`
	Source            string   `json:"source"`
	Timestamp         string   `json:"timestamp"`
}

// DeploymentReportResponse is a subset of scenario-dependency-analyzer deployment report.
type DeploymentReportResponse struct {
	Scenario     string                         `json:"scenario"`
	GeneratedAt  time.Time                      `json:"generated_at"`
	Dependencies []DeploymentDependencyNode     `json:"dependencies"`
	Aggregates   map[string]DeploymentAggregate `json:"aggregates"`
	MetadataGaps *DeploymentMetadataGaps        `json:"metadata_gaps,omitempty"`
}

type DeploymentDependencyNode struct {
	Name         string                     `json:"name"`
	Type         string                     `json:"type"`
	ResourceType string                     `json:"resource_type,omitempty"`
	Required     *bool                      `json:"required,omitempty"`
	Enabled      *bool                      `json:"enabled,omitempty"`
	Requirements *DeploymentRequirements    `json:"requirements,omitempty"`
	TierSupport  map[string]TierSupport     `json:"tier_support,omitempty"`
	Children     []DeploymentDependencyNode `json:"children,omitempty"`
}

type TierSupport struct {
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
}

type DeploymentRequirements struct {
	RAMMB    float64 `json:"ram_mb,omitempty"`
	DiskMB   float64 `json:"disk_mb,omitempty"`
	CPUCores float64 `json:"cpu_cores,omitempty"`
}

type DeploymentAggregate struct {
	EstimatedRequirements DeploymentRequirements `json:"estimated_requirements"`
}

type DeploymentMetadataGaps struct {
	TotalGaps int `json:"total_gaps"`
}
