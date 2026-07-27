package smoketest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FailureReport provides comprehensive diagnosis information.
type FailureReport struct {
	SmokeTestID  string        `json:"smoke_test_id"`
	ScenarioName string        `json:"scenario_name"`
	Platform     string        `json:"platform"`
	FailedAt     time.Time     `json:"failed_at"`
	Duration     time.Duration `json:"duration"`

	// Error details
	ErrorKind       ErrorKind         `json:"error_kind"`
	ErrorMessage    string            `json:"error_message"`
	ErrorContext    map[string]string `json:"error_context"`
	SuggestedAction string            `json:"suggested_action"`

	// State timeline
	Transitions []StateTransition `json:"transitions"`
	FinalState  State             `json:"final_state"`

	// Process details (if execution was attempted)
	Command       string            `json:"command,omitempty"`
	ProcessOutput string            `json:"process_output,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`

	// Diagnostic hints
	DiagnosticHints []string `json:"diagnostic_hints"`
}

// GenerateFailureReport creates a comprehensive report from a failed status.
func GenerateFailureReport(status *Status) *FailureReport {
	report := &FailureReport{
		SmokeTestID:     status.SmokeTestID,
		ScenarioName:    status.ScenarioName,
		Platform:        status.Platform,
		FinalState:      status.CurrentState,
		Transitions:     status.Transitions,
		ErrorMessage:    status.Error,
		ErrorContext:    status.ErrorContext,
		SuggestedAction: status.SuggestedAction,
	}

	if status.ErrorKind != nil {
		report.ErrorKind = *status.ErrorKind
	}

	if status.CompletedAt != nil {
		report.FailedAt = *status.CompletedAt
		report.Duration = status.CompletedAt.Sub(status.StartedAt)
	}

	// Generate diagnostic hints based on error kind
	report.DiagnosticHints = generateDiagnosticHints(status)

	return report
}

func generateDiagnosticHints(status *Status) []string {
	hints := []string{}

	if status.ErrorKind == nil {
		return hints
	}

	switch *status.ErrorKind {
	case ErrKindArtifact:
		hints = append(hints, "Check if build stage completed successfully")
		if status.ArtifactPath != "" {
			hints = append(hints, "Verify artifact path exists: "+status.ArtifactPath)
		}
		hints = append(hints, "Check file permissions on artifact")

	case ErrKindExecution:
		hints = append(hints, "Check if app has correct entry point")
		hints = append(hints, "Look for missing shared libraries (ldd on Linux)")
		hints = append(hints, "Verify app runs manually with --smoke-test flag")

	case ErrKindTimeout:
		hints = append(hints, "App may be waiting for user input or network")
		hints = append(hints, "Check if app logs show startup progress")
		hints = append(hints, "Consider increasing SMOKE_TEST_TIMEOUT_MS")

	case ErrKindValidation:
		hints = append(hints, "App must output: SMOKE_TEST_RESULT=passed")
		hints = append(hints, "Check app's smoke test mode implementation")
		hints = append(hints, "Search process output for 'SMOKE_TEST'")

	case ErrKindPlatform:
		switch status.Platform {
		case "linux":
			hints = append(hints, "Install xvfb: apt-get install xvfb")
			hints = append(hints, "Or set DISPLAY environment variable")
		case "mac":
			hints = append(hints, "Check macOS security permissions for the app")
			hints = append(hints, "Verify app is properly signed or allowed in Security preferences")
		case "win":
			hints = append(hints, "Check Windows Defender/antivirus settings")
			hints = append(hints, "Verify app can run in headless mode on Windows")
		}

	case ErrKindTelemetry:
		hints = append(hints, "Check telemetry service availability")
		hints = append(hints, "Verify network connectivity to telemetry endpoint")
		hints = append(hints, "Check if telemetry file was written to expected location")

	case ErrKindStore:
		hints = append(hints, "Check available disk space")
		hints = append(hints, "Verify file permissions on smoke test store directory")

	case ErrKindCancelled:
		hints = append(hints, "Smoke test was cancelled by user or system")
		hints = append(hints, "Re-run smoke test if needed")
	}

	return hints
}

