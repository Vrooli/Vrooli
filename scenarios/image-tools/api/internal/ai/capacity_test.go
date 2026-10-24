package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/blobstore"
	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// fakeBroker records the claim it received and returns a scripted verdict.
type fakeBroker struct {
	degrade      bool
	claimErr     error
	heartbeatErr error
	releaseErr   error
	claimedFor   string
	claimedPref  int64
	claimedOpts  CapacityClaimOptions
	released     string
	claimN       int
}

type memoryTestLease struct {
	releaseErr error
	releases   int
	onRelease  func(context.Context)
}

func (l *memoryTestLease) Heartbeat(context.Context) error { return nil }
func (l *memoryTestLease) Release(ctx context.Context) error {
	l.releases++
	if l.onRelease != nil {
		l.onRelease(ctx)
	}
	return l.releaseErr
}

func TestLocalRAMDenialPreventsGPUClaimAndProviderExecution(t *testing.T) {
	provider := &fakeProvider{}
	gpu := &fakeBroker{}
	engine, id := newCapacityTestEngine(t, "text_to_image", provider, gpu)
	engine.deps.Memory = func(context.Context, string, int64) (MemoryLease, error) {
		return nil, errors.New("host memory PSI above policy threshold")
	}
	err := runGPUJob(t, engine, "text_to_image", id)
	if err == nil || !strings.Contains(err.Error(), "local RAM capacity") || provider.execN != 0 || gpu.claimN != 0 {
		t.Fatalf("unadmitted local work ran: error=%v provider=%d GPU claims=%d", err, provider.execN, gpu.claimN)
	}
}

