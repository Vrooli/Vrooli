package capacity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostpressure"
)

func writeResourceManifest(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "resources", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func admitTestStore(t *testing.T) (func(context.Context) (AdmitStore, error), func() time.Time) {
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	clk := func() time.Time { return time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC) }
	open := func(ctx context.Context) (AdmitStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: clockAdapter(clk)})
	}
	return open, clk
}

func TestAdmitResourceRecordsClaim(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "whisper", `{"name":"whisper","acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{},"claim":{"resource_kind":"vram","default_preferred_bytes":7516192768,"floor_bytes":1073741824,"default_priority":"service","confidence":"estimated"}}}`)
	open, clk := admitTestStore(t)

	res, err := AdmitResource(ctx, AdmitOptions{
		Root: root, ResourceName: "whisper",
		Source:    StaticSource{Inventory: snapshotWith(16, 4)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	})
	if err != nil {
		t.Fatalf("AdmitResource() error = %v", err)
	}
	if res.Skipped {
		t.Fatalf("expected a recorded claim, got skipped: %s", res.Reason)
	}
	if res.Verdict.Kind != VerdictGrant || res.ClaimID == "" {
		t.Errorf("result = %+v, want grant + claim id", res)
	}

	store, _ := open(ctx)
	defer store.Close()
	claims, _ := store.ListClaims(ctx, ClaimFilter{OwnerID: "whisper"})
	if len(claims) != 1 || claims[0].Priority != PriorityService {
		t.Fatalf("ledger = %+v, want one service claim", claims)
	}
}

