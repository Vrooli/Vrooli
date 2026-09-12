package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// ReviewClient calls git-control-tower's unified review API.
// DOC: docs/internal/SEAMS.md#review-client
type ReviewClient interface {
	// TriggerReview starts a review run and returns the job ID.
	TriggerReview(ctx context.Context, req ReviewRequest) (string, error)
	// PollReview checks review job status. Returns result, done flag, and error.
	PollReview(ctx context.Context, jobID string) (*ReviewResult, bool, error)
	// Ping checks whether git-control-tower is reachable.
	Ping(ctx context.Context) error
}

// ReviewThresholds configures readiness criteria passed to git-control-tower.
type ReviewThresholds struct {
	CodeQualityMinScore   float64 `json:"codeQualityMinScore"`
	TestMinPassRate       float64 `json:"testMinPassRate"`
	MaxBlockingViolations int     `json:"maxBlockingViolations"`
	MaxWarnings           int     `json:"maxWarnings"`
	RequireScreenshots    bool    `json:"requireScreenshots"`
	RequireTests          bool    `json:"requireTests"`
}

// ReviewRequest describes what to review.
type ReviewRequest struct {
	ScenarioName  string            `json:"scenarioName"`
	ExpectedPaths []string          `json:"expectedPaths,omitempty"`
	SandboxID     string            `json:"sandboxId,omitempty"`
	Thresholds    *ReviewThresholds `json:"thresholds,omitempty"`
}

// HTTPReviewClient implements ReviewClient using git-control-tower's HTTP API.
type HTTPReviewClient struct {
	httpClient *http.Client
}

// NewHTTPReviewClient creates a review client. If httpClient is nil, a default
// client with 30s timeout is used.
func NewHTTPReviewClient(httpClient *http.Client) *HTTPReviewClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPReviewClient{httpClient: httpClient}
}

func resolveGitControlTowerBaseURL(ctx context.Context) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "git-control-tower")
	if err != nil {
		return "", fmt.Errorf("resolve git-control-tower: %w", err)
	}
	return strings.TrimRight(baseURL, "/"), nil
}

// reviewRunResponse mirrors git-control-tower's ReviewRunResponse.
type reviewRunResponse struct {
	JobID string `json:"jobId"`
}

type reviewRunConflictResponse struct {
	Error string `json:"error"`
	JobID string `json:"jobId,omitempty"`
}

