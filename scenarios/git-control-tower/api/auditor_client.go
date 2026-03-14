package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// AuditorClient is a lightweight HTTP client for scenario-auditor APIs.
type AuditorClient struct {
	httpClient *http.Client
	resolver   *discovery.Resolver
}

// NewAuditorClient creates a new auditor client with the given timeout.
func NewAuditorClient(timeout time.Duration) *AuditorClient {
	return &AuditorClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *AuditorClient) resolveBaseURL(ctx context.Context) (string, error) {
	if c.resolver != nil {
		return c.resolver.ResolveScenarioURLDefault(ctx, "scenario-auditor")
	}
	return discovery.ResolveScenarioURLDefault(ctx, "scenario-auditor")
}

// StartCheck calls POST /api/v1/standards/check/{name} to start an async standards check.
func (c *AuditorClient) StartCheck(ctx context.Context, scenarioName, checkType string) (*AuditorCheckJobResponse, error) {
	req := AuditorCheckRequest{Type: checkType}
	var result AuditorCheckJobResponse
	err := c.doJSONAccept(ctx, "/api/v1/standards/check/"+url.PathEscape(scenarioName), req, &result, http.StatusOK, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetJobStatus calls GET /api/v1/standards/check/jobs/{jobId} to poll job status.
func (c *AuditorClient) GetJobStatus(ctx context.Context, jobID string) (*AuditorJobStatus, error) {
	var result AuditorJobStatus
	err := c.doGet(ctx, "/api/v1/standards/check/jobs/"+url.PathEscape(jobID), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRules calls GET /api/v1/rules to list all rule definitions.
func (c *AuditorClient) ListRules(ctx context.Context) (*AuditorRulesListResponse, error) {
	var result AuditorRulesListResponse
	err := c.doGet(ctx, "/api/v1/rules", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ApplyFix calls POST /api/v1/standards/fix to apply automated fixes.
func (c *AuditorClient) ApplyFix(ctx context.Context, req AuditorFixRequest) (*AuditorFixResponse, error) {
	var result AuditorFixResponse
	err := c.doJSONAccept(ctx, "/api/v1/standards/fix", req, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetViolations calls GET /api/v1/standards/violations?scenario=X to list stored violations.
func (c *AuditorClient) GetViolations(ctx context.Context, scenarioName string) (*AuditorViolationsResponse, error) {
	var result AuditorViolationsResponse
	path := "/api/v1/standards/violations?scenario=" + url.QueryEscape(scenarioName)
	err := c.doGet(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *AuditorClient) doJSONAccept(ctx context.Context, path string, body, result interface{}, acceptStatuses ...int) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve auditor url: %w", err)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auditor request failed: %w", err)
	}
	defer resp.Body.Close()

	accepted := false
	for _, s := range acceptStatuses {
		if resp.StatusCode == s {
			accepted = true
			break
		}
	}
	if !accepted {
		return parseAuditorError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *AuditorClient) doGet(ctx context.Context, path string, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve auditor url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auditor request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseAuditorError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func parseAuditorError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("auditor error: %s", errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("auditor error: %s", errResp.Message)
		}
	}
	return fmt.Errorf("auditor error: status %d, body: %s", resp.StatusCode, string(body))
}
