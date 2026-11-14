package models

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"

	internalbackends "image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internalhosttool "image-tools/internal/hosttool"
	"image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/provision"

	db "github.com/vrooli/api-core/databasetest"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

func newTestHandler(t *testing.T, host capabilities.Host) (*connectHandler, *internalmodels.Registry) {
	t.Helper()
	reg, err := internalmodels.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internalmodels.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	h := NewConnectHandler(Deps{
		Registry: reg,
		Store:    internalmodels.NewStore(d),
		Probe:    capabilities.FakeProbe{Host: host},
		Backends: testBackendRegistry(t),
	})
	return h, reg
}

type captureProvisionJob struct {
	spec jobs.Spec
}

type fakeProvisionToolInspector struct{ states map[string]string }

func (f fakeProvisionToolInspector) Inspect(_ context.Context, tool string) (*internalhosttool.Status, error) {
	state := f.states[tool]
	if state == "" {
		state = "would_install"
	}
	return &internalhosttool.Status{Name: tool, ExecutionState: state, SupportClass: "supported"}, nil
}

func (c *captureProvisionJob) Submit(_ context.Context, spec jobs.Spec) (jobs.Job, error) {
	c.spec = spec
	return jobs.Job{ID: "provision-plan-job"}, nil
}

func TestProvisionDefaultUsesHostFitCoverageAndPersistsExclusions(t *testing.T) {
	const gib = uint64(1 << 30)
	h, reg := newTestHandler(t, capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8, TotalMemoryBytes: 24 * gib, AvailableMemoryBytes: 18 * gib, GPUs: []capabilities.GPU{{VRAMBytes: 16 * gib}}})
	h.deps.Installer = &internalmodels.Installer{Root: t.TempDir(), Reg: reg}
	backends := internalbackends.New()
	for _, model := range reg.Models() {
		if model.Backend == internalmodels.BackendOpenRouter || len(model.Operations) == 0 {
			continue
		}
		if err := backends.Register(testProvider{name: model.Backend, ops: model.Operations, available: true}); err != nil {
			t.Fatal(err)
		}
	}
	h.deps.Backends = backends
	h.deps.Ensurer = fakeProvisionToolInspector{states: map[string]string{"llama-cpp": "manual_action_required"}}
	capture := &captureProvisionJob{}
	h.deps.Jobs = capture
	resp, err := h.ProvisionModels(context.Background(), connect.NewRequest(&modelsv1.ProvisionModelsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.ModelIds) == 0 {
		t.Fatal("host-fit planner selected no local models")
	}
	for _, id := range resp.Msg.ModelIds {
		if id == "qwen-image-edit" {
			t.Fatal("planner selected the 57.7 GB Qwen model that exceeds host RAM/VRAM")
		}
	}
	var request provision.Request
	if err := json.Unmarshal(capture.spec.Payload, &request); err != nil {
		t.Fatal(err)
	}
	if len(request.Plan.CoveredOperations) < 2 {
		t.Fatalf("fixture backend registry should allow multi-operation coverage: %+v", request.Plan)
	}
	t.Logf("fixture host-fit plan: selected=%d/%d catalog models operations=%d uncovered=%d exclusions=%d", len(request.Plan.SelectedModelIDs), len(reg.Models()), len(request.Plan.CoveredOperations), len(request.Plan.UncoveredOperations), len(request.Plan.ExcludedModels))
	if request.Plan.Mode != provision.PlanHostFit || len(request.Plan.SelectedModelIDs) != len(resp.Msg.ModelIds) {
		t.Fatalf("host-fit scope was not persisted: %+v", request.Plan)
	}
	allCovered := append(append([]string(nil), request.Plan.CoveredOperations...), func() []string {
		ops := make([]string, 0, len(request.Plan.UncoveredOperations))
		for _, gap := range request.Plan.UncoveredOperations {
			ops = append(ops, gap.Operation)
		}
		return ops
	}()...)
	sort.Strings(allCovered)
	wantScope := provision.EnabledOperations(reg.Models(), reg.SortedOperations(), func(id string) bool {
		m, ok := reg.ByID(id)
		return ok && internalmodels.EffectiveEnabled(m, nil)
	})
	if !reflect.DeepEqual(allCovered, wantScope) {
		t.Fatalf("host-fit plan omitted enabled capability scope: got %v want %v", allCovered, wantScope)
	}
	foundQwen := false
	for _, excluded := range request.Plan.ExcludedModels {
		if excluded.ModelID == "qwen-image-edit" {
			foundQwen = true
			if excluded.Reason != "requires_32_gib_ram_host_has_18_gib_available" {
				t.Fatalf("unexpected Qwen exclusion: %+v", excluded)
			}
		}
	}
	if !foundQwen {
		t.Fatalf("oversized enabled model omission was not retained: %+v", request.Plan.ExcludedModels)
	}
	foundManualBackend := false
	for _, gap := range request.Plan.CandidateGaps {
		if gap.ModelID == "moondream2" && gap.Operation == "caption" {
			foundManualBackend = gap.Reason == "host_tool_manual:llama-cpp"
		}
	}
	if !foundManualBackend {
		t.Fatalf("manual host-tool backend was not excluded with a precise gap: %+v", request.Plan.CandidateGaps)
	}
}