func TestAdmitResourceIdempotentReuse(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "whisper", `{"acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{},"claim":{"resource_kind":"vram","default_preferred_bytes":7516192768,"floor_bytes":1073741824,"default_priority":"service","confidence":"estimated"}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{
		Root: root, ResourceName: "whisper",
		Source:    StaticSource{Inventory: snapshotWith(16, 0)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	}

	first, err := AdmitResource(ctx, opts)
	if err != nil || first.ClaimID == "" {
		t.Fatalf("first AdmitResource() = %+v, err %v", first, err)
	}
	// Re-admitting the same resident (restart) must reuse the existing active
	// claim, not stack a second one.
	second, err := AdmitResource(ctx, opts)
	if err != nil {
		t.Fatalf("second AdmitResource() error = %v", err)
	}
	if second.ClaimID != first.ClaimID {
		t.Errorf("second claim id = %q, want reuse of %q", second.ClaimID, first.ClaimID)
	}

	store, _ := open(ctx)
	defer store.Close()
	claims, _ := store.ListClaims(ctx, ClaimFilter{OwnerID: "whisper", Statuses: ActiveClaimStatuses()})
	if len(claims) != 1 {
		t.Fatalf("ledger has %d active whisper claims, want exactly 1 (idempotent)", len(claims))
	}
}

func TestAdmitResourceReplacesStaleManifestClaim(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "whisper", `{"acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{},"claim":{"resource_kind":"vram","default_preferred_bytes":8589934592,"floor_bytes":2147483648,"default_priority":"interactive","confidence":"estimated","yield_when_idle":true,"idle_grace_seconds":60}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{
		Root: root, ResourceName: "whisper",
		Source:    StaticSource{Inventory: snapshotWith(16, 0)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	}

	first, err := AdmitResource(ctx, opts)
	if err != nil || first.ClaimID == "" {
		t.Fatalf("first AdmitResource() = %+v, err %v", first, err)
	}
	writeResourceManifest(t, root, "whisper", `{"acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{},"claim":{"resource_kind":"vram","default_preferred_bytes":5368709120,"floor_bytes":2147483648,"default_priority":"interactive","confidence":"estimated","yield_when_idle":true,"idle_grace_seconds":900}}}`)
	second, err := AdmitResource(ctx, opts)
	if err != nil {
		t.Fatalf("second AdmitResource() error = %v", err)
	}
	if second.ClaimID == "" || second.ClaimID == first.ClaimID {
		t.Fatalf("second claim id = %q, want replacement of stale %q", second.ClaimID, first.ClaimID)
	}

	store, _ := open(ctx)
	defer store.Close()
	active, _ := store.ListClaims(ctx, ClaimFilter{OwnerID: "whisper", Statuses: ActiveClaimStatuses()})
	if len(active) != 1 {
		t.Fatalf("active whisper claims = %d, want exactly 1", len(active))
	}
	if active[0].PreferredBytes != 5368709120 || active[0].IdleGrace != 15*time.Minute {
		t.Fatalf("active claim = %+v, want refreshed manifest shape", active[0])
	}
	old, err := store.GetClaim(ctx, first.ClaimID)
	if err != nil {
		t.Fatalf("old claim lookup: %v", err)
	}
	if old.Status != StatusReleased {
		t.Fatalf("old claim status = %q, want released", old.Status)
	}
}

func TestAdmitResourceFlagOffIsByteIdenticalNoop(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	// Declares a profile, but enforcement is OFF -> no claim recorded at all.
	writeResourceManifest(t, root, "whisper", `{"acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{},"claim":{"resource_kind":"vram","default_preferred_bytes":1,"floor_bytes":1,"default_priority":"service","confidence":"estimated"}}}`)
	open, clk := admitTestStore(t)

	res, err := AdmitResource(ctx, AdmitOptions{
		Root: root, ResourceName: "whisper",
		Source:    StaticSource{Inventory: snapshotWith(16, 0)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceOff,
	})
	if err != nil {
		t.Fatalf("AdmitResource() error = %v", err)
	}
	if !res.Skipped {
		t.Fatalf("flag-off admission must skip; got %+v", res)
	}
	store, _ := open(ctx)
	defer store.Close()
	claims, _ := store.ListClaims(ctx, ClaimFilter{})
	if len(claims) != 0 {
		t.Errorf("flag-off must record no claim; ledger = %+v", claims)
	}
}

func TestAdmitResourceNoProfileSkips(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "plain", `{"name":"plain"}`) // no capacity block
	open, clk := admitTestStore(t)

	res, err := AdmitResource(ctx, AdmitOptions{
		Root: root, ResourceName: "plain",
		Source:    StaticSource{Inventory: snapshotWith(16, 0)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	})
	if err != nil {
		t.Fatalf("AdmitResource() error = %v", err)
	}
	if !res.Skipped {
		t.Errorf("no-profile admission must skip; got %+v", res)
	}
}

func TestAdmitResourceMissingManifestSkips(t *testing.T) {
	ctx := context.Background()
	open, clk := admitTestStore(t)
	res, err := AdmitResource(ctx, AdmitOptions{
		Root: t.TempDir(), ResourceName: "ghost",
		Source: StaticSource{Inventory: hostinventory.Snapshot{}}, OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	})
	if err != nil {
		t.Fatalf("AdmitResource() error = %v", err)
	}
	if !res.Skipped {
		t.Errorf("missing manifest must skip cleanly; got %+v", res)
	}
}

func TestLoadResourceClaimSpec(t *testing.T) {
	root := t.TempDir()
	// The claim lives inside the acceleration block, so a resource cannot
	// reserve VRAM without declaring the backend it needs it on.
	writeResourceManifest(t, root, "img", `{"acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{},"claim":{"resource_kind":"vram","default_preferred_bytes":100,"floor_bytes":0,"default_priority":"batch","confidence":"estimated","idle_grace_seconds":900,"yield_when_idle":true,"profile":{"model":{"name":"model.bin","digest":"sha256:fixture"},"steps":[{"label":"fp16","default_amount_bytes":100},{"label":"cpu","default_amount_bytes":0}],"apply":{"verb":"capacity","argv":["degrade","--to","{label}"]},"upshift":true}}}}`)
	spec, ok, err := LoadResourceClaimSpec(root, "img")
	if err != nil || !ok {
		t.Fatalf("LoadResourceClaimSpec() = ok %v err %v", ok, err)
	}
	if spec.IdleGraceSeconds != 900 {
		t.Fatalf("idle_grace_seconds = %d, want 900", spec.IdleGraceSeconds)
	}
	if spec.Profile == nil || len(spec.Profile.Steps) != 2 || spec.Profile.Steps[1].Label != "cpu" {
		t.Errorf("spec profile = %+v", spec.Profile)
	}
	if spec.Profile.Model != (ModelIdentity{Name: "model.bin", Digest: "sha256:fixture"}) {
		t.Fatalf("spec model = %+v, want exact claim model identity", spec.Profile.Model)
	}
}

func TestRealWhisperHasRequiredCapacitySpec(t *testing.T) {
	spec, ok, err := LoadResourceClaimSpec(findAdmissionRepoRoot(t), "whisper")
	if err != nil || !ok {
		t.Fatalf("LoadResourceClaimSpec(whisper) = ok %v err %v, want its mandatory accelerator claim", ok, err)
	}
	if spec.PreferredBytes != 2*1024*1024*1024 || spec.FloorBytes != 0 {
		t.Fatalf("whisper claim = %+v, want 2 GiB preferred and CPU floor", spec)
	}
	wantModel := ModelIdentity{Name: "ggml-medium.bin", Digest: "6c14d5adee5f86394037b4e4e8b59f1673b6cee10e3cf0b11bbdbee79c156208"}
	if spec.Profile == nil || spec.Profile.Model != wantModel {
		t.Fatalf("whisper residency model = %+v, want %+v", spec.Profile, wantModel)
	}
}

func TestRealKyutaiCapacitySpecYieldsWhenIdle(t *testing.T) {
	spec, ok, err := LoadResourceClaimSpec(findAdmissionRepoRoot(t), "kyutai-stt")
	if err != nil || !ok {
		t.Fatalf("LoadResourceClaimSpec(kyutai-stt) = ok %v err %v", ok, err)
	}
	if spec.Protected {
		t.Fatal("kyutai-stt must not be statically protected; activity state owns active protection")
	}
	if !spec.YieldWhenIdle {
		t.Fatal("kyutai-stt must yield when idle")
	}
	if spec.IdleUnloadTTLSeconds != 900 {
		t.Fatalf("kyutai-stt idle_unload_ttl_seconds = %d, want 900", spec.IdleUnloadTTLSeconds)
	}
	if spec.IdleGraceSeconds != 900 {
		t.Fatalf("kyutai-stt idle_grace_seconds = %d, want 900", spec.IdleGraceSeconds)
	}
	if spec.Profile == nil || len(spec.Profile.Steps) != 2 {
		t.Fatalf("kyutai-stt profile = %+v, want gpu/cpu ladder", spec.Profile)
	}
	if got := spec.Profile.Steps[1]; got.Label != "cpu" || got.AmountBytes != 0 {
		t.Fatalf("kyutai-stt floor step = %+v, want cpu/0", got)
	}
	// Every resource answers the broker with the same verb and a runnable CPU
	// floor rather than stopping the dictation engine.
	if spec.Profile.Apply.Verb != "capacity" {
		t.Fatalf("kyutai-stt apply verb = %q, want the fleet-wide %q verb", spec.Profile.Apply.Verb, "capacity")
	}
	if !spec.Profile.Upshift {
		t.Fatal("kyutai-stt profile should upshift from CPU when device headroom returns")
	}
}

func findAdmissionRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate repo root")
	return ""
}

type notifyingAdmissionStore struct {
	AdmitStore
	renewed chan struct{}
}

func (s notifyingAdmissionStore) HeartbeatClaim(ctx context.Context, id string, g int64, ttl time.Duration) (CapacityClaim, error) {
	c, e := s.AdmitStore.HeartbeatClaim(ctx, id, g, ttl)
	select {
	case s.renewed <- struct{}{}:
	default:
	}
	return c, e
}

func livenessFixture(t *testing.T) (AdmitOptions, *testenv.Clock, chan struct{}) {
	t.Helper()
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"resource_kind":"vram","default_preferred_bytes":9663676416,"floor_bytes":9663676416,"default_priority":"batch","idle_grace_seconds":900}}}`)
	clk := testenv.NewClock(time.Date(2026, 10, 17, 1, 0, 0, 0, time.UTC))
	path := filepath.Join(t.TempDir(), "capacity.db")
	renewed := make(chan struct{}, 16)
	opts := AdmitOptions{Root: root, ResourceName: "model", Source: StaticSource{Inventory: snapshotWith(16, 0)}, EnforceEnv: EnforceAdvisory, Clock: clk.Now}
	opts.OpenStore = func(ctx context.Context) (AdmitStore, error) {
		s, e := NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk})
		if e != nil {
			return nil, e
		}
		return notifyingAdmissionStore{s, renewed}, nil
	}
	return opts, clk, renewed
}

func TestStartupLeaseSurvivesMultipleUnobservedTTLsAndHandoff(t *testing.T) {
	opts, clk, renewed := livenessFixture(t)
	ticks := make(chan time.Time)
	err := withResourceAdmissionTicks(context.Background(), opts, ticks, func(ctx context.Context) error {
		store, e := opts.OpenStore(ctx)
		if e != nil {
			return e
		}
		defer store.Close()
		for range 5 {
			clk.Advance(20 * time.Second)
			ticks <- clk.Now()
			select {
			case <-renewed:
			case <-time.After(time.Second):
				t.Fatal("startup did not renew")
			}
			absent := snapshotWithProcs()
			absent.FieldProvenance = map[string]hostinventory.Provenance{"gpu_processes": {ObservedAt: clk.Now()}}
			swept, e := Sweep(ctx, store, absent, fakeAttributor{}, DefaultPolicy(), clk.Now())
			if e != nil || len(swept.Expired) != 0 {
				t.Fatalf("CPU-first startup lost reservation: %+v %v", swept, e)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	store, _ := opts.OpenStore(context.Background())
	defer store.Close()
	claims, e := store.ListClaims(context.Background(), ClaimFilter{OwnerID: "model", Statuses: ActiveClaimStatuses()})
	if e != nil || len(claims) != 1 || claims[0].ActivityState != ActivityIdle || claims[0].HeartbeatDeadlineAt == nil || !claims[0].HeartbeatDeadlineAt.After(clk.Now()) {
		t.Fatalf("handoff must retain one idle, renewed claim: %+v %v", claims, e)
	}
}

func TestFailedOrCancelledStartupRetainsPossibleSurvivingProcessClaim(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			opts, _, _ := livenessFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("model still supervised after health failure")
			err := WithResourceAdmission(ctx, opts, func(context.Context) error {
				if cancelled {
					cancel()
					return ctx.Err()
				}
				return failure
			})
			if cancelled && !errors.Is(err, context.Canceled) || !cancelled && !errors.Is(err, failure) {
				t.Fatalf("outcome lost: %v", err)
			}
			store, _ := opts.OpenStore(context.Background())
			defer store.Close()
			claims, _ := store.ListClaims(context.Background(), ClaimFilter{OwnerID: "model", Statuses: ActiveClaimStatuses()})
			if len(claims) != 1 {
				t.Fatalf("request failure falsely released live process: %+v", claims)
			}
		})
	}
}

func TestVerifiedStopReleasesCapturedOwnerOnlyAndFailurePreserves(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint(failed), func(t *testing.T) {
			opts, _, _ := livenessFixture(t)
			ctx := context.Background()
			admitted, e := AdmitResource(ctx, opts)
			if e != nil {
				t.Fatal(e)
			}
			store, _ := opts.OpenStore(ctx)
			defer store.Close()
			other, e := store.CreateClaim(ctx, CapacityClaim{OwnerKind: OwnerKindResource, OwnerID: "ollama", ResourceKind: ResourceKindVRAM, AmountBytes: 1 << 30}, 0)
			if e != nil {
				t.Fatal(e)
			}
			var future CapacityClaim
			failure := errors.New("stop failed; process alive")
			err := ReleaseResourceAdmissionAfterStop(ctx, opts, func() error {
				if failed {
					return failure
				}
				var e error
				future, e = store.CreateClaim(ctx, CapacityClaim{OwnerKind: OwnerKindResource, OwnerID: "model", ResourceKind: ResourceKindVRAM, AmountBytes: 1 << 30}, 0)
				return e
			})
			if failed && !errors.Is(err, failure) || !failed && err != nil {
				t.Fatalf("stop outcome=%v", err)
			}
			own, _ := store.GetClaim(ctx, admitted.ClaimID)
			unrelated, _ := store.GetClaim(ctx, other.ClaimID)
			want := StatusReleased
			if failed {
				want = StatusGranted
			}
			if own.Status != want || unrelated.Status != StatusGranted {
				t.Fatalf("wrong claims changed: own=%+v other=%+v", own, unrelated)
			}
			if !failed {
				after, _ := store.GetClaim(ctx, future.ClaimID)
				if after.Status != StatusGranted {
					t.Fatalf("uncaptured replacement released: %+v", after)
				}
			}
		})
	}
}

func TestStartupLeaseBrokerUnavailableIsObservableAndFailOpen(t *testing.T) {
	opts, _, _ := livenessFixture(t)
	var log bytes.Buffer
	opts.Log = &log
	opts.OpenStore = func(context.Context) (AdmitStore, error) { return nil, errors.New("ledger down") }
	called := false
	want := errors.New("real owner outcome")
	err := WithResourceAdmission(context.Background(), opts, func(context.Context) error { called = true; return want })
	if !called || !errors.Is(err, want) || !strings.Contains(log.String(), "ledger down") {
		t.Fatalf("advisory failure hid owner/error: %t %v %q", called, err, log.String())
	}
}

func TestStartupLeaseNoDeclarationOrEnforcementOffDoesNotOpenStore(t *testing.T) {
	for _, off := range []bool{false, true} {
		t.Run(fmt.Sprint(off), func(t *testing.T) {
			opts, _, _ := livenessFixture(t)
			if off {
				opts.EnforceEnv = EnforceOff
			} else {
				opts.ResourceName = "undeclared"
			}
			opts.OpenStore = func(context.Context) (AdmitStore, error) { t.Fatal("parity path opened ledger"); return nil, nil }
			called := false
			if e := WithResourceAdmission(context.Background(), opts, func(context.Context) error { called = true; return nil }); e != nil || !called {
				t.Fatalf("parity run=%t %v", called, e)
			}
		})
	}
}

func TestStartupCancellationDuringAdmissionNeverInvokesOwner(t *testing.T) {
	opts, _, _ := livenessFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opts.OpenStore = func(context.Context) (AdmitStore, error) { cancel(); return nil, ctx.Err() }
	calls := 0
	err := WithResourceAdmission(ctx, opts, func(context.Context) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("cancelled admission invoked owner %d times: %v", calls, err)
	}
}

func TestAdmitResourceUsesFreshMemoryPSI(t *testing.T) {
	for _, tc := range []struct {
		name            string
		reading         hostpressure.Reading
		available, free uint64
		want            string
	}{
		{"quiet occupied swap", hostpressure.NewRead(0, "fixture"), 90, 20, VerdictGrant},
		{"active pressure low swap", hostpressure.NewRead(99, "fixture"), 90, 90, VerdictDeny},
		{"unread retains occupancy gate", hostpressure.NewUnread("fixture", "unavailable"), 90, 20, VerdictDeny},
		{"quiet still insufficient RAM", hostpressure.NewRead(0, "fixture"), 5, 20, VerdictDeny},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeResourceManifest(t, root, "ram-fixture", `{"acceleration":{"claim":{"resource_kind":"ram","default_preferred_bytes":10,"floor_bytes":10}}}`)
			open, clk := admitTestStore(t)
			out, err := AdmitResource(context.Background(), AdmitOptions{Root: root, ResourceName: "ram-fixture", OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
				Source:         StaticSource{Inventory: hostinventory.Snapshot{Memory: hostinventory.Memory{TotalBytes: 100, AvailableBytes: tc.available}, Swap: hostinventory.Swap{TotalBytes: 100, FreeBytes: tc.free}}},
				MemoryPressure: func() hostpressure.Reading { return tc.reading },
			})
			if err != nil {
				t.Fatal(err)
			}
			if out.Verdict.Kind != tc.want {
				t.Fatalf("verdict=%+v, want %s", out.Verdict, tc.want)
			}
		})
	}
}

func TestAdmitResourceReusesCanonicalGPUClaim(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":1073741824,"floor_bytes":1073741824,"default_priority":"batch"}}}`)
	open, clk := admitTestStore(t)
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	zero := 0
	original, err := store.CreateClaim(ctx, CapacityClaim{OwnerKind: OwnerKindResource, OwnerID: "model", ResourceKind: ResourceKindVRAM, GPUIndex: &zero, AmountBytes: 1 << 30, PreferredBytes: 1 << 30, FloorBytes: 1 << 30, Priority: PriorityBatch}, 0)
	store.Close()
	if err != nil {
		t.Fatal(err)
	}
	result, err := AdmitResource(ctx, AdmitOptions{Root: root, ResourceName: "model", OpenStore: open, Clock: clk, Source: StaticSource{Inventory: snapshotWith(16, 0)}, EnforceEnv: EnforceAdvisory})
	if err != nil {
		t.Fatal(err)
	}
	if result.ClaimID != original.ClaimID {
		t.Fatalf("GPU0 identity changed claim %s to %s", original.ClaimID, result.ClaimID)
	}
}

