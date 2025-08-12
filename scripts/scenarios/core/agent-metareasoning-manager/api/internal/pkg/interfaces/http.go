package interfaces

import "net/http"

// HTTPClientFactory defines the interface for HTTP client management
type HTTPClientFactory interface {
	GetClient(clientType HTTPClientType) *http.Client
}

// HTTPClientType represents different types of HTTP clients
type HTTPClientType int

const (
	HealthCheckClient HTTPClientType = iota
	WorkflowClient
	AIClient
)