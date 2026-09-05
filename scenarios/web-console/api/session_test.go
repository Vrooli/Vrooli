package main

import (
	"bytes"
	"context"
	"testing"
	"time"

	"web-console/internal/config"
	"web-console/internal/ptyfake"
	"web-console/session"
)

// [REQ:P0-002a] PTY Session Backend
func TestSessionManager_Create(t *testing.T) {
	sm := newSessionManager()

	sess, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), sess.ID) }()

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
	sm := newSessionManager()

	s1, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), s1.ID) }()

	s2, err := sm.Create(context.Background(), "", 120, 40, "", nil)
	if err != nil {
		t.Fatalf("Create s2 failed: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), s2.ID) }()

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
	sm := newSessionManager()

	sess, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), sess.ID) }()

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
	sm := newSessionManager()

	sess, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), sess.ID) }()

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
	sm := newSessionManager()

	sess, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = sm.Delete(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("session should not exist after Delete")
	}

	err = sm.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Delete should fail for nonexistent session")
	}
}

// [REQ:P0-002c] Terminal Resize Handling — lease holder applies its declared size
func TestSession_Resize(t *testing.T) {
	sm := newSessionManager()

	sess, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), sess.ID) }()

	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)

	sess.DeclareSize(sub.OutputCh, 120, 40)
	if err := sess.Resize(sub.OutputCh, 120, 40); err != nil {
		t.Fatalf("Resize: %v", err)
	}

	cols, rows := sess.EffectiveSize()
	if cols != 120 {
		t.Errorf("expected cols=120, got %d", cols)
	}
	if rows != 40 {
		t.Errorf("expected rows=40, got %d", rows)
	}
}

// [REQ:P0-002c] A follower cannot resize until it acquires the size lease.
func TestSession_NonLeaderResizeIsRejected(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create(context.Background(), "/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub1 := sess.Subscribe()
	defer sess.Unsubscribe(sub1.OutputCh)
	sub2 := sess.Subscribe()
	defer sess.Unsubscribe(sub2.OutputCh)

	// The first subscriber owns the initial lease.
	sess.DeclareSize(sub1.OutputCh, 120, 40)
	if err := sess.Resize(sub1.OutputCh, 120, 40); err != nil {
		t.Fatalf("leader resize: %v", err)
	}
	cols, rows := sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		t.Errorf("after first resize: expected 120x40, got %dx%d", cols, rows)
	}

	// A follower may declare a new size but cannot apply it.
	sess.DeclareSize(sub2.OutputCh, 60, 20)
	if err := sess.Resize(sub2.OutputCh, 60, 20); err == nil {
		t.Fatal("follower resize unexpectedly succeeded")
	}
	cols, rows = sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		t.Errorf("follower resize changed session: got %dx%d", cols, rows)
	}

	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002c] Input attention transfers the size lease to the follower's
// previously declared grid; declaration alone never changes the PTY.
func TestSession_StdinFromFollowerAcquiresLease(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	sess, err := sm.Create(context.Background(), "/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	leader, follower := sess.Subscribe(), sess.Subscribe()
	defer sess.Unsubscribe(leader.OutputCh)
	defer sess.Unsubscribe(follower.OutputCh)
	sess.DeclareSize(leader.OutputCh, 120, 40)
	if err := sess.Resize(leader.OutputCh, 120, 40); err != nil {
		t.Fatalf("leader resize: %v", err)
	}
	sess.DeclareSize(follower.OutputCh, 60, 20)
	if err := sess.AcquireLease(follower.OutputCh, session.LeaseReasonInput); err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}
	if !sess.HoldsLease(follower.OutputCh) {
		t.Fatal("follower did not receive input lease")
	}
	if cols, rows := sess.EffectiveSize(); cols != 60 || rows != 20 {
		t.Fatalf("input lease size = %dx%d, want 60x20", cols, rows)
	}
}

// [REQ:P0-002c] Every viewer receives the one authoritative terminal grid.
func TestSession_ResizePublishesToAllSubscribers(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	sess, err := sm.Create(context.Background(), "/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	first, second := sess.Subscribe(), sess.Subscribe()
	defer sess.Unsubscribe(first.OutputCh)
	defer sess.Unsubscribe(second.OutputCh)
	sess.DeclareSize(first.OutputCh, 120, 40)
	if err := sess.Resize(first.OutputCh, 120, 40); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	for name, ch := range map[string]chan [2]uint16{"first": first.SizeCh, "second": second.SizeCh} {
		select {
		case got := <-ch:
			if got != [2]uint16{120, 40} {
				t.Errorf("%s got %v", name, got)
			}
		case <-time.After(time.Second):
			t.Errorf("%s did not receive authoritative size", name)
		}
	}
}

func TestSession_LeaseMovesOnLeaderDisconnect(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	sess, err := sm.Create(context.Background(), "/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	first, second := sess.Subscribe(), sess.Subscribe()
	sess.DeclareSize(second.OutputCh, 100, 30)
	sess.Unsubscribe(first.OutputCh)
	if !sess.HoldsLease(second.OutputCh) {
		t.Fatal("remaining subscriber did not receive lease")
	}
	cols, rows := sess.EffectiveSize()
	if cols != 100 || rows != 30 {
		t.Fatalf("expected transferred declared size 100x30, got %dx%d", cols, rows)
	}
	sess.Unsubscribe(second.OutputCh)
}

// [REQ:P0-002c] PTY size unchanged when all clients disconnect
func TestSession_NoClients_SizeUnchanged(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create(context.Background(), "/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe()
	sess.DeclareSize(sub.OutputCh, 120, 40)
	if err := sess.Resize(sub.OutputCh, 120, 40); err != nil {
		t.Fatalf("Resize: %v", err)
	}
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
	sm := newSessionManager()

	sess, err := sm.Create(context.Background(), "/bin/sh", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), sess.ID) }()

	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)

	// Write to stdin - the shell should echo something back
	err = sess.SendInput(session.InputText("echo hello\n"))
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
	sm := newSessionManager()
	sess, err := sm.Create(context.Background(), "/bin/sh", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), sess.ID) }()

	if err := sess.SendInput(session.InputText("echo offline_marker\n")); err != nil {
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
	sm := newSessionManager()
	sess, err := sm.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(context.Background(), sess.ID) }()
	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)
	if len(sub.Snapshot) == 0 {
		t.Fatal("snapshot empty on fresh session")
	}
	if !bytes.HasPrefix(sub.Snapshot, []byte("\x1b[?1049l\x1bc")) {
		t.Fatalf("snapshot must start with alt-exit + full reset; got %q", sub.Snapshot[:16])
	}
}
