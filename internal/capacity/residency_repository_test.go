package capacity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

var residencyTestModel = ModelIdentity{Name: "medium", Digest: "sha256:medium"}

func residencyTestClaim(owner string, amount int64, model ModelIdentity) CapacityClaim {
	return CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: owner, ResourceKind: ResourceKindVRAM,
		AmountBytes: amount, PreferredBytes: amount,
		DegradeProfile: &DegradeProfile{Model: model},
	}
}

func TestResidencyTransitionPersistsReplaysCASAndGatesRelease(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	zeroFloor := int64(0)
	now := time.Now().UTC().Truncate(time.Microsecond)
	open := func() *SQLiteStore {
		t.Helper()
		store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
		if err != nil {
			t.Fatalf("NewSQLiteStore() error = %v", err)
		}
		return store
	}
	store := open()
	claim, err := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		AmountBytes: 2048, PreferredBytes: 2048, FloorBytes: 0,
		DegradeProfile: &DegradeProfile{Model: residencyTestModel},
	}, time.Hour)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}

	start := now.Add(-20 * time.Second)
	completed := now.Add(-time.Second)
	transition := ResidencyTransition{
		ID: "unload-001", Model: residencyTestModel,
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Reason:  "idle reclaim", StartedAt: start,
		PreFootprint: ResidencyFootprint{Bytes: 2048, Source: FootprintSourceMeasured, ObservedAt: start.Add(-time.Second), Verified: true},
	}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		t.Fatalf("BeginResidencyTransition() error = %v", err)
	}
	if started.Revision != 1 || started.Transition.CompletedAt != nil {
		t.Fatalf("started record = %+v, want revision 1 and incomplete", started)
	}
	openForOwner, err := store.GetUnresolvedResidencyTransitionForOwner(ctx, "whisper")
	if err != nil || openForOwner.Transition.ID != transition.ID || openForOwner.ClaimID != claim.ClaimID || openForOwner.Revision != started.Revision {
		t.Fatalf("owner recovery lookup = %+v, %v", openForOwner, err)
	}
	if _, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, ResidencyTransition{
		ID: "unload-002", Model: transition.Model, Desired: transition.Desired, Reason: "duplicate", StartedAt: start,
	}); !errors.Is(err, ErrResidencyTransitionOpen) {
		t.Fatalf("second open transition error = %v, want ErrResidencyTransitionOpen", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	store = open()
	defer store.Close()
	replayed, err := store.GetResidencyTransition(ctx, transition.ID)
	if err != nil {
		t.Fatalf("GetResidencyTransition() after reopen error = %v", err)
	}
	if replayed.Revision != 1 || replayed.ClaimID != claim.ClaimID || !replayed.Transition.StartedAt.Equal(start) {
		t.Fatalf("replayed start = %+v, want durable revision, claim and time", replayed)
	}

	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true, Reason: "owner verified process exit"}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: completed, Verified: true}
	transition.CompletedAt = &completed
	transition.EffectsVerified = true
	finished, err := store.CompleteResidencyTransition(ctx, claim.ClaimID, claim.Generation, 1, transition)
	if err != nil {
		t.Fatalf("CompleteResidencyTransition() error = %v", err)
	}
	if finished.Revision != 2 || finished.Transition.VerifiedReleasedBytes() != 2048 {
		t.Fatalf("completed record = %+v, released bytes = %d", finished, finished.Transition.VerifiedReleasedBytes())
	}
	if openForOwner, err := store.GetUnresolvedResidencyTransitionForOwner(ctx, "whisper"); err != nil || openForOwner.Transition.ID != transition.ID {
		t.Fatalf("owner lookup before claim release = %+v, %v", openForOwner, err)
	}
	if _, err := store.CompleteResidencyTransition(ctx, claim.ClaimID, claim.Generation, 1, transition); !errors.Is(err, ErrResidencyTransitionConflict) {
		t.Fatalf("stale completion error = %v, want CAS conflict", err)
	}

	released, err := store.ReleaseClaimAfterResidency(ctx, claim.OwnerID, claim.ClaimID, claim.Generation, transition.ID, finished.Revision, transition.Model)
	if err != nil {
		t.Fatalf("ReleaseClaimAfterResidency() error = %v", err)
	}
	if released.Status != StatusReleased || released.Generation != claim.Generation+1 {
		t.Fatalf("released claim = %+v, want released at next generation", released)
	}
	if _, err := store.GetUnresolvedResidencyTransitionForOwner(ctx, "whisper"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("owner lookup after verified release = %v, want ErrNotFound", err)
	}
	if _, err := store.ReleaseClaimAfterResidency(ctx, claim.OwnerID, claim.ClaimID, claim.Generation, transition.ID, finished.Revision, transition.Model); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("replayed release error = %v, want stale generation", err)
	}
}

