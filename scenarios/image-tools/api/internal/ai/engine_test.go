package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
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
	if err := RegisterProviders(backendReg, func(string) (string, error) { return "/usr/bin/x", nil }, nil); err != nil {
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
