package persistence

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
)

// [REQ:STC-P0-032] The credential ledger round-trips bindings, acks and
// operations without ever holding a value, resolves descriptors exactly, and
// is a credentials.Store the lifecycle can run on.
func TestCredentialLedgerRoundTrip(t *testing.T) {
	db := openIdentityTestDB(t, "credential-ledger")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("second init: %v", err)
	}
	var store credentials.Store = repo
	for _, id := range []string{"dep-1", "dep-2"} {
		if err := repo.CreateDeployment(ctx, newDeployment(id, "demo-app", "production", "203.0.113."+id[len(id)-1:], id+".example")); err != nil {
			t.Fatalf("create deployment: %v", err)
		}
	}
	descriptor := domain.CredentialDescriptor{LogicalID: "fixture/store", Field: "Pass_Word"}
	created := time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC)
	binding := &domain.CredentialBinding{
		ID: credentials.BindingID("dep-1", descriptor), DeploymentID: "dep-1", Descriptor: descriptor,
		Class: domain.CredentialClassGeneratedDatabasePassword, SourceClass: domain.SecretClassPerInstallGenerated,
		Target:          domain.BundleSecretTarget{Type: "env", Name: "STORE_PASSWORD"},
		Version:         domain.CredentialVersion{Number: 2, ContentRef: "cref_abc", CreatedAt: created},
		PreviousVersion: &domain.CredentialVersion{Number: 1, ContentRef: "cref_old", CreatedAt: created.Add(-time.Hour)},
		ConsumerRefs:    []string{"resource:store", "scenario:app"}, GrantRef: "grant-1", RecoveryKeyRef: "",
		State: domain.CredentialBindingMaterialized, CreatedAt: created, UpdatedAt: created,
	}
	if err := store.UpsertBinding(ctx, binding); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := store.GetBinding(ctx, "dep-1", binding.ID)
	if err != nil || got == nil {
		t.Fatalf("get: %v, %v", got, err)
	}
	if got.Descriptor != descriptor || got.Version.Number != 2 || got.Version.ContentRef != "cref_abc" || got.PreviousVersion == nil || got.PreviousVersion.Number != 1 || len(got.ConsumerRefs) != 2 || got.GrantRef != "grant-1" || got.State != domain.CredentialBindingMaterialized {
		t.Fatalf("round trip = %+v", got)
	}
	if !got.Version.CreatedAt.Equal(created) {
		t.Fatalf("version created_at = %s", got.Version.CreatedAt)
	}
	if other, _ := store.GetBinding(ctx, "dep-2", binding.ID); other != nil {
		t.Fatal("binding visible from another deployment")
	}
	got.Version = domain.CredentialVersion{Number: 3, ContentRef: "cref_new", CreatedAt: created.Add(time.Hour)}
	got.PreviousVersion = nil
	got.State = domain.CredentialBindingRevoked
	if err := store.UpsertBinding(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	again, _ := store.GetBinding(ctx, "dep-1", binding.ID)
	if again.Version.Number != 3 || again.PreviousVersion != nil || again.State != domain.CredentialBindingRevoked {
		t.Fatalf("update not applied: %+v", again)
	}

	// Exact descriptor lookup: case and punctuation are not folded.
	second := *binding
	second.ID = credentials.BindingID("dep-2", descriptor)
	second.DeploymentID = "dep-2"
	if err := store.UpsertBinding(ctx, &second); err != nil {
		t.Fatal(err)
	}
	matches, err := store.FindBindingsByDescriptor(ctx, descriptor)
	if err != nil || len(matches) != 2 {
		t.Fatalf("descriptor matches = %v, %v", matches, err)
	}
	folded, _ := store.FindBindingsByDescriptor(ctx, domain.CredentialDescriptor{LogicalID: "fixture/store", Field: "pass-word"})
	if len(folded) != 0 {
		t.Fatalf("normalised descriptor matched: %v", folded)
	}
	if _, err := credentials.ResolveBinding(ctx, store, "", descriptor); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("cross-deployment ambiguity = %v", err)
	}
	listed, _ := store.ListBindings(ctx, "dep-1")
	if len(listed) != 1 {
		t.Fatalf("list = %v", listed)
	}

	// Acks upsert per (binding, consumer).
	if err := store.RecordAck(ctx, domain.CredentialAck{BindingID: binding.ID, Consumer: "scenario:app", Version: 2, VerifiedAt: created}); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordAck(ctx, domain.CredentialAck{BindingID: binding.ID, Consumer: "scenario:app", Version: 3, VerifiedAt: created.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordAck(ctx, domain.CredentialAck{BindingID: binding.ID, Consumer: "resource:store", Version: 3, VerifiedAt: created.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	acks, _ := store.ListAcks(ctx, binding.ID)
	if len(acks) != 2 || acks[0].Consumer != "resource:store" || acks[1].Version != 3 {
		t.Fatalf("acks = %+v", acks)
	}

	// Rotation documents round-trip whole, including receipts and handoffs.
	handoffAt := created.Add(2 * time.Minute)
	rotation := &domain.CredentialRotation{
		ID: "op-1", DeploymentID: "dep-1", BindingID: binding.ID, Kind: domain.CredentialOperationRotate, FromVersion: 2, ToVersion: 3,
		State:                domain.RotationPendingOperatorInput,
		Consumers:            []domain.CredentialConsumerProgress{{Consumer: "scenario:app", State: domain.ConsumerAcknowledged, Version: 3, UpdatedAt: created}},
		Unreached:            []string{"resource:store"},
		PendingOperatorInput: &domain.OperatorHandoff{Reference: "op-1/revoke-predecessor", Provider: "mailer", Instruction: "revoke in console", ResumeWith: "POST …/resume", RequestedAt: handoffAt},
		Receipts:             []domain.CredentialReceipt{{Step: "new-version", State: "new_version_created", Outcome: "created", At: created, Details: map[string]any{"version": float64(3), "content_ref": "cref_new"}, Limitations: []string{"x"}}},
		CreatedAt:            created, UpdatedAt: created,
	}
	if err := store.SaveRotation(ctx, rotation); err != nil {
		t.Fatalf("save rotation: %v", err)
	}
	loaded, err := store.GetRotation(ctx, "op-1")
	if err != nil || loaded == nil {
		t.Fatalf("get rotation: %v, %v", loaded, err)
	}
	if loaded.State != domain.RotationPendingOperatorInput || loaded.PendingOperatorInput == nil || loaded.PendingOperatorInput.Reference != "op-1/revoke-predecessor" || len(loaded.Receipts) != 1 || loaded.Receipts[0].Details["content_ref"] != "cref_new" || len(loaded.Unreached) != 1 {
		t.Fatalf("rotation round trip = %+v", loaded)
	}
	done := created.Add(time.Hour)
	loaded.State = domain.RotationComplete
	loaded.CompletedAt = &done
	if err := store.SaveRotation(ctx, loaded); err != nil {
		t.Fatal(err)
	}
	later := &domain.CredentialRotation{ID: "op-2", DeploymentID: "dep-1", BindingID: binding.ID, Kind: domain.CredentialOperationRevoke, State: domain.RotationPlanned, CreatedAt: created.Add(time.Minute), UpdatedAt: created.Add(time.Minute)}
	if err := store.SaveRotation(ctx, later); err != nil {
		t.Fatal(err)
	}
	rotations, _ := store.ListRotations(ctx, "dep-1")
	if len(rotations) != 2 || rotations[0].ID != "op-1" || rotations[0].State != domain.RotationComplete || rotations[1].ID != "op-2" {
		t.Fatalf("rotations = %+v", rotations)
	}
	if missing, _ := store.GetRotation(ctx, "op-none"); missing != nil {
		t.Fatal("unknown rotation returned a record")
	}
	if err := store.UpsertBinding(ctx, &domain.CredentialBinding{ID: "x", DeploymentID: "dep-1"}); err == nil {
		t.Fatal("binding without descriptor accepted")
	}
}
