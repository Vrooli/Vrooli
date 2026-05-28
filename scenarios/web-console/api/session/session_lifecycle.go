package session

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"web-console/internal/backend"
	"web-console/internal/policy"
	"web-console/internal/pty"
	"web-console/internal/sessionstore"
	"web-console/terminal"
)

// RecoveryReport summarizes the result of session recovery on startup.
type RecoveryReport struct {
	Recovered        int
	AwaitingRecovery int // persistent rows whose tmux session is gone; preserved for explicit recovery
	OrphanedMetadata int // alias: count of rows that ended up awaiting_recovery (compat with metrics/log line)
	OrphanedTmux     int // live wc-<id> tmux sessions we could neither reattach nor persist (left alive, never killed)
	Adopted          int // live wc-<id> tmux sessions with no metadata row that we reattached + persisted
}

// TmuxAttachFunc creates a PTY by attaching to an existing tmux session.
// Returns the PTY interface so tests can substitute fakes.
// Defaults to tmuxAttachAsPTY; overridden in tests.
type TmuxAttachFunc func(sessionName string) (pty.PTY, error)

// TmuxDiscoverFunc discovers surviving tmux sessions by name prefix.
// Defaults to DiscoverTmuxSessions; overridden in tests.
type TmuxDiscoverFunc func() ([]string, error)

// reattachWatchdogInterval controls how often the watchdog checks for
// persistent sessions that have metadata but are not in the active sessions map
// (i.e., the readLoop failed and auto-remove removed them, but the tmux session
// may still be alive). 30 seconds balances responsiveness with low overhead.
const reattachWatchdogInterval = 30 * time.Second

// Shutdown gracefully detaches from all persistent (tmux) sessions without
// killing them, so they survive for recovery on the next startup. Standard
// sessions are killed. Must be called before closing the database.
func (sm *Manager) Shutdown() {
	sm.mu.Lock()
	sm.shuttingDown = true
	// Snapshot sessions under lock; iterate outside lock to avoid holding
	// it while closing PTYs (which triggers readLoop exit + auto-remove).
	snapshot := make([]*Session, 0, len(sm.sessions))
	for _, sess := range sm.sessions {
		snapshot = append(snapshot, sess)
	}
	sm.mu.Unlock()

	// Mark persistent sessions as closing BEFORE closing PTY fds. This
	// tells readLoop to skip re-attach retries, avoiding churn during
	// shutdown where retries would create new attach processes that get
	// immediately killed.
	for _, sess := range snapshot {
		if sess.Backend == backend.Persistent {
			sess.mu.Lock()
			sess.closing = true
			sess.mu.Unlock()
		}
	}

	for _, sess := range snapshot {
		if sess.Backend == backend.Persistent {
			// Close the attach PTY fd — this detaches from the tmux session
			// without killing it. The readLoop will see EOF and exit, but
			// the auto-remove goroutine checks shuttingDown and preserves
			// the metadata.
			_ = sess.pty.Close()
			log.Printf("shutdown: detached from persistent session %s", sess.ID)
		} else {
			_ = sess.pty.Kill()
			_ = sess.pty.Close()
			log.Printf("shutdown: killed standard session %s", sess.ID)
		}
	}
}

