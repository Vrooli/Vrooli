package sttengine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
)

func TestResolvePrefersAvailableNativeStreamingEngine(t *testing.T) {
	r := New(Manifest{Engines: []Engine{
		{ID: "buffered", Provides: Provides{NativeStreaming: false}},
		{ID: "native", Provides: Provides{NativeStreaming: true}},
	}, SelectionPolicy: SelectionPolicy{RankedPreferences: []string{"native_streaming"}}})
	resolution, err := r.Resolve(func(string) bool { return true })
	if err != nil || resolution.Selected != "native" {
		t.Fatalf("resolution = %+v, err=%v; want native", resolution, err)
	}
	if !resolution.Candidates[1].Selected || resolution.Candidates[0].Verdict != "candidate" {
		t.Fatalf("candidates = %+v, want selected and retained candidate", resolution.Candidates)
	}
}

func TestResolveRejectsUnavailableAndReturnsTypedNoEngine(t *testing.T) {
	r := New(Manifest{Engines: []Engine{{ID: "only"}}})
	resolution, err := r.Resolve(func(string) bool { return false })
	if !errors.Is(err, ErrNoServiceableEngine) {
		t.Fatalf("err = %v, want ErrNoServiceableEngine", err)
	}
	if resolution.Candidates[0].Verdict != "rejected" || resolution.Candidates[0].Reason == "" {
		t.Fatalf("candidates = %+v, want rejected reason", resolution.Candidates)
	}
}

func TestResolveFactsWorkedExamples(t *testing.T) {
	r := New(Manifest{
		Engines: []Engine{
			{ID: "buffered", Provides: Provides{NativeStreaming: false}},
			{ID: "native", Provides: Provides{NativeStreaming: true}},
		},
		SelectionPolicy: SelectionPolicy{
			HardFilters:       []string{"platform_supported", "resource_serviceable"},
			RankedPreferences: []string{"accelerated"},
			TiebreakRank:      []string{"manifest_order"},
		},
	})
	base := map[string]EngineFacts{
		"buffered": {PlatformSupported: true, Installed: true, Running: true, Healthy: true},
		"native":   {PlatformSupported: true, Installed: true, Running: true, Healthy: true, Accelerated: true},
	}
	resolution, err := r.ResolveFacts(base)
	require.NoError(t, err)
	require.Equal(t, "native", resolution.Selected)

	noGPU := map[string]EngineFacts{
		"buffered": base["buffered"],
		"native":   base["native"],
	}
	noGPU["native"] = EngineFacts{PlatformSupported: true, Installed: true, Running: true, Healthy: true}
	resolution, err = r.ResolveFacts(noGPU)
	require.NoError(t, err)
	require.Equal(t, "buffered", resolution.Selected, "manifest order is the tiebreak when neither candidate is accelerated")

	windows := map[string]EngineFacts{
		"buffered": {PlatformSupported: true, Installed: true, Running: true, Healthy: true, Accelerated: true},
		"native":   {PlatformSupported: false, Installed: true, Running: true, Healthy: true, Accelerated: true},
	}
	resolution, err = r.ResolveFacts(windows)
	require.NoError(t, err)
	require.Equal(t, "buffered", resolution.Selected)
	require.Contains(t, resolution.Candidates[1].Reason, "platform_unsupported")

	macOS := map[string]EngineFacts{
		"buffered": {PlatformSupported: false},
		"native":   {PlatformSupported: false},
	}
	_, err = r.ResolveFacts(macOS)
	require.ErrorIs(t, err, ErrNoServiceableEngine)
}

func TestResolveFactsAppliesDeclaredWorkloadFilter(t *testing.T) {
	r := New(Manifest{
		Engines:         []Engine{{ID: "stream-only"}, {ID: "batch"}},
		SelectionPolicy: SelectionPolicy{HardFilters: []string{"workload_capable"}},
	})
	resolution, err := r.ResolveFacts(map[string]EngineFacts{
		"stream-only": {WorkloadCapable: false},
		"batch":       {WorkloadCapable: true},
	})
	require.NoError(t, err)
	require.Equal(t, "batch", resolution.Selected)
	require.Contains(t, resolution.Candidates[0].Reason, "workload_capable=false")
}

func TestLiveResolverCachesControlPlaneFacts(t *testing.T) {
	calls := 0
	r := NewLiveResolver(func(context.Context, ...string) ([]byte, error) {
		calls++
		return []byte(`{"installed":true,"running":true,"healthy":true,"status_code":"ok","resource":{"observed_mode":"cpu"}}`), nil
	}, time.Minute)
	registry := New(Manifest{Engines: []Engine{{ID: "engine", Kind: KindLocalResource, Resource: "resource"}}})
	_, err := r.Resolve(context.Background(), registry)
	require.NoError(t, err)
	_, err = r.Resolve(context.Background(), registry)
	require.NoError(t, err)
	require.Equal(t, 2, calls, "one status and one accelerator observation per engine")
	r.Invalidate()
	_, err = r.Resolve(context.Background(), registry)
	require.NoError(t, err)
	require.Equal(t, 4, calls)
}