func TestResidencyReleaseGateRejectsUnknownOrUnverifiedObservation(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "retriever", ResourceKind: ResourceKindVRAM,
		AmountBytes: 512, PreferredBytes: 512,
		DegradeProfile: &DegradeProfile{Model: residencyTestModel},
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Add(-10 * time.Second)
	completed := start.Add(2 * time.Second)
	transition := ResidencyTransition{
		ID: "unload-unknown", Model: residencyTestModel,
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Reason:  "idle reclaim", StartedAt: start,
		PreFootprint: ResidencyFootprint{Bytes: 512, Source: FootprintSourceMeasured, ObservedAt: start.Add(-time.Second), Verified: true},
	}
	if _, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition); err != nil {
		t.Fatal(err)
	}
	transition.Observed = ResidencyPlacement{Known: false, Reason: "observer unavailable"}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: completed, Verified: true}
	transition.CompletedAt = &completed
	transition.EffectsVerified = true
	finished, err := store.CompleteResidencyTransition(ctx, claim.ClaimID, claim.Generation, 1, transition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReleaseClaimAfterResidency(ctx, claim.OwnerID, claim.ClaimID, claim.Generation, transition.ID, finished.Revision, transition.Model); !errors.Is(err, ErrResidencyReleaseUnverified) {
		t.Fatalf("release with unknown observed backend error = %v, want unverified gate", err)
	}
	current, err := store.GetClaim(ctx, claim.ClaimID)
	if err != nil || current.Status != StatusGranted || current.Generation != claim.Generation {
		t.Fatalf("claim changed after rejected release: claim=%+v error=%v", current, err)
	}
}

func TestResidencyReleaseAllowsVerifiedZeroVRAMUnloadWithoutFreedBytes(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	observedAt := now.Add(-time.Second)
	claim, err := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		AmountBytes: 0, PreferredBytes: 1 << 30, FloorBytes: 0, ObservedBytes: 0, ObservedAt: &observedAt,
		DegradeProfile: &DegradeProfile{Model: residencyTestModel, Steps: []DegradeStep{{Label: "unloaded", AmountBytes: 0}}},
	}, time.Hour)
	if err != nil {
		t.Fatalf("create zero-byte resource claim: %v", err)
	}
	startedAt := time.Now().UTC()
	transition := ResidencyTransition{
		ID: "verified-zero-vram-unload", Model: residencyTestModel,
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_unload", StartedAt: startedAt,
		PreFootprint: ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: observedAt, Verified: true},
	}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	postAt := time.Now().UTC()
	time.Sleep(2 * time.Millisecond)
	completedAt := time.Now().UTC()
	transition.CompletedAt = &completedAt
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: postAt, Verified: true}
	transition.EffectsVerified = true
	completed, err := store.CompleteResidencyTransition(ctx, claim.ClaimID, claim.Generation, started.Revision, transition)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Transition.VerifiedReleasedBytes() != 0 || !completed.Transition.VerifiedUnload() {
		t.Fatalf("zero-VRAM unload evidence = %+v", completed.Transition)
	}
	released, err := store.ReleaseClaimAfterResidency(ctx, claim.OwnerID, claim.ClaimID, claim.Generation, transition.ID, completed.Revision, transition.Model)
	if err != nil || released.Status != StatusReleased {
		t.Fatalf("release verified zero-byte claim = %+v, %v", released, err)
	}
}

func TestResidencyBeginRejectsClaimBoundToDifferentModel(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claimModel := ModelIdentity{Name: "ggml-medium.bin", Digest: "sha256:medium"}
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 512, claimModel), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	transition := ResidencyTransition{
		ID: "wrong-model", Model: ModelIdentity{Name: "ggml-small.bin", Digest: "sha256:small"},
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Reason:  "idle_expiry", StartedAt: startedAt,
		PreFootprint: ResidencyFootprint{Bytes: 512, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true},
	}
	if _, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition); !errors.Is(err, ErrResidencyModelMismatch) {
		t.Fatalf("cross-model begin = %v, want ErrResidencyModelMismatch", err)
	}
	if _, err := store.GetResidencyTransition(ctx, transition.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("rejected transition persisted: err=%v", err)
	}
	whitespace := transition
	whitespace.ID = "whitespace-model"
	whitespace.Model = ModelIdentity{Name: " \t", Digest: " \t"}
	if _, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, whitespace); err == nil {
		t.Fatal("whitespace-only model identity was accepted by repository")
	}
	if _, err := store.GetResidencyTransition(ctx, whitespace.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("whitespace model transition persisted: err=%v", err)
	}
	transition.Model = claimModel
	if _, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition); err != nil {
		t.Fatalf("exact claim model begin = %v", err)
	}
}

