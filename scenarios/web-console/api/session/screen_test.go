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
	sess.SetClientDevice(first.OutputCh, "laptop", "Laptop", "laptop")
	if err := sess.AcquireLease(first.OutputCh, LeaseReasonExplicit); err != nil {
		t.Fatalf("AcquireLease(first): %v", err)
	}
	if !sess.HoldsLease(first.OutputCh) {
		t.Fatal("first client does not hold lease")
	}
	state := sess.SizeLeaseState(first.OutputCh)
	if state.Cols != 100 || state.Rows != 30 || state.Leader != "laptop" || state.LeaderDevice != "Laptop" || !state.HoldsLease || state.ViewerCount != 2 {
		t.Fatalf("lease state = %+v", state)
	}
	if state.LeaderClass != "laptop" {
		t.Fatalf("lease state leader class = %q, want %q", state.LeaderClass, "laptop")
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
	state = sess.SizeLeaseState(second.OutputCh)
	if state.Cols != 90 || state.Rows != 25 || state.Leader != "" || state.LeaderDevice != "" || !state.HoldsLease || state.ViewerCount != 1 {
		t.Fatalf("transferred lease state = %+v", state)
	}
}

func TestSession_ConnectedDevicesGroupsTwoSocketsOfOneDevice(t *testing.T) {
	sess, _, _ := newPipedSession(t)
	first, second := sess.Subscribe(), sess.Subscribe()
	defer sess.Unsubscribe(first.OutputCh)
	defer sess.Unsubscribe(second.OutputCh)
	sess.SetClientDevice(first.OutputCh, "device-1", "Phone", "phone")
	sess.SetClientDevice(second.OutputCh, "device-1", "Phone", "phone")
	devices := sess.ConnectedDevices()
	if len(devices) != 1 || devices[0].DeviceID != "device-1" || len(devices[0].Connections) != 2 {
		t.Fatalf("devices = %+v, want one device with two connections", devices)
	}
}

func TestSession_ConnectedDevicesKeepsUnidentifiedConnectionsSeparate(t *testing.T) {
	sess, _, _ := newPipedSession(t)
	first, second := sess.Subscribe(), sess.Subscribe()
	defer sess.Unsubscribe(first.OutputCh)
	defer sess.Unsubscribe(second.OutputCh)
	devices := sess.ConnectedDevices()
	if len(devices) != 2 {
		t.Fatalf("devices = %+v, want two unidentified devices", devices)
	}
	for _, device := range devices {
		if device.DeviceID != "" || len(device.Connections) != 1 {
			t.Fatalf("device = %+v, want one unidentified connection", device)
		}
	}
}

func TestManager_ConnectedDevicesAggregatesAcrossSessions(t *testing.T) {
	first, _, manager := newPipedSession(t)
	created, err := manager.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer manager.Delete(context.Background(), created.ID)
	left, right := first.Subscribe(), created.Subscribe()
	defer first.Unsubscribe(left.OutputCh)
	defer created.Unsubscribe(right.OutputCh)
	first.SetClientDevice(left.OutputCh, "device-1", "Phone", "phone")
	created.SetClientDevice(right.OutputCh, "device-1", "Phone", "phone")
	devices := manager.ConnectedDevices()
	if len(devices) != 1 || devices[0].ConnCount != 2 || len(devices[0].Sessions) != 2 {
		t.Fatalf("devices = %+v, want one device across two sessions", devices)
	}
}

func TestSession_ReconnectingDeviceReclaimsItsOwnLease(t *testing.T) {
	sess, _, _ := newPipedSession(t)
	old := sess.Subscribe("device-1", "Phone", "phone")
	sess.DeclareSize(old.OutputCh, 100, 30)
	newConn := sess.Subscribe("device-1", "Phone", "phone")
	defer sess.Unsubscribe(newConn.OutputCh)
	if !sess.HoldsLease(newConn.OutputCh) || sess.HoldsLease(old.OutputCh) {
		t.Fatalf("lease owner did not move to reconnecting device")
	}
	if state := sess.SizeLeaseState(newConn.OutputCh); state.Leader != "device-1" || state.HoldsLease != true {
		t.Fatalf("state = %+v, want reconnecting device as leader", state)
	}
	if newConn.ReclaimClient != old.OutputCh {
		t.Fatalf("reclaim client = %v, want old connection", newConn.ReclaimClient)
	}
	sess.Unsubscribe(old.OutputCh)
}

