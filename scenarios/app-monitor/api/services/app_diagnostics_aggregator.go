package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"app-monitor-api/logger"
	"app-monitor-api/repository"
)

// =============================================================================
// Aggregated Diagnostics Types
// =============================================================================

// DiagnosticSeverity represents the overall severity level of diagnostics
type DiagnosticSeverity string

const (
	DiagnosticSeverityOK       DiagnosticSeverity = "ok"
	DiagnosticSeverityWarn     DiagnosticSeverity = "warn"
	DiagnosticSeverityError    DiagnosticSeverity = "error"
	DiagnosticSeverityUnknown  DiagnosticSeverity = "unknown"
	DiagnosticSeverityDegraded DiagnosticSeverity = "degraded"
)

// DiagnosticWarning represents a single warning from any diagnostic source
type DiagnosticWarning struct {
	Source   string `json:"source"`              // "health", "bridge", "localhost", "issues", "status"
	Severity string `json:"severity"`            // "warn", "error", "info"
	Message  string `json:"message"`             // Human-readable warning
	FilePath string `json:"file_path,omitempty"` // Optional file path for code-related warnings
	Line     int    `json:"line,omitempty"`      // Optional line number
}

// TechStackInfo contains parsed technology stack information
type TechStackInfo struct {
	Runtime      string                 `json:"runtime,omitempty"`      // "Go", "Node.js", etc.
	Processes    int                    `json:"processes"`              // Number of running processes
	Resources    []TechStackResource    `json:"resources,omitempty"`    // postgres, redis, ollama
	Tags         []string               `json:"tags,omitempty"`         // From service.json
	Dependencies map[string]interface{} `json:"dependencies,omitempty"` // Additional dependency info
	Ports        map[string]int         `json:"ports,omitempty"`        // Port allocations
}

// TechStackResource represents a single resource dependency
type TechStackResource struct {
	Type     string `json:"type"`              // "postgres", "redis", "ollama", etc.
	Enabled  bool   `json:"enabled"`           // Is this resource enabled?
	Required bool   `json:"required"`          // Is this resource required?
	Purpose  string `json:"purpose,omitempty"` // Why this resource is used
}

// AppDocumentsList contains available documentation files
type AppDocumentsList struct {
	RootDocs []AppDocumentInfo `json:"root_docs"` // README.md, PRD.md, etc. in scenario root
	DocsDocs []AppDocumentInfo `json:"docs_docs"` // Files in docs/ folder
	Total    int               `json:"total"`     // Total document count
}

// AppDocumentInfo represents metadata about a single document
type AppDocumentInfo struct {
	Name       string `json:"name"`        // Filename
	Path       string `json:"path"`        // Relative path from scenario root
	Size       int64  `json:"size"`        // File size in bytes
	IsMarkdown bool   `json:"is_markdown"` // Whether this is a markdown file
	ModifiedAt string `json:"modified_at"` // ISO 8601 timestamp
}

// CompleteDiagnostics aggregates all diagnostic information for an app
type CompleteDiagnostics struct {
	AppID      string    `json:"app_id"`
	Scenario   string    `json:"scenario"`
	CapturedAt time.Time `json:"captured_at"`

	// Status & Health
	ScenarioStatus *AppScenarioStatus    `json:"scenario_status,omitempty"`
	HealthChecks   *AppHealthDiagnostics `json:"health_checks,omitempty"`

	// Issues
	Issues *AppIssuesSummary `json:"issues,omitempty"`

	// Compliance
	BridgeRules    *BridgeDiagnosticsReport `json:"bridge_rules,omitempty"`
	LocalhostUsage *LocalhostUsageReport    `json:"localhost_usage,omitempty"`
	AuditorSummary *ScenarioAuditorSummary  `json:"auditor_summary,omitempty"`

	// Metadata
	TechStack *TechStackInfo    `json:"tech_stack,omitempty"`
	Documents *AppDocumentsList `json:"documents,omitempty"`

	// Aggregated Summary
	Warnings []DiagnosticWarning `json:"warnings"`          // All warnings from all sources
	Severity DiagnosticSeverity  `json:"severity"`          // Overall severity
	Summary  string              `json:"summary,omitempty"` // Human-readable summary
}

