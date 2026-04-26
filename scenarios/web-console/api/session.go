package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
	"web-console/terminal"

	"github.com/google/uuid"
)

// RecoveryReport summarizes the result of session recovery on startup.
type RecoveryReport struct {
	Recovered        int
	OrphanedMetadata int
	OrphanedTmux     int
}

// tmux re-attach retry parameters. When the attach process dies but the tmux
// session itself survives, we retry with exponential backoff before declaring
// the session dead. A single transient failure (brief tmux unresponsiveness,
// resource pressure) should not permanently destroy a session.
const (
	tmuxReattachMaxRetries = 3
	tmuxReattachBaseDelay  = 500 * time.Millisecond
)

// Sentinel errors for session operations. Handlers use these to select the
// correct HTTP status code and user-facing message.
var (
	// ErrSessionLimitReached is returned when MaxSessions is configured and
	// the limit has been reached. Maps to HTTP 429.
	ErrSessionLimitReached = errors.New("session limit reached")

	// ErrPTYSpawnFailed wraps PTY creation failures. Maps to HTTP 500.
	ErrPTYSpawnFailed = errors.New("PTY spawn failed")
)

// SubscribeResult holds the channels and snapshot returned by Subscribe.
// DOC: docs/concepts/ARCHITECTURE.md#terminal-snapshot-replay
type SubscribeResult struct {
	// OutputCh receives PTY output frames after the snapshot has been
	// delivered. Closed when the PTY exits.
	OutputCh chan []byte
	// NotifyCh fires when coalesced frames exceed the configured threshold.
	NotifyCh chan int
	// Snapshot is the self-contained ANSI byte stream reproducing the
	// current emulator state (screen + alt-buffer flag + scrollback).
	// Caller must write it before draining OutputCh so live frames are
	// applied on top of the restored state.
	Snapshot []byte
}

// DOC: docs/concepts/ARCHITECTURE.md#data-flow
// DOC: docs/internal/SEAMS.md#3-domain--session-lifecycle
// Session represents a terminal session backed by a PTY process.
// [REQ:P0-002a] PTY Session Backend
type Session struct {
	ID        string    `json:"id"`
	Shell     string    `json:"shell"`
	CreatedAt time.Time `json:"created_at"`
	Cols      uint16    `json:"cols"`
	Rows      uint16    `json:"rows"`
	Backend   BackendID `json:"backend"`

	pty    PTY
	policy ExpirationPolicy // [REQ:P1-001a] per-session expiration policy

	// Output fan-out: the readLoop goroutine reads from the PTY, feeds the
	// decoded emulator, and broadcasts the live frame to connected
	// WebSocket clients. The emulator owns the durable replay state.
	mu              sync.Mutex
	clients         map[chan []byte]*ClientInfo
	emu             *terminal.Emulator
	processExited   bool // set by readLoop when the PTY read returns an error
	processExitCode int  // exit code from the PTY process (-1 if unknown)

	// utf8Buf holds an incomplete multi-byte UTF-8 sequence from the previous
	// PTY read. Prepended to the next read before broadcasting so that
	// string(data) + JSON encoding never sees partial codepoints.
	utf8Buf []byte

	// Config-driven limits for this session.
	ptyReadBuffer           int
	clientChannelBuffer     int
	coalesceNotifyThreshold int
	sigwinchCooldown        time.Duration

	// inAltBuffer mirrors emu.InAltBuffer() at the most recent broadcast,
	// so SIGWINCH-recovery decisions don't need to lock the emulator. It
	// is updated under s.mu by broadcast.
	inAltBuffer bool

	// lastSIGWINCHRecovery is the wall time of the most recent SIGWINCH
	// emitted by FlushPending's recovery path. Protected by s.mu.
	lastSIGWINCHRecovery time.Time

	// lastAltBufferTransition is the wall time of the most recent
	// alt-buffer enter or exit observed on the PTY output stream.
	// Used by maybeSIGWINCHRecovery to skip SIGWINCH during the
	// brief non-alt windows between alt-buffer cycles of a TUI.
	// Without this guard, Claude Code's footer gets redrawn into the
	// pane's normal buffer during those windows and tmux captures it
	// into scrollback, producing the "footer repeats when scrolling"
	// user-visible duplication. Protected by s.mu.
	lastAltBufferTransition time.Time

	// exitCh is closed when the PTY process exits, signaling the session owner.
	exitCh chan struct{}

	// recovered is true for sessions restored from tmux after a server restart.
	// Resets to false after the first WebSocket connection.
	recovered bool

	// closing is set by Shutdown() before closing the PTY fd. readLoop checks
	// this to skip re-attach retries during graceful shutdown, avoiding churn.
	closing bool

	// reattachFunc is the function readLoop uses to re-attach to a tmux
	// session. Defaults to tmuxAttach; injectable for testing.
	reattachFunc TmuxAttachFunc

	// metrics is optional; when set, readLoop increments re-attach counters.
	metrics *Metrics

	// Conversation side-channel: fan-out of semantic assistant events to
	// WebSocket clients subscribed to this terminal session.
	conversationMu         sync.Mutex
	conversationClients    map[chan ConversationEvent]*conversationSubscriber
	conversationDropLogged bool  // true once any drop has been logged this session (kept for tests / first-drop detection)
	conversationDropCount  int64 // total drops observed; included in log lines so missing sequences can be correlated
}

