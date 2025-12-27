// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains the health check endpoint with dependency status reporting.
//
// Health Design Principles:
//   - Status reflects the service's ability to serve requests
//   - Dependencies are classified as core (required) or optional
//   - Degraded status indicates reduced functionality, not failure
//   - Capabilities surface what features are currently available
//   - Metrics provide insight into service behavior
package handlers


// ServiceStatus represents the overall health state.
type ServiceStatus string

const (
	// StatusHealthy means all core dependencies are working.
	StatusHealthy ServiceStatus = "healthy"

	// StatusDegraded means optional dependencies are unavailable.
	// The service can still handle requests but with reduced functionality.
	StatusDegraded ServiceStatus = "degraded"

	// StatusUnhealthy means core dependencies are failing.
	// The service cannot reliably handle requests.
	StatusUnhealthy ServiceStatus = "unhealthy"
)

// DependencyStatus represents the state of a single dependency.
type DependencyStatus string

const (
	// DepConnected means the dependency is available and responding.
	DepConnected DependencyStatus = "connected"

	// DepDisconnected means the dependency is not available.
	DepDisconnected DependencyStatus = "disconnected"

	// DepDegraded means the dependency is available but slow or partially working.
	DepDegraded DependencyStatus = "degraded"

	// DepUnknown means the dependency status could not be determined.
	DepUnknown DependencyStatus = "unknown"

	// DepNotConfigured means the dependency is not enabled.
	DepNotConfigured DependencyStatus = "not_configured"
)

// HealthResponse is the structured health check response.
type HealthResponse struct {
	// Status is the overall service health.
	Status ServiceStatus `json:"status"`

	// Service identifies this service.
	Service string `json:"service"`

	// Version is the service version.
	Version string `json:"version"`

	// Readiness indicates if the service can accept traffic.
	Readiness bool `json:"readiness"`

	// Timestamp is when this health check was performed.
	Timestamp string `json:"timestamp"`

	// RequestID is the request ID for this health check.
	RequestID string `json:"request_id,omitempty"`

	// Dependencies shows the status of each dependency.
	Dependencies DependencyStatuses `json:"dependencies"`

	// Capabilities shows which features are available.
	Capabilities CapabilityStatuses `json:"capabilities"`

	// Degradations lists any active degradations with details.
	Degradations []DegradationInfo `json:"degradations,omitempty"`
}

// DependencyStatuses maps dependency names to their status details.
type DependencyStatuses struct {
	Database DatabaseDependency `json:"database"`
	Ollama   OllamaDependency   `json:"ollama"`
}

// DatabaseDependency represents database health.
type DatabaseDependency struct {
	Status   DependencyStatus `json:"status"`
	Latency  string           `json:"latency,omitempty"`
	Required bool             `json:"required"`
}

// OllamaDependency represents Ollama service health.
type OllamaDependency struct {
	Status   DependencyStatus `json:"status"`
	Latency  string           `json:"latency,omitempty"`
	Required bool             `json:"required"`
}

// CapabilityStatuses indicates which features are functional.
type CapabilityStatuses struct {
	ChatCompletion bool `json:"chat_completion"`
	AutoNaming     bool `json:"auto_naming"`
	ToolExecution  bool `json:"tool_execution"`
}

// DegradationInfo describes a specific degradation.
type DegradationInfo struct {
	// Component is the affected component or feature.
	Component string `json:"component"`

	// Reason explains why the degradation occurred.
	Reason string `json:"reason"`

	// Impact describes how this affects users.
	Impact string `json:"impact"`

	// Recovery suggests what action to take.
	Recovery string `json:"recovery"`
}

