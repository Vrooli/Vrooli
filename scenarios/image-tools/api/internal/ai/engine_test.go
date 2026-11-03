package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"image-tools/internal/adapters"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/blobstore"
)

// fakeProvider is a standalone backends.Provider that writes a fixed output and
// records the request it received, so tests assert the full vertical (select →
// execute → persist) without a real backend binary.
type fakeProvider struct {
	name      string
	ops       []string
	out       []byte
	meta      map[string]string
	metas     []map[string]string
	lastReq   backends.Request
	execN     int
	failWith  error
	failOn    int
	failMeta  map[string]string
	onExecute func()
}

func TestOutputFormatRetainsGeometryArchive(t *testing.T) {
	for _, meta := range []map[string]string{nil, {"media_type": "application/zip"}} {
		mime, ext := outputFormat([]byte{'P', 'K', 3, 4, 0, 0, 0, 0}, meta)
		if mime != "application/zip" || ext != ".zip" {
			t.Fatalf("archive mislabeled as an image: %s %s", mime, ext)
		}
	}
}

func (p *fakeProvider) Name() string                   { return p.name }
func (p *fakeProvider) Operations() []string           { return p.ops }
func (p *fakeProvider) Standalone() bool               { return true }
func (p *fakeProvider) IsCloud() bool                  { return false }
func (p *fakeProvider) Available(context.Context) bool { return true }

func (p *fakeProvider) Execute(_ context.Context, req backends.Request) (backends.Result, error) {
	p.lastReq = req
	p.execN++
	if p.onExecute != nil {
		p.onExecute()
	}
	if p.failWith != nil {
		if p.failOn == 0 || p.execN == p.failOn {
			return backends.Result{Meta: p.failMeta}, p.failWith
		}
	}
	out := p.out
	if out == nil {
		out = []byte("fake-output-bytes")
	}
	if err := os.WriteFile(req.Output.LocalPath, out, 0o600); err != nil {
		return backends.Result{}, err
	}
	meta := p.meta
	if p.execN <= len(p.metas) && p.execN > 0 {
		meta = p.metas[p.execN-1]
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: meta}, nil
}

