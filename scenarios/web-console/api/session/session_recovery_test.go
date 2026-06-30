package session

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/policy"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/terminal"
)

// --- Session Recovery Tests ---
// These test the Recover() path that runs on API startup to restore
// persistent tmux sessions that survived a restart.

func TestRecover_NoDetachedSessions(t *testing.T) {
	useIsolatedSessionState(t)
	useIsolatedTmuxSocket(t)

	store := sessionstore.NewInMemory()
	reg := backend.New()
	sm := NewManagerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	report := sm.Recover(store, reg)

	if report.Recovered != 0 || report.OrphanedMetadata != 0 || report.OrphanedTmux != 0 {
		t.Errorf("expected empty report, got recovered=%d orphanMeta=%d orphanTmux=%d",
			report.Recovered, report.OrphanedMetadata, report.OrphanedTmux)
	}
}

func TestRecover_OrphanedMetadata_NoTmuxSession_PreservesRow(t *testing.T) {
	useIsolatedSessionState(t)
	useIsolatedTmuxSocket(t)

	// Persistent metadata exists but the tmux session is gone (host reboot,
	// scope kill, OOM, manual kill-server). Recovery must preserve the row
	// and mark it awaiting_recovery so the recovery endpoints can reattach
	// the agent on demand. Deleting the row strands the user's CODEX_HOME
	// and conversation history with no DB pointer to find them.
	store := sessionstore.NewInMemory()
	_ = store.Save(sessionstore.Metadata{
		ID:             "dead-session",
		Backend:        backend.Persistent,
		Shell:          "/bin/bash",
		Cols:           80,
		Rows:           24,
		Created:        time.Now(),
		Detached:       true,
		AgentType:      sessionstore.AgentCodex,
		AgentSessionID: "codex-uuid-123",
	})

	reg := backend.New()
	sm := NewManagerWithFactory(nil)

	report := sm.Recover(store, reg)

	if report.AwaitingRecovery != 1 {
		t.Errorf("expected 1 awaiting_recovery, got %d", report.AwaitingRecovery)
	}
	if report.OrphanedMetadata != 1 {
		t.Errorf("expected OrphanedMetadata alias to be 1, got %d", report.OrphanedMetadata)
	}
	if report.Recovered != 0 {
		t.Errorf("expected 0 recovered, got %d", report.Recovered)
	}

	got, err := store.Get("dead-session")
	if err != nil {
		t.Fatalf("expected metadata to be preserved, got: %v", err)
	}
	if got.Status != sessionstore.StatusAwaitingRecovery {
		t.Errorf("expected status=awaiting_recovery, got %q", got.Status)
	}
	if got.OrphanedAt.IsZero() {
		t.Errorf("expected orphaned_at to be set")
	}
	if got.AgentSessionID != "codex-uuid-123" {
		t.Errorf("agent_session_id was clobbered: %q", got.AgentSessionID)
	}
}

func TestRecover_AwaitingRecoveryReattachOnNextStart(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	useIsolatedSessionState(t)
	useIsolatedTmuxSocket(t)

	// Simulate the case where a row was previously marked awaiting_recovery
	// (tmux had died) and tmux comes back before recovery is invoked
	// explicitly. Next Recover() should transition the row back to live.
	store := sessionstore.NewInMemory()
	id := "reattach-test"
	_ = store.Save(sessionstore.Metadata{
		ID:       id,
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now().Add(-time.Hour),
		Detached: true,
		Status:   sessionstore.StatusLive, // ListDetached requires live
	})

	// Create a real tmux session under the isolated socket so attach succeeds.
	sessionName := tmuxSessionPrefix + id
	if err := tmuxCmd("new-session", "-d", "-s", sessionName, "/bin/sh").Run(); err != nil {
		t.Fatalf("create tmux session: %v", err)
	}
	t.Cleanup(func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() })

	// First mark it orphaned to simulate previous recovery cycle.
	if err := store.MarkOrphaned(id, time.Now()); err != nil {
		t.Fatalf("MarkOrphaned: %v", err)
	}
	// Then "live" again so the next Recover() picks it up via ListDetached.
	if err := store.MarkLive(id); err != nil {
		t.Fatalf("MarkLive: %v", err)
	}

	sm := NewManagerWithFactory(nil)
	setupRealTmuxHooks(t, sm)
	report := sm.Recover(store, backend.New())

	if report.Recovered != 1 {
		t.Errorf("expected 1 recovered, got %d", report.Recovered)
	}
	got, _ := store.Get(id)
	if got.Status != sessionstore.StatusLive {
		t.Errorf("expected status=live after reattach, got %q", got.Status)
	}
	if !got.OrphanedAt.IsZero() {
		t.Errorf("expected orphaned_at to be cleared")
	}
}

