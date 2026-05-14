package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"web-console/internal/pty"
	"web-console/internal/ptyfake"
)

// newPipedSession returns a session whose PTY is a FakePTYWithOutput so
// the test can drive PTY stdout deterministically by writing to OutW.
func newPipedSession(t *testing.T) (*Session, *ptyfake.FakePTYWithOutput, *Manager) {
	t.Helper()
	var fake *ptyfake.FakePTYWithOutput
	sm := NewManagerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		fake = ptyfake.NewFakePTYWithOutput()
		return fake, nil
	})
	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(sess.ID) })
	return sess, fake, sm
}

func waitForFrame(t *testing.T, sess *Session, max time.Duration) {
	t.Helper()
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		sess.mu.Lock()
		ready := !sess.lastFrameAt.IsZero()
		sess.mu.Unlock()
		if ready {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("no frame arrived within deadline")
}

func TestSession_Screen_AfterPTYOutput(t *testing.T) {
	sess, fake, _ := newPipedSession(t)

	if _, err := fake.OutW.Write([]byte("hello world\r\n")); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	waitForFrame(t, sess, 500*time.Millisecond)

	view := sess.Screen()
	if view.Cols != 80 || view.Rows != 24 {
		t.Errorf("dims: got %dx%d, want 80x24", view.Cols, view.Rows)
	}
	// Row 0 starts with "hello world".
	row := view.Cells[0]
	runes := make([]rune, 0, 11)
	for i := 0; i < 11 && i < len(row); i++ {
		runes = append(runes, row[i].Rune)
	}
	if string(runes) != "hello world" {
		t.Errorf("row 0: got %q, want %q", string(runes), "hello world")
	}
	plain := sess.PlainText(false)
	if !strings.Contains(plain, "hello world") {
		t.Errorf("PlainText: missing 'hello world'\ngot: %q", plain)
	}
}

func TestSession_WaitIdle_BecomesIdle(t *testing.T) {
	sess, _, _ := newPipedSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	reason, waited, err := sess.WaitIdle(ctx, 100*time.Millisecond, 1*time.Second)
	if err != nil {
		t.Fatalf("WaitIdle: %v", err)
	}
	if reason != "idle" {
		t.Errorf("reason: got %q, want %q", reason, "idle")
	}
	if waited < 100*time.Millisecond {
		t.Errorf("waited too short: %v", waited)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("WaitIdle exceeded test timeout")
	}
}

func TestSession_WaitIdle_TimeoutWhileActive(t *testing.T) {
	sess, fake, _ := newPipedSession(t)

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				_, _ = fake.OutW.Write([]byte("."))
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	reason, _, err := sess.WaitIdle(ctx, 200*time.Millisecond, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitIdle: %v", err)
	}
	if reason != "timeout" {
		t.Errorf("reason: got %q, want %q", reason, "timeout")
	}
}

func TestSession_SendInput_RoutesText(t *testing.T) {
	sess, _, _ := newPipedSession(t)

	if err := sess.SendInput(InputText("echo hi\n")); err != nil {
		t.Fatalf("SendInput: %v", err)
	}
	// The FakePTY drains stdin into a buffer; the only observable is that
	// no error occurred and that the call did not panic. Concrete keymap
	// resolution + paste flag coverage lives in input_test.go.
}
