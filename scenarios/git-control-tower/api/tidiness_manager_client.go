package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// TidinessManagerClient is a lightweight HTTP client for tidiness-manager APIs.
type TidinessManagerClient struct {
	httpClient *http.Client
	resolver   *discovery.Resolver
}

// NewTidinessManagerClient creates a new tidiness-manager client with the given timeout.
func NewTidinessManagerClient(timeout time.Duration) *TidinessManagerClient {
	return &TidinessManagerClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *TidinessManagerClient) resolveBaseURL(ctx context.Context) (string, error) {
	if c.resolver != nil {
		return c.resolver.ResolveScenarioURLDefault(ctx, "tidiness-manager")
	}
	return discovery.ResolveScenarioURLDefault(ctx, "tidiness-manager")
}

// GetTidinessScore calls GET /api/v1/scenarios/{scenario}/tidiness.
func (c *TidinessManagerClient) GetTidinessScore(ctx context.Context, scenario string) (*TidinessScoreResponse, error) {
	var result TidinessScoreResponse
	err := c.doGet(ctx, "/api/v1/scenarios/"+url.PathEscape(scenario)+"/tidiness", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetIssues calls GET /api/v1/agent/issues with query parameters.
func (c *TidinessManagerClient) GetIssues(ctx context.Context, scenario, file, category, severity string, limit int) ([]TidinessIssue, error) {
	params := url.Values{}
	params.Set("scenario", scenario)
	if file != "" {
		params.Set("file", file)
	}
	if category != "" {
		params.Set("category", category)
	}
	if severity != "" {
		params.Set("severity", severity)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var result []TidinessIssue
	err := c.doGet(ctx, "/api/v1/agent/issues?"+params.Encode(), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetStaleness calls GET /api/v1/agent/staleness?scenario=X.
func (c *TidinessManagerClient) GetStaleness(ctx context.Context, scenario string) (*TidinessStalenessInfo, error) {
	var result TidinessStalenessInfo
	err := c.doGet(ctx, "/api/v1/agent/staleness?scenario="+url.QueryEscape(scenario), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// TriggerLightScan calls POST /api/v1/scan/light.
func (c *TidinessManagerClient) TriggerLightScan(ctx context.Context, req TidinessLightScanRequest) (*TidinessLightScanResult, error) {
	var result TidinessLightScanResult
	err := c.doJSON(ctx, "/api/v1/scan/light", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetScenarioDetail calls GET /api/v1/agent/scenarios/{name}.
func (c *TidinessManagerClient) GetScenarioDetail(ctx context.Context, scenario string) (*TidinessScenarioDetail, error) {
	var result TidinessScenarioDetail
	err := c.doGet(ctx, "/api/v1/agent/scenarios/"+url.PathEscape(scenario), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *TidinessManagerClient) doJSON(ctx context.Context, path string, body, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve tidiness-manager url: %w", err)
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
		return fmt.Errorf("tidiness-manager request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseTidinessManagerError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *TidinessManagerClient) doGet(ctx context.Context, path string, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve tidiness-manager url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tidiness-manager request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseTidinessManagerError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func parseTidinessManagerError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("tidiness-manager error: %s", errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("tidiness-manager error: %s", errResp.Message)
		}
	}
	return fmt.Errorf("tidiness-manager error: status %d, body: %s", resp.StatusCode, string(body))
}
