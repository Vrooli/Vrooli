package smoketest

import (
	"regexp"
	"strings"
)

// AppError represents a structured error reported by the app via SMOKE_TEST_ERROR marker.
type AppError struct {
	Kind    string // config, network, runtime, validation
	Message string
}

// DefaultOutputParser implements OutputParser using configurable markers.
type DefaultOutputParser struct {
	config Config
}

// NewOutputParser creates a new output parser with the given config.
func NewOutputParser(config Config) *DefaultOutputParser {
	return &DefaultOutputParser{config: config}
}

// appErrorRegex matches SMOKE_TEST_ERROR kind=<kind> msg="<message>"
// The message can contain escaped quotes (\")
var appErrorRegex = regexp.MustCompile(`SMOKE_TEST_ERROR kind=(\w+) msg="((?:[^"\\]|\\.)*)"`)

// sessionIDRegex matches SMOKE_TEST_INIT=started session_id=<uuid>
var sessionIDRegex = regexp.MustCompile(`SMOKE_TEST_INIT=started session_id=([a-f0-9-]+)`)

// ExtractAppError parses SMOKE_TEST_ERROR markers from output.
// Returns nil if no app error marker is found.
func (p *DefaultOutputParser) ExtractAppError(output string) *AppError {
	matches := appErrorRegex.FindStringSubmatch(output)
	if len(matches) == 3 {
		// Unescape the message (replace \" with ")
		message := strings.ReplaceAll(matches[2], `\"`, `"`)
		return &AppError{Kind: matches[1], Message: message}
	}
	return nil
}

// ExtractLastLifecycleState returns the last lifecycle marker reached.
// Returns empty string if no lifecycle markers are found.
// Possible values (in order of progression):
// Basic: "init", "ready", "result", "exit"
// Granular (bundled mode): "bundle_resolving", "runtime_starting", "runtime_healthz", "runtime_readyz", "runtime_ports"
func (p *DefaultOutputParser) ExtractLastLifecycleState(output string) string {
	lastState := ""

	// Basic lifecycle markers
	if p.config.InitMarker != "" && strings.Contains(output, p.config.InitMarker) {
		lastState = "init"
	}

	// Granular bundled-mode lifecycle markers (these occur between init and ready)
	granular := p.config.GranularLifecycleMarkers
	if granular.BundleResolving != "" && strings.Contains(output, granular.BundleResolving) {
		lastState = "bundle_resolving"
	}
	if granular.RuntimeStarting != "" && strings.Contains(output, granular.RuntimeStarting) {
		lastState = "runtime_starting"
	}
	if granular.WaitingForToken != "" && strings.Contains(output, granular.WaitingForToken) {
		lastState = "waiting_for_token"
	}
	if granular.RuntimeHealthz != "" && strings.Contains(output, granular.RuntimeHealthz) {
		lastState = "runtime_healthz"
	}
	if granular.RuntimeReadyz != "" && strings.Contains(output, granular.RuntimeReadyz) {
		lastState = "runtime_readyz"
	}
	if granular.RuntimePorts != "" && strings.Contains(output, granular.RuntimePorts) {
		lastState = "runtime_ports"
	}
	if granular.UIServerCheck != "" && strings.Contains(output, granular.UIServerCheck) {
		lastState = "ui_server_check"
	}

	// Continue with basic markers
	if p.config.ReadyMarker != "" && strings.Contains(output, p.config.ReadyMarker) {
		lastState = "ready"
	}
	if strings.Contains(output, p.config.SuccessMarker) || strings.Contains(output, "SMOKE_TEST_RESULT=failed") {
		lastState = "result"
	}
	if p.config.ExitMarker != "" && strings.Contains(output, p.config.ExitMarker) {
		lastState = "exit"
	}
	return lastState
}

