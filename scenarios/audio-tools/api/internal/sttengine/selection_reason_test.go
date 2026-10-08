package sttengine

import "testing"

func candidates(entries ...Candidate) []Candidate { return entries }

func streaming(id string, native bool, verdict string) Candidate {
	c := Candidate{Verdict: verdict}
	c.Engine.ID = id
	c.Engine.Provides.NativeStreaming = native
	return c
}

// The regression this file exists for: the selected engine's own fact was used
// to assert a claim about every candidate. That sentence made a GPU engine
// stalling out of the running look like a routine tie-break, and dictation ran
// on a CPU batch engine for days while the CLI reported nothing wrong.
func TestSelectionReasonNeverAssertsAUniversalItDidNotCheck(t *testing.T) {
	policy := SelectionPolicy{RankedPreferences: []string{"accelerated"}}
	list := candidates(
		streaming("whisper-local", false, "selected"),
		streaming("kyutai", true, "candidate"),
	)
	facts := map[string]EngineFacts{
		"whisper-local": {Accelerated: false},
		"kyutai":        {Accelerated: true},
	}
	got := selectionReason(policy, list, 0, facts)
	if got == "accelerated=false for all candidates; tiebreak=manifest_order" {
		t.Fatalf("selectionReason() repeated the false universal claim: %q", got)
	}
	if want := "selected engine is not accelerated while 1 accelerated candidate(s) were eligible; check ranked preference order"; got != want {
		t.Fatalf("selectionReason() = %q, want %q", got, want)
	}
}

func TestSelectionReasonUniversalClaimRequiresAnEmptyField(t *testing.T) {
	policy := SelectionPolicy{RankedPreferences: []string{"accelerated"}}
	list := candidates(
		streaming("whisper-local", false, "selected"),
		streaming("kyutai", true, "rejected"), // not eligible; must not be counted
	)
	facts := map[string]EngineFacts{
		"whisper-local": {Accelerated: false},
		"kyutai":        {Accelerated: true},
	}
	if want, got := "no eligible candidate reported acceleration; tiebreak=manifest_order", selectionReason(policy, list, 0, facts); got != want {
		t.Fatalf("selectionReason() = %q, want %q", got, want)
	}
}

func TestSelectionReasonNamesNativeStreamingInversion(t *testing.T) {
	policy := SelectionPolicy{RankedPreferences: []string{"native_streaming", "accelerated"}}
	list := candidates(
		streaming("whisper-local", false, "selected"),
		streaming("kyutai", true, "candidate"),
	)
	facts := map[string]EngineFacts{"whisper-local": {}, "kyutai": {}}
	want := "selected engine is buffered while 1 native-streaming candidate(s) were eligible; check ranked preference order"
	if got := selectionReason(policy, list, 0, facts); got != want {
		t.Fatalf("selectionReason() = %q, want %q", got, want)
	}
}

// A streaming session must not be seated on a batch engine while a
// native-streaming engine is serviceable. This is the policy that decides it.
func TestResolveFactsPrefersNativeStreamingOverAcceleration(t *testing.T) {
	registry := &Registry{manifest: Manifest{
		SelectionPolicy: SelectionPolicy{
			HardFilters:       []string{"platform_supported", "resource_serviceable"},
			RankedPreferences: []string{"native_streaming", "accelerated"},
		},
		Engines: []Engine{
			{ID: "whisper-local"},
			{ID: "kyutai", Provides: Provides{NativeStreaming: true}},
		},
	}}
	serviceable := EngineFacts{PlatformSupported: true, Installed: true, Running: true, Healthy: true, WorkloadCapable: true}
	accelerated := serviceable
	accelerated.Accelerated = true

	resolution, err := registry.ResolveFacts(map[string]EngineFacts{
		"whisper-local": accelerated, // GPU, but batch
		"kyutai":        serviceable, // no acceleration signal, but streams
	})
	if err != nil {
		t.Fatalf("ResolveFacts() error = %v", err)
	}
	if resolution.Selected != "kyutai" {
		t.Fatalf("Selected = %q, want kyutai: a batch engine cannot stream at all, so acceleration must not outrank it", resolution.Selected)
	}
}
