package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	bastelemetry "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

// Package-level configuration for adhoc workflow cleanup.
// These are shared across execution lifecycle methods (see workflow_service_lifecycle.go and workflow_service_adhoc.go).
var (
	adhocExecutionCleanupInterval = 5 * time.Second
	adhocExecutionRetentionPeriod = 10 * time.Minute
	adhocExecutionCleanupTimeout  = 6 * time.Hour
)

func init() {
	execCfg := config.Load().Execution
	if execCfg.AdhocCleanupInterval > 0 {
		adhocExecutionCleanupInterval = execCfg.AdhocCleanupInterval
	}
	if execCfg.AdhocRetentionPeriod > 0 {
		adhocExecutionRetentionPeriod = execCfg.AdhocRetentionPeriod
	}
	// Keep a generous ceiling by default (6h), but allow operators to raise it via GlobalRequest.
	globalTimeout := config.Load().Timeouts.GlobalRequest
	if globalTimeout > adhocExecutionCleanupTimeout {
		adhocExecutionCleanupTimeout = globalTimeout
	}
}

// IsTerminalExecutionStatus reports whether the supplied status represents a terminal execution state.
// Delegates to the canonical ExecutionStatus type in automation/contracts for consistency.
func IsTerminalExecutionStatus(status string) bool {
	if strings.TrimSpace(status) == "" {
		return false
	}
	return autocontracts.ExecutionStatus(strings.ToLower(status)).IsTerminal()
}

// ExecuteWorkflow executes a workflow.
// This is the primary entry point for executing persisted workflows.
// For ephemeral workflow execution (e.g., testing), see ExecuteAdhocWorkflow in workflow_service_adhoc.go.
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	// Verify workflow exists
	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if workflow.ProjectID != nil {
		if err := s.syncProjectWorkflows(ctx, *workflow.ProjectID); err != nil {
			return nil, err
		}
		workflow, err = s.repo.GetWorkflow(ctx, workflowID)
		if err != nil {
			return nil, err
		}
	}

	if strings.EqualFold(strings.TrimSpace(workflow.WorkflowType), "case") {
		if !workflowHasNodeType(workflow.FlowDefinition, "assert") && len(workflow.ExpectedOutcome) == 0 {
			return nil, ErrWorkflowCaseExpectationMissing
		}
	}

	if err := s.ensureWorkflowChangeMetadata(ctx, workflow); err != nil {
		return nil, err
	}

	execution := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      workflowID,
		WorkflowVersion: workflow.Version,
		Status:          autocontracts.ExecutionStatusPending.String(),
		TriggerType:     "manual",
		Parameters:      database.JSONMap(parameters),
		StartedAt:       time.Now(),
		Progress:        0,
		CurrentStep:     "Initializing workflow",
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, err
	}

	// Start async execution (implementation in workflow_service_lifecycle.go)
	s.startExecutionRunner(execution, workflow)

	return execution, nil
}

func workflowHasNodeType(definition database.JSONMap, nodeType string) bool {
	if definition == nil {
		return false
	}
	rawNodes, ok := definition["nodes"]
	if !ok {
		return false
	}
	nodes, ok := rawNodes.([]any)
	if !ok {
		return false
	}
	want := strings.ToLower(strings.TrimSpace(nodeType))
	for _, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok {
			if coerced, ok := raw.(database.JSONMap); ok {
				node = map[string]any(coerced)
			} else {
				continue
			}
		}
		if strings.ToLower(strings.TrimSpace(fmt.Sprint(node["type"]))) == want {
			return true
		}
	}
	return false
}

// GetExecutionScreenshots reads screenshots from the execution result file.
// Returns proto ExecutionScreenshot types for type safety.
func (s *WorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	// Get execution to find result file path
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}

	if execution.ResultPath == "" {
		// No result file yet - return empty
		return []*basexecution.ExecutionScreenshot{}, nil
	}

	// Read and parse result file
	resultData, err := s.readExecutionResult(execution.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*basexecution.ExecutionScreenshot{}, nil
		}
		return nil, fmt.Errorf("read result file: %w", err)
	}

	// Extract screenshots from artifacts
	screenshots := make([]*basexecution.ExecutionScreenshot, 0)
	for _, artifact := range resultData.Artifacts {
		if artifact.ArtifactType != "screenshot" && artifact.ArtifactType != "screenshot_inline" {
			continue
		}

		// Build proto screenshot
		screenshot := &basexecution.ExecutionScreenshot{
			Screenshot: &bastelemetry.TimelineScreenshot{
				ArtifactId:   artifact.ArtifactID,
				Url:          artifact.StorageURL,
				ThumbnailUrl: artifact.ThumbnailURL,
				ContentType:  artifact.ContentType,
			},
		}

		// Extract dimensions from payload
		if artifact.Payload != nil {
			if w, ok := artifact.Payload["width"].(float64); ok {
				screenshot.Screenshot.Width = int32(w)
			}
			if h, ok := artifact.Payload["height"].(float64); ok {
				screenshot.Screenshot.Height = int32(h)
			}
		}

		if artifact.SizeBytes != nil {
			screenshot.Screenshot.SizeBytes = artifact.SizeBytes
		}

		// Set step metadata
		if artifact.StepIndex != nil {
			screenshot.StepIndex = int32(*artifact.StepIndex)
		}
		screenshot.StepLabel = &artifact.Label

		// Find the step to get node ID
		for _, step := range resultData.Steps {
			if artifact.StepIndex != nil && step.StepIndex == *artifact.StepIndex {
				screenshot.NodeId = step.NodeID
				break
			}
		}

		screenshots = append(screenshots, screenshot)
	}

	return screenshots, nil
}

// readExecutionResult reads and parses the execution result JSON file.
func (s *WorkflowService) readExecutionResult(resultPath string) (*autorecorder.ExecutionResultData, error) {
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, err
	}

	var result autorecorder.ExecutionResultData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse result JSON: %w", err)
	}

	return &result, nil
}

// GetExecution gets an execution by ID
func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return s.repo.GetExecution(ctx, id)
}

// ListExecutions lists executions with optional workflow filtering
func (s *WorkflowService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return s.repo.ListExecutions(ctx, workflowID, limit, offset)
}

// The following methods are implemented in separate focused files to improve maintainability:
//
// Adhoc workflow execution (workflow_service_adhoc.go):
//   - ExecuteAdhocWorkflow
//   - scheduleAdhocWorkflowCleanup
//
// Execution lifecycle management (workflow_service_lifecycle.go):
//   - StopExecution
//   - startExecutionRunner
//   - storeExecutionCancel
//   - cancelExecutionByID
//   - executeWorkflowAsync
//   - recordExecutionMarker
//   - applyCapabilityError
//   - closeEventSink
//
// Export preview and status (workflow_service_export.go):
//   - DescribeExecutionExport
//
// Automation engine integration (workflow_service_automation.go):
//   - executeWithAutomationEngine
//   - unsupportedAutomationNodes
//   - extractString
