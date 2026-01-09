package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LighthouseHandler handles Lighthouse audit requests
type LighthouseHandler struct {
	appRoot string
}

const (
	lighthouseConfigRelativePath = ".vrooli/lighthouse.json"
	lighthouseSetupHint          = "Create .vrooli/lighthouse.json (see scenarios/test-genie/docs/phases/performance/lighthouse.md) so the performance phase can run Lighthouse audits."
)

// NewLighthouseHandler creates a new Lighthouse handler
func NewLighthouseHandler() *LighthouseHandler {
	appRoot := os.Getenv("APP_ROOT")
	if appRoot == "" {
		appRoot = os.Getenv("HOME") + "/Vrooli"
	}
	return &LighthouseHandler{
		appRoot: appRoot,
	}
}

// LighthouseRunRequest represents a request to run Lighthouse audits
type LighthouseRunRequest struct {
	Pages []string `json:"pages"` // Optional: specific page IDs to test
}

// LighthouseReport represents a single Lighthouse audit result
type LighthouseReport struct {
	ID        string             `json:"id"`
	Timestamp time.Time          `json:"timestamp"`
	PageID    string             `json:"page_id"`
	PageLabel string             `json:"page_label"`
	URL       string             `json:"url"`
	Viewport  string             `json:"viewport"`
	Status    string             `json:"status"`
	Scores    map[string]float64 `json:"scores"`
	Failures  []map[string]any   `json:"failures"`
	Warnings  []map[string]any   `json:"warnings"`
	ReportURL string             `json:"report_url"`
	JSONPath  string             `json:"json_path"`
}

// LighthouseHistory represents historical Lighthouse data
type LighthouseHistory struct {
	Scenario string             `json:"scenario"`
	Reports  []LighthouseReport `json:"reports"`
	Trend    *TrendData         `json:"trend"`
}

// TrendData represents performance trends over time
type TrendData struct {
	Performance   []TrendPoint `json:"performance"`
	Accessibility []TrendPoint `json:"accessibility"`
	BestPractices []TrendPoint `json:"best_practices"`
	SEO           []TrendPoint `json:"seo"`
}

// TrendPoint represents a single data point in a trend
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
	PageID    string    `json:"page_id"`
}

// RunLighthouse executes Lighthouse audits for a scenario
// POST /api/v1/scenarios/:scenario/lighthouse/run
func (h *LighthouseHandler) RunLighthouse(c *gin.Context) {
	scenarioName := c.Param("scenario")
	if scenarioName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scenario name is required"})
		return
	}

	var req LighthouseRunRequest
	if err := c.BindJSON(&req); err != nil {
		// Ignore binding errors - pages param is optional
		req.Pages = nil
	}

	scenarioPath := h.scenarioPath(scenarioName)
	configPath := h.configPath(scenarioName)
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			h.respondMissingConfig(c, scenarioName)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("Failed to inspect Lighthouse config for %s", scenarioName),
			"details": err.Error(),
		})
		return
	}

	// Track phase results timestamp so we can detect freshly generated output
	phaseResultsPath := h.phaseResultsPath(scenarioName)
	prevPhaseMod := fileModTime(phaseResultsPath)

	// Execute Lighthouse via the Go-native test-genie performance phase
	cmd := exec.CommandContext(
		c.Request.Context(),
		"test-genie", "execute", scenarioName, "performance", "--no-stream", "--skip", "structure,dependencies,unit,integration,business",
	)
	cmd.Dir = h.appRoot

	// Set environment
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("APP_ROOT=%s", h.appRoot),
		fmt.Sprintf("TESTING_LIGHTHOUSE_PAGES=%s", strings.Join(req.Pages, ",")),
	)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with timeout
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	resultsUpdated := fileModTime(phaseResultsPath).After(prevPhaseMod)

	// Load reports whenever the run succeeded or new artifacts were produced
	var reports []LighthouseReport
	if err == nil || resultsUpdated {
		var loadErr error
		reports, loadErr = h.loadLatestReports(scenarioName)
		if loadErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to load Lighthouse reports",
				"details": loadErr.Error(),
				"output":  output,
			})
			return
		}
	}

	if err != nil {
		// If the runner exited with a non-zero status but still produced fresh artifacts,
		// return them with a "failed" status so the UI can surface the new results.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && resultsUpdated {
			c.JSON(http.StatusOK, gin.H{
				"scenario":  scenarioName,
				"timestamp": time.Now().UTC(),
				"reports":   reports,
				"output":    output,
				"status":    "failed",
				"exit_code": exitErr.ExitCode(),
				"run_error": "Lighthouse audits reported failures",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Lighthouse execution failed",
			"output":  output,
			"cmd":     cmd.String(),
			"status":  "error",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scenario":  scenarioName,
		"timestamp": time.Now().UTC(),
		"reports":   reports,
		"output":    output,
		"status":    "passed",
	})
}

