package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/policygate"
	agentmanagerclient "github.com/vrooli/vrooli/packages/agentmanager-client-go"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// AgentManagerClient is the generated Connect client for Agent Manager. The
// attachment upload endpoint remains HTTP because its wire format is multipart.
type AgentManagerClient struct {
	BaseClient

	mu             sync.Mutex
	cachedBaseURL  string
	resolveURL     func(context.Context) (string, error)
	maxRetries     int
	retryBaseDelay time.Duration
	shared         *agentmanagerclient.Client
}

type agentManagerResolver struct{ owner *AgentManagerClient }

func (r agentManagerResolver) ResolveBaseURL(ctx context.Context) (string, error) {
	return r.owner.resolveBaseURL(ctx)
}

func NewAgentManagerClient(timeout time.Duration) *AgentManagerClient {
	client := &AgentManagerClient{
		BaseClient: NewBaseClient("agent-manager", timeout),
		maxRetries: 2, retryBaseDelay: 200 * time.Millisecond,
	}
	client.sharedClient()
	return client
}

func (c *AgentManagerClient) sharedClient() *agentmanagerclient.Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.shared != nil {
		return c.shared
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	c.shared = agentmanagerclient.NewWithResolver(0, agentManagerResolver{owner: c},
		agentmanagerclient.WithHTTPClient(httpClient),
		agentmanagerclient.WithProto(),
		agentmanagerclient.WithRequestHeaders(policygate.VerifiedOwnerHeaders),
	)
	return c.shared
}

func (c *AgentManagerClient) resolveBaseURL(ctx context.Context) (string, error) {
	if c.resolveURL != nil {
		return c.resolveURL(ctx)
	}
	c.mu.Lock()
	cached := c.cachedBaseURL
	c.mu.Unlock()
	if cached != "" {
		return cached, nil
	}
	url, err := c.BaseClient.resolveBaseURL(ctx)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.cachedBaseURL = url
	c.mu.Unlock()
	return url, nil
}

func (c *AgentManagerClient) clearCachedURL() {
	c.mu.Lock()
	c.cachedBaseURL = ""
	c.mu.Unlock()
}

func (c *AgentManagerClient) connectClient(ctx context.Context) (agentmanagerclient.Service, error) {
	return c.sharedClient().Connect(ctx)
}

// callConnect retries only transport-level failures, clearing the cached
// address before each retry so discovery observes Agent Manager port changes.
// Application errors retain their Connect status and are never retried.
func callConnect[T any](ctx context.Context, c *AgentManagerClient, call func(agentmanagerclient.Service) (*T, error)) (*T, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, err := c.connectClient(ctx)
		if err != nil {
			lastErr = err
		} else {
			result, callErr := call(client)
			if callErr == nil {
				return result, nil
			}
			lastErr = callErr
			code := connect.CodeOf(callErr)
			if code != connect.CodeUnavailable && code != connect.CodeDeadlineExceeded {
				return nil, callErr
			}
		}
		if attempt == c.maxRetries {
			break
		}
		c.clearCachedURL()
		if err := waitForAgentManagerRetry(ctx, c.retryBaseDelay, attempt); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("agent-manager Connect retry exhausted after %d attempts: %w", c.maxRetries+1, lastErr)
}

func waitForAgentManagerRetry(ctx context.Context, base time.Duration, attempt int) error {
	delay := base
	for i := 0; i < attempt && delay < 2*time.Second; i++ {
		delay *= 2
	}
	if delay > 2*time.Second {
		delay = 2 * time.Second
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *AgentManagerClient) ListProfiles(ctx context.Context) (*apipb.ListProfilesResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.ListProfilesResponse, error) {
		return client.ListProfiles(ctx)
	})
}

func (c *AgentManagerClient) CreateTask(ctx context.Context, task *ampb.Task) (*apipb.CreateTaskResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.CreateTaskResponse, error) {
		return client.CreateTaskResponse(ctx, task)
	})
}

