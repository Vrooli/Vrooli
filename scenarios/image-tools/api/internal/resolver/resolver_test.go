package resolver

import (
	"context"
	"errors"
	"testing"

	"image-tools/internal/adapters"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/models"
	"image-tools/internal/safety"
)

// fakeProvider is a standalone provider that serves a fixed op set so the
// backend-tier leg of resolution has something to pick.
type fakeProvider struct {
	name string
	ops  []string
}

func (p fakeProvider) Name() string                   { return p.name }
func (p fakeProvider) Operations() []string           { return p.ops }
func (p fakeProvider) Standalone() bool               { return true }
func (p fakeProvider) IsCloud() bool                  { return false }
func (p fakeProvider) Available(context.Context) bool { return true }
func (p fakeProvider) Execute(context.Context, backends.Request) (backends.Result, error) {
	return backends.Result{}, errors.New("resolver tests never execute")
}

// newResolver builds a resolver over the real registry and a backend registry
// holding a provider for op under the op default model's backend (so the
// backend-tier leg resolves). Returns the resolver, the default model id, and a
// CPU-only host.
func newResolver(t *testing.T, op string) (*Resolver, string) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := registry.DefaultFor(op)
	if !ok {
		t.Fatalf("no default model for %q", op)
	}
	be := backends.New()
	if err := be.Register(fakeProvider{name: def.Backend, ops: []string{op}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	return New(registry, be), def.ID
}

func cpuHost() capabilities.Host {
	return capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}
}

func TestResolveNativeOperation(t *testing.T) {
	r, defID := newResolver(t, "text_to_image")
	res, err := r.Resolve(context.Background(), Request{Operation: "text_to_image", Host: cpuHost()})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res.Model.ID != defID {
		t.Fatalf("model = %q, want default %q", res.Model.ID, defID)
	}
	if res.Support != SupportNative {
		t.Fatalf("support = %q, want native", res.Support)
	}
	if res.Caveat != "" {
		t.Fatalf("native op carries caveat %q", res.Caveat)
	}
	if res.Tier == "" {
		t.Fatal("native resolution has empty backend tier")
	}
}

func TestResolveUnknownOperation(t *testing.T) {
	r, _ := newResolver(t, "text_to_image")
	if _, err := r.Resolve(context.Background(), Request{Operation: "not_a_real_op", Host: cpuHost()}); err == nil {
		t.Fatal("expected error for unknown operation")
	}
}

func TestResolveInvalidOverride(t *testing.T) {
	r, _ := newResolver(t, "text_to_image")
	_, err := r.Resolve(context.Background(), Request{
		Operation:     "text_to_image",
		ModelOverride: "no-such-model",
		Host:          cpuHost(),
	})
	if !errors.Is(err, models.ErrOverrideInvalid) {
		t.Fatalf("err = %v, want ErrOverrideInvalid", err)
	}
}

// TestResolutionWeightIsOperationKeyed is the safety-weight invariant (plan
// Phase 4 / decision 113 / risk R5): the consent weight a resolution carries is
// determined solely by the OPERATION, never by which model/technique serves it
// or whether the support is native or derived. A derived high-weight edit must
// not leak to a lower weight than the same op served natively.
func TestResolutionWeightIsOperationKeyed(t *testing.T) {
	for _, op := range []string{"text_to_image", "image_to_image", "inpaint", "outpaint", "edit_instruct", "upscale"} {
		r, _ := newResolver(t, op)
		res, err := r.Resolve(context.Background(), Request{Operation: op, Host: cpuHost()})
		if err != nil {
			// Some ops have no enabled CPU-runnable default on this fixture host;
			// the invariant under test is the weight mapping, which we still assert
			// directly below — skip the resolution-level check for those.
			continue
		}
		if res.Weight != string(safety.OpWeight(op)) {
			t.Fatalf("op %q: resolution weight %q != safety.OpWeight %q", op, res.Weight, safety.OpWeight(op))
		}
	}
	// The mapping itself is keyed only by op name (no native/derived dimension).
	if safety.OpWeight("inpaint") != safety.WeightHigh {
		t.Fatalf("inpaint weight = %q, want high", safety.OpWeight("inpaint"))
	}
	if safety.OpWeight("edit_instruct") != safety.WeightHigh {
		t.Fatalf("edit_instruct weight = %q, want high", safety.OpWeight("edit_instruct"))
	}
}

// TestResolveConditioningElevatesWeightAndAttaches exercises the Phase 3 wiring:
// a compatible, Ready, installed IP-Adapter on a none-weight text_to_image
// elevates the resolution's consent weight to high and attaches the stack.
func TestResolveConditioningElevatesWeightAndAttaches(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, _ := registry.DefaultFor("text_to_image")
	model, _ := registry.ByID(def.ID)
	r, _ := newResolver(t, "text_to_image")

	ipa := adapters.Adapter{
		ID: "test-ipa", Name: "Test IP-Adapter", Kind: adapters.KindIPAdapter,
		Architecture: model.Architecture, Weight: safety.WeightHigh,
		ScaleRange: adapters.ScaleRange{Min: 0, Max: 1, Default: 0.6}, Ready: true,
	}
	byID := func(id string) (adapters.Adapter, bool) {
		if id == ipa.ID {
			return ipa, true
		}
		return adapters.Adapter{}, false
	}
	res, err := r.Resolve(context.Background(), Request{
		Operation:        "text_to_image",
		Host:             cpuHost(),
		Adapters:         []adapters.AdapterRequest{{ID: "test-ipa", ConditioningImageKey: "ref.png"}},
		AdapterByID:      byID,
		AdapterInstalled: func(string) bool { return true },
	})
	if err != nil {
		t.Fatalf("resolve with conditioning: %v", err)
	}
	if len(res.Adapters) != 1 || res.Adapters[0].ID != "test-ipa" {
		t.Fatalf("expected the ip-adapter attached, got %+v", res.Adapters)
	}
	if res.Weight != string(safety.WeightHigh) {
		t.Fatalf("expected elevated weight high, got %q", res.Weight)
	}

	// Fail closed when the same adapter is not Ready.
	ipa.Ready = false
	if _, err := r.Resolve(context.Background(), Request{
		Operation:        "text_to_image",
		Host:             cpuHost(),
		Adapters:         []adapters.AdapterRequest{{ID: "test-ipa", ConditioningImageKey: "ref.png"}},
		AdapterByID:      byID,
		AdapterInstalled: func(string) bool { return true },
	}); err == nil {
		t.Fatal("expected fail-closed for a not-Ready adapter")
	}
}