func TestRecover_StandardSessionsIgnored(t *testing.T) {
	useIsolatedSessionState(t)
	useIsolatedTmuxSocket(t)

	// Standard (non-detached) sessions should not appear in ListDetached
	store := sessionstore.NewInMemory()
	_ = store.Save(sessionstore.Metadata{
		ID:       "standard-session",
		Backend:  backend.Standard,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: false, // not detached
	})

	reg := backend.New()
	sm := NewManagerWithFactory(nil)

	report := sm.Recover(store, reg)

	// Standard sessions aren't in ListDetached, so nothing to recover
	if report.Recovered != 0 || report.OrphanedMetadata != 0 {
		t.Errorf("expected no activity for standard sessions, got recovered=%d orphaned=%d",
			report.Recovered, report.OrphanedMetadata)
	}
}

func TestRecover_UsesConfiguredTmuxSocketIsolation(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	useIsolatedSessionState(t)
	testSocket := useIsolatedTmuxSocket(t)
	liveSocket := "wc-live-" + sanitizeTestIdentifier(t.TempDir())
	liveSessionName := tmuxSessionPrefix + "live-simulated"
	testSessionName := tmuxSessionPrefix + "test-simulated"

	if err := tmuxCmdForSocket(liveSocket, "new-session", "-d", "-s", liveSessionName, "/bin/sh").Run(); err != nil {
		t.Fatalf("create live-simulated tmux session: %v", err)
	}
	t.Cleanup(func() { _ = tmuxCmdForSocket(liveSocket, "kill-session", "-t", liveSessionName).Run() })

	if err := tmuxCmdForSocket(testSocket, "new-session", "-d", "-s", testSessionName, "/bin/sh").Run(); err != nil {
		t.Fatalf("create isolated test tmux session: %v", err)
	}

	sm := NewManagerWithFactory(nil)
	setupRealTmuxHooks(t, sm)
	report := sm.Recover(sessionstore.NewInMemory(), backend.New())

	// The configured-socket session has no metadata row. Recovery must ADOPT it
	// (reattach + persist), never kill it — killing a live session whose
	// metadata was merely lost is the 2026-05-27 data-loss bug.
	if report.Adopted != 1 {
		t.Fatalf("expected the configured-socket session to be adopted, got adopted=%d orphanedTmux=%d", report.Adopted, report.OrphanedTmux)
	}
	if report.OrphanedTmux != 0 {
		t.Fatalf("adopted sessions must not be counted as orphaned/killed, got %d", report.OrphanedTmux)
	}

	// Socket isolation: recovery must not touch sessions on other sockets.
	if err := tmuxCmdForSocket(liveSocket, "has-session", "-t", liveSessionName).Run(); err != nil {
		t.Fatalf("recovery touched a different tmux socket and killed a live-simulated session: %v", err)
	}

	// The adopted session on the configured socket must SURVIVE recovery.
	if err := tmuxCmdForSocket(testSocket, "has-session", "-t", testSessionName).Run(); err != nil {
		t.Fatalf("adopted configured-socket session should survive recovery, but it is gone: %v", err)
	}
	t.Cleanup(func() { _ = tmuxCmdForSocket(testSocket, "kill-session", "-t", testSessionName).Run() })
}

