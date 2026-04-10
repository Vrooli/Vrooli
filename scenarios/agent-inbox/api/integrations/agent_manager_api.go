package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// StartAgentChat creates a task and run for agent mode chat.
// This is the entry point for starting an agentic coding session.
// On connection failure, re-resolves the agent-manager URL and retries once.
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
	taskBody, err := protoMarshalOpts.Marshal(taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, "POST", "/api/v1/tasks", taskBody)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create task: %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResult apipb.CreateTaskResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &taskResult); err != nil {
		return nil, fmt.Errorf("failed to parse task response: %w", err)
	}

	task := taskResult.GetTask()
	if task == nil {
		return nil, fmt.Errorf("task response missing task")
	}
	return task, nil
}

// createRun creates a new run for an existing task in agent-manager.
func (c *AgentManagerClient) createRun(ctx context.Context, taskID string, cfg AgentChatConfig) (*domainpb.Run, error) {
	runMode := domainpb.RunMode_RUN_MODE_IN_PLACE
	profileKey := "agent-inbox-" + string(cfg.RunnerType)
	runReq := &apipb.CreateRunRequest{
		TaskId:  taskID,
		RunMode: &runMode,
		ProfileRef: &apipb.ProfileRef{
			ProfileKey: profileKey,
			Defaults: &domainpb.AgentProfile{
				ProfileKey: profileKey,
				Name:       profileKey,
				RunnerType: localRunnerTypeToProto(cfg.RunnerType),
				Model:      cfg.Model,
				MaxTurns:   int32(cfg.MaxTurns),
			},
		},
	}
	runBody, err := protoMarshalOpts.Marshal(runReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, "POST", "/api/v1/runs", runBody)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create run: %d: %s", resp.StatusCode, string(respBody))
	}

	var runResult apipb.CreateRunResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &runResult); err != nil {
		return nil, fmt.Errorf("failed to parse run response: %w", err)
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
	body, err := protoMarshalOpts.Marshal(reqProto)
	if err != nil {
		return fmt.Errorf("failed to marshal continue request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, "POST", "/api/v1/runs/"+runID+"/continue", body)
	if err != nil {
		return fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("continue failed: %d: %s", resp.StatusCode, string(respBody))
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

	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, fmt.Errorf("agent-manager not available: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/attachments/upload", &buf)
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
