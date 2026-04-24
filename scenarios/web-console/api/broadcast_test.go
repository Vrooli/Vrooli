package main

import (
	"bytes"
	"testing"
	"time"
)

// newBroadcastSession builds a minimal Session suitable for exercising
// broadcast/deliver/FlushPending without a real PTY. Only the fields
// those functions read are populated.
func newBroadcastSession(clientBuf int, offlineMax int) *Session {
	return &Session{
		ID:                      "broadcast-test",
		clients:                 make(map[chan []byte]*ClientInfo),
		offlineBufferMax:        offlineMax,
		clientChannelBuffer:     clientBuf,
		coalesceNotifyThreshold: 8,
	}
}

// registerClient attaches a ClientInfo on a freshly created channel
// matching what Subscribe would do. Returns the channels the test uses
// for assertions.
func registerClient(s *Session, chBuf int) (chan []byte, chan int, chan bool, *ClientInfo) {
	ch := make(chan []byte, chBuf)
	notifyCh := make(chan int, 1)
	stateCh := make(chan bool, 4)
	info := &ClientInfo{NotifyCh: notifyCh, StateCh: stateCh}
	s.mu.Lock()
	s.clients[ch] = info
	s.mu.Unlock()
	return ch, notifyCh, stateCh, info
}

// TestBroadcast_DeliversToFastClient covers the steady-state path:
// one broadcast call, channel has room, bytes land immediately.
func TestBroadcast_DeliversToFastClient(t *testing.T) {
	s := newBroadcastSession(4, 1024)
	ch, _, _, _ := registerClient(s, 4)
	s.broadcast([]byte("hello"))
	select {
	case got := <-ch:
		if string(got) != "hello" {
			t.Errorf("got %q want %q", got, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast delivery")
	}
}

// TestBroadcast_CoalescesWhenChannelFull covers the slow-client path:
// the channel buffer fills, subsequent broadcasts accumulate in the
// pending buffer, and FlushPending drains them.
func TestBroadcast_CoalescesWhenChannelFull(t *testing.T) {
	// ch buffer = 1 so the second broadcast coalesces.
	s := newBroadcastSession(1, 4096)
	ch, _, _, info := registerClient(s, 1)
	s.broadcast([]byte("first"))
	s.broadcast([]byte("second"))
	s.broadcast([]byte("third"))

	// Drain the first frame.
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out reading first frame")
	}

	// The coalesce buffer should hold "second" + "third".
	s.mu.Lock()
	pending := append([]byte(nil), info.pending...)
	s.mu.Unlock()
	if !bytes.Contains(pending, []byte("second")) || !bytes.Contains(pending, []byte("third")) {
		t.Errorf("pending buffer missing coalesced frames: %q", pending)
	}
	if info.CoalescedFrames < 2 {
		t.Errorf("CoalescedFrames=%d want >=2", info.CoalescedFrames)
	}

	// FlushPending drains the pending buffer into the channel.
	s.FlushPending(ch)
	select {
	case drained := <-ch:
		if !bytes.Contains(drained, []byte("second")) {
			t.Errorf("drained frame missing 'second': %q", drained)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out reading flushed frame")
	}
}

// TestBroadcast_PendingTrimSetsSIGWINCHFlag exercises the pending-cap
// trim branch: when pending exceeds offlineBufferMax, the buffer is
// snapped to a clean boundary and pendingTrimmed=true. FlushPending
// then fires maybeSIGWINCHRecovery. This test avoids exercising the
// actual SetSize call by asserting on the pendingTrimmed transition.
func TestBroadcast_PendingTrimSetsSIGWINCHFlag(t *testing.T) {
	// Small offline buffer forces a trim on the second broadcast.
	s := newBroadcastSession(1, 16)
	ch, _, _, info := registerClient(s, 1)
	s.broadcast([]byte("first-line\n"))         // fills channel
	s.broadcast(bytes.Repeat([]byte("x"), 128)) // pending starts
	s.broadcast(bytes.Repeat([]byte("y"), 128)) // pending trimmed

	s.mu.Lock()
	trimmed := info.pendingTrimmed
	pendingLen := len(info.pending)
	s.mu.Unlock()
	if !trimmed {
		t.Errorf("pendingTrimmed not set after over-cap coalesce")
	}
	if pendingLen > s.offlineBufferMax+len(sgrReset)+1 {
		t.Errorf("pending len=%d exceeds offlineBufferMax=%d (+SGR reset)", pendingLen, s.offlineBufferMax)
	}

	// Drain first frame so FlushPending runs.
	<-ch
}

// TestSIGWINCHGatedByAltBuffer locks in the alt-buffer SIGWINCH guard.
// When the ptyState tracker reports the foreground process is in the
// alt buffer, SetSize must not be fired by the recovery path, because
// it would race the TUI's redraw and cause the "striping" artifact.
// Today this is enforced both by the greenfield assertion test and by
// the runtime check inside maybeSIGWINCHRecovery; we exercise the
// latter here by verifying the function is a no-op in alt-buffer.
func TestSIGWINCHGatedByAltBuffer(t *testing.T) {
	s := newBroadcastSession(1, 4096)
	// Feed an alt-buffer-enter sequence to the ptyState tracker.
	s.ptyState.Observe([]byte("\x1b[?1049h"))
	if !s.ptyState.IsAltBuffer() {
		t.Fatal("ptyState did not register alt-buffer entry")
	}
	before := s.lastSIGWINCHRecovery
	s.mu.Lock()
	s.maybeSIGWINCHRecovery()
	s.mu.Unlock()
	if s.lastSIGWINCHRecovery != before {
		t.Errorf("maybeSIGWINCHRecovery fired while in alt-buffer (lastSIGWINCHRecovery advanced)")
	}
}

// TestSIGWINCHGatedByRecentAltBufferTransition is the regression test
// for the "Claude Code footer repeats in scrollback" bug. Heavy TUIs
// briefly exit the alt-buffer between render cycles; if SIGWINCH
// fires during that short window, the TUI redraws its footer into
// the pane's normal buffer, which tmux captures into scrollback.
// The fix widens the guard to refuse SIGWINCH within
// altBufferSettleWindow of ANY alt-buffer transition.
func TestSIGWINCHGatedByRecentAltBufferTransition(t *testing.T) {
	s := newBroadcastSession(1, 4096)
	s.sigwinchCooldown = 0 // isolate the alt-buffer-recency guard
	// Simulate a just-observed alt-buffer exit: enter, then exit.
	s.broadcast([]byte("\x1b[?1049h")) // enter alt-buffer
	s.broadcast([]byte("\x1b[?1049l")) // immediately exit alt-buffer
	if s.ptyState.IsAltBuffer() {
		t.Fatal("alt-buffer should have exited after DECRST 1049")
	}
	if s.lastAltBufferTransition.IsZero() {
		t.Fatal("lastAltBufferTransition not recorded on alt-buffer cycle")
	}
	before := s.lastSIGWINCHRecovery
	s.mu.Lock()
	s.maybeSIGWINCHRecovery()
	s.mu.Unlock()
	if s.lastSIGWINCHRecovery != before {
		t.Errorf("maybeSIGWINCHRecovery fired within altBufferSettleWindow of a transition")
	}
}

// TestSIGWINCHSkippedOnPersistentBackend locks in that tmux-backed
// sessions never fire SIGWINCH recovery via the coalesce-trim path.
// Tmux handles pane sizing itself; an extra SIGWINCH adds no
// recovery value and reintroduces the Claude Code scrollback-
// duplication bug.
func TestSIGWINCHSkippedOnPersistentBackend(t *testing.T) {
	s := newBroadcastSession(1, 4096)
	s.Backend = BackendPersistent
	before := s.lastSIGWINCHRecovery
	s.mu.Lock()
	s.maybeSIGWINCHRecovery()
	s.mu.Unlock()
	if s.lastSIGWINCHRecovery != before {
		t.Errorf("maybeSIGWINCHRecovery fired on persistent backend (lastSIGWINCHRecovery advanced)")
	}
}