func TestSession_ReclaimAppliesTheArrivingConnectionDeclaredSize(t *testing.T) {
	sess, fake, _ := newPipedSession(t)
	old := sess.Subscribe("device-1", "Phone", "phone")
	sess.DeclareSize(old.OutputCh, 100, 30)
	if err := sess.AcquireLease(old.OutputCh, LeaseReasonExplicit); err != nil {
		t.Fatal(err)
	}
	newConn := sess.Subscribe("device-1", "Phone", "phone")
	defer sess.Unsubscribe(newConn.OutputCh)
	sess.DeclareSize(newConn.OutputCh, 140, 42)
	// A declaration arrives after Subscribe in the current client protocol;
	// explicitly reclaim once more to exercise the established resize path.
	if err := sess.AcquireLease(newConn.OutputCh, LeaseReasonDeviceReclaim); err != nil {
		t.Fatal(err)
	}
	if fake.Cols != 140 || fake.Rows != 42 {
		t.Fatalf("PTY size = %dx%d, want 140x42", fake.Cols, fake.Rows)
	}
	sess.Unsubscribe(old.OutputCh)
}

// The leader's keyboard state is presentational, so it must reach followers,
// and a follower's own keyboard must not wake anybody.
func TestSession_KeyboardStateFollowsTheLeaseOwner(t *testing.T) {
	sess, _, _ := newPipedSession(t)
	leader := sess.Subscribe()
	follower := sess.Subscribe()
	defer sess.Unsubscribe(leader.OutputCh)
	defer sess.Unsubscribe(follower.OutputCh)

	drain := func(ch <-chan PresenceState) {
		for {
			select {
			case <-ch:
			default:
				return
			}
		}
	}
	awaitPresence := func(ch <-chan PresenceState) PresenceState {
		t.Helper()
		select {
		case state := <-ch:
			return state
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for presence state")
			return PresenceState{}
		}
	}

	sess.SetClientDevice(leader.OutputCh, "phone-id", "iPhone", "phone")
	if err := sess.AcquireLease(leader.OutputCh, LeaseReasonExplicit); err != nil {
		t.Fatalf("AcquireLease(leader): %v", err)
	}
	drain(leader.PresenceCh)
	drain(follower.PresenceCh)

	sess.SetClientKeyboard(leader.OutputCh, true)
	if got := awaitPresence(follower.PresenceCh); !got.LeaderKbOpen || got.LeaderClass != "phone" {
		t.Fatalf("follower presence after leader keyboard opened = %+v", got)
	}
	if got := sess.SizeLeaseState(follower.OutputCh); !got.LeaderKbOpen {
		t.Fatalf("snapshot did not report the leader keyboard: %+v", got)
	}

	// Repeating the same state is not a change and must not republish.
	drain(follower.PresenceCh)
	sess.SetClientKeyboard(leader.OutputCh, true)
	select {
	case got := <-follower.PresenceCh:
		t.Fatalf("unchanged keyboard state republished presence: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}

	// A follower's own keyboard is nobody else's business. Drain the leader's
	// own copy of the earlier publish first, so this asserts on new traffic.
	drain(leader.PresenceCh)
	sess.SetClientKeyboard(follower.OutputCh, true)
	select {
	case got := <-leader.PresenceCh:
		t.Fatalf("follower keyboard published presence to the leader: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}

	sess.SetClientKeyboard(leader.OutputCh, false)
	if got := awaitPresence(follower.PresenceCh); got.LeaderKbOpen {
		t.Fatalf("follower presence after leader keyboard closed = %+v", got)
	}
}

// A handover must present the new leader immediately, including a keyboard
// that was already open before the lease moved.
func TestSession_LeaseHandoverCarriesDevicePresentation(t *testing.T) {
	sess, _, _ := newPipedSession(t)
	first := sess.Subscribe()
	second := sess.Subscribe()
	defer sess.Unsubscribe(first.OutputCh)
	defer sess.Unsubscribe(second.OutputCh)

	sess.SetClientDevice(first.OutputCh, "desk-id", "Desktop", "monitor")
	sess.SetClientDevice(second.OutputCh, "phone-id", "iPhone", "phone")
	sess.SetClientKeyboard(second.OutputCh, true)
	if err := sess.AcquireLease(first.OutputCh, LeaseReasonExplicit); err != nil {
		t.Fatalf("AcquireLease(first): %v", err)
	}
	if got := sess.SizeLeaseState(second.OutputCh); got.LeaderClass != "monitor" || got.LeaderKbOpen {
		t.Fatalf("presentation before handover = %+v", got)
	}
	if err := sess.AcquireLease(second.OutputCh, LeaseReasonInput); err != nil {
		t.Fatalf("AcquireLease(second): %v", err)
	}
	got := sess.SizeLeaseState(first.OutputCh)
	if got.LeaderClass != "phone" || !got.LeaderKbOpen || got.LeaderDevice != "iPhone" {
		t.Fatalf("presentation after handover = %+v", got)
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

	sess.SetClientDevice(second.OutputCh, "phone", "Phone", "phone")
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
