package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"test-genie/internal/structure/types"
)

// Validator validates API health endpoints.
type Validator interface {
	// Validate performs all API health checks.
	Validate(ctx context.Context) ValidationResult
}

// HTTPClient abstracts HTTP operations for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ValidationResult contains the outcome of API validation.
type ValidationResult struct {
	types.Result

	// HealthEndpoint is the endpoint that was checked.
	HealthEndpoint string

	// ResponseTimeMs is the observed response time in milliseconds.
	ResponseTimeMs int64

	// StatusCode is the HTTP status code received.
	StatusCode int
}

// Config holds configuration for API validation.
type Config struct {
	// BaseURL is the base URL of the API (e.g., "http://localhost:8080").
	BaseURL string

	// HealthEndpoint is the health check endpoint path (default: "/health").
	HealthEndpoint string

	// MaxResponseMs is the maximum acceptable response time in milliseconds (default: 1000).
	MaxResponseMs int64

	// ExpectedStatus is the expected HTTP status code (default: 200).
	ExpectedStatus int
}

// validator implements the Validator interface.
type validator struct {
	config    Config
	client    HTTPClient
	logWriter io.Writer
}

// Option configures a validator.
type Option func(*validator)

// New creates a new API validator.
func New(config Config, opts ...Option) Validator {
	// Apply defaults
	if config.HealthEndpoint == "" {
		config.HealthEndpoint = "/health"
	}
	if config.MaxResponseMs == 0 {
		config.MaxResponseMs = 1000
	}
	if config.ExpectedStatus == 0 {
		config.ExpectedStatus = http.StatusOK
	}

	v := &validator{
		config:    config,
		client:    &http.Client{Timeout: 10 * time.Second},
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// WithLogger sets the log writer.
func WithLogger(w io.Writer) Option {
	return func(v *validator) {
		v.logWriter = w
	}
}

// WithHTTPClient sets a custom HTTP client (for testing).
func WithHTTPClient(client HTTPClient) Option {
	return func(v *validator) {
		v.client = client
	}
}

// Validate performs all API health checks.
func (v *validator) Validate(ctx context.Context) ValidationResult {
	if err := ctx.Err(); err != nil {
		return ValidationResult{
			Result: types.FailSystem(err, "Context cancelled"),
		}
	}

	var observations []types.Observation
	observations = append(observations, types.NewSectionObservation("🌐", "Validating API health..."))

	// Build the full URL
	url := v.config.BaseURL + v.config.HealthEndpoint
	v.logStep("Checking health endpoint: %s", url)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ValidationResult{
			Result:         types.FailSystem(fmt.Errorf("failed to create request: %w", err), "Check that the API URL is valid."),
			HealthEndpoint: v.config.HealthEndpoint,
		}
	}

	// Execute request and measure time
	start := time.Now()
	resp, err := v.client.Do(req)
	elapsed := time.Since(start)
	responseMs := elapsed.Milliseconds()

	if err != nil {
		observations = append(observations, types.NewErrorObservation(fmt.Sprintf("health check failed: %v", err)))
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("health endpoint unreachable: %w", err),
				FailureClass: types.FailureClassSystem,
				Remediation:  fmt.Sprintf("Ensure the scenario is running and %s is accessible.", url),
				Observations: observations,
			},
			HealthEndpoint: v.config.HealthEndpoint,
			ResponseTimeMs: responseMs,
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != v.config.ExpectedStatus {
		observations = append(observations, types.NewErrorObservation(fmt.Sprintf("unexpected status: %d (expected %d)", resp.StatusCode, v.config.ExpectedStatus)))
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("health endpoint returned %d, expected %d", resp.StatusCode, v.config.ExpectedStatus),
				FailureClass: types.FailureClassSystem,
				Remediation:  "Check API logs for errors. Ensure the health endpoint is correctly implemented.",
				Observations: observations,
			},
			HealthEndpoint: v.config.HealthEndpoint,
			ResponseTimeMs: responseMs,
			StatusCode:     resp.StatusCode,
		}
	}
	v.logStep("Health endpoint returned status %d", resp.StatusCode)
	observations = append(observations, types.NewSuccessObservation(fmt.Sprintf("health endpoint returned %d", resp.StatusCode)))

	// Check response time
	if responseMs > v.config.MaxResponseMs {
		observations = append(observations, types.NewWarningObservation(fmt.Sprintf("response time %dms exceeds threshold %dms", responseMs, v.config.MaxResponseMs)))
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("response time %dms exceeds threshold %dms", responseMs, v.config.MaxResponseMs),
				FailureClass: types.FailureClassSystem,
				Remediation:  "Investigate API performance. Consider optimizing the health check or increasing the threshold.",
				Observations: observations,
			},
			HealthEndpoint: v.config.HealthEndpoint,
			ResponseTimeMs: responseMs,
			StatusCode:     resp.StatusCode,
		}
	}
	v.logStep("Response time: %dms (threshold: %dms)", responseMs, v.config.MaxResponseMs)
	observations = append(observations, types.NewSuccessObservation(fmt.Sprintf("response time %dms within threshold", responseMs)))

	// Try to parse response body for additional info
	var healthResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err == nil {
		if status, ok := healthResponse["status"].(string); ok {
			v.logStep("Health status: %s", status)
			observations = append(observations, types.NewInfoObservation(fmt.Sprintf("health status: %s", status)))
		}
	}

	return ValidationResult{
		Result: types.Result{
			Success:      true,
			Observations: observations,
		},
		HealthEndpoint: v.config.HealthEndpoint,
		ResponseTimeMs: responseMs,
		StatusCode:     resp.StatusCode,
	}
}

// logStep writes a step message to the log.
func (v *validator) logStep(format string, args ...interface{}) {
	if v.logWriter == nil {
		return
	}
	fmt.Fprintf(v.logWriter, format+"\n", args...)
}
