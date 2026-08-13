package checks

import (
	"context"
	"testing"
	"time"
)

func TestNewAutoHealPolicyFromGlobal(t *testing.T) {
	policy, err := NewAutoHealPolicyFromGlobal(300, 3, 30, 300, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.BaseCooldown != 5*time.Minute {
		t.Fatalf("BaseCooldown = %v, want %v", policy.BaseCooldown, 5*time.Minute)
	}
	if policy.MaxRestartAttempts != 3 {
		t.Fatalf("MaxRestartAttempts = %d, want 3", policy.MaxRestartAttempts)
	}
	if policy.FastActionTimeout != 30*time.Second {
		t.Fatalf("FastActionTimeout = %v, want %v", policy.FastActionTimeout, 30*time.Second)
	}
	if policy.RestartActionTimeout != 5*time.Minute {
		t.Fatalf("RestartActionTimeout = %v, want %v", policy.RestartActionTimeout, 5*time.Minute)
	}
	if policy.TimeoutRetryCooldown != 30*time.Second {
		t.Fatalf("TimeoutRetryCooldown = %v, want %v", policy.TimeoutRetryCooldown, 30*time.Second)
	}
}

func TestNewAutoHealPolicyFromGlobal_AppliesTimeoutDefaults(t *testing.T) {
	policy, err := NewAutoHealPolicyFromGlobal(300, 3, 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.FastActionTimeout != DefaultFastActionTimeout {
		t.Fatalf("FastActionTimeout = %v, want %v", policy.FastActionTimeout, DefaultFastActionTimeout)
	}
	if policy.RestartActionTimeout != DefaultRestartActionTimeout {
		t.Fatalf("RestartActionTimeout = %v, want %v", policy.RestartActionTimeout, DefaultRestartActionTimeout)
	}
	if policy.TimeoutRetryCooldown != DefaultTimeoutRetryCooldown {
		t.Fatalf("TimeoutRetryCooldown = %v, want %v", policy.TimeoutRetryCooldown, DefaultTimeoutRetryCooldown)
	}
}

func TestNewAutoHealPolicyFromGlobal_InvalidValues(t *testing.T) {
	if _, err := NewAutoHealPolicyFromGlobal(0, 3, 30, 300, 30); err == nil {
		t.Fatal("expected error for zero restart cooldown")
	}
	if _, err := NewAutoHealPolicyFromGlobal(60, 0, 30, 300, 30); err == nil {
		t.Fatal("expected error for max restart attempts < 1")
	}
}

func TestSetAutoHealPolicy_RejectsInvalidPolicy(t *testing.T) {
	reg := NewRegistry(testPlatform())
	err := reg.SetAutoHealPolicy(AutoHealPolicy{
		BaseCooldown:       0,
		MaxRestartAttempts: 1,
	})
	if err == nil {
		t.Fatal("expected invalid policy to be rejected")
	}
}

func TestCalculateFailureCooldown_CapsAtMaxFailureCooldown(t *testing.T) {
	policy := AutoHealPolicy{
		BaseCooldown:         5 * time.Minute,
		MaxRestartAttempts:   3,
		FastActionTimeout:    DefaultFastActionTimeout,
		RestartActionTimeout: DefaultRestartActionTimeout,
		TimeoutRetryCooldown: DefaultTimeoutRetryCooldown,
	}
	if err := policy.Validate(); err != nil {
		t.Fatalf("policy invalid: %v", err)
	}

	// Below threshold: BaseCooldown (capped).
	if got := policy.CalculateFailureCooldown(1); got != policy.BaseCooldown {
		t.Errorf("failure=1: got %v, want %v", got, policy.BaseCooldown)
	}

	// At threshold (3): 5m * 2^1 = 10m.
	if got := policy.CalculateFailureCooldown(3); got != 10*time.Minute {
		t.Errorf("failure=3: got %v, want 10m", got)
	}

	// Past the cap: must saturate at MaxFailureCooldown.
	for _, failures := range []int{20, 50, 1000} {
		if got := policy.CalculateFailureCooldown(failures); got != MaxFailureCooldown {
			t.Errorf("failure=%d: got %v, want %v (cap)", failures, got, MaxFailureCooldown)
		}
	}

	// Extreme value must not overflow into negative durations.
	if got := policy.CalculateFailureCooldown(1 << 30); got != MaxFailureCooldown {
		t.Errorf("very large failures: got %v, want %v", got, MaxFailureCooldown)
	}
}

func TestHealTrackerCooldownHelpers(t *testing.T) {
	now := time.Now()
	tracker := HealTracker{
		CooldownUntil: now.Add(2 * time.Second),
	}

	if !tracker.IsInCooldown() {
		t.Fatal("expected tracker to be in cooldown")
	}
	if tracker.CooldownRemaining() <= 0 {
		t.Fatal("expected positive cooldown remaining")
	}

	tracker.CooldownUntil = now.Add(-1 * time.Second)
	if tracker.IsInCooldown() {
		t.Fatal("expected tracker cooldown to be expired")
	}
	if tracker.CooldownRemaining() != 0 {
		t.Fatalf("expected zero cooldown remaining, got %v", tracker.CooldownRemaining())
	}
}

func TestRunAutoHeal_RequiresPolicy(t *testing.T) {
	reg := NewRegistry(testPlatform())
	check := &mockHealableCheck{
		id: "healable-check",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"healable-check": true},
	})

	results := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Attempted {
		t.Fatal("expected no attempt without policy")
	}
	if results[0].Reason != "auto-heal policy not configured" {
		t.Fatalf("unexpected reason: %s", results[0].Reason)
	}
}

