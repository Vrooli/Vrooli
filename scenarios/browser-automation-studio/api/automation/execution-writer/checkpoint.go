package executionwriter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	corestorage "github.com/vrooli/api-core/storage"
)

const CheckpointFileName = "checkpoint.json"
const CheckpointVersion = 1

// Checkpoint is private recovery state. It must never be projected into public
// step outcomes, timeline previews or optional evidence artifacts.
type Checkpoint struct {
	SchemaVersion int            `json:"schema_version"`
	ExecutionID   uuid.UUID      `json:"execution_id"`
	WorkflowID    uuid.UUID      `json:"workflow_id"`
	LastStepIndex int            `json:"last_step_index"`
	NodeID        string         `json:"node_id"`
	TotalSteps    int            `json:"total_steps"`
	Store         map[string]any `json:"store"`
}

func (c Checkpoint) validate() error {
	if c.SchemaVersion != CheckpointVersion {
		return fmt.Errorf("unsupported checkpoint schema %d", c.SchemaVersion)
	}
	if c.ExecutionID == uuid.Nil || c.WorkflowID == uuid.Nil || c.LastStepIndex < 0 || c.NodeID == "" || c.TotalSteps <= 0 || c.Store == nil {
		return fmt.Errorf("incomplete execution checkpoint")
	}
	return nil
}

func (r *FileWriter) RecordCheckpoint(ctx context.Context, checkpoint Checkpoint) error {
	if err := checkpoint.validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("encode recovery state: %w", err)
	}
	resultPath, err := r.resultFilePath(ctx, checkpoint.ExecutionID)
	if err != nil {
		return err
	}
	if err := corestorage.WriteFileAtomic(filepath.Join(filepath.Dir(resultPath), CheckpointFileName), raw, corestorage.SecretFilePerm); err != nil {
		return fmt.Errorf("commit recovery state: %w", err)
	}
	r.root.RecordWrite(ctx)
	return nil
}

// ReadCheckpoint uses the execution index's authoritative result location and
// rejects evidence from another execution or workflow before exposing its store.
func ReadCheckpoint(resultPath string, executionID, workflowID uuid.UUID) (*Checkpoint, error) {
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(resultPath), CheckpointFileName))
	if err != nil {
		return nil, err
	}
	var checkpoint Checkpoint
	if err := json.Unmarshal(raw, &checkpoint); err != nil {
		return nil, fmt.Errorf("decode recovery state: %w", err)
	}
	if err := checkpoint.validate(); err != nil {
		return nil, err
	}
	if checkpoint.ExecutionID != executionID || checkpoint.WorkflowID != workflowID {
		return nil, fmt.Errorf("checkpoint execution identity does not match")
	}
	return &checkpoint, nil
}
