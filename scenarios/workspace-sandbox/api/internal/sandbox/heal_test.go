package sandbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"

	"github.com/vrooli/api-core/schedule"
)

// --- helpers ---

func newHealTestService(drv *mocks.FakeDriver, repo *mocks.FakeRepository) *Service {
	clk := schedule.System()
	return NewService(repo, drv, ServiceConfig{
		DefaultProjectRoot: "/tmp/project",
	}, clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter())
}

func activeSandbox(id uuid.UUID, lastUsed time.Time) *types.Sandbox {
	return &types.Sandbox{
		ID:         id,
		Status:     types.StatusActive,
		LastUsedAt: lastUsed,
		CreatedAt:  lastUsed.Add(-1 * time.Hour),
		ScopePath:  "/tmp/scope",
		MergedDir:  "/tmp/merged",
		UpperDir:   "/tmp/upper",
		WorkDir:    "/tmp/work",
		LowerDir:   "/tmp/lower",
	}
}

// --- ReconcileActiveMounts tests ---

func TestReconcileActiveMounts_StaleMount_AutoHeals(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.VerifyMountErr = errors.New("merged directory is not mounted (may be stale)")
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	tracker := newHealTracker()
	cfg := DefaultHealConfig()
	cfg.IdleGracePeriod = 0

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)

	// Sandbox should have been stopped then started (healed).
	healed, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("expected sandbox to exist after heal, got: %v", err)
	}
	if healed.Status != types.StatusActive {
		t.Errorf("expected StatusActive after heal, got %s", healed.Status)
	}

	// Tracker should be clean.
	if state := tracker.get(id); state != nil {
		t.Errorf("expected tracker to be reset after successful heal, got %+v", state)
	}
}

func TestReconcileActiveMounts_HealthyMount_Skipped(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	// verifyMountErr is nil — mount is healthy
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	tracker := newHealTracker()
	cfg := DefaultHealConfig()

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)

	got, _ := svc.Get(context.Background(), id)
	if got.Status != types.StatusActive {
		t.Errorf("expected sandbox to remain active, got %s", got.Status)
	}
}

func TestReconcileActiveMounts_StoppedSandbox_Skipped(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.VerifyMountErr = errors.New("stale")
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	tracker := newHealTracker()
	cfg := DefaultHealConfig()

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	sb.Status = types.StatusStopped
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)

	got, _ := svc.Get(context.Background(), id)
	if got.Status != types.StatusStopped {
		t.Errorf("expected stopped sandbox to remain stopped, got %s", got.Status)
	}
}

func TestReconcileActiveMounts_RecentlyUsed_Skipped(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.VerifyMountErr = errors.New("stale")
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	tracker := newHealTracker()
	cfg := DefaultHealConfig()
	cfg.IdleGracePeriod = 10 * time.Minute

	// LastUsedAt is 1 second ago — within grace period.
	sb := activeSandbox(id, time.Now().Add(-1*time.Second))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)

	// Should remain active without being Stop+Start'd.
	got, _ := svc.Get(context.Background(), id)
	if got.Status != types.StatusActive {
		t.Errorf("expected recently-used sandbox to remain active, got %s", got.Status)
	}
}

func TestReconcileActiveMounts_BackoffRespected(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.VerifyMountErr = errors.New("stale")
	drv.MountErr = errors.New("mount failed") // Ensure heal fails
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	tracker := newHealTracker()
	cfg := DefaultHealConfig()
	cfg.IdleGracePeriod = 0
	cfg.BaseBackoff = 1 * time.Hour // Very long backoff

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	// First attempt: should try and fail.
	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)
	state := tracker.get(id)
	if state == nil || state.consecutiveFailures != 1 {
		t.Fatal("expected 1 failure after first attempt")
	}

	// Reset the sandbox back to Active for second attempt.
	sb2, _ := svc.Get(context.Background(), id)
	sb2.Status = types.StatusActive
	_ = repo.Update(context.Background(), sb2)

	// Second attempt immediately: should be skipped due to backoff.
	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)
	state = tracker.get(id)
	if state.consecutiveFailures != 1 {
		t.Errorf("expected failures to remain 1 during backoff, got %d", state.consecutiveFailures)
	}
}

func TestReconcileActiveMounts_MaxFailures_SetsError(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.VerifyMountErr = errors.New("stale")
	drv.MountErr = errors.New("mount failed")
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	tracker := newHealTracker()
	cfg := DefaultHealConfig()
	cfg.IdleGracePeriod = 0
	cfg.MaxConsecutiveFailures = 2
	cfg.BaseBackoff = 0 // No backoff for this test

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	// First failure
	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)
	// Reset to Active for next attempt
	current, _ := svc.Get(context.Background(), id)
	current.Status = types.StatusActive
	_ = repo.Update(context.Background(), current)

	// Second failure — should hit max and transition to Error
	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)

	got, _ := svc.Get(context.Background(), id)
	if got.Status != types.StatusError {
		t.Errorf("expected StatusError after max failures, got %s", got.Status)
	}
	if got.ErrorMsg == "" {
		t.Error("expected ErrorMsg to be set after max failures")
	}
}

