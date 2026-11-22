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
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
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
}

func NewAgentManagerClient(timeout time.Duration) *AgentManagerClient {
	return &AgentManagerClient{
		BaseClient: NewBaseClient("agent-manager", timeout),
		maxRetries: 2, retryBaseDelay: 200 * time.Millisecond,
	}
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

func (c *AgentManagerClient) connectClient(ctx context.Context) (apiconnect.AgentManagerServiceClient, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve agent-manager url: %w", err)
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	return apiconnect.NewAgentManagerServiceClient(httpClient, baseURL), nil
}

// callConnect retries only transport-level failures, clearing the cached
// address before each retry so discovery observes Agent Manager port changes.
// Application errors retain their Connect status and are never retried.
func callConnect[T any](ctx context.Context, c *AgentManagerClient, call func(apiconnect.AgentManagerServiceClient) (*T, error)) (*T, error) {
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

func agentManagerRequest[T any](ctx context.Context, message *T) *connect.Request[T] {
	request := connect.NewRequest(message)
	for name, values := range policygate.VerifiedOwnerHeaders(ctx) {
		request.Header()[name] = append([]string(nil), values...)
	}
	return request
}

func (c *AgentManagerClient) ListProfiles(ctx context.Context) (*apipb.ListProfilesResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.ListProfilesResponse, error) {
		response, err := client.ListProfiles(ctx, agentManagerRequest(ctx, &apipb.ListProfilesRequest{}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) CreateTask(ctx context.Context, task *ampb.Task) (*apipb.CreateTaskResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.CreateTaskResponse, error) {
		response, err := client.CreateTask(ctx, agentManagerRequest(ctx, &apipb.CreateTaskRequest{Task: task}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) GetTask(ctx context.Context, taskID string) (*apipb.GetTaskResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.GetTaskResponse, error) {
		response, err := client.GetTask(ctx, agentManagerRequest(ctx, &apipb.GetTaskRequest{TaskId: taskID}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) CreateRun(ctx context.Context, request *apipb.CreateRunRequest) (*apipb.CreateRunResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.CreateRunResponse, error) {
		response, err := client.CreateRun(ctx, agentManagerRequest(ctx, request))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) GetRun(ctx context.Context, runID string) (*apipb.GetRunResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.GetRunResponse, error) {
		response, err := client.GetRun(ctx, agentManagerRequest(ctx, &apipb.GetRunRequest{RunId: runID}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) GetRunReport(ctx context.Context, runID string) (*apipb.RunReport, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.RunReport, error) {
		response, err := client.GetRunReport(ctx, agentManagerRequest(ctx, &apipb.GetRunReportRequest{RunId: runID}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) GetRunEvents(ctx context.Context, runID string, afterSequence, limit int) (*apipb.GetRunEventsResponse, error) {
	after, pageSize := int64(afterSequence), int32(limit)
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.GetRunEventsResponse, error) {
		response, err := client.GetRunEvents(ctx, agentManagerRequest(ctx, &apipb.GetRunEventsRequest{
			RunId: runID, AfterSequence: &after, Limit: &pageSize,
		}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) GetRunDiff(ctx context.Context, runID string) (*apipb.GetRunDiffResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.GetRunDiffResponse, error) {
		response, err := client.GetRunDiff(ctx, agentManagerRequest(ctx, &apipb.GetRunDiffRequest{RunId: runID}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) ContinueRun(ctx context.Context, runID string, request AgentContinueRequest) (*ampb.ContinueRunResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*ampb.ContinueRunResponse, error) {
		response, err := client.ContinueRun(ctx, agentManagerRequest(ctx, &ampb.ContinueRunRequest{
			RunId: runID, Message: request.Message, AttachmentIds: request.AttachmentIDs,
		}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) ApproveRun(ctx context.Context, runID string, request AgentApproveRequest) (*apipb.ApproveRunResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.ApproveRunResponse, error) {
		response, err := client.ApproveRun(ctx, agentManagerRequest(ctx, &apipb.ApproveRunRequest{
			RunId: runID, Actor: optionalString(request.Actor), CommitMsg: optionalString(request.CommitMsg),
		}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) RejectRun(ctx context.Context, runID string, request AgentRejectRequest) (*apipb.RejectRunResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.RejectRunResponse, error) {
		response, err := client.RejectRun(ctx, agentManagerRequest(ctx, &apipb.RejectRunRequest{
			RunId: runID, Actor: optionalString(request.Actor), Reason: request.Reason,
		}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) StopRun(ctx context.Context, runID string) (*apipb.StopRunResponse, error) {
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.StopRunResponse, error) {
		response, err := client.StopRun(ctx, agentManagerRequest(ctx, &apipb.StopRunRequest{RunId: runID}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
}

func (c *AgentManagerClient) ListRuns(ctx context.Context, tagPrefix string, limit int) (*apipb.ListRunsResponse, error) {
	prefix, pageSize := tagPrefix, int32(limit)
	return callConnect(ctx, c, func(client apiconnect.AgentManagerServiceClient) (*apipb.ListRunsResponse, error) {
		response, err := client.ListRuns(ctx, agentManagerRequest(ctx, &apipb.ListRunsRequest{TagPrefix: &prefix, Limit: &pageSize}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
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