func TestLocalCPUInferenceReservesDeclaredRAMAndReleasesAfterOutput(t *testing.T) {
	for _, failRelease := range []bool{false, true} {
		t.Run(fmt.Sprint(failRelease), func(t *testing.T) {
			provider := &fakeProvider{}
			engine, store, id := newTestEngine(t, "denoise", provider)
			storeInput(t, store, "input/source.png", []byte("source"))
			model, _ := engine.deps.Registry.ByID(id)
			lease := &memoryTestLease{onRelease: func(ctx context.Context) {
				if ctx.Err() != nil || provider.execN != 1 {
					t.Errorf("release before completed work or with cancelled cleanup context")
				}
			}}
			if failRelease {
				lease.releaseErr = errors.New("ledger unavailable during release")
			}
			engine.deps.Memory = func(_ context.Context, owner string, bytes int64) (MemoryLease, error) {
				if !strings.HasPrefix(owner, "image-tools:") || bytes != int64(model.Hardware.MinRAMGB)<<30 {
					t.Fatalf("wrong RAM request: %s %d", owner, bytes)
				}
				return lease, nil
			}
			ref, err := runJob(t, engine, "denoise", Payload{ModelID: id, InputKey: "input/source.png"})
			if ref == "" || lease.releases != 1 {
				t.Fatalf("missing output or cleanup: %q %v releases=%d", ref, err, lease.releases)
			}
			if failRelease {
				if err == nil || !strings.Contains(err.Error(), "release local RAM capacity") {
					t.Fatalf("release failure hidden: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func (b *fakeBroker) Claim(_ context.Context, ownerID string, preferredBytes int64, opts CapacityClaimOptions) (CapacityLease, error) {
	b.claimN++
	b.claimedFor = ownerID
	b.claimedPref = preferredBytes
	b.claimedOpts = opts
	if b.claimErr != nil {
		return CapacityLease{}, b.claimErr
	}
	return CapacityLease{ClaimID: "clm-test", DegradeToCPU: b.degrade}, nil
}

func (b *fakeBroker) Release(_ context.Context, lease CapacityLease) error {
	b.released = lease.ClaimID
	return b.releaseErr
}

func (b *fakeBroker) Heartbeat(_ context.Context, lease CapacityLease) (CapacityLease, error) {
	return lease, b.heartbeatErr
}

// newCapacityTestEngine mirrors newTestEngine but wires a CapacityBroker and a
// GPU-capable fake provider so the GPU-tier arbitration branch is exercised.
func newCapacityTestEngine(t *testing.T, op string, fp *fakeProvider, broker CapacityBroker) (*Engine, string) {
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

	eng, err := NewEngine(Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8, GPUs: []capabilities.GPU{{Index: 0, UUID: "GPU-test", VRAMBytes: 16 << 30}}}},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
		ModelsRoot:     t.TempDir(),
		Capacity:       broker,
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return eng, def.ID
}

func runGPUJob(t *testing.T, eng *Engine, op, modelID string) error {
	t.Helper()
	raw, err := json.Marshal(Payload{Operation: op, ModelID: modelID, GPU: true, CapacityPriority: "service", AllowReclaim: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	runner := eng.BuildRunners()[op]
	_, err = runner(context.Background(), internaljobs.Job{ID: "job-xyz", Operation: op, Payload: raw}, func(int, string) {})
	return err
}

func TestCapacity_GrantKeepsGPU(t *testing.T) {
	fp := &fakeProvider{}
	broker := &fakeBroker{degrade: false}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)

	if err := runGPUJob(t, eng, "text_to_image", modelID); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !fp.lastReq.GPU {
		t.Errorf("grant verdict should leave the job on GPU; got CPU")
	}
	if broker.claimedOpts.GPUUUID != "GPU-test" {
		t.Fatalf("physical UUID not passed to shared broker: %+v", broker.claimedOpts)
	}
	if broker.claimN != 1 {
		t.Errorf("expected exactly one claim, got %d", broker.claimN)
	}
	if !strings.HasPrefix(broker.claimedFor, "image-tools:") {
		t.Errorf("owner id %q should be scoped image-tools:<job>", broker.claimedFor)
	}
	if broker.claimedPref != modelVRAMClaimBytes(models.Model{Hardware: models.Hardware{MinVRAMGB: 2}}) {
		t.Errorf("preferred bytes = %d, want selected model VRAM requirement", broker.claimedPref)
	}
	if broker.released != "clm-test" {
		t.Errorf("claim should be released; released=%q", broker.released)
	}
}

func TestCapacity_ReleaseFailureIsRetainedWithSuccessfulOutput(t *testing.T) {
	fp := &fakeProvider{}
	broker := &fakeBroker{releaseErr: errors.New("ledger unavailable")}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)
	raw, err := json.Marshal(Payload{Operation: "text_to_image", ModelID: modelID, GPU: true})
	if err != nil {
		t.Fatal(err)
	}
	result, err := eng.BuildRunners()["text_to_image"](context.Background(), internaljobs.Job{ID: "release-warning", Payload: raw}, func(int, string) {})
	if err != nil || result.Ref == "" {
		t.Fatalf("cleanup failure discarded successful inference: %+v, %v", result, err)
	}
	if !strings.Contains(result.Meta["capacity_release_warning"], "ledger unavailable") || broker.released != "clm-test" {
		t.Fatalf("cleanup failure lost: %+v, released=%s", result, broker.released)
	}
}

func TestCapacityPhysicalDeviceSelection(t *testing.T) {
	host := capabilities.Host{GPUs: []capabilities.GPU{
		{Index: 0, UUID: "GPU-small", VRAMBytes: 4 << 30, VRAMUsedBytes: 3 << 30},
		{Index: 2, UUID: "GPU-large", VRAMBytes: 16 << 30, VRAMUsedBytes: 2 << 30},
	}}
	g, err := capacityGPU(host, 6<<30, &execProvider{name: "diffusers"})
	if err != nil || g.Index != 2 || g.UUID != "GPU-large" {
		t.Fatalf("wrong physical binding: %+v, %v", g, err)
	}
	if _, err := capacityGPU(host, 6<<30, &execProvider{name: "stable-diffusion.cpp"}); err == nil {
		t.Fatal("unqualified multi-GPU backend silently accepted arbitrary device")
	}
	if _, err := capacityGPU(capabilities.Host{GPUs: []capabilities.GPU{{Index: 0, VRAMBytes: 16 << 30}}}, 6<<30, &execProvider{name: "stable-diffusion.cpp"}); err == nil {
		t.Fatal("unidentified physical GPU accepted")
	}
}

func TestCapacity_DegradeSwitchesToCPU(t *testing.T) {
	fp := &fakeProvider{}
	broker := &fakeBroker{degrade: true}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)

	if err := runGPUJob(t, eng, "text_to_image", modelID); err != nil {
		t.Fatalf("run: %v", err)
	}
	if fp.lastReq.GPU {
		t.Errorf("degrade verdict should switch the job to CPU; still ran on GPU")
	}
	if broker.released != "clm-test" {
		t.Errorf("claim should be released even after degrade; released=%q", broker.released)
	}
}

func TestCapacity_ClaimErrorDoesNotRunUnadmittedGPUWork(t *testing.T) {
	fp := &fakeProvider{}
	broker := &fakeBroker{claimErr: context.DeadlineExceeded}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)

	if err := runGPUJob(t, eng, "text_to_image", modelID); err == nil || !strings.Contains(err.Error(), "admission failed") {
		t.Fatalf("expected capacity admission error: %v", err)
	}
	if fp.lastReq.Operation != "" {
		t.Errorf("unadmitted job executed: %+v", fp.lastReq)
	}
}

func TestCapacity_RenewalLossCancelsConsumer(t *testing.T) {
	b := &fakeBroker{heartbeatErr: errors.New("reservation expired")}
	ctx, stop := keepCapacityAlive(context.Background(), b, CapacityLease{ClaimID: "claim", Generation: 1}, time.Millisecond)
	defer stop()
	select {
	case <-ctx.Done():
		if !strings.Contains(context.Cause(ctx).Error(), "renewal lost") {
			t.Fatal(context.Cause(ctx))
		}
	case <-time.After(time.Second):
		t.Fatal("consumer continued without admission")
	}
}

// The shared client tests own wire-level policy. These checks cover the engine
// adapter's distinction between a valid denial and an admission/cleanup error.
type capacityRunner struct{ fail bool }

func (r capacityRunner) Run(ctx context.Context, _ string, args ...string) ([]byte, error) {
	if r.fail {
		return nil, errors.New("broker unavailable")
	}
	if len(args) > 1 && args[0] == "host" {
		return []byte(`{"gpus":[{"index":0,"uuid":"GPU-test"}]}`), nil
	}
	return []byte(`{"verdict":{"kind":"deny","reason":"contended","warnings":["held by another owner"],"reclaim_bytes":42},"enforce":"advisory"}`), nil
}
func (r capacityRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.Run(ctx, name, args...)
}
func TestSharedCapacityAdapterDenialPermitsCPUButTransportFailureDoesNot(t *testing.T) {
	for _, fail := range []bool{false, true} {
		b := &CLICapacityBroker{Client: vroolicli.New(vroolicli.WithRunner(capacityRunner{fail: fail}))}
		lease, err := b.Claim(context.Background(), "image-tools:job", 1<<30, CapacityClaimOptions{GPUUUID: "GPU-test"})
		if fail {
			if err == nil || lease.DegradeToCPU {
				t.Fatalf("transport error became CPU fallback: %+v %v", lease, err)
			}
		} else {
			if err != nil || !lease.DegradeToCPU || lease.ReclaimBytes != 42 || len(lease.Warnings) != 2 || lease.Warnings[1] != "contended" || lease.ClaimID != "" {
				t.Fatalf("denial metadata lost: %+v %v", lease, err)
			}
			if err := b.Release(context.Background(), lease); err != nil {
				t.Fatal(err)
			}
		}
	}
}
