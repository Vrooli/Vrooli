package main

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

// TidinessManagerClient is a lightweight HTTP client for tidiness-manager APIs.
type TidinessManagerClient struct {
	BaseClient
}

// NewTidinessManagerClient creates a new tidiness-manager client with the given timeout.
func NewTidinessManagerClient(timeout time.Duration) *TidinessManagerClient {
	return &TidinessManagerClient{
		BaseClient: NewBaseClient("tidiness-manager", timeout),
	}
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