func TestRecover_AdoptsOrphanedTmuxWithoutMetadata(t *testing.T) {
	// REGRESSION (2026-05-27): a storage-path migration left recovery reading an
	// empty DB; it then killed every live wc-<id> tmux session as an "orphan with
	// no metadata", destroying 13 running Claude panes. Recovery must instead
	// ADOPT such sessions — persist a fresh metadata row and reattach — never
	// kill, because the missing metadata means the row was lost, not the session.
	useIsolatedSessionState(t)

	store := sessionstore.NewInMemory()
	killed := map[string]bool{}

	sm := NewManagerWithFactory(nil)
	sm.tmuxDiscoverFunc = func() ([]string, error) { return []string{"ghost-session"}, nil }
	sm.tmuxAttachFunc = func(string) (pty.PTY, error) { return ptyfake.NewFakePTYWithOutput(), nil }
	sm.killTmuxSessionFunc = func(name string) { killed[name] = true }

	report := sm.Recover(store, backend.New())

	if report.Adopted != 1 {
		t.Fatalf("Adopted = %d, want 1", report.Adopted)
	}
	if report.OrphanedTmux != 0 {
		t.Errorf("OrphanedTmux = %d, want 0 (adoption must not kill)", report.OrphanedTmux)
	}
	if len(killed) != 0 {
		t.Errorf("recovery killed tmux sessions %v; adoption must never kill", killed)
	}
	// A metadata row must now exist so the session is first-class and persists
	// across future restarts (persistent metadata is never deleted on exit).
	if _, err := store.Get("ghost-session"); err != nil {
		t.Errorf("adopted session has no persisted metadata row: %v", err)
	}
}

func TestRecoveryProgress_TracksLifecycleAndCounts(t *testing.T) {
	// Recovery now runs asynchronously so the HTTP listener comes up without
	// waiting. RecoveryProgress() is what the API exposes (and the UI shows) so
	// a client opening the app mid-recovery sees an honest state. Assert the
	// snapshot reflects begin → counts → done.
	useIsolatedSessionState(t)
	useIsolatedTmuxSocket(t)

	store := sessionstore.NewInMemory()
	// One detached row whose tmux survives (will be Recovered) and one whose
	// tmux is gone (will be AwaitingRecovery). Total counts detached rows only.
	_ = store.Save(sessionstore.Metadata{
		ID: "alive", Backend: backend.Persistent, Shell: "/bin/bash",
		Cols: 80, Rows: 24, Created: time.Now(), Detached: true,
	})
	_ = store.Save(sessionstore.Metadata{
		ID: "dead", Backend: backend.Persistent, Shell: "/bin/bash",
		Cols: 80, Rows: 24, Created: time.Now(), Detached: true,
	})

	sm := NewManagerWithFactory(nil)
	sm.tmuxDiscoverFunc = func() ([]string, error) { return []string{"alive"}, nil }
	sm.tmuxAttachFunc = func(string) (pty.PTY, error) { return ptyfake.NewFakePTYWithOutput(), nil }

	// Before recovery: not in progress, nothing counted.
	if p := sm.RecoveryProgress(); p.InProgress || p.Total != 0 {
		t.Fatalf("pre-recovery progress = %+v, want zero/not-in-progress", p)
	}

	report := sm.Recover(store, backend.New())

	p := sm.RecoveryProgress()
	if p.InProgress {
		t.Errorf("InProgress = true after Recover returned; want false")
	}
	if p.Total != 2 {
		t.Errorf("Total = %d, want 2 (detached rows)", p.Total)
	}
	if p.Recovered != report.Recovered || p.Recovered != 1 {
		t.Errorf("Recovered snapshot = %d, report = %d, want 1", p.Recovered, report.Recovered)
	}
	if p.AwaitingRecovery != 1 {
		t.Errorf("AwaitingRecovery = %d, want 1", p.AwaitingRecovery)
	}
	if p.StartedAt.IsZero() || p.CompletedAt.IsZero() {
		t.Errorf("StartedAt/CompletedAt must be set after recovery: %+v", p)
	}
	if p.CompletedAt.Before(p.StartedAt) {
		t.Errorf("CompletedAt %v before StartedAt %v", p.CompletedAt, p.StartedAt)
	}
}

// --- Session Store Tests ---