func TestAdmitResourceReuseReportsActualRung(t *testing.T) {
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":7516192768,"floor_bytes":1073741824,"default_priority":"batch"}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{Root: root, ResourceName: "model", OpenStore: open, Clock: clk, Source: StaticSource{Inventory: snapshotWith(16, 14)}, EnforceEnv: EnforceAdvisory}
	first, err := AdmitResource(context.Background(), opts)
	if err != nil || first.Verdict.Kind != VerdictDegrade {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := AdmitResource(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if second.ClaimID != first.ClaimID || second.Verdict.GrantedBytes != 1<<30 || second.Verdict.Kind != VerdictDegrade {
		t.Fatalf("reused claim fabricated preferred grant: %+v", second)
	}
}

type failedRenewalStore struct{ AdmitStore }

func (s failedRenewalStore) HeartbeatClaim(context.Context, string, int64, time.Duration) (CapacityClaim, error) {
	return CapacityClaim{}, errors.New("renewal unavailable")
}

func TestAdmitResourceReuseRequiresRenewal(t *testing.T) {
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":1073741824,"floor_bytes":1073741824,"default_priority":"batch"}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{Root: root, ResourceName: "model", OpenStore: open, Clock: clk, Source: StaticSource{Inventory: snapshotWith(16, 0)}, EnforceEnv: EnforceAdvisory}
	if _, err := AdmitResource(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	opts.OpenStore = func(ctx context.Context) (AdmitStore, error) { s, e := open(ctx); return failedRenewalStore{s}, e }
	if result, err := AdmitResource(context.Background(), opts); err == nil {
		t.Fatalf("accepted unrenewed claim: %+v", result)
	}
}

func TestNoReclaimStartupUsesManifestAndPreservesOtherOwners(t *testing.T) {
	for _, mode := range []string{EnforceAdvisory, EnforceOn} {
		for _, free := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/free=%t", mode, free), func(t *testing.T) {
				ctx := context.Background()
				root := t.TempDir()
				writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":10737418240,"floor_bytes":10737418240,"default_priority":"batch","yield_when_idle":true,"idle_grace_seconds":5}}}`)
				open, clk := admitTestStore(t)
				store, err := open(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer store.Close()
				other, err := store.CreateClaim(ctx, CapacityClaim{OwnerKind: OwnerKindResource, OwnerID: "shared", ResourceKind: ResourceKindVRAM, AmountBytes: 8 << 30, PreferredBytes: 8 << 30, Priority: PriorityBatch, YieldWhenIdle: true, ActivityState: ActivityIdle, LastActiveAt: func() *time.Time { v := clk().Add(-time.Hour); return &v }()}, time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				total := 16
				used := 8
				if free {
					total = 24
				}
				ran := false
				opts := AdmitOptions{Root: root, ResourceName: "model", NoReclaim: true, OpenStore: open, Clock: clk, EnforceEnv: mode, Source: StaticSource{Inventory: snapshotWith(int64(total), int64(used))}}
				err = WithResourceAdmission(ctx, opts, func(context.Context) error { ran = true; return nil })
				if ran != free || (err == nil) != free {
					t.Fatalf("ran=%v err=%v, want admitted=%v", ran, err, free)
				}
				current, err := store.GetClaim(ctx, other.ClaimID)
				if err != nil || current.Status != other.Status || current.AmountBytes != other.AmountBytes || current.Generation != other.Generation {
					t.Fatalf("changed other owner: %+v err=%v", current, err)
				}
				own, err := store.ListClaims(ctx, ClaimFilter{OwnerID: "model", Statuses: ActiveClaimStatuses()})
				if err != nil {
					t.Fatal(err)
				}
				if !free && len(own) != 0 {
					t.Fatalf("denied startup leaked reservation: %+v", own)
				}
				if free && (len(own) != 1 || own[0].AmountBytes != 10<<30 || !own[0].YieldWhenIdle || own[0].IdleGrace != 5*time.Second) {
					t.Fatalf("manifest not retained: %+v", own)
				}
			})
		}
	}
}

func TestNoReclaimStartupFailsClosed(t *testing.T) {
	for _, failure := range []string{"sensor", "ledger", "off", "undeclared"} {
		t.Run(failure, func(t *testing.T) {
			root := t.TempDir()
			if failure != "undeclared" {
				writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":1073741824,"floor_bytes":1073741824}}}`)
			}
			open, clk := admitTestStore(t)
			opts := AdmitOptions{Root: root, ResourceName: "model", NoReclaim: true, OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory, Source: StaticSource{Inventory: snapshotWith(16, 0)}}
			switch failure {
			case "sensor":
				opts.Source = StaticSource{Err: errors.New("sensor unavailable")}
			case "ledger":
				opts.OpenStore = func(context.Context) (AdmitStore, error) { return nil, errors.New("ledger unavailable") }
			case "off":
				opts.EnforceEnv = EnforceOff
			}
			ran := false
			err := WithResourceAdmission(context.Background(), opts, func(context.Context) error { ran = true; return nil })
			if ran || err == nil {
				t.Fatalf("startup ran without required admission: ran=%v err=%v", ran, err)
			}
		})
	}
}

func TestAdvisoryReservationDoesNotBecomeGrantOnReuse(t *testing.T) {
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":1073741824,"floor_bytes":1073741824}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{Root: root, ResourceName: "model", OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory, Source: StaticSource{Inventory: snapshotWith(16, 16)}}
	for i := 0; i < 2; i++ {
		got, err := AdmitResource(context.Background(), opts)
		if err != nil || got.Verdict.Kind != VerdictDeny {
			t.Fatalf("attempt%d fabricated grant: %+v %v", i, got, err)
		}
	}
}

type failedCreateStore struct{ AdmitStore }

func (s failedCreateStore) CreateClaim(context.Context, CapacityClaim, time.Duration) (CapacityClaim, error) {
	return CapacityClaim{}, errors.New("write unavailable")
}

func TestResourceReplacementPreservesPriorClaimOnFailure(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":1073741824,"floor_bytes":1073741824}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{Root: root, ResourceName: "model", OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory, Source: StaticSource{Inventory: snapshotWith(16, 0)}}
	first, err := AdmitResource(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":2147483648,"floor_bytes":2147483648}}}`)
	opts.OpenStore = func(ctx context.Context) (AdmitStore, error) { s, e := open(ctx); return failedCreateStore{s}, e }
	if _, err := AdmitResource(ctx, opts); err == nil {
		t.Fatal("expected replacement write failure")
	}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.GetClaim(ctx, first.ClaimID)
	if err != nil || !IsActiveClaimStatus(got.Status) {
		t.Fatalf("failed replacement erased old claim: %+v %v", got, err)
	}
}

func TestNoReclaimManifestReplacementKeepsConstraintAndPriorReservation(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":1073741824,"floor_bytes":1073741824}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{Root: root, ResourceName: "model", OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory, Source: StaticSource{Inventory: snapshotWith(16, 0)}}
	first, err := AdmitResource(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":10737418240,"floor_bytes":10737418240,"yield_when_idle":true,"idle_grace_seconds":5}}}`)
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	last := clk().Add(-time.Hour)
	other, err := store.CreateClaim(ctx, CapacityClaim{OwnerKind: OwnerKindResource, OwnerID: "shared", ResourceKind: ResourceKindVRAM, AmountBytes: 8 << 30, PreferredBytes: 8 << 30, Priority: PriorityBatch, YieldWhenIdle: true, ActivityState: ActivityIdle, LastActiveAt: &last}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	opts.NoReclaim = true
	opts.Source = StaticSource{Inventory: snapshotWith(16, 8)}
	refused, err := AdmitResource(ctx, opts)
	if err == nil || refused.Verdict.Kind != VerdictDeny || len(refused.Verdict.ReclaimTargets) != 0 {
		t.Fatalf("replacement lost no-reclaim: %+v %v", refused, err)
	}
	old, err := store.GetClaim(ctx, first.ClaimID)
	if err != nil || !IsActiveClaimStatus(old.Status) {
		t.Fatalf("denial erased existing accounting: %+v %v", old, err)
	}
	opts.Source = StaticSource{Inventory: snapshotWith(24, 8)}
	admitted, err := AdmitResource(ctx, opts)
	if err != nil || !admitted.Verdict.Granted() || admitted.ClaimID == first.ClaimID || admitted.Verdict.ReclaimBytes != 0 {
		t.Fatalf("fresh manifest admission failed: %+v %v", admitted, err)
	}
	old, err = store.GetClaim(ctx, first.ClaimID)
	if err != nil || old.Status != StatusReleased {
		t.Fatalf("old claim not replaced: %+v %v", old, err)
	}
	same, err := AdmitResource(ctx, opts)
	if err != nil || same.ClaimID != admitted.ClaimID {
		t.Fatalf("constrained reuse changed claim identity: %+v %v", same, err)
	}
	current, err := store.GetClaim(ctx, other.ClaimID)
	if err != nil || current.Status != other.Status || current.Generation != other.Generation {
		t.Fatalf("other owner changed: %+v %v", current, err)
	}
}

type failedReleaseStore struct{ AdmitStore }

func (s failedReleaseStore) ReleaseClaim(context.Context, string) (CapacityClaim, error) {
	return CapacityClaim{}, errors.New("release unavailable")
}

func TestResourceReplacementFailureRetainsRecoveryIdentity(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":1073741824,"floor_bytes":1073741824}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{Root: root, ResourceName: "model", OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory, Source: StaticSource{Inventory: snapshotWith(16, 0)}}
	first, err := AdmitResource(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":2147483648,"floor_bytes":2147483648}}}`)
	opts.OpenStore = func(ctx context.Context) (AdmitStore, error) { s, e := open(ctx); return failedReleaseStore{s}, e }
	next, err := AdmitResource(ctx, opts)
	if err == nil || next.ClaimID == "" || next.ClaimID == first.ClaimID || !strings.Contains(err.Error(), next.ClaimID) || !strings.Contains(err.Error(), first.ClaimID) {
		t.Fatalf("partial replacement hid recovery identities: %+v %v", next, err)
	}
	store, err := open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, id := range []string{first.ClaimID, next.ClaimID} {
		c, e := store.GetClaim(ctx, id)
		if e != nil || !IsActiveClaimStatus(c.Status) {
			t.Fatalf("unproven release: %+v %v", c, e)
		}
	}
}

func TestNoReclaimStartupRequiresSelectedManifestBudget(t *testing.T) {
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":7516192768,"floor_bytes":1073741824,"profile":{"steps":[{"label":"full","amount_bytes":7516192768},{"label":"small","amount_bytes":1073741824}]}}}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{Root: root, ResourceName: "model", NoReclaim: true, OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory, Source: StaticSource{Inventory: snapshotWith(16, 14)}}
	ran := false
	err := WithResourceAdmission(context.Background(), opts, func(context.Context) error { ran = true; return nil })
	if err == nil || ran {
		t.Fatalf("lower-rung recommendation launched unchanged manifest budget: ran=%v err=%v", ran, err)
	}
}
