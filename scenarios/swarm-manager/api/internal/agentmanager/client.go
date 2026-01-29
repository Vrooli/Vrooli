// Package agentmanager provides a small HTTP client for agent-manager integration.
package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	return NewHTTPClientWithTimeout(20 * time.Second)
}

// NewHTTPClientWithTimeout creates a new agent-manager HTTP client with custom timeout.
func NewHTTPClientWithTimeout(timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &HTTPClient{
		baseURLResolver: resolveAgentManagerBaseURL,
		httpClient:      &http.Client{Timeout: timeout},
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

var protoJSONMarshal = protojson.MarshalOptions{
	UseProtoNames: false, // match existing clients (lowerCamelCase)
}

var protoJSONUnmarshal = protojson.UnmarshalOptions{
	DiscardUnknown: false,
}

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

// ResolveURL returns the base URL for agent-manager.
func (c *HTTPClient) ResolveURL(ctx context.Context) (string, error) {
	return c.baseURLResolver(ctx)
}

// Health checks agent-manager availability.
func (c *HTTPClient) Health(ctx context.Context) (bool, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/health", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// EnsureProfile resolves or creates a profile by key.
func (c *HTTPClient) EnsureProfile(ctx context.Context, req *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error) {
	body, err := protoJSONMarshal.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/profiles/ensure", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
	}

	var result apipb.EnsureProfileResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTask creates a new task using proto JSON payloads.
func (c *HTTPClient) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	req := &apipb.CreateTaskRequest{Task: task}
	body, err := protoJSONMarshal.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/tasks", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
	}

	var result apipb.CreateTaskResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.Task == nil || strings.TrimSpace(result.Task.Id) == "" {
		return nil, fmt.Errorf("%w: missing task id", ErrRequestFailed)
	}
	return result.Task, nil
}

// CreateRun creates a new run using proto JSON payloads.
func (c *HTTPClient) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	body, err := protoJSONMarshal.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/runs", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
	}

	var result apipb.CreateRunResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.Run == nil || strings.TrimSpace(result.Run.Id) == "" {
		return nil, fmt.Errorf("%w: missing run id", ErrRequestFailed)
	}
	return result.Run, nil
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

func (c *HTTPClient) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	return resp, nil
}

func decodeProtoResponse(resp *http.Response, msg proto.Message) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return protoJSONUnmarshal.Unmarshal(body, msg)
}

func resolveAgentManagerBaseURL(ctx context.Context) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	return baseURL, nil
}