func TestInMemorySessionStore_SaveAndGet(t *testing.T) {
	store := sessionstore.NewInMemory()

	meta := sessionstore.Metadata{
		ID:      "test-1",
		Backend: backend.Persistent,
		Shell:   "/bin/bash",
		Cols:    80,
		Rows:    24,
		Policy: policy.Policy{
			Mode: policy.Never,
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
	if got.Backend != backend.Persistent {
		t.Errorf("Backend = %q, want %q", got.Backend, backend.Persistent)
	}
	if !got.Detached {
		t.Error("expected Detached=true")
	}
}

func TestInMemorySessionStore_ListDetached(t *testing.T) {
	store := sessionstore.NewInMemory()

	_ = store.Save(sessionstore.Metadata{ID: "s1", Detached: true, Backend: backend.Persistent})
	_ = store.Save(sessionstore.Metadata{ID: "s2", Detached: false, Backend: backend.Standard})
	_ = store.Save(sessionstore.Metadata{ID: "s3", Detached: true, Backend: backend.Persistent})

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
	store := sessionstore.NewInMemory()

	_ = store.Save(sessionstore.Metadata{ID: "del-me", Detached: true})

	if err := store.Delete("del-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Get("del-me")
	if err == nil {
		t.Error("expected Get to fail after Delete")
	}
}

func TestInMemorySessionStore_UpdatePolicy(t *testing.T) {
	store := sessionstore.NewInMemory()

	_ = store.Save(sessionstore.Metadata{
		ID:     "policy-test",
		Policy: policy.Policy{Mode: policy.Never},
	})

	newPolicy := policy.Policy{Mode: policy.Preset, Duration: "1h"}
	if err := store.UpdatePolicy("policy-test", newPolicy); err != nil {
		t.Fatalf("UpdatePolicy: %v", err)
	}

	got, _ := store.Get("policy-test")
	if got.Policy.Mode != policy.Preset || got.Policy.Duration != "1h" {
		t.Errorf("policy = %+v, want mode=preset duration=1h", got.Policy)
	}
}

// --- ReadLoop Re-attach Tests ---
// These verify that persistent sessions attempt re-attach when the
// PTY read loop encounters an error.

func TestSession_ReadLoop_ExitsOnStandardBackend(t *testing.T) {
	// Standard backend sessions should NOT attempt re-attach.
	r, w := io.Pipe()
	sess := &Session{
		ID:      "std-test",
		Backend: backend.Standard,
		pty: &ptyfake.FakePTY{
			StdoutReader: r,
			StdinWriter:  w,
		},
		clients:       make(map[chan []byte]*ClientInfo),
		exitCh:        make(chan struct{}),
		ptyReadBuffer: 4096,
		emu:           terminal.New(terminal.Options{Cols: 80, Rows: 24, ScrollbackLines: 1000}),
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
	sm := NewManagerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
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
	reg := backend.New()
	reg.Register(backend.Descriptor{ID: backend.Standard, Available: true},
		func(spec pty.LaunchSpec) (pty.PTY, error) {
			return ptyfake.NewFakePTYWithOutput(), nil
		})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)

	_, err := sm.Create("", 80, 24, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestSessionManager_Create_WithRegistry_UnavailableBackend(t *testing.T) {
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:        backend.Persistent,
		Available: false,
		Reason:    "tmux not installed",
	}, nil)

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)

	_, err := sm.Create("", 80, 24, backend.Persistent, nil)
	if err == nil {
		t.Fatal("expected error for unavailable backend")
	}
}

func TestSessionManager_Create_WithRegistry_PersistsMetadata(t *testing.T) {
	store := sessionstore.NewInMemory()
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:              backend.Persistent,
		Available:       true,
		SurvivesRestart: true,
	}, func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, backend.Persistent, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	if sess.Backend != backend.Persistent {
		t.Errorf("Backend = %q, want %q", sess.Backend, backend.Persistent)
	}

	// Verify metadata was persisted
	meta, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	if meta.Backend != backend.Persistent {
		t.Errorf("stored Backend = %q, want %q", meta.Backend, backend.Persistent)
	}
	if !meta.Detached {
		t.Error("expected stored Detached=true for persistent backend")
	}
}

func TestSessionManager_Create_StandardBackend_NotDetached(t *testing.T) {
	store := sessionstore.NewInMemory()
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:              backend.Standard,
		Available:       true,
		SurvivesRestart: false,
	}, func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, backend.Standard, nil)
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
	store := sessionstore.NewInMemory()
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:              backend.Persistent,
		Available:       true,
		SurvivesRestart: true,
	}, func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, backend.Persistent, nil)
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
	store := sessionstore.NewInMemory()
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:        backend.Standard,
		Available: true,
	}, func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, backend.Standard, nil)
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

// --- Re-attach Retry Tests ---

