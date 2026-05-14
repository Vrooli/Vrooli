package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"web-console/internal/backend"
	"web-console/internal/config"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/policy"
	"web-console/internal/pty"
	"web-console/internal/sessionstore"
	"web-console/terminal"

	"github.com/google/uuid"
)

// Sentinel errors for session operations. Handlers use these to select the
// correct HTTP status code and user-facing message.
var (
	// ErrSessionLimitReached is returned when MaxSessions is configured and
	// the limit has been reached. Maps to HTTP 429.
	ErrSessionLimitReached = errors.New("session limit reached")

	// ErrPTYSpawnFailed wraps PTY creation failures. Maps to HTTP 500.
	ErrPTYSpawnFailed = errors.New("PTY spawn failed")

	// ErrBackendUnavailable is returned when the requested backend is not available.
	ErrBackendUnavailable = errors.New("backend unavailable")

	// ErrBackendUnknown is returned when the requested backend is not registered.
	ErrBackendUnknown = errors.New("unknown backend")
)

// SessionManager tracks all active terminal sessions.
// [REQ:P0-002a] PTY Session Backend
type SessionManager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	ptyFactory   pty.Factory
	cfgMu        sync.RWMutex // protects cfg from concurrent read/write (session-defaults handler vs Create)
	cfg          config.Config
	registry     *backend.Registry
	store        sessionstore.Store
	shuttingDown bool // set by Shutdown(); prevents auto-remove from deleting persistent session metadata

	// Seams for testability: injectable tmux operations. After Session moves
	// to its own sub-package these break the dependency on tmux helpers that
	// live with the persistent backend.
	tmuxAttachFunc       TmuxAttachFunc
	tmuxDiscoverFunc     TmuxDiscoverFunc
	applyTmuxOptionsFunc func(sessionName string)
	killTmuxSessionFunc  func(sessionName string)
	tmuxSessionPrefix    string

	// uploadDirFunc resolves the per-session upload root. Defaults to
	// resolveUploadDir; injectable so internal/session/ doesn't import the
	// upload handler.
	uploadDirFunc func() string

	// envForSession returns the per-session environment variables to inject
	// into the spawned PTY (CODEX_HOME etc). Set by package main; the
	// session package does not know about backend-specific env keys.
	envForSession func(sessionID string) map[string]string

	// onSessionCreate/onSessionDelete are best-effort lifecycle hooks
	// called after a session is added to or removed from the active map.
	// Used by package main to manage the conversation fan-out registry
	// without coupling the session package to ConversationEvent.
	onSessionCreate func(sessionID string)
	onSessionDelete func(sessionID string)

	// Observability: optional metrics and event logger for session lifecycle.
	metrics *metrics.Metrics
	events  *events.Logger

	// reattachStopCh signals the periodic re-attach watchdog to stop.
	reattachStopCh chan struct{}
}

// NewSessionManager creates a new session manager with the default PTY factory
// and configuration loaded from environment variables.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:             make(map[string]*Session),
		ptyFactory:           defaultPTYFactory,
		cfg:                  config.Load(),
		tmuxAttachFunc:       tmuxAttachAsPTY,
		tmuxDiscoverFunc:     DiscoverTmuxSessions,
		applyTmuxOptionsFunc: applyTmuxOptions,
		killTmuxSessionFunc:  func(name string) { _ = tmuxCmd("kill-session", "-t", name).Run() },
		tmuxSessionPrefix:    tmuxSessionPrefix,
		uploadDirFunc:        resolveUploadDir,
		envForSession:        defaultSessionEnv,
	}
}

// NewSessionManagerWithFactory creates a session manager with a custom PTY factory.
// Use this in tests to substitute a fake PTY implementation.
func NewSessionManagerWithFactory(factory pty.Factory) *SessionManager {
	return &SessionManager{
		sessions:             make(map[string]*Session),
		ptyFactory:           factory,
		cfg:                  config.Default(),
		tmuxAttachFunc:       tmuxAttachAsPTY,
		tmuxDiscoverFunc:     DiscoverTmuxSessions,
		applyTmuxOptionsFunc: applyTmuxOptions,
		killTmuxSessionFunc:  func(name string) { _ = tmuxCmd("kill-session", "-t", name).Run() },
		tmuxSessionPrefix:    tmuxSessionPrefix,
		uploadDirFunc:        resolveUploadDir,
		envForSession:        defaultSessionEnv,
	}
}

