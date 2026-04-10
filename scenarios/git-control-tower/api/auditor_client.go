package main

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// AuditorClient is a lightweight HTTP client for scenario-auditor APIs.
type AuditorClient struct {
	BaseClient
}

// NewAuditorClient creates a new auditor client with the given timeout.
func NewAuditorClient(timeout time.Duration) *AuditorClient {
	return &AuditorClient{
		BaseClient: NewBaseClient("scenario-auditor", timeout),
	}
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