// Recover discovers surviving tmux sessions, matches them against persisted
// metadata, and re-registers them. Called once at server startup.
func (sm *Manager) Recover(store sessionstore.Store, registry *backend.Registry) RecoveryReport {
	report := RecoveryReport{}

	// 1. Load persisted metadata for detached sessions
	metaList, err := store.ListDetached()
	if err != nil {
		log.Printf("recovery: failed to list detached sessions: %v", err)
		return report
	}
	metaMap := make(map[string]sessionstore.Metadata, len(metaList))
	for _, m := range metaList {
		metaMap[m.ID] = m
	}

	// 2. Discover live tmux sessions
	tmuxSessions, err := sm.tmuxDiscoverFunc()
	if err != nil {
		log.Printf("recovery: failed to discover tmux sessions: %v", err)
		return report
	}
	tmuxSet := make(map[string]bool, len(tmuxSessions))
	for _, id := range tmuxSessions {
		tmuxSet[id] = true
	}

	// 3. For each metadata row, try to recover or mark for explicit recovery.
	// Per the persistent-session-recovery-hardening plan we never delete
	// detached metadata here: the row is the only DB pointer to the per-session
	// CODEX_HOME and conversation history, so destroying it strands the user.
	for id, meta := range metaMap {
		if !tmuxSet[id] {
			// tmux session is gone (host reboot, scope kill, OOM, manual
			// kill-server). Preserve the row and mark it awaiting_recovery so
			// the recovery endpoints can reattach the agent on demand.
			if err := store.MarkOrphaned(id, time.Now()); err != nil {
				log.Printf("recovery: failed to mark orphan %s: %v", id, err)
			}
			report.AwaitingRecovery++
			report.OrphanedMetadata++
			log.Printf("recovery: marked session %s as awaiting_recovery (tmux gone, agent=%s session_id=%s)",
				id, meta.AgentType, meta.AgentSessionID)
			continue
		}

		if !sm.reattachSession(store, id, meta) {
			// All retries exhausted. Preserve BOTH metadata and the tmux
			// session so the next server restart can try again. Previous
			// behavior deleted metadata here and then killed the tmux
			// session as an "orphan" — permanently destroying a
			// recoverable session on a transient failure.
			log.Printf("recovery: preserving session %s for future recovery (attach failed)", id)
			delete(tmuxSet, id) // prevent orphan handling in step 4
			report.OrphanedMetadata++
			continue
		}

		report.Recovered++
		log.Printf("recovery: recovered session %s (backend=%s)", id, meta.Backend)
		delete(tmuxSet, id)
	}

	// 4. Adopt orphaned tmux sessions (a live wc-<id> with no metadata row).
	// The metadata row is the ONLY DB pointer to a session, so its absence means
	// the row was lost — the DB was moved to a new path, cleared, or corrupted —
	// NOT that the tmux session is rogue. Killing here would destroy a live
	// session and its running agent. This is exactly the 2026-05-27 data-loss
	// incident: a storage-path migration left recovery reading an empty DB and it
	// killed 13 live Claude panes as "orphans". Instead, synthesize a default
	// metadata row and reattach so the live session (and its agent) survive and
	// become first-class again. We never kill a session on the configured socket.
	for id := range tmuxSet {
		shell, cols, rows := sm.applySessionDefaults("", 0, 0)
		meta := sessionstore.Metadata{
			ID:       id,
			Backend:  backend.Persistent,
			Shell:    shell,
			Cols:     cols,
			Rows:     rows,
			Policy:   policy.Policy{Mode: policy.Never},
			Created:  time.Now(),
			Detached: true,
			Status:   sessionstore.StatusLive,
		}
		if err := store.Save(meta); err != nil {
			// Could not persist — leave the tmux session ALIVE for a later
			// attempt rather than killing it.
			log.Printf("recovery: failed to persist adopted session %s (left alive): %v", id, err)
			report.OrphanedTmux++
			continue
		}
		if sm.reattachSession(store, id, meta) {
			report.Adopted++
			log.Printf("recovery: adopted orphaned tmux session %s (metadata was missing; reattached and persisted)", id)
		} else {
			// Persisted but couldn't attach right now; the watchdog and the next
			// restart will retry. Never kill.
			report.OrphanedTmux++
			log.Printf("recovery: orphaned tmux session %s persisted but reattach failed; preserved for retry", id)
		}
	}

	// 5. Record recovery metrics and emit events for observability.
	if sm.metrics != nil {
		sm.metrics.RecoveryRecovered.Add(int64(report.Recovered))
		sm.metrics.RecoveryOrphanedMeta.Add(int64(report.OrphanedMetadata))
		sm.metrics.RecoveryOrphanedTmux.Add(int64(report.OrphanedTmux))
	}
	if sm.events != nil {
		sm.events.Emit("session.recovery_complete", "", map[string]string{
			"recovered":         fmt.Sprintf("%d", report.Recovered),
			"awaiting_recovery": fmt.Sprintf("%d", report.AwaitingRecovery),
			"orphaned_meta":     fmt.Sprintf("%d", report.OrphanedMetadata),
			"orphaned_tmux":     fmt.Sprintf("%d", report.OrphanedTmux),
			"adopted":           fmt.Sprintf("%d", report.Adopted),
			"metadata_entries":  fmt.Sprintf("%d", len(metaList)),
			"tmux_sessions":     fmt.Sprintf("%d", len(tmuxSessions)),
		})
	}

	return report
}

