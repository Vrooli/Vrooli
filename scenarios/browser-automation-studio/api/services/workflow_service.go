package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless"
	"github.com/vrooli/browser-automation-studio/browserless/events"
	"github.com/vrooli/browser-automation-studio/database"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const (
	workflowJSONStartMarker = "<WORKFLOW_JSON>"
	workflowJSONEndMarker   = "</WORKFLOW_JSON>"
)

// WorkflowService handles workflow business logic
type WorkflowService struct {
	repo        database.Repository
	browserless *browserless.Client
	wsHub       *wsHub.Hub
	log         *logrus.Logger
	aiClient    *OpenRouterClient
}

// AIWorkflowError represents a structured error returned by the AI generator when
// it cannot produce a valid workflow definition for the given prompt.
type AIWorkflowError struct {
	Reason string
}

// Error implements the error interface.
func (e *AIWorkflowError) Error() string {
	return e.Reason
}

// ExecutionExportPreview summarises the export readiness state for an execution.
type ExecutionExportPreview struct {
	ExecutionID uuid.UUID            `json:"execution_id"`
	Status      string               `json:"status"`
	Message     string               `json:"message"`
	Package     *ReplayExportPackage `json:"package,omitempty"`
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(repo database.Repository, browserless *browserless.Client, wsHub *wsHub.Hub, log *logrus.Logger) *WorkflowService {
	return &WorkflowService{
		repo:        repo,
		browserless: browserless,
		wsHub:       wsHub,
		log:         log,
		aiClient:    NewOpenRouterClient(log),
	}
}

// CheckHealth checks the health of all dependencies
func (s *WorkflowService) CheckHealth() string {
	status := "healthy"

	// Check browserless health
	if err := s.browserless.CheckBrowserlessHealth(); err != nil {
		s.log.WithError(err).Warn("Browserless health check failed")
		status = "degraded"
	}

	return status
}

// Project methods

// CreateProject creates a new project
func (s *WorkflowService) CreateProject(ctx context.Context, project *database.Project) error {
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	return s.repo.CreateProject(ctx, project)
}

// GetProject gets a project by ID
func (s *WorkflowService) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return s.repo.GetProject(ctx, id)
}

// GetProjectByName gets a project by name
func (s *WorkflowService) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return s.repo.GetProjectByName(ctx, name)
}

// GetProjectByFolderPath gets a project by folder path
func (s *WorkflowService) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return s.repo.GetProjectByFolderPath(ctx, folderPath)
}

// UpdateProject updates a project
func (s *WorkflowService) UpdateProject(ctx context.Context, project *database.Project) error {
	project.UpdatedAt = time.Now()
	return s.repo.UpdateProject(ctx, project)
}

// DeleteProject deletes a project and all its workflows
func (s *WorkflowService) DeleteProject(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteProject(ctx, id)
}

// ListProjects lists all projects
func (s *WorkflowService) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return s.repo.ListProjects(ctx, limit, offset)
}

// GetProjectStats gets statistics for a project
func (s *WorkflowService) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]interface{}, error) {
	return s.repo.GetProjectStats(ctx, projectID)
}

// ListWorkflowsByProject lists workflows for a specific project
func (s *WorkflowService) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return s.repo.ListWorkflowsByProject(ctx, projectID, limit, offset)
}

// DeleteProjectWorkflows deletes a set of workflows within a project boundary
func (s *WorkflowService) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	if len(workflowIDs) == 0 {
		return nil
	}
	return s.repo.DeleteProjectWorkflows(ctx, projectID, workflowIDs)
}

// Workflow methods

