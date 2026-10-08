package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/envkit-go"

	"github.com/gin-gonic/gin"
	repocontract "github.com/vrooli/repo-contract-go"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// LighthouseHandler handles Lighthouse audit requests
type LighthouseHandler struct {
	repoRoot string
}

const (
	lighthouseConfigRelativePath = ".vrooli/lighthouse.json"
	lighthouseSetupHint          = "Create .vrooli/lighthouse.json (see scenarios/test-genie/docs/phases/performance/lighthouse.md) so the performance phase can run Lighthouse audits."
)

// NewLighthouseHandler creates a new Lighthouse handler
func NewLighthouseHandler() *LighthouseHandler {
	return &LighthouseHandler{
		repoRoot: resolveRepoRoot(),
	}
}

func resolveRepoRoot() string {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		panic(fmt.Sprintf("resolve repo root: %v", err))
	}
	return root
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
	previousRunID := h.latestRunID(c.Request.Context(), scenarioName)

	// Execute Lighthouse via the Go-native test-genie performance phase
	cmd := exec.CommandContext(
		c.Request.Context(),
		"test-genie", "execute", scenarioName, "performance", "--no-stream", "--skip", "structure,dependencies,unit,integration,business",
	)
	cmd.Dir = h.repoRoot

	// Set environment
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{
		fmt.Sprintf("VROOLI_ROOT=%s", h.repoRoot),
		fmt.Sprintf("TESTING_LIGHTHOUSE_PAGES=%s", strings.Join(req.Pages, ",")),
	})

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

	latestRunID := h.latestRunID(c.Request.Context(), scenarioName)
	resultsUpdated := latestRunID != "" && latestRunID != previousRunID

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
	scenariosDir := filepath.Join(h.repoRoot, "scenarios")
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
	return filepath.Join(h.repoRoot, "scenarios", name)
}

func (h *LighthouseHandler) configPath(name string) string {
	return filepath.Join(h.scenarioPath(name), lighthouseConfigRelativePath)
}

func (h *LighthouseHandler) runsClient(ctx context.Context) (runsconnect.RunsServiceClient, string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return nil, "", err
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return runsconnect.NewRunsServiceClient(&http.Client{Timeout: 10 * time.Second}, baseURL), baseURL, nil
}

func (h *LighthouseHandler) latestRunID(ctx context.Context, scenario string) string {
	client, _, err := h.runsClient(ctx)
	if err != nil {
		return ""
	}
	response, err := client.ListRuns(ctx, connect.NewRequest(&runspb.ListRunsRequest{Target: scenario, Limit: 1}))
	if err != nil || len(response.Msg.GetRuns()) == 0 {
		return ""
	}
	return response.Msg.GetRuns()[0].GetRunId()
}

func (h *LighthouseHandler) respondMissingConfig(c *gin.Context, scenarioName string) {
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

	ext := ".html"
	if format == "json" {
		ext = ".json"
	}

	client, baseURL, err := h.runsClient(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "test-genie unavailable"})
		return
	}
	runs, err := client.ListRuns(c.Request.Context(), connect.NewRequest(&runspb.ListRunsRequest{Target: scenarioName}))
	if err == nil {
		for _, run := range runs.Msg.GetRuns() {
			catalog, listErr := client.ListRunArtifacts(c.Request.Context(), connect.NewRequest(&runspb.ListRunArtifactsRequest{Target: scenarioName, RunId: run.GetRunId()}))
			if listErr != nil {
				continue
			}
			for _, artifact := range catalog.Msg.GetArtifacts() {
				if !strings.Contains(artifact.GetLabel(), reportID+ext) {
					continue
				}
				request, requestErr := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, baseURL+artifact.GetAccessPath(), nil)
				if requestErr != nil {
					continue
				}
				response, requestErr := http.DefaultClient.Do(request)
				if requestErr != nil {
					continue
				}
				defer response.Body.Close()
				c.Header("Content-Type", artifact.GetMediaType())
				c.Status(response.StatusCode)
				_, _ = io.Copy(c.Writer, response.Body)
				return
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Report %s not found", reportID)})
}

// loadLatestReports loads the most recent Lighthouse reports for a scenario
func (h *LighthouseHandler) loadLatestReports(scenarioName string) ([]LighthouseReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, _, err := h.runsClient(ctx)
	if err != nil {
		return nil, err
	}
	runs, err := client.ListRuns(ctx, connect.NewRequest(&runspb.ListRunsRequest{Target: scenarioName, Limit: 1}))
	if err != nil || len(runs.Msg.GetRuns()) == 0 {
		return nil, fmt.Errorf("no test-genie run available for %s", scenarioName)
	}
	return h.loadRunReports(ctx, client, scenarioName, runs.Msg.GetRuns()[0].GetRunId())
}

func (h *LighthouseHandler) loadRunReports(ctx context.Context, client runsconnect.RunsServiceClient, scenarioName, runID string) ([]LighthouseReport, error) {
	response, err := client.GetPhaseArtifact(ctx, connect.NewRequest(&runspb.GetPhaseArtifactRequest{Target: scenarioName, RunId: runID, Phase: "lighthouse"}))
	if err != nil {
		return nil, fmt.Errorf("read lighthouse phase result from test-genie: %w", err)
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

	if err := json.Unmarshal([]byte(response.Msg.GetContent()), &phaseResults); err != nil {
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
			JSONPath:  fmt.Sprintf("test-genie://%s/%s/lighthouse", scenarioName, runID),
		})
	}

	return reports, nil
}

// loadAllReports loads all Lighthouse reports for a scenario (for history)
func (h *LighthouseHandler) loadAllReports(scenarioName string) ([]LighthouseReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, _, err := h.runsClient(ctx)
	if err != nil {
		return nil, err
	}
	runs, err := client.ListRuns(ctx, connect.NewRequest(&runspb.ListRunsRequest{Target: scenarioName}))
	if err != nil {
		return nil, err
	}
	reports := make([]LighthouseReport, 0)
	for _, run := range runs.Msg.GetRuns() {
		runReports, runErr := h.loadRunReports(ctx, client, scenarioName, run.GetRunId())
		if runErr == nil {
			reports = append(reports, runReports...)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Timestamp.After(reports[j].Timestamp)
	})

	return reports, nil
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
