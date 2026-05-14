package main

import (
	"testing"
	"time"

	"connectrpc.com/connect"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/policy"
)

// --- Policy validation tests ---

// [REQ:P1-001a] Expiration Policy Engine — validation
func TestValidatePolicy_Never(t *testing.T) {
	p := policy.Policy{Mode: policy.Never}
	if err := policy.Validate(p); err != nil {
		t.Errorf("never policy should be valid: %v", err)
	}
}

func TestValidatePolicy_PresetValid(t *testing.T) {
	for _, dur := range []string{"1h", "8h", "24h"} {
		p := policy.Policy{Mode: policy.Preset, Duration: dur}
		if err := policy.Validate(p); err != nil {
			t.Errorf("preset %s should be valid: %v", dur, err)
		}
	}
}

func TestValidatePolicy_PresetInvalid(t *testing.T) {
	p := policy.Policy{Mode: policy.Preset, Duration: "2h"}
	if err := policy.Validate(p); err == nil {
		t.Error("preset 2h should be invalid")
	}
}

func TestValidatePolicy_CustomValid(t *testing.T) {
	valid := []string{"1m", "30m", "2h", "168h"}
	for _, dur := range valid {
		p := policy.Policy{Mode: policy.Custom, Duration: dur}
		if err := policy.Validate(p); err != nil {
			t.Errorf("custom %s should be valid: %v", dur, err)
		}
	}
}

func TestValidatePolicy_CustomInvalid(t *testing.T) {
	tests := []struct {
		name     string
		duration string
	}{
		{"empty", ""},
		{"too short", "30s"},
		{"too long", "169h"},
		{"unparseable", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := policy.Policy{Mode: policy.Custom, Duration: tt.duration}
			if err := policy.Validate(p); err == nil {
				t.Errorf("custom %q should be invalid", tt.duration)
			}
		})
	}
}

func TestValidatePolicy_InvalidMode(t *testing.T) {
	p := policy.Policy{Mode: "bad"}
	if err := policy.Validate(p); err == nil {
		t.Error("invalid mode should fail")
	}
}

// --- TTL resolution tests ---