// CreateWorkflow creates a new workflow
func (s *WorkflowService) CreateWorkflow(ctx context.Context, name, folderPath string, flowDefinition map[string]interface{}, aiPrompt string) (*database.Workflow, error) {
	workflow := &database.Workflow{
		ID:         uuid.New(),
		Name:       name,
		FolderPath: folderPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Tags:       []string{},
		Version:    1,
	}

	if aiPrompt != "" {
		generated, genErr := s.generateWorkflowFromPrompt(ctx, aiPrompt)
		if genErr != nil {
			return nil, fmt.Errorf("ai workflow generation failed: %w", genErr)
		}
		workflow.FlowDefinition = generated
	} else if flowDefinition != nil {
		workflow.FlowDefinition = database.JSONMap(flowDefinition)
	} else {
		workflow.FlowDefinition = database.JSONMap{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		}
	}

	if err := s.repo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// CreateWorkflowWithProject creates a new workflow with optional project association
func (s *WorkflowService) CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]interface{}, aiPrompt string) (*database.Workflow, error) {
	workflow := &database.Workflow{
		ID:         uuid.New(),
		ProjectID:  projectID,
		Name:       name,
		FolderPath: folderPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Tags:       []string{},
		Version:    1,
	}

	if aiPrompt != "" {
		generated, genErr := s.generateWorkflowFromPrompt(ctx, aiPrompt)
		if genErr != nil {
			return nil, fmt.Errorf("ai workflow generation failed: %w", genErr)
		}
		workflow.FlowDefinition = generated
	} else if flowDefinition != nil {
		workflow.FlowDefinition = database.JSONMap(flowDefinition)
	} else {
		workflow.FlowDefinition = database.JSONMap{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		}
	}

	if err := s.repo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// ListWorkflows lists workflows with optional filtering
func (s *WorkflowService) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return s.repo.ListWorkflows(ctx, folderPath, limit, offset)
}

// GetWorkflow gets a workflow by ID
func (s *WorkflowService) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return s.repo.GetWorkflow(ctx, id)
}

// ExecuteWorkflow executes a workflow
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]interface{}) (*database.Execution, error) {
	// Verify workflow exists
	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	execution := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      workflowID,
		WorkflowVersion: workflow.Version,
		Status:          "pending",
		TriggerType:     "manual",
		Parameters:      database.JSONMap(parameters),
		StartedAt:       time.Now(),
		Progress:        0,
		CurrentStep:     "Initializing",
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, err
	}

	// Start async execution
	go s.executeWorkflowAsync(execution, workflow)

	return execution, nil
}

// DescribeExecutionExport returns the current replay export status for an execution.
func (s *WorkflowService) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*ExecutionExportPreview, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}

	var workflow *database.Workflow
	if wf, wfErr := s.repo.GetWorkflow(ctx, execution.WorkflowID); wfErr == nil {
		workflow = wf
	} else if !errors.Is(wfErr, database.ErrNotFound) {
		return nil, wfErr
	}

	timeline, err := s.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		return nil, err
	}

	exportPackage, err := BuildReplayExport(execution, workflow, timeline)
	if err != nil {
		return nil, err
	}

	frameCount := exportPackage.Summary.FrameCount
	if frameCount == 0 {
		frameCount = len(timeline.Frames)
	}
	message := fmt.Sprintf("Replay export ready (%d frames, %dms)", frameCount, exportPackage.Summary.TotalDurationMs)

	return &ExecutionExportPreview{
		ExecutionID: execution.ID,
		Status:      "ready",
		Message:     message,
		Package:     exportPackage,
	}, nil
}

// GetExecutionScreenshots gets screenshots for an execution
func (s *WorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return s.repo.GetExecutionScreenshots(ctx, executionID)
}

// GetExecution gets an execution by ID
func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return s.repo.GetExecution(ctx, id)
}

// ListExecutions lists executions with optional workflow filtering
func (s *WorkflowService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return s.repo.ListExecutions(ctx, workflowID, limit, offset)
}

// StopExecution stops a running execution
func (s *WorkflowService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	// Get current execution
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return err
	}

	// Only stop if currently running
	if execution.Status != "running" && execution.Status != "pending" {
		return nil // Already stopped/completed
	}

	// Update execution status
	execution.Status = "cancelled"
	now := time.Now()
	execution.CompletedAt = &now

	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		return err
	}

	// Log the cancellation
	logEntry := &database.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       "info",
		StepName:    "execution_cancelled",
		Message:     "Execution cancelled by user request",
	}
	s.repo.CreateExecutionLog(ctx, logEntry)

	// Broadcast cancellation
	s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
		Type:        "cancelled",
		ExecutionID: execution.ID,
		Status:      "cancelled",
		Progress:    execution.Progress,
		CurrentStep: execution.CurrentStep,
		Message:     "Execution cancelled by user",
	})

	s.log.WithField("execution_id", executionID).Info("Execution stopped by user request")
	return nil
}

