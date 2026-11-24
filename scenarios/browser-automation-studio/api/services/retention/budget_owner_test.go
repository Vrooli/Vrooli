package retention

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/browser-automation-studio/database"
)

type observedEvidenceRepository struct {
	evidenceRepository
	lookedUp []uuid.UUID
}

func (r *observedEvidenceRepository) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	r.lookedUp = append(r.lookedUp, id)
	return r.evidenceRepository.GetExecution(ctx, id)
}

func TestRecordingBudgetSharesExecutionDirectoryIDAcrossEligibilityAndRemoval(t *testing.T) {
	t.Setenv("BAS_EVIDENCE_RETENTION_BATCH_SIZE", "")
	root := t.TempDir()
	executionID := uuid.New()
	executionDir := filepath.Join(root, executionID.String())
	if err := os.Mkdir(executionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(executionDir, "artifact"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(executionDir, old, old); err != nil {
		t.Fatal(err)
	}
	repo := &observedEvidenceRepository{evidenceRepository: evidenceRepository{
		states: map[uuid.UUID]string{executionID: "completed"},
	}}
	service := &Service{recordingsRoot: root, store: repo}

	result, err := service.EnforceEvidenceBudget(context.Background(), "recordings", coreRetention.Budget{
		Name: "recordings", MaxAge: time.Hour, MaxBytes: 1,
	}, 0)
	if err != nil {
		t.Fatalf("EnforceEvidenceBudget: %v", err)
	}
	if result.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", result.Deleted)
	}
	if len(repo.lookedUp) < 2 {
		t.Fatalf("execution lookups = %v, want eligibility and removal checks", repo.lookedUp)
	}
	for _, got := range repo.lookedUp {
		if got != executionID {
			t.Fatalf("execution lookup ID = %s, want %s", got, executionID)
		}
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != executionID {
		t.Fatalf("deleted execution IDs = %v, want [%s]", repo.deleted, executionID)
	}
	if _, err := os.Stat(executionDir); !os.IsNotExist(err) {
		t.Fatalf("execution directory remains after temp-fixture cleanup: %v", err)
	}
}