func TestReconcileActiveMounts_DeletedDuringHeal_NoError(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.VerifyMountErr = errors.New("stale")
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	tracker := newHealTracker()
	cfg := DefaultHealConfig()
	cfg.IdleGracePeriod = 0

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	// Delete sandbox before heal runs, simulating concurrent deletion.
	delete(repo.Sandboxes, id)

	// Should not panic.
	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)
}

func TestReconcileActiveMounts_AlreadyFixed_NoOp(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.VerifyMountErr = errors.New("stale")
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	tracker := newHealTracker()
	cfg := DefaultHealConfig()
	cfg.IdleGracePeriod = 0

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	// Clear stale error after VerifyMountIntegrity but before Stop.
	// Since Stop on Active is idempotent and Start on Active is no-op,
	// this should complete successfully.
	svc.ReconcileActiveMounts(context.Background(), tracker, cfg)

	got, _ := svc.Get(context.Background(), id)
	if got.Status != types.StatusActive {
		t.Errorf("expected StatusActive, got %s", got.Status)
	}
}

// --- healSandbox tests ---

func TestHealSandbox_StopFails(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.UnmountErr = errors.New("unmount failed")
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	err := svc.healSandbox(context.Background(), id)
	if err == nil {
		t.Fatal("expected error from healSandbox when stop fails")
	}
}

func TestHealSandbox_StartFails(t *testing.T) {
	id := uuid.New()
	drv := mocks.NewFakeDriver()
	drv.MountErr = errors.New("mount failed")
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)

	sb := activeSandbox(id, time.Now().Add(-5*time.Minute))
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatal(err)
	}

	err := svc.healSandbox(context.Background(), id)
	if err == nil {
		t.Fatal("expected error from healSandbox when start fails")
	}
}

// --- isEligibleForHeal tests ---

func TestIsEligibleForHeal_Table(t *testing.T) {
	now := time.Now()
	cfg := HealConfig{
		IdleGracePeriod:        30 * time.Second,
		MaxConsecutiveFailures: 5,
		BaseBackoff:            30 * time.Second,
	}

	tests := []struct {
		name     string
		sandbox  *types.Sandbox
		state    *healState
		cfg      HealConfig
		want     bool
		wantDesc string
	}{
		{
			name:    "nil state, idle sandbox",
			sandbox: &types.Sandbox{LastUsedAt: now.Add(-5 * time.Minute)},
			state:   nil,
			cfg:     cfg,
			want:    true,
		},
		{
			name:     "recently used",
			sandbox:  &types.Sandbox{LastUsedAt: now.Add(-5 * time.Second)},
			state:    nil,
			cfg:      cfg,
			want:     false,
			wantDesc: "recently used",
		},
		{
			name:    "zero grace period always eligible",
			sandbox: &types.Sandbox{LastUsedAt: now},
			state:   nil,
			cfg:     HealConfig{IdleGracePeriod: 0, MaxConsecutiveFailures: 5, BaseBackoff: 30 * time.Second},
			want:    true,
		},
		{
			name:     "max failures exceeded",
			sandbox:  &types.Sandbox{LastUsedAt: now.Add(-5 * time.Minute)},
			state:    &healState{consecutiveFailures: 5},
			cfg:      cfg,
			want:     false,
			wantDesc: "max failures exceeded",
		},
		{
			name:     "backing off",
			sandbox:  &types.Sandbox{LastUsedAt: now.Add(-5 * time.Minute)},
			state:    &healState{consecutiveFailures: 2, lastAttempt: now.Add(-10 * time.Second)},
			cfg:      cfg,
			want:     false,
			wantDesc: "backing off",
		},
		{
			name:    "backoff elapsed",
			sandbox: &types.Sandbox{LastUsedAt: now.Add(-5 * time.Minute)},
			state:   &healState{consecutiveFailures: 1, lastAttempt: now.Add(-2 * time.Minute)},
			cfg:     cfg,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := isEligibleForHeal(tt.sandbox, tt.state, tt.cfg, now)
			if got != tt.want {
				t.Errorf("isEligibleForHeal() = %v, want %v (reason: %q)", got, tt.want, reason)
			}
			if tt.wantDesc != "" && reason != tt.wantDesc {
				t.Errorf("reason = %q, want %q", reason, tt.wantDesc)
			}
		})
	}
}

// --- backoffDuration tests ---

func TestBackoffDuration(t *testing.T) {
	base := 30 * time.Second
	tests := []struct {
		failures int
		want     time.Duration
	}{
		{0, 0},
		{1, 30 * time.Second},
		{2, 60 * time.Second},
		{3, 120 * time.Second},
		{4, 240 * time.Second},
		{5, 480 * time.Second},
		{100, maxBackoff}, // Should be capped at 1h
	}

	for _, tt := range tests {
		got := backoffDuration(tt.failures, base)
		if got != tt.want {
			t.Errorf("backoffDuration(%d, %v) = %v, want %v", tt.failures, base, got, tt.want)
		}
	}
}

// --- Verify heal test helper uses mock correctly ---

func TestHealTestService_CreatesValidService(t *testing.T) {
	drv := mocks.NewFakeDriver()
	repo := mocks.NewFakeRepository()
	svc := newHealTestService(drv, repo)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	// Ensure the driver and repo are wired.
	var _ driver.Driver = drv
	_, err := svc.List(context.Background(), &types.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("expected list to succeed, got: %v", err)
	}
}