func TestRunFailureRetainsGatewayVariationEvidenceAndMarksUnstarted(t *testing.T) {
	firstAttempt, err := json.Marshal(gatewayExecutionAttempt{Outcome: "succeeded", ExecutionID: "gw-first", Status: "SUCCEEDED", CostState: "known", CostUSD: ptrCost(0)})
	if err != nil {
		t.Fatal(err)
	}
	failedAttempt, err := json.Marshal(gatewayExecutionAttempt{Outcome: "outcome_unknown", ExecutionID: "gw-second", CostState: "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	fp := &fakeProvider{
		metas: []map[string]string{
			{MetaGatewayExecutionAttempt: string(firstAttempt), MetaCostUSD: "0", MetaCostProvenance: ProviderReportedCostProvenance},
			{MetaGatewayExecutionAttempt: string(failedAttempt)},
		},
		failOn:   2,
		failWith: errors.New("gateway wait interrupted"),
		failMeta: map[string]string{MetaGatewayExecutionAttempt: string(failedAttempt)},
	}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)
	runner := eng.BuildRunners()["text_to_image"]
	payload, err := json.Marshal(Payload{Operation: "text_to_image", ModelID: modelID, GPU: true, Variations: 3, Params: map[string]string{"prompt": "partial evidence"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner(context.Background(), internaljobs.Job{Operation: "text_to_image", Payload: payload}, func(int, string) {})
	if err == nil {
		t.Fatal("expected second variation failure")
	}
	if len(result.Meta) != 1 {
		t.Fatalf("failed result metadata = %#v", result.Meta)
	}
	var evidence gatewayExecutionEvidence
	if err := json.Unmarshal([]byte(result.Meta[internaljobs.MetaGatewayExecutionEvidence]), &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.SchemaVersion != 1 || evidence.Complete || evidence.RequestedVariations != 3 || len(evidence.Variations) != 3 {
		t.Fatalf("evidence header = %+v", evidence)
	}
	if first := evidence.Variations[0]; first.Outcome != "succeeded" || first.ExecutionID != "gw-first" || first.CostState != "known" || first.CostUSD == nil || *first.CostUSD != 0 {
		t.Fatalf("first variation evidence = %+v", first)
	}
	if second := evidence.Variations[1]; second.Outcome != "outcome_unknown" || second.ExecutionID != "gw-second" || second.CostState != "unknown" || second.CostUSD != nil {
		t.Fatalf("second variation evidence = %+v", second)
	}
	if third := evidence.Variations[2]; third.Outcome != "not_started" || third.CostState != "unknown" || third.CostUSD != nil {
		t.Fatalf("third variation evidence = %+v", third)
	}
}

func ptrCost(value float64) *float64 { return &value }

func TestVariationCostRequiresEveryProviderReportedContribution(t *testing.T) {
	tests := []struct {
		name  string
		metas []map[string]string
		want  string
	}{
		{name: "all known including zero", metas: []map[string]string{
			{MetaCostUSD: "0.1", MetaCostProvenance: ProviderReportedCostProvenance},
			{MetaCostUSD: "0", MetaCostProvenance: ProviderReportedCostProvenance},
			{MetaCostUSD: "0.3", MetaCostProvenance: ProviderReportedCostProvenance},
		}, want: "0.4"},
		{name: "unknown first", metas: []map[string]string{
			{}, {MetaCostUSD: "0.2", MetaCostProvenance: ProviderReportedCostProvenance}, {MetaCostUSD: "0.3", MetaCostProvenance: ProviderReportedCostProvenance},
		}},
		{name: "unknown middle", metas: []map[string]string{
			{MetaCostUSD: "0.1", MetaCostProvenance: ProviderReportedCostProvenance}, {}, {MetaCostUSD: "0.3", MetaCostProvenance: ProviderReportedCostProvenance},
		}},
		{name: "unknown last", metas: []map[string]string{
			{MetaCostUSD: "0.1", MetaCostProvenance: ProviderReportedCostProvenance}, {MetaCostUSD: "0.2", MetaCostProvenance: ProviderReportedCostProvenance}, {},
		}},
		{name: "malformed contribution", metas: []map[string]string{
			{MetaCostUSD: "0.1", MetaCostProvenance: ProviderReportedCostProvenance}, {MetaCostUSD: "NaN", MetaCostProvenance: ProviderReportedCostProvenance}, {MetaCostUSD: "0.3", MetaCostProvenance: ProviderReportedCostProvenance},
		}},
		{name: "unmarked amount is unknown", metas: []map[string]string{
			{MetaCostUSD: "0.1"}, {MetaCostUSD: "0.2", MetaCostProvenance: ProviderReportedCostProvenance}, {MetaCostUSD: "0.3", MetaCostProvenance: ProviderReportedCostProvenance},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fp := &fakeProvider{metas: tc.metas}
			eng, _, modelID := newTestEngine(t, "text_to_image", fp)
			runner := eng.BuildRunners()["text_to_image"]
			payload, err := json.Marshal(Payload{Operation: "text_to_image", ModelID: modelID, GPU: true, Variations: 3, Params: map[string]string{"prompt": "cost test"}})
			if err != nil {
				t.Fatal(err)
			}
			result, err := runner(context.Background(), internaljobs.Job{Operation: "text_to_image", Payload: payload}, func(int, string) {})
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if _, ok := result.Meta[MetaCostUSD]; ok {
					t.Fatalf("partial or unmarked cost published: %#v", result.Meta)
				}
				if _, ok := result.Meta[MetaCostProvenance]; ok {
					t.Fatalf("provenance published without complete cost: %#v", result.Meta)
				}
				return
			}
			if result.Meta[MetaCostUSD] != tc.want || result.Meta[MetaCostProvenance] != ProviderReportedCostProvenance {
				t.Fatalf("aggregated cost metadata = %#v", result.Meta)
			}
		})
	}
}

func TestRunLeasesModelAcrossVariationsAndReleasesBeforeAutoScan(t *testing.T) {
	gate := models.NewModelUseGate()
	fp := &fakeProvider{}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)
	eng.deps.AcquireModelUse = gate.Acquire
	eng.deps.AutoScan = func(context.Context, []byte) (bool, float64, error) {
		release, err := gate.AcquireExclusive(modelID)
		if err != nil {
			t.Errorf("model lease should be released before NSFW auto-scan; exclusive acquire err=%v", err)
		} else {
			release()
		}
		return false, 0, nil
	}
	fp.onExecute = func() {
		release, err := gate.AcquireExclusive(modelID)
		if err == nil {
			t.Error("model lease should cover backend execution; exclusive acquire succeeded")
			release()
		} else if !errors.Is(err, models.ErrModelInUse) {
			t.Errorf("exclusive acquire returned unexpected error: %v", err)
		}
	}
	payload, err := json.Marshal(Payload{
		Operation: "text_to_image", ModelID: modelID, GPU: true, Variations: 2,
		AutoScanNSFW: true, Params: map[string]string{"prompt": "lease coverage"},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := eng.BuildRunners()["text_to_image"]
	if _, err := runner(context.Background(), internaljobs.Job{Operation: "text_to_image", Payload: payload}, func(int, string) {}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if fp.execN != 2 {
		t.Fatalf("backend executions=%d, want two variations", fp.execN)
	}
	release, err := gate.AcquireExclusive(modelID)
	if err != nil {
		t.Fatalf("model lease remained after run: %v", err)
	}
	release()
}

type failingProbe struct{ err error }

func (p failingProbe) Inventory(context.Context) (capabilities.Host, error) {
	return capabilities.Host{}, p.err
}

type unavailableProvider struct {
	name string
	ops  []string
}

func (p unavailableProvider) Name() string                   { return p.name }
func (p unavailableProvider) Operations() []string           { return p.ops }
func (p unavailableProvider) Standalone() bool               { return true }
func (p unavailableProvider) IsCloud() bool                  { return false }
func (p unavailableProvider) Available(context.Context) bool { return false }
func (p unavailableProvider) Execute(context.Context, backends.Request) (backends.Result, error) {
	return backends.Result{}, errors.New("unavailable provider should not execute")
}

// newTestEngine builds an engine over the real model registry, a memory-backed
// blob store, a fake host probe, and a single fake provider registered for op
// under model's backend name (so selection matches it).
func newTestEngine(t *testing.T, op string, fp *fakeProvider) (*Engine, *storage.Store, string) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := registry.DefaultFor(op)
	if !ok {
		t.Fatalf("no default model for %q", op)
	}
	fp.name = def.Backend
	fp.ops = []string{op}

	backendReg := backends.New()
	if err := backendReg.Register(fp); err != nil {
		t.Fatalf("register fake provider: %v", err)
	}

	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	host := capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}

	eng, err := NewEngine(Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: host},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
		ModelsRoot:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return eng, store, def.ID
}

// runJob drives the op's runner with a payload, returning the result ref.
func runJob(t *testing.T, eng *Engine, op string, pl Payload) (string, error) {
	t.Helper()
	raw, err := json.Marshal(pl)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	runner := eng.BuildRunners()[op]
	if runner == nil {
		t.Fatalf("no runner for %q", op)
	}
	result, err := runner(context.Background(), internaljobs.Job{Operation: op, Payload: raw}, func(int, string) {})
	return result.Ref, err
}

func TestRunRejectsPersistedModelUnsupportedOnCurrentHost(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	base, ok := registry.DefaultFor("text_to_image")
	if !ok {
		t.Fatal("text_to_image default missing")
	}
	restricted := base
	restricted.ID = "restricted-host-test"
	restricted.Name = "Restricted host test"
	restricted.DefaultFor = nil
	restricted.Hardware.OSArch = []string{"linux/amd64"}
	fp := &fakeProvider{name: base.Backend, ops: []string{"text_to_image"}}
	backendReg := backends.New()
	if err := backendReg.Register(fp); err != nil {
		t.Fatal(err)
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	eng, err := NewEngine(Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "windows", Arch: "arm64", Cores: 8}},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
		CustomModels:   func(context.Context) ([]models.Model, error) { return []models.Model{restricted}, nil },
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	_, runErr := runJob(t, eng, "text_to_image", Payload{
		Operation: "text_to_image", ModelID: restricted.ID, GPU: false,
		Params: map[string]string{"prompt": "must not reach provider"},
	})
	if !errors.Is(runErr, ErrModelUnsupportedHost) {
		t.Fatalf("runner error = %v, want ErrModelUnsupportedHost", runErr)
	}
	if fp.execN != 0 {
		t.Fatalf("unsupported model reached provider %d times", fp.execN)
	}
}

// storeInput writes bytes under a key in the store (helper for image-input ops).
func storeInput(t *testing.T, store *storage.Store, key string, data []byte) {
	t.Helper()
	if err := store.Put(context.Background(), key, bytes.NewReader(data), "image/png"); err != nil {
		t.Fatalf("store input: %v", err)
	}
}

func TestRunJob_PersistsProviderMediaType(t *testing.T) {
	fp := &fakeProvider{
		out:  []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		meta: map[string]string{"media_type": "image/svg+xml"},
	}
	eng, store, modelID := newTestEngine(t, "text_to_image", fp)

	ref, err := runJob(t, eng, "text_to_image", Payload{
		Operation: "text_to_image",
		ModelID:   modelID,
		GPU:       true,
		Params:    map[string]string{"prompt": "logo"},
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}
	if !strings.HasSuffix(ref, ".svg") {
		t.Fatalf("result ref = %q, want .svg", ref)
	}
	rc, mime, err := store.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("get stored output: %v", err)
	}
	defer func() { _ = rc.Close() }()
	if mime != "image/svg+xml" {
		t.Fatalf("mime = %q, want image/svg+xml", mime)
	}
}

func TestPlan_ModelNotInstalled(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	backendReg := backends.New()
	if err := RegisterProviders(backendReg, func(string) (string, error) { return "/usr/bin/x", nil }, nil, ""); err != nil {
		t.Fatalf("register providers: %v", err)
	}
	eng, err := NewEngine(Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:          storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir()),
		ModelInstalled: func(string) bool { return false },
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	_, err = eng.Plan(context.Background(), PlanRequest{Operation: "text_to_image"})
	if err == nil {
		t.Fatal("expected ErrModelNotInstalled")
	}
}

func TestPlan_SuccessUsesDefaultOverrideWarningsAndETA(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := registry.DefaultFor("naturalize")
	if !ok {
		t.Fatal("naturalize default missing")
	}
	fp := &fakeProvider{name: def.Backend, ops: []string{"naturalize"}}
	backendReg := backends.New()
	if err := backendReg.Register(fp); err != nil {
		t.Fatalf("register fake provider: %v", err)
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	eng, err := NewEngine(Deps{
		Registry: registry,
		Backends: backendReg,
		Probe: capabilities.FakeProbe{Host: capabilities.Host{
			OS: "linux", Arch: "amd64", Cores: 8,
			GPUs: []capabilities.GPU{{Name: "unknown", VRAMBytes: 0}},
		}},
		Store: store,
		Enabled: func(context.Context) (models.EnabledFunc, error) {
			return func(id string) bool { return id == def.ID }, nil
		},
		DefaultOverride: func(_ context.Context, op string) (string, error) {
			if op != "naturalize" {
				t.Fatalf("default override requested for %q", op)
			}
			return def.ID, nil
		},
		ModelInstalled: func(id string) bool { return id == def.ID },
		ModelRuntimeReadiness: func(context.Context, string) (bool, string) {
			return true, "ready"
		},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	plan, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize"})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.ModelID != def.ID || plan.Tier != backends.TierLocalCPU.String() || plan.EstimatedSeconds != 2 {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.GPUViable {
		t.Fatal("builtin naturalize should not be GPU-viable")
	}
	if len(plan.Warnings) == 0 {
		t.Fatalf("expected CPU warning in plan")
	}
	eng.deps.ModelRuntimeReadiness = func(context.Context, string) (bool, string) {
		return false, "smoke_pending"
	}
	if _, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize"}); !errors.Is(err, ErrModelNotRuntimeReady) {
		t.Fatalf("Plan error = %v, want ErrModelNotRuntimeReady", err)
	}
}

func TestPlan_ErrorSurfaces(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := registry.DefaultFor("naturalize")
	if !ok {
		t.Fatal("naturalize default missing")
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	backendReg := backends.New()
	if err := backendReg.Register(&fakeProvider{name: def.Backend, ops: []string{"naturalize"}}); err != nil {
		t.Fatalf("register fake provider: %v", err)
	}
	base := Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
	}
	eng, err := NewEngine(base)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	if _, err := eng.Plan(context.Background(), PlanRequest{Operation: "teleport"}); err == nil || !strings.Contains(err.Error(), "unknown operation") {
		t.Fatalf("expected unknown operation error, got %v", err)
	}

	hostErrDeps := base
	hostErrDeps.Probe = failingProbe{err: errors.New("inventory down")}
	eng, err = NewEngine(hostErrDeps)
	if err != nil {
		t.Fatalf("new engine host err: %v", err)
	}
	if _, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize"}); err == nil || !strings.Contains(err.Error(), "host probe") {
		t.Fatalf("expected host probe error, got %v", err)
	}

	enabledErrDeps := base
	enabledErrDeps.Enabled = func(context.Context) (models.EnabledFunc, error) { return nil, errors.New("overlay down") }
	eng, err = NewEngine(enabledErrDeps)
	if err != nil {
		t.Fatalf("new engine enabled err: %v", err)
	}
	if _, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize"}); err == nil || !strings.Contains(err.Error(), "load enabled overlay") {
		t.Fatalf("expected enabled overlay error, got %v", err)
	}

	defaultErrDeps := base
	defaultErrDeps.DefaultOverride = func(context.Context, string) (string, error) { return "", errors.New("settings down") }
	eng, err = NewEngine(defaultErrDeps)
	if err != nil {
		t.Fatalf("new engine default err: %v", err)
	}
	if _, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize"}); err == nil || !strings.Contains(err.Error(), "load op default") {
		t.Fatalf("expected default override error, got %v", err)
	}

	unavailableReg := backends.New()
	if err := unavailableReg.Register(unavailableProvider{name: def.Backend, ops: []string{"naturalize"}}); err != nil {
		t.Fatalf("register unavailable provider: %v", err)
	}
	backendErrDeps := base
	backendErrDeps.Backends = unavailableReg
	eng, err = NewEngine(backendErrDeps)
	if err != nil {
		t.Fatalf("new engine backend err: %v", err)
	}
	if _, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize"}); err == nil || !errors.Is(err, backends.ErrNoneAvailable) {
		t.Fatalf("expected backend availability error, got %v", err)
	}
}

