package executionwriter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

type noopRepo struct{}

func (noopRepo) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return nil, nil
}

func (noopRepo) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	return nil
}

func (noopRepo) UpdateExecutionResultPath(ctx context.Context, id uuid.UUID, resultPath string, updatedAt time.Time) error {
	return nil
}

func TestRecordExecutionArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	artifactPath := filepath.Join(dataDir, "video.webm")
	if err := os.WriteFile(artifactPath, []byte("fake-video"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	memStore := storage.NewMemoryStorage()
	writer := NewFileWriter(noopRepo{}, memStore, nil, dataDir)
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

	var result bastimeline.ExecutionTimeline
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.ExecutionId != plan.ExecutionID.String() {
		t.Fatalf("expected execution id %s, got %s", plan.ExecutionID, result.ExecutionId)
	}
	if memStore.ObjectCount() != 1 {
		t.Fatalf("expected 1 stored artifact, got %d", memStore.ObjectCount())
	}
}
