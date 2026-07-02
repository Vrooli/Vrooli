package services

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sync"
	"time"

	"app-monitor-api/repository"
)

// Package-level errors
var (
	ErrAppIdentifierRequired         = errors.New("app identifier is required")
	ErrAppNotFound                   = errors.New("app not found")
	ErrDatabaseUnavailable           = errors.New("database not available")
	ErrScenarioAuditorUnavailable    = errors.New("scenario-auditor unavailable")
	ErrScenarioBridgeScenarioMissing = errors.New("scenario missing for bridge audit")
	ErrSwarmManagerUnavailable       = errors.New("swarm-manager unavailable")
	ErrPresetNotFound                = errors.New("workspace preset not found")
	ErrPresetNameRequired            = errors.New("preset name is required")
)

// Cache and timing constants
const (
	orchestratorCacheTTL = 90 * time.Second // Increased from 60s to reduce cache misses during slow scenario status calls
	partialCacheTTL      = 45 * time.Second // Increased proportionally
	enrichmentCacheTTL   = 90 * time.Second // Per-scenario tech stack / dependency insights
	completenessCacheTTL = 24 * time.Hour   // Completeness scores change less frequently than runtime status
	fixBacklogCacheTTL   = 30 * time.Second
)

// Issue attachment constants
const (
	attachmentLifecycleName  = "lifecycle.txt"
	attachmentConsoleName    = "console.json"
	attachmentNetworkName    = "network.json"
	attachmentScreenshotName = "screenshot.png"
	attachmentHealthName     = "health.json"
	attachmentStatusName     = "status.txt"
	attachmentReportName     = "report.json"
	swarmManagerScenarioID   = "swarm-manager"
	reportTitleMaxLength     = 120 // Maximum length for issue report titles
	reportLabelMaxLength     = 100 // Maximum length for capture labels
)

// Issue Report Sanitization Limits
const (
	MaxReportLogs           = 300
	MaxConsoleLogEntries    = 200
	MaxNetworkEntries       = 150
	MaxConsoleTextLength    = 2000
	MaxNetworkURLLength     = 2048
	MaxNetworkErrorLength   = 1500
	MaxRequestIDLength      = 128
	MaxHealthCheckEntries   = 20
	MaxHealthNameLength     = 120
	MaxHealthEndpointLength = 512
	MaxHealthMessageLength  = 400
	MaxHealthCodeLength     = 120
	MaxHealthResponseLength = 4000
	MaxStatusLines          = 120
	MaxCaptureEntries       = 12
	MaxCaptureNoteLength    = 600
	MaxCaptureLabelLength   = 160
	MaxCaptureTextLength    = 900
)

// Localhost scanning configuration
var (
	localhostPatterns = []struct {
		Regex *regexp.Regexp
		Label string
	}{
		{Regex: regexp.MustCompile(`(?i)https?://(?:127\.0\.0\.1|localhost|0\.0\.0\.0)`), Label: "HTTP"},
		{Regex: regexp.MustCompile(`(?i)wss?://(?:127\.0\.0\.1|localhost|0\.0\.0\.0)`), Label: "WebSocket"},
		{Regex: regexp.MustCompile(`(?i)(?:^|[^\w])(127\.0\.0\.1|localhost|0\.0\.0\.0):(\d+)`), Label: "HostPort"},
	}

	localhostSkipDirectories = map[string]struct{}{
		".git":         {},
		".hg":          {},
		".svn":         {},
		".cache":       {},
		".next":        {},
		".nuxt":        {},
		"dist":         {},
		"build":        {},
		"node_modules": {},
		"vendor":       {},
		".venv":        {},
		".idea":        {},
		".vscode":      {},
		"coverage":     {},
		"tmp":          {},
		// Test infrastructure directories. Files inside these (e.g. setup.ts,
		// helpers.ts) are not named *.test.ts or *.spec.ts, so isTestFile()
		// alone won't catch them. Scanning them would false-positive on
		// hardcoded localhost URLs used in test configuration / mock data.
		"__tests__":     {},
		"__mocks__":     {},
		"__fixtures__":  {},
		"__snapshots__": {},
	}

	localhostSkipFiles = map[string]struct{}{
		"package-lock.json": {},
		"package-lock.yaml": {},
		"yarn.lock":         {},
		"pnpm-lock.yaml":    {},
	}

	localhostAllowedExtensions = map[string]struct{}{
		".cjs":    {},
		".css":    {},
		".go":     {},
		".htm":    {},
		".html":   {},
		".js":     {},
		".jsx":    {},
		".less":   {},
		".mjs":    {},
		".sass":   {},
		".scss":   {},
		".svelte": {},
		".ts":     {},
		".tsx":    {},
		".vue":    {},
	}

	maxLocalhostScanFileSize int64 = 1 << 20 // 1 MiB
)