func TestResidencyReleaseRejectsClaimWhoseBoundModelChanged(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	model := ModelIdentity{Name: "medium", Digest: "sha256:medium"}
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 512, model), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	transition := ResidencyTransition{
		ID: "changed-claim-model", Model: model,
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Reason:  "idle_expiry", StartedAt: startedAt,
		PreFootprint: ResidencyFootprint{Bytes: 512, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true},
	}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	completedAt := time.Now().UTC()
	transition.CompletedAt = &completedAt
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: completedAt.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	finished, err := store.CompleteResidencyTransition(ctx, claim.ClaimID, claim.Generation, started.Revision, transition)
	if err != nil {
		t.Fatal(err)
	}
	changedProfile, err := json.Marshal(&DegradeProfile{Model: ModelIdentity{Name: "small", Digest: "sha256:small"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_claims SET degrade_profile=? WHERE claim_id=?`, string(changedProfile), claim.ClaimID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReleaseClaimAfterResidency(ctx, claim.OwnerID, claim.ClaimID, claim.Generation, transition.ID, finished.Revision, model); !errors.Is(err, ErrResidencyModelMismatch) {
		t.Fatalf("release after claim model changed = %v, want ErrResidencyModelMismatch", err)
	}
	current, err := store.GetClaim(ctx, claim.ClaimID)
	if err != nil || current.Status != StatusGranted || current.Generation != claim.Generation {
		t.Fatalf("claim changed after mismatched release: claim=%+v err=%v", current, err)
	}
}

func TestCurrentCapacitySchemaRecreatesResidencyTableWithoutVersionChange(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	zeroFloor := int64(0)
	open := func() *SQLiteStore {
		t.Helper()
		store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
		if err != nil {
			t.Fatalf("NewSQLiteStore() error = %v", err)
		}
		return store
	}
	store := open()
	var version int
	if err := store.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != SchemaVersion {
		t.Fatalf("schema version = %d, want current %d", version, SchemaVersion)
	}
	if _, err := store.db.ExecContext(ctx, `DROP TABLE capacity_residency_transitions`); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open()
	defer store.Close()
	var table string
	if err := store.db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='capacity_residency_transitions'`).Scan(&table); err != nil {
		t.Fatalf("residency table was not recreated: %v", err)
	}
	var reopenedVersion int
	if err := store.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&reopenedVersion); err != nil {
		t.Fatal(err)
	}
	if reopenedVersion != version {
		t.Fatalf("schema version changed from %d to %d", version, reopenedVersion)
	}
}

