package deployment

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/persistence"

	_ "modernc.org/sqlite"
)

func TestOperationFinishedSuccessPersistsDeploymentTerminalStatusAndResult(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:deployment-operation-success-lifecycle?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := persistence.NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}
	now := time.Now().UTC()
	dep := &domain.Deployment{
		ID:         "dep-operation-success",
		Name:       "success lifecycle",
		ScenarioID: "demo",
		Status:     domain.StatusPending,
		Manifest:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := repo.CreateDeployment(ctx, dep); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	orchestrator := NewOrchestrator(OrchestratorConfig{Repo: repo, ProgressHub: NewHub()})
	op := &domain.CloudOperation{ID: "op-success", DeploymentID: dep.ID}
	orchestrator.OperationStarted(ctx, op, &execplan.Plan{Scope: execplan.ScopeRuntime})
	active, err := repo.GetDeployment(ctx, dep.ID)
	if err != nil || active == nil || active.Status != domain.StatusDeploying {
		t.Fatalf("operation start status = %+v, err=%v; want deploying", active, err)
	}

	result := json.RawMessage(`{"ok":true,"plan_digest":"sha256:success","actions":[{"id":"install","status":"completed"}]}`)
	if err := repo.UpdateDeploymentDeployResult(ctx, dep.ID, result); err != nil {
		t.Fatalf("persist success result: %v", err)
	}
	orchestrator.OperationFinished(ctx, op, nil, operations.Succeeded, nil, nil)

	finished, err := repo.GetDeployment(ctx, dep.ID)
	if err != nil || finished == nil {
		t.Fatalf("reload deployment: deployment=%+v err=%v", finished, err)
	}
	if finished.Status != domain.StatusDeployed {
		t.Fatalf("successful operation status = %q, want %q", finished.Status, domain.StatusDeployed)
	}
	if !finished.DeployResult.Valid || string(finished.DeployResult.Data) != string(result) {
		t.Fatalf("successful operation result = %+v, want persisted evidence %s", finished.DeployResult, result)
	}
}
