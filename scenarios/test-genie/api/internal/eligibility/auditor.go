// Package eligibility computes whether a scenario qualifies for test-genie's
// routed e2e path and exposes the auditor-fetch wiring that test-genie's
// playbooks and standards phases share.
//
// The package is the single owner of the auditor HTTP calls; phase_standards
// and phase_playbooks both consume it so the scan call site is not
// duplicated.
package eligibility

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
)

// SummaryResponse mirrors the scenario-auditor /summary envelope.
type SummaryResponse struct {
	Summary *ViolationSummary `json:"summary"`
}

// ScanArtifactRef points to the saved auditor scan output.
type ScanArtifactRef struct {
	Path string `json:"path"`
}

// RuleCount aggregates per-rule counts in the auditor summary.
type RuleCount struct {
	RuleID   string `json:"rule_id"`
	Count    int    `json:"count"`
	Title    string `json:"title,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// ViolationExcerpt is one line of the scan's top-violations list. It feeds the
// standards phase's finding projection (standardsArchFindings); routing
// eligibility no longer reads it (that judgment moved to storage-health's L2
// verdict — see eligibility/router.go).
type ViolationExcerpt struct {
	Severity   string `json:"severity"`
	RuleID     string `json:"rule_id,omitempty"`
	Title      string `json:"title,omitempty"`
	FilePath   string `json:"file_path,omitempty"`
	LineNumber int    `json:"line_number,omitempty"`
}

// ViolationSummary is the auditor's structured scan result.
type ViolationSummary struct {
	Total           int                          `json:"total"`
	BySeverity      map[string]int               `json:"by_severity"`
	ByRule          []RuleCount                  `json:"by_rule"`
	HighestSeverity string                       `json:"highest_severity"`
	TopViolations   []ViolationExcerpt           `json:"top_violations"`
	Artifact        *ScanArtifactRef             `json:"artifact,omitempty"`
	Recommended     []string                     `json:"recommended_steps,omitempty"`
	Assessment      *commonv1.MaturityAssessment `json:"assessment,omitempty"`
}

// StandardsStartResponse decodes the POST .../standards/check/<scenario> response.
type StandardsStartResponse struct {
	JobID  string          `json:"job_id"`
	Status StandardsStatus `json:"status"`
}

// StandardsStatus decodes job status replies.
type StandardsStatus struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// ResolveBaseURL is the seam test-genie phases use to discover the
// scenario-auditor base URL. Tests override it.
var ResolveBaseURL = func(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, "scenario-auditor")
}

// HTTPClient is the HTTP client used by the auditor calls; tests may override.
var HTTPClient = &http.Client{Timeout: 10 * time.Second}

// FetchSummary runs the full scan flow: starts a job, polls until it
// completes, and returns the parsed summary.
func FetchSummary(ctx context.Context, logWriter io.Writer, baseURL, scenarioName string, mapping workspace.Mapping, summaryLimit int) (*ViolationSummary, error) {
	jobID, err := StartScan(ctx, baseURL, scenarioName, mapping)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(jobID) == "" {
		return nil, fmt.Errorf("scenario-auditor returned an empty standards job id")
	}

	pollInterval := 2 * time.Second
	for {
		status, err := FetchStatus(ctx, baseURL, jobID)
		if err != nil {
			return nil, err
		}
		state := strings.ToLower(strings.TrimSpace(status.Status))
		switch state {
		case "completed", "success":
			return FetchSummaryByJob(ctx, baseURL, jobID, summaryLimit)
		case "failed", "error":
			return nil, fmt.Errorf("scenario-auditor standards scan failed: %s", firstNonEmpty(status.Error, status.Message, "unknown failure"))
		case "cancelled", "canceled":
			return nil, fmt.Errorf("scenario-auditor standards scan cancelled: %s", firstNonEmpty(status.Message, "scan cancelled"))
		}

		if logWriter != nil && strings.TrimSpace(status.Message) != "" {
			shared.LogStep(logWriter, "scenario-auditor status=%s (%s)", state, strings.TrimSpace(status.Message))
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// StartScan starts a standards scan and returns its job id.
func StartScan(ctx context.Context, baseURL, scenarioName string, mapping workspace.Mapping) (string, error) {
	payload := map[string]any{"type": "full"}
	scenarioPath := mapping.PhysicalScenarioDir
	if strings.TrimSpace(scenarioPath) != "" {
		payload["scenario_path"] = strings.TrimSpace(scenarioPath)
	}
	if mapping.HasLogicalPlacement() {
		payload["logical_repo_root"] = mapping.LogicalRepoRoot
		payload["logical_scenario_relpath"] = mapping.LogicalScenarioRelPath
	}

	responseBody, err := RequestJSON(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1/standards/check/"+scenarioName, payload)
	if err != nil {
		return "", err
	}

	var response StandardsStartResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("decode scenario-auditor standards start response: %w", err)
	}
	if strings.TrimSpace(response.JobID) != "" {
		return strings.TrimSpace(response.JobID), nil
	}
	if strings.TrimSpace(response.Status.ID) != "" {
		return strings.TrimSpace(response.Status.ID), nil
	}
	return "", fmt.Errorf("scenario-auditor standards start response did not include a job id")
}

// FetchStatus polls a job for status.
func FetchStatus(ctx context.Context, baseURL, jobID string) (*StandardsStatus, error) {
	responseBody, err := RequestJSON(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/standards/check/jobs/"+jobID, nil)
	if err != nil {
		return nil, err
	}
	var status StandardsStatus
	if err := json.Unmarshal(responseBody, &status); err != nil {
		return nil, fmt.Errorf("decode scenario-auditor standards status response: %w", err)
	}
	return &status, nil
}

// FetchSummaryByJob reads the structured summary for a completed job.
func FetchSummaryByJob(ctx context.Context, baseURL, jobID string, summaryLimit int) (*ViolationSummary, error) {
	url := fmt.Sprintf("%s/api/v1/standards/check/jobs/%s/summary?limit=%d&min_severity=info", strings.TrimRight(baseURL, "/"), jobID, summaryLimit)
	responseBody, err := RequestJSON(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return ParseSummary(string(responseBody))
}

// RequestJSON is the shared HTTP helper used by all auditor calls.
func RequestJSON(ctx context.Context, method, endpoint string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode scenario-auditor request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create scenario-auditor request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("read scenario-auditor response: %w", readErr)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("scenario-auditor returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

// ParseSummary decodes the scan summary JSON envelope.
func ParseSummary(raw string) (*ViolationSummary, error) {
	payload := strings.TrimSpace(raw)
	if payload == "" {
		return nil, fmt.Errorf("scenario-auditor produced no output")
	}

	var envelope SummaryResponse
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse scenario-auditor JSON: %w", err)
	}
	if envelope.Summary == nil {
		return nil, fmt.Errorf("scenario-auditor JSON missing 'summary' payload")
	}

	summary := *envelope.Summary
	if summary.BySeverity == nil {
		summary.BySeverity = map[string]int{}
	}
	summary.HighestSeverity = NormalizeSeverity(summary.HighestSeverity)
	if summary.HighestSeverity == "" && summary.Total > 0 {
		summary.HighestSeverity = "info"
	}
	if len(summary.BySeverity) > 0 {
		normalized := make(map[string]int, len(summary.BySeverity))
		for k, v := range summary.BySeverity {
			key := NormalizeSeverity(k)
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(k))
				if key == "" {
					continue
				}
			}
			normalized[key] += v
		}
		summary.BySeverity = normalized
	}
	for i := range summary.ByRule {
		summary.ByRule[i].Severity = NormalizeSeverity(summary.ByRule[i].Severity)
	}
	for i := range summary.TopViolations {
		summary.TopViolations[i].Severity = NormalizeSeverity(summary.TopViolations[i].Severity)
	}
	sort.Slice(summary.ByRule, func(i, j int) bool {
		return summary.ByRule[i].Count > summary.ByRule[j].Count
	})
	return &summary, nil
}

// NormalizeSeverity collapses casing variants to the canonical lowercase form.
func NormalizeSeverity(raw string) string {
	return shared.NormalizeAuditorSeverity(raw)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