func TestSession_ReadLoop_RetriesReattachForPersistent(t *testing.T) {
	// When a persistent session's attach process dies, readLoop should retry
	// re-attach with backoff before declaring the session dead.
	r, w := io.Pipe()
	w.Close() // trigger immediate read error

	var mu sync.Mutex
	attempts := 0

	sess := &Session{
		ID:      "retry-test",
		Backend: backend.Persistent,
		pty: &ptyfake.FakePTY{
			StdoutReader: r,
			StdinWriter:  w,
		},
		clients:       make(map[chan []byte]*ClientInfo),
		exitCh:        make(chan struct{}),
		ptyReadBuffer: 4096,
		emu:           terminal.New(terminal.Options{Cols: 80, Rows: 24, ScrollbackLines: 1000}),
		reattachFunc: func(sessionName string) (pty.PTY, error) {
			mu.Lock()
			attempts++
			n := attempts
			mu.Unlock()
			if n < tmuxReattachMaxRetries {
				return nil, fmt.Errorf("transient failure %d", n)
			}
			// Final attempt succeeds — return a PTY that blocks on read
			blockR, _ := io.Pipe()
			return &ptyfake.FakePTY{
				StdoutReader: blockR,
			}, nil
		},
	}

	go sess.readLoop()

	// Give enough time for retries (500ms + 1s + 2s = 3.5s max)
	time.Sleep(5 * time.Second)

	mu.Lock()
	got := attempts
	mu.Unlock()

	if got < tmuxReattachMaxRetries {
		t.Errorf("expected at least %d re-attach attempts, got %d", tmuxReattachMaxRetries, got)
	}

	// Session should NOT be dead (re-attach succeeded on last attempt)
	if sess.IsDead() {
		t.Error("session should still be alive after successful re-attach")
	}
}

func TestSession_ReadLoop_SkipsRetriesWhenClosing(t *testing.T) {
	// When the closing flag is set, readLoop should NOT retry re-attach.
	r, w := io.Pipe()
	w.Close()

	var mu sync.Mutex
	attempts := 0

	sess := &Session{
		ID:      "closing-test",
		Backend: backend.Persistent,
		closing: true, // pre-set as closing
		pty: &ptyfake.FakePTY{
			StdoutReader: r,
			StdinWriter:  w,
		},
		clients:       make(map[chan []byte]*ClientInfo),
		exitCh:        make(chan struct{}),
		ptyReadBuffer: 4096,
		emu:           terminal.New(terminal.Options{Cols: 80, Rows: 24, ScrollbackLines: 1000}),
		reattachFunc: func(sessionName string) (pty.PTY, error) {
			mu.Lock()
			attempts++
			mu.Unlock()
			return nil, fmt.Errorf("should not be called")
		},
	}

	go sess.readLoop()

	select {
	case <-sess.Done():
		// Expected: session exits without retry
	case <-time.After(3 * time.Second):
		t.Fatal("readLoop did not exit promptly when closing")
	}

	mu.Lock()
	got := attempts
	mu.Unlock()

	if got != 0 {
		t.Errorf("expected 0 re-attach attempts when closing, got %d", got)
	}
}

func TestRecover_PreservesSessionOnAttachFailure(t *testing.T) {
	// When tmuxAttach fails during recovery, the session metadata and
	// tmux session should be preserved for future recovery attempts.
	// Previous behavior: deleted metadata and killed the tmux session.
	store := sessionstore.NewInMemory()
	_ = store.Save(sessionstore.Metadata{
		ID:       "stubborn-session",
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: true,
	})

	killedSessions := map[string]bool{}

	sm := NewManagerWithFactory(nil)
	sm.tmuxDiscoverFunc = func() ([]string, error) {
		return []string{"stubborn-session"}, nil
	}
	sm.tmuxAttachFunc = func(sessionName string) (pty.PTY, error) {
		return nil, fmt.Errorf("simulated transient failure")
	}

	report := sm.Recover(store, backend.New())

	// Metadata must be preserved (not deleted)
	if _, err := store.Get("stubborn-session"); err != nil {
		t.Errorf("metadata was deleted; should be preserved for future recovery: %v", err)
	}

	// The tmux session must NOT be killed (it's still in tmux)
	if killedSessions["stubborn-session"] {
		t.Error("tmux session was killed; should be preserved for future recovery")
	}

	// Session should not be registered (we couldn't attach)
	if _, ok := sm.Get("stubborn-session"); ok {
		t.Error("session should not be in active map after failed attach")
	}

	// OrphanedTmux should be 0 (we explicitly prevented orphan kill)
	if report.OrphanedTmux != 0 {
		t.Errorf("OrphanedTmux = %d, want 0", report.OrphanedTmux)
	}

	if report.Recovered != 0 {
		t.Errorf("Recovered = %d, want 0", report.Recovered)
	}
}

