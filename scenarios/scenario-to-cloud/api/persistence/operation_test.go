package persistence

import (
	"context"
	"encoding/json"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// TestCreateOperationRequestKeyScope [REQ:STC-P0-015] proves P03-A04: the
// same request key with the same plan digest returns the existing operation
// (idempotent admission), and the same key with a different plan digest is
// refused with the typed request_key_conflict without writing a row.
func TestCreateOperationRequestKeyScope(t *testing.T) {
	db := openIdentityTestDB(t, "operations-request-key")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-a", "demo-app", "production", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	first, err := repo.CreateOperation(ctx, &domain.CloudOperation{
		ID:           "op-1",
		DeploymentID: "dep-a",
		RequestKey:   "apply-2026-09-09",
		PlanDigest:   "sha256:plan-a",
		Plan:         json.RawMessage(`{"schema_version":"1","actions":[]}`),
		Fence:        1,
	})
	if err != nil {
		t.Fatalf("admit first: %v", err)
	}
	if first.ID != "op-1" || first.State != domain.OperationAdmitted || first.Fence != 1 {
		t.Fatalf("admitted = %+v", first)
	}

	replay, err := repo.CreateOperation(ctx, &domain.CloudOperation{
		ID:           "op-2",
		DeploymentID: "dep-a",
		RequestKey:   "apply-2026-09-09",
		PlanDigest:   "sha256:plan-a",
	})
	if err != nil {
		t.Fatalf("replay with same digest: %v", err)
	}
	if replay.ID != "op-1" {
		t.Fatalf("replay must return the existing operation, got %s", replay.ID)
	}

	_, err = repo.CreateOperation(ctx, &domain.CloudOperation{
		ID:           "op-3",
		DeploymentID: "dep-a",
		RequestKey:   "apply-2026-09-09",
		PlanDigest:   "sha256:plan-b",
	})
	typed := apierrors.As(err)
	if typed.Code != apierrors.CodeRequestKeyConflict || typed.Status() != 409 {
		t.Fatalf("different digest: code=%q status=%d err=%v", typed.Code, typed.Status(), err)
	}
	if typed.Details["existing_operation_id"] != "op-1" || typed.Details["submitted_plan_digest"] != "sha256:plan-b" {
		t.Fatalf("conflict details = %v", typed.Details)
	}
	if apierrors.ExitCodeFor(typed.Code) != apierrors.ExitRefused {
		t.Fatalf("request_key_conflict must be a refusal exit code")
	}

	if op, err := repo.GetOperation(ctx, "op-3"); err != nil || op != nil {
		t.Fatalf("conflicting operation must not be written: %+v, %v", op, err)
	}
	if op, err := repo.GetOperation(ctx, "op-1"); err != nil || op == nil || op.PlanDigest != "sha256:plan-a" || string(op.Plan) != `{"schema_version":"1","actions":[]}` {
		t.Fatalf("admitted operation = %+v, %v", op, err)
	}

	// Request keys are scoped per deployment: another deployment may reuse it.
	if err := repo.CreateDeployment(ctx, newDeployment("dep-b", "demo-app", "staging", "203.0.113.20", "staging.demo.example")); err != nil {
		t.Fatalf("create second deployment: %v", err)
	}
	other, err := repo.CreateOperation(ctx, &domain.CloudOperation{ID: "op-4", DeploymentID: "dep-b", RequestKey: "apply-2026-09-09", PlanDigest: "sha256:plan-b"})
	if err != nil || other.ID != "op-4" {
		t.Fatalf("same key on another deployment = %+v, %v", other, err)
	}
}

// TestCreateOperationRequiresIdentity proves admission refuses an operation
// without its identity fields instead of writing a half record.
func TestCreateOperationRequiresIdentity(t *testing.T) {
	db := openIdentityTestDB(t, "operations-identity")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	_, err := repo.CreateOperation(ctx, &domain.CloudOperation{ID: "op-1", DeploymentID: "dep-a", RequestKey: "k"})
	if !apierrors.Is(err, apierrors.CodeInvalidRequest) {
		t.Fatalf("missing plan digest: %v", err)
	}
	if _, err := repo.CreateOperation(ctx, nil); err == nil {
		t.Fatalf("nil operation must fail")
	}
}
