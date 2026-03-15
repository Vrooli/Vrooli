// Package agentmanager provides a small HTTP client for agent-manager integration.
//
// DOC: docs/concepts/ARCHITECTURE.md#api-boundaries
// DOC: docs/internal/SEAMS.md
// DOC: docs/internal/INTEROP_AUDIT.md
package agentmanager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

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
		return nil, readErrorResponse(resp)
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
		return nil, readErrorResponse(resp)
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
		return nil, readErrorResponse(resp)
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

// GetRun retrieves a run by ID.
func (c *HTTPClient) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/runs/"+url.PathEscape(trimmed), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp)
	}

	var result apipb.GetRunResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.Run == nil || strings.TrimSpace(result.Run.Id) == "" {
		return nil, fmt.Errorf("%w: missing run id", ErrRequestFailed)
	}
	return result.Run, nil
}

// StopRun stops a running run by ID.
func (c *HTTPClient) StopRun(ctx context.Context, runID string) error {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/runs/"+url.PathEscape(trimmed)+"/stop", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readErrorResponse(resp)
	}

	return nil
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

// maxErrorBodyBytes limits how much of an error response body to read.
const maxErrorBodyBytes = 1024

// readErrorResponse builds an ErrRequestFailed error that includes the
// status code and (truncated) response body so callers can diagnose
// upstream validation failures.
func readErrorResponse(resp *http.Response) error {
	detail := ""
	if resp.Body != nil {
		limited := io.LimitReader(resp.Body, maxErrorBodyBytes)
		raw, _ := io.ReadAll(limited)
		detail = strings.TrimSpace(string(raw))
	}
	if detail != "" {
		return fmt.Errorf("%w: status %d: %s", ErrRequestFailed, resp.StatusCode, detail)
	}
	return fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
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
