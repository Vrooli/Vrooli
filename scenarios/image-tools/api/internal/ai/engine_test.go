package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"image-tools/internal/adapters"
	aigateway "image-tools/internal/aigateway"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/api-core/blobstore"
)

func allowRightsForTest(context.Context, json.RawMessage, string) (models.RightsDecision, error) {
	return models.RightsDecision{Status: "allowed", DecisionSHA256: "test"}, nil
}

func TestRequiredWarmGPUCapacityDenialIsRetryable(t *testing.T) {
	err := &warmGPUCapacityFallbackError{detail: "GPU capacity denied warm model residency"}
	if !errors.Is(err, internaljobs.ErrCapacityBusy) {
		t.Fatalf("warm GPU denial must remain a retryable capacity refusal: %v", err)
	}
}

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
func (p *fakeProvider) GPUCapable() bool               { return true }

type mutableProvider struct {
	fakeProvider
	available bool
}

func (p *mutableProvider) Available(context.Context) bool { return p.available }

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
	firstAttempt, err := json.Marshal(aigateway.Attempt{Outcome: "succeeded", ExecutionID: "gw-first", Status: "SUCCEEDED", CostState: "known", CostUSD: ptrCost(0)})
	if err != nil {
		t.Fatal(err)
	}
	failedAttempt, err := json.Marshal(aigateway.Attempt{Outcome: "outcome_unknown", ExecutionID: "gw-second", CostState: "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	fp := &fakeProvider{
		metas: []map[string]string{
			{aigateway.MetaExecutionAttempt: string(firstAttempt), aigateway.MetaCostUSD: "0", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance},
			{aigateway.MetaExecutionAttempt: string(failedAttempt)},
		},
		failOn:   2,
		failWith: errors.New("gateway wait interrupted"),
		failMeta: map[string]string{aigateway.MetaExecutionAttempt: string(failedAttempt)},
	}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)
	runner := eng.BuildRunners()["text_to_image"]
	payload, err := marshalPinnedPayload(t, eng, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID, GPU: true, Variations: 3, Params: map[string]string{"prompt": "partial evidence"}})
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
	var evidence aigateway.Evidence
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
			{aigateway.MetaCostUSD: "0.1", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance},
			{aigateway.MetaCostUSD: "0", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance},
			{aigateway.MetaCostUSD: "0.3", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance},
		}, want: "0.4"},
		{name: "unknown first", metas: []map[string]string{
			{}, {aigateway.MetaCostUSD: "0.2", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}, {aigateway.MetaCostUSD: "0.3", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance},
		}},
		{name: "unknown middle", metas: []map[string]string{
			{aigateway.MetaCostUSD: "0.1", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}, {}, {aigateway.MetaCostUSD: "0.3", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance},
		}},
		{name: "unknown last", metas: []map[string]string{
			{aigateway.MetaCostUSD: "0.1", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}, {aigateway.MetaCostUSD: "0.2", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}, {},
		}},
		{name: "malformed contribution", metas: []map[string]string{
			{aigateway.MetaCostUSD: "0.1", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}, {aigateway.MetaCostUSD: "NaN", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}, {aigateway.MetaCostUSD: "0.3", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance},
		}},
		{name: "unmarked amount is unknown", metas: []map[string]string{
			{aigateway.MetaCostUSD: "0.1"}, {aigateway.MetaCostUSD: "0.2", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}, {aigateway.MetaCostUSD: "0.3", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fp := &fakeProvider{metas: tc.metas}
			eng, _, modelID := newTestEngine(t, "text_to_image", fp)
			runner := eng.BuildRunners()["text_to_image"]
			payload, err := marshalPinnedPayload(t, eng, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID, GPU: true, Variations: 3, Params: map[string]string{"prompt": "cost test"}})
			if err != nil {
				t.Fatal(err)
			}
			result, err := runner(context.Background(), internaljobs.Job{Operation: "text_to_image", Payload: payload}, func(int, string) {})
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if _, ok := result.Meta[aigateway.MetaCostUSD]; ok {
					t.Fatalf("partial or unmarked cost published: %#v", result.Meta)
				}
				if _, ok := result.Meta[aigateway.MetaCostProvenance]; ok {
					t.Fatalf("provenance published without complete cost: %#v", result.Meta)
				}
				return
			}
			if result.Meta[aigateway.MetaCostUSD] != tc.want || result.Meta[aigateway.MetaCostProvenance] != aigateway.ProviderReportedCostProvenance {
				t.Fatalf("aggregated cost metadata = %#v", result.Meta)
			}
		})
	}
}