var backgroundViewCommandRegex = regexp.MustCompile(`^View:\s+vrooli\s+scenario\s+logs\s+(\S+)\s+--step\s+([^\s]+)`)

// =============================================================================
// Dependency Interfaces
// =============================================================================

// HTTPClient defines the interface for HTTP operations, allowing for testing with mocks
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// TimeProvider defines the interface for time operations, allowing for testing with controlled time
type TimeProvider func() time.Time

// ScenarioURLResolver resolves another scenario's API base URL.
type ScenarioURLResolver func(ctx context.Context, scenarioSlug string) (string, error)

// =============================================================================
// Cache Types
// =============================================================================

// orchestratorCache caches orchestrator data to prevent excessive command execution
type orchestratorCache struct {
	data      []repository.App
	timestamp time.Time
	mu        sync.RWMutex
	isPartial bool
	loading   bool
}

// isFresh returns true if the cache contains valid, non-partial data within TTL
func (c *orchestratorCache) isFresh() bool {
	return time.Since(c.timestamp) < orchestratorCacheTTL &&
		len(c.data) > 0 &&
		!c.isPartial
}

// viewStatsEntry tracks view statistics for an app
type viewStatsEntry struct {
	Count       int64
	FirstViewed time.Time
	HasFirst    bool
	LastViewed  time.Time
	HasLast     bool
}

// fixCacheEntry stores cached Swarm Manager fix backlog information.
type fixCacheEntry struct {
	active    []AppFixSummary
	archived  []AppFixSummary
	scenario  string
	appID     string
	fixesURL  string
	fetchedAt time.Time
}

// enrichmentCacheEntry caches per-scenario tech stack and dependency data
type enrichmentCacheEntry struct {
	techStack    []string
	dependencies []repository.AppDependency
	fetchedAt    time.Time
}

// AppService handles business logic for application management
type AppService struct {
	repo               repository.AppRepository
	httpClient         HTTPClient
	timeNow            TimeProvider
	cache              *orchestratorCache
	completenessCache  *completenessCache
	viewStatsMu        sync.RWMutex
	viewStats          map[string]*viewStatsEntry
	issueCacheMu       sync.RWMutex
	issueCache         map[string]*fixCacheEntry
	issueCacheTTL      time.Duration
	repoRoot           string
	scenarioURL        ScenarioURLResolver
	browserlessService *BrowserlessService
	enrichmentMu       sync.RWMutex
	enrichmentCache    map[string]*enrichmentCacheEntry // key: lowercase scenario name
	uiServerPort       string
	backgroundWg       sync.WaitGroup
}

// =============================================================================
// Orchestrator Types
// =============================================================================

// scenarioMetadata captures static scenario details such as description and filesystem path
type scenarioMetadata struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Path        string         `json:"path"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
	Tags        []string       `json:"tags"`
	Ports       []scenarioPort `json:"ports"`
}

type scenarioPort struct {
	Key  string      `json:"key"`
	Step string      `json:"step"`
	Port interface{} `json:"port"`
}

// =============================================================================
// Log Types
// =============================================================================

// AppLogsResult captures lifecycle logs and background step logs for a scenario.
type AppLogsResult struct {
	Lifecycle  []string
	Background []BackgroundLog
}

// BackgroundLog describes a single background step log stream.
type BackgroundLog struct {
	Step    string
	Phase   string
	Label   string
	Command string
	Lines   []string
}

type backgroundLogCandidate struct {
	Step    string
	Phase   string
	Label   string
	Command string
}

// =============================================================================
// Fix Backlog Types
// =============================================================================

// AppFixSummary represents a Swarm Manager fix backlog item targeting a scenario.
type AppFixSummary struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Priority   int    `json:"priority,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	ArchivedAt string `json:"archived_at,omitempty"`
	Initiative string `json:"initiative,omitempty"`
	Path       string `json:"path,omitempty"`
	URL        string `json:"url,omitempty"`
}

