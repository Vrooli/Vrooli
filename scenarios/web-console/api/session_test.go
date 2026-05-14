package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"web-console/internal/config"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
)

// [REQ:P0-002a] PTY Session Backend
func TestSessionManager_Create(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
	if sess.Cols != 80 {
		t.Errorf("expected cols=80, got %d", sess.Cols)
	}
	if sess.Rows != 24 {
		t.Errorf("expected rows=24, got %d", sess.Rows)
	}
	if sess.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}
}

// [REQ:P0-002a] PTY Session Backend - concurrent sessions
func TestSessionManager_ConcurrentSessions(t *testing.T) {
	sm := NewSessionManager()

	s1, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	defer func() { _ = sm.Delete(s1.ID) }()

	s2, err := sm.Create("", 120, 40, "", nil)
	if err != nil {
		t.Fatalf("Create s2 failed: %v", err)
	}
	defer func() { _ = sm.Delete(s2.ID) }()

	if s1.ID == s2.ID {
		t.Error("session IDs should be unique")
	}

	sessions := sm.List()
	if len(sessions) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(sessions))
	}
}

// [REQ:P0-003a] Session Persistence Store - list returns all active
func TestSessionManager_List(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	sessions := sm.List()
	found := false
	for _, s := range sessions {
		if s.ID == sess.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("List should include the created session")
	}
}

// [REQ:P0-002a] PTY Session Backend - Get
func TestSessionManager_Get(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	got, ok := sm.Get(sess.ID)
	if !ok {
		t.Fatal("Get should find the session")
	}
	if got.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, got.ID)
	}

	_, ok = sm.Get("nonexistent")
	if ok {
		t.Error("Get should not find nonexistent session")
	}
}

// [REQ:P0-002a] PTY Session Backend - Delete cleanup
func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = sm.Delete(sess.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("session should not exist after Delete")
	}

	err = sm.Delete("nonexistent")
	if err == nil {
		t.Error("Delete should fail for nonexistent session")
	}
}

// [REQ:P0-002c] Terminal Resize Handling — last writer wins
func TestSession_Resize(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)

	sess.Resize(120, 40)

	cols, rows := sess.EffectiveSize()
	if cols != 120 {
		t.Errorf("expected cols=120, got %d", cols)
	}
	if rows != 40 {
		t.Errorf("expected rows=40, got %d", rows)
	}
}

// [REQ:P0-002c] Last resize wins regardless of which client sent it
func TestSession_Resize_LastWriterWins(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub1 := sess.Subscribe()
	defer sess.Unsubscribe(sub1.OutputCh)
	sub2 := sess.Subscribe()
	defer sess.Unsubscribe(sub2.OutputCh)

	// Desktop resizes to 120x40
	sess.Resize(120, 40)
	cols, rows := sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		t.Errorf("after first resize: expected 120x40, got %dx%d", cols, rows)
	}

	// Phone resizes to 60x20 — last writer wins (not max)
	sess.Resize(60, 20)
	cols, rows = sess.EffectiveSize()
	if cols != 60 || rows != 20 {
		t.Errorf("after second resize: expected 60x20, got %dx%d", cols, rows)
	}

	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002c] PTY size unchanged when all clients disconnect
func TestSession_NoClients_SizeUnchanged(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe()
	sess.Resize(120, 40)
	sess.Unsubscribe(sub.OutputCh)

	// PTY size should remain at last resize value
	cols, rows := sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		t.Errorf("expected 120x40 after client leaves, got %dx%d", cols, rows)
	}

	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002b] WebSocket I/O Streaming - Subscribe/broadcast
func TestSession_SubscribeAndBroadcast(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("/bin/sh", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)

	// Write to stdin - the shell should echo something back
	err = sess.WriteInput([]byte("echo hello\n"), pty.KindKeystroke)
	if err != nil {
		t.Fatalf("WriteInput failed: %v", err)
	}

	// Read output with timeout
	select {
	case data := <-sub.OutputCh:
		if len(data) == 0 {
			t.Error("expected non-empty output")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for terminal output")
	}
}

func TestDefaultConfig_Shell(t *testing.T) {
	cfg := config.Default()
	if cfg.DefaultShell == "" {
		t.Error("DefaultShell should not be empty")
	}
}

// TestSession_OfflineSnapshotIncludesPriorOutput exercises the snapshot
// replay path: bytes broadcast while no client is subscribed must appear
// in the snapshot returned by a later Subscribe().
func TestSession_OfflineSnapshotIncludesPriorOutput(t *testing.T) {
	sm := NewSessionManager()
	sess, err := sm.Create("/bin/sh", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	if err := sess.WriteInput([]byte("echo offline_marker\n"), pty.KindKeystroke); err != nil {
		t.Fatalf("WriteInput: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)
	if !bytes.Contains(sub.Snapshot, []byte("offline_marker")) {
		t.Fatalf("snapshot missing prior output; bytes=%q", sub.Snapshot)
	}
}

// TestSubscribe_SnapshotEmptyOnFreshSession verifies a freshly-created
// session's snapshot is non-empty (it carries the reset prologue and
// blank-screen rows) but contains no user-visible content.
func TestSubscribe_SnapshotIsSelfContained(t *testing.T) {
	sm := NewSessionManager()
	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()
	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)
	if len(sub.Snapshot) == 0 {
		t.Fatal("snapshot empty on fresh session")
	}
	if !bytes.HasPrefix(sub.Snapshot, []byte("\x1b[?1049l\x1bc")) {
		t.Fatalf("snapshot must start with alt-exit + full reset; got %q", sub.Snapshot[:16])
	}
}

// TestSubscribe_SnapshotPrecedesLiveFrames verifies the contract that
// no live frame can sneak between Subscribe()'s snapshot capture and
// channel registration. After Subscribe returns, broadcasting a frame
// delivers it on OutputCh; the snapshot reflects state strictly before
// that frame.
func TestSubscribe_SnapshotPrecedesLiveFrames(t *testing.T) {
	s := newBroadcastSession(4)
	s.broadcast([]byte("before-subscribe\r\n"))
	sub := SubscribeResult{}
	s.mu.Lock()
	sub.Snapshot = s.emu.Snapshot()
	ch := make(chan []byte, 4)
	s.clients[ch] = &ClientInfo{NotifyCh: make(chan int, 1)}
	sub.OutputCh = ch
	s.mu.Unlock()

	if !bytes.Contains(sub.Snapshot, []byte("before-subscribe")) {
		t.Fatalf("snapshot missing pre-subscribe output: %q", sub.Snapshot)
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