func TestProvisionExplicitModelsRemainExact(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	h.deps.Installer = &internalmodels.Installer{Root: t.TempDir(), Reg: reg}
	capture := &captureProvisionJob{}
	h.deps.Jobs = capture
	resp, err := h.ProvisionModels(context.Background(), connect.NewRequest(&modelsv1.ProvisionModelsRequest{ModelIds: []string{"mask-fill"}}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.ModelIds) != 1 || resp.Msg.ModelIds[0] != "mask-fill" {
		t.Fatalf("explicit ids changed: %v", resp.Msg.ModelIds)
	}
	var request provision.Request
	if err := json.Unmarshal(capture.spec.Payload, &request); err != nil {
		t.Fatal(err)
	}
	if request.Plan.Mode != provision.PlanExplicit || len(request.ModelIDs) != 1 || request.ModelIDs[0] != "mask-fill" {
		t.Fatalf("explicit request was expanded or re-planned: %+v", request)
	}
}

type testProvider struct {
	name      string
	ops       []string
	available bool
}

func (p testProvider) Name() string         { return p.name }
func (p testProvider) Operations() []string { return append([]string(nil), p.ops...) }
func (p testProvider) Standalone() bool     { return true }
func (p testProvider) IsCloud() bool        { return false }
func (p testProvider) GPUCapable() bool     { return false }
func (p testProvider) Available(context.Context) bool {
	return p.available
}

func (p testProvider) Availability(context.Context) internalbackends.Availability {
	return internalbackends.Availability{
		Available: p.available,
		Detail:    "test backend readiness",
		Provision: "test provision",
	}
}

func (p testProvider) Execute(context.Context, internalbackends.Request) (internalbackends.Result, error) {
	return internalbackends.Result{}, nil
}

func testBackendRegistry(t *testing.T) *internalbackends.Registry {
	t.Helper()
	reg := internalbackends.New()
	if err := reg.Register(testProvider{name: "builtin", ops: []string{"naturalize"}, available: true}); err != nil {
		t.Fatalf("register backend: %v", err)
	}
	if err := reg.Register(testProvider{name: "python-sidecar", ops: []string{"colorize"}, available: true}); err != nil {
		t.Fatalf("register backend: %v", err)
	}
	if err := reg.Register(testProvider{name: "python-sidecar", ops: []string{"face_restore", "old_photo_restore"}, available: false}); err != nil {
		t.Fatalf("register backend: %v", err)
	}
	return reg
}

// cpuOnlyHost is a deterministic GPU-less host so selection takes the
// CPU-capable default path.
var cpuOnlyHost = capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8, TotalMemoryBytes: 16 << 30}

func TestListModelsAllAndFiltered(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	ctx := context.Background()

	all, err := h.ListModels(ctx, connect.NewRequest(&modelsv1.ListModelsRequest{}))
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all.Msg.Models) != len(reg.Models()) {
		t.Fatalf("list all returned %d, want %d", len(all.Msg.Models), len(reg.Models()))
	}

	op := reg.SortedOperations()[0]
	filtered, err := h.ListModels(ctx, connect.NewRequest(&modelsv1.ListModelsRequest{Operation: op}))
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if len(filtered.Msg.Models) == 0 {
		t.Fatalf("expected models for op %q", op)
	}
	for _, m := range filtered.Msg.Models {
		if !containsStr(m.Operations, op) {
			t.Fatalf("model %q does not serve filtered op %q", m.Id, op)
		}
	}
}