// reviewJobStatus mirrors git-control-tower's ReviewJobStatus.
type reviewJobStatus struct {
	JobID     string                 `json:"jobId"`
	Status    string                 `json:"status"`
	Checks    map[string]string      `json:"checks"`
	Summary   *reviewSummaryResponse `json:"summary,omitempty"`
	StartedAt string                 `json:"startedAt,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// reviewSummaryResponse mirrors git-control-tower's ReviewSummaryResponse.
type reviewSummaryResponse struct {
	ScenarioName      string            `json:"scenarioName"`
	Readiness         string            `json:"readiness"`
	Dimensions        json.RawMessage   `json:"dimensions"`
	DimensionStatuses map[string]string `json:"dimensionStatuses,omitempty"`
	Capabilities      map[string]bool   `json:"capabilities"`
	Timestamp         string            `json:"timestamp"`
}

// TriggerReview starts a review run and returns the job ID.
func (c *HTTPReviewClient) TriggerReview(ctx context.Context, req ReviewRequest) (string, error) {
	baseURL, err := resolveGitControlTowerBaseURL(ctx)
	if err != nil {
		return "", err
	}
	return c.triggerReviewAtBaseURL(ctx, baseURL, req)
}

func (c *HTTPReviewClient) triggerReviewAtBaseURL(ctx context.Context, baseURL string, req ReviewRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal review request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/review/run", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create review request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("review request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read review response: %w", err)
	}
	if resp.StatusCode == http.StatusConflict {
		var conflict reviewRunConflictResponse
		if json.Unmarshal(respBody, &conflict) == nil &&
			strings.Contains(strings.ToLower(conflict.Error), "already in progress") &&
			strings.TrimSpace(conflict.JobID) != "" {
			return strings.TrimSpace(conflict.JobID), nil
		}
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("review run returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result reviewRunResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal review response: %w", err)
	}
	if strings.TrimSpace(result.JobID) == "" {
		return "", fmt.Errorf("review response missing jobId")
	}
	return result.JobID, nil
}

// PollReview checks review job status. Returns result, done flag, and error.
func (c *HTTPReviewClient) PollReview(ctx context.Context, jobID string) (*ReviewResult, bool, error) {
	baseURL, err := resolveGitControlTowerBaseURL(ctx)
	if err != nil {
		return nil, false, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/review/run/"+jobID, nil)
	if err != nil {
		return nil, false, fmt.Errorf("create poll request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, false, fmt.Errorf("poll request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("read poll response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("poll returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var job reviewJobStatus
	if err := json.Unmarshal(respBody, &job); err != nil {
		return nil, false, fmt.Errorf("unmarshal poll response: %w", err)
	}

	return mapJobToResult(job)
}

// mapJobToResult converts a git-control-tower job status to an execution ReviewResult.
func mapJobToResult(job reviewJobStatus) (*ReviewResult, bool, error) {
	status := strings.ToLower(strings.TrimSpace(job.Status))

	switch status {
	case "completed":
		result := &ReviewResult{
			JobID:      job.JobID,
			ReviewedAt: nowRFC3339(),
		}
		if job.Summary != nil {
			result.Classification = mapReadinessToClassification(job.Summary.Readiness)
			result.Summary = fmt.Sprintf("Scenario %s readiness: %s", job.Summary.ScenarioName, job.Summary.Readiness)
			result.Dimensions = parseDimensions(job.Summary.Dimensions, job.Summary.DimensionStatuses)
			result.RawDimensions = job.Summary.Dimensions
		} else {
			result.Classification = "not_assessable"
			result.Summary = "review completed without summary"
		}
		return result, true, nil

	case "failed":
		result := &ReviewResult{
			JobID:          job.JobID,
			Classification: "not_assessable",
			Summary:        "review failed: " + job.Error,
			ReviewedAt:     nowRFC3339(),
		}
		return result, true, nil

	default:
		// Still running (pending, running, etc.)
		return nil, false, nil
	}
}

func mapReadinessToClassification(readiness string) string {
	switch strings.ToLower(strings.TrimSpace(readiness)) {
	case "green":
		return "ready"
	case "yellow":
		return "ready_with_notes"
	case "red":
		return "needs_work"
	default:
		return "not_assessable"
	}
}

// dimensionEntry is used for parsing the raw dimensions JSON.
type dimensionEntry struct {
	Available          bool    `json:"available"`
	Score              float64 `json:"score"`
	Violations         int     `json:"violations"`
	BlockingViolations int     `json:"blockingViolations"`
	Passed             bool    `json:"passed"`
	Total              int     `json:"total"`
	PassedCount        int     `json:"passedCount"`
	FailedCount        int     `json:"failedCount"`
	Warnings           int     `json:"warnings"`
	Stale              bool    `json:"stale"`
	ScreenshotCount    int     `json:"screenshotCount"`
	TracedFiles        int     `json:"tracedFiles"`
	TotalFiles         int     `json:"totalFiles"`
}

// Ping checks whether git-control-tower is reachable by issuing a HEAD request
// to its health endpoint.
func (c *HTTPReviewClient) Ping(ctx context.Context) error {
	baseURL, err := resolveGitControlTowerBaseURL(ctx)
	if err != nil {
		return err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(pingCtx, http.MethodHead, baseURL+"/api/v1/health", nil)
	if err != nil {
		return fmt.Errorf("create ping request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GCT unreachable: %w", err)
	}
	if closeErr := resp.Body.Close(); closeErr != nil {
		slog.Debug("execution: close GCT ping body failed", "err", closeErr)
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("GCT returned status %d", resp.StatusCode)
	}
	return nil
}

// parseDimensions builds ReviewDimension entries using GCT-provided statuses
// (single source of truth) and raw dimension data for detail text.
func parseDimensions(raw json.RawMessage, statuses map[string]string) []ReviewDimension {
	if len(raw) == 0 {
		return nil
	}
	var dims map[string]dimensionEntry
	if err := json.Unmarshal(raw, &dims); err != nil {
		return nil
	}
	var result []ReviewDimension
	for name, dim := range dims {
		status := statuses[name]
		if status == "" {
			if !dim.Available {
				status = "skipped"
			} else {
				status = "green"
			}
		}

		var details string
		switch name {
		case "codeQuality":
			if dim.Violations > 0 || dim.Score > 0 {
				details = fmt.Sprintf("%.1f score, %d violations", dim.Score, dim.Violations)
			}
		case "tests":
			if dim.Total > 0 {
				details = fmt.Sprintf("%d/%d passed", dim.PassedCount, dim.Total)
			}
		case "standards":
			if dim.BlockingViolations > 0 {
				details = fmt.Sprintf("%d blocking violations", dim.BlockingViolations)
			} else if dim.Warnings > 0 {
				details = fmt.Sprintf("%d warnings", dim.Warnings)
			}
		}

		result = append(result, ReviewDimension{
			Name:    name,
			Status:  status,
			Details: details,
		})
	}
	return result
}
