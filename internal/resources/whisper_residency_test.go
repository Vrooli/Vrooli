package resources

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	capacityengine "github.com/vrooli/vrooli/internal/capacity"
)

func TestFreshWhisperVRAMSampleRequiresExactCurrentZeroVRAMClaim(t *testing.T) {
	started := time.Now().UTC().Add(-time.Minute)
	now := time.Now().UTC()
	observedAt := now.Add(-time.Second)
	claim := capacityengine.CapacityClaim{ClaimID: "claim-whisper", OwnerID: "whisper", OwnerKind: capacityengine.OwnerKindResource, ResourceKind: capacityengine.ResourceKindVRAM, Generation: 4, ObservedBytes: 0, ObservedAt: &observedAt}
	observation := capacityapp.ResidencyObservation{Claim: claim, Bytes: 0, ObservedAt: observedAt}
	got, at, ok := freshWhisperVRAMSample(observation, claim.ClaimID, claim.Generation, started, now)
	if !ok || got != 0 || !at.Equal(observedAt) {
		t.Fatalf("fresh exact zero-VRAM sample = %d/%v/%v", got, at, ok)
	}

	cases := map[string]func(*capacityapp.ResidencyObservation){
		"wrong claim":         func(o *capacityapp.ResidencyObservation) { o.Claim.ClaimID = "other" },
		"stale generation":    func(o *capacityapp.ResidencyObservation) { o.Claim.Generation++ },
		"wrong owner":         func(o *capacityapp.ResidencyObservation) { o.Claim.OwnerID = "other" },
		"wrong resource kind": func(o *capacityapp.ResidencyObservation) { o.Claim.ResourceKind = "ram" },
		"stale observation": func(o *capacityapp.ResidencyObservation) {
			stale := started.Add(-time.Second)
			o.ObservedAt = stale
		},
		"future observation": func(o *capacityapp.ResidencyObservation) { future := now.Add(time.Second); o.ObservedAt = future },
	}
	nonzero := observation
	nonzero.Bytes = 1536 * 1024
	measured, measuredAt, ok := freshWhisperVRAMSample(nonzero, claim.ClaimID, claim.Generation, started, now)
	if !ok || measured != 1536*1024 || whisperCPUTransitionVerified(capacityengine.ResidencyPlacement{Backend: capacityengine.ResidencyBackendCPU, Known: true}, capacityengine.ResidencyFootprint{Bytes: measured, Source: capacityengine.FootprintSourceMeasured, ObservedAt: measuredAt, Verified: true}) {
		t.Fatal("nonzero GPU sample incorrectly verified CPU VRAM release")
	}
	if !whisperCPUTransitionVerified(capacityengine.ResidencyPlacement{Backend: capacityengine.ResidencyBackendCPU, Known: true}, capacityengine.ResidencyFootprint{Bytes: 0, Source: capacityengine.FootprintSourceMeasured, ObservedAt: at, Verified: true}) {
		t.Fatal("fresh zero-VRAM sample failed CPU verification")
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := observation
			mutate(&candidate)
			_, _, ok := freshWhisperVRAMSample(candidate, claim.ClaimID, claim.Generation, started, now)
			if ok {
				t.Fatal("invalid GPU observation was accepted")
			}
		})
	}
	sameSecondStart := time.Date(2026, 11, 11, 20, 0, 0, 100_000_000, time.UTC)
	sameSecondSample := sameSecondStart.Add(200 * time.Millisecond)
	sameSecondNow := sameSecondStart.Add(300 * time.Millisecond)
	sameSecondClaim := claim
	sameSecondClaim.ObservedAt = &sameSecondSample
	sameSecondObservation := capacityapp.ResidencyObservation{Claim: sameSecondClaim, Bytes: 0, ObservedAt: sameSecondSample}
	if _, at, ok := freshWhisperVRAMSample(sameSecondObservation, claim.ClaimID, claim.Generation, sameSecondStart, sameSecondNow); !ok || !at.Equal(sameSecondSample) {
		t.Fatalf("same-second post-transition sample lost precision: at=%v ok=%v", at, ok)
	}
}

func TestBeginWhisperCPUTransitionPersistsAgainstExactResourceClaim(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	zeroFloor := int64(0)
	open := func(ctx context.Context) (capacityapp.Store, error) {
		return capacityengine.NewSQLiteStore(ctx, capacityengine.Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	model := capacityengine.ModelIdentity{Name: "ggml-medium.bin", Digest: "sha256:medium"}
	observedAt := time.Now().UTC().Add(-time.Second)
	gpu := 0
	claim, err := store.CreateClaim(ctx, capacityengine.CapacityClaim{
		OwnerKind: capacityengine.OwnerKindResource, OwnerID: "whisper", ResourceKind: capacityengine.ResourceKindVRAM, GPUIndex: &gpu,
		AmountBytes: 1 << 30, PreferredBytes: 2 << 30, FloorBytes: 0, Status: capacityengine.StatusGranted, ActivityState: capacityengine.ActivityIdle,
		ObservedBytes: 1 << 30, ObservedAt: &observedAt,
		DegradeProfile: &capacityengine.DegradeProfile{Model: model, Steps: []capacityengine.DegradeStep{{Label: "vulkan", AmountBytes: 2 << 30}, {Label: "cpu", AmountBytes: 0}}},
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	service := capacityapp.Service{
		OpenStore:  open,
		Source:     capacityengine.StaticSource{},
		Attributor: residencyTestAttributor{},
	}
	manifest := ResourceManifest{Name: "whisper", Acceleration: &AccelerationSpec{Claim: &capacityengine.ResourceClaimSpec{Profile: &capacityengine.DegradeProfile{Model: model}}}}
	transition, err := beginWhisperCPUTransitionWithService(ctx, &Controller{Root: t.TempDir()}, manifest, service)
	if err != nil {
		t.Fatalf("begin Whisper CPU transition: %v", err)
	}
	if transition.request.ClaimID != claim.ClaimID || transition.request.ClaimGeneration != claim.Generation || transition.started.Transition.Desired.Backend != capacityengine.ResidencyBackendCPU || transition.started.Transition.Model != model {
		t.Fatalf("started transition lost exact claim/model/placement binding: %+v", transition)
	}
	got, err := service.ResidencyGet(ctx, transition.started.Transition.ID)
	if err != nil || got.Revision != 1 || got.Transition.PreFootprint.Bytes != 1<<30 {
		t.Fatalf("durable begin replay = %+v, %v", got, err)
	}
}

type residencyTestAttributor struct{}

func (residencyTestAttributor) Attribute(context.Context, int) capacityengine.Attribution {
	return capacityengine.Attribution{OwnerID: capacityengine.OwnerUnknown}
}
