// Package vrooli provides Vrooli-specific health checks
// [REQ:VROOLI-API-001]
package vrooli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// APICheck monitors the main Vrooli API health endpoint.
type APICheck struct {
	url     string
	timeout time.Duration
}

// APICheckOption configures an APICheck.
type APICheckOption func(*APICheck)

// WithAPIURL sets the API health endpoint URL.
func WithAPIURL(url string) APICheckOption {
	return func(c *APICheck) {
		c.url = url
	}
}

// WithAPITimeout sets the HTTP request timeout.
func WithAPITimeout(timeout time.Duration) APICheckOption {
	return func(c *APICheck) {
		c.timeout = timeout
	}
}

// NewAPICheck creates a Vrooli API health check.
// Default URL: http://127.0.0.1:8092/health
// Default timeout: 5 seconds
func NewAPICheck(opts ...APICheckOption) *APICheck {
	c := &APICheck{
		url:     "http://127.0.0.1:8092/health",
		timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *APICheck) ID() string          { return "vrooli-api" }
func (c *APICheck) Title() string       { return "Vrooli API" }
func (c *APICheck) Description() string { return "Checks the main Vrooli API health endpoint" }
func (c *APICheck) Importance() string {
	return "The Vrooli API is the central orchestration layer for all scenarios and resources"
}
func (c *APICheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *APICheck) IntervalSeconds() int       { return 60 }
func (c *APICheck) Platforms() []platform.Type { return nil } // all platforms

func (c *APICheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{
			"url":     c.url,
			"timeout": c.timeout.String(),
		},
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: c.timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to create request"
		result.Details["error"] = err.Error()
		return result
	}

	// Execute request
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	result.Details["responseTimeMs"] = elapsed.Milliseconds()

	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Vrooli API is not responding"
		result.Details["error"] = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.Details["statusCode"] = resp.StatusCode

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read API response"
		result.Details["error"] = err.Error()
		return result
	}

	// Parse JSON response
	var healthResponse struct {
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}

	if err := json.Unmarshal(body, &healthResponse); err != nil {
		// If not JSON, check HTTP status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Status = checks.StatusOK
			result.Message = "Vrooli API is responding (non-JSON response)"
			return result
		}
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Vrooli API returned HTTP %d", resp.StatusCode)
		return result
	}

	result.Details["apiStatus"] = healthResponse.Status

	// Calculate health score based on response time
	score := 100
	if elapsed > 3*time.Second {
		score = 50
	} else if elapsed > 1*time.Second {
		score = 75
	}

	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "api-response",
				Passed: healthResponse.Status == "ok" || healthResponse.Status == "healthy",
				Detail: fmt.Sprintf("Status: %s, Response time: %dms", healthResponse.Status, elapsed.Milliseconds()),
			},
		},
	}

	// Interpret health status
	switch healthResponse.Status {
	case "ok", "healthy":
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Vrooli API healthy (response time: %dms)", elapsed.Milliseconds())
	case "degraded":
		result.Status = checks.StatusWarning
		result.Message = "Vrooli API is degraded"
		if healthResponse.Message != "" {
			result.Details["degradedReason"] = healthResponse.Message
		}
	case "unhealthy", "failed", "error":
		result.Status = checks.StatusCritical
		result.Message = "Vrooli API reports unhealthy status"
		if healthResponse.Message != "" {
			result.Details["errorMessage"] = healthResponse.Message
		}
	default:
		// Unknown status - assume OK if HTTP 2xx
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Status = checks.StatusOK
			result.Message = fmt.Sprintf("Vrooli API responding (status: %s)", healthResponse.Status)
		} else {
			result.Status = checks.StatusCritical
			result.Message = fmt.Sprintf("Vrooli API returned unexpected status: %s", healthResponse.Status)
		}
	}

	return result
}
