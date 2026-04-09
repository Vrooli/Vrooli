package pipeline

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// pipelineStatus represents a full pipeline status response with error info.
type pipelineStatus struct {
	PipelineID      string                  `json:"pipeline_id"`
	Status          string                  `json:"status"`
	CurrentStage    string                  `json:"current_stage,omitempty"`
	CurrentState    string                  `json:"current_state,omitempty"`
	ProgressPercent int                     `json:"progress_percent,omitempty"`
	ProgressMessage string                  `json:"progress_message,omitempty"`
	ScenarioName    string                  `json:"scenario_name,omitempty"`
	FinalArtifacts  map[string]string       `json:"final_artifacts,omitempty"`
	StartedAt       int64                   `json:"started_at,omitempty"`
	CompletedAt     int64                   `json:"completed_at,omitempty"`
	Error           string                  `json:"error,omitempty"`
	Stages          map[string]*stageResult `json:"stages,omitempty"`
	Config          *pipelineConfig         `json:"config,omitempty"`
}

type pipelineConfig struct {
	Version   string   `json:"version,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
	Deploy    *struct {
		ScenarioName  string `json:"scenario_name,omitempty"`
		RemoteProfile string `json:"remote_profile,omitempty"`
		AppKey        string `json:"app_key,omitempty"`
		Channel       string `json:"channel,omitempty"`
	} `json:"deploy,omitempty"`
}

// stageResult represents a stage result with optional error info.
type stageResult struct {
	Status    string          `json:"status"`
	Error     string          `json:"error,omitempty"`
	ErrorInfo *stageErrorInfo `json:"error_info,omitempty"`
	Logs      []string        `json:"logs,omitempty"`
	Details   json.RawMessage `json:"details,omitempty"`
}

type deployStageDetails struct {
	UpdateURL string `json:"update_url"`
}

// stageErrorInfo contains structured error information for stage failures.
type stageErrorInfo struct {
	Code         string      `json:"code,omitempty"`
	Message      string      `json:"message,omitempty"`
	Recovery     string      `json:"recovery,omitempty"`
	RecoveryHint string      `json:"recovery_hint,omitempty"`
	AutoFix      *autoFix    `json:"auto_fix,omitempty"`
	ManualSteps  []string    `json:"manual_steps,omitempty"`
	Diagnostic   *diagnostic `json:"diagnostic,omitempty"`
}

// autoFix contains an auto-fix command suggestion.
type autoFix struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
	Safe        bool   `json:"safe,omitempty"`
}

// diagnostic contains process diagnostic information.
type diagnostic struct {
	Process *processDiagnostic `json:"process,omitempty"`
}

// processDiagnostic contains process execution details.
type processDiagnostic struct {
	LastOutput string `json:"last_output,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
}

// smokeTestDetails contains smoke test execution details from the API response.
type smokeTestDetails struct {
	LastStdout            string               `json:"last_stdout,omitempty"`
	LastStderr            string               `json:"last_stderr,omitempty"`
	Error                 string               `json:"error,omitempty"`
	ErrorKind             int                  `json:"error_kind,omitempty"`
	ErrorContext          map[string]string    `json:"error_context,omitempty"`
	CurrentState          string               `json:"current_state,omitempty"`
	AppReportedError      *appReportedErrorDTO `json:"app_reported_error,omitempty"`
	AppSessionID          string               `json:"app_session_id,omitempty"`
	AppReportedErrorStale bool                 `json:"app_reported_error_stale,omitempty"`
	ErrorSessionMismatch  bool                 `json:"error_session_mismatch,omitempty"`
	Logs                  []string             `json:"logs,omitempty"`
	SmokeTestID           string               `json:"smoke_test_id,omitempty"`
	ScreenRecording       *screenRecordingDTO  `json:"screen_recording,omitempty"`
}

// screenRecordingDTO represents screen recording results from the smoke test.
type screenRecordingDTO struct {
	Recorded      bool   `json:"recorded"`
	VideoPath     string `json:"video_path,omitempty"`
	DurationMs    int64  `json:"duration_ms,omitempty"`
	FileSizeBytes int64  `json:"file_size_bytes,omitempty"`
	Error         string `json:"error,omitempty"`
}

// appReportedErrorDTO represents an error extracted from app telemetry.
type appReportedErrorDTO struct {
	Event          string `json:"event"`
	Message        string `json:"message"`
	DeploymentMode string `json:"deployment_mode,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	Timestamp      string `json:"timestamp,omitempty"`
}

// versionUpdateNotice tracks version update request details for post-pipeline warnings.
type versionUpdateNotice struct {
	requested       bool
	expectedVersion string
	bumpRequested   bool
	bumpValue       string
}

// getSmokeTestDetails extracts smoke test details from the raw JSON details.
func (s *stageResult) getSmokeTestDetails() *smokeTestDetails {
	if len(s.Details) == 0 {
		return nil
	}
	var details smokeTestDetails
	if err := json.Unmarshal(s.Details, &details); err != nil {
		return nil
	}
	// Check if we actually got meaningful data
	if details.LastStdout == "" && details.LastStderr == "" && len(details.ErrorContext) == 0 && details.ScreenRecording == nil {
		return nil
	}
	return &details
}

// getStderr returns stderr from either LastStderr or ErrorContext.
func (d *smokeTestDetails) getStderr() string {
	if d.LastStderr != "" {
		return d.LastStderr
	}
	if d.ErrorContext != nil {
		return d.ErrorContext["stderr"]
	}
	return ""
}

// getLifecycleState returns the last lifecycle state from ErrorContext.
func (d *smokeTestDetails) getLifecycleState() string {
	if d.ErrorContext != nil {
		return d.ErrorContext["last_lifecycle_state"]
	}
	return ""
}

// getAppReportedError returns formatted app-reported error info if available.
func (d *smokeTestDetails) getAppReportedError() string {
	if d.AppReportedError == nil || d.AppReportedError.Message == "" {
		return ""
	}
	return d.AppReportedError.Message
}

// getAppReportedErrorContext returns additional context about the app-reported error.
func (d *smokeTestDetails) getAppReportedErrorContext() string {
	if d.AppReportedError == nil {
		return ""
	}
	var parts []string
	if d.AppReportedError.DeploymentMode != "" {
		parts = append(parts, fmt.Sprintf("deployment_mode=%s", d.AppReportedError.DeploymentMode))
	}
	if d.AppReportedError.Event != "" {
		parts = append(parts, fmt.Sprintf("event=%s", d.AppReportedError.Event))
	}
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return ""
}

// getProgressStages extracts SMOKE_TEST_STAGE markers from stdout.
// Returns the stages completed before failure.
func (d *smokeTestDetails) getProgressStages() []string {
	if d == nil {
		return nil
	}
	return extractProgressStages(d.LastStdout)
}

// getPrereqWarnings extracts prerequisite check warnings from logs.
// Returns warnings that may be relevant to the failure.
func (d *smokeTestDetails) getPrereqWarnings() []string {
	if d == nil || len(d.Logs) == 0 {
		return nil
	}

	var warnings []string
	prereqWarningPattern := regexp.MustCompile(`\[prereq:(\w+)\].*\(suggestion: ([^)]+)\)`)

	for _, log := range d.Logs {
		matches := prereqWarningPattern.FindStringSubmatch(log)
		if len(matches) >= 3 {
			// Extract the warning type and suggestion
			warningType := matches[1]
			suggestion := matches[2]
			warnings = append(warnings, fmt.Sprintf("%s: %s", warningType, suggestion))
		}
	}
	return warnings
}
