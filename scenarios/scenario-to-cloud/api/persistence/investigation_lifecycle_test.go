package persistence

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"scenario-to-cloud/domain"
)

func newInvestigationLifecycleRepo(t *testing.T) (*Repository, context.Context) {
	t.Helper()
	db := openIdentityTestDB(t, "investigation-lifecycle")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-1", "demo", "production", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	return repo, ctx
}

func createLifecycleInvestigation(t *testing.T, repo *Repository, ctx context.Context, status domain.InvestigationStatus) {
	t.Helper()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	if err := repo.CreateInvestigation(ctx, &domain.Investigation{
		ID: "inv-1", DeploymentID: "dep-1", Status: status, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create investigation: %v", err)
	}
}

func TestInvestigationRepositoryEnforcesTransitionsAndPreservesTerminalOutcome(t *testing.T) {
	repo, ctx := newInvestigationLifecycleRepo(t)
	createLifecycleInvestigation(t, repo, ctx, domain.InvestigationStatusPending)

	if err := repo.UpdateInvestigationStatus(ctx, "inv-1", domain.InvestigationStatusRunning); err != nil {
		t.Fatalf("pending -> running: %v", err)
	}
	if err := repo.UpdateInvestigationFindings(ctx, "inv-1", "observed result", json.RawMessage(`{"source":"agent-manager"}`)); err != nil {
		t.Fatalf("running -> completed: %v", err)
	}
	if err := repo.UpdateInvestigationErrorWithDetails(ctx, "inv-1", "late failure", json.RawMessage(`{"source":"late"}`)); err == nil {
		t.Fatal("completed investigation accepted a later failure outcome")
	}

	got, err := repo.GetInvestigation(ctx, "inv-1")
	if err != nil {
		t.Fatalf("read investigation: %v", err)
	}
	if got == nil || got.Status != domain.InvestigationStatusCompleted || got.Findings == nil || *got.Findings != "observed result" || got.ErrorMessage != nil {
		t.Fatalf("terminal result was replaced: %+v", got)
	}
}

func TestInvestigationRepositoryRejectsSkippedAndResurrectedTransitions(t *testing.T) {
	repo, ctx := newInvestigationLifecycleRepo(t)
	createLifecycleInvestigation(t, repo, ctx, domain.InvestigationStatusPending)

	if err := repo.UpdateInvestigationStatus(ctx, "inv-1", domain.InvestigationStatusCompleted); err == nil {
		t.Fatal("pending investigation skipped running")
	}
	if err := repo.UpdateInvestigationError(ctx, "inv-1", "failed before launch"); err != nil {
		t.Fatalf("pending -> failed: %v", err)
	}
	if err := repo.UpdateInvestigationStatus(ctx, "inv-1", domain.InvestigationStatusRunning); err == nil {
		t.Fatal("failed investigation was reopened")
	}

	got, err := repo.GetInvestigation(ctx, "inv-1")
	if err != nil || got == nil || got.Status != domain.InvestigationStatusFailed || got.ErrorMessage == nil || *got.ErrorMessage != "failed before launch" {
		t.Fatalf("persisted failure = %+v, err=%v", got, err)
	}
}