// ListMissingConfigs returns all scenarios that are missing Lighthouse configuration.
// GET /api/v1/lighthouse/missing-configs
func (h *LighthouseHandler) ListMissingConfigs(c *gin.Context) {
	scenariosDir := filepath.Join(h.appRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to inspect scenarios directory",
			"details": err.Error(),
		})
		return
	}

	missing := make([]map[string]string, 0)
	total := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		total++
		configPath := h.configPath(name)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			missing = append(missing, map[string]string{
				"scenario":      name,
				"expected_path": configPath,
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool {
		return missing[i]["scenario"] < missing[j]["scenario"]
	})

	c.JSON(http.StatusOK, gin.H{
		"missing": missing,
		"count":   len(missing),
		"total":   total,
	})
}

func (h *LighthouseHandler) scenarioPath(name string) string {
	return filepath.Join(h.appRoot, "scenarios", name)
}

func (h *LighthouseHandler) configPath(name string) string {
	return filepath.Join(h.scenarioPath(name), lighthouseConfigRelativePath)
}

func (h *LighthouseHandler) phaseResultsPath(name string) string {
	return filepath.Join(h.scenarioPath(name), "coverage", "phase-results", "lighthouse.json")
}

func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func (h *LighthouseHandler) respondMissingConfig(c *gin.Context, scenarioName string) {
	scenarioDir := filepath.Join("scenarios", scenarioName)
	createConfig := fmt.Sprintf("Create %s (see scenarios/test-genie/docs/phases/performance/lighthouse.md)", h.configPath(scenarioName))

	c.JSON(http.StatusNotFound, gin.H{
		"error":           fmt.Sprintf("Scenario %s has no %s", scenarioName, lighthouseConfigRelativePath),
		"missing_config":  true,
		"scenario":        scenarioName,
		"expected_path":   h.configPath(scenarioName),
		"hint":            lighthouseSetupHint,
		"suggested_steps": []string{createConfig, fmt.Sprintf("Then run: test-genie execute %s performance --no-stream --skip structure,dependencies,unit,integration,business (from repo root)", scenarioName)},
	})
}

// GetLighthouseHistory returns historical Lighthouse audit results
// GET /api/v1/scenarios/:scenario/lighthouse/history
func (h *LighthouseHandler) GetLighthouseHistory(c *gin.Context) {
	scenarioName := c.Param("scenario")
	if scenarioName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scenario name is required"})
		return
	}

	// Load all reports
	reports, err := h.loadAllReports(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load Lighthouse history",
			"details": err.Error(),
		})
		return
	}

	// Calculate trends
	trend := h.calculateTrends(reports)

	c.JSON(http.StatusOK, LighthouseHistory{
		Scenario: scenarioName,
		Reports:  reports,
		Trend:    trend,
	})
}

// GetLighthouseReport returns a specific Lighthouse report
// GET /api/v1/scenarios/:scenario/lighthouse/report/:reportId
func (h *LighthouseHandler) GetLighthouseReport(c *gin.Context) {
	scenarioName := c.Param("scenario")
	reportID := c.Param("reportId")

	if scenarioName == "" || reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scenario and reportId are required"})
		return
	}

	// Determine report type (html or json)
	format := c.DefaultQuery("format", "html")

	// Find report file
	artifactsDir := filepath.Join(h.appRoot, "scenarios", scenarioName, "test", "artifacts", "lighthouse")
	ext := ".html"
	if format == "json" {
		ext = ".json"
	}

	// reportID format: pageId_timestamp
	pattern := reportID + ext
	reportPath := filepath.Join(artifactsDir, pattern)

	// Check if file exists
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Report %s not found", reportID),
		})
		return
	}

	// Serve file
	if format == "html" {
		c.Header("Content-Type", "text/html")
	} else {
		c.Header("Content-Type", "application/json")
	}

	c.File(reportPath)
}

// loadLatestReports loads the most recent Lighthouse reports for a scenario
func (h *LighthouseHandler) loadLatestReports(scenarioName string) ([]LighthouseReport, error) {
	artifactsDir := filepath.Join(h.appRoot, "scenarios", scenarioName, "test", "artifacts", "lighthouse")

	// Read phase results for structured data
	phaseResultsPath := filepath.Join(h.appRoot, "scenarios", scenarioName, "coverage", "phase-results", "lighthouse.json")
	data, err := os.ReadFile(phaseResultsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lighthouse phase results: %w", err)
	}

	var phaseResults struct {
		Pages []struct {
			PageID    string             `json:"page_id"`
			PageLabel string             `json:"page_label"`
			URL       string             `json:"url"`
			Viewport  string             `json:"viewport"`
			Status    string             `json:"status"`
			Scores    map[string]float64 `json:"scores"`
			Failures  []map[string]any   `json:"failures"`
			Warnings  []map[string]any   `json:"warnings"`
			Timestamp string             `json:"timestamp"`
		} `json:"pages"`
	}

	if err := json.Unmarshal(data, &phaseResults); err != nil {
		return nil, fmt.Errorf("failed to parse lighthouse phase results: %w", err)
	}

	reports := make([]LighthouseReport, 0, len(phaseResults.Pages))
	for _, page := range phaseResults.Pages {
		timestamp, _ := time.Parse(time.RFC3339, page.Timestamp)
		timestampMs := timestamp.UnixMilli()

		reportID := fmt.Sprintf("%s_%d", page.PageID, timestampMs)

		reports = append(reports, LighthouseReport{
			ID:        reportID,
			Timestamp: timestamp,
			PageID:    page.PageID,
			PageLabel: page.PageLabel,
			URL:       page.URL,
			Viewport:  page.Viewport,
			Status:    page.Status,
			Scores:    page.Scores,
			Failures:  page.Failures,
			Warnings:  page.Warnings,
			ReportURL: fmt.Sprintf("/api/v1/scenarios/%s/lighthouse/report/%s?format=html", scenarioName, reportID),
			JSONPath:  filepath.Join(artifactsDir, reportID+".json"),
		})
	}

	return reports, nil
}

