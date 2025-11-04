package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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
	WhiteScreen      bool                  `json:"whiteScreen"`
	HTTPError        *HTTPErrorDetails     `json:"httpError,omitempty"`
	LoadError        string                `json:"loadError,omitempty"`
	ModuleError      string                `json:"moduleError,omitempty"`
	ResourceCount    int                   `json:"resourceCount"`
	LoadTimeMs       float64               `json:"loadTimeMs"`
	PerformanceMetrics map[string]interface{} `json:"performanceMetrics,omitempty"`
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
	Source          string                      `json:"source"`
	CapturedAt      time.Time                   `json:"capturedAt"`
	URL             string                      `json:"url"`
	Title           string                      `json:"title,omitempty"`
}

// BrowserlessService handles browserless command execution for fallback diagnostics
type BrowserlessService struct {
	timeout    time.Duration
	cacheMutex sync.RWMutex
	cache      map[string]*cacheEntry
	cacheTTL   time.Duration
}

type cacheEntry struct {
	result    *BrowserlessFallbackResult
	expiresAt time.Time
}

// NewBrowserlessService creates a new browserless service
func NewBrowserlessService() *BrowserlessService {
	return &BrowserlessService{
		timeout:  30 * time.Second,
		cache:    make(map[string]*cacheEntry),
		cacheTTL: 30 * time.Second, // Cache results for 30 seconds
	}
}

// GetFallbackDiagnostics retrieves console logs, network requests, and page status using browserless
func (s *BrowserlessService) GetFallbackDiagnostics(ctx context.Context, url string) (*BrowserlessFallbackResult, error) {
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

	result := &BrowserlessFallbackResult{
		URL:        url,
		Source:     "browserless",
		CapturedAt: time.Now(),
	}

	// Run diagnostics in parallel
	var wg sync.WaitGroup
	var errMutex sync.Mutex
	var errors []error

	// Collect console logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		logs, title, err := s.getConsoleLogs(ctx, url)
		if err != nil {
			errMutex.Lock()
			errors = append(errors, fmt.Errorf("console logs: %w", err))
			errMutex.Unlock()
		} else {
			result.ConsoleLogs = logs
			if title != "" {
				result.Title = title
			}
		}
	}()

	// Collect network requests
	wg.Add(1)
	go func() {
		defer wg.Done()
		requests, title, err := s.getNetworkRequests(ctx, url)
		if err != nil {
			errMutex.Lock()
			errors = append(errors, fmt.Errorf("network requests: %w", err))
			errMutex.Unlock()
		} else {
			result.NetworkRequests = requests
			if title != "" && result.Title == "" {
				result.Title = title
			}
		}
	}()

	// Analyze page status
	wg.Add(1)
	go func() {
		defer wg.Done()
		status, err := s.getPageStatus(ctx, url)
		if err != nil {
			errMutex.Lock()
			errors = append(errors, fmt.Errorf("page status: %w", err))
			errMutex.Unlock()
		} else {
			result.PageStatus = status
		}
	}()

	wg.Wait()

	// If all operations failed, return error
	if len(errors) == 3 {
		return nil, fmt.Errorf("all browserless operations failed: %v", errors)
	}

	// Log partial failures
	if len(errors) > 0 {
		logger.Warn("Some browserless operations failed: %v", errors)
	}

	// Update cache
	s.cacheMutex.Lock()
	s.cache[cacheKey] = &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMutex.Unlock()

	return result, nil
}

// getConsoleLogs executes browserless console command
func (s *BrowserlessService) getConsoleLogs(ctx context.Context, url string) ([]BrowserlessConsoleLog, string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "resource-browserless", "console", url, "--timeout", "10000", "--wait-ms", "3000")
	output, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("console command failed: %w", err)
	}

	var response struct {
		Success bool                    `json:"success"`
		Logs    []BrowserlessConsoleLog `json:"logs"`
		Title   string                  `json:"title"`
		Error   string                  `json:"error"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, "", fmt.Errorf("failed to parse console output: %w", err)
	}

	if !response.Success {
		return nil, "", fmt.Errorf("console capture failed: %s", response.Error)
	}

	return response.Logs, response.Title, nil
}

// getNetworkRequests executes browserless network command
func (s *BrowserlessService) getNetworkRequests(ctx context.Context, url string) ([]BrowserlessNetworkRequest, string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "resource-browserless", "network", url, "--timeout", "10000", "--wait-ms", "2000")
	output, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("network command failed: %w", err)
	}

	var response struct {
		Success  bool                        `json:"success"`
		Requests []BrowserlessNetworkRequest `json:"requests"`
		Title    string                      `json:"title"`
		Error    string                      `json:"error"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, "", fmt.Errorf("failed to parse network output: %w", err)
	}

	if !response.Success {
		return nil, "", fmt.Errorf("network capture failed: %s", response.Error)
	}

	return response.Requests, response.Title, nil
}

// getPageStatus combines health-check and performance data to analyze page status
func (s *BrowserlessService) getPageStatus(ctx context.Context, url string) (*BrowserlessPageStatus, error) {
	status := &BrowserlessPageStatus{}

	// Get performance metrics
	perfCtx, perfCancel := context.WithTimeout(ctx, 15*time.Second)
	defer perfCancel()

	perfCmd := exec.CommandContext(perfCtx, "resource-browserless", "performance", url, "--timeout", "10000")
	perfOutput, err := perfCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("performance command failed: %w", err)
	}

	var perfResponse struct {
		Success bool `json:"success"`
		Metrics struct {
			DOMInteractive      float64 `json:"dom_interactive"`
			DOMComplete         float64 `json:"dom_complete"`
			TotalLoadTime       float64 `json:"total_load_time"`
			FirstPaint          *float64 `json:"first_paint"`
			FirstContentfulPaint *float64 `json:"first_contentful_paint"`
			ResourceCount       int     `json:"resource_count"`
		} `json:"metrics"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(perfOutput, &perfResponse); err != nil {
		return nil, fmt.Errorf("failed to parse performance output: %w", err)
	}

	if !perfResponse.Success {
		return nil, fmt.Errorf("performance capture failed: %s", perfResponse.Error)
	}

	// Analyze metrics for issues
	status.ResourceCount = perfResponse.Metrics.ResourceCount
	status.LoadTimeMs = perfResponse.Metrics.TotalLoadTime

	// White screen detection: fast load + very few resources + no paint events
	if perfResponse.Metrics.TotalLoadTime < 100 &&
		perfResponse.Metrics.ResourceCount < 5 &&
		perfResponse.Metrics.FirstPaint == nil {
		status.WhiteScreen = true
	}

	// Store full metrics
	status.PerformanceMetrics = map[string]interface{}{
		"domInteractive":       perfResponse.Metrics.DOMInteractive,
		"domComplete":          perfResponse.Metrics.DOMComplete,
		"totalLoadTime":        perfResponse.Metrics.TotalLoadTime,
		"firstPaint":           perfResponse.Metrics.FirstPaint,
		"firstContentfulPaint": perfResponse.Metrics.FirstContentfulPaint,
		"resourceCount":        perfResponse.Metrics.ResourceCount,
	}

	return status, nil
}

// ClearCache clears the browserless result cache
func (s *BrowserlessService) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cache = make(map[string]*cacheEntry)
	logger.Info("Browserless cache cleared")
}
