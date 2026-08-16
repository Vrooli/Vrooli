package capacity

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

func testService(t *testing.T, snap hostinventory.Snapshot, attr engine.Attributor) Service {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	sourceRoot := t.TempDir()
	clk := func() time.Time { return time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC) }
	return Service{
		OpenStore: func(ctx context.Context) (Store, error) {
			return engine.NewSQLiteStore(ctx, engine.Config{DBPath: dbPath, Clock: clockFunc(clk)})
		},
		Source:     engine.StaticSource{Inventory: snap},
		Attributor: attr,
		Clock:      clk,
		SourceRoot: sourceRoot,
	}
}

type clockFunc func() time.Time

func (c clockFunc) Now() time.Time { return c() }

func gib(n int64) int64 { return n << 30 }

func gpuSnapshot(totalGiB, usedGiB int64) hostinventory.Snapshot {
	return hostinventory.Snapshot{GPUs: []hostinventory.GPU{{
		Index: 0, Name: "Test", Source: "nvidia-smi",
		VRAMBytes: uint64(gib(totalGiB)), VRAMUsedBytes: uint64(gib(usedGiB)),
	}}}
}

func TestServiceClaimGrantAndList(t *testing.T) {
	ctx := context.Background()
	svc := testService(t, gpuSnapshot(16, 4), nil)

	out, err := svc.Claim(ctx, ClaimRequest{
		OwnerKind: engine.OwnerKindResource, OwnerID: "whisper",
		ResourceKind: engine.ResourceKindVRAM, PreferredBytes: gib(7), FloorBytes: gib(1),
		PriorityTier: "service",
	})
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if out.Verdict.Kind != engine.VerdictGrant {
		t.Errorf("verdict = %q, want grant (%s)", out.Verdict.Kind, out.Verdict.Reason)
	}
	if out.Enforce != engine.EnforceAdvisory {
		t.Errorf("enforce = %q, want advisory", out.Enforce)
	}

	list, err := svc.List(ctx, ListRequest{ActiveOnly: true})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Claims) != 1 || list.Claims[0].OwnerID != "whisper" {
		t.Fatalf("list = %+v, want one whisper claim", list.Claims)
	}
}