// DiagnosticOptions controls which diagnostics to fetch
type DiagnosticOptions struct {
	IncludeHealth         bool `json:"include_health"`
	IncludeIssues         bool `json:"include_issues"`
	IncludeBridgeRules    bool `json:"include_bridge_rules"`
	IncludeLocalhostScan  bool `json:"include_localhost_scan"`
	IncludeTechStack      bool `json:"include_tech_stack"`
	IncludeDocuments      bool `json:"include_documents"`
	IncludeStatus         bool `json:"include_status"`
	IncludeAuditorSummary bool `json:"include_auditor_summary"`
}

// DefaultDiagnosticOptions returns options with all diagnostics enabled
func DefaultDiagnosticOptions() DiagnosticOptions {
	return DiagnosticOptions{
		IncludeHealth:         true,
		IncludeIssues:         true,
		IncludeBridgeRules:    true,
		IncludeLocalhostScan:  true,
		IncludeTechStack:      true,
		IncludeDocuments:      true,
		IncludeStatus:         true,
		IncludeAuditorSummary: true,
	}
}

// FastDiagnosticOptions returns options for quick checks (health + status only)
func FastDiagnosticOptions() DiagnosticOptions {
	return DiagnosticOptions{
		IncludeHealth:         true,
		IncludeStatus:         true,
		IncludeTechStack:      true,
		IncludeAuditorSummary: false,
	}
}

// =============================================================================
// Main Aggregation Method
// =============================================================================

// GetCompleteDiagnostics fetches and aggregates all diagnostic information for an app.
// This is the primary method that should be used by UI components needing comprehensive diagnostics.
func (s *AppService) GetCompleteDiagnostics(ctx context.Context, appID string, opts DiagnosticOptions) (*CompleteDiagnostics, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	// First, get the app to validate it exists and get basic info
	app, err := s.GetApp(ctx, id)
	if err != nil {
		return nil, err
	}

	scenarioName := strings.TrimSpace(app.ScenarioName)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(app.ID)
	}
	if scenarioName == "" {
		scenarioName = id
	}

	capturedAt := s.timeNow().UTC()
	result := &CompleteDiagnostics{
		AppID:      app.ID,
		Scenario:   scenarioName,
		CapturedAt: capturedAt,
		Warnings:   make([]DiagnosticWarning, 0),
		Severity:   DiagnosticSeverityUnknown,
	}

	// Use goroutines to fetch diagnostics in parallel for better performance
	var wg sync.WaitGroup
	var mu sync.Mutex // Protects concurrent access to result

	// Fetch scenario status (if requested)
	if opts.IncludeStatus {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := s.GetAppScenarioStatus(ctx, id)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to fetch scenario status for %s", id), err)
				mu.Lock()
				result.Warnings = append(result.Warnings, DiagnosticWarning{
					Source:   "status",
					Severity: "warn",
					Message:  fmt.Sprintf("Failed to fetch scenario status: %v", err),
				})
				mu.Unlock()
			} else {
				mu.Lock()
				result.ScenarioStatus = status
				mu.Unlock()
			}
		}()
	}

	// Fetch health checks (if requested)
	if opts.IncludeHealth {
		wg.Add(1)
		go func() {
			defer wg.Done()
			health, err := s.CheckAppHealth(ctx, id)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to fetch health checks for %s", id), err)
				mu.Lock()
				result.Warnings = append(result.Warnings, DiagnosticWarning{
					Source:   "health",
					Severity: "warn",
					Message:  fmt.Sprintf("Failed to fetch health checks: %v", err),
				})
				mu.Unlock()
			} else {
				mu.Lock()
				result.HealthChecks = health
				mu.Unlock()
			}
		}()
	}

	// Fetch issues (if requested)
	if opts.IncludeIssues {
		wg.Add(1)
		go func() {
			defer wg.Done()
			issues, err := s.ListScenarioIssues(ctx, id)
			if err != nil {
				// Issues are non-critical, so just log the warning
				logger.Warn(fmt.Sprintf("failed to fetch issues for %s", id), err)
			} else {
				mu.Lock()
				result.Issues = issues
				mu.Unlock()
			}
		}()
	}

	// Fetch bridge rules (if requested)
	if opts.IncludeBridgeRules {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bridge, err := s.CheckIframeBridgeRule(ctx, id)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to fetch bridge rules for %s", id), err)
				mu.Lock()
				result.Warnings = append(result.Warnings, DiagnosticWarning{
					Source:   "bridge",
					Severity: "warn",
					Message:  fmt.Sprintf("Failed to check iframe bridge rules: %v", err),
				})
				mu.Unlock()
			} else {
				mu.Lock()
				result.BridgeRules = bridge
				mu.Unlock()
			}
		}()
	}

	// Fetch localhost scan (if requested)
	if opts.IncludeLocalhostScan {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localhost, err := s.CheckLocalhostUsage(ctx, id)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to scan localhost usage for %s", id), err)
				mu.Lock()
				result.Warnings = append(result.Warnings, DiagnosticWarning{
					Source:   "localhost",
					Severity: "warn",
					Message:  fmt.Sprintf("Failed to scan localhost usage: %v", err),
				})
				mu.Unlock()
			} else {
				mu.Lock()
				result.LocalhostUsage = localhost
				mu.Unlock()
			}
		}()
	}

	// Fetch tech stack (if requested)
	if opts.IncludeTechStack {
		wg.Add(1)
		go func() {
			defer wg.Done()
			techStack := s.extractTechStack(app, result.ScenarioStatus)
			mu.Lock()
			result.TechStack = techStack
			mu.Unlock()
		}()
	}

	// Fetch documents list (if requested)
	if opts.IncludeDocuments {
		wg.Add(1)
		go func() {
			defer wg.Done()
			docs, err := s.ListAppDocuments(ctx, id)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to list documents for %s", id), err)
			} else {
				mu.Lock()
				result.Documents = docs
				mu.Unlock()
			}
		}()
	}

	// Fetch scenario-auditor summary (if requested)
	if opts.IncludeAuditorSummary {
		wg.Add(1)
		go func() {
			defer wg.Done()
			summary, err := s.fetchScenarioAuditorSummary(ctx, scenarioName)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to fetch scenario-auditor summary for %s", id), err)
				mu.Lock()
				result.Warnings = append(result.Warnings, DiagnosticWarning{
					Source:   "scenario-auditor",
					Severity: "info",
					Message:  fmt.Sprintf("Scenario-auditor summary unavailable: %v", err),
				})
				mu.Unlock()
				return
			}
			if summary == nil {
				return
			}
			mu.Lock()
			result.AuditorSummary = summary
			mu.Unlock()
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Aggregate warnings from all sources
	result.Warnings = s.aggregateWarnings(result)

	// Determine overall severity
	result.Severity = s.calculateOverallSeverity(result)

	// Generate summary
	result.Summary = s.generateDiagnosticSummary(result)

	return result, nil
}

