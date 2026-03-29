package main

import (
	"io"
	"sync"
	"testing"
	"time"
)

// --- Session Recovery Tests ---
// These test the Recover() path that runs on API startup to restore
// persistent tmux sessions that survived a restart.

func TestRecover_NoDetachedSessions(t *testing.T) {
	store := NewInMemorySessionStore()
	reg := NewBackendRegistry()
	sm := NewSessionManagerWithFactory(func(spec SessionLaunchSpec) (PTY, error) {
		return newFakePTYWithOutput(), nil
	})

	report := sm.Recover(store, reg)

	if report.Recovered != 0 || report.OrphanedMetadata != 0 || report.OrphanedTmux != 0 {
		t.Errorf("expected empty report, got recovered=%d orphanMeta=%d orphanTmux=%d",
			report.Recovered, report.OrphanedMetadata, report.OrphanedTmux)
	}
}

func TestRecover_OrphanedMetadata_NoTmuxSession(t *testing.T) {
	// Metadata exists in the store but the tmux session is gone.
	// Recovery should clean up the stale metadata.
	store := NewInMemorySessionStore()
	_ = store.Save(SessionMetadata{
		ID:       "dead-session",
		Backend:  BackendPersistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: true,
	})

	reg := NewBackendRegistry()
	sm := NewSessionManagerWithFactory(nil)

	report := sm.Recover(store, reg)

	if report.OrphanedMetadata != 1 {
		t.Errorf("expected 1 orphaned metadata, got %d", report.OrphanedMetadata)
	}
	if report.Recovered != 0 {
		t.Errorf("expected 0 recovered, got %d", report.Recovered)
	}

	// Verify metadata was cleaned up
	_, err := store.Get("dead-session")
	if err == nil {
		t.Error("expected stale metadata to be deleted from store")
	}
}

func TestRecover_StandardSessionsIgnored(t *testing.T) {
	// Standard (non-detached) sessions should not appear in ListDetached
	store := NewInMemorySessionStore()
	_ = store.Save(SessionMetadata{
		ID:       "standard-session",
		Backend:  BackendStandard,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: false, // not detached
	})

	reg := NewBackendRegistry()
	sm := NewSessionManagerWithFactory(nil)

	report := sm.Recover(store, reg)

	// Standard sessions aren't in ListDetached, so nothing to recover
	if report.Recovered != 0 || report.OrphanedMetadata != 0 {
		t.Errorf("expected no activity for standard sessions, got recovered=%d orphaned=%d",
			report.Recovered, report.OrphanedMetadata)
	}
}

// --- Session Store Tests ---

func TestInMemorySessionStore_SaveAndGet(t *testing.T) {
	store := NewInMemorySessionStore()

	meta := SessionMetadata{
		ID:      "test-1",
		Backend: BackendPersistent,
		Shell:   "/bin/bash",
		Cols:    80,
		Rows:    24,
		Policy: ExpirationPolicy{
			Mode: PolicyNever,
		},
		Created:  time.Now(),
		Detached: true,
	}

	if err := store.Save(meta); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get("test-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Backend != BackendPersistent {
		t.Errorf("Backend = %q, want %q", got.Backend, BackendPersistent)
	}
	if !got.Detached {
		t.Error("expected Detached=true")
	}
}

func TestInMemorySessionStore_ListDetached(t *testing.T) {
	store := NewInMemorySessionStore()

	_ = store.Save(SessionMetadata{ID: "s1", Detached: true, Backend: BackendPersistent})
	_ = store.Save(SessionMetadata{ID: "s2", Detached: false, Backend: BackendStandard})
	_ = store.Save(SessionMetadata{ID: "s3", Detached: true, Backend: BackendPersistent})

	detached, err := store.ListDetached()
	if err != nil {
		t.Fatalf("ListDetached: %v", err)
	}
	if len(detached) != 2 {
		t.Fatalf("expected 2 detached sessions, got %d", len(detached))
	}

	ids := map[string]bool{}
	for _, m := range detached {
		ids[m.ID] = true
	}
	if !ids["s1"] || !ids["s3"] {
		t.Errorf("expected s1 and s3, got %v", ids)
	}
}

