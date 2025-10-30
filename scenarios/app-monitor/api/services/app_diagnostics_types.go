package services

import "time"

// =============================================================================
// Scenario Status Diagnostics Types
// =============================================================================

// ScenarioStatusSeverity describes the overall health of a scenario status snapshot.
type ScenarioStatusSeverity string

const (
	ScenarioStatusSeverityOK    ScenarioStatusSeverity = "ok"
	ScenarioStatusSeverityWarn  ScenarioStatusSeverity = "warn"
	ScenarioStatusSeverityError ScenarioStatusSeverity = "error"
)

// AppScenarioStatus captures a sanitized snapshot of `vrooli scenario status` for a single scenario.
type AppScenarioStatus struct {
	AppID           string                 `json:"appId"`
	Scenario        string                 `json:"scenario"`
	CapturedAt      string                 `json:"capturedAt,omitempty"`
	StatusLabel     string                 `json:"statusLabel"`
	Severity        ScenarioStatusSeverity `json:"severity"`
	Runtime         string                 `json:"runtime,omitempty"`
	ProcessCount    int                    `json:"processCount,omitempty"`
	Ports           map[string]int         `json:"ports,omitempty"`
	Recommendations []string               `json:"recommendations,omitempty"`
	Details         []string               `json:"details"`
}

// scenarioStatusCLIResponse represents the parsed JSON output from `vrooli scenario status --json`
type scenarioStatusCLIResponse struct {
	Success      bool   `json:"success"`
	ScenarioName string `json:"scenario_name"`
	ScenarioData struct {
		Status         string                  `json:"status"`
		Runtime        string                  `json:"runtime"`
		StartedAt      string                  `json:"started_at"`
		AllocatedPorts map[string]int          `json:"allocated_ports"`
		Processes      []scenarioStatusProcess `json:"processes"`
	} `json:"scenario_data"`
	Diagnostics struct {
		HealthChecks map[string]scenarioStatusHealthCheck `json:"health_checks"`
	} `json:"diagnostics"`
	TestInfrastructure scenarioStatusTestInfrastructure `json:"test_infrastructure"`
	Recommendations    []string                         `json:"recommendations"`
	Metadata           struct {
		Timestamp string `json:"timestamp"`
	} `json:"metadata"`
	RawResponse struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	} `json:"raw_response"`
}

type scenarioStatusProcess struct {
	PID      int               `json:"pid"`
	Status   string            `json:"status"`
	StepName string            `json:"step_name"`
	Ports    map[string]int    `json:"ports"`
	Meta     map[string]string `json:"meta"`
}

type scenarioStatusConnectivity struct {
	Connected bool     `json:"connected"`
	APIURL    string   `json:"api_url"`
	Error     string   `json:"error"`
	LatencyMs *float64 `json:"latency_ms"`
}

type scenarioStatusDependency struct {
	Connected bool   `json:"connected"`
	Status    string `json:"status"`
}

type scenarioStatusHealthCheck struct {
	Name            string                      `json:"name"`
	Status          string                      `json:"status"`
	Port            int                         `json:"port"`
	Available       bool                        `json:"available"`
	ResponseTime    *float64                    `json:"response_time"`
	SchemaValid     *bool                       `json:"schema_valid"`
	APIConnectivity *scenarioStatusConnectivity `json:"api_connectivity"`
	Dependencies    map[string]interface{}      `json:"dependencies"`
	Message         string                      `json:"message"`
}

type scenarioStatusTestEntry struct {
	Status          string   `json:"status"`
	Message         string   `json:"message"`
	Recommendation  string   `json:"recommendation"`
	Recommendations []string `json:"recommendations"`
	Types           []string `json:"types"`
	Tests           []string `json:"tests"`
	Workflows       []string `json:"workflows"`
}

type scenarioStatusTestInfrastructure struct {
	Overall         *scenarioStatusTestEntry `json:"overall"`
	TestLifecycle   *scenarioStatusTestEntry `json:"test_lifecycle"`
	PhasedStructure *scenarioStatusTestEntry `json:"phased_structure"`
	UnitTests       *scenarioStatusTestEntry `json:"unit_tests"`
	CliTests        *scenarioStatusTestEntry `json:"cli_tests"`
	UiTests         *scenarioStatusTestEntry `json:"ui_tests"`
}

