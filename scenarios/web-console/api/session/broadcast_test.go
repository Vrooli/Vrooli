package session

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"web-console/terminal"
)

func newBroadcastSession(clientBuf int) *Session {
	return &Session{ID: "broadcast-test", clients: make(map[chan []byte]*ClientInfo), emu: terminal.New(terminal.Options{Cols: 80, Rows: 24, ScrollbackLines: 1000}), clientChannelBuffer: clientBuf, coalesceNotifyThreshold: 8}
}

func registerClient(s *Session, chBuf int) (chan []byte, chan int, *ClientInfo) {
	ch := make(chan []byte, chBuf)
	notifyCh := make(chan int, 1)
	info := &ClientInfo{NotifyCh: notifyCh}
	s.emuMu.Lock()
	s.clients[ch] = info
	s.emuMu.Unlock()
	return ch, notifyCh, info
}

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

func TestBroadcast_CoalescesWhenChannelFull(t *testing.T) {
	s := newBroadcastSession(1)
	ch, _, info := registerClient(s, 1)
	s.broadcast([]byte("first"))
	s.broadcast([]byte("second"))
	s.broadcast([]byte("third"))
	<-ch
	s.emuMu.Lock()
	pending := append([]byte(nil), info.pending...)
	s.emuMu.Unlock()
	if !bytes.Contains(pending, []byte("second")) || !bytes.Contains(pending, []byte("third")) {
		t.Errorf("pending buffer missing coalesced frames: %q", pending)
	}
	if info.CoalescedFrames < 2 {
		t.Errorf("CoalescedFrames=%d want >=2", info.CoalescedFrames)
	}
	if s.FlushPending(ch) {
		t.Fatal("ordinary coalescing requested resync")
	}
	select {
	case drained := <-ch:
		if !bytes.Contains(drained, []byte("second")) {
			t.Errorf("drained frame missing 'second': %q", drained)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out reading flushed frame")
	}
}

func TestBroadcast_FeedsEmulator(t *testing.T) {
	s := newBroadcastSession(4)
	s.broadcast([]byte("hello\r\n"))
	if !bytes.Contains(s.emu.Snapshot(), []byte("hello")) {
		t.Fatalf("snapshot missing broadcasted bytes")
	}
}

func TestBroadcast_RetainsCursorBoundedFramesForReconnect(t *testing.T) {
	s := newBroadcastSession(4)
	first := s.Subscribe()
	s.Unsubscribe(first.OutputCh)
	s.broadcast([]byte("one"))
	s.broadcast([]byte("two"))
	frames, cursor, ok := s.ReplayFrom(3)
	if !ok || cursor != 6 || len(frames) != 1 {
		t.Fatalf("ReplayFrom(3) = frames=%d cursor=%d ok=%v", len(frames), cursor, ok)
	}
	if string(frames[0].Data) != "two" || frames[0].StartCursor != 3 || frames[0].EndCursor != 6 {
		t.Fatalf("unexpected replay frame: %+v", frames[0])
	}
	if _, _, ok := s.ReplayFrom(2); ok {
		t.Fatal("replay accepted a cursor that is not on a frame boundary")
	}
}

func TestBroadcast_PreservesSplitSynchronizedOutputFraming(t *testing.T) {
	s := newBroadcastSession(4)
	ch, _, _ := registerClient(s, 4)
	s.broadcast([]byte("before\x1b[?2026"))
	s.broadcast([]byte("hpaint\x1b[?2026lafter"))

	var got []byte
	got = append(got, (<-ch)...)
	got = append(got, (<-ch)...)
	if string(got) != "before\x1b[?2026hpaint\x1b[?2026lafter" {
		t.Fatalf("client stream changed synchronized-output framing: %q", got)
	}
}

func TestBroadcast_OverflowRequestsResync(t *testing.T) {
	s := newBroadcastSession(1)
	ch, _, info := registerClient(s, 1)
	s.broadcast([]byte("first"))
	for i := 0; i < pendingBufferMax/5+1; i++ {
		s.broadcast([]byte("\x1b[31mfragment"))
	}
	<-ch
	if !s.FlushPending(ch) {
		t.Fatal("overflow did not request resync")
	}
	snapshot, generation, ok := s.Resync(ch)
	if !ok || len(snapshot) == 0 || generation == 0 {
		t.Fatalf("Resync() = (%d bytes, %d, %v)", len(snapshot), generation, ok)
	}
	if info.pending != nil {
		t.Fatalf("pending buffer retained after overflow: %d bytes", len(info.pending))
	}
	s.CompleteResync(ch, generation)
	if _, _, ok := s.Resync(ch); ok {
		t.Fatal("completed resync remained pending")
	}
}

func TestSubscribe_SnapshotPrecedesLiveFrames(t *testing.T) {
	s := newBroadcastSession(4)
	s.broadcast([]byte("before-subscribe\r\n"))
	sub := SubscribeResult{}
	s.emuMu.Lock()
	sub.Snapshot = s.emu.Snapshot()
	ch := make(chan []byte, 4)
	s.clients[ch] = &ClientInfo{NotifyCh: make(chan int, 1)}
	sub.OutputCh = ch
	s.emuMu.Unlock()
	if !bytes.Contains(sub.Snapshot, []byte("before-subscribe")) {
		t.Fatalf("snapshot missing pre-subscribe output")
	}
	s.broadcast([]byte("after-subscribe"))
	select {
	case got := <-sub.OutputCh:
		if !strings.Contains(string(got), "after-subscribe") {
			t.Fatalf("unexpected live frame: %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("missed live frame")
	}
}

func TestSubscribe_SnapshotCacheIsInvalidatedByFeed(t *testing.T) {
	s := newBroadcastSession(4)
	first := s.Subscribe()
	s.Unsubscribe(first.OutputCh)
	s.broadcast([]byte("cache invalidation marker\r\n"))
	second := s.Subscribe()
	s.Unsubscribe(second.OutputCh)
	if !bytes.Contains(second.Snapshot, []byte("cache invalidation marker")) {
		t.Fatalf("snapshot after feed is stale")
	}
}

func TestSubscribe_RepeatedSubscribeReusesCache(t *testing.T) {
	s := newBroadcastSession(4)
	first := s.Subscribe()
	second := s.Subscribe()
	defer s.Unsubscribe(first.OutputCh)
	defer s.Unsubscribe(second.OutputCh)
	if len(first.Snapshot) == 0 || len(second.Snapshot) == 0 {
		t.Fatal("snapshot cache returned an empty snapshot")
	}
	if &first.Snapshot[0] != &second.Snapshot[0] {
		t.Fatal("repeated subscribe regenerated instead of reusing the cached snapshot")
	}
}