func TestAutoHealCooldown_UsesConfiguredPolicy(t *testing.T) {
	reg := NewRegistry(testPlatform())
	if err := reg.SetAutoHealPolicy(AutoHealPolicy{
		BaseCooldown:         5 * time.Minute,
		MaxRestartAttempts:   3,
		FastActionTimeout:    DefaultFastActionTimeout,
		RestartActionTimeout: DefaultRestartActionTimeout,
		TimeoutRetryCooldown: DefaultTimeoutRetryCooldown,
	}); err != nil {
		t.Fatalf("set policy: %v", err)
	}

	clk := &fixedClock{current: time.Date(2026, 2, 19, 2, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "healable-check",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"healable-check": true},
	})

	first := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	})
	if len(first) != 1 || !first[0].Attempted {
		t.Fatalf("expected first attempt to run, got %+v", first)
	}

	clk.current = clk.current.Add(4 * time.Minute)
	second := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	})
	if len(second) != 1 {
		t.Fatalf("expected second result, got %d", len(second))
	}
	if second[0].Attempted {
		t.Fatalf("expected cooldown skip, got attempt: %+v", second[0])
	}
	if second[0].CooldownRemaining <= 0 {
		t.Fatalf("expected positive cooldown remaining, got %v", second[0].CooldownRemaining)
	}
}