func TestResidencyReplayRejectsRowPayloadIdentityCorruption(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		AmountBytes: 128, PreferredBytes: 128,
		DegradeProfile: &DegradeProfile{Model: residencyTestModel},
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, ResidencyTransition{
		ID: "corrupt-replay", Model: residencyTestModel,
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Reason:  "idle reclaim", StartedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	started.Revision++
	corruptPayload, err := json.Marshal(started)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_residency_transitions SET transition_json=? WHERE transition_id=?`, string(corruptPayload), started.Transition.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetResidencyTransition(ctx, started.Transition.ID); err == nil {
		t.Fatal("GetResidencyTransition() accepted row/payload revision mismatch")
	}
}

func TestResidencyReplayAndCapacityGatesRejectRowPayloadTimestampCorruption(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	newClaim := func() CapacityClaim {
		t.Helper()
		claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 128, residencyTestModel), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return claim
	}
	first := newClaim()
	startAt := time.Now().UTC().Add(-time.Second)
	transition := ResidencyTransition{
		ID: "timestamp-corrupt", Model: residencyTestModel,
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Reason:  "idle reclaim", StartedAt: startAt,
		PreFootprint: ResidencyFootprint{Bytes: 128, Source: FootprintSourceMeasured, ObservedAt: startAt.Add(-time.Millisecond), Verified: true},
	}
	started, err := store.BeginResidencyTransition(ctx, first.ClaimID, first.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	completedAt := time.Now().UTC()
	transition.CompletedAt = &completedAt
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: completedAt.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	completed, err := store.CompleteResidencyTransition(ctx, first.ClaimID, first.Generation, started.Revision, transition)
	if err != nil {
		t.Fatal(err)
	}

	corruptStart := startAt.Add(-time.Minute)
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_residency_transitions SET started_at=? WHERE transition_id=?`, formatTime(corruptStart), transition.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetResidencyTransition(ctx, transition.ID); err == nil {
		t.Fatal("GetResidencyTransition() accepted row/payload start time mismatch")
	}
	if _, err := store.GetUnresolvedResidencyTransitionForOwner(ctx, first.OwnerID); err == nil {
		t.Fatal("owner recovery lookup accepted row/payload start time mismatch")
	}
	second := newClaim()
	other := transition
	other.ID = "blocked-by-corrupt-row"
	other.StartedAt = time.Now().UTC()
	other.CompletedAt = nil
	other.Observed = ResidencyPlacement{}
	other.Footprint = ResidencyFootprint{}
	other.EffectsVerified = false
	if _, err := store.BeginResidencyTransition(ctx, second.ClaimID, second.Generation, other); err == nil {
		t.Fatal("resource transition admission ignored a corrupt earlier row")
	}

	// Restore the row start so the test can isolate completed_at validation in
	// the proof-gated release path.
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_residency_transitions SET started_at=? WHERE transition_id=?`, formatTime(startAt), transition.ID); err != nil {
		t.Fatal(err)
	}
	alternateOffset := time.FixedZone("alternate", 60*60)
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_residency_transitions SET started_at=? WHERE transition_id=?`, startAt.In(alternateOffset).Format(time.RFC3339Nano), transition.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetResidencyTransition(ctx, transition.ID); err == nil {
		t.Fatal("GetResidencyTransition() accepted an equivalent but noncanonical SQL timestamp")
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_residency_transitions SET started_at=? WHERE transition_id=?`, formatTime(startAt), transition.ID); err != nil {
		t.Fatal(err)
	}
	corruptCompletion := completedAt.Add(time.Hour)
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_residency_transitions SET completed_at=? WHERE transition_id=?`, formatTime(corruptCompletion), transition.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReleaseClaimAfterResidency(ctx, first.OwnerID, first.ClaimID, first.Generation, transition.ID, completed.Revision, transition.Model); err == nil {
		t.Fatal("ReleaseClaimAfterResidency() accepted row/payload completion time mismatch")
	}
	claim, err := store.GetClaim(ctx, first.ClaimID)
	if err != nil || claim.Status != StatusGranted || claim.Generation != first.Generation {
		t.Fatalf("claim changed after corrupt release proof was rejected: claim=%+v err=%v", claim, err)
	}
}

func TestResidencyHistoryValidatesRowsHiddenByIndexedOrdering(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 256, residencyTestModel), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	firstStart := time.Now().UTC().Add(-10 * time.Second)
	first := ResidencyTransition{
		ID: "history-completed", Model: residencyTestModel,
		Desired: ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true},
		Reason:  "pressure", StartedAt: firstStart,
		PreFootprint: ResidencyFootprint{Bytes: 128, Source: FootprintSourceMeasured, ObservedAt: firstStart.Add(-time.Millisecond), Verified: true},
	}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, first)
	if err != nil {
		t.Fatal(err)
	}
	firstComplete := time.Now().UTC()
	first.CompletedAt = &firstComplete
	first.Observed = ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true}
	first.Footprint = ResidencyFootprint{Bytes: 64, Source: FootprintSourceMeasured, ObservedAt: firstComplete.Add(-time.Millisecond), Verified: true}
	first.EffectsVerified = true
	if _, err := store.CompleteResidencyTransition(ctx, claim.ClaimID, claim.Generation, started.Revision, first); err != nil {
		t.Fatal(err)
	}

	secondStart := time.Now().UTC()
	second := ResidencyTransition{
		ID: "history-open", Model: residencyTestModel,
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Reason:  "idle reclaim", StartedAt: secondStart,
		PreFootprint: ResidencyFootprint{Bytes: 64, Source: FootprintSourceMeasured, ObservedAt: secondStart.Add(-time.Millisecond), Verified: true},
	}
	if _, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, second); err != nil {
		t.Fatal(err)
	}
	// Make SQLite sort the newer open row before the completed row while its
	// replay payload still names the true later start time.
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_residency_transitions SET started_at=? WHERE transition_id=?`, formatTime(firstStart.Add(-time.Minute)), second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetLatestResidencyTransitionForClaim(ctx, claim.ClaimID); err == nil {
		t.Fatal("latest lookup ignored a corrupted newer row hidden by indexed ordering")
	}
	if _, err := store.ResizeClaim(ctx, claim.ClaimID, claim.Generation, 128); err == nil {
		t.Fatal("ResizeClaim() mutated a claim with a hidden corrupted open transition")
	}
	if _, err := store.ReleaseClaim(ctx, claim.ClaimID); err == nil {
		t.Fatal("ReleaseClaim() released a claim with a hidden corrupted open transition")
	}
	current, err := store.GetClaim(ctx, claim.ClaimID)
	if err != nil || current.Status != StatusGranted || current.Generation != claim.Generation {
		t.Fatalf("claim changed after hidden transition was rejected: claim=%+v err=%v", current, err)
	}
}

func TestResidencySerializesSameResourceModelAcrossDifferentClaims(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	newClaim := func() CapacityClaim {
		claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 512, residencyTestModel), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return claim
	}
	first, second := newClaim(), newClaim()
	model := residencyTestModel
	startAt := time.Now().UTC().Add(-time.Second)
	start := ResidencyTransition{ID: "resource-open", Model: model, Desired: ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true}, Reason: "pressure", StartedAt: startAt, PreFootprint: ResidencyFootprint{Bytes: 8, Source: FootprintSourceMeasured, ObservedAt: startAt.Add(-time.Millisecond), Verified: true}}
	if _, err := store.BeginResidencyTransition(ctx, first.ClaimID, first.Generation, start); err != nil {
		t.Fatal(err)
	}
	other := start
	other.ID = "second-claim"
	if _, err := store.BeginResidencyTransition(ctx, second.ClaimID, second.Generation, other); !errors.Is(err, ErrResidencyResourceTransitionOpen) {
		t.Fatalf("cross-claim begin = %v, want resource transition conflict", err)
	}

	completed := time.Now().UTC()
	start.CompletedAt = &completed
	start.Observed = ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true}
	start.Footprint = ResidencyFootprint{Bytes: 4, Source: FootprintSourceMeasured, ObservedAt: completed.Add(-time.Millisecond), Verified: true}
	start.EffectsVerified = true
	if _, err := store.CompleteResidencyTransition(ctx, first.ClaimID, first.Generation, 1, start); err != nil {
		t.Fatal(err)
	}
	if _, err := store.BeginResidencyTransition(ctx, second.ClaimID, second.Generation, other); err != nil {
		t.Fatalf("known completed CPU placement should allow next transition: %v", err)
	}
}

func TestResidencyStartedTransitionBlocksGenericReleaseAndReplacementClaim(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	newClaim := func() CapacityClaim {
		claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return claim
	}
	first := newClaim()
	startedAt := time.Now().UTC().Add(-time.Second)
	transition := ResidencyTransition{ID: "crashed-start", Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startedAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	if _, err := store.BeginResidencyTransition(ctx, first.ClaimID, first.Generation, transition); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReleaseClaim(ctx, first.ClaimID); !errors.Is(err, ErrResidencyTransitionOpen) {
		t.Fatalf("generic release with started transition = %v, want transition gate", err)
	}
	current, err := store.GetClaim(ctx, first.ClaimID)
	if err != nil || current.Status != StatusGranted || current.Generation != first.Generation {
		t.Fatalf("rejected release mutated claim: claim=%+v err=%v", current, err)
	}
	second := newClaim()
	transition.ID = "replacement-claim"
	if _, err := store.BeginResidencyTransition(ctx, second.ClaimID, second.Generation, transition); !errors.Is(err, ErrResidencyResourceTransitionOpen) {
		t.Fatalf("replacement claim ignored crashed transition: %v", err)
	}
}

func TestResidencyStartedTransitionBlocksClaimGenerationMutations(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	transition := ResidencyTransition{ID: "generation-mutation-fence", Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startedAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	if _, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition); err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name string
		run  func() error
	}{
		{"degrade", func() error {
			_, err := store.DegradeClaim(ctx, claim.ClaimID, claim.Generation, "smaller", 50)
			return err
		}},
		{"resize", func() error { _, err := store.ResizeClaim(ctx, claim.ClaimID, claim.Generation, 50); return err }},
		{"upshift", func() error {
			_, err := store.UpshiftClaim(ctx, claim.ClaimID, claim.Generation, "larger", 150)
			return err
		}},
		{"preempt", func() error { _, err := store.PreemptClaim(ctx, claim.ClaimID, claim.Generation, "idle"); return err }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			if err := mutation.run(); !errors.Is(err, ErrResidencyTransitionOpen) {
				t.Fatalf("mutation with started residency transition = %v, want ErrResidencyTransitionOpen", err)
			}
			current, err := store.GetClaim(ctx, claim.ClaimID)
			if err != nil || current.Status != StatusGranted || current.Generation != claim.Generation || current.AmountBytes != 100 {
				t.Fatalf("rejected mutation changed claim: claim=%+v err=%v", current, err)
			}
		})
	}
}

func TestResidencyReleaseCanReconcileFreshEvidenceAfterDelay(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startAt := time.Now().UTC().Add(-10 * time.Minute)
	preAt := startAt.Add(-time.Second)
	transition := ResidencyTransition{ID: "delayed-release", Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: preAt, Verified: true}}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	staleAt := time.Now().UTC().Add(-3 * time.Minute)
	transition.CompletedAt = &staleAt
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: staleAt.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	finished, err := store.CompleteResidencyTransition(ctx, claim.ClaimID, claim.Generation, started.Revision, transition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReleaseClaimAfterResidency(ctx, claim.OwnerID, claim.ClaimID, claim.Generation, transition.ID, finished.Revision, transition.Model); !errors.Is(err, ErrResidencyReleaseUnverified) {
		t.Fatalf("stale release = %v, want fresh-evidence gate", err)
	}
	reconciledAt := time.Now().UTC()
	transition.CompletedAt = &reconciledAt
	transition.Footprint.ObservedAt = reconciledAt.Add(-time.Millisecond)
	reconciled, err := store.ReconcileResidencyTransition(ctx, claim.ClaimID, claim.Generation, finished.Revision, transition)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Revision != finished.Revision+1 {
		t.Fatalf("reconciled revision=%d, old=%d", reconciled.Revision, finished.Revision)
	}
	if _, err := store.ReleaseClaimAfterResidency(ctx, claim.OwnerID, claim.ClaimID, claim.Generation, transition.ID, reconciled.Revision, transition.Model); err != nil {
		t.Fatalf("release after fresh reconciliation: %v", err)
	}
}

func TestResidencyRecoveryAfterRestartPersistsAuditAndFencesModel(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, ModelIdentity{Name: "medium", Digest: "sha256:medium-a"}), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startAt := time.Now().UTC().Add(-10 * time.Second)
	transition := ResidencyTransition{ID: "restart-recovery", Model: ModelIdentity{Name: "medium", Digest: "sha256:medium-a"}, Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startAt.Add(-time.Millisecond), Verified: true}}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// Recovery after reopening the durable store resolves only the exact
	// resource, claim generation, revision, transition and model identity.
	store, err = NewSQLiteStore(ctx, Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	transition.CompletedAt = &now
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true, Reason: "owner probe confirms model absent"}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: now.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	if _, err := store.RecoverResidencyTransition(ctx, "other-resource", claim.ClaimID, claim.Generation, claim.Generation, started.Revision, transition); err == nil {
		t.Fatal("recovery accepted a foreign resource owner")
	}
	if _, err := store.RecoverResidencyTransition(ctx, "whisper", "other-claim", claim.Generation, claim.Generation, started.Revision, transition); err == nil {
		t.Fatal("recovery accepted a foreign claim")
	}
	if _, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation+1, started.Revision, transition); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("stale recovery generation error=%v, want stale generation", err)
	}
	wrongModel := transition
	wrongModel.Model.Digest = "sha256:other-model"
	if _, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation, started.Revision, wrongModel); err == nil {
		t.Fatal("recovery accepted a cross-model observation")
	}
	staleObservation := transition
	staleObservation.Footprint.ObservedAt = now.Add(-maxResidencyFootprintAge - time.Second)
	if _, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation, started.Revision, staleObservation); err == nil {
		t.Fatal("recovery accepted stale physical evidence")
	}
	preTransitionObservation := transition
	preTransitionObservation.Footprint.ObservedAt = startAt.Add(-time.Millisecond)
	if _, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation, started.Revision, preTransitionObservation); err == nil {
		t.Fatal("recovery accepted a sample predating the transition")
	}
	if _, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation, started.Revision+1, transition); !errors.Is(err, ErrResidencyTransitionConflict) {
		t.Fatalf("wrong revision recovery error=%v, want CAS conflict", err)
	}
	recovered, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation, started.Revision, transition)
	if err != nil {
		t.Fatalf("recover after restart: %v", err)
	}
	if recovered.Revision != started.Revision+1 || recovered.Transition.Model != transition.Model || recovered.Transition.VerifiedReleasedBytes() != 100 {
		t.Fatalf("recovered record=%+v", recovered)
	}
	wrongReleaseModel := transition.Model
	wrongReleaseModel.Digest = "sha256:other-model"
	if _, err := store.ReleaseClaimAfterResidency(ctx, "whisper", claim.ClaimID, claim.Generation, transition.ID, recovered.Revision, wrongReleaseModel); err == nil {
		t.Fatal("release accepted a cross-model proof")
	}
	events, err := store.ListResidencyTransitionEvents(ctx, transition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].EventKind != "started" || events[1].EventKind != "recovered" || events[1].Record.Revision != recovered.Revision {
		t.Fatalf("audit history=%+v, want started then recovered snapshots", events)
	}
	if _, err := store.ReleaseClaimAfterResidency(ctx, "whisper", claim.ClaimID, claim.Generation, transition.ID, recovered.Revision, transition.Model); err != nil {
		t.Fatalf("release exact recovered model: %v", err)
	}
	events, err = store.ListResidencyTransitionEvents(ctx, transition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[2].EventKind != "claim_released" {
		t.Fatalf("post-release audit history=%+v, want claim_released event", events)
	}
}

func TestResidencyUnknownRecoveryStaysLockedUntilKnownObservation(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	newClaim := func() CapacityClaim {
		claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return claim
	}
	claim := newClaim()
	startAt := time.Now().UTC().Add(-5 * time.Second)
	transition := ResidencyTransition{ID: "unknown-recovery", Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startAt.Add(-time.Millisecond), Verified: true}}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	unknownAt := time.Now().UTC()
	transition.Observed = ResidencyPlacement{Known: false, Reason: "owner could not verify model state"}
	transition.Footprint = ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: unknownAt, Verified: true}
	unknown, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation, started.Revision, transition)
	if err != nil {
		t.Fatalf("persist unknown recovery observation: %v", err)
	}
	if unknown.Revision != 2 {
		t.Fatalf("unknown revision=%d, want 2", unknown.Revision)
	}
	second := newClaim()
	other := transition
	other.ID = "must-stay-blocked"
	if _, err := store.BeginResidencyTransition(ctx, second.ClaimID, second.Generation, other); !errors.Is(err, ErrResidencyResourceTransitionOpen) {
		t.Fatalf("unknown recovery allowed conflicting claim: %v", err)
	}
	resolvedAt := time.Now().UTC()
	transition.CompletedAt = &resolvedAt
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true, Reason: "fresh owner measurement after restart"}
	transition.Footprint = ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: resolvedAt.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	resolved, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation, unknown.Revision, transition)
	if err != nil {
		t.Fatalf("resolve with fresh known state: %v", err)
	}
	if resolved.Revision != unknown.Revision+1 || resolved.Transition.Observed.Backend != ResidencyBackendCPU {
		t.Fatalf("resolved record=%+v", resolved)
	}
	events, err := store.ListResidencyTransitionEvents(ctx, transition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[1].EventKind != "recovery_unknown" || events[2].EventKind != "recovered" {
		t.Fatalf("audit history=%+v, want start, unknown, recovered", events)
	}
	if _, err := store.BeginResidencyTransition(ctx, second.ClaimID, second.Generation, other); err != nil {
		t.Fatalf("known current state should permit serialized next transition: %v", err)
	}
}

func TestResidencyConcurrentClaimsSerializeSamePhysicalModel(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claims := make([]CapacityClaim, 2)
	for i := range claims {
		claims[i], err = store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	start := make(chan struct{})
	type outcome struct {
		claimID string
		err     error
	}
	results := make(chan outcome, len(claims))
	for i, claim := range claims {
		transition := ResidencyTransition{ID: fmt.Sprintf("contended-%d", i), Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true}, Reason: "pressure", StartedAt: startedAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
		go func(claim CapacityClaim, transition ResidencyTransition) {
			<-start
			_, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
			results <- outcome{claimID: claim.ClaimID, err: err}
		}(claim, transition)
	}
	close(start)
	var succeeded, blocked int
	for range claims {
		result := <-results
		if result.err == nil {
			succeeded++
		} else if errors.Is(result.err, ErrResidencyResourceTransitionOpen) {
			blocked++
		} else {
			t.Fatalf("begin for claim %s failed unexpectedly: %v", result.claimID, result.err)
		}
	}
	if succeeded != 1 || blocked != 1 {
		t.Fatalf("same-model concurrent begins: succeeded=%d blocked=%d; want exactly one each", succeeded, blocked)
	}
}

func TestResidencyCrossClaimLockRejectsPreTransitionObservation(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	newClaim := func() CapacityClaim {
		claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return claim
	}
	first, second := newClaim(), newClaim()
	startedAt := time.Now().UTC().Add(-time.Second)
	transition := ResidencyTransition{ID: "stale-postcondition", Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true}, Reason: "pressure", StartedAt: startedAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	started, err := store.BeginResidencyTransition(ctx, first.ClaimID, first.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	completedAt := time.Now().UTC()
	transition.CompletedAt = &completedAt
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true}
	transition.Footprint = ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	if _, err := store.CompleteResidencyTransition(ctx, first.ClaimID, first.Generation, started.Revision, transition); err != nil {
		t.Fatal(err)
	}
	transition.ID = "second-claim"
	transition.StartedAt = completedAt.Add(time.Millisecond)
	transition.CompletedAt = nil
	transition.Observed = ResidencyPlacement{}
	transition.EffectsVerified = false
	transition.Footprint = ResidencyFootprint{}
	transition.PreFootprint.ObservedAt = transition.StartedAt.Add(-time.Millisecond)
	if _, err := store.BeginResidencyTransition(ctx, second.ClaimID, second.Generation, transition); !errors.Is(err, ErrResidencyResourceTransitionOpen) {
		t.Fatalf("pre-transition sample cleared cross-claim lock: %v", err)
	}
}

func TestResidencyExpirySkipsClaimWithStartedTransition(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	transition := ResidencyTransition{ID: "expiry-blocked", Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startedAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	if _, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition); err != nil {
		t.Fatal(err)
	}
	observedAt := time.Now().UTC()
	expired, err := store.ExpireStaleClaims(ctx, observedAt.Add(time.Hour), AbsenceProofs{claim.ClaimID: {Generation: claim.Generation, ObservedAt: observedAt}})
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 0 {
		t.Fatalf("expiry terminated claim with unresolved residency transition: %+v", expired)
	}
	current, err := store.GetClaim(ctx, claim.ClaimID)
	if err != nil || current.Status != StatusGranted || current.Generation != claim.Generation {
		t.Fatalf("expiry changed fenced claim: claim=%+v err=%v", current, err)
	}
}

func TestResidencyRecoversHistoricalTerminalClaimWithExactFreshUnload(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC().Add(-10 * time.Second)
	transition := ResidencyTransition{ID: "historical-terminal", Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startedAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	// Preserve an exact fixture for a terminal row written by the pre-guard
	// implementation. New ReleaseClaim/expiry paths cannot create this state.
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_claims SET status=?,generation=generation+1 WHERE claim_id=?`, StatusExpired, claim.ClaimID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	transition.CompletedAt = &now
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true, Reason: "fresh process and device inventory confirm absence"}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: now.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	if _, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation, started.Revision, transition); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("terminal recovery without current claim generation = %v, want stale generation", err)
	}
	recovered, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation+1, started.Revision, transition)
	if err != nil {
		t.Fatalf("recover terminal transition with exact terminal generation: %v", err)
	}
	if recovered.Revision != started.Revision+1 || recovered.Transition.VerifiedReleasedBytes() != 100 {
		t.Fatalf("terminal recovery result=%+v", recovered)
	}
	events, err := store.ListResidencyTransitionEvents(ctx, transition.ID)
	if err != nil || len(events) != 2 || events[1].EventKind != "recovered" || events[1].Record.Revision != recovered.Revision {
		t.Fatalf("terminal recovery audit events=%+v err=%v", events, err)
	}
	newClaim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	next := transition
	next.ID = "after-terminal-recovery"
	next.StartedAt = now.Add(time.Millisecond)
	next.CompletedAt = nil
	next.Observed = ResidencyPlacement{}
	next.Footprint = ResidencyFootprint{}
	next.EffectsVerified = false
	next.PreFootprint = ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: next.StartedAt.Add(-time.Millisecond), Verified: true}
	if _, err := store.BeginResidencyTransition(ctx, newClaim.ClaimID, newClaim.Generation, next); err != nil {
		t.Fatalf("fresh exact terminal unload should clear resource lock: %v", err)
	}
}