func TestInMemorySessionStore_Delete(t *testing.T) {
	store := NewInMemorySessionStore()

	_ = store.Save(SessionMetadata{ID: "del-me", Detached: true})

	if err := store.Delete("del-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Get("del-me")
	if err == nil {
		t.Error("expected Get to fail after Delete")
	}
}

func TestInMemorySessionStore_UpdatePolicy(t *testing.T) {
	store := NewInMemorySessionStore()

	_ = store.Save(SessionMetadata{
		ID:     "policy-test",
		Policy: ExpirationPolicy{Mode: PolicyNever},
	})

	newPolicy := ExpirationPolicy{Mode: PolicyPreset, Duration: "1h"}
	if err := store.UpdatePolicy("policy-test", newPolicy); err != nil {
		t.Fatalf("UpdatePolicy: %v", err)
	}

	got, _ := store.Get("policy-test")
	if got.Policy.Mode != PolicyPreset || got.Policy.Duration != "1h" {
		t.Errorf("policy = %+v, want mode=preset duration=1h", got.Policy)
	}
}

// --- ReadLoop Re-attach Tests ---
// These verify that persistent sessions attempt re-attach when the
// PTY read loop encounters an error.

// reattachableFakePTY simulates a tmux PTY that fails on first read
// then succeeds after "re-attach" (swapping the underlying reader).
type reattachableFakePTY struct {
	mu         sync.Mutex
	readCount  int
	firstRead  io.Reader // returns error on read
	secondRead io.Reader // succeeds after re-attach
	closed     bool
	exitCode   int
}

func (p *reattachableFakePTY) Read(buf []byte) (int, error) {
	p.mu.Lock()
	p.readCount++
	count := p.readCount
	p.mu.Unlock()
	if count <= 1 {
		return p.firstRead.Read(buf)
	}
	return p.secondRead.Read(buf)
}

func (p *reattachableFakePTY) Write(buf []byte) (int, error)   { return len(buf), nil }
func (p *reattachableFakePTY) SetSize(cols, rows uint16) error { return nil }
func (p *reattachableFakePTY) Close() error                    { return nil }
func (p *reattachableFakePTY) Kill() error                     { return nil }
func (p *reattachableFakePTY) ExitCode() int                   { return p.exitCode }
func (p *reattachableFakePTY) HasChildProcess() bool           { return false }

func TestSession_ReadLoop_ExitsOnStandardBackend(t *testing.T) {
	// Standard backend sessions should NOT attempt re-attach.
	r, w := io.Pipe()
	sess := &Session{
		ID:      "std-test",
		Backend: BackendStandard,
		pty: &fakePTY{
			stdoutReader: r,
			stdinWriter:  w,
		},
		clients:             make(map[chan []byte]*ClientInfo),
		exitCh:              make(chan struct{}),
		ptyReadBuffer:       4096,
		offlineBufferMax:    1024 * 1024,
		conversationClients: make(map[chan ConversationEvent]struct{}),
	}

	// Close the pipe to trigger readLoop exit
	w.Close()

	go sess.readLoop()

	select {
	case <-sess.Done():
		// Expected: standard session exits without re-attach
	case <-time.After(2 * time.Second):
		t.Fatal("readLoop did not exit for standard backend")
	}

	if !sess.IsDead() {
		t.Error("expected session to be dead after readLoop exit")
	}
}

// --- Backend Selection in Create Tests ---

func TestSessionManager_Create_DefaultBackend(t *testing.T) {
	sm := NewSessionManagerWithFactory(func(spec SessionLaunchSpec) (PTY, error) {
		return newFakePTYWithOutput(), nil
	})

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	// Default backend when registry is nil should use injected factory
	if sess.Backend != "" {
		t.Errorf("Backend = %q, want empty (no registry)", sess.Backend)
	}
}

func TestSessionManager_Create_WithRegistry_UnknownBackend(t *testing.T) {
	reg := NewBackendRegistry()
	reg.Register(BackendDescriptor{ID: BackendStandard, Available: true},
		func(spec SessionLaunchSpec) (PTY, error) {
			return newFakePTYWithOutput(), nil
		})

	sm := NewSessionManagerWithFactory(nil)
	sm.SetRegistry(reg)

	_, err := sm.Create("", 80, 24, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestSessionManager_Create_WithRegistry_UnavailableBackend(t *testing.T) {
	reg := NewBackendRegistry()
	reg.Register(BackendDescriptor{
		ID:        BackendPersistent,
		Available: false,
		Reason:    "tmux not installed",
	}, nil)

	sm := NewSessionManagerWithFactory(nil)
	sm.SetRegistry(reg)

	_, err := sm.Create("", 80, 24, BackendPersistent, nil)
	if err == nil {
		t.Fatal("expected error for unavailable backend")
	}
}

func TestSessionManager_Create_WithRegistry_PersistsMetadata(t *testing.T) {
	store := NewInMemorySessionStore()
	reg := NewBackendRegistry()
	reg.Register(BackendDescriptor{
		ID:              BackendPersistent,
		Available:       true,
		SurvivesRestart: true,
	}, func(spec SessionLaunchSpec) (PTY, error) {
		return newFakePTYWithOutput(), nil
	})

	sm := NewSessionManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, BackendPersistent, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	if sess.Backend != BackendPersistent {
		t.Errorf("Backend = %q, want %q", sess.Backend, BackendPersistent)
	}

	// Verify metadata was persisted
	meta, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	if meta.Backend != BackendPersistent {
		t.Errorf("stored Backend = %q, want %q", meta.Backend, BackendPersistent)
	}
	if !meta.Detached {
		t.Error("expected stored Detached=true for persistent backend")
	}
}

func TestSessionManager_Create_StandardBackend_NotDetached(t *testing.T) {
	store := NewInMemorySessionStore()
	reg := NewBackendRegistry()
	reg.Register(BackendDescriptor{
		ID:              BackendStandard,
		Available:       true,
		SurvivesRestart: false,
	}, func(spec SessionLaunchSpec) (PTY, error) {
		return newFakePTYWithOutput(), nil
	})

	sm := NewSessionManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, BackendStandard, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	meta, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	if meta.Detached {
		t.Error("expected Detached=false for standard backend")
	}
}

// --- Shutdown Tests ---

func TestSessionManager_Shutdown_PreservesPersistentMetadata(t *testing.T) {
	store := NewInMemorySessionStore()
	reg := NewBackendRegistry()
	reg.Register(BackendDescriptor{
		ID:              BackendPersistent,
		Available:       true,
		SurvivesRestart: true,
	}, func(spec SessionLaunchSpec) (PTY, error) {
		return newFakePTYWithOutput(), nil
	})

	sm := NewSessionManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, BackendPersistent, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify metadata was persisted
	if _, err := store.Get(sess.ID); err != nil {
		t.Fatalf("metadata not saved: %v", err)
	}

	// Shutdown should close PTY but preserve metadata for recovery
	sm.Shutdown()

	// Wait for the session to be removed from the map
	select {
	case <-sess.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("session did not exit after Shutdown")
	}

	// Metadata must survive shutdown (so Recover can find it on next startup)
	if _, err := store.Get(sess.ID); err != nil {
		t.Errorf("persistent session metadata was deleted during shutdown: %v", err)
	}
}

func TestSessionManager_Shutdown_DeletesStandardMetadata(t *testing.T) {
	store := NewInMemorySessionStore()
	reg := NewBackendRegistry()
	reg.Register(BackendDescriptor{
		ID:        BackendStandard,
		Available: true,
	}, func(spec SessionLaunchSpec) (PTY, error) {
		return newFakePTYWithOutput(), nil
	})

	sm := NewSessionManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, BackendStandard, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sessID := sess.ID

	sm.Shutdown()

	select {
	case <-sess.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("session did not exit after Shutdown")
	}

	// The auto-remove goroutine races with sess.Done(); give it time to run.
	deadline := time.After(2 * time.Second)
	for {
		if _, err := store.Get(sessID); err != nil {
			return // metadata deleted as expected
		}
		select {
		case <-deadline:
			t.Fatal("standard session metadata was not deleted during shutdown within timeout")
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestBackendRegistry_ResolveAutoBackend(t *testing.T) {
	t.Run("persistent available", func(t *testing.T) {
		reg := NewBackendRegistry()
		reg.Register(BackendDescriptor{ID: BackendStandard, Available: true}, nil)
		reg.Register(BackendDescriptor{ID: BackendPersistent, Available: true}, nil)

		if got := reg.ResolveAutoBackend(); got != BackendPersistent {
			t.Errorf("ResolveAutoBackend() = %q, want %q", got, BackendPersistent)
		}
	})

	t.Run("persistent unavailable", func(t *testing.T) {
		reg := NewBackendRegistry()
		reg.Register(BackendDescriptor{ID: BackendStandard, Available: true}, nil)
		reg.Register(BackendDescriptor{ID: BackendPersistent, Available: false, Reason: "tmux not installed"}, nil)

		if got := reg.ResolveAutoBackend(); got != BackendStandard {
			t.Errorf("ResolveAutoBackend() = %q, want %q", got, BackendStandard)
		}
	})
}
