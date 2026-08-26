package session

import (
	"context"
	"fmt"
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
	sess, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(context.Background(), sess.ID) })
	return sess, fake, sm
}

func waitForFrame(t *testing.T, sess *Session, max time.Duration) {
	t.Helper()
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		sess.emuMu.Lock()
		ready := !sess.lastFrameAt.IsZero()
		sess.emuMu.Unlock()
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

func TestSession_ScreenWithTextKeepsGridAndTextTogetherDuringOutput(t *testing.T) {
	sess, fake, _ := newPipedSession(t)

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			_, _ = fmt.Fprintf(fake.OutW, "frame-%03d\r\n", i%1000)
			time.Sleep(time.Microsecond)
		}
	}()

	for i := 0; i < 100; i++ {
		view, plain := sess.ScreenWithText(false)
		var rows []string
		for _, row := range view.Cells {
			end := len(row)
			for end > 0 && row[end-1].Rune == ' ' {
				end--
			}
			runes := make([]rune, end)
			for x := 0; x < end; x++ {
				runes[x] = row[x].Rune
			}
			rows = append(rows, string(runes))
		}
		if got := strings.Join(rows, "\n"); got != plain {
			t.Fatalf("torn screen read at iteration %d: view text %q != plain text %q", i, got, plain)
		}
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

func TestSession_SizeLeaseTransfersAndResizes(t *testing.T) {
	sess, _, _ := newPipedSession(t)
	first := sess.Subscribe()
	second := sess.Subscribe()
	defer sess.Unsubscribe(second.OutputCh)

	sess.DeclareSize(first.OutputCh, 100, 30)
	sess.SetClientDevice(first.OutputCh, "laptop", "Laptop")
	if err := sess.AcquireLease(first.OutputCh, LeaseReasonExplicit); err != nil {
		t.Fatalf("AcquireLease(first): %v", err)
	}
	if !sess.HoldsLease(first.OutputCh) {
		t.Fatal("first client does not hold lease")
	}
	cols, rows, leader, label, holds, viewers := sess.SizeLeaseState(first.OutputCh)
	if cols != 100 || rows != 30 || leader != "laptop" || label != "Laptop" || !holds || viewers != 2 {
		t.Fatalf("lease state = %d x %d, leader=%q/%q holds=%v viewers=%d", cols, rows, leader, label, holds, viewers)
	}
	if err := sess.Resize(second.OutputCh, 120, 40); err != ErrLeaseNotHeld {
		t.Fatalf("follower Resize error = %v, want %v", err, ErrLeaseNotHeld)
	}
	if err := sess.Resize(first.OutputCh, 120, 40); err != nil {
		t.Fatalf("leader Resize: %v", err)
	}
	sess.DeclareSize(second.OutputCh, 90, 25)
	if err := sess.AcquireLease(second.OutputCh, LeaseReasonInput); err != nil {
		t.Fatalf("AcquireLease(second): %v", err)
	}
	sess.Unsubscribe(first.OutputCh)
	cols, rows, leader, label, holds, viewers = sess.SizeLeaseState(second.OutputCh)
	if cols != 90 || rows != 25 || leader != "" || label != "" || !holds || viewers != 1 {
		t.Fatalf("transferred lease state = %d x %d, leader=%q/%q holds=%v viewers=%d", cols, rows, leader, label, holds, viewers)
	}
}

func TestSession_PresencePublishesViewerAndLeaseChanges(t *testing.T) {
	sess, _, _ := newPipedSession(t)
	first := sess.Subscribe()
	second := sess.Subscribe()
	defer sess.Unsubscribe(first.OutputCh)
	defer sess.Unsubscribe(second.OutputCh)

	readPresence := func(ch <-chan PresenceState) PresenceState {
		t.Helper()
		select {
		case state := <-ch:
			return state
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for presence state")
			return PresenceState{}
		}
	}
	if got := readPresence(first.PresenceCh); got.ViewerCount != 2 {
		t.Fatalf("first viewer count = %d, want 2", got.ViewerCount)
	}
	if got := readPresence(second.PresenceCh); got.ViewerCount != 2 {
		t.Fatalf("second initial presence = %+v, want viewer count 2", got)
	}

	sess.SetClientDevice(second.OutputCh, "phone", "Phone")
	if err := sess.AcquireLease(second.OutputCh, LeaseReasonExplicit); err != nil {
		t.Fatalf("AcquireLease(second): %v", err)
	}
	firstState := readPresence(first.PresenceCh)
	secondState := readPresence(second.PresenceCh)
	if firstState.ViewerCount != 2 || firstState.HoldsLease || firstState.Leader != "phone" || firstState.LeaderDevice != "Phone" {
		t.Fatalf("first follower presence = %+v", firstState)
	}
	if secondState.ViewerCount != 2 || !secondState.HoldsLease || secondState.Leader != "phone" || secondState.LeaderDevice != "Phone" {
		t.Fatalf("second leader presence = %+v", secondState)
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