func TestAutoHealBackoff_UsesConfiguredMaxRestartAttempts(t *testing.T) {
	reg := NewRegistry(testPlatform())
	if err := reg.SetAutoHealPolicy(AutoHealPolicy{
		BaseCooldown:         60 * time.Second,
		MaxRestartAttempts:   2,
		FastActionTimeout:    DefaultFastActionTimeout,
		RestartActionTimeout: DefaultRestartActionTimeout,
		TimeoutRetryCooldown: DefaultTimeoutRetryCooldown,
	}); err != nil {
		t.Fatalf("set policy: %v", err)
	}

	clk := &fixedClock{current: time.Date(2026, 2, 19, 2, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "failing-check",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: false, Error: "failed"},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"failing-check": true},
	})

	// Failure #1: base cooldown (60s).
	r1 := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "failing-check", Status: StatusCritical},
	})
	if len(r1) != 1 || !r1[0].Attempted {
		t.Fatalf("first failure should attempt heal, got %+v", r1)
	}

	clk.current = clk.current.Add(61 * time.Second)
	// Failure #2: threshold reached -> 120s cooldown.
	r2 := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "failing-check", Status: StatusCritical},
	})
	if len(r2) != 1 || !r2[0].Attempted {
		t.Fatalf("second failure should attempt heal, got %+v", r2)
	}

	tracker, ok := reg.GetHealTracker("failing-check")
	if !ok {
		t.Fatal("expected heal tracker")
	}
	if tracker.ConsecutiveFailures != 2 {
		t.Fatalf("ConsecutiveFailures = %d, want 2", tracker.ConsecutiveFailures)
	}
	if got := tracker.CooldownUntil.Sub(clk.current); got != 120*time.Second {
		t.Fatalf("cooldown after 2nd failure = %v, want 120s", got)
	}

	clk.current = clk.current.Add(121 * time.Second)
	// Failure #3: exponential growth -> 240s cooldown.
	r3 := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "failing-check", Status: StatusCritical},
	})
	if len(r3) != 1 || !r3[0].Attempted {
		t.Fatalf("third failure should attempt heal, got %+v", r3)
	}

	tracker, ok = reg.GetHealTracker("failing-check")
	if !ok {
		t.Fatal("expected heal tracker")
	}
	if tracker.ConsecutiveFailures != 3 {
		t.Fatalf("ConsecutiveFailures = %d, want 3", tracker.ConsecutiveFailures)
	}
	if got := tracker.CooldownUntil.Sub(clk.current); got != 240*time.Second {
		t.Fatalf("cooldown after 3rd failure = %v, want 240s", got)
	}
}

type mockHealTrackerStore struct {
	trackers map[string]*HealTracker
	saveCh   chan string
}

func (m *mockHealTrackerStore) SaveHealTracker(ctx context.Context, checkID string, tracker *HealTracker) error {
	if m.trackers == nil {
		m.trackers = make(map[string]*HealTracker)
	}
	copyTracker := *tracker
	m.trackers[checkID] = &copyTracker
	if m.saveCh != nil {
		select {
		case m.saveCh <- checkID:
		default:
		}
	}
	return nil
}

func (m *mockHealTrackerStore) GetAllHealTrackers(ctx context.Context) (map[string]*HealTracker, error) {
	out := make(map[string]*HealTracker, len(m.trackers))
	for k, v := range m.trackers {
		copyTracker := *v
		out[k] = &copyTracker
	}
	return out, nil
}

func (m *mockHealTrackerStore) DeleteHealTracker(ctx context.Context, checkID string) error {
	delete(m.trackers, checkID)
	return nil
}

func TestLoadHealTrackers_RestoresState(t *testing.T) {
	reg := newTestRegistry()
	store := &mockHealTrackerStore{
		trackers: map[string]*HealTracker{
			"resource-postgres": {
				ConsecutiveFailures: 4,
				CooldownUntil:       time.Now().Add(5 * time.Minute),
			},
		},
	}
	reg.SetHealTrackerStore(store)

	if err := reg.LoadHealTrackers(context.Background()); err != nil {
		t.Fatalf("LoadHealTrackers failed: %v", err)
	}

	tracker, ok := reg.GetHealTracker("resource-postgres")
	if !ok {
		t.Fatal("expected tracker to be loaded")
	}
	if tracker.ConsecutiveFailures != 4 {
		t.Fatalf("ConsecutiveFailures = %d, want 4", tracker.ConsecutiveFailures)
	}
}

func TestUpdateHealTracker_PersistsToStore(t *testing.T) {
	reg := newTestRegistry()
	clk := &fixedClock{current: time.Date(2026, 2, 19, 2, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	store := &mockHealTrackerStore{saveCh: make(chan string, 1)}
	reg.SetHealTrackerStore(store)

	check := &mockHealableCheck{
		id: "resource-postgres",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: false},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"resource-postgres": true},
	})

	_ = reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "resource-postgres", Status: StatusCritical},
	})

	select {
	case <-store.saveCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected heal tracker save to be called")
	}

	if _, ok := store.trackers["resource-postgres"]; !ok {
		t.Fatal("expected stored tracker for resource-postgres")
	}
}

