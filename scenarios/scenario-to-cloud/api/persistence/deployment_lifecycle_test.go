package persistence

import (
	"encoding/json"
	"errors"
	"testing"

	"scenario-to-cloud/domain"
)

func TestCreateDeploymentRejectsNonPendingInitialStatus(t *testing.T) {
	repo, ctx := newInvestigationLifecycleRepo(t)

	for _, status := range []domain.DeploymentStatus{"", "unrecognized"} {
		deployment := newDeployment("dep-invalid-"+string(status), "another-app", "production", "203.0.113.99", "invalid.example")
		deployment.Status = status
		if err := repo.CreateDeployment(ctx, deployment); !errors.Is(err, ErrInvalidInitialDeploymentStatus) {
			t.Errorf("initial status %q error=%v, want invalid initial status", status, err)
		}
		if got, err := repo.GetDeployment(ctx, deployment.ID); err != nil || got != nil {
			t.Errorf("invalid initial status %q was persisted: deployment=%+v err=%v", status, got, err)
		}
	}
}

func TestDeploymentRepositoryRejectsInvalidStatusTransitions(t *testing.T) {
	repo, ctx := newInvestigationLifecycleRepo(t)

	if err := repo.UpdateDeploymentStatus(ctx, "dep-1", domain.StatusDeployed, nil, nil); !errors.Is(err, domain.ErrInvalidDeploymentTransition) {
		t.Fatalf("pending -> deployed error=%v, want invalid transition", err)
	}
	if err := repo.BeginDeploymentRun(ctx, "dep-1", domain.StatusSetupRunning); err != nil {
		t.Fatalf("pending -> setup_running: %v", err)
	}
	if err := repo.UpdateDeploymentDeployResult(ctx, "dep-1", json.RawMessage(`{"ok":false}`)); err != nil {
		t.Fatalf("store deploy result: %v", err)
	}
	if err := repo.UpdateDeploymentStatus(ctx, "dep-1", domain.StatusFailed, nil, nil); err != nil {
		t.Fatalf("setup_running -> failed: %v", err)
	}
	if err := repo.UpdateDeploymentStatus(ctx, "dep-1", domain.StatusDeployed, nil, nil); !errors.Is(err, domain.ErrInvalidDeploymentTransition) {
		t.Fatalf("failed -> deployed error=%v, want invalid transition", err)
	}

	got, err := repo.GetDeployment(ctx, "dep-1")
	if err != nil || got == nil || got.Status != domain.StatusFailed || string(got.DeployResult.Data) != `{"ok":false}` {
		t.Fatalf("failed deployment changed after invalid transition: deployment=%+v err=%v", got, err)
	}
}

func TestDeployResultDoesNotPreemptOperationTerminalStatus(t *testing.T) {
	repo, ctx := newInvestigationLifecycleRepo(t)
	if err := repo.BeginDeploymentRun(ctx, "dep-1", domain.StatusDeploying); err != nil {
		t.Fatalf("pending -> deploying: %v", err)
	}
	if err := repo.UpdateDeploymentDeployResult(ctx, "dep-1", json.RawMessage(`{"ok":true}`)); err != nil {
		t.Fatalf("store deploy evidence: %v", err)
	}
	active, err := repo.GetDeployment(ctx, "dep-1")
	if err != nil || active == nil || active.Status != domain.StatusDeploying {
		t.Fatalf("deploy evidence changed active status: deployment=%+v err=%v", active, err)
	}
	if err := repo.UpdateDeploymentStatus(ctx, "dep-1", domain.StatusDeployed, nil, nil); err != nil {
		t.Fatalf("operation completion -> deployed: %v", err)
	}
	finished, err := repo.GetDeployment(ctx, "dep-1")
	if err != nil || finished == nil || finished.Status != domain.StatusDeployed {
		t.Fatalf("terminal projection = %+v err=%v", finished, err)
	}
}

func TestUpdateDeploymentRoutesStatusResetThroughTransitionOwner(t *testing.T) {
	repo, ctx := newInvestigationLifecycleRepo(t)
	if err := repo.BeginDeploymentRun(ctx, "dep-1", domain.StatusSetupRunning); err != nil {
		t.Fatalf("pending -> setup_running: %v", err)
	}
	deployment, err := repo.GetDeployment(ctx, "dep-1")
	if err != nil || deployment == nil {
		t.Fatalf("read deployment: deployment=%+v err=%v", deployment, err)
	}
	if err := deployment.TransitionStatus(domain.StatusPending); err != nil {
		t.Fatalf("setup_running -> pending: %v", err)
	}
	deployment.Name = "updated name"
	if err := repo.UpdateDeployment(ctx, deployment); err != nil {
		t.Fatalf("persist transitioned deployment: %v", err)
	}
	got, err := repo.GetDeployment(ctx, "dep-1")
	if err != nil || got == nil || got.Status != domain.StatusPending || got.Name != "updated name" {
		t.Fatalf("deployment update = %+v err=%v", got, err)
	}
}