// generateWorkflowFromPrompt generates a workflow definition via OpenRouter.
func (s *WorkflowService) generateWorkflowFromPrompt(ctx context.Context, prompt string) (database.JSONMap, error) {
	if s.aiClient == nil {
		return nil, errors.New("openrouter client not configured")
	}

	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return nil, errors.New("empty AI prompt")
	}

	instruction := fmt.Sprintf(`You are a strict JSON generator. Produce exactly one JSON object with the following structure:
{
  "nodes": [
    {
      "id": "node-1",
      "type": "navigate" | "click" | "type" | "wait" | "screenshot" | "extract" | "workflowCall",
      "position": {"x": <number>, "y": <number>},
      "data": { ... } // include all parameters needed for the step (url, selector, text, waitMs, etc.)
    }
  ],
  "edges": [
    {"id": "edge-1", "source": "node-1", "target": "node-2"}
  ]
}

Rules:
1. Tailor every node and selector to the user's request. Never use placeholders such as "https://example.com" or generic selectors.
2. Provide only the fields needed to execute the step (e.g., url, selector, text, waitMs, timeoutMs, screenshot name). Keep the response concise.
3. Arrange nodes with sensible coordinates (e.g., x increments by ~180 horizontally, y by ~120 vertically for branches).
4. Include necessary wait/ensure steps before interactions to make the automation reliable. Use the "wait" type for waits/ensure conditions.
5. Valid node types are limited to: navigate, click, type, wait, screenshot, extract, workflowCall. Do not invent new types.
6. Wrap the JSON in markers exactly like this: <WORKFLOW_JSON>{...}</WORKFLOW_JSON>.
7. The response MUST start with '<WORKFLOW_JSON>{' and end with '}</WORKFLOW_JSON>'. Output minified JSON on a single line (no spaces or newlines) and keep it under 1200 characters in total.
8. If you cannot produce a valid workflow, respond with <WORKFLOW_JSON>{"error":"reason"}</WORKFLOW_JSON>.

User prompt:
%s`, trimmed)

	start := time.Now()
	response, err := s.aiClient.ExecutePrompt(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("openrouter execution error: %w", err)
	}
	s.log.WithFields(logrus.Fields{
		"model":            s.aiClient.model,
		"duration_ms":      time.Since(start).Milliseconds(),
		"response_preview": truncateForLog(response, 400),
	}).Info("AI workflow generated via OpenRouter")

	cleaned := extractJSONObject(stripJSONCodeFence(response))

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"raw_response": truncateForLog(response, 2000),
			"cleaned":      truncateForLog(cleaned, 2000),
		}).Error("Failed to parse workflow JSON returned by OpenRouter")
		return nil, fmt.Errorf("failed to parse OpenRouter JSON: %w", err)
	}

	if err := detectAIWorkflowError(payload); err != nil {
		return nil, err
	}

	definition, err := normalizeFlowDefinition(payload)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func stripJSONCodeFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		if idx := strings.Index(trimmed, "\n"); idx != -1 {
			trimmed = trimmed[idx+1:]
		}
		if idx := strings.LastIndex(trimmed, "```"); idx != -1 {
			trimmed = trimmed[:idx]
		}
	}
	return strings.TrimSpace(trimmed)
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return strings.TrimSpace(raw)
	}
	return strings.TrimSpace(raw[start : end+1])
}

func normalizeFlowDefinition(payload map[string]interface{}) (database.JSONMap, error) {
	candidate := payload
	if workflow, ok := payload["workflow"].(map[string]interface{}); ok {
		candidate = workflow
	}

	if err := detectAIWorkflowError(candidate); err != nil {
		return nil, err
	}

	rawNodes, ok := candidate["nodes"]
	if !ok {
		if steps, ok := candidate["steps"].([]interface{}); ok {
			candidate["nodes"] = steps
			rawNodes = steps
		}
	}

	nodes, ok := rawNodes.([]interface{})
	if !ok || len(nodes) == 0 {
		return nil, &AIWorkflowError{Reason: "AI workflow generation did not return any steps. Specify real URLs, selectors, and actions, then try again."}
	}

	for i, rawNode := range nodes {
		node, ok := rawNode.(map[string]interface{})
		if !ok {
			return nil, errors.New("AI node payload is not an object")
		}
		if _, ok := node["id"].(string); !ok {
			node["id"] = fmt.Sprintf("node-%d", i+1)
		}
		if _, ok := node["position"].(map[string]interface{}); !ok {
			node["position"] = map[string]interface{}{
				"x": float64(100 + i*200),
				"y": float64(100 + i*120),
			}
		}
		nodes[i] = node
	}
	candidate["nodes"] = nodes

	edgesRaw, hasEdges := candidate["edges"]
	edges, _ := edgesRaw.([]interface{})
	if !hasEdges || edges == nil {
		edges = []interface{}{}
	}

	if len(edges) == 0 && len(nodes) > 1 {
		edges = make([]interface{}, 0, len(nodes)-1)
		for i := 0; i < len(nodes)-1; i++ {
			source := nodes[i].(map[string]interface{})["id"].(string)
			target := nodes[i+1].(map[string]interface{})["id"].(string)
			edges = append(edges, map[string]interface{}{
				"id":     fmt.Sprintf("edge-%d", i+1),
				"source": source,
				"target": target,
			})
		}
	}
	candidate["edges"] = edges

	return database.JSONMap(candidate), nil
}