// newTimeoutPolicyRegistry returns a registry pre-configured with short
// fast/restart timeouts and a short timeout-retry cooldown so tests can run
// quickly. BaseCooldown is also short so success cooldown doesn't dominate.
func newTimeoutPolicyRegistry(t *testing.T, fast, restart, timeoutRetry, base time.Duration) *Registry {
	t.Helper()
	reg := NewRegistry(testPlatform())
	policy := AutoHealPolicy{
		BaseCooldown:         base,
		MaxRestartAttempts:   3,
		FastActionTimeout:    fast,
		RestartActionTimeout: restart,
		TimeoutRetryCooldown: timeoutRetry,
	}
	if err := reg.SetAutoHealPolicy(policy); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	return reg
}

// TestActionTimeoutFor_RestartActionsGetRestartBudget verifies the timeout
// dispatcher hands restart-class actions the generous budget and everything
// else the short budget. This is the core of D1: a 90s outer cap killed a
// legitimately slow restart before verifyRecovery could observe success.
func TestActionTimeoutFor_RestartActionsGetRestartBudget(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 30*time.Second, 5*time.Minute, 30*time.Second, time.Minute)

	cases := []struct {
		action string
		want   time.Duration
	}{
		{"restart", 5 * time.Minute},
		{"restart-clean", 5 * time.Minute},
		{"setup-restart", 5 * time.Minute},
		{"start", 5 * time.Minute},
		{"stop", 5 * time.Minute},
		{"logs", 30 * time.Second},
		{"diagnose", 30 * time.Second},
		{"cleanup-ports", 30 * time.Second},
		{"kill", 30 * time.Second},
		{"unknown-future-action", 30 * time.Second},
	}
	for _, tc := range cases {
		if got := reg.actionTimeoutFor(tc.action); got != tc.want {
			t.Errorf("actionTimeoutFor(%q) = %v, want %v", tc.action, got, tc.want)
		}
	}
}

// TestExecuteAutoHealAction_RestartCompletesWithinExtendedBudget proves that
// a 200ms restart succeeds under a 1s restart budget but would have been killed
// by the old 100ms fast budget. The reverse-config (fast budget for a restart
// action) is verified in the timeout-not-failure test below.
func TestExecuteAutoHealAction_RestartCompletesWithinExtendedBudget(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 50*time.Millisecond, time.Second, 100*time.Millisecond, time.Minute)

	check := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true, Message: "restarted"},
		executeSleep:  200 * time.Millisecond,
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-app-monitor": true},
	})

	results := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "scenario-app-monitor", Status: StatusCritical},
	})

	if len(results) != 1 || !results[0].Attempted {
		t.Fatalf("expected one attempted result, got %+v", results)
	}
	if !results[0].ActionResult.Success {
		t.Fatalf("expected restart to succeed inside extended budget, got %+v", results[0].ActionResult)
	}
	if results[0].TimedOut || results[0].ActionResult.TimedOut {
		t.Fatalf("expected TimedOut=false, got %+v", results[0])
	}
	tracker, _ := reg.GetHealTracker("scenario-app-monitor")
	if tracker.ConsecutiveFailures != 0 {
		t.Fatalf("ConsecutiveFailures = %d, want 0", tracker.ConsecutiveFailures)
	}
}

// TestExecuteAutoHealAction_TimeoutDoesNotRatchetFailures is the heart of D2.
// Before the fix, a timed-out heal was indistinguishable from a failure and
// pushed ConsecutiveFailures into the exponential cooldown, silencing autoheal
// for hours. After the fix, a timeout records TotalTimeouts, applies the short
// retry cooldown, and leaves ConsecutiveFailures untouched.
func TestExecuteAutoHealAction_TimeoutDoesNotRatchetFailures(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 50*time.Millisecond, 50*time.Millisecond, 75*time.Millisecond, time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
		executeSleep:  500 * time.Millisecond, // far longer than 50ms restart budget
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-app-monitor": true},
	})

	results := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "scenario-app-monitor", Status: StatusCritical},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].TimedOut {
		t.Fatalf("expected TimedOut=true, got %+v", results[0])
	}
	if !results[0].ActionResult.TimedOut {
		t.Fatalf("expected ActionResult.TimedOut=true, got %+v", results[0].ActionResult)
	}
	if results[0].ActionResult.Success {
		t.Fatalf("expected Success=false on timeout")
	}

	tracker, ok := reg.GetHealTracker("scenario-app-monitor")
	if !ok {
		t.Fatal("expected tracker to exist")
	}
	if tracker.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0 (timeouts must not ratchet failures)", tracker.ConsecutiveFailures)
	}
	if tracker.TotalTimeouts != 1 {
		t.Errorf("TotalTimeouts = %d, want 1", tracker.TotalTimeouts)
	}
	expectedCooldownUntil := clk.current.Add(75 * time.Millisecond)
	if !tracker.CooldownUntil.Equal(expectedCooldownUntil) {
		t.Errorf("CooldownUntil = %v, want %v (short timeout-retry cooldown)", tracker.CooldownUntil, expectedCooldownUntil)
	}
}