// loadAllReports loads all Lighthouse reports for a scenario (for history)
func (h *LighthouseHandler) loadAllReports(scenarioName string) ([]LighthouseReport, error) {
	artifactsDir := filepath.Join(h.appRoot, "scenarios", scenarioName, "test", "artifacts", "lighthouse")

	// Check if directory exists
	if _, err := os.Stat(artifactsDir); os.IsNotExist(err) {
		// Directory doesn't exist - return empty reports (not an error)
		return []LighthouseReport{}, nil
	}

	// List all JSON files
	files, err := os.ReadDir(artifactsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifacts directory: %w", err)
	}

	reports := make([]LighthouseReport, 0)

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(artifactsDir, file.Name())
		report, err := h.parseReportFile(filePath, scenarioName)
		if err != nil {
			// Skip invalid reports
			continue
		}

		reports = append(reports, *report)
	}

	// Sort by timestamp (newest first)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Timestamp.After(reports[j].Timestamp)
	})

	return reports, nil
}

// parseReportFile parses a Lighthouse JSON report file
func (h *LighthouseHandler) parseReportFile(filePath, scenarioName string) (*LighthouseReport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var lhr struct {
		FetchTime  string `json:"fetchTime"`
		FinalURL   string `json:"finalUrl"`
		Categories map[string]struct {
			Score float64 `json:"score"`
		} `json:"categories"`
		ConfigSettings struct {
			FormFactor string `json:"formFactor"`
		} `json:"configSettings"`
	}

	if err := json.Unmarshal(data, &lhr); err != nil {
		return nil, err
	}

	timestamp, _ := time.Parse(time.RFC3339, lhr.FetchTime)

	// Extract page ID and timestamp from filename
	basename := filepath.Base(filePath)
	basename = strings.TrimSuffix(basename, ".json")
	parts := strings.Split(basename, "_")
	pageID := strings.Join(parts[:len(parts)-1], "_")

	scores := make(map[string]float64)
	for category, data := range lhr.Categories {
		scores[category] = data.Score
	}

	reportID := strings.TrimSuffix(basename, ".json")

	return &LighthouseReport{
		ID:        reportID,
		Timestamp: timestamp,
		PageID:    pageID,
		URL:       lhr.FinalURL,
		Viewport:  lhr.ConfigSettings.FormFactor,
		Scores:    scores,
		ReportURL: fmt.Sprintf("/api/v1/scenarios/%s/lighthouse/report/%s?format=html", scenarioName, reportID),
		JSONPath:  filePath,
	}, nil
}

// calculateTrends computes performance trends from historical reports
func (h *LighthouseHandler) calculateTrends(reports []LighthouseReport) *TrendData {
	trend := &TrendData{
		Performance:   make([]TrendPoint, 0),
		Accessibility: make([]TrendPoint, 0),
		BestPractices: make([]TrendPoint, 0),
		SEO:           make([]TrendPoint, 0),
	}

	for _, report := range reports {
		if score, ok := report.Scores["performance"]; ok {
			trend.Performance = append(trend.Performance, TrendPoint{
				Timestamp: report.Timestamp,
				Score:     score,
				PageID:    report.PageID,
			})
		}
		if score, ok := report.Scores["accessibility"]; ok {
			trend.Accessibility = append(trend.Accessibility, TrendPoint{
				Timestamp: report.Timestamp,
				Score:     score,
				PageID:    report.PageID,
			})
		}
		if score, ok := report.Scores["best-practices"]; ok {
			trend.BestPractices = append(trend.BestPractices, TrendPoint{
				Timestamp: report.Timestamp,
				Score:     score,
				PageID:    report.PageID,
			})
		}
		if score, ok := report.Scores["seo"]; ok {
			trend.SEO = append(trend.SEO, TrendPoint{
				Timestamp: report.Timestamp,
				Score:     score,
				PageID:    report.PageID,
			})
		}
	}

	return trend
}