// =============================================================================
// Tech Stack Extraction
// =============================================================================

// extractTechStack extracts technology stack information from app and status data
func (s *AppService) extractTechStack(app *repository.App, status *AppScenarioStatus) *TechStackInfo {
	if app == nil {
		return nil
	}

	techStack := &TechStackInfo{
		Resources:    make([]TechStackResource, 0),
		Tags:         dedupeStrings(app.Tags),
		Dependencies: make(map[string]interface{}),
		Ports:        make(map[string]int),
	}

	// Extract runtime
	if app.Runtime != "" {
		techStack.Runtime = app.Runtime
	} else if status != nil && status.Runtime != "" {
		techStack.Runtime = status.Runtime
	}

	// Extract process count
	if status != nil {
		techStack.Processes = status.ProcessCount
	} else if count, ok := app.Config["process_count"].(int); ok {
		techStack.Processes = count
	}

	// Extract ports
	if status != nil && len(status.Ports) > 0 {
		for k, v := range status.Ports {
			techStack.Ports[k] = v
		}
	} else if len(app.PortMappings) > 0 {
		for k, v := range app.PortMappings {
			if port, err := parsePortFromAny(v); err == nil {
				techStack.Ports[k] = port
			}
		}
	}

	// Extract resources from tags (postgres, redis, ollama, etc.)
	resourceTypes := []string{"postgres", "redis", "ollama", "qdrant", "n8n", "browserless", "vault"}
	for _, tag := range app.Tags {
		normalized := normalizeLower(tag)
		for _, resType := range resourceTypes {
			if strings.Contains(normalized, resType) {
				techStack.Resources = append(techStack.Resources, TechStackResource{
					Type:    resType,
					Enabled: true,
					// We don't have required/purpose info without parsing service.json
				})
				break
			}
		}
	}

	// Dedupe resources
	seen := make(map[string]bool)
	uniqueResources := make([]TechStackResource, 0, len(techStack.Resources))
	for _, res := range techStack.Resources {
		if !seen[res.Type] {
			seen[res.Type] = true
			uniqueResources = append(uniqueResources, res)
		}
	}
	techStack.Resources = uniqueResources

	return techStack
}