func TestServiceListShowsWarmAndColdIdleState(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	svc := Service{
		OpenStore: func(ctx context.Context) (Store, error) {
			return engine.NewSQLiteStore(ctx, engine.Config{DBPath: dbPath, Clock: clockFunc(func() time.Time { return now })})
		},
		Source: engine.StaticSource{Inventory: gpuSnapshot(16, 4)},
		Clock:  func() time.Time { return now },
	}

	claimed, err := svc.Claim(ctx, ClaimRequest{
		OwnerKind: engine.OwnerKindResource, OwnerID: "kyutai-stt",
		ResourceKind: engine.ResourceKindVRAM, PreferredBytes: gib(3), FloorBytes: 0,
		PriorityTier: "service", YieldWhenIdle: true, IdleGrace: 15 * time.Minute,
		TTL: 30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if claimed.Claim.IdleReclaimState != "warm_idle" {
		t.Fatalf("initial idle state = %q, want warm_idle", claimed.Claim.IdleReclaimState)
	}

	now = now.Add(16 * time.Minute)
	list, err := svc.List(ctx, ListRequest{ActiveOnly: true})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Claims) != 1 {
		t.Fatalf("claims = %+v, want one", list.Claims)
	}
	if list.Claims[0].IdleGrace != "15m0s" {
		t.Fatalf("idle_grace = %q, want 15m0s", list.Claims[0].IdleGrace)
	}
	if list.Claims[0].IdleReclaimState != "cold_idle" {
		t.Fatalf("idle state = %q, want cold_idle", list.Claims[0].IdleReclaimState)
	}
}

func TestServiceClaimDegradeInAdvisoryStillRecords(t *testing.T) {
	ctx := context.Background()
	svc := testService(t, gpuSnapshot(16, 14), nil) // only 2 GiB free
	profile := `{"steps":[{"label":"large","amount_bytes":7516192768},{"label":"small","amount_bytes":1073741824}],"upshift":true}`

	out, err := svc.Claim(ctx, ClaimRequest{
		OwnerID: "whisper", ResourceKind: engine.ResourceKindVRAM,
		PreferredBytes: gib(7), FloorBytes: gib(1), PriorityTier: "service", ProfileJSON: profile,
	})
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if out.Verdict.Kind != engine.VerdictDegrade || out.Verdict.Step != "small" {
		t.Errorf("verdict = %q/%q, want degrade/small (%s)", out.Verdict.Kind, out.Verdict.Step, out.Verdict.Reason)
	}
	if out.Claim.Status != engine.StatusDegraded {
		t.Errorf("recorded status = %q, want degraded", out.Claim.Status)
	}
}

type recordingApplyExecutor struct {
	owner string
	verb  string
	argv  []string
}

func (r *recordingApplyExecutor) Apply(_ context.Context, owner, verb string, argv []string) error {
	r.owner, r.verb, r.argv = owner, verb, append([]string(nil), argv...)
	return nil
}

func TestServiceClaimEnforceActuatesIdleProfile(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	clk := func() time.Time { return now }
	exec := &recordingApplyExecutor{}
	svc := Service{
		OpenStore: func(ctx context.Context) (Store, error) {
			return engine.NewSQLiteStore(ctx, engine.Config{DBPath: dbPath, Clock: clockFunc(clk)})
		},
		Source: engine.StaticSource{Inventory: hostinventory.Snapshot{GPUs: []hostinventory.GPU{{
			Index: 0, Name: "Test", Source: "nvidia-smi", VRAMBytes: uint64(gib(16)), VRAMUsedBytes: uint64(15*gib(1) + gib(1)/2),
		}}}},
		Exec:  exec,
		Clock: clk,
	}
	profile := `{"steps":[{"label":"large","amount_bytes":1447034880},{"label":"small","amount_bytes":633339904}],"apply":{"verb":"models","argv":["activate","--model","{label}"]},"upshift":true}`
	t.Setenv("VROOLI_CAPACITY_ENFORCE", engine.EnforceOn)
	if _, err := svc.Claim(ctx, ClaimRequest{
		OwnerKind: engine.OwnerKindResource, OwnerID: "reranker", ResourceKind: engine.ResourceKindVRAM,
		PreferredBytes: 1447034880, FloorBytes: 633339904, PriorityTier: "service", YieldWhenIdle: true,
		IdleGrace: time.Second, ProfileJSON: profile,
	}); err != nil {
		t.Fatalf("seed reranker claim: %v", err)
	}
	now = now.Add(2 * time.Second)
	out, err := svc.Claim(ctx, ClaimRequest{
		OwnerKind: engine.OwnerKindScenario, OwnerID: "interactive-request", ResourceKind: engine.ResourceKindVRAM,
		PreferredBytes: 1600000000, FloorBytes: 1000000000, PriorityTier: "interactive",
	})
	if err != nil {
		t.Fatalf("enforced claim: %v", err)
	}
	if out.Verdict.ReclaimBytes <= 0 {
		t.Fatalf("verdict reclaim_bytes = %d, want positive", out.Verdict.ReclaimBytes)
	}
	if exec.owner != "reranker" || exec.verb != "models" || len(exec.argv) != 3 || exec.argv[0] != "activate" {
		t.Fatalf("actuator call = %s %s %#v, want reranker models activate <profile-label>", exec.owner, exec.verb, exec.argv)
	}
	if exec.argv[1] != "--model" || exec.argv[2] != "small" {
		t.Fatalf("actuator argv = %#v, want [activate --model small]", exec.argv)
	}
}

func TestServiceActivityAndHeartbeatLifecycle(t *testing.T) {
	ctx := context.Background()
	svc := testService(t, gpuSnapshot(16, 0), nil)
	out, err := svc.Claim(ctx, ClaimRequest{OwnerID: "agent-manager", PreferredBytes: gib(2), FloorBytes: gib(1), PriorityTier: "interactive"})
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	id := out.Claim.ClaimID
	gen := out.Claim.Generation

	active, err := svc.Activity(ctx, Ref{ClaimID: id, Generation: gen, State: engine.ActivityActive})
	if err != nil {
		t.Fatalf("Activity() error = %v", err)
	}
	if !active.Protected || active.ActivityState != engine.ActivityActive {
		t.Errorf("active claim = %+v, want protected+active", active)
	}

	if _, err := svc.Release(ctx, Ref{ClaimID: id}); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
}

func TestServicePolicyRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc := testService(t, gpuSnapshot(16, 0), nil)

	if _, err := svc.PolicySet(ctx, "idle_grace", "90s"); err != nil {
		t.Fatalf("PolicySet() error = %v", err)
	}
	got, err := svc.PolicyGet(ctx, "idle_grace")
	if err != nil {
		t.Fatalf("PolicyGet() error = %v", err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Value != "1m30s" {
		t.Errorf("idle_grace = %+v, want 1m30s", got.Entries)
	}

	all, err := svc.PolicyGet(ctx, "")
	if err != nil {
		t.Fatalf("PolicyGet(all) error = %v", err)
	}
	if len(all.Entries) != len(engine.PolicyKeys) {
		t.Errorf("policy entries = %d, want %d", len(all.Entries), len(engine.PolicyKeys))
	}
}

func TestServiceReconcileNamesUnclaimedConsumers(t *testing.T) {
	ctx := context.Background()
	snap := hostinventory.Snapshot{
		GPUs:         []hostinventory.GPU{{Index: 0, Name: "Test", Source: "nvidia-smi", VRAMBytes: uint64(gib(16))}},
		GPUProcesses: []hostinventory.GPUProcess{{GPUIndex: 0, PID: 4242, ProcessName: "python", UsedBytes: uint64(gib(7))}},
	}
	attr := stubAttributor{4242: engine.Attribution{ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}
	svc := testService(t, snap, attr)

	out, err := svc.Reconcile(ctx)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if len(out.Findings) != 1 || out.Findings[0].Class != engine.FindingUnclaimed {
		t.Fatalf("findings = %+v, want one unclaimed", out.Findings)
	}
	if out.Findings[0].OwnerID != "whisper" {
		t.Errorf("owner = %q, want whisper", out.Findings[0].OwnerID)
	}
}

func TestServiceSweepRefreshesObservedResidentClaim(t *testing.T) {
	ctx := context.Background()
	snap := hostinventory.Snapshot{
		GPUs:         []hostinventory.GPU{{Index: 0, Name: "Test", Source: "nvidia-smi", VRAMBytes: uint64(gib(16))}},
		GPUProcesses: []hostinventory.GPUProcess{{GPUIndex: 0, PID: 4242, ProcessName: "whisper", UsedBytes: uint64(gib(3))}},
	}
	attr := stubAttributor{4242: engine.Attribution{ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}
	svc := testService(t, snap, attr)

	claimed, err := svc.Claim(ctx, ClaimRequest{
		OwnerKind: engine.OwnerKindResource, OwnerID: "whisper",
		ResourceKind: engine.ResourceKindVRAM, PreferredBytes: gib(3), FloorBytes: gib(1),
		PriorityTier: "service",
	})
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	out, err := svc.Sweep(ctx)
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(out.Refreshed) != 1 || out.Refreshed[0].ClaimID != claimed.Claim.ClaimID {
		t.Fatalf("refreshed = %+v, want [%s]", out.Refreshed, claimed.Claim.ClaimID)
	}
	if len(out.Expired) != 0 {
		t.Fatalf("expired = %+v, want none", out.Expired)
	}
}

func TestServiceActivityAutoResolvesGeneration(t *testing.T) {
	ctx := context.Background()
	svc := testService(t, gpuSnapshot(16, 4), nil)

	claimed, err := svc.Claim(ctx, ClaimRequest{
		OwnerKind: engine.OwnerKindResource, OwnerID: "whisper",
		ResourceKind: engine.ResourceKindVRAM, PreferredBytes: gib(3), FloorBytes: gib(1),
		PriorityTier: "interactive",
	})
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	// Report active WITHOUT supplying a generation — the service resolves the
	// current generation so the consumer never fights the concurrency guard.
	view, err := svc.Activity(ctx, Ref{ClaimID: claimed.Claim.ClaimID, State: engine.ActivityActive})
	if err != nil {
		t.Fatalf("Activity(active) error = %v", err)
	}
	if view.ActivityState != engine.ActivityActive {
		t.Errorf("activity = %q, want active", view.ActivityState)
	}
	// Interactive-tier active auto-sets protected.
	if !view.Protected {
		t.Error("interactive claim should be protected while active")
	}

	// A second report (after the generation bumped) still succeeds without a
	// supplied generation.
	idle, err := svc.Activity(ctx, Ref{ClaimID: claimed.Claim.ClaimID, State: engine.ActivityIdle})
	if err != nil {
		t.Fatalf("Activity(idle) error = %v", err)
	}
	if idle.ActivityState != engine.ActivityIdle || idle.Protected {
		t.Errorf("after idle: activity=%q protected=%v, want idle/false", idle.ActivityState, idle.Protected)
	}
}

type stubAttributor map[int]engine.Attribution

func (s stubAttributor) Attribute(_ context.Context, pid int) engine.Attribution {
	if a, ok := s[pid]; ok {
		a.PID = pid
		return a
	}
	return engine.Attribution{PID: pid, OwnerID: engine.OwnerUnknown}
}