func TestRecover_RetriesAttachBeforeGivingUp(t *testing.T) {
	// Recovery should retry tmuxAttach before giving up.
	store := sessionstore.NewInMemory()
	_ = store.Save(sessionstore.Metadata{
		ID:       "retry-recover",
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: true,
	})

	var mu sync.Mutex
	attempts := 0

	sm := NewManagerWithFactory(nil)
	sm.SetStore(store)
	sm.tmuxDiscoverFunc = func() ([]string, error) {
		return []string{"retry-recover"}, nil
	}
	sm.tmuxAttachFunc = func(sessionName string) (pty.PTY, error) {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		if n <= 2 {
			return nil, fmt.Errorf("transient failure %d", n)
		}
		return ptyfake.NewFakePTYWithOutput(), nil
	}

	report := sm.Recover(store, backend.New())

	mu.Lock()
	got := attempts
	mu.Unlock()

	if got < 3 {
		t.Errorf("expected at least 3 attach attempts, got %d", got)
	}

	if report.Recovered != 1 {
		t.Errorf("Recovered = %d, want 1", report.Recovered)
	}

	if _, ok := sm.Get("retry-recover"); !ok {
		t.Error("session should be in active map after successful recovery")
	}
}

func TestSessionManager_Shutdown_SetsClosingBeforeClose(t *testing.T) {
	// Verify the contract: Shutdown() must set the closing flag on persistent
	// sessions BEFORE closing the PTY, so readLoop sees it and skips retries.
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:              backend.Persistent,
		Available:       true,
		SurvivesRestart: true,
	}, func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)

	sess, err := sm.Create("", 80, 24, backend.Persistent, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Before shutdown, closing should be false
	sess.mu.Lock()
	if sess.closing {
		t.Error("closing should be false before Shutdown")
	}
	sess.mu.Unlock()

	sm.Shutdown()

	// After shutdown, closing should have been set
	sess.mu.Lock()
	wasClosing := sess.closing
	sess.mu.Unlock()
	if !wasClosing {
		t.Error("Shutdown did not set closing flag on persistent session")
	}
}

func TestRecoveryMetrics_Emitted(t *testing.T) {
	// Verify recovery emits metrics when a metrics collector is set.
	store := sessionstore.NewInMemory()
	_ = store.Save(sessionstore.Metadata{
		ID:       "metrics-test",
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: true,
	})

	metrics := metrics.New()
	events := events.NewLogger(100)

	sm := NewManagerWithFactory(nil)
	sm.SetMetrics(metrics)
	sm.SetEvents(events)
	sm.tmuxDiscoverFunc = func() ([]string, error) {
		return []string{"metrics-test"}, nil
	}
	sm.tmuxAttachFunc = func(sessionName string) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	}

	report := sm.Recover(store, backend.New())

	if report.Recovered != 1 {
		t.Fatalf("Recovered = %d, want 1", report.Recovered)
	}
	if metrics.RecoveryRecovered.Load() != 1 {
		t.Errorf("RecoveryRecovered metric = %d, want 1", metrics.RecoveryRecovered.Load())
	}

	// Check that a recovery_complete event was emitted
	recent := events.Recent(10)
	found := false
	for _, evt := range recent {
		if evt.Type == "session.recovery_complete" {
			found = true
			if evt.Details["recovered"] != "1" {
				t.Errorf("event recovered detail = %q, want '1'", evt.Details["recovered"])
			}
		}
	}
	if !found {
		t.Error("expected session.recovery_complete event to be emitted")
	}
}

