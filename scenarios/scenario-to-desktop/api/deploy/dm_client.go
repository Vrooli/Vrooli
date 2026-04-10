package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// DMClient calls deployment-manager approval endpoints.
type DMClient struct {
	resolver   BaseURLResolver
	httpClient HTTPDoer
}

// NewDMClient creates a client that discovers deployment-manager via service discovery.
func NewDMClient(ctx context.Context) (*DMClient, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
	if err != nil {
		return nil, fmt.Errorf("resolve deployment-manager URL: %w", err)
	}
	return &DMClient{
		resolver:   func(_ context.Context) (string, error) { return baseURL, nil },
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// NewDMClientWithResolver creates a client with custom resolver and HTTP client (for testing).
func NewDMClientWithResolver(resolver BaseURLResolver, httpClient HTTPDoer) *DMClient {
	return &DMClient{
		resolver:   resolver,
		httpClient: httpClient,
	}
}

// Approval represents a deployment approval record.
type Approval struct {
	ID            string `json:"id"`
	ProfileID     string `json:"profile_id"`
	GitCommitHash string `json:"git_commit_hash"`
	Platform      string `json:"platform"`
	Status        string `json:"status"`
	Reviewer      string `json:"reviewer,omitempty"`
	Notes         string `json:"notes,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	DecidedAt     string `json:"decided_at,omitempty"`
}

// PlatformGateStatus describes the gate status for a single platform.
type PlatformGateStatus struct {
	Platform string `json:"platform"`
	Status   string `json:"status"`
	Ready    bool   `json:"ready"`
}

// ReleaseGateStatus describes whether a release is cleared to deploy.
type ReleaseGateStatus struct {
	Ready     bool                 `json:"ready"`
	Platforms []PlatformGateStatus `json:"platforms"`
}

// BlockedPlatforms returns the platforms that are not yet approved.
func (g *ReleaseGateStatus) BlockedPlatforms() []string {
	var blocked []string
	for _, p := range g.Platforms {
		if !p.Ready {
			blocked = append(blocked, p.Platform)
		}
	}
	return blocked
}

// CreateApproval creates a pending approval for a platform build.
// Returns the existing approval on 409 (idempotent).
func (c *DMClient) CreateApproval(ctx context.Context, profileID, commitHash, platform string) (*Approval, error) {
	body := map[string]string{
		"git_commit_hash": commitHash,
		"platform":        platform,
	}
	respBody, statusCode, err := c.request(ctx, "POST",
		fmt.Sprintf("/api/v1/profiles/%s/approvals", url.PathEscape(profileID)), body)
	if err != nil {
		return nil, fmt.Errorf("create approval: %w", err)
	}

	if statusCode == http.StatusConflict {
		var approval Approval
		if err := json.Unmarshal(respBody, &approval); err != nil {
			return nil, fmt.Errorf("decode existing approval (409): %w", err)
		}
		return &approval, nil
	}

	var approval Approval
	if err := json.Unmarshal(respBody, &approval); err != nil {
		return nil, fmt.Errorf("decode approval: %w", err)
	}
	return &approval, nil
}

// CheckReleaseGate checks whether the release gate is satisfied for a commit.
func (c *DMClient) CheckReleaseGate(ctx context.Context, profileID, commitHash string) (*ReleaseGateStatus, error) {
	path := fmt.Sprintf("/api/v1/profiles/%s/release-gate?commit=%s",
		url.PathEscape(profileID), url.QueryEscape(commitHash))
	respBody, _, err := c.request(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("check release gate: %w", err)
	}
	var status ReleaseGateStatus
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("decode release gate: %w", err)
	}
	return &status, nil
}

// ListApprovals lists approvals for a commit.
func (c *DMClient) ListApprovals(ctx context.Context, profileID, commitHash string) ([]Approval, error) {
	path := fmt.Sprintf("/api/v1/profiles/%s/approvals?commit=%s",
		url.PathEscape(profileID), url.QueryEscape(commitHash))
	respBody, _, err := c.request(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	var approvals []Approval
	if err := json.Unmarshal(respBody, &approvals); err != nil {
		return nil, fmt.Errorf("decode approvals: %w", err)
	}
	return approvals, nil
}

// request makes an HTTP request to deployment-manager.
// Returns (body, statusCode, error). For 4xx/5xx responses other than 409,
// an error is returned. 409 is returned as a non-error so callers can handle idempotency.
func (c *DMClient) request(ctx context.Context, method, path string, payload interface{}) ([]byte, int, error) {
	baseURL, err := c.resolver(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("resolve deployment-manager URL: %w", err)
	}

	fullURL := strings.TrimRight(baseURL, "/") + path

	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("encode request: %w", err)
		}
		bodyReader = strings.NewReader(string(data))
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusConflict {
		return respBody, resp.StatusCode, nil
	}

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("%s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}