// reattachSession reattaches to a surviving wc-<id> tmux session and registers
// it as a live in-memory session, marking the metadata row live. It returns
// false when the attach fails after retries; callers must then preserve both
// the tmux session and its metadata row for a later attempt (never kill). Used
// by Recover for both known-metadata rows (step 3) and adopted orphans (step 4).
func (sm *Manager) reattachSession(store sessionstore.Store, id string, meta sessionstore.Metadata) bool {
	sessionName := sm.tmuxSessionPrefix + id
	// Re-apply tmux options (mouse mode, history limit) in case they were set by
	// an older version that didn't configure them.
	sm.applyTmuxOptionsFunc(sessionName)

	// Re-attach with retries. A transient failure (tmux server briefly busy at
	// startup) must not permanently destroy the session.
	var p pty.PTY
	var attachErr error
	for attempt := 0; attempt <= tmuxReattachMaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(tmuxReattachBaseDelay << (attempt - 1))
		}
		p, attachErr = sm.tmuxAttachFunc(sessionName)
		if attachErr == nil {
			break
		}
		log.Printf("recovery: reattach session %s attempt %d/%d failed: %v",
			id, attempt+1, tmuxReattachMaxRetries+1, attachErr)
	}
	if attachErr != nil {
		return false
	}

	sess := &Session{
		ID:                      id,
		Shell:                   meta.Shell,
		CreatedAt:               meta.Created,
		Cols:                    meta.Cols,
		Rows:                    meta.Rows,
		Backend:                 meta.Backend,
		pty:                     p,
		policy:                  meta.Policy,
		clients:                 make(map[chan []byte]*ClientInfo),
		exitCh:                  make(chan struct{}),
		emu:                     terminal.New(terminal.Options{Cols: int(meta.Cols), Rows: int(meta.Rows), ScrollbackLines: sm.cfg.TerminalScrollbackLines}),
		ptyReadBuffer:           sm.cfg.PTYReadBuffer,
		clientChannelBuffer:     sm.cfg.ClientChannelBuffer,
		coalesceNotifyThreshold: sm.cfg.CoalesceNotifyThreshold,
		sigwinchCooldown:        time.Duration(sm.cfg.SIGWINCHCooldownMs) * time.Millisecond,
		recovered:               true,
		reattachFunc:            sm.tmuxAttachFunc,
		sessionPrefix:           sm.tmuxSessionPrefix,
		metrics:                 sm.metrics,
	}

	sm.mu.Lock()
	sm.sessions[id] = sess
	sm.mu.Unlock()
	if sm.onSessionCreate != nil {
		sm.onSessionCreate(id)
	}

	sess.startAnsiResponder()
	go sess.readLoop()
	go func(sessID string, bid backend.ID) {
		<-sess.Done()
		log.Printf("session %s: recovered process exited (backend=%s)", sessID, bid)
		sm.mu.Lock()
		delete(sm.sessions, sessID)
		sm.mu.Unlock()
		if sm.onSessionDelete != nil {
			sm.onSessionDelete(sessID)
		}
		// Persistent sessions: preserve metadata for future recovery.
		// Standard sessions: delete metadata (they cannot survive).
		if sm.store != nil && bid != backend.Persistent {
			_ = sm.store.Delete(sessID)
		}
		uploadDir := filepath.Join(sm.uploadDirFunc(), sessID)
		if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
			log.Printf("session %s: failed to clean up upload dir: %v", sessID, err)
		}
	}(id, meta.Backend)

	// Reattach succeeded: ensure the row is marked live (covers the case where
	// it was previously awaiting_recovery and tmux came back, e.g. the user
	// restarted the wc-tmux-server scope by hand).
	if err := store.MarkLive(id); err != nil {
		log.Printf("recovery: failed to mark session %s live: %v", id, err)
	}
	return true
}

// StartReattachWatchdog launches a background goroutine that periodically
// checks for persistent sessions with metadata but no active in-memory session.
// When found, it attempts to re-attach to the tmux session and re-register it.
// This handles the case where a transient failure kills the attach process
// during normal operation — the session recovers without requiring a full
// server restart.
func (sm *Manager) StartReattachWatchdog() {
	sm.mu.Lock()
	if sm.reattachStopCh != nil {
		sm.mu.Unlock()
		return
	}
	sm.reattachStopCh = make(chan struct{})
	sm.mu.Unlock()

	go sm.reattachWatchdogLoop()
}

