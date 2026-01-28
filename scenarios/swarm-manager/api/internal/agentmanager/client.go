// Package agentmanager provides a small HTTP client for agent-manager integration.
package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// ResearchRequest describes a research task to be executed by agent-manager.
type ResearchRequest struct {
	Title       string
	Description string
	Prompt      string
	ScopePath   string
	ProjectRoot string
	Tag         string
	CreatedBy   string
}

// ResearchResponse describes the created agent run.
type ResearchResponse struct {
	TaskID    string `json:"taskId"`
	RunID     string `json:"runId"`
	BaseURL   string `json:"baseUrl"`
	CreatedAt string `json:"created"`
}

// Client defines the seam for agent-manager integration.
type Client interface {
	CreateResearchRun(ctx context.Context, req ResearchRequest) (ResearchResponse, error)
}

// HTTPClient implements Client using HTTP calls to agent-manager.
type HTTPClient struct {
	baseURLResolver BaseURLResolver
	httpClient      HTTPDoer
}

// BaseURLResolver resolves the base URL for agent-manager.
type BaseURLResolver func(ctx context.Context) (string, error)

// HTTPDoer allows injecting HTTP client for tests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewHTTPClient creates a new agent-manager HTTP client.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		baseURLResolver: resolveAgentManagerBaseURL,
		httpClient:      &http.Client{Timeout: 20 * time.Second},
	}
}

// NewHTTPClientWithResolver creates a client with custom resolver (for tests).
func NewHTTPClientWithResolver(resolver BaseURLResolver, httpClient HTTPDoer) *HTTPClient {
	if resolver == nil {
		resolver = resolveAgentManagerBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &HTTPClient{
		baseURLResolver: resolver,
		httpClient:      httpClient,
	}
}

// ErrNotAvailable is returned when agent-manager cannot be reached.
var ErrNotAvailable = errors.New("agent-manager not available")

// ErrRequestFailed is returned when agent-manager returns a non-2xx response.
var ErrRequestFailed = errors.New("agent-manager request failed")

// CreateResearchRun creates a task and run in agent-manager.
func (c *HTTPClient) CreateResearchRun(ctx context.Context, req ResearchRequest) (ResearchResponse, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return ResearchResponse{}, err
	}

	taskID, err := c.createTask(ctx, baseURL, req)
	if err != nil {
		return ResearchResponse{}, err
	}

	runID, err := c.createRun(ctx, baseURL, taskID, req)
	if err != nil {
		return ResearchResponse{}, err
	}

	return ResearchResponse{
		TaskID:    taskID,
		RunID:     runID,
		BaseURL:   baseURL,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

type createTaskRequest struct {
	Task taskPayload `json:"task"`
}

type taskPayload struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ScopePath   string `json:"scope_path"`
	ProjectRoot string `json:"project_root,omitempty"`
	CreatedBy   string `json:"created_by,omitempty"`
}

type createTaskResponse struct {
	Task struct {
		ID string `json:"id"`
	} `json:"task"`
}

func (c *HTTPClient) createTask(ctx context.Context, baseURL string, req ResearchRequest) (string, error) {
	payload := createTaskRequest{
		Task: taskPayload{
			Title:       strings.TrimSpace(req.Title),
			Description: strings.TrimSpace(req.Description),
			ScopePath:   strings.TrimSpace(req.ScopePath),
			ProjectRoot: strings.TrimSpace(req.ProjectRoot),
			CreatedBy:   strings.TrimSpace(req.CreatedBy),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/tasks", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
	}

	var result createTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if strings.TrimSpace(result.Task.ID) == "" {
		return "", fmt.Errorf("%w: missing task id", ErrRequestFailed)
	}

	return result.Task.ID, nil
}

type createRunRequest struct {
	TaskID string  `json:"task_id"`
	Prompt *string `json:"prompt,omitempty"`
	Tag    *string `json:"tag,omitempty"`
}

type createRunResponse struct {
	Run struct {
		ID string `json:"id"`
	} `json:"run"`
}

func (c *HTTPClient) createRun(ctx context.Context, baseURL, taskID string, req ResearchRequest) (string, error) {
	prompt := strings.TrimSpace(req.Prompt)
	tag := strings.TrimSpace(req.Tag)

	payload := createRunRequest{
		TaskID: taskID,
	}
	if prompt != "" {
		payload.Prompt = &prompt
	}
	if tag != "" {
		payload.Tag = &tag
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/runs", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
	}

	var result createRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if strings.TrimSpace(result.Run.ID) == "" {
		return "", fmt.Errorf("%w: missing run id", ErrRequestFailed)
	}

	return result.Run.ID, nil
}

func resolveAgentManagerBaseURL(ctx context.Context) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	return baseURL, nil
}
