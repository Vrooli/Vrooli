package session

import (
	"context"
	"errors"
	"testing"

	"web-console/internal/ptyfake"
)

type cleanupFailingPTY struct {
	*ptyfake.FakePTY
	killErr  error
	closeErr error
}

func (p *cleanupFailingPTY) Kill() error { return p.killErr }
func (p *cleanupFailingPTY) Close() error {
	return errors.Join(p.closeErr, p.FakePTY.Close())
}

func TestManagerDeleteSurfacesProcessCleanupFailure(t *testing.T) {
	killErr := errors.New("kill failed")
	closeErr := errors.New("close failed")
	base := ptyfake.NewFakePTYWithOutput()
	pty := &cleanupFailingPTY{FakePTY: &base.FakePTY, killErr: killErr, closeErr: closeErr}
	sm := NewManagerWithFactory(ptyfake.Factory(pty))
	sess, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = sm.Delete(context.Background(), sess.ID)
	if !errors.Is(err, killErr) || !errors.Is(err, closeErr) {
		t.Fatalf("Delete error = %v, want kill and close failures", err)
	}
	if _, ok := sm.Get(sess.ID); ok {
		t.Fatal("failed process cleanup must still remove the runtime handle")
	}
}

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
	s, _ := sm.Create(context.Background(), "", 0, 0, "", nil)
	defer func() { _ = sm.Delete(context.Background(), s.ID) }()
	if sm.isSessionLimitReached() {
		t.Error("1 session with limit 5 should not be reached")
	}
}

func TestIsSessionLimitReached_AtLimit(t *testing.T) {
	sm := NewManagerWithFactory(ptyfake.NewFactory())
	sm.cfg.MaxSessions = 1
	s, _ := sm.Create(context.Background(), "", 0, 0, "", nil)
	defer func() { _ = sm.Delete(context.Background(), s.ID) }()
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

	s1, err := sm.Create(context.Background(), "", 0, 0, "", nil)
	if err != nil {
		t.Fatalf("first session: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), s1.ID) }()

	s2, err := sm.Create(context.Background(), "", 0, 0, "", nil)
	if err != nil {
		t.Fatalf("second session: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), s2.ID) }()

	_, err = sm.Create(context.Background(), "", 0, 0, "", nil)
	if err == nil {
		t.Error("third session should be rejected when MaxSessions=2")
	}
}
