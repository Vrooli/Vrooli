package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"scenario-to-cloud/domain"

	_ "modernc.org/sqlite"
)

func TestCloudRecoveryOperationIsDurableAndIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", "file:cloud-recovery-operations-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO deployments (id, name, scenario_id, status, manifest) VALUES ($1, $2, $3, $4, $5)`, "dep-1", "demo", "demo", "deployed", `{}`); err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	created := time.Date(2026, 9, 8, 21, 0, 0, 0, time.UTC)
	first := &domain.RecoveryOperation{
		ID:                "op-1",
		DeploymentID:      "dep-1",
		IdempotencyKey:    "dep-1:rollback:current:repair",
		Action:            "rollback",
		ExpectedBundleSHA: "current",
		RepairBundleSHA:   "repair",
		RepairBundlePath:  "/var/lib/demo/repair.tar.gz",
		DataCompatibility: "compatible",
		CreatedAt:         created,
	}
	stored, err := repo.CreateCloudRecoveryOperation(ctx, first)
	if err != nil {
		t.Fatalf("create operation: %v", err)
	}
	if stored == nil || stored.ID != first.ID || stored.Status != "pending" {
		t.Fatalf("stored operation = %#v", stored)
	}

	duplicate, err := repo.CreateCloudRecoveryOperation(ctx, &domain.RecoveryOperation{
		ID:                "op-duplicate",
		DeploymentID:      "dep-1",
		IdempotencyKey:    first.IdempotencyKey,
		Action:            "rollback",
		ExpectedBundleSHA: "current",
		RepairBundleSHA:   "repair",
		RepairBundlePath:  "/other/path",
		DataCompatibility: "compatible",
	})
	if err != nil {
		t.Fatalf("create idempotent duplicate: %v", err)
	}
	if duplicate == nil || duplicate.ID != first.ID || duplicate.RepairBundlePath != first.RepairBundlePath {
		t.Fatalf("duplicate resolved to %#v", duplicate)
	}

	claimed, err := repo.ClaimCloudRecoveryOperation(ctx, first.ID)
	if err != nil || !claimed {
		t.Fatalf("first claim = %v, %v", claimed, err)
	}
	claimed, err = repo.ClaimCloudRecoveryOperation(ctx, first.ID)
	if err != nil || claimed {
		t.Fatalf("second claim = %v, %v", claimed, err)
	}

	receipt := domain.CloudRecoveryReceipt{
		SchemaVersion:   1,
		DeploymentID:    "dep-1",
		OperationID:     first.ID,
		Action:          "rollback",
		Outcome:         "rolled_back",
		Health:          "healthy",
		BundleSHA256:    "repair",
		ExternalReceipt: "scenario-to-cloud:recovery:op-1",
		ObservedAt:      created.Add(time.Minute).Format(time.RFC3339Nano),
	}
	if err := repo.CompleteCloudRecoveryOperation(ctx, first.ID, receipt); err != nil {
		t.Fatalf("complete operation: %v", err)
	}
	completed, err := repo.GetCloudRecoveryOperation(ctx, first.ID)
	if err != nil {
		t.Fatalf("read completed operation: %v", err)
	}
	if completed == nil || completed.Status != "succeeded" || completed.Receipt == nil || completed.Receipt.ExternalReceipt != receipt.ExternalReceipt {
		t.Fatalf("completed operation = %#v", completed)
	}

	failed, err := repo.CreateCloudRecoveryOperation(ctx, &domain.RecoveryOperation{
		ID:                "op-2",
		DeploymentID:      "dep-1",
		IdempotencyKey:    "dep-1:forward_repair:current:repair-2",
		Action:            "forward_repair",
		RepairBundleSHA:   "repair-2",
		RepairBundlePath:  "/var/lib/demo/repair-2.tar.gz",
		DataCompatibility: "compatible",
	})
	if err != nil {
		t.Fatalf("create failed operation: %v", err)
	}
	if err := repo.FailCloudRecoveryOperation(ctx, failed.ID, "owner health check failed"); err != nil {
		t.Fatalf("fail operation: %v", err)
	}
	failed, err = repo.GetCloudRecoveryOperation(ctx, failed.ID)
	if err != nil {
		t.Fatalf("read failed operation: %v", err)
	}
	if failed == nil || failed.Status != "failed" || failed.ErrorMessage != "owner health check failed" || failed.Receipt != nil {
		t.Fatalf("failed operation = %#v", failed)
	}
}
