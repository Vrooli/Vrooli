package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless"
	"github.com/vrooli/browser-automation-studio/database"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// WorkflowService handles workflow business logic
type WorkflowService struct {
	repo        database.Repository
	browserless *browserless.Client
	wsHub       *wsHub.Hub
	log         *logrus.Logger
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(repo database.Repository, browserless *browserless.Client, wsHub *wsHub.Hub, log *logrus.Logger) *WorkflowService {
	return &WorkflowService{
		repo:        repo,
		browserless: browserless,
		wsHub:       wsHub,
		log:         log,
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
		// Generate workflow from AI prompt
		workflow.FlowDefinition = s.generateWorkflowFromPrompt(aiPrompt)
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
		// Generate workflow from AI prompt
		workflow.FlowDefinition = s.generateWorkflowFromPrompt(aiPrompt)
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

// generateWorkflowFromPrompt generates a workflow from an AI prompt
func (s *WorkflowService) generateWorkflowFromPrompt(prompt string) database.JSONMap {
	// Mock AI generation - in real implementation, call Ollama or Claude API
	// TODO: Integrate with Ollama for actual AI workflow generation
	s.log.WithField("prompt", prompt).Info("Generating workflow from AI prompt")
	
	return database.JSONMap{
		"nodes": []interface{}{
			map[string]interface{}{
				"id":       "node-1",
				"type":     "navigate",
				"position": map[string]int{"x": 100, "y": 100},
				"data":     map[string]interface{}{"url": "https://example.com"},
			},
			map[string]interface{}{
				"id":       "node-2",
				"type":     "screenshot",
				"position": map[string]int{"x": 100, "y": 200},
				"data":     map[string]interface{}{"name": "homepage"},
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

// executeWorkflowAsync executes a workflow asynchronously
func (s *WorkflowService) executeWorkflowAsync(execution *database.Execution, workflow *database.Workflow) {
	ctx := context.Background()
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

	// Use browserless client to execute the workflow
	if err := s.browserless.ExecuteWorkflow(ctx, execution, workflow); err != nil {
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
	}

	// Final update
	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		s.log.WithError(err).Error("Failed to update final execution status")
	}
}