func TestBackendRegistry_ResolveAutoBackend(t *testing.T) {
	t.Run("persistent available", func(t *testing.T) {
		reg := backend.New()
		reg.Register(backend.Descriptor{ID: backend.Standard, Available: true}, nil)
		reg.Register(backend.Descriptor{ID: backend.Persistent, Available: true}, nil)

		if got := reg.ResolveAuto(); got != backend.Persistent {
			t.Errorf("ResolveAutoBackend() = %q, want %q", got, backend.Persistent)
		}
	})

	t.Run("persistent unavailable", func(t *testing.T) {
		reg := backend.New()
		reg.Register(backend.Descriptor{ID: backend.Standard, Available: true}, nil)
		reg.Register(backend.Descriptor{ID: backend.Persistent, Available: false, Reason: "tmux not installed"}, nil)

		if got := reg.ResolveAuto(); got != backend.Standard {
			t.Errorf("ResolveAutoBackend() = %q, want %q", got, backend.Standard)
		}
	})
}

// --- Persistent Session Metadata Preservation Tests ---
// These verify that persistent session metadata is NEVER deleted by the
// auto-remove goroutine, regardless of whether the server is shutting down.
// This is the key fix for the "persistent sessions disappear" bug.

func TestAutoRemove_PreservesMetadata_PersistentSession_NormalOperation(t *testing.T) {
	// REGRESSION: Previously, when a persistent session's readLoop failed
	// during normal operation (not shutdown), the auto-remove goroutine
	// deleted the metadata. This orphaned the tmux session, which would
	// then be killed on the next startup. The fix: always preserve
	// persistent session metadata.
	store := sessionstore.NewInMemory()
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:              backend.Persistent,
		Available:       true,
		SurvivesRestart: true,
	}, func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, backend.Persistent, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sessID := sess.ID

	// Verify metadata was persisted
	if _, err := store.Get(sessID); err != nil {
		t.Fatalf("metadata not saved: %v", err)
	}

	// Simulate the PTY dying (closes the readLoop's read pipe)
	_ = sess.pty.Kill()
	_ = sess.pty.Close()

	// Wait for the session to be removed from the active map
	select {
	case <-sess.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session did not exit")
	}

	// Give the auto-remove goroutine time to run
	time.Sleep(100 * time.Millisecond)

	// Session should be removed from the active map
	if _, ok := sm.Get(sessID); ok {
		t.Error("session should be removed from active map after PTY death")
	}

	// CRITICAL: Metadata must be PRESERVED (not deleted!) so that the
	// re-attach watchdog or next startup recovery can find it.
	if _, err := store.Get(sessID); err != nil {
		t.Errorf("persistent session metadata was deleted during normal operation: %v", err)
	}
}