func TestListModelsUnknownOperationIsInvalidArgument(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	_, err := h.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{Operation: "not-a-real-op"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument (err=%v)", connect.CodeOf(err), err)
	}
}

func TestGetModelNotFound(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	_, err := h.GetModel(context.Background(), connect.NewRequest(&modelsv1.GetModelRequest{Id: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", connect.CodeOf(err))
	}
}

func TestSetModelEnabledPersistsAndReflectsInListGet(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	ctx := context.Background()

	// Find a seeded-enabled model and disable it.
	var target string
	for _, m := range reg.Models() {
		if m.Enabled {
			target = m.ID
			break
		}
	}
	if target == "" {
		t.Fatal("no enabled seed model")
	}

	resp, err := h.SetModelEnabled(ctx, connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: target, Enabled: false}))
	if err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	if resp.Msg.Model.Enabled {
		t.Fatal("response should report disabled")
	}

	got, err := h.GetModel(ctx, connect.NewRequest(&modelsv1.GetModelRequest{Id: target}))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Msg.Model.Enabled {
		t.Fatal("overlay disable did not persist into GetModel")
	}
}

func TestSetModelEnabledUnknownIsNotFound(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	_, err := h.SetModelEnabled(context.Background(), connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: "nope", Enabled: true}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", connect.CodeOf(err))
	}
}

func TestSetModelEnabledCannotEnableConditionalRightsModel(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	var conditional internalmodels.Model
	for _, model := range reg.Models() {
		if model.CapabilityLabels.CommercialUse == internalmodels.CommercialUseConditional {
			conditional = model
			break
		}
	}
	if conditional.ID == "" {
		t.Fatal("seed should include a conditional-rights model for this regression")
	}
	_, err := h.SetModelEnabled(context.Background(), connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: conditional.ID, Enabled: true}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("enable conditional model error = %v, want FailedPrecondition", err)
	}
}

func TestSelectModelCPUDefault(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	op := reg.SortedOperations()[0]
	resp, err := h.SelectModel(context.Background(), connect.NewRequest(&modelsv1.SelectModelRequest{Operation: op}))
	if err != nil {
		t.Fatalf("select %q: %v", op, err)
	}
	if resp.Msg.Model == nil || resp.Msg.Model.Id == "" {
		t.Fatal("select returned no model")
	}
	if resp.Msg.GpuViable {
		t.Fatal("GPU-less host must not report gpu_viable")
	}
	if resp.Msg.Reason == "" {
		t.Fatal("select must surface a reason")
	}
}