// SetLifecycleHooks registers callbacks invoked after a session is added to
// or removed from the active map. Either may be nil. Used by package main to
// keep the conversation fan-out registry in sync without coupling the session
// package to ConversationEvent.
func (sm *SessionManager) SetLifecycleHooks(onCreate, onDelete func(sessionID string)) {
	sm.onSessionCreate = onCreate
	sm.onSessionDelete = onDelete
}

// SetRegistry sets the backend registry for backend-aware session creation.
func (sm *SessionManager) SetRegistry(reg *backend.Registry) {
	sm.registry = reg
}

// SetStore sets the session metadata store for persistence.
func (sm *SessionManager) SetStore(store sessionstore.Store) {
	sm.store = store
}

// SetMetrics sets the metrics collector for session lifecycle counters.
func (sm *SessionManager) SetMetrics(m *metrics.Metrics) {
	sm.metrics = m
}

// SetEvents sets the event logger for structured session lifecycle events.
func (sm *SessionManager) SetEvents(el *events.Logger) {
	sm.events = el
}

// GetConfig returns a snapshot of the current configuration. Thread-safe.
func (sm *SessionManager) GetConfig() config.Config {
	sm.cfgMu.RLock()
	defer sm.cfgMu.RUnlock()
	return sm.cfg
}

// SetConfigField updates a mutable config field under the write lock.
// Only use for fields that can change at runtime (default backend/policy).
func (sm *SessionManager) SetConfigField(fn func(cfg *config.Config)) {
	sm.cfgMu.Lock()
	defer sm.cfgMu.Unlock()
	fn(&sm.cfg)
}

// applySessionDefaults fills in zero-valued parameters with configured defaults.
// The convention is: zero/empty from the caller means "use server default".
func (sm *SessionManager) applySessionDefaults(shell string, cols, rows uint16) (string, uint16, uint16) {
	sm.cfgMu.RLock()
	defer sm.cfgMu.RUnlock()
	if shell == "" {
		shell = sm.cfg.DefaultShell
	}
	if cols == 0 {
		cols = sm.cfg.DefaultCols
	}
	if rows == 0 {
		rows = sm.cfg.DefaultRows
	}
	return shell, cols, rows
}

// isSessionLimitReached decides whether a new session should be rejected based
// on the configured MaxSessions cap. A cap of 0 means unlimited.
func (sm *SessionManager) isSessionLimitReached() bool {
	if sm.cfg.MaxSessions <= 0 {
		return false
	}
	sm.mu.RLock()
	count := len(sm.sessions)
	sm.mu.RUnlock()
	return count >= sm.cfg.MaxSessions
}

