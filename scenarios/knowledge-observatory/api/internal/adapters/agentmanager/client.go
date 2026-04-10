// Package agentmanager provides an HTTP client for the agent-manager service.
package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Client is an HTTP client for the agent-manager API.
type Client struct {
	httpClient *http.Client
	jsonOpts   protojson.MarshalOptions
	baseURL    string
}

// NewClient creates a new agent-manager client.
func NewClient(timeout time.Duration) *Client {
	return NewClientWithBaseURL(timeout, "")
}

// NewClientWithBaseURL creates a new agent-manager client with a fixed base URL.
func NewClientWithBaseURL(timeout time.Duration, baseURL string) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		jsonOpts: protojson.MarshalOptions{
			UseProtoNames: false, // lowerCamelCase to match agent-manager HTTP handlers
		},
		baseURL: baseURL,
	}
}

// Health checks the agent-manager service health.
func (c *Client) Health(ctx context.Context) (bool, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/health", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// EnsureProfile resolves a profile by key, creating it with defaults if needed.
func (c *Client) EnsureProfile(ctx context.Context, req *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error) {
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/profiles/ensure", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result apipb.EnsureProfileResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	req := &apipb.CreateTaskRequest{Task: task}
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/tasks", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result apipb.CreateTaskResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Task, nil
}

// CreateRun starts a new run for a task.
func (c *Client) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/runs", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result apipb.CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/runs/%s", runID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.GetRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// GetRunDiff retrieves the diff for a run.
func (c *Client) GetRunDiff(ctx context.Context, runID string) (*domainpb.RunDiff, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/runs/%s/diff", runID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.GetRunDiffResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Diff, nil
}

// GetRunEvents retrieves events for a run.
func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	path := fmt.Sprintf("/api/v1/runs/%s/events", runID)
	if afterSequence > 0 {
		path = fmt.Sprintf("%s?after_sequence=%d", path, afterSequence)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.GetRunEventsResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Events, nil
}

// ApproveRun approves and applies changes from a sandboxed run.
func (c *Client) ApproveRun(ctx context.Context, req *apipb.ApproveRunRequest) (*apipb.ApproveRunResponse, error) {
	if req == nil || strings.TrimSpace(req.RunId) == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/runs/%s/approve", req.RunId), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	var result apipb.ApproveRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RejectRun rejects and discards changes from a sandboxed run.
func (c *Client) RejectRun(ctx context.Context, req *apipb.RejectRunRequest) (*apipb.RejectRunResponse, error) {
	if req == nil || strings.TrimSpace(req.RunId) == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/runs/%s/reject", req.RunId), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	var result apipb.RejectRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) resolveBaseURL(ctx context.Context) (string, error) {
	if strings.TrimSpace(c.baseURL) != "" {
		return c.baseURL, nil
	}
	url, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("resolve agent-manager url: %w", err)
	}
	return url, nil
}

func (c *Client) parseResponse(resp *http.Response, msg proto.Message) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(body, msg); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("agent-manager error: %s", errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("agent-manager error: %s", errResp.Message)
		}
	}
	return fmt.Errorf("agent-manager error: status %d, body: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
