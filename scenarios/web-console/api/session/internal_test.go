package session

import (
	"testing"
	"web-console/internal/ptyfake"
)

// Tests that touch private Manager methods/fields live in the same package
// so they can exercise the seams without forcing them onto the public API.

func TestApplySessionDefaults_AllZeros(t *testing.T) {
	sm := NewManagerWithFactory(ptyfake.NewFactory())
	shell, cols, rows := sm.applySessionDefaults("", 0, 0)
	if shell == "" {
		t.Error("shell should be filled with default")
	}
	if cols == 0 {
		t.Error("cols should be filled with default")
	}
	if rows == 0 {
		t.Error("rows should be filled with default")
	}
}

func TestApplySessionDefaults_ExplicitValues(t *testing.T) {
	sm := NewManagerWithFactory(ptyfake.NewFactory())
	shell, cols, rows := sm.applySessionDefaults("/bin/zsh", 120, 40)
	if shell != "/bin/zsh" {
		t.Errorf("explicit shell should be preserved, got %s", shell)
	}
	if cols != 120 {
		t.Errorf("explicit cols should be preserved, got %d", cols)
	}
	if rows != 40 {
		t.Errorf("explicit rows should be preserved, got %d", rows)
	}
}

func TestApplySessionDefaults_MixedZeroAndExplicit(t *testing.T) {
	sm := NewManagerWithFactory(ptyfake.NewFactory())
	shell, cols, rows := sm.applySessionDefaults("/bin/fish", 0, 40)
	if shell != "/bin/fish" {
		t.Error("explicit shell should be preserved")
	}
	if cols == 0 {
		t.Error("zero cols should be replaced with default")
	}
	if rows != 40 {
		t.Error("explicit rows should be preserved")
	}
}

func TestIsSessionLimitReached_Unlimited(t *testing.T) {
	sm := NewManagerWithFactory(ptyfake.NewFactory())
	sm.cfg.MaxSessions = 0
	if sm.isSessionLimitReached() {
		t.Error("MaxSessions=0 means unlimited, should never be reached")
	}
}

func TestIsSessionLimitReached_UnderLimit(t *testing.T) {
	sm := NewManagerWithFactory(ptyfake.NewFactory())
	sm.cfg.MaxSessions = 5
	s, _ := sm.Create("", 0, 0, "", nil)
	defer func() { _ = sm.Delete(s.ID) }()
	if sm.isSessionLimitReached() {
		t.Error("1 session with limit 5 should not be reached")
	}
}

func TestIsSessionLimitReached_AtLimit(t *testing.T) {
	sm := NewManagerWithFactory(ptyfake.NewFactory())
	sm.cfg.MaxSessions = 1
	s, _ := sm.Create("", 0, 0, "", nil)
	defer func() { _ = sm.Delete(s.ID) }()
	if !sm.isSessionLimitReached() {
		t.Error("1 session with limit 1 should be reached")
	}
}

// TestMaxSessions_Enforcement exercises Create's cap behavior; lives here so
// the test can set sm.cfg.MaxSessions directly.
//
// [REQ:P1-001a] Session Policy Controls - max sessions enforcement
func TestMaxSessions_Enforcement(t *testing.T) {
	sm := NewManagerWithFactory(ptyfake.NewFactory())
	sm.cfg.MaxSessions = 2

	s1, err := sm.Create("", 0, 0, "", nil)
	if err != nil {
		t.Fatalf("first session: %v", err)
	}
	defer func() { _ = sm.Delete(s1.ID) }()

	s2, err := sm.Create("", 0, 0, "", nil)
	if err != nil {
		t.Fatalf("second session: %v", err)
	}
	defer func() { _ = sm.Delete(s2.ID) }()

	_, err = sm.Create("", 0, 0, "", nil)
	if err == nil {
		t.Error("third session should be rejected when MaxSessions=2")
	}
}
