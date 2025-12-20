package executionwriter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
)

type noopRepo struct{}

func (noopRepo) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return nil, nil
}

func (noopRepo) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	return nil
}

func TestRecordExecutionArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	artifactPath := filepath.Join(dataDir, "video.webm")
	if err := os.WriteFile(artifactPath, []byte("fake-video"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	writer := NewFileWriter(noopRepo{}, storage.NewMemoryStorage(), nil, dataDir)
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
	}

	err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{
		{
			ArtifactType: "video_meta",
			Label:        "video-1",
			Path:         artifactPath,
			Payload: map[string]any{
				"page_index": 0,
			},
		},
	})
	if err != nil {
		t.Fatalf("record artifacts: %v", err)
	}

	resultPath := writer.resultFilePath(plan.ExecutionID)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}

	var result ExecutionResultData
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(result.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(result.Artifacts))
	}

	artifact := result.Artifacts[0]
	if artifact.ArtifactType != "video_meta" {
		t.Fatalf("expected video_meta artifact, got %s", artifact.ArtifactType)
	}
	if artifact.Payload["path"] != artifactPath {
		t.Fatalf("expected path payload to match")
	}
	if artifact.Payload["base64"] == "" {
		t.Fatalf("expected inline base64 payload")
	}
}