// TestMaterializeAdapterImages covers the Phase 5/6 conditioning-image plumbing:
// each adapter's ConditioningImageKey blob is fetched to a local file and the key
// rewritten to that path; LoRAs (no image) pass through untouched.
func TestMaterializeAdapterImages(t *testing.T) {
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	ctx := context.Background()
	if err := store.Put(ctx, "blob/cond.png", bytes.NewReader([]byte("controlmap")), "image/png"); err != nil {
		t.Fatalf("put: %v", err)
	}
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	eng, err := NewEngine(Deps{
		Registry:       registry,
		Backends:       backends.New(),
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	dir := t.TempDir()
	in := []adapters.ResolvedAdapter{
		{ID: "lcm", Kind: adapters.KindLoRA}, // no image — untouched
		{ID: "cn", Kind: adapters.KindControlNet, ConditioningImageKey: "blob/cond.png"},
	}
	out, err := eng.materializeAdapterImages(ctx, dir, in)
	if err != nil {
		t.Fatalf("materializeAdapterImages: %v", err)
	}
	if out[0].ConditioningImageKey != "" {
		t.Fatalf("lora image key should stay empty, got %q", out[0].ConditioningImageKey)
	}
	if !strings.HasPrefix(out[1].ConditioningImageKey, dir) {
		t.Fatalf("controlnet image not materialized under tmp dir: %q", out[1].ConditioningImageKey)
	}
	if data, rerr := os.ReadFile(out[1].ConditioningImageKey); rerr != nil || string(data) != "controlmap" {
		t.Fatalf("materialized file mismatch: data=%q err=%v", data, rerr)
	}
	// A missing blob fails closed.
	if _, err := eng.materializeAdapterImages(ctx, dir, []adapters.ResolvedAdapter{{ID: "x", Kind: adapters.KindIPAdapter, ConditioningImageKey: "blob/missing.png"}}); err == nil {
		t.Fatal("expected error for a missing conditioning blob")
	}
}

func TestEngineHelpers(t *testing.T) {
	if LaneForTier("local-gpu") != internaljobs.LaneGPU {
		t.Fatalf("local-gpu tier should use the serialized local queue")
	}
	if LaneForTier("byok-cloud") != internaljobs.LaneNetwork {
		t.Fatalf("byok-cloud tier should use the network lane")
	}
	if estimateSeconds("naturalize", false) != 2 {
		t.Fatal("naturalize should keep deterministic short ETA")
	}
	if got := estimateSeconds("upscale", false); got != 90 {
		t.Fatalf("CPU enhancement ETA = %d, want 90", got)
	}
	if got := estimateSeconds("text_to_image", true); got != 30 {
		t.Fatalf("GPU generation ETA = %d, want 30", got)
	}
}