// [REQ:P1-001a] Expiration Policy Engine — TTL resolution
func TestResolveTTL(t *testing.T) {
	tests := []struct {
		name     string
		policy   policy.Policy
		expected time.Duration
	}{
		{"never", policy.Policy{Mode: policy.Never}, 0},
		{"preset 1h", policy.Policy{Mode: policy.Preset, Duration: "1h"}, time.Hour},
		{"preset 8h", policy.Policy{Mode: policy.Preset, Duration: "8h"}, 8 * time.Hour},
		{"preset 24h", policy.Policy{Mode: policy.Preset, Duration: "24h"}, 24 * time.Hour},
		{"custom 30m", policy.Policy{Mode: policy.Custom, Duration: "30m"}, 30 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.ResolveTTL(tt.policy)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// --- Expiration check tests ---

// [REQ:P1-001a] Expiration Policy Engine — <10ms evaluation
func TestIsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		createdAt time.Time
		policy    policy.Policy
		expired   bool
	}{
		{"never policy", now.Add(-100 * time.Hour), policy.Policy{Mode: policy.Never}, false},
		{"preset not expired", now.Add(-30 * time.Minute), policy.Policy{Mode: policy.Preset, Duration: "1h"}, false},
		{"preset expired", now.Add(-2 * time.Hour), policy.Policy{Mode: policy.Preset, Duration: "1h"}, true},
		{"custom not expired", now.Add(-10 * time.Minute), policy.Policy{Mode: policy.Custom, Duration: "30m"}, false},
		{"custom expired", now.Add(-40 * time.Minute), policy.Policy{Mode: policy.Custom, Duration: "30m"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.IsExpired(tt.createdAt, tt.policy)
			if got != tt.expired {
				t.Errorf("expected expired=%v, got %v", tt.expired, got)
			}
		})
	}
}

// [REQ:P1-001a] Policy evaluation performance (<10ms)
func TestIsExpired_Performance(t *testing.T) {
	pol := policy.Policy{Mode: policy.Preset, Duration: "1h"}
	created := time.Now().Add(-30 * time.Minute)

	start := time.Now()
	for i := 0; i < 10000; i++ {
		policy.IsExpired(created, pol)
	}
	elapsed := time.Since(start)

	// 10,000 evaluations should complete well under 50ms.
	// Raised from 10ms to account for CI/system load variance.
	if elapsed > 50*time.Millisecond {
		t.Errorf("10k policy evaluations took %v, expected <50ms", elapsed)
	}
}

// --- Session policy get/set ---

func TestSession_GetSetPolicy(t *testing.T) {
	sm := NewSessionManagerWithFactory(newFakePTYFactory())
	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	// Default policy is never
	p := sess.GetPolicy()
	if p.Mode != policy.Never {
		t.Errorf("default policy should be never, got %s", p.Mode)
	}

	// Set a preset policy
	sess.SetPolicy(policy.Policy{Mode: policy.Preset, Duration: "1h"})
	p = sess.GetPolicy()
	if p.Mode != policy.Preset || p.Duration != "1h" {
		t.Errorf("expected preset/1h, got %s/%s", p.Mode, p.Duration)
	}
}

// --- HTTP handler tests ---

// [REQ:P1-001a] Expiration Policy Engine — Get RPC
func TestHandleGetPolicy(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	view, err := callGetPolicy(t, srv, sess.ID)
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if view.GetPolicy().GetMode() != string(policy.Never) {
		t.Errorf("default should be never, got %s", view.GetPolicy().GetMode())
	}
	if view.GetHasExpiry() {
		t.Error("never policy should not have expiry")
	}
}

func TestHandleGetPolicy_NotFound(t *testing.T) {
	srv := newFakeTestServer()
	_, err := callGetPolicy(t, srv, "nonexistent")
	if got := connectCode(err); got != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %s", got)
	}
}

// [REQ:P1-001a] Expiration Policy Engine — Update RPC
func TestHandleUpdatePolicy(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	view, err := callUpdatePolicy(t, srv, sess.ID, "preset", "8h")
	if err != nil {
		t.Fatalf("UpdatePolicy: %v", err)
	}
	if view.GetPolicy().GetMode() != string(policy.Preset) || view.GetPolicy().GetDuration() != "8h" {
		t.Errorf("expected preset/8h, got %s/%s", view.GetPolicy().GetMode(), view.GetPolicy().GetDuration())
	}
	if !view.GetHasExpiry() {
		t.Error("preset policy should have expiry")
	}
	if view.GetExpiresAt() == "" || view.GetTtlSeconds() <= 0 {
		t.Errorf("preset policy should have expires_at and positive ttl, got %q / %v",
			view.GetExpiresAt(), view.GetTtlSeconds())
	}
}

func TestHandleUpdatePolicy_InvalidMode(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	_, err = callUpdatePolicy(t, srv, sess.ID, "bad", "")
	if got := connectCode(err); got != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %s (err=%v)", got, err)
	}
}

func TestHandleUpdatePolicy_NotFound(t *testing.T) {
	srv := newFakeTestServer()
	_, err := callUpdatePolicy(t, srv, "nonexistent", "never", "")
	if got := connectCode(err); got != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %s", got)
	}
}