// conversationSubscriber tracks per-client state for the conversation fan-out.
// The resync channel is a 1-buffered signal that fires when SendConversation
// drops an event for this client because its event channel is full. The WS
// writer loop listens on this channel and emits a conversation_out_of_sync
// message so the client can refetch via GET /conversation?since_sequence=.
type conversationSubscriber struct {
	resync chan struct{}
}

// WriteInput delivers client-origin bytes to the PTY. Thread-safe —
// the PTY reference may be swapped during tmux re-attach. The kind
// parameter selects the delivery mechanism; see PTY.WriteInput and
// InputKind for details.
func (s *Session) WriteInput(data []byte, kind InputKind) error {
	s.mu.Lock()
	p := s.pty
	s.mu.Unlock()
	return p.WriteInput(data, kind)
}

// ProbeReady blocks until the PTY pipeline for this session is confirmed to
// be accepting writes. For persistent (tmux) sessions this waits for the
// attach-session handshake; for synchronous PTYs it returns immediately.
// Thread-safe against tmux re-attach (the PTY pointer is snapshotted).
func (s *Session) ProbeReady(ctx context.Context) error {
	s.mu.Lock()
	p := s.pty
	s.mu.Unlock()
	return p.ProbeReady(ctx)
}

// Subscribe registers a new client and returns a self-contained ANSI
// snapshot of the current emulator state plus channels for receiving live
// PTY output and coalescing notifications.
//
// The snapshot is generated under s.mu so no live frame can be broadcast
// between the snapshot and channel registration: the client's restored
// state plus the channel's live frames reproduce the same triple
// (screen, alt-buffer, scrollback) the server holds.
//
// Caller must call Unsubscribe when done.
// [REQ:P0-003b] Reconnect State Restoration
// DOC: docs/concepts/ARCHITECTURE.md#terminal-snapshot-replay
func (s *Session) Subscribe() SubscribeResult {
	notifyCh := make(chan int, 1)
	s.mu.Lock()
	snap := s.emu.Snapshot()
	ch := make(chan []byte, s.clientChannelBuffer)
	s.clients[ch] = &ClientInfo{NotifyCh: notifyCh}
	bctrace("subscribe", s.ID, fmt.Sprintf("snapshot_bytes=%d alt=%v", len(snap), s.inAltBuffer), nil)
	s.mu.Unlock()

	return SubscribeResult{
		OutputCh: ch,
		NotifyCh: notifyCh,
		Snapshot: snap,
	}
}

// Unsubscribe removes a client channel. The PTY size is unchanged.
func (s *Session) Unsubscribe(ch chan []byte) {
	s.mu.Lock()
	delete(s.clients, ch)
	s.mu.Unlock()
}

// Resize sets the PTY dimensions directly. Last caller wins.
// [REQ:P0-002c] Terminal Resize Handling
func (s *Session) Resize(cols, rows uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cols == s.Cols && rows == s.Rows {
		bctrace("resize_noop", s.ID, fmt.Sprintf("cols=%d rows=%d", cols, rows), nil)
		return
	}
	bctrace("resize", s.ID, fmt.Sprintf("cols=%d->%d rows=%d->%d alt=%v", s.Cols, cols, s.Rows, rows, s.inAltBuffer), nil)
	s.Cols = cols
	s.Rows = rows
	s.emu.Resize(int(cols), int(rows))
	_ = s.pty.SetSize(cols, rows)
}

