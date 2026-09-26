package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// CheckpointState represents the execution state at a specific checkpoint.
// Used for resuming executions from a previous point.
type CheckpointState struct {
	// AdmissionParameters retains browser, artifact and other execution settings.
	AdmissionParameters *basexecution.ExecutionParameters
	// LastStepIndex is the index of the last successfully completed step.
	// -1 means no steps have completed.
	LastStepIndex int

	// TotalSteps is the total number of steps in the workflow.
	TotalSteps int

	// Variables contains accumulated @store/ values from all completed steps.
	Variables map[string]any

	// Params contains the original @params/ values from the execution.
	Params map[string]any

	// Env contains the original @env/ values from the execution.
	Env map[string]any

	// WorkflowVersion is the workflow version at the time of execution.
	WorkflowVersion int
}

// ExtractCheckpointState reads the execution timeline and snapshot to extract
// the state at the last successful checkpoint.
func (s *WorkflowService) ExtractCheckpointState(ctx context.Context, executionID uuid.UUID) (*CheckpointState, error) {
	if s == nil {
		return nil, errors.New("workflow service is nil")
	}

	// Get execution index from database
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, fmt.Errorf("execution not found: %s", executionID)
		}
		return nil, fmt.Errorf("get execution: %w", err)
	}

	// Initialize checkpoint state
	checkpoint := &CheckpointState{
		LastStepIndex: -1,
		TotalSteps:    0,
		Variables:     make(map[string]any),
		Params:        make(map[string]any),
		Env:           make(map[string]any),
	}

	// Load initial execution parameters from snapshot
	snapshot, err := s.readExecutionSnapshot(ctx, executionID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("read execution snapshot: %w", err)
	}
	if snapshot != nil {
		checkpoint.AdmissionParameters = snapshot.Parameters
		checkpoint.WorkflowVersion = int(snapshot.WorkflowVersion)
		if snapshot.Parameters != nil {
			checkpoint.Params = jsonValueMapToAnyMap(snapshot.Parameters.InitialParams)
			checkpoint.Variables = jsonValueMapToAnyMap(snapshot.Parameters.InitialStore)
			checkpoint.Env = jsonValueMapToAnyMap(snapshot.Parameters.Env)
		}
	}

	// Compare the committed cursor with durable successful outcome evidence.
	if execution.ResultPath != "" {
		timeline, err := s.readExecutionTimelineForCheckpoint(execution.ResultPath)
		if err != nil {
			// If timeline doesn't exist, return checkpoint with no steps completed
			if os.IsNotExist(err) {
				return checkpoint, nil
			}
			return nil, fmt.Errorf("read timeline: %w", err)
		}

		var last *bastimeline.TimelineEntry
		lastSuccessPosition := -1
		for position, entry := range timeline.Entries {
			if entry != nil && entry.GetContext().GetSuccess() {
				last = entry
				lastSuccessPosition = position
			}
		}
		for _, entry := range timeline.Entries[lastSuccessPosition+1:] {
			if entry == nil || entry.GetContext().GetErrorCode() != autocontracts.FailureCodeInstructionOutcomeUncertain {
				continue
			}
			step := entry.GetNodeId()
			if step == "" && entry.StepIndex != nil {
				step = fmt.Sprintf("step %d", entry.GetStepIndex())
			}
			if step == "" {
				step = "an unknown step"
			}
			return nil, fmt.Errorf("%w: browser outcome for %s is uncertain; reconcile it before resuming", ErrExecutionNotResumable, step)
		}
		if last == nil {
			return checkpoint, nil
		}
		committed, err := executionwriter.ReadCheckpoint(execution.ResultPath, execution.ID, execution.WorkflowID)
		if err != nil {
			return nil, fmt.Errorf("%w: committed state unavailable: %v", ErrExecutionNotResumable, err)
		}
		if last.StepIndex == nil || committed.LastStepIndex != int(*last.StepIndex) || committed.NodeID != last.GetNodeId() {
			return nil, fmt.Errorf("%w: committed state disagrees with successful outcome", ErrExecutionNotResumable)
		}
		checkpoint.LastStepIndex = committed.LastStepIndex
		checkpoint.TotalSteps = committed.TotalSteps
		checkpoint.Variables = committed.Store
	}

	return checkpoint, nil
}

// readExecutionTimelineForCheckpoint reads the timeline.proto.json file.
func (s *WorkflowService) readExecutionTimelineForCheckpoint(resultPath string) (*bastimeline.ExecutionTimeline, error) {
	// The resultPath points to result.json, but we want timeline.proto.json
	dir := filepath.Dir(resultPath)
	timelinePath := filepath.Join(dir, "timeline.proto.json")

	data, err := os.ReadFile(timelinePath)
	if err != nil {
		return nil, err
	}

	var timeline bastimeline.ExecutionTimeline
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &timeline); err != nil {
		return nil, fmt.Errorf("parse timeline: %w", err)
	}

	return &timeline, nil
}

// jsonValueMapToAnyMap converts a proto JsonValue map to a Go any map.
func jsonValueMapToAnyMap(protoMap map[string]*commonv1.JsonValue) map[string]any {
	result := make(map[string]any)
	for k, v := range protoMap {
		result[k] = typeconv.JsonValueToAny(v)
	}
	return result
}

// convertParamsToProto converts a Go map to a proto JsonValue map for execution parameters.
func convertParamsToProto(params map[string]any) map[string]*commonv1.JsonValue {
	result := make(map[string]*commonv1.JsonValue)
	for k, v := range params {
		result[k] = typeconv.AnyToJsonValue(v)
	}
	return result
}
