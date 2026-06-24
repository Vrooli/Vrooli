package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

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
	name     string
	ops      []string
	out      []byte
	lastReq  backends.Request
	execN    int
	failWith error
}

func (p *fakeProvider) Name() string                   { return p.name }
func (p *fakeProvider) Operations() []string           { return p.ops }
func (p *fakeProvider) Standalone() bool               { return true }
func (p *fakeProvider) IsCloud() bool                  { return false }
func (p *fakeProvider) Available(context.Context) bool { return true }

func (p *fakeProvider) Execute(_ context.Context, req backends.Request) (backends.Result, error) {
	p.lastReq = req
	p.execN++
	if p.failWith != nil {
		return backends.Result{}, p.failWith
	}
	out := p.out
	if out == nil {
		out = []byte("fake-output-bytes")
	}
	if err := os.WriteFile(req.Output.LocalPath, out, 0o600); err != nil {
		return backends.Result{}, err
	}
	return backends.Result{OutputRef: req.Output.LocalPath}, nil
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
	return runner(context.Background(), internaljobs.Job{Operation: op, Payload: raw}, func(int, string) {})
}

// storeInput writes bytes under a key in the store (helper for image-input ops).
func storeInput(t *testing.T, store *storage.Store, key string, data []byte) {
	t.Helper()
	if err := store.Put(context.Background(), key, bytes.NewReader(data), "image/png"); err != nil {
		t.Fatalf("store input: %v", err)
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

func TestEngineHelpers(t *testing.T) {
	if Lane("anything") != internaljobs.LaneGPU {
		t.Fatalf("AI lane should be GPU lane")
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
