package persistence

import (
	"context"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

func recoveryPointFixture(id, deploymentID string, at time.Time) *domain.RecoveryPoint {
	return &domain.RecoveryPoint{
		ID: id, DeploymentID: deploymentID, OperationID: "op-" + id, BindingIDs: []string{"records-db", "uploads"},
		Bindings:      []domain.DataBinding{{ID: "records-db", Kind: "sql", Provider: "postgres", Locator: "fixture_records"}, {ID: "uploads", Kind: "files", Provider: "object_store", Locator: "/srv/uploads"}},
		SchemaVersion: "v1", ConfigurationDigest: "cfg", ReleaseDigest: "sha256:r1", CredentialVersionRefs: []string{"fixture/store:password@v1"},
		ConsistencyToken: "tok", CapturedAt: at, Provider: "data-backup-manager", ProviderRef: "records-db=data-backup-manager:1",
		Checksums: map[string]domain.BindingChecksum{"records-db": {Count: 8, Checksum: "abc"}, "uploads": {Count: 3, Checksum: "def", Comparable: true}},
		Encrypted: true, RecoveryKeyRef: "fixture/recovery:key", RetentionPolicy: "keep-7", MigrationPosture: "production_evolution",
		Location: "/tmp/rp/" + id, ManifestDigest: "digest-" + id,
	}
}

// TestRecoveryPointsAndRestoreReceiptsRoundTrip [REQ:STC-P0-029] proves the
// routed-storage tables hold the full typed record, list newest first,
// replay an identical capture, refuse a conflicting one and never delete a
// protected point.
func TestRecoveryPointsAndRestoreReceiptsRoundTrip(t *testing.T) {
	db := openIdentityTestDB(t, "recovery-points")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-a", "demo-app", "production", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	first, err := repo.CreateRecoveryPoint(ctx, recoveryPointFixture("rp-1", "dep-a", now))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := repo.CreateRecoveryPoint(ctx, recoveryPointFixture("rp-2", "dep-a", now.Add(time.Hour))); err != nil {
		t.Fatalf("create second: %v", err)
	}
	replay, err := repo.CreateRecoveryPoint(ctx, recoveryPointFixture("rp-1", "dep-a", now))
	if err != nil || replay.ManifestDigest != first.ManifestDigest {
		t.Fatalf("identical capture must replay: %v %+v", err, replay)
	}
	conflict := recoveryPointFixture("rp-1", "dep-a", now)
	conflict.ManifestDigest = "other"
	if _, err := repo.CreateRecoveryPoint(ctx, conflict); !apierrors.Is(err, apierrors.CodeRecoveryPointConflict) {
		t.Fatalf("different capture under the same id must conflict: %v", err)
	}
	plain := recoveryPointFixture("rp-plain", "dep-a", now)
	plain.Encrypted = false
	if _, err := repo.CreateRecoveryPoint(ctx, plain); !apierrors.Is(err, apierrors.CodeInvalidRequest) {
		t.Fatalf("unencrypted point must be refused: %v", err)
	}

	got, err := repo.GetRecoveryPoint(ctx, "rp-1")
	if err != nil || got == nil {
		t.Fatalf("get: %v %+v", err, got)
	}
	if got.Checksums["uploads"].Count != 3 || !got.Checksums["uploads"].Comparable || got.Bindings[0].Locator != "fixture_records" || got.CredentialVersionRefs[0] != "fixture/store:password@v1" || got.RecoveryKeyRef != "fixture/recovery:key" || got.MigrationPosture != "production_evolution" {
		t.Fatalf("record lost fields: %+v", got)
	}
	list, err := repo.ListRecoveryPoints(ctx, "dep-a")
	if err != nil || len(list) != 2 || list[0].ID != "rp-2" {
		t.Fatalf("list = %+v err=%v", list, err)
	}
	if missing, err := repo.GetRecoveryPoint(ctx, "nope"); err != nil || missing != nil {
		t.Fatalf("absent point must be nil: %v %+v", err, missing)
	}

	if _, err := repo.SetRecoveryPointProtection(ctx, "rp-1", true, []string{"release:active:sha256:r1"}); err != nil {
		t.Fatalf("protect: %v", err)
	}
	if err := repo.DeleteRecoveryPoint(ctx, "rp-1"); !apierrors.Is(err, apierrors.CodeRecoveryPointProtected) {
		t.Fatalf("protected delete must be refused: %v", err)
	}
	if kept, _ := repo.GetRecoveryPoint(ctx, "rp-1"); kept == nil || !kept.Protected || kept.ProtectedBy[0] != "release:active:sha256:r1" {
		t.Fatalf("protected point must remain: %+v", kept)
	}
	if err := repo.DeleteRecoveryPoint(ctx, "rp-2"); err != nil {
		t.Fatalf("unprotected delete: %v", err)
	}
	if err := repo.DeleteRecoveryPoint(ctx, "rp-2"); !apierrors.Is(err, apierrors.CodeRecoveryPointNotFound) {
		t.Fatalf("second delete must be not found: %v", err)
	}

	receipt := &domain.RestoreReceipt{ID: "rr-1", RecoveryPointID: "rp-1", DeploymentID: "dep-a", TargetRef: "host-b", StartedAt: now, CompletedAt: now.Add(3 * time.Second), MeasuredRTO: 3 * time.Second, RecoveryPointAge: time.Minute, Outcome: domain.RestoreOutcomeSucceeded, WithinBudgets: true, InvariantResults: []domain.InvariantResult{{Binding: "uploads", Check: "checksum", Expected: "def", Observed: "def", Passed: true}}}
	if _, err := repo.CreateRestoreReceipt(ctx, receipt); err != nil {
		t.Fatalf("create receipt: %v", err)
	}
	if _, err := repo.CreateRestoreReceipt(ctx, &domain.RestoreReceipt{ID: "rr-2", RecoveryPointID: "rp-1", DeploymentID: "dep-a", StartedAt: now.Add(time.Hour), CompletedAt: now.Add(time.Hour), Outcome: domain.RestoreOutcomeRefused, ErrorCode: apierrors.CodeRestoreTargetNotClean}); err != nil {
		t.Fatalf("create refused receipt: %v", err)
	}
	receipts, err := repo.ListRestoreReceipts(ctx, "rp-1")
	if err != nil || len(receipts) != 2 || receipts[0].ID != "rr-2" || receipts[1].MeasuredRTO != 3*time.Second || !receipts[1].WithinBudgets || receipts[1].InvariantResults[0].Check != "checksum" {
		t.Fatalf("receipts = %+v err=%v", receipts, err)
	}
	if _, err := repo.CreateRestoreReceipt(ctx, &domain.RestoreReceipt{ID: "rr-3"}); !apierrors.Is(err, apierrors.CodeInvalidRequest) {
		t.Fatalf("receipt without identity must be refused: %v", err)
	}
}
