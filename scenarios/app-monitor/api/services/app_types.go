package services

import (
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
	ErrIssueTrackerUnavailable       = errors.New("app-issue-tracker unavailable")
)

// Cache and timing constants
const (
	orchestratorCacheTTL   = 90 * time.Second // Increased from 60s to reduce cache misses during slow scenario status calls
	partialCacheTTL        = 45 * time.Second // Increased proportionally
	issueTrackerCacheTTL   = 30 * time.Second
	issueTrackerFetchLimit = 50
)

// Issue attachment constants
const (
	attachmentLifecycleName  = "lifecycle.txt"
	attachmentConsoleName    = "console.json"
	attachmentNetworkName    = "network.json"
	attachmentScreenshotName = "screenshot.png"
	attachmentHealthName     = "health.json"
	attachmentStatusName     = "status.txt"
	issueTrackerScenarioID   = "app-issue-tracker"
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

// issueCacheEntry stores cached issue information
type issueCacheEntry struct {
	issues      []AppIssueSummary
	scenario    string
	appID       string
	trackerURL  string
	fetchedAt   time.Time
	openCount   int
	activeCount int
	totalCount  int
}

// AppService handles business logic for application management
type AppService struct {
	repo          repository.AppRepository
	httpClient    HTTPClient
	timeNow       TimeProvider
	cache         *orchestratorCache
	viewStatsMu   sync.RWMutex
	viewStats     map[string]*viewStatsEntry
	issueCacheMu  sync.RWMutex
	issueCache    map[string]*issueCacheEntry
	issueCacheTTL time.Duration
	repoRoot      string
}

// =============================================================================
// Orchestrator Types
// =============================================================================

// OrchestratorResponse represents the response from vrooli scenario status --json
type OrchestratorResponse struct {
	Success bool `json:"success"`
	Summary struct {
		TotalScenarios int `json:"total_scenarios"`
		Running        int `json:"running"`
		Stopped        int `json:"stopped"`
	} `json:"summary"`
	Scenarios []OrchestratorApp `json:"scenarios"`
}

// ecosystemManagerResponse represents the direct API response from ecosystem-manager
type ecosystemManagerResponse struct {
	Success bool              `json:"success"`
	Data    []OrchestratorApp `json:"data"`
}

// OrchestratorApp represents an app from the orchestrator
type OrchestratorApp struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	HealthStatus *string        `json:"health_status"`
	Ports        map[string]int `json:"ports"`
	Processes    int            `json:"processes"`
	Runtime      string         `json:"runtime"`
	StartedAt    string         `json:"started_at,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
}

// scenarioListResponse represents the output from `vrooli scenario list --json`
type scenarioListResponse struct {
	Success   bool               `json:"success"`
	Scenarios []scenarioMetadata `json:"scenarios"`
}

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
// Issue Tracking Types
// =============================================================================

// AppIssueSummary represents a simplified issue entry from app-issue-tracker.
type AppIssueSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Priority  string `json:"priority,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Reporter  string `json:"reporter,omitempty"`
	IssueURL  string `json:"issue_url,omitempty"`
}

// AppIssuesSummary provides aggregated issue information for an app/scenario.
type AppIssuesSummary struct {
	Scenario    string            `json:"scenario"`
	AppID       string            `json:"app_id"`
	Issues      []AppIssueSummary `json:"issues"`
	OpenCount   int               `json:"open_count"`
	ActiveCount int               `json:"active_count"`
	TotalCount  int               `json:"total_count"`
	TrackerURL  string            `json:"tracker_url,omitempty"`
	LastFetched string            `json:"last_fetched"`
	FromCache   bool              `json:"from_cache"`
	Stale       bool              `json:"stale"`
}

type issueTrackerAPIResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Error   string                 `json:"error"`
	Data    map[string]interface{} `json:"data"`
}

// =============================================================================
// Diagnostics Types
// =============================================================================
// NOTE: Diagnostic types have been moved to app_diagnostics_types.go for better organization.
// This separation makes it easier to work with diagnostic features and improves maintainability.
//
// See app_diagnostics_types.go for:
// - ScenarioStatusSeverity, AppScenarioStatus (scenario status diagnostics)
// - AppHealthDiagnostics, IssueHealthCheckEntry (health checks)
// - BridgeRuleViolation, BridgeDiagnosticsReport (iframe bridge rules)
// - LocalhostUsageFinding, LocalhostUsageReport (localhost scanning)

// =============================================================================
// Issue Reporting Types
// =============================================================================

type IssueReportRequest struct {
	AppID                     string                  `json:"-"`
	Message                   string                  `json:"message"`
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

// IssueReportResult represents the outcome of forwarding an issue report
type IssueReportResult struct {
	IssueID  string
	Message  string
	IssueURL string
}