// Create starts a new shell session with a PTY.
// [REQ:P0-002a] PTY Session Backend
func (sm *SessionManager) Create(shell string, cols, rows uint16, bid backend.ID, pol *policy.Policy) (*Session, error) {
	shell, cols, rows = sm.applySessionDefaults(shell, cols, rows)

	// Resolve backend (read default under lock to avoid data race with settings handler)
	if bid == "" || bid == "auto" {
		sm.cfgMu.RLock()
		bid = backend.ID(sm.cfg.DefaultBackend)
		sm.cfgMu.RUnlock()
	}

	// Look up factory from registry if available, otherwise use injected factory
	var factory pty.Factory
	if sm.registry != nil {
		// Resolve "auto" via registry when it wasn't resolved at startup (e.g. tests)
		if bid == "auto" {
			bid = sm.registry.ResolveAuto()
		}
		desc, ok := sm.registry.Get(bid)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrBackendUnknown, bid)
		}
		if !desc.Available {
			return nil, fmt.Errorf("%w: %s — %s", ErrBackendUnavailable, bid, desc.Reason)
		}
		factory, _ = sm.registry.Factory(bid)
	} else {
		// No registry (test path) — use injected factory, clear backend ID
		if bid == "auto" {
			bid = ""
		}
		factory = sm.ptyFactory
	}

	if sm.isSessionLimitReached() {
		return nil, fmt.Errorf("%w (%d)", ErrSessionLimitReached, sm.cfg.MaxSessions)
	}

	sessionID := uuid.New().String()
	spec := pty.LaunchSpec{
		SessionID: sessionID,
		Shell:     shell,
		Cols:      cols,
		Rows:      rows,
		Env:       sm.envForSession(sessionID),
	}

	p, err := factory(spec)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPTYSpawnFailed, err)
	}

	// Resolve policy (read defaults under lock to avoid data race with settings handler)
	var sessionPolicy policy.Policy
	if pol != nil {
		sessionPolicy = *pol
	} else {
		sm.cfgMu.RLock()
		mode := sm.cfg.DefaultPolicyMode
		dur := sm.cfg.DefaultPolicyDuration
		sm.cfgMu.RUnlock()
		if mode != "" {
			sessionPolicy = policy.Policy{
				Mode:     policy.Mode(mode),
				Duration: dur,
			}
		} else {
			sessionPolicy = policy.Default()
		}
	}

	sess := &Session{
		ID:                      sessionID,
		Shell:                   shell,
		CreatedAt:               time.Now(),
		Cols:                    cols,
		Rows:                    rows,
		Backend:                 bid,
		pty:                     p,
		policy:                  sessionPolicy,
		clients:                 make(map[chan []byte]*ClientInfo),
		exitCh:                  make(chan struct{}),
		emu:                     terminal.New(terminal.Options{Cols: int(cols), Rows: int(rows), ScrollbackLines: sm.cfg.TerminalScrollbackLines}),
		ptyReadBuffer:           sm.cfg.PTYReadBuffer,
		clientChannelBuffer:     sm.cfg.ClientChannelBuffer,
		coalesceNotifyThreshold: sm.cfg.CoalesceNotifyThreshold,
		sigwinchCooldown:        time.Duration(sm.cfg.SIGWINCHCooldownMs) * time.Millisecond,
		reattachFunc:            sm.tmuxAttachFunc,
		sessionPrefix:           sm.tmuxSessionPrefix,
		metrics:                 sm.metrics,
	}

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	if sm.onSessionCreate != nil {
		sm.onSessionCreate(sess.ID)
	}

	// Persist metadata if store is configured
	if sm.store != nil {
		detached := bid == backend.Persistent
		_ = sm.store.Save(sessionstore.Metadata{
			ID:       sess.ID,
			Backend:  bid,
			Shell:    shell,
			Cols:     cols,
			Rows:     rows,
			Policy:   sessionPolicy,
			Created:  sess.CreatedAt,
			Detached: detached,
		})
	}

	// Start the PTY output reader; it will close exitCh when the process exits.
	go sess.readLoop()

	// Auto-remove: when the PTY exits, clean up the session map entry and
	// any upload temp directory so List()/Get() no longer return a terminated session.
	go func() {
		<-sess.Done()
		log.Printf("session %s: process exited (backend=%s)", sess.ID, bid)
		sm.mu.Lock()
		delete(sm.sessions, sess.ID)
		sm.mu.Unlock()
		if sm.onSessionDelete != nil {
			sm.onSessionDelete(sess.ID)
		}
		// Persistent sessions: ALWAYS preserve metadata so recovery can
		// re-attach on the next startup. The tmux session survives in its
		// own systemd scope even when the attach process dies. Deleting
		// metadata here would orphan the tmux session, causing recovery to
		// kill it — permanently destroying a recoverable session.
		//
		// Standard sessions: always delete metadata (they cannot survive).
		if sm.store != nil && bid != backend.Persistent {
			_ = sm.store.Delete(sess.ID)
		}
		// Clean up session upload directory
		uploadDir := filepath.Join(sm.uploadDirFunc(), sess.ID)
		if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
			log.Printf("session %s: failed to clean up upload dir: %v", sess.ID, err)
		}
	}()

	return sess, nil
}

// Get returns a session by ID.
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sess, ok := sm.sessions[id]
	return sess, ok
}

// List returns all active sessions.
// [REQ:P0-003a] Session Persistence Store
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	result := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		result = append(result, s)
	}
	return result
}

// Delete terminates a session and cleans up resources.
func (sm *SessionManager) Delete(id string) error {
	sm.mu.Lock()
	sess, ok := sm.sessions[id]
	if !ok {
		sm.mu.Unlock()
		return fmt.Errorf("session %s not found", id)
	}
	delete(sm.sessions, id)
	sm.mu.Unlock()

	_ = sess.pty.Kill()
	_ = sess.pty.Close()
	if sm.onSessionDelete != nil {
		sm.onSessionDelete(id)
	}
	// Clean up persisted metadata
	if sm.store != nil {
		_ = sm.store.Delete(id)
	}
	// Clean up session upload directory
	uploadDir := filepath.Join(sm.uploadDirFunc(), id)
	if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
		log.Printf("session %s: failed to clean up upload dir on delete: %v", id, err)
	}
	return nil
}