func (c *AgentManagerClient) GetTask(ctx context.Context, taskID string) (*apipb.GetTaskResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.GetTaskResponse, error) {
		return client.GetTask(ctx, taskID)
	})
}

func (c *AgentManagerClient) CreateRun(ctx context.Context, request *apipb.CreateRunRequest) (*apipb.CreateRunResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.CreateRunResponse, error) {
		return client.CreateRunResponse(ctx, request)
	})
}

func (c *AgentManagerClient) GetRun(ctx context.Context, runID string) (*apipb.GetRunResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.GetRunResponse, error) {
		return client.GetRunResponse(ctx, runID)
	})
}

func (c *AgentManagerClient) GetRunReport(ctx context.Context, runID string) (*apipb.RunReport, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.RunReport, error) {
		return client.GetRunReport(ctx, runID)
	})
}

func (c *AgentManagerClient) GetRunEvents(ctx context.Context, runID string, afterSequence, limit int) (*apipb.GetRunEventsResponse, error) {
	after, pageSize := int64(afterSequence), int32(limit)
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.GetRunEventsResponse, error) {
		return client.GetRunEventsResponse(ctx, runID, agentmanagerclient.RunEventsOptions{AfterSequence: &after, Limit: &pageSize})
	})
}

func (c *AgentManagerClient) GetRunDiff(ctx context.Context, runID string) (*apipb.GetRunDiffResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.GetRunDiffResponse, error) {
		return client.GetRunDiffResponse(ctx, runID)
	})
}

func (c *AgentManagerClient) ContinueRun(ctx context.Context, runID string, request AgentContinueRequest) (*ampb.ContinueRunResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*ampb.ContinueRunResponse, error) {
		return client.ContinueRun(ctx, runID, request.Message, request.AttachmentIDs)
	})
}

func (c *AgentManagerClient) ApproveRun(ctx context.Context, runID string, request AgentApproveRequest) (*apipb.ApproveRunResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.ApproveRunResponse, error) {
		return client.ApproveRun(ctx, &apipb.ApproveRunRequest{
			RunId: runID, Actor: optionalString(request.Actor), CommitMsg: optionalString(request.CommitMsg),
		})
	})
}

func (c *AgentManagerClient) RejectRun(ctx context.Context, runID string, request AgentRejectRequest) (*apipb.RejectRunResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.RejectRunResponse, error) {
		return client.RejectRun(ctx, &apipb.RejectRunRequest{
			RunId: runID, Actor: optionalString(request.Actor), Reason: request.Reason,
		})
	})
}

func (c *AgentManagerClient) StopRun(ctx context.Context, runID string) (*apipb.StopRunResponse, error) {
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.StopRunResponse, error) {
		return client.StopRunResponse(ctx, runID)
	})
}

func (c *AgentManagerClient) ListRuns(ctx context.Context, tagPrefix string, limit int) (*apipb.ListRunsResponse, error) {
	prefix, pageSize := tagPrefix, int32(limit)
	return callConnect(ctx, c, func(client agentmanagerclient.Service) (*apipb.ListRunsResponse, error) {
		return client.ListRunsResponse(ctx, &apipb.ListRunsRequest{TagPrefix: &prefix, Limit: &pageSize})
	})
}

// UploadAttachment keeps Agent Manager's multipart contract and forwards only
// the already verified human credential, matching the Connect owner requests.
func (c *AgentManagerClient) UploadAttachment(ctx context.Context, body io.Reader, contentType string) (*wireUploadAttachmentResponse, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve agent-manager url: %w", err)
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/attachments/upload", body)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Accept", "application/json")
	for name, values := range policygate.VerifiedOwnerHeaders(ctx) {
		request.Header[name] = append([]string(nil), values...)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		c.clearCachedURL()
		return nil, fmt.Errorf("agent-manager upload failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return nil, c.parseError(response)
	}
	var result wireUploadAttachmentResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}
	return &result, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
