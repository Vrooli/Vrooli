package capacity

import (
	"context"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
)

type fakeProvider struct {
	name    string
	uuidCap bool
}

func (p *fakeProvider) Name() string                   { return p.name }
func (p *fakeProvider) Operations() []string           { return []string{"text_to_image"} }
func (p *fakeProvider) Standalone() bool               { return true }
func (p *fakeProvider) IsCloud() bool                  { return false }
func (p *fakeProvider) Available(context.Context) bool { return true }
func (p *fakeProvider) Execute(context.Context, backends.Request) (backends.Result, error) {
	return backends.Result{}, nil
}
func (p *fakeProvider) SupportsGPUUUID() bool { return p.uuidCap }

func TestGPUForPrefersMostFreeCompatibleDevice(t *testing.T) {
	host := capabilities.Host{GPUs: []capabilities.GPU{
		{Index: 0, UUID: "gpu-0", VRAMBytes: 16 << 30, VRAMUsedBytes: 12 << 30},
		{Index: 1, UUID: "gpu-1", VRAMBytes: 16 << 30, VRAMUsedBytes: 4 << 30},
	}}
	got, err := GPUFor(host, 6<<30, &fakeProvider{name: "diffusers", uuidCap: true})
	if err != nil {
		t.Fatalf("GPUFor: %v", err)
	}
	if got.UUID != "gpu-1" {
		t.Fatalf("selected %q, want the device with the most free VRAM", got.UUID)
	}
}

func TestGPUForRequiresKnownCapacity(t *testing.T) {
	// VRAMBytes == 0 means the probe could not measure the device. Admitting a
	// claim against an unknown device is how two owners end up on one GPU.
	host := capabilities.Host{GPUs: []capabilities.GPU{{Index: 0, UUID: "gpu-0"}}}
	if _, err := GPUFor(host, 1<<30, &fakeProvider{name: "diffusers", uuidCap: true}); err == nil {
		t.Fatal("unmeasured GPU admitted")
	}
}

func TestGPUForWithoutUUIDBindingRequiresSingleDevice(t *testing.T) {
	two := capabilities.Host{GPUs: []capabilities.GPU{
		{Index: 0, UUID: "gpu-0", VRAMBytes: 16 << 30},
		{Index: 1, UUID: "gpu-1", VRAMBytes: 16 << 30},
	}}
	if _, err := GPUFor(two, 6<<30, &fakeProvider{name: "stable-diffusion.cpp"}); err == nil {
		t.Fatal("multi-GPU host admitted a backend that cannot select a device")
	}

	one := capabilities.Host{GPUs: []capabilities.GPU{{Index: 0, UUID: "gpu-0", VRAMBytes: 16 << 30}}}
	got, err := GPUFor(one, 6<<30, &fakeProvider{name: "stable-diffusion.cpp"})
	if err != nil || got.UUID != "gpu-0" {
		t.Fatalf("single-GPU selection = (%q, %v)", got.UUID, err)
	}
}

func TestPriorityContextRoundTrip(t *testing.T) {
	ctx := WithPriority(context.Background(), "interactive")
	if got := capacityPriority(ctx); got != "interactive" {
		t.Fatalf("priority = %q, want interactive", got)
	}
	if got := capacityPriority(context.Background()); got != "service" {
		t.Fatalf("default priority = %q, want service", got)
	}
	for input, want := range map[string]string{"batch": "batch", "": "service", "bogus": "service"} {
		if got := normalizeCapacityPriority(input); got != want {
			t.Errorf("normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

type fakeBroker struct {
	heartbeats int
	releases   int
}

func (b *fakeBroker) Claim(context.Context, string, int64, ClaimOptions) (Lease, error) {
	return Lease{ClaimID: "claim"}, nil
}
func (b *fakeBroker) Heartbeat(_ context.Context, lease Lease) (Lease, error) {
	b.heartbeats++
	return lease, nil
}
func (b *fakeBroker) Release(_ context.Context, _ Lease) error {
	b.releases++
	return nil
}

func TestMemoryLeaseRenewsAndReleasesThroughTheBroker(t *testing.T) {
	broker := &fakeBroker{}
	lease := NewMemoryLease(broker, Lease{ClaimID: "claim"})
	if err := lease.Heartbeat(context.Background()); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatalf("release: %v", err)
	}
	if broker.heartbeats != 1 || broker.releases != 1 {
		t.Fatalf("broker calls = heartbeat %d release %d, want 1/1", broker.heartbeats, broker.releases)
	}
}
