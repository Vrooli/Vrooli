package main

import (
	"bytes"
	"testing"
	"time"

	"web-console/terminal"
)

// newBroadcastSession builds a minimal Session suitable for exercising
// broadcast/deliver/FlushPending without a real PTY. Only the fields
// those functions read are populated.
func newBroadcastSession(clientBuf int) *Session {
	return &Session{
		ID:                      "broadcast-test",
		clients:                 make(map[chan []byte]*ClientInfo),
		emu:                     terminal.New(terminal.Options{Cols: 80, Rows: 24, ScrollbackLines: 1000}),
		clientChannelBuffer:     clientBuf,
		coalesceNotifyThreshold: 8,
	}
}

// registerClient attaches a ClientInfo on a freshly created channel
// matching what Subscribe would do.
func registerClient(s *Session, chBuf int) (chan []byte, chan int, *ClientInfo) {
	ch := make(chan []byte, chBuf)
	notifyCh := make(chan int, 1)
	info := &ClientInfo{NotifyCh: notifyCh}
	s.mu.Lock()
	s.clients[ch] = info
	s.mu.Unlock()
	return ch, notifyCh, info
}

// TestBroadcast_DeliversToFastClient covers the steady-state path:
// one broadcast call, channel has room, bytes land immediately.
func TestBroadcast_DeliversToFastClient(t *testing.T) {
	s := newBroadcastSession(4)
	ch, _, _ := registerClient(s, 4)
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
	s := newBroadcastSession(1)
	ch, _, info := registerClient(s, 1)
	s.broadcast([]byte("first"))
	s.broadcast([]byte("second"))
	s.broadcast([]byte("third"))

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out reading first frame")
	}

	s.mu.Lock()
	pending := append([]byte(nil), info.pending...)
	s.mu.Unlock()
	if !bytes.Contains(pending, []byte("second")) || !bytes.Contains(pending, []byte("third")) {
		t.Errorf("pending buffer missing coalesced frames: %q", pending)
	}
	if info.CoalescedFrames < 2 {
		t.Errorf("CoalescedFrames=%d want >=2", info.CoalescedFrames)
	}

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

// TestBroadcast_FeedsEmulator verifies bytes flow into the durable
// emulator so reconnecting subscribers see them in the snapshot.
func TestBroadcast_FeedsEmulator(t *testing.T) {
	s := newBroadcastSession(4)
	s.broadcast([]byte("hello\r\n"))
	snap := s.emu.Snapshot()
	if !bytes.Contains(snap, []byte("hello")) {
		t.Fatalf("snapshot missing broadcasted bytes: %q", snap)
	}
}

// TestBroadcast_AltBufferFlagFollowsEmulator verifies the alt-buffer
// transition observed by the emulator updates s.inAltBuffer and
// timestamps the transition.
func TestBroadcast_AltBufferFlagFollowsEmulator(t *testing.T) {
	s := newBroadcastSession(4)
	registerClient(s, 4)
	s.broadcast([]byte("\x1b[?1049h"))
	if !s.inAltBuffer {
		t.Fatal("inAltBuffer not set after \\x1b[?1049h")
	}
	if s.lastAltBufferTransition.IsZero() {
		t.Fatal("lastAltBufferTransition not recorded on alt-buffer enter")
	}
}

// TestSIGWINCH_GatedByAltBuffer locks in that SetSize must not fire while
// the foreground process is in the alt buffer.
func TestSIGWINCH_GatedByAltBuffer(t *testing.T) {
	s := newBroadcastSession(1)
	s.inAltBuffer = true
	before := s.lastSIGWINCHRecovery
	s.mu.Lock()
	s.maybeSIGWINCHRecovery()
	s.mu.Unlock()
	if s.lastSIGWINCHRecovery != before {
		t.Errorf("maybeSIGWINCHRecovery fired while in alt-buffer")
	}
}

// TestSIGWINCH_GatedByRecentAltTransition: heavy TUIs briefly leave
// alt-buffer between render cycles; SIGWINCH must refuse to fire
// within altBufferSettleWindow of any transition.
func TestSIGWINCH_GatedByRecentAltTransition(t *testing.T) {
	s := newBroadcastSession(1)
	s.sigwinchCooldown = 0
	s.broadcast([]byte("\x1b[?1049h"))
	s.broadcast([]byte("\x1b[?1049l"))
	if s.inAltBuffer {
		t.Fatal("alt-buffer should have exited after DECRST 1049")
	}
	if s.lastAltBufferTransition.IsZero() {
		t.Fatal("lastAltBufferTransition not recorded")
	}
	before := s.lastSIGWINCHRecovery
	s.mu.Lock()
	s.maybeSIGWINCHRecovery()
	s.mu.Unlock()
	if s.lastSIGWINCHRecovery != before {
		t.Errorf("maybeSIGWINCHRecovery fired within altBufferSettleWindow")
	}
}

// TestSIGWINCH_SkippedOnPersistentBackend locks in that tmux-backed
// sessions never fire SIGWINCH recovery.
func TestSIGWINCH_SkippedOnPersistentBackend(t *testing.T) {
	s := newBroadcastSession(1)
	s.Backend = BackendPersistent
	before := s.lastSIGWINCHRecovery
	s.mu.Lock()
	s.maybeSIGWINCHRecovery()
	s.mu.Unlock()
	if s.lastSIGWINCHRecovery != before {
		t.Errorf("maybeSIGWINCHRecovery fired on persistent backend")
	}
}
