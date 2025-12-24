package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// CheckpointState represents the execution state at a specific checkpoint.
// Used for resuming executions from a previous point.
type CheckpointState struct {
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
		checkpoint.WorkflowVersion = int(snapshot.WorkflowVersion)
		if snapshot.Parameters != nil {
			checkpoint.Params = jsonValueMapToAnyMap(snapshot.Parameters.InitialParams)
			checkpoint.Variables = jsonValueMapToAnyMap(snapshot.Parameters.InitialStore)
			checkpoint.Env = jsonValueMapToAnyMap(snapshot.Parameters.Env)
		}
	}

	// Read timeline to find last successful step and accumulated state
	if execution.ResultPath != "" {
		timeline, err := s.readExecutionTimelineForCheckpoint(execution.ResultPath)
		if err != nil {
			// If timeline doesn't exist, return checkpoint with no steps completed
			if os.IsNotExist(err) {
				return checkpoint, nil
			}
			return nil, fmt.Errorf("read timeline: %w", err)
		}

		// Parse timeline entries to find last successful step
		checkpoint.LastStepIndex, checkpoint.TotalSteps = extractCheckpointFromTimeline(timeline, checkpoint.Variables)
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

// extractCheckpointFromTimeline analyzes timeline entries to find the last
// successful step and accumulates extracted data into variables.
func extractCheckpointFromTimeline(timeline *bastimeline.ExecutionTimeline, variables map[string]any) (lastStepIndex, totalSteps int) {
	lastStepIndex = -1
	totalSteps = 0

	if timeline == nil || len(timeline.Entries) == 0 {
		return lastStepIndex, totalSteps
	}

	// Sort entries by step index to process in order
	entries := make([]*bastimeline.TimelineEntry, len(timeline.Entries))
	copy(entries, timeline.Entries)
	sort.Slice(entries, func(i, j int) bool {
		iIdx := int32(0)
		jIdx := int32(0)
		if entries[i].StepIndex != nil {
			iIdx = *entries[i].StepIndex
		}
		if entries[j].StepIndex != nil {
			jIdx = *entries[j].StepIndex
		}
		return iIdx < jIdx
	})

	// Track the highest step index seen (for total count estimate)
	maxStepIndex := int32(-1)

	// Process entries to find last successful step and accumulate extracted data
	for _, entry := range entries {
		if entry == nil {
			continue
		}

		stepIndex := int32(-1)
		if entry.StepIndex != nil {
			stepIndex = *entry.StepIndex
		}

		if stepIndex > maxStepIndex {
			maxStepIndex = stepIndex
		}

		// Check if this step succeeded
		if entry.Context != nil && entry.Context.Success != nil && *entry.Context.Success {
			lastStepIndex = int(stepIndex)

			// Accumulate extracted data from successful steps
			if entry.Aggregates != nil && entry.Aggregates.ExtractedDataPreview != nil {
				extractedData := typeconv.JsonValueToAny(entry.Aggregates.ExtractedDataPreview)
				if dataMap, ok := extractedData.(map[string]any); ok {
					for k, v := range dataMap {
						variables[k] = v
					}
				}
			}
		}
	}

	// Estimate total steps from max index seen
	if maxStepIndex >= 0 {
		totalSteps = int(maxStepIndex) + 1
	}

	return lastStepIndex, totalSteps
}

// getExecutionWorkflowVersion retrieves the workflow version from the execution snapshot.
func (s *WorkflowService) getExecutionWorkflowVersion(ctx context.Context, executionID uuid.UUID) (int, error) {
	snapshot, err := s.readExecutionSnapshot(ctx, executionID)
	if err != nil {
		return 0, err
	}
	return int(snapshot.WorkflowVersion), nil
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