// GetPolicy returns the session's expiration policy.
// [REQ:P1-001a] Expiration Policy Engine
func (s *Session) GetPolicy() ExpirationPolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policy
}

// SetPolicy updates the session's expiration policy.
// [REQ:P1-001a] Expiration Policy Engine
func (s *Session) SetPolicy(p ExpirationPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = p
}

// IsDead reports whether the underlying PTY process has exited.
// A dead session cannot accept new input; callers should open a new session.
func (s *Session) IsDead() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.processExited
}

// Done returns a channel that is closed when the PTY process exits.
// Callers can select on this to react to session termination without callbacks.
func (s *Session) Done() <-chan struct{} {
	return s.exitCh
}

// ExitCode returns the PTY process exit code. Only valid after Done() fires.
// Returns -1 if the exit code could not be determined.
func (s *Session) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.processExitCode
}

// readLoop continuously reads PTY output and broadcasts to subscribers.
// Each read is split at UTF-8 codepoint boundaries so that partial multi-byte
// sequences are buffered and prepended to the next read, preventing JSON
// encoding from replacing incomplete sequences with U+FFFD.
//
// On PTY read error (including normal process exit), it:
//  1. Flushes any buffered UTF-8 remainder
//  2. Marks the session as exited
//  3. Closes all client channels (triggering "exit" messages in WS handlers)
//  4. Signals exitCh so the SessionManager can clean up
//
// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
func (s *Session) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("session %s: readLoop panic (recovered): %v", s.ID, r)
			// Ensure exit signaling even after a panic so waiters don't hang.
			s.mu.Lock()
			if !s.processExited {
				s.processExited = true
				s.processExitCode = -1
				for ch := range s.clients {
					close(ch)
					delete(s.clients, ch)
				}
			}
			s.mu.Unlock()
			select {
			case <-s.exitCh:
			default:
				close(s.exitCh)
			}
		}
	}()
	buf := make([]byte, s.ptyReadBuffer)
	for {
		s.mu.Lock()
		p := s.pty
		s.mu.Unlock()
		n, err := p.Read(buf)
		if n > 0 {
			data := buf[:n]
			// Answer terminal capability queries server-side (DA1/DA3/DECRQM)
			// so TUI programs like Claude Code don't stall waiting for a
			// response xterm.js will not send. Write the reply straight to
			// the PTY master — it arrives at the foreground process as
			// stdin, just as a real terminal emulator's reply would. Only
			// the standard backend needs this: tmux answers queries for its
			// own panes.
			if s.Backend != BackendPersistent {
				if reply := generateAnsiResponses(data); len(reply) > 0 {
					// Server-origin reply; deliver as keystroke. For the
					// standard backend this is just a pipe write.
					if werr := p.WriteInput(reply, InputKindKeystroke); werr != nil {
						log.Printf("session %s: ansi-responder write failed: %v", s.ID, werr)
					}
				}
			}
			// Prepend any incomplete UTF-8 bytes from the previous read.
			if len(s.utf8Buf) > 0 {
				data = append(s.utf8Buf, data...)
				s.utf8Buf = nil
			}
			complete, remainder := splitCompleteUTF8(data)
			if len(complete) > 0 {
				s.broadcast(complete)
			}
			// Copy remainder — it may alias the read buffer which is reused.
			if len(remainder) > 0 {
				s.utf8Buf = append([]byte(nil), remainder...)
			}
		}
		if err != nil {
			// For persistent (tmux) sessions, the attach process can die while
			// the underlying tmux session survives. Retry re-attach with
			// exponential backoff before declaring the session dead. A single
			// transient failure should not permanently destroy a session.
			if s.Backend == BackendPersistent {
				s.mu.Lock()
				isClosing := s.closing
				s.mu.Unlock()
				if !isClosing {
					sessionName := tmuxSessionPrefix + s.ID
					reattached := false
					for attempt := 0; attempt < tmuxReattachMaxRetries; attempt++ {
						delay := tmuxReattachBaseDelay << attempt // 500ms, 1s, 2s
						time.Sleep(delay)
						// Re-check closing in case Shutdown() was called during backoff.
						s.mu.Lock()
						isClosing = s.closing
						s.mu.Unlock()
						if isClosing {
							log.Printf("session %s: shutdown detected during re-attach backoff, stopping retries", s.ID)
							break
						}
						if s.metrics != nil {
							s.metrics.ReattachAttempts.Add(1)
						}
						newPTY, reattachErr := s.reattachFunc(sessionName)
						if reattachErr == nil {
							log.Printf("session %s: tmux attach process died, re-attached successfully (attempt %d)", s.ID, attempt+1)
							if s.metrics != nil {
								s.metrics.ReattachSuccesses.Add(1)
							}
							s.mu.Lock()
							oldPTY := s.pty
							s.pty = newPTY
							s.mu.Unlock()
							// Close the old PTY fd to prevent file descriptor leaks.
							// The old attach process has already exited (that's why
							// we're here), but its PTY master fd is still open.
							_ = oldPTY.Close()
							_ = newPTY.SetSize(s.Cols, s.Rows)
							reattached = true
							break
						}
						log.Printf("session %s: tmux re-attach attempt %d/%d failed (read err: %v, attach err: %v)",
							s.ID, attempt+1, tmuxReattachMaxRetries, err, reattachErr)
					}
					if reattached {
						continue
					}
					if s.metrics != nil {
						s.metrics.ReattachFailures.Add(1)
					}
					log.Printf("session %s: all re-attach attempts exhausted, declaring session dead", s.ID)
				}
			}

			// Flush any remaining incomplete UTF-8 bytes — there is no more
			// data coming, so send them as-is (the terminal will handle them).
			if len(s.utf8Buf) > 0 {
				s.broadcast(s.utf8Buf)
				s.utf8Buf = nil
			}
			exitCode := s.pty.ExitCode()
			s.mu.Lock()
			s.processExited = true
			s.processExitCode = exitCode
			for ch := range s.clients {
				close(ch)
				delete(s.clients, ch)
			}
			s.mu.Unlock()
			close(s.exitCh)
			return
		}
	}
}