// =============================================================================
// Warning Aggregation
// =============================================================================

// aggregateWarnings collects warnings from all diagnostic sources
func (s *AppService) aggregateWarnings(diag *CompleteDiagnostics) []DiagnosticWarning {
	warnings := make([]DiagnosticWarning, 0)

	// Add any existing warnings (from fetch failures)
	warnings = append(warnings, diag.Warnings...)

	// Health check warnings
	if diag.HealthChecks != nil {
		for _, check := range diag.HealthChecks.Checks {
			if check.Status != "pass" {
				severity := "warn"
				if check.Status == "fail" {
					severity = "error"
				}
				warnings = append(warnings, DiagnosticWarning{
					Source:   "health",
					Severity: severity,
					Message:  fmt.Sprintf("%s: %s", check.Name, check.Message),
				})
			}
		}
		for _, err := range diag.HealthChecks.Errors {
			warnings = append(warnings, DiagnosticWarning{
				Source:   "health",
				Severity: "error",
				Message:  err,
			})
		}
	}

	// Bridge rule violations
	if diag.BridgeRules != nil {
		for _, violation := range diag.BridgeRules.Violations {
			severity := "warn"
			if normalizeLower(violation.Severity) == "error" || normalizeLower(violation.Severity) == "critical" {
				severity = "error"
			}
			warnings = append(warnings, DiagnosticWarning{
				Source:   "bridge",
				Severity: severity,
				Message:  violation.Title,
				FilePath: violation.FilePath,
				Line:     violation.Line,
			})
		}
		for _, warn := range diag.BridgeRules.Warnings {
			warnings = append(warnings, DiagnosticWarning{
				Source:   "bridge",
				Severity: "warn",
				Message:  warn,
			})
		}
	}

	// Localhost usage findings
	if diag.LocalhostUsage != nil && len(diag.LocalhostUsage.Findings) > 0 {
		count := len(diag.LocalhostUsage.Findings)
		warnings = append(warnings, DiagnosticWarning{
			Source:   "localhost",
			Severity: "warn",
			Message:  fmt.Sprintf("Found %d hard-coded localhost reference%s", count, plural(count)),
		})
		for _, warn := range diag.LocalhostUsage.Warnings {
			warnings = append(warnings, DiagnosticWarning{
				Source:   "localhost",
				Severity: "warn",
				Message:  warn,
			})
		}
	}

	// Scenario status recommendations
	if diag.ScenarioStatus != nil {
		for _, rec := range diag.ScenarioStatus.Recommendations {
			warnings = append(warnings, DiagnosticWarning{
				Source:   "status",
				Severity: "info",
				Message:  rec,
			})
		}
	}

	// Scenario-auditor summary
	if diag.AuditorSummary != nil && diag.AuditorSummary.Total > 0 {
		sev := strings.ToLower(diag.AuditorSummary.HighestSeverity)
		severity := "info"
		if sev == "critical" {
			severity = "error"
		} else if sev == "high" {
			severity = "warn"
		}
		warnings = append(warnings, DiagnosticWarning{
			Source:   "scenario-auditor",
			Severity: severity,
			Message:  fmt.Sprintf("%d standards/security violation%s (max severity %s)", diag.AuditorSummary.Total, plural(diag.AuditorSummary.Total), diag.AuditorSummary.HighestSeverity),
		})
	}

	// Issue tracker warnings
	if diag.Issues != nil && diag.Issues.OpenCount > 0 {
		warnings = append(warnings, DiagnosticWarning{
			Source:   "issues",
			Severity: "info",
			Message:  fmt.Sprintf("%d open issue%s", diag.Issues.OpenCount, plural(diag.Issues.OpenCount)),
		})
	}

	return warnings
}

// =============================================================================
// Severity Calculation
// =============================================================================

