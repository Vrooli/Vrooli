package workflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
)

type screenshotOwnerRepository struct {
	database.Repository
	execution *database.ExecutionIndex
}

func (r screenshotOwnerRepository) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if r.execution.ID != id {
		return nil, database.ErrNotFound
	}
	return r.execution, nil
}

func TestWorkflowScreenshotCallerUsesCanonicalTimelineOwner(t *testing.T) {
	dir := t.TempDir()
	resultPath := filepath.Join(dir, "result.json")
	require.NoError(t, os.WriteFile(resultPath, []byte(`{"entries":[]}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "timeline.proto.json"), []byte(`{"entries":[{"step_index":4,"node_id":"00000000-0000-0000-0000-000000000004","action":{"metadata":{"label":"Saved step"}},"telemetry":{"screenshot":{"artifact_id":"shot-4","content_type":"image/png"}}}]}`), 0o600))
	executionID := uuid.New()
	service := &WorkflowService{repo: screenshotOwnerRepository{execution: &database.ExecutionIndex{
		ID: executionID, WorkflowID: uuid.New(), Status: database.ExecutionStatusCompleted, ResultPath: resultPath,
	}}}

	got, err := service.GetExecutionScreenshots(context.Background(), executionID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "shot-4", got[0].GetScreenshot().GetArtifactId())
	require.Equal(t, "Saved step", got[0].GetStepLabel())
}

func TestResumeCheckpointCallerUsesCanonicalTimelineLegacyFallback(t *testing.T) {
	dir := t.TempDir()
	resultPath := filepath.Join(dir, "result.json")
	const nodeID = "00000000-0000-0000-0000-000000000005"
	require.NoError(t, os.WriteFile(resultPath, []byte(`{"entries":[{"step_index":0,"node_id":"`+nodeID+`","context":{"success":true}}]}`), 0o600))
	executionID := uuid.New()
	workflowID := uuid.New()
	checkpoint := executionwriter.Checkpoint{
		SchemaVersion: executionwriter.CheckpointVersion,
		ExecutionID:   executionID,
		WorkflowID:    workflowID,
		LastStepIndex: 0,
		NodeID:        nodeID,
		TotalSteps:    1,
		Store:         map[string]any{"saved": true},
	}
	rawCheckpoint, err := json.Marshal(checkpoint)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, executionwriter.CheckpointFileName), rawCheckpoint, 0o600))
	service := &WorkflowService{repo: screenshotOwnerRepository{execution: &database.ExecutionIndex{
		ID: executionID, WorkflowID: workflowID, Status: database.ExecutionStatusFailed, ResultPath: resultPath,
	}}}

	got, err := service.ExtractCheckpointState(context.Background(), executionID)
	require.NoError(t, err)
	require.Equal(t, 0, got.LastStepIndex)
	require.Equal(t, 1, got.TotalSteps)
	require.Equal(t, true, got.Variables["saved"])
}
