package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/capacity"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/blobstore"
	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// fakeBroker records the claim it received and returns a scripted verdict.
type fakeBroker struct {
	degrade      bool
	reclaimBytes int64
	claimErr     error
	heartbeatErr error
	releaseErr   error
	claimedFor   string
	claimedPref  int64
	claimedOpts  capacity.ClaimOptions
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
	engine.deps.Memory = func(context.Context, string, int64) (capacity.MemoryLease, error) {
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
			engine.deps.Memory = func(_ context.Context, owner string, bytes int64) (capacity.MemoryLease, error) {
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

func (b *fakeBroker) Claim(_ context.Context, ownerID string, preferredBytes int64, opts capacity.ClaimOptions) (capacity.Lease, error) {
	b.claimN++
	b.claimedFor = ownerID
	b.claimedPref = preferredBytes
	b.claimedOpts = opts
	if b.claimErr != nil {
		return capacity.Lease{}, b.claimErr
	}
	return capacity.Lease{ClaimID: "clm-test", DegradeToCPU: b.degrade, ReclaimBytes: b.reclaimBytes}, nil
}

func (b *fakeBroker) Release(_ context.Context, lease capacity.Lease) error {
	b.released = lease.ClaimID
	return b.releaseErr
}

func (b *fakeBroker) Heartbeat(_ context.Context, lease capacity.Lease) (capacity.Lease, error) {
	return lease, b.heartbeatErr
}

// newCapacityTestEngine mirrors newTestEngine but wires a capacity.Broker and a
// GPU-capable fake provider so the GPU-tier arbitration branch is exercised.
func newCapacityTestEngine(t *testing.T, op string, fp *fakeProvider, broker capacity.Broker) (*Engine, string) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := registry.DefaultFor(op)
	if !ok {
		t.Fatalf("no default model for %q", op)
	}
	return newCapacityTestEngineForModel(t, op, def, fp, broker)
}

func newCapacityTestEngineForModel(t *testing.T, op string, def models.Model, fp *fakeProvider, broker capacity.Broker) (*Engine, string) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	fp.name = def.Backend
	fp.ops = []string{op}

	backendReg := backends.New()
	if err := backendReg.Register(fp); err != nil {
		t.Fatalf("register fake provider: %v", err)
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())

	eng, err := NewEngine(Deps{
		Registry:        registry,
		Backends:        backendReg,
		Probe:           capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8, GPUs: []capabilities.GPU{{Index: 0, UUID: "GPU-test", VRAMBytes: 16 << 30}}}},
		Store:           store,
		RightsEvaluator: allowRightsForTest,
		ModelInstalled:  func(string) bool { return true },
		ModelsRoot:      t.TempDir(),
		Capacity:        broker,
		Disk:            &fakeDiskAdmission{},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return eng, def.ID
}

func runGPUJob(t *testing.T, eng *Engine, op, modelID string) error {
	t.Helper()
	raw, err := marshalPinnedPayload(t, eng, op, Payload{Operation: op, ModelID: modelID, GPU: true, CapacityPriority: "service", AllowReclaim: true})
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
	if broker.claimedPref != modelVRAMClaimBytes(models.Model{Hardware: models.Hardware{MinVRAMGB: 6}}) {
		t.Errorf("preferred bytes = %d, want selected model VRAM requirement", broker.claimedPref)
	}
	if broker.released != "clm-test" {
		t.Errorf("claim should be released; released=%q", broker.released)
	}
}

func TestCapacity_ReleaseFailureFailsWhileRetainingDirectEngineOutput(t *testing.T) {
	fp := &fakeProvider{}
	broker := &fakeBroker{releaseErr: errors.New("ledger unavailable")}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)
	raw, err := marshalPinnedPayload(t, eng, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID, GPU: true})
	if err != nil {
		t.Fatal(err)
	}
	result, err := eng.BuildRunners()["text_to_image"](context.Background(), internaljobs.Job{ID: "release-warning", Payload: raw}, func(int, string) {})
	if err == nil || !strings.Contains(err.Error(), "ledger unavailable") || result.Ref == "" {
		t.Fatalf("cleanup failure must fail explicitly while preserving direct-engine output: %+v, %v", result, err)
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
	g, err := capacity.GPUFor(host, 6<<30, &execProvider{backendProcess: backendProcess{name: "diffusers"}})
	if err != nil || g.Index != 2 || g.UUID != "GPU-large" {
		t.Fatalf("wrong physical binding: %+v, %v", g, err)
	}
	if _, err := capacity.GPUFor(host, 6<<30, &execProvider{backendProcess: backendProcess{name: "stable-diffusion.cpp"}}); err == nil {
		t.Fatal("unqualified multi-GPU backend silently accepted arbitrary device")
	}
	if _, err := capacity.GPUFor(capabilities.Host{GPUs: []capabilities.GPU{{Index: 0, VRAMBytes: 16 << 30}}}, 6<<30, &execProvider{backendProcess: backendProcess{name: "stable-diffusion.cpp"}}); err == nil {
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

func TestCapacity_RequireGPURejectsDegradationAndReleasesClaim(t *testing.T) {
	fp := &fakeProvider{}
	broker := &fakeBroker{degrade: true}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)
	raw, err := marshalPinnedPayload(t, eng, "text_to_image", Payload{
		Operation: "text_to_image", ModelID: modelID, GPU: true, RequireGPU: true,
		CapacityPriority: "interactive", AllowReclaim: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := eng.BuildRunners()["text_to_image"]
	_, err = runner(context.Background(), internaljobs.Job{ID: "require-gpu-degrade", Operation: "text_to_image", Payload: raw}, func(int, string) {})
	if !errors.Is(err, internaljobs.ErrCapacityBusy) {
		t.Fatalf("required GPU degradation error = %v, want capacity busy", err)
	}
	if fp.execN != 0 {
		t.Fatalf("provider executed %d times after the broker degraded the GPU claim", fp.execN)
	}
	if broker.released != "clm-test" {
		t.Fatalf("degraded GPU claim not released: %q", broker.released)
	}
}

func TestAdmission_RequireGPUReleasesCPUAndRAMOnBrokerDegradation(t *testing.T) {
	provider := &fakeProvider{}
	broker := &fakeBroker{degrade: true}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", provider, broker)
	cpuLease, ramLease := &memoryTestLease{}, &memoryTestLease{}
	eng.deps.CPU = func(context.Context, string, int64) (capacity.MemoryLease, error) { return cpuLease, nil }
	eng.deps.Memory = func(context.Context, string, int64) (capacity.MemoryLease, error) { return ramLease, nil }

	raw, err := marshalPinnedPayload(t, eng, "text_to_image", Payload{
		Operation: "text_to_image", ModelID: modelID, GPU: true, RequireGPU: true,
		CapacityPriority: "interactive", AllowReclaim: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := eng.Admit(context.Background(), internaljobs.Job{
		ID: "require-gpu-releases-partial-claims", Operation: "text_to_image", Payload: raw,
	})
	if !errors.Is(err, internaljobs.ErrCapacityBusy) {
		t.Fatalf("required GPU degradation error = %v, want capacity busy", err)
	}
	if admission.Release == nil {
		t.Fatal("failed admission did not return cleanup for acquired CPU/RAM claims")
	}
	if err := admission.Release(context.Background()); err != nil {
		t.Fatalf("release partial admission: %v", err)
	}
	if cpuLease.releases != 1 || ramLease.releases != 1 || broker.released != "clm-test" {
		t.Fatalf("partial claims not fully released: CPU=%d RAM=%d GPU=%q", cpuLease.releases, ramLease.releases, broker.released)
	}
	if provider.execN != 0 {
		t.Fatalf("provider executed %d times after required GPU admission failed", provider.execN)
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
	ctx, stop := capacity.KeepAlive(context.Background(), b, capacity.Lease{ClaimID: "claim", Generation: 1}, time.Millisecond)
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
		b := &capacity.CLIBroker{Client: vroolicli.New(vroolicli.WithRunner(capacityRunner{fail: fail}))}
		lease, err := b.Claim(context.Background(), "image-tools:job", 1<<30, capacity.ClaimOptions{GPUUUID: "GPU-test"})
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

// The CPU budget charged to the shared broker is only honest if the backend
// runs with that many threads; the provider must receive the admitted count.
func TestLocalCPUInferenceBindsProviderToAdmittedCPUBudget(t *testing.T) {
	provider := &fakeProvider{}
	engine, store, id := newTestEngine(t, "denoise", provider)
	storeInput(t, store, "input/source.png", []byte("source"))
	var admittedMilli int64
	engine.deps.CPU = func(_ context.Context, _ string, milli int64) (capacity.MemoryLease, error) {
		admittedMilli = milli
		return &memoryTestLease{}, nil
	}
	if _, err := runJob(t, engine, "denoise", Payload{ModelID: id, InputKey: "input/source.png"}); err != nil {
		t.Fatal(err)
	}
	if admittedMilli <= 0 || int64(provider.lastReq.CPUThreads)*1000 != admittedMilli {
		t.Fatalf("provider threads %d do not match admitted CPU budget %d millicores", provider.lastReq.CPUThreads, admittedMilli)
	}
}

// scriptedBroker answers each claim in order, so a test can deny the
// GPU-resident footprint and then grant the streamed floor.
type scriptedBroker struct {
	denyAbove int64 // claims larger than this are denied
	claims    []int64
	released  []string
}

func (b *scriptedBroker) Claim(_ context.Context, _ string, preferredBytes int64, _ capacity.ClaimOptions) (capacity.Lease, error) {
	b.claims = append(b.claims, preferredBytes)
	if preferredBytes > b.denyAbove {
		return capacity.Lease{DegradeToCPU: true, Warnings: []string{"idle residents are still inside their grace window"}}, nil
	}
	return capacity.Lease{ClaimID: fmt.Sprintf("clm-%d", preferredBytes>>30), ReclaimBytes: 4 << 30}, nil
}
func (b *scriptedBroker) Release(_ context.Context, lease capacity.Lease) error {
	b.released = append(b.released, lease.ClaimID)
	return nil
}
func (b *scriptedBroker) Heartbeat(_ context.Context, lease capacity.Lease) (capacity.Lease, error) {
	return lease, nil
}

func TestCapacity_StagedModelAsksForResidentFootprintThenFallsBackToStreamedFloor(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	klein, ok := registry.ByID("flux-2-klein-4b")
	if !ok || klein.Hardware.ResidentVRAMGB <= klein.Hardware.MinVRAMGB {
		t.Fatalf("klein must declare a resident footprint above its streamed floor: %+v", klein.Hardware)
	}
	for _, tc := range []struct {
		name       string
		denyAbove  int64
		wantClaims []int64
		residency  string
	}{
		{"resident granted after reclaim", 16 << 30, []int64{12 << 30}, "resident"},
		{"resident denied, streamed floor granted", 6 << 30, []int64{12 << 30, 6 << 30}, "streamed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fp := &fakeProvider{}
			broker := &scriptedBroker{denyAbove: tc.denyAbove}
			eng, _ := newCapacityTestEngineForModel(t, "edit_instruct", klein, fp, broker)
			raw, err := marshalPinnedPayload(t, eng, "edit_instruct", Payload{Operation: "edit_instruct", ModelID: klein.ID, GPU: true, CapacityPriority: "service", AllowReclaim: true})
			if err != nil {
				t.Fatal(err)
			}
			job := internaljobs.Job{ID: "job-resident", Operation: "edit_instruct", Payload: raw}
			admission, err := eng.Admit(context.Background(), job)
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(broker.claims) != fmt.Sprint(tc.wantClaims) {
				t.Fatalf("claims = %v, want %v", broker.claims, tc.wantClaims)
			}
			if admission.Meta["gpu_residency"] != tc.residency {
				t.Fatalf("gpu_residency = %q, meta %v", admission.Meta["gpu_residency"], admission.Meta)
			}
			if tc.residency == "streamed" && !strings.Contains(admission.Meta["gpu_residency_unavailable"], "grace") {
				t.Fatalf("the reason a resident footprint was not granted must be recorded: %v", admission.Meta)
			}
			if err := admission.Release(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(broker.released) != 1 || broker.released[0] != fmt.Sprintf("clm-%d", tc.wantClaims[len(tc.wantClaims)-1]>>30) {
				t.Fatalf("exactly the granted claim must be released, released %v", broker.released)
			}
		})
	}
}

// A ledger call that timed out on the fleet-wide admission lock is a condition
// that clears by itself: the job must requeue as capacity-busy, not fail.
func TestCapacity_LedgerTimeoutRequeuesInsteadOfFailing(t *testing.T) {
	timeout := fmt.Errorf("run vrooli capacity claim: %w", fmt.Errorf("claim timed out after 29.9s: %w", context.DeadlineExceeded))
	for name, fail := range map[string]error{"timeout": timeout, "other error": errors.New("ledger corrupt")} {
		t.Run(name, func(t *testing.T) {
			broker := &fakeBroker{claimErr: fail}
			eng, modelID := newCapacityTestEngine(t, "text_to_image", &fakeProvider{}, broker)
			err := runGPUJob(t, eng, "text_to_image", modelID)
			if err == nil {
				t.Fatal("a failed claim must not run the job")
			}
			if busy := errors.Is(err, internaljobs.ErrCapacityBusy); busy != (name == "timeout") {
				t.Fatalf("capacity-busy = %v for %q: %v", busy, name, err)
			}
		})
	}
}

// What the broker reclaimed from idle owners to admit the job is part of the
// job's record.
func TestCapacity_JobMetaRecordsWhatWasReclaimed(t *testing.T) {
	broker := &fakeBroker{reclaimBytes: 3 << 30}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", &fakeProvider{}, broker)
	raw, err := marshalPinnedPayload(t, eng, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID, GPU: true, CapacityPriority: "service", AllowReclaim: true})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := eng.Admit(context.Background(), internaljobs.Job{ID: "job-xyz", Operation: "text_to_image", Payload: raw})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = admission.Release(context.Background()) }()
	if got := admission.Meta["gpu_reclaimed_bytes"]; got != fmt.Sprint(int64(3<<30)) {
		t.Fatalf("meta = %v", admission.Meta)
	}
}