func TestResidencyTerminalUnknownRecoveryRemainsLocked(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC().Add(-5 * time.Second)
	transition := ResidencyTransition{ID: "historical-unknown-terminal", Model: residencyTestModel, Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startedAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	started, err := store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE capacity_claims SET status=?,generation=generation+1 WHERE claim_id=?`, StatusReleased, claim.ClaimID); err != nil {
		t.Fatal(err)
	}
	observedAt := time.Now().UTC()
	transition.Observed = ResidencyPlacement{Known: false, Reason: "physical state unavailable"}
	transition.Footprint = ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: observedAt, Verified: true}
	unknown, err := store.RecoverResidencyTransition(ctx, "whisper", claim.ClaimID, claim.Generation, claim.Generation+1, started.Revision, transition)
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Transition.Observed.Known || unknown.Revision != started.Revision+1 {
		t.Fatalf("unknown terminal observation=%+v", unknown)
	}
	events, err := store.ListResidencyTransitionEvents(ctx, transition.ID)
	if err != nil || len(events) != 2 || events[1].EventKind != "recovery_unknown" || events[1].Record.Revision != unknown.Revision {
		t.Fatalf("unknown terminal recovery audit events=%+v err=%v", events, err)
	}
	newClaim, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, residencyTestModel), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	next := transition
	next.ID = "must-remain-blocked"
	next.StartedAt = observedAt.Add(time.Second)
	next.PreFootprint = ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: observedAt, Verified: true}
	next.CompletedAt = nil
	next.Observed = ResidencyPlacement{}
	next.Footprint = ResidencyFootprint{}
	next.EffectsVerified = false
	if _, err := store.BeginResidencyTransition(ctx, newClaim.ClaimID, newClaim.Generation, next); !errors.Is(err, ErrResidencyResourceTransitionOpen) {
		t.Fatalf("unknown terminal state allowed replacement transition: %v", err)
	}
}

func TestResidencySerializesDifferentModelsAcrossClaims(t *testing.T) {
	ctx := context.Background()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, ModelIdentity{Name: "medium", Digest: "sha256:model-a"}), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateClaim(ctx, residencyTestClaim("whisper", 100, ModelIdentity{Name: "small", Digest: "sha256:model-b"}), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	start := ResidencyTransition{ID: "model-a-open", Model: ModelIdentity{Name: "medium", Digest: "sha256:model-a"}, Desired: ResidencyPlacement{Backend: ResidencyBackendCPU, Known: true}, Reason: "pressure", StartedAt: startedAt, PreFootprint: ResidencyFootprint{Bytes: 100, Source: FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	if _, err := store.BeginResidencyTransition(ctx, first.ClaimID, first.Generation, start); err != nil {
		t.Fatal(err)
	}
	other := start
	other.ID = "model-b-open"
	other.Model = ModelIdentity{Name: "small", Digest: "sha256:model-b"}
	if _, err := store.BeginResidencyTransition(ctx, second.ClaimID, second.Generation, other); !errors.Is(err, ErrResidencyResourceTransitionOpen) {
		t.Fatalf("distinct model transition bypassed physical-resource lock: %v", err)
	}
}