// =============================================================================
// Health Check Diagnostics Types
// =============================================================================

// AppHealthDiagnostics captures health check results for the previewed application.
type AppHealthDiagnostics struct {
	AppID      string                  `json:"app_id"`
	AppName    string                  `json:"app_name,omitempty"`
	Scenario   string                  `json:"scenario,omitempty"`
	CapturedAt string                  `json:"captured_at"`
	Ports      map[string]int          `json:"ports,omitempty"`
	Checks     []IssueHealthCheckEntry `json:"checks"`
	Errors     []string                `json:"errors,omitempty"`
}

// IssueHealthCheckEntry represents a single health check result in issue reports
type IssueHealthCheckEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Endpoint  string `json:"endpoint,omitempty"`
	LatencyMs *int   `json:"latencyMs,omitempty"`
	Message   string `json:"message,omitempty"`
	Code      string `json:"code,omitempty"`
	Response  string `json:"response,omitempty"`
}

// =============================================================================
// Iframe Bridge Rule Validation Types
// =============================================================================

// BridgeRuleViolation represents a single iframe bridge rule violation
type BridgeRuleViolation struct {
	RuleID         string `json:"rule_id,omitempty"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path"`
	Line           int    `json:"line"`
	Recommendation string `json:"recommendation"`
	Severity       string `json:"severity"`
	Standard       string `json:"standard,omitempty"`
}

type BridgeRuleReport struct {
	RuleID       string                `json:"rule_id"`
	Name         string                `json:"name,omitempty"`
	Scenario     string                `json:"scenario"`
	FilesScanned int                   `json:"files_scanned"`
	DurationMs   int64                 `json:"duration_ms"`
	Warning      string                `json:"warning,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
	Targets      []string              `json:"targets,omitempty"`
	Violations   []BridgeRuleViolation `json:"violations"`
	CheckedAt    time.Time             `json:"checked_at"`
}

type BridgeDiagnosticsReport struct {
	Scenario     string                `json:"scenario"`
	CheckedAt    time.Time             `json:"checked_at"`
	FilesScanned int                   `json:"files_scanned"`
	DurationMs   int64                 `json:"duration_ms"`
	Warning      string                `json:"warning,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
	Targets      []string              `json:"targets,omitempty"`
	Violations   []BridgeRuleViolation `json:"violations"`
	Results      []BridgeRuleReport    `json:"results"`
}

// scenarioAuditorRuleResponse represents the API response from scenario-auditor rule tests
type scenarioAuditorRuleResponse struct {
	RuleID       string                     `json:"rule_id"`
	Scenario     string                     `json:"scenario"`
	FilesScanned int                        `json:"files_scanned"`
	Violations   []scenarioAuditorViolation `json:"violations"`
	Targets      []string                   `json:"targets"`
	DurationMs   int64                      `json:"duration_ms"`
	Warning      string                     `json:"warning"`
}

type scenarioAuditorViolation struct {
	ID             string `json:"id"`
	ScenarioName   string `json:"scenario_name"`
	Type           string `json:"type"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path"`
	LineNumber     int    `json:"line_number"`
	Recommendation string `json:"recommendation"`
	Standard       string `json:"standard"`
}

// =============================================================================
// Localhost Usage Scanning Types
// =============================================================================

type LocalhostUsageFinding struct {
	FilePath string `json:"file_path"`
	Line     int    `json:"line"`
	Snippet  string `json:"snippet"`
	Pattern  string `json:"pattern"`
}

type LocalhostUsageReport struct {
	Scenario  string                  `json:"scenario"`
	CheckedAt time.Time               `json:"checked_at"`
	Findings  []LocalhostUsageFinding `json:"findings"`
	Scanned   int                     `json:"files_scanned"`
	Warnings  []string                `json:"warnings,omitempty"`
}
