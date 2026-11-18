package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// StartAgentChat creates a task and run for agent mode chat.
// This is the entry point for starting an agentic coding session.
// Bounded retries resolve the current agent-manager URL for each attempt.
func (c *AgentManagerClient) StartAgentChat(ctx context.Context, message string, cfg AgentChatConfig) (*AgentChatSession, error) {
	task, err := c.createTask(ctx, message, cfg)
	if err != nil {
		return nil, err
	}

	taskID := task.GetId()
	if taskID == "" {
		return nil, fmt.Errorf("task response missing id")
	}

	run, err := c.createRun(ctx, taskID, cfg)
	if err != nil {
		return nil, err
	}

	return &AgentChatSession{
		TaskID:    taskID,
		RunID:     run.GetId(),
		SessionID: run.GetSessionId(),
	}, nil
}

// createTask creates a new task in agent-manager for a chat session.
func (c *AgentManagerClient) createTask(ctx context.Context, message string, cfg AgentChatConfig) (*domainpb.Task, error) {
	taskReq := &apipb.CreateTaskRequest{
		Task: &domainpb.Task{
			Title:       "Agent Chat Session",
			Description: message,
			ScopePath:   cfg.ProjectPath,
			ProjectRoot: cfg.ProjectPath,
		},
	}
	taskResult, err := callAgentManager(ctx, c, false, func(callCtx context.Context, client apiconnect.AgentManagerServiceClient) (*apipb.CreateTaskResponse, error) {
		resp, err := client.CreateTask(callCtx, connect.NewRequest(taskReq))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	task := taskResult.GetTask()
	if task == nil {
		return nil, fmt.Errorf("task response missing task")
	}
	return task, nil
}

// createRun creates a new run for an existing task in agent-manager.
func (c *AgentManagerClient) createRun(ctx context.Context, taskID string, cfg AgentChatConfig) (*domainpb.Run, error) {
	const profileKey = "agent-inbox/default"
	// Let the profile's sandbox policy determine the run mode. An explicit
	// in-place override conflicts with Agent Manager's protected profile default.
	runReq := &apipb.CreateRunRequest{
		TaskId: taskID,
		ProfileRef: &apipb.ProfileRef{
			ProfileKey: profileKey,
		},
	}
	runResult, err := callAgentManager(ctx, c, false, func(callCtx context.Context, client apiconnect.AgentManagerServiceClient) (*apipb.CreateRunResponse, error) {
		resp, err := client.CreateRun(callCtx, connect.NewRequest(runReq))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}
	run := runResult.GetRun()
	if run == nil {
		return nil, fmt.Errorf("run response missing run")
	}
	return run, nil
}

// ContinueChat sends a follow-up message to an existing agent run.
// Uses the continuation API to maintain conversation state.
// attachmentIDs optionally references previously uploaded attachments to include.
func (c *AgentManagerClient) ContinueChat(ctx context.Context, runID, message string, attachmentIDs []string) error {
	reqProto := &domainpb.ContinueRunRequest{
		RunId:         runID,
		Message:       message,
		AttachmentIds: attachmentIDs,
	}
	_, err := callAgentManager(ctx, c, false, func(callCtx context.Context, client apiconnect.AgentManagerServiceClient) (*domainpb.ContinueRunResponse, error) {
		resp, err := client.ContinueRun(callCtx, connect.NewRequest(reqProto))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	if err != nil {
		return fmt.Errorf("continue failed: %w", err)
	}
	return nil
}

// UploadAttachment proxies a file upload to agent-manager's attachment endpoint.
// Uses a direct HTTP request (not doWithRetry) because multipart uploads require
// a custom Content-Type header with the boundary parameter.
func (c *AgentManagerClient) UploadAttachment(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*UploadResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", header.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}
	writer.Close()

	baseURL, err := c.getBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent-manager not available: %w", err)
	}
	uploadURL, err := url.JoinPath(baseURL, "/api/v1/attachments/upload")
	if err != nil {
		return nil, fmt.Errorf("build agent-manager upload URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed: %d: %s", resp.StatusCode, string(respBody))
	}

	var result UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode upload response: %w", err)
	}

	return &result, nil
}