// TestExecuteAutoHealAction_GenuineFailureStillRatchets verifies the failure
// path is unchanged when the action explicitly reports Success=false without
// timing out. ConsecutiveFailures must increment so the exponential cooldown
// still kicks in for genuinely-broken actions.
func TestExecuteAutoHealAction_GenuineFailureStillRatchets(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, time.Second, time.Second, 30*time.Second, 2*time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "resource-postgres",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: false, Error: "exit 1"},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"resource-postgres": true},
	})

	results := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "resource-postgres", Status: StatusCritical},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TimedOut {
		t.Fatalf("genuine failure must not be marked TimedOut")
	}
	tracker, _ := reg.GetHealTracker("resource-postgres")
	if tracker.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1", tracker.ConsecutiveFailures)
	}
	if tracker.TotalTimeouts != 0 {
		t.Errorf("TotalTimeouts = %d, want 0", tracker.TotalTimeouts)
	}
	expectedCooldownUntil := clk.current.Add(2 * time.Minute)
	if !tracker.CooldownUntil.Equal(expectedCooldownUntil) {
		t.Errorf("CooldownUntil = %v, want %v", tracker.CooldownUntil, expectedCooldownUntil)
	}
}

// TestUpdateHealTracker_ConsecutiveTimeoutCapFallsThroughToFailure verifies
// the safety cap: after MaxRestartAttempts consecutive timeouts, the next
// timeout falls through to the regular failure ratchet so a permanently-stuck
// action eventually cools down on the exponential backoff.
func TestUpdateHealTracker_ConsecutiveTimeoutCapFallsThroughToFailure(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, time.Second, time.Second, time.Minute, time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	// Drive MaxRestartAttempts=3 consecutive timeouts directly.
	for i := 0; i < 3; i++ {
		reg.updateHealTracker("scenario-x", outcomeTimeout)
	}
	tracker, _ := reg.GetHealTracker("scenario-x")
	if tracker.ConsecutiveFailures != 0 {
		t.Fatalf("after 3 timeouts ConsecutiveFailures = %d, want 0", tracker.ConsecutiveFailures)
	}
	if tracker.ConsecutiveTimeouts != 3 {
		t.Fatalf("ConsecutiveTimeouts = %d, want 3", tracker.ConsecutiveTimeouts)
	}

	// 4th consecutive timeout exceeds MaxRestartAttempts and falls through
	// to the failure ratchet.
	reg.updateHealTracker("scenario-x", outcomeTimeout)
	tracker, _ = reg.GetHealTracker("scenario-x")
	if tracker.ConsecutiveFailures != 1 {
		t.Errorf("after 4th consecutive timeout ConsecutiveFailures = %d, want 1", tracker.ConsecutiveFailures)
	}
}