func detectAIWorkflowError(payload map[string]interface{}) error {
	if reason := extractAIErrorMessage(payload); reason != "" {
		return &AIWorkflowError{Reason: reason}
	}
	return nil
}

func extractAIErrorMessage(value interface{}) string {
	switch typed := value.(type) {
	case map[string]interface{}:
		if msg, ok := typed["error"].(string); ok {
			trimmed := strings.TrimSpace(msg)
			if trimmed != "" {
				return trimmed
			}
		}
		if msg, ok := typed["message"].(string); ok {
			trimmed := strings.TrimSpace(msg)
			if trimmed != "" {
				return trimmed
			}
		}
		if workflow, ok := typed["workflow"].(map[string]interface{}); ok {
			if nested := extractAIErrorMessage(workflow); nested != "" {
				return nested
			}
		}
		for _, nested := range typed {
			if nestedMsg := extractAIErrorMessage(nested); nestedMsg != "" {
				return nestedMsg
			}
		}
	}
	return ""
}

func defaultWorkflowDefinition() database.JSONMap {
	return database.JSONMap{
		"nodes": []interface{}{
			map[string]interface{}{
				"id":   "node-1",
				"type": "navigate",
				"position": map[string]interface{}{
					"x": float64(100),
					"y": float64(100),
				},
				"data": map[string]interface{}{
					"url": "https://example.com",
				},
			},
			map[string]interface{}{
				"id":   "node-2",
				"type": "screenshot",
				"position": map[string]interface{}{
					"x": float64(350),
					"y": float64(220),
				},
				"data": map[string]interface{}{
					"name": "homepage",
				},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{
				"id":     "edge-1",
				"source": "node-1",
				"target": "node-2",
			},
		},
	}
}

// ModifyWorkflow applies an AI-driven modification to an existing workflow.
func (s *WorkflowService) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, modificationPrompt string, overrideFlow map[string]interface{}) (*database.Workflow, error) {
	if s.aiClient == nil {
		return nil, errors.New("openrouter client not configured")
	}
	if strings.TrimSpace(modificationPrompt) == "" {
		return nil, errors.New("modification prompt is required")
	}

	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	baseFlow := map[string]interface{}{}
	if overrideFlow != nil {
		baseFlow = overrideFlow
	} else if workflow.FlowDefinition != nil {
		bytes, err := json.Marshal(workflow.FlowDefinition)
		if err == nil {
			_ = json.Unmarshal(bytes, &baseFlow)
		}
	}

	baseFlowJSON, err := json.MarshalIndent(baseFlow, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal base workflow: %w", err)
	}

	instruction := fmt.Sprintf(`You are a strict JSON generator. Update the existing workflow so it satisfies the user's request.

Rules:
1. Respond with a single JSON object that uses the same schema as the original workflow ("nodes" array + "edges" array).
2. Preserve existing node IDs when the step remains applicable. Modify node types/data/positions only where necessary, and keep data concise (only the parameters required to execute the step).
3. Keep the graph valid: edges must describe a reachable execution path.
4. Fill in realistic selectors, URLs, filenames, waits, etc.—no placeholders. Only use the allowed node types: navigate, click, type, wait, screenshot, extract, workflowCall.
5. Wrap the JSON in markers exactly like this: <WORKFLOW_JSON>{...}</WORKFLOW_JSON>.
6. The response MUST start with '<WORKFLOW_JSON>{' and end with '}</WORKFLOW_JSON>'. Output minified JSON on a single line (no spaces or newlines) and keep it shorter than 1200 characters.
7. If the request cannot be satisfied, respond with <WORKFLOW_JSON>{"error":"reason"}</WORKFLOW_JSON>.

Original workflow JSON:
%s

User requested modifications:
%s`, string(baseFlowJSON), strings.TrimSpace(modificationPrompt))

	start := time.Now()
	response, err := s.aiClient.ExecutePrompt(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("openrouter execution error: %w", err)
	}
	s.log.WithFields(logrus.Fields{
		"model":       s.aiClient.model,
		"duration_ms": time.Since(start).Milliseconds(),
		"workflow_id": workflowID,
	}).Info("AI workflow modification generated via OpenRouter")

	cleaned := extractJSONObject(stripJSONCodeFence(response))

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"raw_response": truncateForLog(response, 2000),
			"cleaned":      truncateForLog(cleaned, 2000),
			"workflow_id":  workflowID,
		}).Error("Failed to parse modified workflow JSON returned by OpenRouter")
		return nil, fmt.Errorf("failed to parse OpenRouter JSON: %w", err)
	}

	definition, err := normalizeFlowDefinition(payload)
	if err != nil {
		return nil, err
	}

	workflow.FlowDefinition = definition
	workflow.Version++
	workflow.UpdatedAt = time.Now()

	if err := s.repo.UpdateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// executeWorkflowAsync executes a workflow asynchronously
