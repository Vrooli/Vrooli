// Package agentmanager is a small HTTP client + integration seam for spawning
// and polling agent-manager runs from web-search's L3 research path. It mirrors
// the swarm-manager agentmanager client (CreateTask -> CreateRun -> poll
// GetRun) trimmed to what L3 needs. The Client interface is the seam the L3
// agent depends on, so tests fake it with canned run ids/status and no live
// agent-manager.
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

// Client is the agent-manager seam the L3 research path depends on.
type Client interface {
	// CreateTask creates a task and returns it (with a server-assigned id).
	CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error)
	// CreateRun starts a run for a task and returns it (with a run id + status).
	CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error)
	// GetRun fetches a run by id for status polling.
	GetRun(ctx context.Context, runID string) (*domainpb.Run, error)
}

// HTTPDoer allows injecting an HTTP client for tests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// BaseURLResolver resolves the base URL for agent-manager.
type BaseURLResolver func(ctx context.Context) (string, error)

// HTTPClient implements Client over HTTP calls to agent-manager.
type HTTPClient struct {
	baseURLResolver BaseURLResolver
	httpClient      HTTPDoer
}

// ErrNotAvailable is returned when agent-manager cannot be reached.
var ErrNotAvailable = errors.New("agent-manager not available")

// ErrRequestFailed is returned when agent-manager returns a non-2xx response.
var ErrRequestFailed = errors.New("agent-manager request failed")

var (
	protoJSONMarshal   = protojson.MarshalOptions{UseProtoNames: false}
	protoJSONUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// NewHTTPClient creates an agent-manager HTTP client with the default 20s
// timeout and the standard discovery-based URL resolver.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		baseURLResolver: resolveAgentManagerBaseURL,
		httpClient:      &http.Client{Timeout: 20 * time.Second},
	}
}

// NewHTTPClientWithResolver creates a client with a custom resolver and doer
// (for tests).
func NewHTTPClientWithResolver(resolver BaseURLResolver, doer HTTPDoer) *HTTPClient {
	if resolver == nil {
		resolver = resolveAgentManagerBaseURL
	}
	if doer == nil {
		doer = &http.Client{Timeout: 20 * time.Second}
	}
	return &HTTPClient{baseURLResolver: resolver, httpClient: doer}
}

// ResolveURL returns the agent-manager base URL.
func (c *HTTPClient) ResolveURL(ctx context.Context) (string, error) {
	return c.baseURLResolver(ctx)
}

// CreateTask creates a task in agent-manager.
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
	defer func() { _ = resp.Body.Close() }()
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

// CreateRun starts a run in agent-manager.
func (c *HTTPClient) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	body, err := protoJSONMarshal.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/runs", body)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
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

// GetRun fetches a run by id.
func (c *HTTPClient) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/runs/"+url.PathEscape(trimmed), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
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
		return nil, fmt.Errorf("%w: %w", ErrNotAvailable, err)
	}
	return resp, nil
}

const maxErrorBodyBytes = 1024

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
		return "", fmt.Errorf("%w: %w", ErrNotAvailable, err)
	}
	return baseURL, nil
}

// Compile-time guarantee that the HTTP client satisfies the seam.
var _ Client = (*HTTPClient)(nil)
