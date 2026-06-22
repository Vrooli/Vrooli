package capacity

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFixedClock(t time.Time) *fixedClock { return &fixedClock{now: t.UTC()} }

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestStore(t *testing.T, clk Clock) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(context.Background(), Config{
		DBPath: filepath.Join(t.TempDir(), "capacity.db"),
		Clock:  clk,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func gpu(i int) *int { return &i }

func sampleClaim() CapacityClaim {
	return CapacityClaim{
		OwnerKind:      OwnerKindResource,
		OwnerID:        "whisper",
		ResourceKind:   ResourceKindVRAM,
		GPUIndex:       gpu(0),
		AmountBytes:    7 << 30,
		PreferredBytes: 7 << 30,
		FloorBytes:     1 << 30,
		Priority:       PriorityService,
	}
}

func TestCreateClaimDefaultsAndRead(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateClaim(ctx, sampleClaim(), 0)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	if created.ClaimID == "" {
		t.Fatal("expected generated claim id")
	}
	if created.Status != StatusGranted {
		t.Errorf("status = %q, want granted", created.Status)
	}
	if created.ActivityState != ActivityIdle {
		t.Errorf("activity = %q, want idle", created.ActivityState)
	}
	if created.Generation != 1 {
		t.Errorf("generation = %d, want 1", created.Generation)
	}
	if created.HeartbeatDeadlineAt == nil || !created.HeartbeatDeadlineAt.Equal(clk.Now().Add(DefaultHeartbeatTTL)) {
		t.Errorf("heartbeat deadline = %v, want %v", created.HeartbeatDeadlineAt, clk.Now().Add(DefaultHeartbeatTTL))
	}

	got, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.OwnerID != "whisper" || got.AmountBytes != 7<<30 || *got.GPUIndex != 0 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestHeartbeatRenewsLivenessWithoutBumpingGeneration(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateClaim(ctx, sampleClaim(), 10*time.Second)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	clk.Advance(5 * time.Second)
	hb, err := store.HeartbeatClaim(ctx, created.ClaimID, created.Generation, 10*time.Second)
	if err != nil {
		t.Fatalf("HeartbeatClaim() error = %v", err)
	}
	if hb.Generation != created.Generation {
		t.Errorf("heartbeat bumped generation %d -> %d; want stable", created.Generation, hb.Generation)
	}
	want := clk.Now().Add(10 * time.Second)
	if hb.HeartbeatDeadlineAt == nil || !hb.HeartbeatDeadlineAt.Equal(want) {
		t.Errorf("deadline = %v, want %v", hb.HeartbeatDeadlineAt, want)
	}
}

func TestReportActivityBumpsGenerationAndAutoProtectsInteractive(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	claim := sampleClaim()
	claim.OwnerID = "agent-manager"
	claim.Priority = PriorityInteractive
	created, err := store.CreateClaim(ctx, claim, 0)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	if created.Protected {
		t.Fatal("idle interactive claim should not start protected")
	}

	active, err := store.ReportActivity(ctx, created.ClaimID, created.Generation, ActivityActive)
	if err != nil {
		t.Fatalf("ReportActivity(active) error = %v", err)
	}
	if !active.Protected {
		t.Error("active interactive claim must be auto-protected")
	}
	if active.Generation != created.Generation+1 {
		t.Errorf("activity should bump generation %d -> %d", created.Generation, active.Generation)
	}
	if active.LastActiveAt == nil {
		t.Error("active should stamp last_active_at")
	}

	// A heartbeat with the now-stale generation must fail, signaling re-read.
	if _, err := store.HeartbeatClaim(ctx, created.ClaimID, created.Generation, 0); !errors.Is(err, ErrStaleGeneration) {
		t.Errorf("stale heartbeat error = %v, want ErrStaleGeneration", err)
	}

	idle, err := store.ReportActivity(ctx, active.ClaimID, active.Generation, ActivityIdle)
	if err != nil {
		t.Fatalf("ReportActivity(idle) error = %v", err)
	}
	if idle.Protected {
		t.Error("idle interactive claim must drop auto-protection")
	}
}

func TestDegradeClaimSetsAmountAndStatus(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateClaim(ctx, sampleClaim(), 0)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	degraded, err := store.DegradeClaim(ctx, created.ClaimID, created.Generation, "medium", 2<<30)
	if err != nil {
		t.Fatalf("DegradeClaim() error = %v", err)
	}
	if degraded.Status != StatusDegraded || degraded.AmountBytes != 2<<30 {
		t.Errorf("degraded = {%s, %d}, want {degraded, %d}", degraded.Status, degraded.AmountBytes, int64(2<<30))
	}
	if degraded.Generation != created.Generation+1 {
		t.Errorf("degrade should bump generation")
	}
	// Stale-generation degrade must fail.
	if _, err := store.DegradeClaim(ctx, created.ClaimID, created.Generation, "small", 1<<30); !errors.Is(err, ErrStaleGeneration) {
		t.Errorf("stale degrade error = %v, want ErrStaleGeneration", err)
	}
}

func TestReleaseAndPreemptAreTerminal(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	a, _ := store.CreateClaim(ctx, sampleClaim(), 0)
	released, err := store.ReleaseClaim(ctx, a.ClaimID)
	if err != nil {
		t.Fatalf("ReleaseClaim() error = %v", err)
	}
	if released.Status != StatusReleased {
		t.Errorf("status = %q, want released", released.Status)
	}
	// Heartbeating a released claim fails (not in active statuses).
	if _, err := store.HeartbeatClaim(ctx, a.ClaimID, released.Generation, 0); !errors.Is(err, ErrStaleGeneration) {
		t.Errorf("heartbeat on released = %v, want ErrStaleGeneration", err)
	}

	b, _ := store.CreateClaim(ctx, sampleClaim(), 0)
	preempted, err := store.PreemptClaim(ctx, b.ClaimID, "higher priority workload")
	if err != nil {
		t.Fatalf("PreemptClaim() error = %v", err)
	}
	if preempted.Status != StatusPreempted {
		t.Errorf("status = %q, want preempted", preempted.Status)
	}
}

func TestExpireStaleClaimsSweep(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	live, _ := store.CreateClaim(ctx, sampleClaim(), 60*time.Second)
	stale, _ := store.CreateClaim(ctx, sampleClaim(), 10*time.Second)

	clk.Advance(30 * time.Second)
	expired, err := store.ExpireStaleClaims(ctx, clk.Now())
	if err != nil {
		t.Fatalf("ExpireStaleClaims() error = %v", err)
	}
	if len(expired) != 1 || expired[0].ClaimID != stale.ClaimID {
		t.Fatalf("expired = %+v, want only %s", expired, stale.ClaimID)
	}

	got, _ := store.GetClaim(ctx, stale.ClaimID)
	if got.Status != StatusExpired {
		t.Errorf("stale claim status = %q, want expired", got.Status)
	}
	gotLive, _ := store.GetClaim(ctx, live.ClaimID)
	if gotLive.Status != StatusGranted {
		t.Errorf("live claim status = %q, want still granted", gotLive.Status)
	}
}

func TestListClaimsFilter(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	w := sampleClaim()
	w.OwnerID = "whisper"
	o := sampleClaim()
	o.OwnerID = "ollama"
	if _, err := store.CreateClaim(ctx, w, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateClaim(ctx, o, 0); err != nil {
		t.Fatal(err)
	}

	all, err := store.ListClaims(ctx, ClaimFilter{ResourceKind: ResourceKindVRAM, Statuses: ActiveClaimStatuses()})
	if err != nil {
		t.Fatalf("ListClaims() error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("len(all) = %d, want 2", len(all))
	}
	only, err := store.ListClaims(ctx, ClaimFilter{OwnerID: "ollama"})
	if err != nil {
		t.Fatalf("ListClaims(owner) error = %v", err)
	}
	if len(only) != 1 || only[0].OwnerID != "ollama" {
		t.Errorf("owner filter = %+v, want only ollama", only)
	}
}

func TestProfileRoundTrip(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	claim := sampleClaim()
	claim.DegradeProfile = &DegradeProfile{
		Steps: []DegradeStep{
			{Label: "large-v3", AmountBytes: 7 << 30},
			{Label: "medium", AmountBytes: 2 << 30},
			{Label: "small", AmountBytes: 1 << 30},
		},
		Apply:   DegradeApply{Verb: "capacity-degrade", Argv: []string{"--to", "{label}"}},
		Upshift: true,
	}
	created, err := store.CreateClaim(ctx, claim, 0)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	got, _ := store.GetClaim(ctx, created.ClaimID)
	if got.DegradeProfile == nil || len(got.DegradeProfile.Steps) != 3 {
		t.Fatalf("profile not round-tripped: %+v", got.DegradeProfile)
	}
	if got.DegradeProfile.Steps[1].Label != "medium" || !got.DegradeProfile.Upshift {
		t.Errorf("profile mismatch: %+v", got.DegradeProfile)
	}
}

func TestNotFound(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)))
	if _, err := store.GetClaim(ctx, "clm-missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetClaim(missing) = %v, want ErrNotFound", err)
	}
}

func TestPolicyGetSetRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)))

	def, err := store.GetPolicy(ctx)
	if err != nil {
		t.Fatalf("GetPolicy() error = %v", err)
	}
	if def.Enforce != EnforceAdvisory || def.PreemptEnabled {
		t.Errorf("default policy = %+v, want advisory + preempt off", def)
	}

	if _, err := store.SetPolicyKey(ctx, "idle_grace", "120s"); err != nil {
		t.Fatalf("SetPolicyKey(idle_grace) error = %v", err)
	}
	if _, err := store.SetPolicyKey(ctx, "enforce", "on"); err != nil {
		t.Fatalf("SetPolicyKey(enforce) error = %v", err)
	}
	if _, err := store.SetPolicyKey(ctx, "auto_stop_allowlist", "image-tools, kyutai-stt"); err != nil {
		t.Fatalf("SetPolicyKey(allowlist) error = %v", err)
	}

	got, err := store.GetPolicy(ctx)
	if err != nil {
		t.Fatalf("GetPolicy() error = %v", err)
	}
	if got.IdleGrace != 120*time.Second {
		t.Errorf("idle_grace = %v, want 120s", got.IdleGrace)
	}
	if got.Enforce != EnforceOn {
		t.Errorf("enforce = %q, want on", got.Enforce)
	}
	if !got.IsAutoStopAllowed("image-tools") || got.IsAutoStopAllowed("whisper") {
		t.Errorf("allowlist = %v, want [image-tools kyutai-stt]", got.AutoStopAllowlist)
	}

	// Invalid values are rejected.
	if _, err := store.SetPolicyKey(ctx, "enforce", "maybe"); !errors.Is(err, ErrInvalidClaim) {
		t.Errorf("SetPolicyKey(invalid enforce) = %v, want ErrInvalidClaim", err)
	}
}

func TestConcurrentHeartbeatsAreSerializedSafely(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateClaim(ctx, sampleClaim(), 30*time.Second)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, hbErr := store.HeartbeatClaim(ctx, created.ClaimID, created.Generation, 30*time.Second); hbErr != nil {
				errCh <- hbErr
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent heartbeat error = %v", err)
	}
}