// TestUpdateHealTracker_SuccessResetsBothCounters verifies success clears
// both consecutive counters so the tracker returns to a clean state.
func TestUpdateHealTracker_SuccessResetsBothCounters(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, time.Second, time.Second, 30*time.Second, time.Minute)

	reg.updateHealTracker("scenario-x", outcomeTimeout)
	reg.updateHealTracker("scenario-x", outcomeFailure)
	reg.updateHealTracker("scenario-x", outcomeSuccess)
	tracker, _ := reg.GetHealTracker("scenario-x")
	if tracker.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures after success = %d, want 0", tracker.ConsecutiveFailures)
	}
	if tracker.ConsecutiveTimeouts != 0 {
		t.Errorf("ConsecutiveTimeouts after success = %d, want 0", tracker.ConsecutiveTimeouts)
	}
	if tracker.TotalSuccesses != 1 || tracker.TotalTimeouts != 1 {
		t.Errorf("totals = (s=%d, t=%d), want (1, 1)", tracker.TotalSuccesses, tracker.TotalTimeouts)
	}
}

// TestAutoHeal_TimeoutThenSuccess_Recovers is the end-to-end D2 regression.
// Before the fix, a slow first attempt timed out, ratcheted failures, and the
// next tick was silenced by the exponential cooldown. After the fix the short
// timeout-retry cooldown lets the next attempt succeed.
func TestAutoHeal_TimeoutThenSuccess_Recovers(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 50*time.Millisecond, 50*time.Millisecond, 1*time.Millisecond, time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true, Message: "restarted"},
		executeSleep:  300 * time.Millisecond, // exceeds 50ms budget on first attempt
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-app-monitor": true},
	})

	results := []Result{{CheckID: "scenario-app-monitor", Status: StatusCritical}}

	// First attempt: times out (sleep > budget).
	first := reg.RunAutoHeal(context.Background(), results)
	if len(first) != 1 || !first[0].TimedOut {
		t.Fatalf("first attempt: expected TimedOut, got %+v", first)
	}

	// Advance clock past the 1ms timeout-retry cooldown so the cooldown gate
	// allows the next attempt.
	clk.current = clk.current.Add(10 * time.Millisecond)

	// Make the mock fast and successful for the second attempt.
	check.mu.Lock()
	check.executeSleep = 0
	check.mu.Unlock()

	second := reg.RunAutoHeal(context.Background(), results)
	if len(second) != 1 {
		t.Fatalf("second attempt result count = %d", len(second))
	}
	if !second[0].Attempted {
		t.Fatalf("second attempt should be attempted, got reason=%q (cooldown=%v)", second[0].Reason, second[0].CooldownRemaining)
	}
	if !second[0].ActionResult.Success {
		t.Fatalf("second attempt should succeed, got %+v", second[0].ActionResult)
	}

	tracker, _ := reg.GetHealTracker("scenario-app-monitor")
	if tracker.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures after timeout-then-success = %d, want 0", tracker.ConsecutiveFailures)
	}
	if tracker.TotalTimeouts != 1 || tracker.TotalSuccesses != 1 {
		t.Errorf("totals = (t=%d, s=%d), want (1, 1)", tracker.TotalTimeouts, tracker.TotalSuccesses)
	}
}

// TestExecuteAutoHealAction_HealTrackerPersistsTimeoutCounters checks that the
// persistence seam writes a tracker carrying the new TotalTimeouts counter so
// the on-disk record matches in-memory state. This guards against future code
// inadvertently dropping the new fields.
func TestExecuteAutoHealAction_HealTrackerPersistsTimeoutCounters(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 50*time.Millisecond, 50*time.Millisecond, 30*time.Second, time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)
	store := &mockHealTrackerStore{saveCh: make(chan string, 1)}
	reg.SetHealTrackerStore(store)

	check := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
		executeSleep:  500 * time.Millisecond,
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-app-monitor": true},
	})

	_ = reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "scenario-app-monitor", Status: StatusCritical},
	})

	select {
	case <-store.saveCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected persistence save")
	}

	persisted := store.trackers["scenario-app-monitor"]
	if persisted == nil {
		t.Fatal("expected persisted tracker")
	}
	if persisted.TotalTimeouts != 1 {
		t.Errorf("persisted TotalTimeouts = %d, want 1", persisted.TotalTimeouts)
	}
	if persisted.ConsecutiveFailures != 0 {
		t.Errorf("persisted ConsecutiveFailures = %d, want 0", persisted.ConsecutiveFailures)
	}
}
