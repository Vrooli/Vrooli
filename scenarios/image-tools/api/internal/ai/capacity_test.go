package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/blobstore"
)

// fakeBroker records the claim it received and returns a scripted verdict.
type fakeBroker struct {
	degrade     bool
	claimErr    error
	claimedFor  string
	claimedPref int64
	claimedOpts CapacityClaimOptions
	released    string
	claimN      int
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

func (b *fakeBroker) Release(_ context.Context, claimID string) { b.released = claimID }

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
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
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

func TestCLICapacityBroker_ParsesVerdict(t *testing.T) {
	cases := []struct {
		name        string
		stdout      string
		wantDegrade bool
		wantReclaim int64
	}{
		{"grant fp16-gpu keeps GPU when no reclaim needed", `{"verdict":{"kind":"grant","step":"fp16-gpu"},"claim":{"claim_id":"clm-1"},"enforce":"advisory"}`, false, 0},
		{"advisory reclaim grant skips GPU", `{"verdict":{"kind":"grant","step":"fp16-gpu","reclaim_bytes":1073741824},"claim":{"claim_id":"clm-1"},"enforce":"advisory"}`, true, 1073741824},
		{"enforced reclaim grant keeps GPU", `{"verdict":{"kind":"grant","step":"fp16-gpu","reclaim_bytes":1073741824},"claim":{"claim_id":"clm-1"},"enforce":"on"}`, false, 1073741824},
		{"degrade to cpu", `{"verdict":{"kind":"degrade","step":"cpu"},"claim":{"claim_id":"clm-2"}}`, true, 0},
		{"queue -> cpu", `{"verdict":{"kind":"queue","step":""},"claim":{"claim_id":"clm-3"}}`, true, 0},
		{"deny -> cpu", `{"verdict":{"kind":"deny","step":""},"claim":{"claim_id":""}}`, true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotArgs []string
			b := &CLICapacityBroker{Exec: func(_ context.Context, args ...string) ([]byte, error) {
				gotArgs = args
				return []byte(tc.stdout), nil
			}}
			lease, err := b.Claim(context.Background(), "image-tools:job-1", fallbackGPUVRAMEstimateBytes, CapacityClaimOptions{Priority: "service", AllowReclaim: true})
			if err != nil {
				t.Fatalf("claim: %v", err)
			}
			if lease.DegradeToCPU != tc.wantDegrade {
				t.Errorf("DegradeToCPU=%v want %v", lease.DegradeToCPU, tc.wantDegrade)
			}
			if lease.ReclaimBytes != tc.wantReclaim {
				t.Errorf("ReclaimBytes=%d want %d", lease.ReclaimBytes, tc.wantReclaim)
			}
			joined := strings.Join(gotArgs, " ")
			for _, want := range []string{"capacity claim", "--owner-kind op", "--owner-id image-tools:job-1", "--resource-kind vram", "--priority service", "--json"} {
				if !strings.Contains(joined, want) {
					t.Errorf("argv missing %q; got %v", want, gotArgs)
				}
			}
		})
	}
}

func TestCLICapacityBroker_DisallowReclaimUsesBatchPriority(t *testing.T) {
	var gotArgs []string
	b := &CLICapacityBroker{Exec: func(_ context.Context, args ...string) ([]byte, error) {
		gotArgs = args
		return []byte(`{"verdict":{"kind":"grant","step":"fp16-gpu"},"claim":{"claim_id":"clm-1"},"enforce":"on"}`), nil
	}}

	_, err := b.Claim(context.Background(), "image-tools:job-1", fallbackGPUVRAMEstimateBytes, CapacityClaimOptions{Priority: "service", AllowReclaim: false})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "--priority batch") {
		t.Fatalf("allow_reclaim=false should map to batch priority; got %v", gotArgs)
	}
}

func TestCapacity_ClaimErrorProceedsOnGPU(t *testing.T) {
	fp := &fakeProvider{}
	broker := &fakeBroker{claimErr: context.DeadlineExceeded}
	eng, modelID := newCapacityTestEngine(t, "text_to_image", fp, broker)

	if err := runGPUJob(t, eng, "text_to_image", modelID); err != nil {
		t.Fatalf("run: %v", err)
	}
	// Advisory: a broker error must not block or degrade the job.
	if !fp.lastReq.GPU {
		t.Errorf("claim error should leave the job on GPU (advisory); got CPU")
	}
}
