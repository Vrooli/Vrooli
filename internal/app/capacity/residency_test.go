package capacity

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestResidencyServiceUsesGenerationBoundRepositoryAndVerifiedRelease(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capacity.db")
	service := Service{OpenStore: func(ctx context.Context) (Store, error) {
		return engine.NewSQLiteStore(ctx, engine.Config{DBPath: path})
	}}
	store, err := service.openStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.CreateClaim(ctx, engine.CapacityClaim{OwnerKind: "resource", OwnerID: "whisper", ResourceKind: "vram", AmountBytes: 2 << 30, PreferredBytes: 2 << 30, FloorBytes: 0, Status: engine.StatusGranted, DegradeProfile: &engine.DegradeProfile{Model: engine.ModelIdentity{Name: "ggml-medium.bin", Digest: "6c14d5adee5f86394037b4e4e8b59f1673b6cee10e3cf0b11bbdbee79c156208"}}}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()

	startedAt := time.Now().UTC().Add(-time.Second)
	transition := engine.ResidencyTransition{ID: "whisper-test", Model: engine.ModelIdentity{Name: "ggml-medium.bin", Digest: "6c14d5adee5f86394037b4e4e8b59f1673b6cee10e3cf0b11bbdbee79c156208"}, Desired: engine.ResidencyPlacement{Backend: engine.ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startedAt, PreFootprint: engine.ResidencyFootprint{Bytes: 1 << 30, Source: engine.FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	started, err := service.ResidencyBegin(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, Transition: transition})
	if err != nil {
		t.Fatal(err)
	}
	if started.Revision != 1 || started.Transition.CompletedAt != nil {
		t.Fatalf("begin = %+v", started)
	}
	if _, err := service.ResidencyBegin(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, Transition: transition}); !errors.Is(err, engine.ErrResidencyTransitionOpen) {
		t.Fatalf("duplicate begin = %v", err)
	}

	completedAt := time.Now().UTC()
	transition.CompletedAt = &completedAt
	transition.Observed = engine.ResidencyPlacement{Backend: engine.ResidencyBackendUnloaded, Known: true}
	transition.Footprint = engine.ResidencyFootprint{Bytes: 0, Source: engine.FootprintSourceMeasured, ObservedAt: completedAt.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	finished, err := service.ResidencyComplete(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, CurrentClaimGeneration: claim.Generation, Revision: started.Revision, Transition: transition})
	if err != nil {
		t.Fatal(err)
	}
	if finished.Revision != 2 || finished.Transition.VerifiedReleasedBytes() != 1<<30 {
		t.Fatalf("complete = %+v", finished)
	}
	if _, err := service.ResidencyRelease(ctx, ResidencyReleaseRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, TransitionID: transition.ID, Revision: finished.Revision, Model: transition.Model}); err != nil {
		t.Fatal(err)
	}
}

func TestResidencyObserveClaimTakesFreshSnapshotAndKeepsNanoseconds(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capacity.db")
	zeroFloor := int64(0)
	open := func(ctx context.Context) (Store, error) {
		return engine.NewSQLiteStore(ctx, engine.Config{DBPath: path, MinimumDiskFloorBytes: &zeroFloor})
	}
	service := Service{OpenStore: open, Attributor: engine.NewDockerAttributor()}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	priorObservation := time.Now().UTC().Add(-time.Minute)
	claim, err := store.CreateClaim(ctx, engine.CapacityClaim{
		OwnerKind: engine.OwnerKindResource, OwnerID: "whisper", ResourceKind: engine.ResourceKindVRAM,
		AmountBytes: 1024, PreferredBytes: 1024, FloorBytes: 0, Status: engine.StatusGranted,
		ObservedBytes: 1024, ObservedAt: &priorObservation,
	}, time.Minute)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	sampleAt := time.Now().UTC()
	service.Clock = func() time.Time { return sampleAt.Add(10 * time.Millisecond) }
	service.Source = engine.StaticSource{Inventory: hostinventory.Snapshot{
		ProbeStatuses:   map[string]string{"nvidia_gpu": "ok", "nvidia_gpu_processes": "ok"},
		FieldProvenance: map[string]hostinventory.Provenance{"gpu_processes": {ObservedAt: sampleAt}},
	}}
	observation, err := service.ResidencyObserveClaim(ctx, claim.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Bytes != 0 || !observation.ObservedAt.Equal(sampleAt) || !observation.ObservedAt.After(claim.UpdatedAt) {
		t.Fatalf("fresh exact observation=%+v; prior claim sample was %v", observation, claim.ObservedAt)
	}
}

func TestResidencyObserveClaimLeavesSensingFailureUnknown(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capacity.db")
	zeroFloor := int64(0)
	open := func(ctx context.Context) (Store, error) {
		return engine.NewSQLiteStore(ctx, engine.Config{DBPath: path, MinimumDiskFloorBytes: &zeroFloor})
	}
	service := Service{OpenStore: open, Source: engine.StaticSource{Err: errors.New("GPU inventory unavailable")}}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.CreateClaim(ctx, engine.CapacityClaim{
		OwnerKind: engine.OwnerKindResource, OwnerID: "whisper", ResourceKind: engine.ResourceKindVRAM,
		AmountBytes: 1024, PreferredBytes: 1024, FloorBytes: 0, Status: engine.StatusGranted,
	}, time.Minute)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResidencyObserveClaim(ctx, claim.ClaimID); err == nil {
		t.Fatal("failed host sensing returned a residency observation")
	}
	reopened, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := reopened.GetClaim(ctx, claim.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != engine.StatusGranted || after.Generation != claim.Generation || after.ObservedAt != nil {
		t.Fatalf("failed sensing mutated claim into evidence: %+v", after)
	}
}

func TestResidencyObserveClaimRejectsHeartbeatAfterSnapshotWitness(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capacity.db")
	zeroFloor := int64(0)
	open := func(ctx context.Context) (Store, error) {
		return engine.NewSQLiteStore(ctx, engine.Config{DBPath: path, MinimumDiskFloorBytes: &zeroFloor})
	}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.CreateClaim(ctx, engine.CapacityClaim{
		OwnerKind: engine.OwnerKindResource, OwnerID: "whisper", ResourceKind: engine.ResourceKindVRAM,
		AmountBytes: 1024, PreferredBytes: 1024, FloorBytes: 0, Status: engine.StatusGranted,
	}, time.Minute)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	service := Service{OpenStore: open, Source: heartbeatAfterSnapshotSource{open: open, claimID: claim.ClaimID, generation: claim.Generation}}
	if _, err := service.ResidencyObserveClaim(ctx, claim.ClaimID); err == nil || !strings.Contains(err.Error(), "cohort changed") {
		t.Fatalf("heartbeat after sample was accepted: %v", err)
	}
}

type heartbeatAfterSnapshotSource struct {
	open       func(context.Context) (Store, error)
	claimID    string
	generation int64
}

func (s heartbeatAfterSnapshotSource) Snapshot(ctx context.Context) (hostinventory.Snapshot, error) {
	observedAt := time.Now().UTC()
	store, err := s.open(ctx)
	if err != nil {
		return hostinventory.Snapshot{}, err
	}
	_, heartbeatErr := store.HeartbeatClaim(ctx, s.claimID, s.generation, time.Minute)
	closeErr := store.Close()
	if heartbeatErr != nil {
		return hostinventory.Snapshot{}, heartbeatErr
	}
	if closeErr != nil {
		return hostinventory.Snapshot{}, closeErr
	}
	return hostinventory.Snapshot{
		ProbeStatuses: map[string]string{"nvidia_gpu": "ok", "nvidia_gpu_processes": "ok"},
		FieldProvenance: map[string]hostinventory.Provenance{
			"gpu_processes": {ObservedAt: observedAt},
		},
	}, nil
}

func TestResidencyServiceRejectsStaleGenerationAndUnverifiedRelease(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capacity.db")
	service := Service{OpenStore: func(ctx context.Context) (Store, error) {
		return engine.NewSQLiteStore(ctx, engine.Config{DBPath: path})
	}}
	store, err := service.openStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.CreateClaim(ctx, engine.CapacityClaim{OwnerKind: "resource", OwnerID: "whisper", ResourceKind: "vram", AmountBytes: 2 << 30, PreferredBytes: 2 << 30, Status: engine.StatusGranted, DegradeProfile: &engine.DegradeProfile{Model: engine.ModelIdentity{Name: "medium", Digest: "sha256"}}}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	start := engine.ResidencyTransition{ID: "whisper-unknown", Model: engine.ModelIdentity{Name: "medium", Digest: "sha256"}, Desired: engine.ResidencyPlacement{Backend: engine.ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: time.Now().UTC(), PreFootprint: engine.ResidencyFootprint{Bytes: 1, Source: engine.FootprintSourceMeasured, ObservedAt: time.Now().UTC(), Verified: true}}
	if _, err := service.ResidencyBegin(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation + 1, Transition: start}); !errors.Is(err, engine.ErrStaleGeneration) {
		t.Fatalf("stale begin = %v", err)
	}
	started, err := service.ResidencyBegin(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, Transition: start})
	if err != nil {
		t.Fatal(err)
	}
	done := time.Now().UTC().Add(time.Second)
	start.CompletedAt = &done
	start.Observed = engine.ResidencyPlacement{Known: false, Reason: "measurement unavailable"}
	start.Footprint = engine.ResidencyFootprint{Bytes: 1, Source: engine.FootprintSourceMeasured, ObservedAt: done, Verified: false}
	finished, err := service.ResidencyComplete(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, Revision: started.Revision, Transition: start})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResidencyRelease(ctx, ResidencyReleaseRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, TransitionID: start.ID, Revision: finished.Revision, Model: start.Model}); !errors.Is(err, engine.ErrResidencyReleaseUnverified) {
		t.Fatalf("unknown release = %v", err)
	}
}

func TestResidencyServiceRecoversUnknownStateWithoutUnlockingThenResolves(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capacity.db")
	service := Service{OpenStore: func(ctx context.Context) (Store, error) {
		return engine.NewSQLiteStore(ctx, engine.Config{DBPath: path})
	}}
	store, err := service.openStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.CreateClaim(ctx, engine.CapacityClaim{OwnerKind: engine.OwnerKindResource, OwnerID: "whisper", ResourceKind: engine.ResourceKindVRAM, AmountBytes: 128, PreferredBytes: 128, DegradeProfile: &engine.DegradeProfile{Model: engine.ModelIdentity{Name: "medium", Digest: "sha256:medium"}}}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	startedAt := time.Now().UTC().Add(-time.Second)
	transition := engine.ResidencyTransition{ID: "service-recover", Model: engine.ModelIdentity{Name: "medium", Digest: "sha256:medium"}, Desired: engine.ResidencyPlacement{Backend: engine.ResidencyBackendUnloaded, Known: true}, Reason: "idle_expiry", StartedAt: startedAt, PreFootprint: engine.ResidencyFootprint{Bytes: 128, Source: engine.FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true}}
	started, err := service.ResidencyBegin(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, Transition: transition})
	if err != nil {
		t.Fatal(err)
	}
	unknownAt := time.Now().UTC()
	transition.Observed = engine.ResidencyPlacement{Known: false, Reason: "probe unavailable"}
	transition.Footprint = engine.ResidencyFootprint{Bytes: 128, Source: engine.FootprintSourceMeasured, ObservedAt: unknownAt, Verified: true}
	unknown, err := service.ResidencyRecover(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, CurrentClaimGeneration: claim.Generation, Revision: started.Revision, Transition: transition})
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Transition.Observed.Known || unknown.Revision != started.Revision+1 {
		t.Fatalf("unknown recovery=%+v", unknown)
	}
	resolvedAt := time.Now().UTC()
	transition.CompletedAt = &resolvedAt
	transition.Observed = engine.ResidencyPlacement{Backend: engine.ResidencyBackendCPU, Known: true, Reason: "fresh owner observation"}
	transition.Footprint = engine.ResidencyFootprint{Bytes: 128, Source: engine.FootprintSourceMeasured, ObservedAt: resolvedAt.Add(-time.Millisecond), Verified: true}
	transition.EffectsVerified = true
	resolved, err := service.ResidencyRecover(ctx, ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, CurrentClaimGeneration: claim.Generation, Revision: unknown.Revision, Transition: transition})
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Transition.Observed.Known || resolved.Transition.Observed.Backend != engine.ResidencyBackendCPU {
		t.Fatalf("resolved recovery=%+v", resolved)
	}
	events, err := service.ResidencyHistory(ctx, transition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[1].EventKind != "recovery_unknown" || events[2].EventKind != "recovered" {
		t.Fatalf("recovery events=%+v", events)
	}
}