func TestResolverSourceDoesNotBranchOnEngineIdentifiers(t *testing.T) {
	raw, err := os.ReadFile("registry.go")
	require.NoError(t, err)
	source := string(raw)
	for _, id := range []string{"whisper-local", "kyutai"} {
		require.NotContains(t, source, "\""+id+"\"", "resolver must remain manifest-driven")
	}
}

// TestEmbeddedManifestValid asserts the checked-in manifest loads + passes the
// cross-field invariants.
func TestEmbeddedManifestValid(t *testing.T) {
	r := Default()
	require.NotEmpty(t, r.Engines())
	_, _, ok := r.ActiveSpeakerMethod()
	require.True(t, ok, "active speaker method must resolve")
}

// TestSchemaJSONIsValidJSON guards against a malformed schema doc.
func TestSchemaJSONIsValidJSON(t *testing.T) {
	var v any
	require.NoError(t, json.Unmarshal(schemaJSON, &v))
}

// TestKnownStrategiesMatchSttchain keeps the manifest's allowed strategy ids in
// lockstep with the canonical sttchain.StrategyKind constants. This is the
// reason registry.go can stay free of an sttchain import (which would cycle via
// egress) without the two sets silently drifting.
func TestKnownStrategiesMatchSttchain(t *testing.T) {
	canonical := map[string]struct{}{
		string(sttchain.StrategyVADSegment):   {},
		string(sttchain.StrategyOverlapAgree): {},
		string(sttchain.StrategyPassthrough):  {},
		string(sttchain.StrategyBuffered):     {},
	}
	require.Equal(t, canonical, knownStrategies, "manifest strategy id set drifted from sttchain.StrategyKind constants")
}

// TestManifestStrategiesAreKnown asserts every strategy every engine lists is a
// real StrategyKind.
func TestManifestStrategiesAreKnown(t *testing.T) {
	r := Default()
	for _, e := range r.Engines() {
		for _, s := range e.Strategies {
			_, ok := knownStrategies[s]
			require.Truef(t, ok, "engine %q lists unknown strategy %q", e.ID, s)
		}
	}
}

// TestLocalResourceEnginesDeclaredInServiceJSON asserts every local_resource
// engine's resource appears in .vrooli/service.json dependencies.resources, so
// the manifest can never reference a resource the scenario does not declare.
func TestLocalResourceEnginesDeclaredInServiceJSON(t *testing.T) {
	declared := serviceResourceDeps(t)
	for _, e := range Default().Engines() {
		if e.Kind != KindLocalResource {
			continue
		}
		_, ok := declared[e.Resource]
		require.Truef(t, ok, "engine %q resource %q not declared in .vrooli/service.json dependencies.resources", e.ID, e.Resource)
	}
}

// TestEligibleStrategiesUnknownEngine returns nil so the selector falls back to
// provider traits.
func TestEligibleStrategiesUnknownEngine(t *testing.T) {
	require.Nil(t, Default().EligibleStrategies("does-not-exist"))
}

func TestValidateRejectsNativeStreamingNonPassthrough(t *testing.T) {
	m := Manifest{
		Engines: []Engine{{
			ID: "bad", DisplayName: "Bad", Kind: KindLocalResource, Resource: "x",
			Provides:   Provides{NativeStreaming: true},
			Strategies: []string{"vad_segment"},
		}},
		SpeakerIsolation: SpeakerIsolation{Active: "verification", Methods: map[string]SpeakerMethod{"verification": {}}},
	}
	require.Error(t, New(m).Validate())
}

func TestValidateRejectsUnknownSelectionSignal(t *testing.T) {
	m := Manifest{Engines: []Engine{{ID: "e", Kind: KindLocalResource, Resource: "x", Strategies: []string{"passthrough"}}}, SelectionPolicy: SelectionPolicy{HardFilters: []string{"invented"}}, SpeakerIsolation: SpeakerIsolation{Active: "off", Methods: map[string]SpeakerMethod{"off": {}}}}
	require.ErrorContains(t, New(m).Validate(), "unknown signal")
}

func TestValidateRejectsUndeclaredActiveSpeakerMethod(t *testing.T) {
	m := Manifest{
		Engines: []Engine{{
			ID: "e", DisplayName: "E", Kind: KindLocalResource, Resource: "x",
			Strategies: []string{"vad_segment"},
		}},
		SpeakerIsolation: SpeakerIsolation{Active: "ghost", Methods: map[string]SpeakerMethod{"verification": {}}},
	}
	require.Error(t, New(m).Validate())
}

// serviceResourceDeps reads dependencies.resources from the scenario's
// .vrooli/service.json (three dirs up from this package).
func serviceResourceDeps(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "read service.json at %s", path)
	var doc struct {
		Dependencies struct {
			Resources map[string]any `json:"resources"`
		} `json:"dependencies"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	return doc.Dependencies.Resources
}