func TestAutoRemove_DeletesMetadata_StandardSession_NormalOperation(t *testing.T) {
	// Standard sessions should still have their metadata deleted when
	// the readLoop exits, since they cannot be recovered.
	store := sessionstore.NewInMemory()
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:        backend.Standard,
		Available: true,
	}, func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	sess, err := sm.Create("", 80, 24, backend.Standard, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sessID := sess.ID

	_ = sess.pty.Kill()
	_ = sess.pty.Close()

	select {
	case <-sess.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session did not exit")
	}

	// Wait for auto-remove goroutine
	deadline := time.After(2 * time.Second)
	for {
		if _, err := store.Get(sessID); err != nil {
			return // metadata deleted as expected
		}
		select {
		case <-deadline:
			t.Fatal("standard session metadata was not deleted within timeout")
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// --- Re-attach Watchdog Tests ---

func TestReattachWatchdog_RecoversOrphanedSession(t *testing.T) {
	// When a persistent session's metadata exists in the store but the
	// session is not in the active map, the watchdog should re-attach it.
	store := sessionstore.NewInMemory()
	_ = store.Save(sessionstore.Metadata{
		ID:       "watchdog-test",
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Policy:   policy.Policy{Mode: policy.Never},
		Created:  time.Now(),
		Detached: true,
	})

	sm := NewManagerWithFactory(nil)
	sm.SetStore(store)
	sm.tmuxAttachFunc = func(sessionName string) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	}

	// Run the watchdog scan directly (not via the ticker)
	sm.reattachOrphanedSessions()

	// Session should now be in the active map
	sess, ok := sm.Get("watchdog-test")
	if !ok {
		t.Fatal("watchdog did not re-attach the orphaned session")
	}
	if sess.Backend != backend.Persistent {
		t.Errorf("Backend = %q, want %q", sess.Backend, backend.Persistent)
	}
	if !sess.recovered {
		t.Error("expected recovered=true on re-attached session")
	}

	// Clean up
	_ = sm.Delete("watchdog-test")
}

func TestReattachWatchdog_CleansUpWhenTmuxGone(t *testing.T) {
	// When the tmux session is gone (attach fails), the watchdog should
	// clean up the stale metadata.
	store := sessionstore.NewInMemory()
	_ = store.Save(sessionstore.Metadata{
		ID:       "gone-session",
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: true,
	})

	sm := NewManagerWithFactory(nil)
	sm.SetStore(store)
	sm.tmuxAttachFunc = func(sessionName string) (pty.PTY, error) {
		return nil, fmt.Errorf("tmux session gone")
	}

	sm.reattachOrphanedSessions()

	// Session should NOT be in the active map
	if _, ok := sm.Get("gone-session"); ok {
		t.Error("session should not be active when tmux session is gone")
	}

	// Metadata should be cleaned up
	if _, err := store.Get("gone-session"); err == nil {
		t.Error("stale metadata should be deleted when tmux session is gone")
	}
}

func TestReattachWatchdog_SkipsActiveSession(t *testing.T) {
	// When a session is already in the active map, the watchdog should
	// skip it (not create a duplicate).
	store := sessionstore.NewInMemory()
	reg := backend.New()
	reg.Register(backend.Descriptor{
		ID:              backend.Persistent,
		Available:       true,
		SurvivesRestart: true,
	}, func(spec pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sm := NewManagerWithFactory(nil)
	sm.SetRegistry(reg)
	sm.SetStore(store)

	attachCalls := 0
	sm.tmuxAttachFunc = func(sessionName string) (pty.PTY, error) {
		attachCalls++
		return ptyfake.NewFakePTYWithOutput(), nil
	}

	// Create an active session
	sess, err := sm.Create("", 80, 24, backend.Persistent, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	callsBefore := attachCalls

	// Run the watchdog — should skip this session since it's already active
	sm.reattachOrphanedSessions()

	if attachCalls != callsBefore {
		t.Errorf("watchdog called tmuxAttach %d extra times for already-active session",
			attachCalls-callsBefore)
	}
}

func TestReattachWatchdog_SkipsDuringShutdown(t *testing.T) {
	store := sessionstore.NewInMemory()
	_ = store.Save(sessionstore.Metadata{
		ID:       "shutdown-skip",
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: true,
	})

	sm := NewManagerWithFactory(nil)
	sm.SetStore(store)
	sm.tmuxAttachFunc = func(sessionName string) (pty.PTY, error) {
		t.Error("tmuxAttach should not be called during shutdown")
		return ptyfake.NewFakePTYWithOutput(), nil
	}

	// Simulate shutdown state
	sm.mu.Lock()
	sm.shuttingDown = true
	sm.mu.Unlock()

	sm.reattachOrphanedSessions()

	// Metadata should be untouched
	if _, err := store.Get("shutdown-skip"); err != nil {
		t.Errorf("metadata should be preserved during shutdown: %v", err)
	}
}

// --- PTY fd leak test ---

func TestSession_ReadLoop_ClosesOldPTY_OnReattach(t *testing.T) {
	// When readLoop successfully re-attaches, the OLD PTY's fd should be
	// closed to prevent file descriptor leaks.
	r, w := io.Pipe()
	w.Close() // trigger immediate read error

	oldPTY := &ptyfake.FakePTY{
		StdoutReader: r,
		StdinWriter:  w,
	}

	sess := &Session{
		ID:            "fd-leak-test",
		Backend:       backend.Persistent,
		pty:           oldPTY,
		clients:       make(map[chan []byte]*ClientInfo),
		exitCh:        make(chan struct{}),
		ptyReadBuffer: 4096,
		emu:           terminal.New(terminal.Options{Cols: 80, Rows: 24, ScrollbackLines: 1000}),
		reattachFunc: func(sessionName string) (pty.PTY, error) {
			// Return a new PTY that blocks on read
			blockR, _ := io.Pipe()
			return &ptyfake.FakePTY{StdoutReader: blockR}, nil
		},
	}

	go sess.readLoop()

	// Wait for re-attach to complete (retries: 500ms + potentially more)
	time.Sleep(5 * time.Second)

	oldPTY.Mu.Lock()
	wasClosed := oldPTY.Closed
	oldPTY.Mu.Unlock()

	if !wasClosed {
		t.Error("old PTY was not closed after successful re-attach (fd leak)")
	}
}