// StopReattachWatchdog terminates the background re-attach watchdog.
func (sm *Manager) StopReattachWatchdog() {
	sm.mu.Lock()
	if sm.reattachStopCh != nil {
		close(sm.reattachStopCh)
		sm.reattachStopCh = nil
	}
	sm.mu.Unlock()
}

func (sm *Manager) reattachWatchdogLoop() {
	ticker := time.NewTicker(reattachWatchdogInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sm.reattachOrphanedSessions()
		case <-sm.reattachStopCh:
			return
		}
	}
}

// reattachOrphanedSessions finds persistent sessions with metadata in the
// store but no corresponding entry in the active sessions map, and attempts
// to re-attach them. This is a lightweight version of Recover that runs
// during normal operation.
func (sm *Manager) reattachOrphanedSessions() {
	if sm.store == nil {
		return
	}
	metaList, err := sm.store.ListDetached()
	if err != nil {
		return
	}

	for _, meta := range metaList {
		// Skip sessions that are already active
		sm.mu.RLock()
		_, active := sm.sessions[meta.ID]
		shutting := sm.shuttingDown
		sm.mu.RUnlock()
		if active || shutting {
			continue
		}

		sessionName := sm.tmuxSessionPrefix + meta.ID
		p, attachErr := sm.tmuxAttachFunc(sessionName)
		if attachErr != nil {
			// tmux session is gone — clean up stale metadata
			_ = sm.store.Delete(meta.ID)
			log.Printf("reattach-watchdog: session %s tmux session gone, cleaned up metadata", meta.ID)
			continue
		}

		sess := &Session{
			ID:                      meta.ID,
			Shell:                   meta.Shell,
			CreatedAt:               meta.Created,
			Cols:                    meta.Cols,
			Rows:                    meta.Rows,
			Backend:                 meta.Backend,
			pty:                     p,
			policy:                  meta.Policy,
			clients:                 make(map[chan []byte]*ClientInfo),
			exitCh:                  make(chan struct{}),
			emu:                     terminal.New(terminal.Options{Cols: int(meta.Cols), Rows: int(meta.Rows), ScrollbackLines: sm.cfg.TerminalScrollbackLines}),
			ptyReadBuffer:           sm.cfg.PTYReadBuffer,
			clientChannelBuffer:     sm.cfg.ClientChannelBuffer,
			coalesceNotifyThreshold: sm.cfg.CoalesceNotifyThreshold,
			sigwinchCooldown:        time.Duration(sm.cfg.SIGWINCHCooldownMs) * time.Millisecond,
			recovered:               true,
			reattachFunc:            sm.tmuxAttachFunc,
			sessionPrefix:           sm.tmuxSessionPrefix,
			metrics:                 sm.metrics,
		}

		sm.mu.Lock()
		// Double-check another goroutine didn't re-add it
		if _, exists := sm.sessions[meta.ID]; exists {
			sm.mu.Unlock()
			_ = p.Close()
			continue
		}
		sm.sessions[meta.ID] = sess
		sm.mu.Unlock()
		if sm.onSessionCreate != nil {
			sm.onSessionCreate(meta.ID)
		}

		sess.startAnsiResponder()
		go sess.readLoop()
		go func(sessID string, bid backend.ID) {
			<-sess.Done()
			log.Printf("session %s: re-attached process exited (backend=%s)", sessID, bid)
			sm.mu.Lock()
			delete(sm.sessions, sessID)
			sm.mu.Unlock()
			if sm.onSessionDelete != nil {
				sm.onSessionDelete(sessID)
			}
			if sm.store != nil && bid != backend.Persistent {
				_ = sm.store.Delete(sessID)
			}
			uploadDir := filepath.Join(sm.uploadDirFunc(), sessID)
			if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
				log.Printf("session %s: failed to clean up upload dir: %v", sessID, err)
			}
		}(meta.ID, meta.Backend)

		log.Printf("reattach-watchdog: re-attached session %s", meta.ID)
		if sm.events != nil {
			sm.events.Emit("session.reattach_watchdog", meta.ID, map[string]string{
				"backend": string(meta.Backend),
			})
		}
		if sm.metrics != nil {
			sm.metrics.ReattachSuccesses.Add(1)
		}
	}
}
