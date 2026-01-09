package health

import "time"

// Status represents the health status of the sidecar.
type Status string

const (
	// StatusHealthy indicates the sidecar is running and healthy.
	StatusHealthy Status = "healthy"

	// StatusDegraded indicates the sidecar is running but has issues
	// (e.g., browser crashed but sidecar is still responding).
	StatusDegraded Status = "degraded"

	// StatusUnhealthy indicates the sidecar is not responding.
	StatusUnhealthy Status = "unhealthy"

	// StatusRestarting indicates the supervisor is restarting the sidecar.
	StatusRestarting Status = "restarting"

	// StatusUnrecoverable indicates the sidecar has crashed too many times.
	StatusUnrecoverable Status = "unrecoverable"
)

// DriverHealth represents the current health state of the playwright-driver sidecar.
// This is broadcast to UI clients via WebSocket.
type DriverHealth struct {
	// Status is the overall health status.
	Status Status `json:"status"`

	// CircuitBreaker is the state of the circuit breaker: "open", "closed", "half-open".
	CircuitBreaker string `json:"circuitBreaker"`

	// ActiveSessions is the number of active browser sessions.
	ActiveSessions int `json:"activeSessions"`

	// RestartCount is the number of restarts within the current window.
	RestartCount int `json:"restartCount"`

	// UptimeMS is how long the sidecar has been running (milliseconds).
	UptimeMS int64 `json:"uptimeMs"`

	// LastError is the most recent error message, if any.
	LastError *string `json:"lastError,omitempty"`

	// EstimatedRecoveryMS is the estimated time until recovery (when unhealthy).
	// Based on circuit breaker timeout and backoff calculation.
	EstimatedRecoveryMS *int64 `json:"estimatedRecoveryMs,omitempty"`

	// UpdatedAt is when this health state was last updated.
	UpdatedAt time.Time `json:"updatedAt"`
}

// DriverHealthResponse is the response from the sidecar's /health endpoint.
type DriverHealthResponse struct {
	// Status is the health status: "healthy", "degraded", "unhealthy".
	Status string `json:"status"`

	// ActiveSessions is the number of active browser sessions.
	ActiveSessions int `json:"activeSessions"`

	// BrowserStatus is the browser status: "running", "crashed", "starting", "stopped".
	BrowserStatus string `json:"browserStatus"`

	// Uptime is how long the sidecar has been running (milliseconds).
	Uptime int64 `json:"uptime"`

	// LastError is the most recent error message, if any.
	LastError *string `json:"lastError,omitempty"`

	// Version is the sidecar version.
	Version string `json:"version,omitempty"`
}

// IsHealthy returns true if the status indicates a healthy sidecar.
func (s Status) IsHealthy() bool {
	return s == StatusHealthy || s == StatusDegraded
}

// IsAvailable returns true if the sidecar is available for requests.
func (s Status) IsAvailable() bool {
	return s == StatusHealthy || s == StatusDegraded
}

// NewHealthyState creates a DriverHealth from a successful health check response.
func NewHealthyState(resp *DriverHealthResponse, restartCount int, cbState string) DriverHealth {
	status := StatusHealthy
	if resp.Status == "degraded" || resp.BrowserStatus == "crashed" {
		status = StatusDegraded
	}

	return DriverHealth{
		Status:         status,
		CircuitBreaker: cbState,
		ActiveSessions: resp.ActiveSessions,
		RestartCount:   restartCount,
		UptimeMS:       resp.Uptime,
		LastError:      resp.LastError,
		UpdatedAt:      time.Now(),
	}
}

// NewUnhealthyState creates a DriverHealth for a failed health check.
func NewUnhealthyState(err error, restartCount int, cbState string, estimatedRecoveryMS *int64) DriverHealth {
	errStr := err.Error()
	return DriverHealth{
		Status:              StatusUnhealthy,
		CircuitBreaker:      cbState,
		RestartCount:        restartCount,
		LastError:           &errStr,
		EstimatedRecoveryMS: estimatedRecoveryMS,
		UpdatedAt:           time.Now(),
	}
}

// NewRestartingState creates a DriverHealth for when the supervisor is restarting.
func NewRestartingState(restartCount int, cbState string) DriverHealth {
	return DriverHealth{
		Status:         StatusRestarting,
		CircuitBreaker: cbState,
		RestartCount:   restartCount,
		UpdatedAt:      time.Now(),
	}
}

// NewUnrecoverableState creates a DriverHealth for when max restarts are exceeded.
func NewUnrecoverableState(restartCount int, cbState string) DriverHealth {
	return DriverHealth{
		Status:         StatusUnrecoverable,
		CircuitBreaker: cbState,
		RestartCount:   restartCount,
		UpdatedAt:      time.Now(),
	}
}