// calculateOverallSeverity determines the overall severity based on all diagnostics
func (s *AppService) calculateOverallSeverity(diag *CompleteDiagnostics) DiagnosticSeverity {
	hasError := false
	hasWarn := false

	// Check warnings
	for _, warn := range diag.Warnings {
		if warn.Severity == "error" {
			hasError = true
		} else if warn.Severity == "warn" {
			hasWarn = true
		}
	}

	// Check scenario status
	if diag.ScenarioStatus != nil {
		switch diag.ScenarioStatus.Severity {
		case ScenarioStatusSeverityError:
			hasError = true
		case ScenarioStatusSeverityWarn:
			hasWarn = true
		}
	}

	// Scenario-auditor summary severity
	if diag.AuditorSummary != nil && diag.AuditorSummary.Total > 0 {
		switch strings.ToLower(diag.AuditorSummary.HighestSeverity) {
		case "critical":
			hasError = true
		case "high":
			hasWarn = true
		default:
			hasWarn = true
		}
	}

	// Check health checks
	if diag.HealthChecks != nil {
		for _, check := range diag.HealthChecks.Checks {
			if check.Status == "fail" {
				hasError = true
			} else if check.Status == "warn" {
				hasWarn = true
			}
		}
	}

	// Determine overall severity
	if hasError {
		return DiagnosticSeverityError
	}
	if hasWarn {
		return DiagnosticSeverityWarn
	}
	return DiagnosticSeverityOK
}

// =============================================================================
// Summary Generation
// =============================================================================

// generateDiagnosticSummary creates a human-readable summary of diagnostics
func (s *AppService) generateDiagnosticSummary(diag *CompleteDiagnostics) string {
	parts := make([]string, 0)

	// Status summary
	if diag.ScenarioStatus != nil {
		parts = append(parts, fmt.Sprintf("Status: %s", diag.ScenarioStatus.StatusLabel))
	}

	// Health summary
	if diag.HealthChecks != nil {
		passCount := 0
		failCount := 0
		for _, check := range diag.HealthChecks.Checks {
			if check.Status == "pass" {
				passCount++
			} else {
				failCount++
			}
		}
		if failCount > 0 {
			parts = append(parts, fmt.Sprintf("%d health check%s failing", failCount, plural(failCount)))
		} else if passCount > 0 {
			parts = append(parts, fmt.Sprintf("All %d health check%s passing", passCount, plural(passCount)))
		}
	}

	// Issues summary
	if diag.Issues != nil && diag.Issues.OpenCount > 0 {
		parts = append(parts, fmt.Sprintf("%d open issue%s", diag.Issues.OpenCount, plural(diag.Issues.OpenCount)))
	}

	// Compliance summary
	violationCount := 0
	if diag.BridgeRules != nil {
		violationCount += len(diag.BridgeRules.Violations)
	}
	if diag.LocalhostUsage != nil {
		violationCount += len(diag.LocalhostUsage.Findings)
	}
	if diag.AuditorSummary != nil {
		violationCount += diag.AuditorSummary.Total
	}
	if violationCount > 0 {
		parts = append(parts, fmt.Sprintf("%d compliance finding%s", violationCount, plural(violationCount)))
	}
	if diag.AuditorSummary != nil && diag.AuditorSummary.Total > 0 {
		parts = append(parts, fmt.Sprintf("scenario-auditor: %d issue%s (max %s)", diag.AuditorSummary.Total, plural(diag.AuditorSummary.Total), diag.AuditorSummary.HighestSeverity))
	}

	if len(parts) == 0 {
		return "No issues detected"
	}

	return strings.Join(parts, " • ")
}

type scenarioAuditorSummaryResponse struct {
	Summary *ScenarioAuditorSummary `json:"summary"`
}

func (s *AppService) fetchScenarioAuditorSummary(ctx context.Context, scenarioName string) (*ScenarioAuditorSummary, error) {
	baseURL, err := s.scenarioAuditorBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	values.Set("scenario", scenarioName)
	values.Set("limit", "20")
	requestURL := fmt.Sprintf("%s/api/v1/standards/violations/summary?%s", baseURL, values.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("scenario not found in scenario-auditor")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("scenario-auditor summary returned status %d", resp.StatusCode)
	}

	var payload scenarioAuditorSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return payload.Summary, nil
}

func (s *AppService) scenarioAuditorBaseURL(ctx context.Context) (string, error) {
	if override := strings.TrimSpace(os.Getenv("SCENARIO_AUDITOR_API_BASE_URL")); override != "" {
		return override, nil
	}
	port, err := s.locateScenarioAuditorAPIPort(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://localhost:%d", port), nil
}