func TestSelectModelUnknownOperationInvalidArgument(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	_, err := h.SelectModel(context.Background(), connect.NewRequest(&modelsv1.SelectModelRequest{Operation: "nope"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestSelectModelDisabledOverlayFailsPrecondition(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	ctx := context.Background()
	op := reg.SortedOperations()[0]

	// Disable every model serving the op; selection must then fail precondition.
	for _, m := range reg.ForOperation(op) {
		if _, err := h.SetModelEnabled(ctx, connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: m.ID, Enabled: false})); err != nil {
			t.Fatalf("disable %q: %v", m.ID, err)
		}
	}
	_, err := h.SelectModel(ctx, connect.NewRequest(&modelsv1.SelectModelRequest{Operation: op}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("got code %v, want FailedPrecondition (err=%v)", connect.CodeOf(err), err)
	}
}

func TestListOperationsAndBlocklist(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	ctx := context.Background()

	ops, err := h.ListOperations(ctx, connect.NewRequest(&modelsv1.ListOperationsRequest{}))
	if err != nil {
		t.Fatalf("list operations: %v", err)
	}
	if len(ops.Msg.Operations) != len(reg.Operations()) {
		t.Fatalf("operations count %d, want %d", len(ops.Msg.Operations), len(reg.Operations()))
	}

	bl, err := h.ListBlocklist(ctx, connect.NewRequest(&modelsv1.ListBlocklistRequest{}))
	if err != nil {
		t.Fatalf("list blocklist: %v", err)
	}
	if len(bl.Msg.Entries) != len(reg.Blocklist()) {
		t.Fatalf("blocklist count %d, want %d", len(bl.Msg.Entries), len(reg.Blocklist()))
	}
}

func TestDoctorCatalogGatesIncompleteEnabledCapabilityCoverage(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	h.deps.Installer = &internalmodels.Installer{Root: t.TempDir(), Reg: h.deps.Registry}
	resp, err := h.DoctorCatalog(context.Background(), connect.NewRequest(&modelsv1.DoctorCatalogRequest{}))
	if err != nil {
		t.Fatalf("doctor catalog: %v", err)
	}
	if resp.Msg.Ok {
		t.Fatalf("doctor reported a ready catalog despite missing host-fit enabled capabilities: %+v", resp.Msg.Findings)
	}
	var gaps int
	for _, f := range resp.Msg.Findings {
		if f.Code == "enabled_capability_not_provisionable" {
			if f.Operation == "" || f.Message == "" || f.Severity != modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_ERROR {
				t.Fatalf("doctor returned an incomplete capability gap: %+v", f)
			}
			gaps++
		}
	}
	if gaps == 0 {
		t.Fatalf("doctor did not expose the unsupported enabled capability set: %+v", resp.Msg.Findings)
	}
}

func TestDoctorCatalogCannotPassWithoutLifecycleOrHostEvidence(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	request := connect.NewRequest(&modelsv1.DoctorCatalogRequest{})
	withoutInstaller, err := h.DoctorCatalog(context.Background(), request)
	if err != nil {
		t.Fatalf("doctor without installer: %v", err)
	}
	if withoutInstaller.Msg.Ok || !hasDoctorFinding(withoutInstaller.Msg.Findings, "model_lifecycle_unavailable") {
		t.Fatalf("doctor passed without runtime lifecycle evidence: %+v", withoutInstaller.Msg)
	}
	h.deps.Installer = &internalmodels.Installer{Root: t.TempDir(), Reg: reg}
	h.deps.Probe = nil
	withoutHost, err := h.DoctorCatalog(context.Background(), request)
	if err != nil {
		t.Fatalf("doctor without host inventory: %v", err)
	}
	if withoutHost.Msg.Ok || !hasDoctorFinding(withoutHost.Msg.Findings, "host_inventory_unavailable") {
		t.Fatalf("doctor passed without host inventory: %+v", withoutHost.Msg)
	}
}

func TestDefaultProvisionAndDoctorRejectEmptyEnabledCapabilityScope(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	for _, model := range reg.Models() {
		if err := h.deps.Store.SetEnabled(context.Background(), model.ID, false); err != nil {
			t.Fatalf("disable %s: %v", model.ID, err)
		}
	}
	h.deps.Installer = &internalmodels.Installer{Root: t.TempDir(), Reg: reg}
	capture := &captureProvisionJob{}
	h.deps.Jobs = capture
	_, err := h.ProvisionModels(context.Background(), connect.NewRequest(&modelsv1.ProvisionModelsRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("empty default provision scope error = %v, want FailedPrecondition", err)
	}
	if capture.spec.Operation != "" {
		t.Fatalf("empty capability scope must not submit a success-shaped job: %+v", capture.spec)
	}
	resp, err := h.DoctorCatalog(context.Background(), connect.NewRequest(&modelsv1.DoctorCatalogRequest{}))
	if err != nil {
		t.Fatalf("doctor empty scope: %v", err)
	}
	if resp.Msg.Ok || !hasDoctorFinding(resp.Msg.Findings, "enabled_capability_scope_empty") {
		t.Fatalf("doctor passed with no enabled capability scope: %+v", resp.Msg)
	}
}

func hasDoctorFinding(findings []*modelsv1.CatalogFinding, code string) bool {
	for _, finding := range findings {
		if finding.GetCode() == code && finding.GetSeverity() == modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_ERROR {
			return true
		}
	}
	return false
}

func TestDoctorBackendsReportsCatalogDeclaredBackendReadiness(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	resp, err := h.DoctorBackends(context.Background(), connect.NewRequest(&modelsv1.DoctorBackendsRequest{}))
	if err != nil {
		t.Fatalf("doctor backends: %v", err)
	}
	if resp.Msg == nil || resp.Msg.Ok {
		t.Fatalf("doctor backends should flag enabled catalog backends missing runtime providers: %+v", resp.Msg)
	}
	var sawBuiltin, sawPythonSidecar, sawLlama bool
	for _, b := range resp.Msg.Backends {
		switch b.Name {
		case "builtin":
			sawBuiltin = true
			if !b.Available {
				t.Fatalf("registered builtin backend should remain available: %+v", b)
			}
		case "python-sidecar":
			sawPythonSidecar = true
			ops := containsStr(b.Operations, "colorize") || containsStr(b.Operations, "face_restore")
			if !ops || b.Detail != "test backend readiness" {
				t.Fatalf("python-sidecar row should come from registered runtime providers: %+v", b)
			}
		case "llama.cpp":
			sawLlama = true
			if b.Available || !containsStr(b.Operations, "caption") {
				t.Fatalf("llama.cpp row should be an unavailable catalog-declared gap: %+v", b)
			}
		}
	}
	if !sawBuiltin || !sawPythonSidecar || !sawLlama {
		t.Fatalf("missing expected backend rows (builtin=%t python-sidecar=%t llama=%t): %+v", sawBuiltin, sawPythonSidecar, sawLlama, resp.Msg.Backends)
	}
}

func TestDoctorBackendsOverlaysHostToolReadiness(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	resp, err := h.DoctorBackends(context.Background(), connect.NewRequest(&modelsv1.DoctorBackendsRequest{}))
	if err != nil {
		t.Fatalf("doctor backends: %v", err)
	}
	var sawLlamaTool bool
	for _, b := range resp.Msg.Backends {
		if b.Name == "llama.cpp" {
			sawLlamaTool = true
			if b.HostTool != "llama-cpp" {
				t.Fatalf("llama.cpp host_tool = %q, want llama-cpp", b.HostTool)
			}
			if b.HostToolReady {
				t.Fatalf("unavailable backend should report host_tool_ready=false")
			}
			if b.Remediation != "vrooli host install llama-cpp" {
				t.Fatalf("remediation = %q, want `vrooli host install llama-cpp`", b.Remediation)
			}
		}
	}
	if !sawLlamaTool {
		t.Fatalf("expected a llama.cpp backend row with host-tool overlay")
	}
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// SelectModel is a preview of the run a caller is about to submit, so it has to
// be given the run's routing policy. Without it the preview answered with the
// local default for a caller routing to the BYOK rung — and a caller that sizes
// its request from the previewed model then sizes it from a model that was
// never going to serve it. That is not hypothetical: a Backdrop Studio
// frontier-tier render reached OpenRouter having been sized against sd-1.5's
// 768px cap, which OpenRouter does not have.
func TestSelectModelHonoursTheCallersRoutingPolicy(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	ctx := context.Background()
	const op = "text_to_image"

	local, err := h.SelectModel(ctx, connect.NewRequest(&modelsv1.SelectModelRequest{Operation: op}))
	if err != nil {
		t.Fatalf("select local: %v", err)
	}
	byok, err := h.SelectModel(ctx, connect.NewRequest(&modelsv1.SelectModelRequest{
		Operation: op, QualityPolicy: "quality", AllowByok: true,
	}))
	if err != nil {
		t.Fatalf("select byok: %v", err)
	}
	if local.Msg.GetModel().GetId() == byok.Msg.GetModel().GetId() {
		t.Fatalf("both policies previewed %q; the routing policy is not reaching the selector",
			local.Msg.GetModel().GetId())
	}
	if got := byok.Msg.GetModel().GetBackend(); got != "openrouter" {
		t.Fatalf("BYOK-permitted selection chose backend %q, want the cloud rung", got)
	}
	// And the geometry travels with the choice, which is the fact the caller
	// actually needs: a cloud entry declares none, because the provider owns it.
	if g := byok.Msg.GetModel().GetGeometry(); g.GetNativeWidth() != 0 || g.GetNativeHeight() != 0 {
		t.Fatalf("cloud model reported a native geometry %v; the provider owns it", g)
	}
	if g := local.Msg.GetModel().GetGeometry(); g.GetNativeWidth() == 0 || g.GetSizeQuantum() == 0 {
		t.Fatalf("local model reported no geometry %v; the caller has nothing to size from", g)
	}
}