// FormatForTerminal returns a human-readable failure report.
func (r *FailureReport) FormatForTerminal() string {
	var b strings.Builder

	b.WriteString("\n=== SMOKE TEST FAILURE REPORT ===\n")
	b.WriteString(fmt.Sprintf("ID:       %s\n", r.SmokeTestID))
	b.WriteString(fmt.Sprintf("Scenario: %s\n", r.ScenarioName))
	b.WriteString(fmt.Sprintf("Platform: %s\n", r.Platform))
	b.WriteString(fmt.Sprintf("Duration: %s\n", r.Duration))

	b.WriteString("\n--- Error ---\n")
	b.WriteString(fmt.Sprintf("Kind:    %s\n", r.ErrorKind.String()))
	b.WriteString(fmt.Sprintf("Message: %s\n", r.ErrorMessage))

	if len(r.ErrorContext) > 0 {
		b.WriteString("\n--- Context ---\n")
		for k, v := range r.ErrorContext {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	if len(r.Transitions) > 0 {
		b.WriteString("\n--- State Timeline ---\n")
		for _, t := range r.Transitions {
			b.WriteString(fmt.Sprintf("  [%s] %s -> %s: %s\n",
				t.Timestamp.Format("15:04:05.000"),
				t.From, t.To, t.Message))
		}
	}

	if r.SuggestedAction != "" {
		b.WriteString("\n--- Suggested Action ---\n")
		b.WriteString(fmt.Sprintf("  %s\n", r.SuggestedAction))
	}

	if len(r.DiagnosticHints) > 0 {
		b.WriteString("\n--- Diagnostic Hints ---\n")
		for _, hint := range r.DiagnosticHints {
			b.WriteString(fmt.Sprintf("  - %s\n", hint))
		}
	}

	return b.String()
}

// FormatForJSON returns a JSON-friendly representation (the struct itself is JSON-tagged).
func (r *FailureReport) FormatForJSON() *FailureReport {
	return r
}

// StructuredOutput provides a machine-parseable output format for CI/CD systems.
type StructuredOutput struct {
	Version   string         `json:"version"`
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"` // "error", "warning", "info"
	SmokeTest *FailureReport `json:"smoke_test,omitempty"`
	Summary   OutputSummary  `json:"summary"`
}

// OutputSummary provides a quick overview of the smoke test result.
type OutputSummary struct {
	Status       string `json:"status"` // "passed", "failed"
	ErrorKind    string `json:"error_kind,omitempty"`
	DurationMs   int64  `json:"duration_ms"`
	StateCount   int    `json:"state_count"`
	TelemetryOK  bool   `json:"telemetry_ok"`
	Retryable    bool   `json:"retryable"`
	ActionNeeded string `json:"action_needed,omitempty"`
}

// FormatForCI returns a machine-parseable JSON string for CI integration.
func (r *FailureReport) FormatForCI() string {
	output := StructuredOutput{
		Version:   "1.0",
		Timestamp: time.Now(),
		Level:     "error",
		SmokeTest: r,
		Summary: OutputSummary{
			Status:       "failed",
			ErrorKind:    r.ErrorKind.String(),
			DurationMs:   r.Duration.Milliseconds(),
			StateCount:   len(r.Transitions),
			TelemetryOK:  false,
			Retryable:    isRetryable(r.ErrorKind),
			ActionNeeded: r.SuggestedAction,
		},
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal: %s"}`, err.Error())
	}
	return string(data)
}

// isRetryable returns true if the error kind is typically recoverable through retry.
func isRetryable(kind ErrorKind) bool {
	switch kind {
	case ErrKindTimeout, ErrKindTelemetry, ErrKindStore, ErrKindExecution:
		return true
	default:
		return false
	}
}

// DiagnosticCommand represents a runnable debug command.
type DiagnosticCommand struct {
	Description string `json:"description"`
	Command     string `json:"command"`
	Platform    string `json:"platform,omitempty"` // Empty means all platforms
	Category    string `json:"category"`           // "check", "debug", "fix"
}

// DiagnosticCommands generates runnable debug commands based on the error kind.
func (r *FailureReport) DiagnosticCommands() []DiagnosticCommand {
	commands := []DiagnosticCommand{}

	switch r.ErrorKind {
	case ErrKindArtifact:
		artifactPath := r.ErrorContext["artifact_path"]
		if artifactPath == "" {
			artifactPath = "<artifact_path>"
		}
		commands = append(commands,
			DiagnosticCommand{
				Description: "Check if artifact exists and permissions",
				Command:     fmt.Sprintf("ls -la %s", artifactPath),
				Category:    "check",
			},
			DiagnosticCommand{
				Description: "Check file type",
				Command:     fmt.Sprintf("file %s", artifactPath),
				Category:    "check",
			},
		)

	case ErrKindExecution:
		artifactPath := r.ErrorContext["artifact_path"]
		if artifactPath == "" {
			artifactPath = "<artifact_path>"
		}
		commands = append(commands,
			DiagnosticCommand{
				Description: "Check shared library dependencies",
				Command:     fmt.Sprintf("ldd %s", artifactPath),
				Platform:    "linux",
				Category:    "debug",
			},
			DiagnosticCommand{
				Description: "Check executable permissions",
				Command:     fmt.Sprintf("stat %s", artifactPath),
				Category:    "check",
			},
			DiagnosticCommand{
				Description: "Run manually with verbose output",
				Command:     fmt.Sprintf("%s --smoke-test 2>&1", artifactPath),
				Category:    "debug",
			},
		)

	case ErrKindPlatform:
		switch r.Platform {
		case "linux":
			commands = append(commands,
				DiagnosticCommand{
					Description: "Check if xvfb-run is installed",
					Command:     "which xvfb-run",
					Platform:    "linux",
					Category:    "check",
				},
				DiagnosticCommand{
					Description: "Check DISPLAY environment variable",
					Command:     "echo $DISPLAY",
					Platform:    "linux",
					Category:    "check",
				},
				DiagnosticCommand{
					Description: "Install xvfb (requires sudo)",
					Command:     "sudo apt-get install -y xvfb",
					Platform:    "linux",
					Category:    "fix",
				},
			)
		case "mac":
			commands = append(commands,
				DiagnosticCommand{
					Description: "Check Gatekeeper status",
					Command:     "spctl --status",
					Platform:    "mac",
					Category:    "check",
				},
			)
		case "win":
			commands = append(commands,
				DiagnosticCommand{
					Description: "Check if app is blocked by Windows Defender",
					Command:     "Get-MpThreatDetection | Select-Object -Last 5",
					Platform:    "win",
					Category:    "check",
				},
			)
		}

	case ErrKindTimeout:
		commands = append(commands,
			DiagnosticCommand{
				Description: "Check system resources",
				Command:     "top -b -n 1 | head -20",
				Platform:    "linux",
				Category:    "debug",
			},
			DiagnosticCommand{
				Description: "Check available memory",
				Command:     "free -h",
				Platform:    "linux",
				Category:    "check",
			},
		)

	case ErrKindTelemetry:
		telemetryPath := r.ErrorContext["telemetry_path"]
		if telemetryPath == "" {
			telemetryPath = "<telemetry_path>"
		}
		commands = append(commands,
			DiagnosticCommand{
				Description: "Check telemetry file exists",
				Command:     fmt.Sprintf("ls -la %s", telemetryPath),
				Category:    "check",
			},
			DiagnosticCommand{
				Description: "View telemetry file contents",
				Command:     fmt.Sprintf("cat %s | head -20", telemetryPath),
				Category:    "debug",
			},
			DiagnosticCommand{
				Description: "Check network connectivity to the Connect telemetry endpoint",
				Command:     "curl -v http://127.0.0.1:8080/vrooli.scenario_to_desktop.v1.domain.TelemetryService/IngestTelemetry",
				Category:    "check",
			},
		)

	case ErrKindStore:
		commands = append(commands,
			DiagnosticCommand{
				Description: "Check disk space",
				Command:     "df -h",
				Category:    "check",
			},
			DiagnosticCommand{
				Description: "Check inode usage",
				Command:     "df -i",
				Category:    "check",
			},
		)

	case ErrKindValidation:
		commands = append(commands,
			DiagnosticCommand{
				Description: "Search for smoke test markers in recent logs",
				Command:     "grep -E 'SMOKE_TEST_(RESULT|INIT|READY|EXIT)' /var/log/syslog | tail -20",
				Platform:    "linux",
				Category:    "debug",
			},
		)
	}

	return commands
}