func TestRunLeasesModelAcrossVariationsAndReleasesBeforeAutoScan(t *testing.T) {
	gate := models.NewModelUseGate(t.TempDir())
	fp := &fakeProvider{}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)
	eng.deps.AcquireModelUse = gate.Acquire
	eng.deps.AutoScan = func(context.Context, []byte) (ScanVerdict, error) {
		release, err := gate.AcquireExclusive(modelID)
		if err != nil {
			t.Errorf("model lease should be released before NSFW auto-scan; exclusive acquire err=%v", err)
		} else {
			release()
		}
		return ScanVerdict{}, nil
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
	payload, err := marshalPinnedPayload(t, eng, "text_to_image", Payload{
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

// newTestEngine builds an engine over the model registry, a memory-backed blob
// store, a fake host probe, and a single fake provider registered for op under
// the model's backend name (so selection matches it). Upscale uses an explicit
// synthetic test-only default because the shipped catalog intentionally has no
// enabled default while SwinIR's rights remain unverified.
func newTestEngine(t *testing.T, op string, fp *fakeProvider) (*Engine, *storage.Store, string) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if op == "upscale" {
		registry = registryWithSyntheticUpscaleDefault(t, registry)
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

	eng, err := NewEngineWithRightsEvaluator(Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: host},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
		ModelsRoot:     t.TempDir(),
		Disk:           &fakeDiskAdmission{},
	}, allowRightsForTest)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return eng, store, def.ID
}

func registryWithSyntheticUpscaleDefault(t *testing.T, base *models.Registry) *models.Registry {
	t.Helper()
	seed, ok := base.DefaultFor("denoise")
	if !ok {
		t.Fatal("synthetic upscale fixture requires the existing denoise default as a valid model template")
	}
	fixture := seed
	fixture.ID = "test-synthetic-upscale"
	fixture.Name = "Synthetic test-only upscale model"
	fixture.Operations = []string{"upscale"}
	fixture.DefaultFor = []string{"upscale"}
	fixture.Enabled = true
	fixture.CapabilityLabels.CommercialUse = models.CommercialUseYes
	fixture.CapabilityLabels.CommercialUseNotes = ""

	modelEntries := append(base.Models(), fixture)
	encoded, err := json.Marshal(struct {
		SchemaVersion string                  `json:"schema_version"`
		Models        []models.Model          `json:"models"`
		Blocklist     []models.BlocklistEntry `json:"blocklist"`
	}{base.SchemaVersion(), modelEntries, base.Blocklist()})
	if err != nil {
		t.Fatalf("marshal synthetic upscale fixture: %v", err)
	}
	registry, err := models.Parse(encoded)
	if err != nil {
		t.Fatalf("parse synthetic upscale registry fixture: %v", err)
	}
	return registry
}

// runJob drives the op's runner with a payload, returning the result ref.
func runJob(t *testing.T, eng *Engine, op string, pl Payload) (string, error) {
	t.Helper()
	raw, err := marshalPinnedPayload(t, eng, op, pl)
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

func marshalPinnedPayload(t *testing.T, eng *Engine, op string, pl Payload) ([]byte, error) {
	t.Helper()
	if pl.Operation == "" {
		pl.Operation = op
	}
	if pl.Backend == "" {
		plan, err := eng.Plan(context.Background(), PlanRequest{Operation: op, ModelOverride: pl.ModelID, AllowBYOK: pl.AllowBYOK})
		if err != nil {
			return nil, err
		}
		pl.Backend = plan.Provider
		if pl.Tier == "" {
			pl.Tier = plan.Tier
		}
	}
	return json.Marshal(pl)
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
		Registry:        registry,
		Backends:        backendReg,
		Probe:           capabilities.FakeProbe{Host: capabilities.Host{OS: "windows", Arch: "arm64", Cores: 8}},
		Store:           store,
		RightsEvaluator: allowRightsForTest,
		ModelInstalled:  func(string) bool { return true },
		CustomModels:    func(context.Context) ([]models.Model, error) { return []models.Model{restricted}, nil },
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	_, runErr := runJob(t, eng, "text_to_image", Payload{
		Operation: "text_to_image", ModelID: restricted.ID, Backend: fp.Name(), GPU: false,
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

func TestExecutionRechecksRightsAfterPlanBeforeProvider(t *testing.T) {
	fp := &fakeProvider{out: []byte("image")}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)
	var evaluations atomic.Int32
	eng.deps.RightsEvaluator = func(context.Context, json.RawMessage, string) (models.RightsDecision, error) {
		if evaluations.Add(1) == 1 {
			return models.RightsDecision{Status: "allowed"}, nil
		}
		return models.RightsDecision{Status: "denied", Reason: "current_context_changed"}, nil
	}
	_, err := runJob(t, eng, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID, Params: map[string]string{"prompt": "caller image"}})
	if err == nil || !strings.Contains(err.Error(), "current_context_changed") {
		t.Fatalf("execution rights after plan = %v, want current-context refusal", err)
	}
	if evaluations.Load() != 2 || fp.execN != 0 {
		t.Fatalf("rights evaluations=%d provider calls=%d; want execution recheck before provider", evaluations.Load(), fp.execN)
	}
}

func TestExecutionRechecksRightsBetweenVariations(t *testing.T) {
	var allowed atomic.Bool
	allowed.Store(true)
	fp := &fakeProvider{out: []byte("image"), onExecute: func() { allowed.Store(false) }}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)
	var evaluations atomic.Int32
	eng.deps.RightsEvaluator = func(context.Context, json.RawMessage, string) (models.RightsDecision, error) {
		evaluations.Add(1)
		if allowed.Load() {
			return models.RightsDecision{Status: "allowed"}, nil
		}
		return models.RightsDecision{Status: "denied", Reason: "context_changed_between_variations"}, nil
	}
	_, err := runJob(t, eng, "text_to_image", Payload{
		Operation: "text_to_image", ModelID: modelID, Variations: 2,
		Params: map[string]string{"prompt": "caller image"},
	})
	if err == nil || !strings.Contains(err.Error(), "context_changed_between_variations") {
		t.Fatalf("second variation after rights change = %v, want refusal", err)
	}
	if evaluations.Load() != 3 || fp.execN != 1 {
		t.Fatalf("rights evaluations=%d provider calls=%d; want initial plan, first call and second-variation recheck", evaluations.Load(), fp.execN)
	}
}

func TestProductionRightsAdapterRechecksPersistedContextBeforeProvider(t *testing.T) {
	fp := &fakeProvider{out: []byte("image")}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)
	registry := eng.deps.Registry
	model, ok := registry.ByID(modelID)
	if !ok {
		t.Fatalf("selected model %q missing", modelID)
	}
	model.Rights, _ = json.Marshal(policy.ArtifactRights{
		ArtifactID: modelID, Revision: "fixture-r1", SHA256: strings.Repeat("a", 64),
		LicenseID: "fixture-conditional", Classification: policy.RightsConditional,
		Evidence:          policy.RightsEvidence{URL: "https://example.invalid/license", Revision: "fixture-r1", SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-14"},
		ConditionalGrants: []policy.ConditionalGrant{{ID: "revenue-cap", Uses: []policy.RightsUse{policy.RightsUseLocalInference}, Conditions: []policy.RightsCondition{{ID: "under-cap", Kind: "annual_revenue", Comparator: "lt", Threshold: 10_000_000, Unit: "USD", EntityScope: "operator-group", Interpretation: "synthetic test condition"}}}},
	})
	modelsInRegistry := registry.Models()
	for i := range modelsInRegistry {
		if modelsInRegistry[i].ID == modelID {
			modelsInRegistry[i] = model
		}
	}
	encoded, err := json.Marshal(struct {
		SchemaVersion string                  `json:"schema_version"`
		Models        []models.Model          `json:"models"`
		Blocklist     []models.BlocklistEntry `json:"blocklist"`
	}{registry.SchemaVersion(), modelsInRegistry, registry.Blocklist()})
	if err != nil {
		t.Fatal(err)
	}
	registry, err = models.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	eng.deps.Registry = registry

	contextPath := filepath.Join(t.TempDir(), "rights-context.json")
	writeContext := func(version, revenueStatus string) {
		t.Helper()
		data, err := json.Marshal(map[string]any{"schema_version": "1", "context": map[string]any{
			"id": "test-operator", "version": version, "entity_scope": "operator-group",
			"revenue_status": revenueStatus, "as_of": "2026-11-14", "source": "operator_statement",
		}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(contextPath, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(policy.RightsContextPathEnv, contextPath)
	writeContext("below", "pre_revenue")
	eng.deps.RightsEvaluator = models.AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	plan, err := eng.Plan(context.Background(), PlanRequest{Operation: "text_to_image"})
	if err != nil || plan.ModelID != modelID {
		t.Fatalf("source-permitted plan = %+v, %v; want selected model %s", plan, err, modelID)
	}
	writeContext("above", "at_or_above_threshold")
	_, _, err = eng.runOnce(context.Background(), "text_to_image", model, backends.Selection{Provider: fp, Tier: backends.TierLocalCPU}, t.TempDir(), inputFiles{}, Payload{Params: map[string]string{"prompt": "test"}}, 0, func(float64, string) {})
	if err == nil || !strings.Contains(err.Error(), "revenue_condition_not_satisfied") || fp.execN != 0 {
		t.Fatalf("over-threshold execution error=%v provider calls=%d; want refusal before provider", err, fp.execN)
	}
	writeContext("restored", "pre_revenue")
	_, _, err = eng.runOnce(context.Background(), "text_to_image", model, backends.Selection{Provider: fp, Tier: backends.TierLocalCPU}, t.TempDir(), inputFiles{}, Payload{Params: map[string]string{"prompt": "test"}}, 0, func(float64, string) {})
	if err != nil || fp.execN != 1 {
		t.Fatalf("restored permitted execution error=%v provider calls=%d; want one provider call", err, fp.execN)
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
		Registry:        registry,
		Backends:        backendReg,
		Probe:           capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:           storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir()),
		RightsEvaluator: allowRightsForTest,
		ModelInstalled:  func(string) bool { return false },
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	_, err = eng.Plan(context.Background(), PlanRequest{Operation: "text_to_image"})
	if err == nil {
		t.Fatal("expected ErrModelNotInstalled")
	}
}

func TestPlan_RequireGPURejectsCPUResolution(t *testing.T) {
	eng, _, _ := newTestEngine(t, "naturalize", &fakeProvider{})
	_, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize", RequireGPU: true})
	if !errors.Is(err, ErrGPURequired) {
		t.Fatalf("GPU-required CPU plan error = %v, want ErrGPURequired", err)
	}
}

func TestPlan_ForceCPURejectsRequireGPU(t *testing.T) {
	eng, _, _ := newTestEngine(t, "naturalize", &fakeProvider{})
	_, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize", RequireGPU: true, ForceCPU: true})
	if err == nil || !strings.Contains(err.Error(), "cannot both") {
		t.Fatalf("conflicting execution tier request error = %v, want a clear conflict", err)
	}
}

func TestAdmissionPinsResolvedProviderAndRejectsLegacyPayload(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	model, ok := registry.DefaultFor("text_to_image")
	if !ok {
		t.Fatal("text_to_image default missing")
	}
	native := &mutableProvider{
		fakeProvider: fakeProvider{name: model.Backend, ops: []string{"text_to_image"}},
		available:    true,
	}
	substitute := &fakeProvider{name: "substitute", ops: []string{"text_to_image"}}
	backendRegistry := backends.New()
	if err := backendRegistry.Register(native); err != nil {
		t.Fatal(err)
	}
	if err := backendRegistry.Register(substitute); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(Deps{
		Registry:        registry,
		Backends:        backendRegistry,
		Probe:           capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:           storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir()),
		RightsEvaluator: allowRightsForTest,
		ModelInstalled:  func(string) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}

	plan, err := eng.Plan(context.Background(), PlanRequest{Operation: "text_to_image"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Provider != native.Name() {
		t.Fatalf("plan selected %q, want model backend %q", plan.Provider, native.Name())
	}
	native.available = false

	newPayload, err := json.Marshal(Payload{Operation: "text_to_image", ModelID: model.ID, Backend: plan.Provider})
	if err != nil {
		t.Fatal(err)
	}
	_, err = eng.Admit(context.Background(), internaljobs.Job{ID: "pinned", Operation: "text_to_image", Payload: newPayload})
	if !errors.Is(err, backends.ErrNoneAvailable) || substitute.execN != 0 {
		t.Fatalf("new payload silently changed provider after selection: err=%v substitute executions=%d", err, substitute.execN)
	}

	legacyPayload, err := json.Marshal(Payload{Operation: "text_to_image", ModelID: model.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Admit(context.Background(), internaljobs.Job{ID: "legacy", Operation: "text_to_image", Payload: legacyPayload}); err == nil || !strings.Contains(err.Error(), "no resolved provider") {
		t.Fatalf("legacy payload admission error = %v, want missing-provider retirement boundary", err)
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
		Store:           store,
		RightsEvaluator: allowRightsForTest,
		Enabled: func(context.Context) (map[string]bool, error) {
			return map[string]bool{def.ID: true}, nil
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
	if plan.Provider != fp.Name() {
		t.Fatalf("plan provider = %q, want %q", plan.Provider, fp.Name())
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

func TestPlanCustomModelUsesEnabledOverlayAgainstEffectiveRegistry(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	base, ok := registry.DefaultFor("naturalize")
	if !ok {
		t.Fatal("naturalize default missing")
	}
	custom := base
	custom.ID = "custom-naturalize-enabled"
	custom.Name = "Custom naturalize enabled"
	custom.DefaultFor = nil
	fp := &fakeProvider{name: custom.Backend, ops: []string{"naturalize"}}
	backendReg := backends.New()
	if err := backendReg.Register(fp); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(Deps{
		Registry:        registry,
		Backends:        backendReg,
		Probe:           capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:           storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir()),
		RightsEvaluator: allowRightsForTest,
		CustomModels: func(context.Context) ([]models.Model, error) {
			return []models.Model{custom}, nil
		},
		Enabled: func(context.Context) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		ModelInstalled:        func(id string) bool { return id == custom.ID },
		ModelRuntimeReadiness: func(context.Context, string) (bool, string) { return true, "ready" },
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	plan, err := eng.Plan(context.Background(), PlanRequest{Operation: "naturalize", ModelOverride: custom.ID})
	if err != nil {
		t.Fatalf("plan for enabled custom model: %v", err)
	}
	if plan.ModelID != custom.ID {
		t.Fatalf("plan selected %q, want custom model %q", plan.ModelID, custom.ID)
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
		Registry:        registry,
		Backends:        backendReg,
		Probe:           capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:           store,
		RightsEvaluator: allowRightsForTest,
		ModelInstalled:  func(string) bool { return true },
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
	enabledErrDeps.Enabled = func(context.Context) (map[string]bool, error) { return nil, errors.New("overlay down") }
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
		Registry:        registry,
		Backends:        backends.New(),
		Probe:           capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:           store,
		RightsEvaluator: allowRightsForTest,
		ModelInstalled:  func(string) bool { return true },
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

func TestMaterializeConditioningImagesDerivesPoseMapWithinJob(t *testing.T) {
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	ctx := context.Background()
	if err := store.Put(ctx, "blob/source.png", bytes.NewReader([]byte("synthetic source")), "image/png"); err != nil {
		t.Fatalf("put: %v", err)
	}
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	var gotSource string
	var gotModel string
	var gotWidth, gotHeight int
	eng, err := NewEngine(Deps{
		Registry:        registry,
		Backends:        backends.New(),
		Probe:           capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:           store,
		RightsEvaluator: allowRightsForTest,
		ModelInstalled:  func(string) bool { return true },
		PosePreprocessor: func(_ context.Context, model, source, output string, width, height int) error {
			gotModel = model
			gotSource = source
			gotWidth, gotHeight = width, height
			return os.WriteFile(output, []byte("synthetic OpenPose map"), 0o600)
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "input.png")
	if err := os.WriteFile(input, []byte("already materialized source"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := eng.materializeConditioningImages(ctx, dir, Payload{
		InputKey: "blob/source.png",
		Params:   map[string]string{"width": "640", "height": "480"},
		Adapters: []adapters.ResolvedAdapter{{ID: "controlnet-openpose-sd15", Kind: adapters.KindControlNet, Dir: filepath.Join("/models", "adapters", "controlnet-openpose-sd15"), Preprocessor: adapters.PreprocessorPose, ConditioningImageKey: "blob/source.png"}},
	}, input)
	if err != nil {
		t.Fatalf("materializeConditioningImages: %v", err)
	}
	wantModel := filepath.Join("/models", "adapters", "controlnet-openpose-sd15", "pose", "pose_landmarker_full.task")
	if gotModel != wantModel || gotSource != input || gotWidth != 640 || gotHeight != 480 {
		t.Fatalf("pose model=%q input=%q size=%dx%d; want model=%q and source 640x480", gotModel, gotSource, gotWidth, gotHeight, wantModel)
	}
	if !strings.HasPrefix(resolved[0].ConditioningImageKey, dir) {
		t.Fatalf("pose map not retained in job temp: %q", resolved[0].ConditioningImageKey)
	}
	if data, err := os.ReadFile(resolved[0].ConditioningImageKey); err != nil || string(data) != "synthetic OpenPose map" {
		t.Fatalf("pose output=%q, err=%v", data, err)
	}
}

func TestMaterializeConditioningImagesFailsClosedWithoutPoseRuntime(t *testing.T) {
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	ctx := context.Background()
	if err := store.Put(ctx, "blob/source.png", bytes.NewReader([]byte("synthetic source")), "image/png"); err != nil {
		t.Fatalf("put: %v", err)
	}
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	eng, err := NewEngine(Deps{Registry: registry, Backends: backends.New(), Probe: capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}}, Store: store, RightsEvaluator: allowRightsForTest, ModelInstalled: func(string) bool { return true }})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	_, err = eng.materializeConditioningImages(ctx, t.TempDir(), Payload{InputKey: "blob/source.png", Adapters: []adapters.ResolvedAdapter{{ID: "controlnet-openpose-sd15", Kind: adapters.KindControlNet, Preprocessor: adapters.PreprocessorPose, ConditioningImageKey: "blob/source.png"}}}, "")
	if err == nil || !strings.Contains(err.Error(), "pose preprocessor is unavailable") {
		t.Fatalf("error=%v, want unavailable pose preprocessor", err)
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