// AppFixesSummary provides aggregated Swarm Manager fix information for an app/scenario.
type AppFixesSummary struct {
	Scenario      string          `json:"scenario"`
	AppID         string          `json:"app_id"`
	Active        []AppFixSummary `json:"active"`
	Archived      []AppFixSummary `json:"archived"`
	Fixes         []AppFixSummary `json:"fixes"`
	ActiveCount   int             `json:"active_count"`
	ArchivedCount int             `json:"archived_count"`
	TotalCount    int             `json:"total_count"`
	SwarmURL      string          `json:"swarm_url,omitempty"`
	LastFetched   string          `json:"last_fetched"`
	FromCache     bool            `json:"from_cache"`
	Stale         bool            `json:"stale"`
}

type swarmBacklogItemResponse struct {
	Item struct {
		Name   string `json:"name"`
		Title  string `json:"title"`
		Kind   string `json:"kind"`
		Status string `json:"status"`
	} `json:"item"`
}

type swarmScenarioContextResponse struct {
	ScenarioName string `json:"scenario_name"`
	Fixes        struct {
		Active   []swarmScenarioFix `json:"active"`
		Archived []swarmScenarioFix `json:"archived"`
	} `json:"fixes"`
}

type swarmScenarioFix struct {
	Name       string  `json:"name"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Priority   int     `json:"priority"`
	Initiative string  `json:"initiative,omitempty"`
	Updated    string  `json:"updated,omitempty"`
	ArchivedAt *string `json:"archived_at,omitempty"`
	Path       string  `json:"path"`
}

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

type scenarioStatusConnectivity struct {
	Connected bool     `json:"connected"`
	APIURL    string   `json:"api_url"`
	Error     string   `json:"error"`
	LatencyMs *float64 `json:"latency_ms"`
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

// ScenarioAuditorArtifactRef references persisted scan artifacts
type ScenarioAuditorArtifactRef struct {
	Path      string `json:"path"`
	Checksum  string `json:"checksum,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// ScenarioAuditorViolationExcerpt mirrors the summary payload for violations
type ScenarioAuditorViolationExcerpt struct {
	ID             string `json:"id"`
	Severity       string `json:"severity"`
	RuleID         string `json:"rule_id,omitempty"`
	Title          string `json:"title,omitempty"`
	FilePath       string `json:"file_path,omitempty"`
	LineNumber     int    `json:"line_number,omitempty"`
	Scenario       string `json:"scenario,omitempty"`
	Source         string `json:"source,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// ScenarioAuditorSummary captures the actionable digest from scenario-auditor
type ScenarioAuditorSummary struct {
	Total            int                               `json:"total"`
	BySeverity       map[string]int                    `json:"by_severity"`
	ByRule           []map[string]any                  `json:"by_rule,omitempty"`
	HighestSeverity  string                            `json:"highest_severity"`
	TopViolations    []ScenarioAuditorViolationExcerpt `json:"top_violations"`
	Artifact         *ScenarioAuditorArtifactRef       `json:"artifact,omitempty"`
	RecommendedSteps []string                          `json:"recommended_steps,omitempty"`
	GeneratedAt      string                            `json:"generated_at"`
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

// =============================================================================
// Issue Reporting Types
// =============================================================================

type IssueReportRequest struct {
	AppID                     string                  `json:"-"`
	Message                   string                  `json:"message"`
	Targets                   []IssueTarget           `json:"targets"`
	IncludeScreenshot         *bool                   `json:"includeScreenshot"`
	PreviewURL                *string                 `json:"previewUrl"`
	AppName                   *string                 `json:"appName"`
	ScenarioName              *string                 `json:"scenarioName"`
	Source                    *string                 `json:"source"`
	ScreenshotData            *string                 `json:"screenshotData"`
	Captures                  []IssueCapture          `json:"captures"`
	Logs                      []string                `json:"logs"`
	LogsTotal                 *int                    `json:"logsTotal"`
	LogsCapturedAt            *string                 `json:"logsCapturedAt"`
	ConsoleLogs               []IssueConsoleLogEntry  `json:"consoleLogs"`
	ConsoleLogsTotal          *int                    `json:"consoleLogsTotal"`
	ConsoleCapturedAt         *string                 `json:"consoleLogsCapturedAt"`
	NetworkRequests           []IssueNetworkEntry     `json:"networkRequests"`
	NetworkTotal              *int                    `json:"networkRequestsTotal"`
	NetworkCapturedAt         *string                 `json:"networkCapturedAt"`
	HealthChecks              []IssueHealthCheckEntry `json:"healthChecks"`
	HealthChecksTotal         *int                    `json:"healthChecksTotal"`
	HealthChecksCapturedAt    *string                 `json:"healthChecksCapturedAt"`
	AppStatusLines            []string                `json:"appStatusLines"`
	AppStatusLabel            *string                 `json:"appStatusLabel"`
	AppStatusSeverity         *string                 `json:"appStatusSeverity"`
	AppStatusCapturedAt       *string                 `json:"appStatusCapturedAt"`
	PrimaryDescription        *string                 `json:"primaryDescription"`
	IncludeDiagnosticsSummary *bool                   `json:"includeDiagnosticsSummary"`
}

type IssueTarget struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type IssueConsoleLogEntry struct {
	Timestamp int64  `json:"ts"`
	Level     string `json:"level"`
	Source    string `json:"source"`
	Text      string `json:"text"`
}

type IssueNetworkEntry struct {
	Timestamp  int64  `json:"ts"`
	Kind       string `json:"kind"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	Status     *int   `json:"status,omitempty"`
	OK         *bool  `json:"ok,omitempty"`
	DurationMs *int   `json:"durationMs,omitempty"`
	Error      string `json:"error,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
}

type IssueCapture struct {
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	Width       int              `json:"width"`
	Height      int              `json:"height"`
	Data        string           `json:"data"`
	Note        string           `json:"note,omitempty"`
	Selector    string           `json:"selector,omitempty"`
	TagName     string           `json:"tagName,omitempty"`
	ElementID   string           `json:"elementId,omitempty"`
	Classes     []string         `json:"classes,omitempty"`
	Label       string           `json:"label,omitempty"`
	AriaDesc    string           `json:"ariaDescription,omitempty"`
	Title       string           `json:"title,omitempty"`
	Role        string           `json:"role,omitempty"`
	Text        string           `json:"text,omitempty"`
	BoundingBox *IssueCaptureBox `json:"boundingBox,omitempty"`
	Clip        *IssueCaptureBox `json:"clip,omitempty"`
	Mode        string           `json:"mode,omitempty"`
	Filename    string           `json:"filename,omitempty"`
	CreatedAt   string           `json:"createdAt,omitempty"`
}

type IssueCaptureBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// IssueReportResult represents the outcome of creating a Swarm Manager fix item.
type IssueReportResult struct {
	Kind    string
	Name    string
	URL     string
	Message string
}

// =============================================================================
// Completeness Score Types
// =============================================================================

// CompletenessResponse represents the output from
// `vrooli scenario completeness score get <name> --json`. This data is owned by
// the scenario-completeness-scoring scenario (which has its own proto contract);
// app-monitor only needs the composite score + classification, so this maps that
// subset. Full typing belongs to that scenario's proto, not vrooli.cli.v1.
type CompletenessResponse struct {
	Scenario        string                       `json:"scenario"`
	Category        string                       `json:"category"`
	Composite       CompletenessComposite        `json:"composite"`
	Recommendations []CompletenessRecommendation `json:"recommendations,omitempty"`
}

// CompletenessComposite is the rolled-up score block.
type CompletenessComposite struct {
	Score          int    `json:"score"`
	Classification string `json:"classification"`
}

// CompletenessRecommendation is one prioritized improvement suggestion.
type CompletenessRecommendation struct {
	Priority     string  `json:"priority"`
	Description  string  `json:"description"`
	ImpactPoints float64 `json:"impact_points"`
}

// CompletenessScore represents human-readable completeness output for display
type CompletenessScore struct {
	Scenario string   `json:"scenario"`
	Details  []string `json:"details"`
}

// completenessCache caches completeness scores to prevent excessive recalculation
type completenessCache struct {
	data      map[string]*CompletenessResponse // key: scenario name
	timestamp time.Time
	mu        sync.RWMutex
	loading   bool
}