// LifecycleStateDescription returns a human-readable description for a lifecycle state.
// This helps users understand where the app failed during startup.
func LifecycleStateDescription(state string) string {
	descriptions := map[string]string{
		"":                  "App crashed before smoke test code ran. Check for missing dependencies or Electron initialization failures.",
		"init":              "App started smoke test but crashed during initialization. A bundled service likely failed to start.",
		"bundle_resolving":  "App is locating the bundle directory. Check if bundle is packaged correctly in extraResources.",
		"runtime_starting":  "App is spawning the bundled runtime process. Check runtime binary permissions and dependencies.",
		"waiting_for_token": "App is waiting for runtime auth token file. The runtime process started but may not be creating its token.",
		"runtime_healthz":   "App is waiting for runtime /healthz endpoint. The runtime may still be starting or crashed.",
		"runtime_readyz":    "App is waiting for runtime /readyz endpoint. Runtime started but services not ready. A bundled service likely failed to start.",
		"runtime_ports":     "App is querying runtime /ports endpoint. Services are ready but port configuration may be wrong.",
		"ui_server_check":   "App is verifying the UI server responds with HTTP 2xx. The server may be returning an error status (e.g., 404).",
		"ready":             "App initialized successfully but didn't report final result. Check for server connectivity issues.",
		"result":            "App reported result but didn't exit cleanly. This is usually non-fatal.",
		"exit":              "App completed full lifecycle. Should be success - check for race conditions if failed.",
	}
	if desc, ok := descriptions[state]; ok {
		return desc
	}
	return "Unknown state: " + state
}

// ParseResult analyzes smoke test output and returns the result.
func (p *DefaultOutputParser) ParseResult(output string) OutputResult {
	result := OutputResult{
		Passed:               strings.Contains(output, p.config.SuccessMarker),
		TelemetryUploaded:    strings.Contains(output, p.config.UploadSuccessMarker),
		TelemetryUploadError: strings.Contains(output, p.config.UploadErrorMarker),
		InitComplete:         strings.Contains(output, p.config.InitMarker),
		CleanShutdown:        strings.Contains(output, p.config.ExitMarker),
		Warnings:             []string{},
	}

	// Check for potential issues and add warnings
	if result.Passed && !result.InitComplete && p.config.InitMarker != "" {
		result.Warnings = append(result.Warnings, "success marker found but init marker missing")
	}
	if result.Passed && !result.CleanShutdown && p.config.ExitMarker != "" {
		result.Warnings = append(result.Warnings, "success marker found but clean exit marker missing")
	}

	return result
}

// ValidateSequence performs detailed validation of the smoke test output sequence.
func (p *DefaultOutputParser) ValidateSequence(output string) SequenceValidation {
	validation := SequenceValidation{
		Valid:            true,
		Stages:           []SequenceStage{},
		MissingStages:    []string{},
		OutOfOrderStages: []string{},
		Errors:           []string{},
	}

	lines := strings.Split(output, "\n")

	// Define expected sequence order
	type markerDef struct {
		name   string
		marker string
	}
	expectedOrder := []markerDef{
		{"init", p.config.InitMarker},
		{"ready", p.config.ReadyMarker},
		{"passed", p.config.SuccessMarker},
		{"exit", p.config.ExitMarker},
	}

	// Track which stages we've seen and their line numbers
	stageLines := make(map[string]int)

	// Scan output for markers
	for lineNum, line := range lines {
		for _, m := range expectedOrder {
			if m.marker != "" && strings.Contains(line, m.marker) {
				if _, seen := stageLines[m.name]; !seen {
					stageLines[m.name] = lineNum + 1 // 1-based line numbers
					validation.Stages = append(validation.Stages, SequenceStage{
						Name:       m.name,
						LineNumber: lineNum + 1,
					})
				}
			}
		}
	}

	// Check for missing stages (only if marker is configured)
	for _, m := range expectedOrder {
		if m.marker == "" {
			continue
		}
		if _, found := stageLines[m.name]; !found {
			// init and exit are optional for backwards compatibility
			if m.name == "init" || m.name == "exit" || m.name == "ready" {
				continue
			}
			validation.MissingStages = append(validation.MissingStages, m.name)
			validation.Valid = false
			validation.Errors = append(validation.Errors, "missing required stage: "+m.name)
		}
	}

	// Check sequence order
	lastLine := 0
	for _, m := range expectedOrder {
		if m.marker == "" {
			continue
		}
		if line, found := stageLines[m.name]; found {
			if line < lastLine {
				validation.OutOfOrderStages = append(validation.OutOfOrderStages, m.name)
				validation.Valid = false
				validation.Errors = append(validation.Errors, "stage out of order: "+m.name)
			}
			lastLine = line
		}
	}

	return validation
}

// ExtractSessionID parses the session ID from the SMOKE_TEST_INIT marker.
// Returns empty string if no session ID is found.
func (p *DefaultOutputParser) ExtractSessionID(output string) string {
	matches := sessionIDRegex.FindStringSubmatch(output)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}