// splitCompleteUTF8 splits data at the last complete UTF-8 codepoint boundary.
// Returns (complete, remainder) where remainder contains any trailing incomplete
// multi-byte sequence that should be buffered for the next read.
//
// UTF-8 encoding rules used:
//   - 0xxxxxxx: 1-byte (ASCII)
//   - 110xxxxx: 2-byte leading byte
//   - 1110xxxx: 3-byte leading byte
//   - 11110xxx: 4-byte leading byte
//   - 10xxxxxx: continuation byte
//
// If the trailing bytes are orphaned continuation bytes with no leading byte,
// they are treated as complete (passed through as-is) since they represent
// pre-existing corruption, not a split boundary.
func splitCompleteUTF8(data []byte) (complete, remainder []byte) {
	if len(data) == 0 {
		return nil, nil
	}

	// Walk backward from the end to find the start of a potential
	// incomplete multi-byte sequence.
	i := len(data) - 1

	// Skip continuation bytes (10xxxxxx).
	contCount := 0
	for i >= 0 && data[i]&0xC0 == 0x80 {
		contCount++
		i--
	}

	// If we consumed the entire slice as continuation bytes, there's no
	// leading byte — treat as pre-existing corruption, pass through.
	if i < 0 {
		return data, nil
	}

	b := data[i]
	var expectedLen int
	switch {
	case b&0x80 == 0:
		// ASCII byte — not a multi-byte sequence leader.
		return data, nil
	case b&0xE0 == 0xC0:
		expectedLen = 2
	case b&0xF0 == 0xE0:
		expectedLen = 3
	case b&0xF8 == 0xF0:
		expectedLen = 4
	default:
		// Invalid leading byte — pass through.
		return data, nil
	}

	actualLen := contCount + 1 // leading byte + continuation bytes
	if actualLen >= expectedLen {
		// Sequence is complete.
		return data, nil
	}

	// Incomplete sequence: split before the leading byte.
	if i == 0 {
		return nil, data
	}
	return data[:i], data[i:]
}

// TmuxAttachFunc creates a PTY by attaching to an existing tmux session.
// Returns the PTY interface so tests can substitute fakes.
// Defaults to tmuxAttachAsPTY; overridden in tests.
type TmuxAttachFunc func(sessionName string) (PTY, error)

// TmuxDiscoverFunc discovers surviving tmux sessions by name prefix.
// Defaults to DiscoverTmuxSessions; overridden in tests.
type TmuxDiscoverFunc func() ([]string, error)

