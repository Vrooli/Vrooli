package capacity

import (
	"testing"
	"time"
)

func TestResidencyTransitionKeepsIntentObservationAndReleaseEvidenceSeparate(t *testing.T) {
	now := time.Now().UTC()
	started := now.Add(-2 * time.Second)
	completed := now.Add(-time.Second)
	transition := ResidencyTransition{
		ID:       "transition-1",
		Model:    ModelIdentity{Name: "model-a", Digest: "sha256:abc"},
		Desired:  ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true, Reason: "idle expiry"},
		Observed: ResidencyPlacement{Backend: "unknown", Known: false, Reason: "stop outcome uncertain"},
		Reason:   "idle expiry", StartedAt: started, CompletedAt: &completed,
		PreFootprint:    ResidencyFootprint{Bytes: 6 << 30, Source: FootprintSourceMeasured, ObservedAt: started.Add(-time.Second), Verified: true},
		Footprint:       ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: completed, Verified: true},
		EffectsVerified: false,
	}
	if transition.verifiedReleasedBytesAt(now) != 0 {
		t.Fatalf("partial transition released %d bytes; want no capacity claim", transition.verifiedReleasedBytesAt(now))
	}
	if transition.Desired.Backend != ResidencyBackendUnloaded || transition.Observed.Known || transition.Observed.Backend != "unknown" {
		t.Fatalf("desired and observed placement were collapsed: %+v", transition)
	}
	if transition.Model.Complete() != true || transition.ID == "" || transition.StartedAt.IsZero() || transition.CompletedAt == nil {
		t.Fatalf("transition identity/time/model evidence missing: %+v", transition)
	}

	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true, Reason: "postcondition verified"}
	transition.EffectsVerified = true
	if got := transition.verifiedReleasedBytesAt(now); got != 6<<30 {
		t.Fatalf("verified released bytes = %d, want measured delta %d", got, int64(6)<<30)
	}
	if got := transition.VerifiedReleasedBytes(); got != 6<<30 {
		t.Fatalf("VerifiedReleasedBytes() = %d, want measured delta %d", got, int64(6)<<30)
	}

	for name, mutate := range map[string]func(*ResidencyTransition){
		"unverified effects":             func(value *ResidencyTransition) { value.EffectsVerified = false },
		"unverified pre footprint":       func(value *ResidencyTransition) { value.PreFootprint.Verified = false },
		"unverified footprint":           func(value *ResidencyTransition) { value.Footprint.Verified = false },
		"manifest pre footprint":         func(value *ResidencyTransition) { value.PreFootprint.Source = FootprintSourceManifestDefault },
		"manifest post footprint":        func(value *ResidencyTransition) { value.Footprint.Source = FootprintSourceManifestDefault },
		"missing pre timestamp":          func(value *ResidencyTransition) { value.PreFootprint.ObservedAt = time.Time{} },
		"missing post timestamp":         func(value *ResidencyTransition) { value.Footprint.ObservedAt = time.Time{} },
		"pre timestamp after transition": func(value *ResidencyTransition) { value.PreFootprint.ObservedAt = value.StartedAt.Add(time.Nanosecond) },
		"post timestamp before transition": func(value *ResidencyTransition) {
			value.Footprint.ObservedAt = value.StartedAt
		},
		"post timestamp after completion": func(value *ResidencyTransition) {
			value.Footprint.ObservedAt = value.CompletedAt.Add(time.Nanosecond)
		},
		"stale pre sample": func(value *ResidencyTransition) {
			value.PreFootprint.ObservedAt = now.Add(-maxResidencyFootprintAge - 5*time.Second)
		},
		"stale post sample": func(value *ResidencyTransition) {
			value.Footprint.ObservedAt = now.Add(-maxResidencyFootprintAge - time.Second)
		},
		"nonzero footprint":              func(value *ResidencyTransition) { value.Footprint.Bytes = 1 },
		"negative pre footprint":         func(value *ResidencyTransition) { value.PreFootprint.Bytes = -1 },
		"negative footprint":             func(value *ResidencyTransition) { value.Footprint.Bytes = -1 },
		"post exceeds pre footprint":     func(value *ResidencyTransition) { value.Footprint.Bytes = 7 << 30 },
		"unknown desired placement":      func(value *ResidencyTransition) { value.Desired.Known = false },
		"invalid desired backend":        func(value *ResidencyTransition) { value.Desired.Backend = "typo" },
		"valid but non-unloaded desired": func(value *ResidencyTransition) { value.Desired.Backend = ResidencyBackendCPU },
		"unknown observation":            func(value *ResidencyTransition) { value.Observed.Known = false },
		"invalid observed backend":       func(value *ResidencyTransition) { value.Observed.Backend = "typo" },
		"valid but non-unloaded observed": func(value *ResidencyTransition) {
			value.Observed.Backend = ResidencyBackendCUDA
		},
		"incomplete model": func(value *ResidencyTransition) { value.Model.Digest = "" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := transition
			mutate(&candidate)
			if got := candidate.verifiedReleasedBytesAt(now); got != 0 {
				t.Fatalf("unproven transition released %d bytes", got)
			}
		})
	}
}

func TestVerifiedUnloadCanCloseZeroVRAMClaimWithoutClaimingFreedBytes(t *testing.T) {
	started := time.Now().UTC().Add(-time.Minute)
	completed := time.Now().UTC().Add(-time.Second)
	transition := ResidencyTransition{
		ID: "cpu-model-unloaded", Model: residencyTestModel,
		Desired:  ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Observed: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true},
		Reason:   "idle_unload", StartedAt: started, CompletedAt: &completed, EffectsVerified: true,
		PreFootprint: ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: started.Add(-time.Second), Verified: true},
		Footprint:    ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: completed.Add(-time.Millisecond), Verified: true},
	}
	if !transition.VerifiedUnload() {
		t.Fatal("fresh owner-verified unload of a zero-VRAM model was rejected")
	}
	if got := transition.VerifiedReleasedBytes(); got != 0 {
		t.Fatalf("zero-VRAM unload claimed %d released bytes", got)
	}
}

func TestResidencyModelIdentityRequiresDigest(t *testing.T) {
	if (ModelIdentity{Name: "model-a"}).Complete() {
		t.Fatal("name-only model identity must remain incomplete")
	}
	if (ModelIdentity{Name: " \t", Digest: "sha256:abc"}).Complete() || (ModelIdentity{Name: "model-a", Digest: " \t"}).Complete() {
		t.Fatal("whitespace-only model name or digest must remain incomplete")
	}
	if !(ModelIdentity{Name: "model-a", Digest: "sha256:abc"}).Complete() {
		t.Fatal("name and digest should identify the artifact")
	}
}