func (s *WorkflowService) executeWorkflowAsync(execution *database.Execution, workflow *database.Workflow) {
	ctx := context.Background()
	emitter := events.NewEmitter(s.wsHub, s.log)
	s.log.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"workflow_id":  execution.WorkflowID,
	}).Info("Starting async workflow execution")

	// Update status to running and broadcast
	execution.Status = "running"
	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		s.log.WithError(err).Error("Failed to update execution status to running")
		return
	}

	// Broadcast execution started
	s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
		Type:        "progress",
		ExecutionID: execution.ID,
		Status:      "running",
		Progress:    0,
		CurrentStep: "Initializing",
		Message:     "Workflow execution started",
	})

	if emitter != nil {
		emitter.Emit(events.NewEvent(
			events.EventExecutionStarted,
			execution.ID,
			execution.WorkflowID,
			events.WithStatus("running"),
			events.WithMessage("Workflow execution started"),
			events.WithProgress(0),
		))
	}

	// Use browserless client to execute the workflow
	if err := s.browserless.ExecuteWorkflow(ctx, execution, workflow, emitter); err != nil {
		s.log.WithError(err).Error("Workflow execution failed")

		// Mark execution as failed
		execution.Status = "failed"
		execution.Error = err.Error()
		now := time.Now()
		execution.CompletedAt = &now

		// Log the error
		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "error",
			StepName:    "execution_failed",
			Message:     "Workflow execution failed: " + err.Error(),
		}
		s.repo.CreateExecutionLog(ctx, logEntry)

		// Broadcast failure
		s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
			Type:        "failed",
			ExecutionID: execution.ID,
			Status:      "failed",
			Progress:    execution.Progress,
			CurrentStep: execution.CurrentStep,
			Message:     "Workflow execution failed: " + err.Error(),
		})

		if emitter != nil {
			emitter.Emit(events.NewEvent(
				events.EventExecutionFailed,
				execution.ID,
				execution.WorkflowID,
				events.WithStatus("failed"),
				events.WithMessage(err.Error()),
				events.WithProgress(execution.Progress),
				events.WithPayload(map[string]any{
					"current_step": execution.CurrentStep,
					"error":        err.Error(),
				}),
			))
		}
	} else {
		// Mark execution as completed
		execution.Status = "completed"
		execution.Progress = 100
		execution.CurrentStep = "Completed"
		now := time.Now()
		execution.CompletedAt = &now
		execution.Result = database.JSONMap{
			"success": true,
			"message": "Workflow completed successfully",
		}

		s.log.WithField("execution_id", execution.ID).Info("Workflow execution completed successfully")

		// Broadcast completion
		s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
			Type:        "completed",
			ExecutionID: execution.ID,
			Status:      "completed",
			Progress:    100,
			CurrentStep: "Completed",
			Message:     "Workflow completed successfully",
			Data: map[string]interface{}{
				"success": true,
				"result":  execution.Result,
			},
		})

		if emitter != nil {
			emitter.Emit(events.NewEvent(
				events.EventExecutionCompleted,
				execution.ID,
				execution.WorkflowID,
				events.WithStatus("completed"),
				events.WithProgress(100),
				events.WithMessage("Workflow completed successfully"),
				events.WithPayload(map[string]any{
					"result": execution.Result,
				}),
			))
		}
	}

	// Final update
	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		s.log.WithError(err).Error("Failed to update final execution status")
	}
}
