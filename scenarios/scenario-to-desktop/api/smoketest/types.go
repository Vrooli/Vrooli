package smoketest

import "time"

// Status represents the status of a desktop smoke test run.
type Status struct {
	SmokeTestID          string     `json:"smoke_test_id"`
	ScenarioName         string     `json:"scenario_name"`
	Platform             string     `json:"platform"`
	Status               string     `json:"status"` // running, passed, failed
	ArtifactPath         string     `json:"artifact_path,omitempty"`
	StartedAt            time.Time  `json:"started_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	Logs                 []string   `json:"logs,omitempty"`
	Error                string     `json:"error,omitempty"`
	TelemetryUploaded    bool       `json:"telemetry_uploaded,omitempty"`
	TelemetryUploadError string     `json:"telemetry_upload_error,omitempty"`
}

// StartRequest represents a request to start a smoke test.
type StartRequest struct {
	ScenarioName string `json:"scenario_name"`
	Platform     string `json:"platform,omitempty"`
}

// StartResponse represents the response from starting a smoke test.
type StartResponse struct {
	SmokeTestID  string    `json:"smoke_test_id"`
	ScenarioName string    `json:"scenario_name"`
	Platform     string    `json:"platform"`
	Status       string    `json:"status"`
	ArtifactPath string    `json:"artifact_path,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	Logs         []string  `json:"logs,omitempty"`
}

// CancelResponse represents the response from cancelling a smoke test.
type CancelResponse struct {
	Status string `json:"status"`
}

// TimeoutSeconds is the default timeout for smoke tests.
const TimeoutSeconds = 30
