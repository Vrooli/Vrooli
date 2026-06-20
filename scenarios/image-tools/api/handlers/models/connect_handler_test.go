package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"

	internalbackends "image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/testutil/db"

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

func TestDoctorCatalogReportsGreenSeed(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)
	resp, err := h.DoctorCatalog(context.Background(), connect.NewRequest(&modelsv1.DoctorCatalogRequest{}))
	if err != nil {
		t.Fatalf("doctor catalog: %v", err)
	}
	if !resp.Msg.Ok {
		t.Fatalf("doctor should pass after Phase 1 catalog hardening; findings: %+v", resp.Msg.Findings)
	}
	for _, f := range resp.Msg.Findings {
		if f.Severity == modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_ERROR {
			t.Fatalf("doctor returned error finding after Phase 1 catalog hardening: %+v", f)
		}
	}
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

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
