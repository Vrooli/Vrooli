package smoketest

import (
	"regexp"
	"sort"
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

// lifecycleMarker pairs a marker string with the state name it represents.
type lifecycleMarker struct {
	marker string
	state  string
}

// lifecycleMarkers returns the ordered list of lifecycle markers to check.
// The order matches the progression: init -> granular stages -> ready -> result -> exit.
func (p *DefaultOutputParser) lifecycleMarkers() []lifecycleMarker {
	g := p.config.GranularLifecycleMarkers
	return []lifecycleMarker{
		{p.config.InitMarker, "init"},
		{g.BundleResolving, "bundle_resolving"},
		{g.RuntimeStarting, "runtime_starting"},
		{g.WaitingForToken, "waiting_for_token"},
		{g.RuntimeHealthz, "runtime_healthz"},
		{g.RuntimeReadyz, "runtime_readyz"},
		{g.RuntimePorts, "runtime_ports"},
		{g.UIServerCheck, "ui_server_check"},
		{p.config.ReadyMarker, "ready"},
	}
}

// ExtractLastLifecycleState returns the last lifecycle marker reached.
// Returns empty string if no lifecycle markers are found.
// Possible values (in order of progression):
// Basic: "init", "ready", "result", "exit"
// Granular (bundled mode): "bundle_resolving", "runtime_starting", "runtime_healthz", "runtime_readyz", "runtime_ports"
func (p *DefaultOutputParser) ExtractLastLifecycleState(output string) string {
	lastState := ""

	for _, m := range p.lifecycleMarkers() {
		if m.marker != "" && strings.Contains(output, m.marker) {
			lastState = m.state
		}
	}

	// Result uses two possible markers (success or failure)
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

// markerDef pairs a stage name with its expected output marker.
type markerDef struct {
	name   string
	marker string
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

	expectedOrder := []markerDef{
		{"init", p.config.InitMarker},
		{"ready", p.config.ReadyMarker},
		{"passed", p.config.SuccessMarker},
		{"exit", p.config.ExitMarker},
	}

	stageLines := scanForMarkers(strings.Split(output, "\n"), expectedOrder)
	for name, lineNum := range stageLines {
		validation.Stages = append(validation.Stages, SequenceStage{Name: name, LineNumber: lineNum})
	}
	sort.Slice(validation.Stages, func(i, j int) bool {
		return validation.Stages[i].LineNumber < validation.Stages[j].LineNumber
	})

	checkMissingStages(&validation, expectedOrder, stageLines)
	checkSequenceOrder(&validation, expectedOrder, stageLines)

	return validation
}

// scanForMarkers finds the first occurrence line number (1-based) of each marker in the output.
func scanForMarkers(lines []string, markers []markerDef) map[string]int {
	stageLines := make(map[string]int)
	for lineNum, line := range lines {
		for _, m := range markers {
			if m.marker != "" && strings.Contains(line, m.marker) {
				if _, seen := stageLines[m.name]; !seen {
					stageLines[m.name] = lineNum + 1
				}
			}
		}
	}
	return stageLines
}

// checkMissingStages adds errors for required stages that were not found.
func checkMissingStages(v *SequenceValidation, markers []markerDef, stageLines map[string]int) {
	for _, m := range markers {
		if m.marker == "" {
			continue
		}
		if _, found := stageLines[m.name]; !found {
			if m.name == "init" || m.name == "exit" || m.name == "ready" {
				continue
			}
			v.MissingStages = append(v.MissingStages, m.name)
			v.Valid = false
			v.Errors = append(v.Errors, "missing required stage: "+m.name)
		}
	}
}

// checkSequenceOrder adds errors for stages that appeared out of expected order.
func checkSequenceOrder(v *SequenceValidation, markers []markerDef, stageLines map[string]int) {
	lastLine := 0
	for _, m := range markers {
		if m.marker == "" {
			continue
		}
		if line, found := stageLines[m.name]; found {
			if line < lastLine {
				v.OutOfOrderStages = append(v.OutOfOrderStages, m.name)
				v.Valid = false
				v.Errors = append(v.Errors, "stage out of order: "+m.name)
			}
			lastLine = line
		}
	}
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
