package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"app-monitor-api/logger"
)

// BrowserlessConsoleLog represents a console log entry captured by browserless
type BrowserlessConsoleLog struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// BrowserlessNetworkRequest represents a network request captured by browserless
type BrowserlessNetworkRequest struct {
	RequestID    string            `json:"requestId"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	ResourceType string            `json:"resourceType"`
	Status       int               `json:"status,omitempty"`
	OK           bool              `json:"ok,omitempty"`
	StatusText   string            `json:"statusText,omitempty"`
	Duration     int64             `json:"duration,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	ContentType  string            `json:"contentType,omitempty"`
	Failed       bool              `json:"failed,omitempty"`
	Error        string            `json:"error,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// BrowserlessPageStatus represents the health/status of a page
type BrowserlessPageStatus struct {
	WhiteScreen        bool                   `json:"whiteScreen"`
	HTTPError          *HTTPErrorDetails      `json:"httpError,omitempty"`
	LoadError          string                 `json:"loadError,omitempty"`
	ModuleError        string                 `json:"moduleError,omitempty"`
	CloudflareError    bool                   `json:"cloudflareError"`
	NotFoundError      bool                   `json:"notFoundError"`
	EmptyBody          bool                   `json:"emptyBody"`
	ResourceCount      int                    `json:"resourceCount"`
	LoadTimeMs         float64                `json:"loadTimeMs"`
	PerformanceMetrics map[string]interface{} `json:"performanceMetrics,omitempty"`
	DetectedIssues     []string               `json:"detectedIssues,omitempty"`
}

// HTTPErrorDetails contains HTTP error information
type HTTPErrorDetails struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// BrowserlessFallbackResult aggregates all fallback diagnostics
type BrowserlessFallbackResult struct {
	ConsoleLogs     []BrowserlessConsoleLog     `json:"consoleLogs"`
	NetworkRequests []BrowserlessNetworkRequest `json:"networkRequests"`
	PageStatus      *BrowserlessPageStatus      `json:"pageStatus"`
	Screenshot      string                      `json:"screenshot,omitempty"`
	HTML            string                      `json:"html,omitempty"`
	Source          string                      `json:"source"`
	CapturedAt      time.Time                   `json:"capturedAt"`
	URL             string                      `json:"url"`
	Title           string                      `json:"title,omitempty"`
}

// BrowserlessService handles browserless command execution for fallback diagnostics
type BrowserlessService struct {
	timeout       time.Duration
	cacheMutex    sync.RWMutex
	cache         map[string]*cacheEntry
	cacheTTL      time.Duration
	healthTimeout time.Duration
}

type cacheEntry struct {
	result    *BrowserlessFallbackResult
	expiresAt time.Time
}

// NewBrowserlessService creates a new browserless service
func NewBrowserlessService() *BrowserlessService {
	return &BrowserlessService{
		timeout:       20 * time.Second, // Overall timeout for all parallel diagnostics (4 commands @ 15s each run in parallel)
		cache:         make(map[string]*cacheEntry),
		cacheTTL:      30 * time.Second, // Cache results for 30 seconds
		healthTimeout: 5 * time.Second,  // Quick health check timeout
	}
}

// GetFallbackDiagnostics retrieves console logs, network requests, and page status using browserless
func (s *BrowserlessService) GetFallbackDiagnostics(ctx context.Context, url string) (*BrowserlessFallbackResult, error) {
	// Check browserless health first
	if err := s.checkHealth(ctx); err != nil {
		return nil, fmt.Errorf("browserless service is not available: %w (try: resource-browserless manage restart)", err)
	}

	// Check cache first
	cacheKey := fmt.Sprintf("fallback:%s", url)
	s.cacheMutex.RLock()
	if entry, exists := s.cache[cacheKey]; exists && time.Now().Before(entry.expiresAt) {
		cached := *entry.result
		s.cacheMutex.RUnlock()
		logger.Info("Returning cached fallback diagnostics for %s", url)
		return &cached, nil
	}
	s.cacheMutex.RUnlock()

	logger.Info("Collecting fallback diagnostics for %s", url)

	// Use the new single-command diagnostics that collects everything in one browser session
	// This is more reliable and faster than running 4 parallel commands
	result, err := s.getDiagnostics(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to collect diagnostics: %w", err)
	}

	// Update cache even with partial results
	s.cacheMutex.Lock()
	s.cache[cacheKey] = &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMutex.Unlock()

	return result, nil
}

// getDiagnostics executes the single browserless diagnostics command that collects all data in one session
func (s *BrowserlessService) getDiagnostics(ctx context.Context, url string) (*BrowserlessFallbackResult, error) {
	// Use aggressive timeout for broken pages - 15 seconds total
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Execute the combined diagnostics command
	// This collects console logs, network requests, performance, and HTML in a single browser session
	cmd := exec.CommandContext(ctx, "resource-browserless", "diagnostics", url, "--timeout", "8000", "--wait-ms", "1000")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("diagnostics command failed: %w", err)
	}

	// Extract JSON from the output (skip INFO lines)
	jsonOutput := extractJSONFromOutput(output)
	if jsonOutput == nil {
		return nil, fmt.Errorf("no valid JSON found in diagnostics output")
	}

	// Parse the combined diagnostics response
	var diagnostics struct {
		ConsoleLogs     []BrowserlessConsoleLog     `json:"consoleLogs"`
		NetworkRequests []BrowserlessNetworkRequest `json:"networkRequests"`
		Performance     map[string]interface{}      `json:"performance"`
		HTML            string                      `json:"html"`
		Title           string                      `json:"title"`
		URL             string                      `json:"url"`
	}

	if err := json.Unmarshal(jsonOutput, &diagnostics); err != nil {
		return nil, fmt.Errorf("failed to parse diagnostics output: %w", err)
	}

	// Build the result from the diagnostics data
	result := &BrowserlessFallbackResult{
		URL:             url,
		Source:          "browserless",
		CapturedAt:      time.Now(),
		ConsoleLogs:     diagnostics.ConsoleLogs,
		NetworkRequests: diagnostics.NetworkRequests,
		HTML:            diagnostics.HTML,
		Title:           diagnostics.Title,
	}

	// Build page status from performance metrics
	status := &BrowserlessPageStatus{
		PerformanceMetrics: diagnostics.Performance,
	}

	// Extract numeric values from performance
	if totalLoadTime, ok := diagnostics.Performance["totalLoadTime"].(float64); ok {
		status.LoadTimeMs = totalLoadTime
	}
	if resourceCount, ok := diagnostics.Performance["resourceCount"].(float64); ok {
		status.ResourceCount = int(resourceCount)
	}

	// White screen detection: fast load + very few resources
	if status.LoadTimeMs < 100 && status.ResourceCount < 5 {
		status.WhiteScreen = true
	}

	result.PageStatus = status

	// Analyze HTML for common error patterns
	if result.HTML != "" && result.PageStatus != nil {
		s.analyzeHTMLForErrors(result.HTML, result.PageStatus, result.ConsoleLogs)
	}

	return result, nil
}

// analyzeHTMLForErrors analyzes HTML content and console logs for common error patterns
func (s *BrowserlessService) analyzeHTMLForErrors(html string, status *BrowserlessPageStatus, consoleLogs []BrowserlessConsoleLog) {
	if status == nil {
		return
	}

	htmlLower := strings.ToLower(html)
	var issues []string

	// Check for 404 Not Found errors
	if strings.Contains(htmlLower, "404") && (strings.Contains(htmlLower, "not found") || strings.Contains(htmlLower, "page not found")) {
		status.NotFoundError = true
		issues = append(issues, "404 Not Found - The page could not be found")
	}

	// Check for Cloudflare bad gateway / errors
	if strings.Contains(htmlLower, "cloudflare") {
		if strings.Contains(htmlLower, "bad gateway") || strings.Contains(htmlLower, "error 502") || strings.Contains(htmlLower, "error 503") {
			status.CloudflareError = true
			issues = append(issues, "Cloudflare gateway error detected (502/503)")
		} else if strings.Contains(htmlLower, "error 520") || strings.Contains(htmlLower, "error 521") || strings.Contains(htmlLower, "error 522") {
			status.CloudflareError = true
			issues = append(issues, "Cloudflare connection error detected (520/521/522)")
		}
	}

	// Check for empty body or minimal content (white screen)
	// Look for body tag with very little content
	bodyStart := strings.Index(htmlLower, "<body")
	bodyEnd := strings.Index(htmlLower, "</body>")
	if bodyStart != -1 && bodyEnd != -1 && bodyEnd > bodyStart {
		// Find the end of the opening body tag
		bodyContentStart := strings.Index(htmlLower[bodyStart:], ">")
		if bodyContentStart != -1 {
			bodyContentStart += bodyStart + 1
			bodyContent := strings.TrimSpace(html[bodyContentStart:bodyEnd])

			// Remove common empty elements
			bodyContent = strings.ReplaceAll(bodyContent, "<div></div>", "")
			bodyContent = strings.ReplaceAll(bodyContent, "<div />", "")
			bodyContent = strings.ReplaceAll(bodyContent, "<script", "") // Scripts don't count as visible content
			bodyContent = strings.ReplaceAll(bodyContent, "</script>", "")
			bodyContent = strings.TrimSpace(bodyContent)

			// If body is effectively empty or very small
			if len(bodyContent) < 50 {
				status.EmptyBody = true
				issues = append(issues, "Empty or minimal body content detected (possible white screen)")
			}
		}
	}

	// Check for module loading errors from console logs
	hasModuleError := false
	for _, log := range consoleLogs {
		if log.Level == "error" {
			logLower := strings.ToLower(log.Message)
			if strings.Contains(logLower, "failed to load module") ||
			   strings.Contains(logLower, "mime type") ||
			   strings.Contains(logLower, "unexpected token") && strings.Contains(logLower, "import") {
				hasModuleError = true
				if status.ModuleError == "" {
					status.ModuleError = log.Message
				}
			}
		}
	}
	if hasModuleError {
		issues = append(issues, "Module loading error detected - bundle may be misconfigured")
	}

	// Check for generic load errors from console logs
	hasLoadError := false
	for _, log := range consoleLogs {
		if log.Level == "error" {
			logLower := strings.ToLower(log.Message)
			if strings.Contains(logLower, "failed to fetch") ||
			   strings.Contains(logLower, "net::err") ||
			   strings.Contains(logLower, "connection refused") {
				hasLoadError = true
				if status.LoadError == "" {
					status.LoadError = log.Message
				}
			}
		}
	}
	if hasLoadError {
		issues = append(issues, "Network/loading error detected in console")
	}

	// Check for white screen based on performance metrics
	if status.WhiteScreen {
		issues = append(issues, "White screen detected - page loaded quickly with few resources and no paint events")
	}

	status.DetectedIssues = issues
}

// ClearCache clears the browserless result cache
func (s *BrowserlessService) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cache = make(map[string]*cacheEntry)
	logger.Info("Browserless cache cleared")
}

// checkHealth verifies that browserless is running and healthy before attempting diagnostics
func (s *BrowserlessService) checkHealth(ctx context.Context) error {
	healthCtx, cancel := context.WithTimeout(ctx, s.healthTimeout)
	defer cancel()

	cmd := exec.CommandContext(healthCtx, "resource-browserless", "status")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check browserless status: %w", err)
	}

	// Parse the status output to check if running and healthy
	statusOutput := string(output)

	// Check if browserless is running
	if !strings.Contains(statusOutput, "Running: Yes") {
		return fmt.Errorf("browserless is not running")
	}

	// Check if browserless is healthy
	if strings.Contains(statusOutput, "Health: Unhealthy") {
		return fmt.Errorf("browserless is running but unhealthy - may need restart")
	}

	return nil
}

// extractJSONFromOutput finds and extracts JSON content from command output
// that may contain INFO logs, summary text, or other non-JSON content.
// It looks for a line that starts with '{' or '[' on its own line (after a newline).
func extractJSONFromOutput(output []byte) []byte {
	if len(output) == 0 {
		return nil
	}

	lines := strings.Split(string(output), "\n")

	// Find the first line that starts with { or [ (actual JSON)
	// Skip lines like "[INFO]" which start with [ but aren't JSON
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}

		// Check if this looks like actual JSON start
		if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) &&
			!strings.HasPrefix(trimmed, "[INFO]") &&
			!strings.HasPrefix(trimmed, "[ERROR]") &&
			!strings.HasPrefix(trimmed, "[WARN]") {
			// Return everything from this line onward
			return []byte(strings.Join(lines[i:], "\n"))
		}
	}

	return nil
}