// SessionManager tracks all active terminal sessions.
// [REQ:P0-002a] PTY Session Backend
type SessionManager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	ptyFactory   PTYFactory
	cfgMu        sync.RWMutex // protects cfg from concurrent read/write (session-defaults handler vs Create)
	cfg          Config
	registry     *BackendRegistry
	store        SessionMetadataStore
	shuttingDown bool // set by Shutdown(); prevents auto-remove from deleting persistent session metadata

	// Seams for testability: injectable tmux operations.
	tmuxAttachFunc   TmuxAttachFunc
	tmuxDiscoverFunc TmuxDiscoverFunc

	// Observability: optional metrics and event logger for session lifecycle.
	metrics *Metrics
	events  *EventLogger

	// reattachStopCh signals the periodic re-attach watchdog to stop.
	reattachStopCh chan struct{}
}

// NewSessionManager creates a new session manager with the default PTY factory
// and configuration loaded from environment variables.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:         make(map[string]*Session),
		ptyFactory:       defaultPTYFactory,
		cfg:              LoadConfig(),
		tmuxAttachFunc:   tmuxAttachAsPTY,
		tmuxDiscoverFunc: DiscoverTmuxSessions,
	}
}

// NewSessionManagerWithFactory creates a session manager with a custom PTY factory.
// Use this in tests to substitute a fake PTY implementation.
func NewSessionManagerWithFactory(factory PTYFactory) *SessionManager {
	return &SessionManager{
		sessions:         make(map[string]*Session),
		ptyFactory:       factory,
		cfg:              DefaultConfig(),
		tmuxAttachFunc:   tmuxAttachAsPTY,
		tmuxDiscoverFunc: DiscoverTmuxSessions,
	}
}

// SetRegistry sets the backend registry for backend-aware session creation.
func (sm *SessionManager) SetRegistry(reg *BackendRegistry) {
	sm.registry = reg
}

// SetStore sets the session metadata store for persistence.
func (sm *SessionManager) SetStore(store SessionMetadataStore) {
	sm.store = store
}

// SetMetrics sets the metrics collector for session lifecycle counters.
func (sm *SessionManager) SetMetrics(m *Metrics) {
	sm.metrics = m
}

// SetEvents sets the event logger for structured session lifecycle events.
func (sm *SessionManager) SetEvents(el *EventLogger) {
	sm.events = el
}

// GetConfig returns a snapshot of the current configuration. Thread-safe.
func (sm *SessionManager) GetConfig() Config {
	sm.cfgMu.RLock()
	defer sm.cfgMu.RUnlock()
	return sm.cfg
}

