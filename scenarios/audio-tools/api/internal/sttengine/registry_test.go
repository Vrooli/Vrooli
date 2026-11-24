package sttengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
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
	require.Equal(t, "not_serviceable: provider health reports unavailable", resolution.Candidates[0].Reason)
	require.NotContains(t, resolution.Candidates[0].Reason, "not_installed")
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
	var calls atomic.Int32
	r := NewLiveResolver(func(context.Context, ...string) ([]byte, error) {
		calls.Add(1)
		return []byte(`{"installed":true,"running":true,"healthy":true,"status_code":"ok","resource":{"observed_mode":"cpu"}}`), nil
	}, time.Minute)
	registry := New(Manifest{Engines: []Engine{{ID: "engine", Kind: KindLocalResource, Resource: "resource"}}})
	_, err := r.Resolve(context.Background(), registry)
	require.NoError(t, err)
	_, err = r.Resolve(context.Background(), registry)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load(), "one status and one accelerator observation per engine")
	r.Invalidate()
	_, err = r.Resolve(context.Background(), registry)
	require.NoError(t, err)
	require.Equal(t, int32(4), calls.Load())
}

func TestLiveResolverAdmissionDoesNotWaitForColdInventory(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	resolver := NewLiveResolver(func(ctx context.Context, args ...string) ([]byte, error) {
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if len(args) > 1 && args[1] == "acceleration" {
			return []byte(`{"selected":"cpu","placement":{"state":"ok"}}`), nil
		}
		return []byte(`{"installed":true,"running":true,"healthy":true,"status_code":"ok","resource":{"observed_mode":"cpu"}}`), nil
	}, time.Minute)
	registry := New(Manifest{Engines: []Engine{{ID: "engine", Kind: KindLocalResource, Resource: "resource"}}})

	startedAt := time.Now()
	resolution, err := resolver.ResolveForAdmission(context.Background(), registry)
	if !errors.Is(err, ErrNoServiceableEngine) {
		t.Fatalf("cold admission error = %v, want ErrNoServiceableEngine", err)
	}
	if resolution.Selected != "" {
		t.Fatalf("cold admission selected %q before inventory was known", resolution.Selected)
	}
	if elapsed := time.Since(startedAt); elapsed > 100*time.Millisecond {
		t.Fatalf("cold admission blocked for %s", elapsed)
	}
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("coalesced cold inventory probes did not start")
		}
	}
	close(release)

	deadline := time.Now().Add(time.Second)
	for {
		resolution, err = resolver.ResolveForAdmission(context.Background(), registry)
		if err == nil && resolution.Selected == "engine" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("refreshed admission = %+v, %v", resolution, err)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestLiveResolverRejectsNilRequestContext(t *testing.T) {
	registry := New(Manifest{Engines: []Engine{{ID: "engine"}}})
	if _, err := NewLiveResolver(nil, time.Minute).Resolve(nil, registry); err == nil {
		t.Fatal("Resolve(nil) unexpectedly substituted an uncancelable context")
	}
}

func TestLiveResolverUsesBoundedProviderLivenessWhileExpiredInventoryRefreshes(t *testing.T) {
	var calls atomic.Int32
	refreshStarted := make(chan struct{}, 2)
	releaseRefresh := make(chan struct{})
	defer close(releaseRefresh)
	resolver := NewLiveResolver(func(ctx context.Context, args ...string) ([]byte, error) {
		if calls.Add(1) > 2 {
			refreshStarted <- struct{}{}
			select {
			case <-releaseRefresh:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if len(args) > 1 && args[0] == "resource" && args[1] == "acceleration" {
			return []byte(`{"selected":"cpu","placement":{"state":"ok"}}`), nil
		}
		return []byte(`{"installed":true,"running":true,"healthy":true,"status_code":"ok","resource":{"observed_mode":"cpu"}}`), nil
	}, 100*time.Millisecond)
	resolver.SetAvailability(func(context.Context, string) bool { return true })
	registry := New(Manifest{Engines: []Engine{{ID: "engine", Kind: KindLocalResource, Resource: "resource", Strategies: []string{"passthrough"}}}})

	if resolution, err := resolver.Resolve(context.Background(), registry); err != nil || resolution.Selected != "engine" {
		t.Fatalf("initial Resolve() = %+v, %v", resolution, err)
	}
	resolver.mu.Lock()
	resolver.checked = time.Now().Add(-150 * time.Millisecond)
	resolver.mu.Unlock()

	started := time.Now()
	if resolution, err := resolver.Resolve(context.Background(), registry); err != nil || resolution.Selected != "engine" {
		t.Fatalf("stale Resolve() = %+v, %v", resolution, err)
	}
	for range 2 {
		select {
		case <-refreshStarted:
		case <-time.After(time.Second):
			resolver.mu.Lock()
			defer resolver.mu.Unlock()
			t.Fatalf("background inventory refresh did not start (calls=%d age=%s refreshing=%t cached=%t)", calls.Load(), time.Since(resolver.checked), resolver.refreshing, resolver.facts != nil)
		}
	}
	for range 5 {
		if resolution, err := resolver.Resolve(context.Background(), registry); err != nil || resolution.Selected != "engine" {
			t.Fatalf("coalesced stale Resolve() = %+v, %v", resolution, err)
		}
	}
	if elapsed := time.Since(started); elapsed > 400*time.Millisecond {
		t.Fatalf("expired-cache admission blocked for %s", elapsed)
	}
	if got := calls.Load(); got != 4 {
		t.Fatalf("control-plane probe count = %d, want one coalesced two-probe refresh plus the initial probes", got)
	}
}

func TestLiveResolverFailsClosedWhenInventoryExceedsFreshnessBound(t *testing.T) {
	var calls atomic.Int32
	refreshStarted := make(chan struct{}, 2)
	releaseRefresh := make(chan struct{})
	resolver := NewLiveResolver(func(ctx context.Context, args ...string) ([]byte, error) {
		if calls.Add(1) > 2 {
			refreshStarted <- struct{}{}
			select {
			case <-releaseRefresh:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if len(args) > 1 && args[1] == "acceleration" {
			return []byte(`{"selected":"cpu","placement":{"state":"ok"}}`), nil
		}
		return []byte(`{"installed":true,"running":true,"healthy":true,"status_code":"ok","resource":{"observed_mode":"cpu"}}`), nil
	}, time.Minute)
	resolver.SetAvailability(func(context.Context, string) bool { return true })
	registry := New(Manifest{Engines: []Engine{{ID: "engine", Kind: KindLocalResource, Resource: "resource", Strategies: []string{"passthrough"}}}})
	if _, err := resolver.Resolve(context.Background(), registry); err != nil {
		t.Fatal(err)
	}
	resolver.mu.Lock()
	resolver.checked = time.Now().Add(-3 * time.Minute)
	resolver.mu.Unlock()

	started := time.Now()
	resolution, err := resolver.Resolve(context.Background(), registry)
	if !errors.Is(err, ErrNoServiceableEngine) {
		t.Fatalf("over-age Resolve() error = %v, want fail-closed no-serviceable error", err)
	}
	if resolution.Selected != "" {
		t.Fatalf("over-age Resolve() selected %q, want none", resolution.Selected)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("over-age admission synchronously waited for inventory refresh: %s", elapsed)
	}
	for range 2 {
		select {
		case <-refreshStarted:
		case <-time.After(time.Second):
			t.Fatal("over-age inventory refresh did not start asynchronously")
		}
	}
	close(releaseRefresh)
	deadline := time.Now().Add(time.Second)
	for {
		resolver.mu.Lock()
		refreshed := resolver.checked.After(time.Now().Add(-time.Second))
		resolver.mu.Unlock()
		if refreshed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("released inventory refresh did not publish fresh facts")
		}
		time.Sleep(time.Millisecond)
	}
	resolution, err = resolver.Resolve(context.Background(), registry)
	if err != nil || resolution.Selected != "engine" {
		t.Fatalf("fresh Resolve() = %+v, %v; want engine selected after refresh", resolution, err)
	}
}

func TestLiveResolverInvalidationWinsOverInFlightBackgroundRefresh(t *testing.T) {
	refreshStarted := make(chan struct{}, 2)
	releaseRefresh := make(chan struct{})
	var ordinaryCalls atomic.Int32
	resolver := NewLiveResolver(func(ctx context.Context, args ...string) ([]byte, error) {
		isRefresh, _ := ctx.Value("background-refresh").(bool)
		if isRefresh {
			refreshStarted <- struct{}{}
			select {
			case <-releaseRefresh:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if len(args) > 1 && args[0] == "resource" && args[1] == "acceleration" {
			mode := "cuda"
			if isRefresh {
				mode = "cpu"
			}
			return []byte(fmt.Sprintf(`{"selected":%q,"placement":{"state":"ok"}}`, mode)), nil
		}
		ordinaryCalls.Add(1)
		return []byte(`{"installed":true,"running":true,"healthy":true,"status_code":"ok","resource":{"observed_mode":"cuda"}}`), nil
	}, time.Minute)
	resolver.SetAvailability(func(context.Context, string) bool { return true })
	registry := New(Manifest{Engines: []Engine{{ID: "engine", Kind: KindLocalResource, Resource: "resource", Strategies: []string{"passthrough"}}}})
	require.NoError(t, func() error { _, err := resolver.Resolve(context.Background(), registry); return err }())

	// Age the snapshot and trigger a held background refresh. Invalidation and
	// the next cold Resolve then publish a newer snapshot before that refresh ends.
	resolver.mu.Lock()
	resolver.checked = time.Now().Add(-90 * time.Second)
	resolver.mu.Unlock()
	refreshCtx := context.WithValue(context.Background(), "background-refresh", true)
	_, err := resolver.Resolve(refreshCtx, registry)
	require.NoError(t, err)
	for range 2 {
		select {
		case <-refreshStarted:
		case <-time.After(time.Second):
			t.Fatal("background refresh probes did not start")
		}
	}

	resolver.Invalidate()
	_, err = resolver.Resolve(context.Background(), registry)
	require.NoError(t, err)
	resolver.mu.Lock()
	require.True(t, resolver.facts["engine"].Accelerated, "post-invalidation observation should publish CUDA facts")
	resolver.mu.Unlock()

	close(releaseRefresh)
	deadline := time.Now().Add(time.Second)
	for {
		resolver.mu.Lock()
		refreshing := resolver.refreshing
		accelerated := resolver.facts["engine"].Accelerated
		resolver.mu.Unlock()
		if !refreshing {
			require.True(t, accelerated, "obsolete refresh must not overwrite post-invalidation facts")
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("background refresh did not finish")
		}
		time.Sleep(time.Millisecond)
	}
	require.GreaterOrEqual(t, ordinaryCalls.Load(), int32(2), "initial and cold observations should each run a control-plane status probe")
}

func TestLiveResolverAcceptsHealthyLifecycleStatusWhenStatusCodeIsAbsent(t *testing.T) {
	var calls atomic.Int32
	r := NewLiveResolver(func(_ context.Context, args ...string) ([]byte, error) {
		calls.Add(1)
		if args[0] == "resource" && args[1] == "acceleration" {
			return []byte(`{"selected":"cuda","placement":{"state":"ok"}}`), nil
		}
		return []byte(`{"installed":true,"running":true,"healthy":true,"status":"healthy","resource":{"observed_mode":"cuda"}}`), nil
	}, time.Minute)
	registry := New(Manifest{Engines: []Engine{{ID: "engine", Kind: KindLocalResource, Resource: "resource", Strategies: []string{"passthrough"}}}})
	resolution, err := r.Resolve(context.Background(), registry)
	require.NoError(t, err)
	require.Equal(t, "engine", resolution.Selected)
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

// TestEmbeddedManifestDeclaresSherpaStreaming pins the native streaming engine
// the provider catalogue represents as sherpa-stt.
func TestEmbeddedManifestDeclaresSherpaStreaming(t *testing.T) {
	engine, ok := Default().Engine("sherpa-streaming")
	require.True(t, ok, "manifest must declare sherpa-streaming")
	require.Equal(t, KindLocalResource, engine.Kind)
	require.Equal(t, "sherpa-onnx", engine.Resource)
	require.True(t, engine.Provides.NativeStreaming)
	require.True(t, engine.Provides.LowFootprint)
	require.False(t, engine.Provides.BuiltinVad)
	require.Empty(t, engine.Provides.ConfidenceSignals)
	require.False(t, engine.Provides.WordTimestamps)
	require.True(t, engine.Requires.PCM16kMono)
	require.Equal(t, []string{"passthrough"}, engine.Strategies)
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

// TestLocalResourceEnginesDeclaredInServiceJSON asserts lifecycle-managed
// local_resource engines appear in dependencies.resources. Explicit attach-only
// runtimes are externally managed and must remain outside scenario lifecycle.
func TestLocalResourceEnginesDeclaredInServiceJSON(t *testing.T) {
	declared := serviceResourceDeps(t)
	for _, e := range Default().Engines() {
		if e.Kind != KindLocalResource {
			continue
		}
		_, ok := declared[e.Resource]
		if e.AttachOnly {
			require.Falsef(t, ok, "attach-only engine %q resource %q must not be started by scenario lifecycle", e.ID, e.Resource)
			continue
		}
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

// TestResolveFactsPrefersLowFootprintStreamingOverAccelerated pins the
// resource-conscious policy: a lightweight native-streaming engine wins over a
// GPU streamer even though the GPU engine reports acceleration. Footprint is a
// product decision, not an accident of whichever engine happens to be up.
func TestResolveFactsPrefersLowFootprintStreamingOverAccelerated(t *testing.T) {
	registry := &Registry{manifest: Manifest{
		SelectionPolicy: SelectionPolicy{
			HardFilters:       []string{"platform_supported", "resource_serviceable"},
			RankedPreferences: []string{"native_streaming", "low_footprint", "accelerated"},
		},
		Engines: []Engine{
			{ID: "sherpa", Provides: Provides{NativeStreaming: true, LowFootprint: true}},
			{ID: "kyutai", Provides: Provides{NativeStreaming: true}},
		},
	}}
	serviceable := EngineFacts{PlatformSupported: true, Installed: true, Running: true, Healthy: true, WorkloadCapable: true}
	gpu := serviceable
	gpu.Accelerated = true
	resolution, err := registry.ResolveFacts(map[string]EngineFacts{
		"sherpa": serviceable,
		"kyutai": gpu,
	})
	require.NoError(t, err)
	require.Equal(t, "sherpa", resolution.Selected)
	require.Contains(t, resolution.Candidates[0].Reason, "native_streaming=true")
}

// The shipped selection policy must make the operator's low-footprint
// preference decisive when more than one native streamer is healthy.
func TestShippedPolicyPrefersSherpaOverKyutai(t *testing.T) {
	registry := Default()
	serviceable := EngineFacts{PlatformSupported: true, Installed: true, Running: true, Healthy: true, WorkloadCapable: true}
	resolution, err := registry.ResolveFacts(map[string]EngineFacts{
		"whisper-local":    serviceable,
		"kyutai":           serviceable,
		"sherpa-streaming": serviceable,
	})
	require.NoError(t, err)
	require.Equal(t, "sherpa-streaming", resolution.Selected)
}

// TestLiveResolverAcceptsAttachedProvider covers attach-only operation: the
// operator runs a qualified adapter outside Vrooli's lifecycle, and the
// provider's own health probe proves it works. The control plane still reports
// the managed artifact as uninstalled/unsupported, which must not hide a
// deliberately attached server.
func TestLiveResolverAcceptsAttachedProvider(t *testing.T) {
	r := NewLiveResolver(func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 1 && args[0] == "resource" && args[1] == "acceleration" {
			return []byte(`{"selected":"cpu","placement":{"state":"ok"}}`), nil
		}
		return []byte(`{"installed":false,"running":false,"healthy":false,"status_code":"unsupported_platform"}`), nil
	}, time.Minute)
	r.SetAvailability(func(context.Context, string) bool { return true })
	registry := New(Manifest{Engines: []Engine{{
		ID: "sherpa-streaming", Kind: KindLocalResource, Resource: "sherpa-onnx",
		Provides: Provides{NativeStreaming: true, LowFootprint: true}, Strategies: []string{"passthrough"},
	}}})

	resolution, err := r.Resolve(context.Background(), registry)
	require.NoError(t, err)
	require.Equal(t, "sherpa-streaming", resolution.Selected)
	require.True(t, resolution.Candidates[0].Reason != "", "attached selection must explain itself")
	require.Contains(t, resolution.Candidates[0].Reason, "attached")
}

// TestLiveResolverRejectsUnhealthyAttachedProvider proves the other side of
// attach-only: a control-plane record alone cannot keep a dead server
// serviceable when the provider health probe fails.
func TestLiveResolverRejectsUnhealthyAttachedProvider(t *testing.T) {
	r := NewLiveResolver(func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 1 && args[0] == "resource" && args[1] == "acceleration" {
			return []byte(`{"selected":"cpu","placement":{"state":"ok"}}`), nil
		}
		return []byte(`{"installed":true,"running":true,"healthy":true,"status_code":"ok"}`), nil
	}, time.Minute)
	r.SetAvailability(func(context.Context, string) bool { return false })
	registry := New(Manifest{Engines: []Engine{{
		ID: "sherpa-streaming", Kind: KindLocalResource, Resource: "sherpa-onnx", Strategies: []string{"passthrough"},
	}}})

	_, err := r.Resolve(context.Background(), registry)
	require.ErrorIs(t, err, ErrNoServiceableEngine)
}