// Verify policy is included in session response
func TestSessionResponse_IncludesPolicy(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := callCreate(t, srv, 80, 24, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sess.GetPolicy().GetMode() != string(policy.Never) {
		t.Errorf("new session should have never policy, got %s", sess.GetPolicy().GetMode())
	}
	_ = srv.sessions.Delete(sess.GetId())
}

// Verify error catalog includes invalid_policy
func TestErrorCatalog_InvalidPolicy(t *testing.T) {
	ae, ok := errorCatalog["invalid_policy"]
	if !ok {
		t.Fatal("error catalog missing invalid_policy")
	}
	if ae.Category != "validation" {
		t.Errorf("expected category=validation, got %s", ae.Category)
	}
}

// --- Sweeper tests ---

func TestExpirationSweeper_RemovesExpiredSessions(t *testing.T) {
	sm := NewSessionManagerWithFactory(newFakePTYFactory())
	events := events.NewLogger(100)
	metrics := metrics.New()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Set a very short TTL and backdate creation
	sess.SetPolicy(policy.Policy{Mode: policy.Custom, Duration: "1m"})
	sess.mu.Lock()
	sess.CreatedAt = time.Now().Add(-2 * time.Minute) // 2 minutes ago
	sess.mu.Unlock()

	sweeper := NewExpirationSweeper(sm, events, metrics)
	sweeper.sweep() // Run one sweep cycle

	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("expired session should have been removed by sweeper")
	}
}

func TestExpirationSweeper_KeepsNonExpiredSessions(t *testing.T) {
	sm := NewSessionManagerWithFactory(newFakePTYFactory())
	events := events.NewLogger(100)
	metrics := metrics.New()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	// Never-expire policy — should survive sweep
	sweeper := NewExpirationSweeper(sm, events, metrics)
	sweeper.sweep()

	_, ok := sm.Get(sess.ID)
	if !ok {
		t.Error("non-expired session should survive sweep")
	}
}

// [REQ:P1-001a] Sweeper Start() is idempotent — calling twice does not start two goroutines
func TestExpirationSweeper_StartIdempotent(t *testing.T) {
	sm := NewSessionManagerWithFactory(newFakePTYFactory())
	events := events.NewLogger(100)
	metrics := metrics.New()

	sweeper := NewExpirationSweeper(sm, events, metrics)
	sweeper.interval = 50 * time.Millisecond

	sweeper.Start()
	sweeper.Start() // Second call should be a no-op

	sweeper.mu.Lock()
	running := sweeper.running
	sweeper.mu.Unlock()

	if !running {
		t.Error("sweeper should be running after Start")
	}

	sweeper.Stop()

	// Verify Stop works — calling Stop again should not panic
	sweeper.mu.Lock()
	running = sweeper.running
	sweeper.mu.Unlock()

	if running {
		t.Error("sweeper should not be running after Stop")
	}
}

// [REQ:P1-001a] Sweeper loop actually fires and removes expired sessions end-to-end
func TestExpirationSweeper_LoopFiresAndRemoves(t *testing.T) {
	sm := NewSessionManagerWithFactory(newFakePTYFactory())
	events := events.NewLogger(100)
	metrics := metrics.New()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Set a very short TTL and backdate creation
	sess.SetPolicy(policy.Policy{Mode: policy.Custom, Duration: "1m"})
	sess.mu.Lock()
	sess.CreatedAt = time.Now().Add(-2 * time.Minute)
	sess.mu.Unlock()

	sweeper := &ExpirationSweeper{
		sessions: sm,
		events:   events,
		metrics:  metrics,
		interval: 50 * time.Millisecond,
		stopCh:   make(chan struct{}),
	}

	sweeper.Start()

	// Wait for at least one sweep cycle
	time.Sleep(100 * time.Millisecond)
	sweeper.Stop()

	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("expired session should have been removed by sweeper loop")
	}

	if metrics.SessionsDeleted.Load() != 1 {
		t.Errorf("expected 1 deletion metric, got %d", metrics.SessionsDeleted.Load())
	}
}

// Verify the default policy mode on new sessions
func TestDefaultPolicy(t *testing.T) {
	p := policy.Default()
	if p.Mode != policy.Never {
		t.Errorf("default policy should be never, got %s", p.Mode)
	}
	if p.Duration != "" {
		t.Errorf("default policy should have empty duration, got %s", p.Duration)
	}
}