// SetConfigField updates a mutable config field under the write lock.
// Only use for fields that can change at runtime (default backend/policy).
func (sm *SessionManager) SetConfigField(fn func(cfg *Config)) {
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

// ErrBackendUnavailable is returned when the requested backend is not available.
var ErrBackendUnavailable = errors.New("backend unavailable")

// ErrBackendUnknown is returned when the requested backend is not registered.
var ErrBackendUnknown = errors.New("unknown backend")

// Create starts a new shell session with a PTY.
// [REQ:P0-002a] PTY Session Backend
func (sm *SessionManager) Create(shell string, cols, rows uint16, backend BackendID, policy *ExpirationPolicy) (*Session, error) {
	shell, cols, rows = sm.applySessionDefaults(shell, cols, rows)

	// Resolve backend (read default under lock to avoid data race with settings handler)
	if backend == "" || backend == "auto" {
		sm.cfgMu.RLock()
		backend = BackendID(sm.cfg.DefaultBackend)
		sm.cfgMu.RUnlock()
	}

	// Look up factory from registry if available, otherwise use injected factory
	var factory PTYFactory
	if sm.registry != nil {
		// Resolve "auto" via registry when it wasn't resolved at startup (e.g. tests)
		if backend == "auto" {
			backend = sm.registry.ResolveAutoBackend()
		}
		desc, ok := sm.registry.Get(backend)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrBackendUnknown, backend)
		}
		if !desc.Available {
			return nil, fmt.Errorf("%w: %s — %s", ErrBackendUnavailable, backend, desc.Reason)
		}
		f, _ := sm.registry.Factory(backend)
		factory = f
	} else {
		// No registry (test path) — use injected factory, clear backend ID
		if backend == "auto" {
			backend = ""
		}
		factory = sm.ptyFactory
	}

	if sm.isSessionLimitReached() {
		return nil, fmt.Errorf("%w (%d)", ErrSessionLimitReached, sm.cfg.MaxSessions)
	}

	sessionID := uuid.New().String()
	spec := SessionLaunchSpec{
		SessionID: sessionID,
		Shell:     shell,
		Cols:      cols,
		Rows:      rows,
		Env: map[string]string{
			"WC_WEB_CONSOLE_SESSION_ID": sessionID,
			"CODEX_HOME":                sessionCodexHome(sessionID),
			"WC_CODEX_SESSIONS_DIR":     sessionCodexSessionsDir(sessionID),
		},
	}

	p, err := factory(spec)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPTYSpawnFailed, err)
	}

	// Resolve policy (read defaults under lock to avoid data race with settings handler)
	var sessionPolicy ExpirationPolicy
	if policy != nil {
		sessionPolicy = *policy
	} else {
		sm.cfgMu.RLock()
		mode := sm.cfg.DefaultPolicyMode
		dur := sm.cfg.DefaultPolicyDuration
		sm.cfgMu.RUnlock()
		if mode != "" {
			sessionPolicy = ExpirationPolicy{
				Mode:     PolicyMode(mode),
				Duration: dur,
			}
		} else {
			sessionPolicy = DefaultPolicy()
		}
	}

	sess := &Session{
		ID:                      sessionID,
		Shell:                   shell,
		CreatedAt:               time.Now(),
		Cols:                    cols,
		Rows:                    rows,
		Backend:                 backend,
		pty:                     p,
		policy:                  sessionPolicy,
		clients:                 make(map[chan []byte]*ClientInfo),
		exitCh:                  make(chan struct{}),
		emu:                     terminal.New(terminal.Options{Cols: int(cols), Rows: int(rows), ScrollbackLines: sm.cfg.TerminalScrollbackLines}),
		ptyReadBuffer:           sm.cfg.PTYReadBuffer,
		clientChannelBuffer:     sm.cfg.ClientChannelBuffer,
		coalesceNotifyThreshold: sm.cfg.CoalesceNotifyThreshold,
		sigwinchCooldown:        time.Duration(sm.cfg.SIGWINCHCooldownMs) * time.Millisecond,
		conversationClients:     make(map[chan ConversationEvent]*conversationSubscriber),
		reattachFunc:            sm.tmuxAttachFunc,
		metrics:                 sm.metrics,
	}

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	// Persist metadata if store is configured
	if sm.store != nil {
		detached := backend == BackendPersistent
		_ = sm.store.Save(SessionMetadata{
			ID:       sess.ID,
			Backend:  backend,
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
		log.Printf("session %s: process exited (backend=%s)", sess.ID, backend)
		sm.mu.Lock()
		delete(sm.sessions, sess.ID)
		sm.mu.Unlock()
		// Persistent sessions: ALWAYS preserve metadata so recovery can
		// re-attach on the next startup. The tmux session survives in its
		// own systemd scope even when the attach process dies. Deleting
		// metadata here would orphan the tmux session, causing recovery to
		// kill it — permanently destroying a recoverable session.
		//
		// Standard sessions: always delete metadata (they cannot survive).
		if sm.store != nil && backend != BackendPersistent {
			_ = sm.store.Delete(sess.ID)
		}
		// Clean up session upload directory
		uploadDir := filepath.Join(resolveUploadDir(), sess.ID)
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
	// Clean up persisted metadata
	if sm.store != nil {
		_ = sm.store.Delete(id)
	}
	// Clean up session upload directory
	uploadDir := filepath.Join(resolveUploadDir(), id)
	if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
		log.Printf("session %s: failed to clean up upload dir on delete: %v", id, err)
	}
	return nil
}

// Shutdown gracefully detaches from all persistent (tmux) sessions without
// killing them, so they survive for recovery on the next startup. Standard
// sessions are killed. Must be called before closing the database.
func (sm *SessionManager) Shutdown() {
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
		if sess.Backend == BackendPersistent {
			sess.mu.Lock()
			sess.closing = true
			sess.mu.Unlock()
		}
	}

	for _, sess := range snapshot {
		if sess.Backend == BackendPersistent {
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
func (sm *SessionManager) Recover(store SessionMetadataStore, registry *BackendRegistry) RecoveryReport {
	report := RecoveryReport{}

	// 1. Load persisted metadata for detached sessions
	metaList, err := store.ListDetached()
	if err != nil {
		log.Printf("recovery: failed to list detached sessions: %v", err)
		return report
	}
	metaMap := make(map[string]SessionMetadata, len(metaList))
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

	// 3. For each metadata row, try to recover or clean up
	for id, meta := range metaMap {
		if !tmuxSet[id] {
			// tmux session is gone — clean up stale metadata
			_ = store.Delete(id)
			report.OrphanedMetadata++
			log.Printf("recovery: cleaned up orphaned metadata for session %s", id)
			continue
		}

		// Re-apply tmux options (mouse mode, history limit) in case the options
		// were set by an older version that didn't configure them.
		sessionName := tmuxSessionPrefix + id
		applyTmuxOptions(sessionName)

		// Re-attach to surviving tmux session with retries. A transient
		// failure (tmux server briefly busy at startup) should not
		// permanently destroy the session.
		var p PTY
		var attachErr error
		for attempt := 0; attempt <= tmuxReattachMaxRetries; attempt++ {
			if attempt > 0 {
				delay := tmuxReattachBaseDelay << (attempt - 1)
				time.Sleep(delay)
			}
			p, attachErr = sm.tmuxAttachFunc(sessionName)
			if attachErr == nil {
				break
			}
			log.Printf("recovery: reattach session %s attempt %d/%d failed: %v",
				id, attempt+1, tmuxReattachMaxRetries+1, attachErr)
		}
		if attachErr != nil {
			// All retries exhausted. Preserve BOTH metadata and the tmux
			// session so the next server restart can try again. Previous
			// behavior deleted metadata here and then killed the tmux
			// session as an "orphan" — permanently destroying a
			// recoverable session on a transient failure.
			log.Printf("recovery: preserving session %s for future recovery (attach failed: %v)", id, attachErr)
			delete(tmuxSet, id) // prevent orphan-kill in step 4
			report.OrphanedMetadata++
			continue
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
			conversationClients:     make(map[chan ConversationEvent]*conversationSubscriber),
			recovered:               true,
			reattachFunc:            sm.tmuxAttachFunc,
			metrics:                 sm.metrics,
		}

		sm.mu.Lock()
		sm.sessions[id] = sess
		sm.mu.Unlock()

		go sess.readLoop()
		go func(sessID string, backend BackendID) {
			<-sess.Done()
			log.Printf("session %s: recovered process exited (backend=%s)", sessID, backend)
			sm.mu.Lock()
			delete(sm.sessions, sessID)
			sm.mu.Unlock()
			// Persistent sessions: preserve metadata for future recovery.
			// Standard sessions: delete metadata (they cannot survive).
			if sm.store != nil && backend != BackendPersistent {
				_ = sm.store.Delete(sessID)
			}
			uploadDir := filepath.Join(resolveUploadDir(), sessID)
			if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
				log.Printf("session %s: failed to clean up upload dir: %v", sessID, err)
			}
		}(id, meta.Backend)

		report.Recovered++
		log.Printf("recovery: recovered session %s (backend=%s)", id, meta.Backend)
		delete(tmuxSet, id)
	}

	// 4. Kill orphaned tmux sessions (no metadata)
	for id := range tmuxSet {
		sessionName := tmuxSessionPrefix + id
		_ = tmuxCmd("kill-session", "-t", sessionName).Run()
		report.OrphanedTmux++
		log.Printf("recovery: killed orphaned tmux session %s", id)
	}

	// 5. Record recovery metrics and emit events for observability.
	if sm.metrics != nil {
		sm.metrics.RecoveryRecovered.Add(int64(report.Recovered))
		sm.metrics.RecoveryOrphanedMeta.Add(int64(report.OrphanedMetadata))
		sm.metrics.RecoveryOrphanedTmux.Add(int64(report.OrphanedTmux))
	}
	if sm.events != nil {
		sm.events.Emit("session.recovery_complete", "", map[string]string{
			"recovered":        fmt.Sprintf("%d", report.Recovered),
			"orphaned_meta":    fmt.Sprintf("%d", report.OrphanedMetadata),
			"orphaned_tmux":    fmt.Sprintf("%d", report.OrphanedTmux),
			"metadata_entries": fmt.Sprintf("%d", len(metaList)),
			"tmux_sessions":    fmt.Sprintf("%d", len(tmuxSessions)),
		})
	}

	return report
}

// reattachWatchdogInterval controls how often the watchdog checks for
// persistent sessions that have metadata but are not in the active sessions map
// (i.e., the readLoop failed and auto-remove removed them, but the tmux session
// may still be alive). 30 seconds balances responsiveness with low overhead.
const reattachWatchdogInterval = 30 * time.Second

// StartReattachWatchdog launches a background goroutine that periodically
// checks for persistent sessions with metadata but no active in-memory session.
// When found, it attempts to re-attach to the tmux session and re-register it.
// This handles the case where a transient failure kills the attach process
// during normal operation — the session recovers without requiring a full
// server restart.
func (sm *SessionManager) StartReattachWatchdog() {
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
func (sm *SessionManager) StopReattachWatchdog() {
	sm.mu.Lock()
	if sm.reattachStopCh != nil {
		close(sm.reattachStopCh)
		sm.reattachStopCh = nil
	}
	sm.mu.Unlock()
}

func (sm *SessionManager) reattachWatchdogLoop() {
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
func (sm *SessionManager) reattachOrphanedSessions() {
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

		sessionName := tmuxSessionPrefix + meta.ID
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
			conversationClients:     make(map[chan ConversationEvent]*conversationSubscriber),
			recovered:               true,
			reattachFunc:            sm.tmuxAttachFunc,
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

		go sess.readLoop()
		go func(sessID string, backend BackendID) {
			<-sess.Done()
			log.Printf("session %s: re-attached process exited (backend=%s)", sessID, backend)
			sm.mu.Lock()
			delete(sm.sessions, sessID)
			sm.mu.Unlock()
			if sm.store != nil && backend != BackendPersistent {
				_ = sm.store.Delete(sessID)
			}
			uploadDir := filepath.Join(resolveUploadDir(), sessID)
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

// EffectiveSize returns the current PTY dimensions. Thread-safe.
func (s *Session) EffectiveSize() (uint16, uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Cols, s.Rows
}

// conversationChannelBuffer sizes each subscriber's conversation event buffer.
// Sized so a briefly throttled tab (background WS reads) can absorb a burst of
// agent output without triggering the out-of-sync resync path.
const conversationChannelBuffer = 256

// SubscribeConversation returns a buffered channel that receives conversation
// events for this session. Caller must call UnsubscribeConversation when done.
func (s *Session) SubscribeConversation() chan ConversationEvent {
	ch := make(chan ConversationEvent, conversationChannelBuffer)
	s.conversationMu.Lock()
	s.conversationClients[ch] = &conversationSubscriber{resync: make(chan struct{}, 1)}
	s.conversationMu.Unlock()
	return ch
}

// ConversationResyncSignal returns the out-of-sync signal channel bound to a
// given conversation subscription. Returns nil if ch is not (or no longer)
// subscribed. The WS writer loop selects on this channel and, on receive,
// emits conversation_out_of_sync so the client can refetch the gap via
// GET /conversation?since_sequence=N.
func (s *Session) ConversationResyncSignal(ch chan ConversationEvent) <-chan struct{} {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	if sub, ok := s.conversationClients[ch]; ok {
		return sub.resync
	}
	return nil
}

// UnsubscribeConversation removes and closes a conversation channel.
// close(ch) must happen inside the lock so that SendConversation (which
// iterates conversationClients under the same lock) can never write to a
// closed channel.
func (s *Session) UnsubscribeConversation(ch chan ConversationEvent) {
	s.conversationMu.Lock()
	delete(s.conversationClients, ch)
	close(ch)
	s.conversationMu.Unlock()
}

// SendConversation fans out a conversation event to all subscribed clients.
// Non-blocking: if a client's channel is full, the message is dropped and the
// subscriber's resync signal is pulsed so the WS writer emits an
// out-of-sync notice and the client refetches the gap.
func (s *Session) SendConversation(event ConversationEvent) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	for ch, sub := range s.conversationClients {
		select {
		case ch <- event:
		default:
			select {
			case sub.resync <- struct{}{}:
			default:
			}
			s.conversationDropCount++
			s.conversationDropLogged = true
			log.Printf("session %s: conversation event dropped (client channel full) — seq=%d id=%s total_drops=%d — resync signaled",
				s.ID, event.Sequence, event.ID, s.conversationDropCount)
		}
	}
}

// HasChildProcess reports whether the shell has any running child processes.
func (s *Session) HasChildProcess() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processExited {
		return false
	}
	return s.pty.HasChildProcess()
}
