package http

import (
	"net/http"
	"time"

	"metareasoning-api/internal/pkg/interfaces"
)

// HTTPClientFactory implements interfaces.HTTPClientFactory
type HTTPClientFactory struct {
	healthClient   *http.Client
	workflowClient *http.Client
	aiClient       *http.Client
}

// NewHTTPClientFactory creates a new HTTP client factory with optimized clients
func NewHTTPClientFactory() interfaces.HTTPClientFactory {
	return &HTTPClientFactory{
		// Health check client: Short timeout for quick checks
		healthClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		// Workflow client: Medium timeout for workflow execution
		workflowClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		// AI client: Long timeout for AI operations
		aiClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     120 * time.Second,
			},
		},
	}
}

// GetClient returns an optimized HTTP client for the specified use case
func (f *HTTPClientFactory) GetClient(clientType interfaces.HTTPClientType) *http.Client {
	switch clientType {
	case interfaces.HealthCheckClient:
		return f.healthClient
	case interfaces.WorkflowClient:
		return f.workflowClient
	case interfaces.AIClient:
		return f.aiClient
	default:
		return f.workflowClient // Default fallback
	}
}