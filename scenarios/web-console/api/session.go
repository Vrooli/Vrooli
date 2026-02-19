package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

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
)

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

	pty    PTY
	policy ExpirationPolicy // [REQ:P1-001a] per-session expiration policy

	// Output fan-out: the readLoop goroutine reads from the PTY and either
	// broadcasts to connected WebSocket clients or buffers for reconnect.
	mu               sync.Mutex
	clients          map[chan []byte]struct{}
	offlineBuf       []byte
	processExited    bool // set by readLoop when the PTY read returns an error
	processExitCode  int  // exit code from the PTY process (-1 if unknown)
	bufferOverflowed bool // set once when offline buffer cap is hit (log once)

	// Config-driven limits for this session
	offlineBufferMax    int
	ptyReadBuffer       int
	clientChannelBuffer int

	// exitCh is closed when the PTY process exits, signaling the session owner.
	exitCh chan struct{}
}

// Write sends data to the PTY stdin.
func (s *Session) Write(data []byte) (int, error) {
	return s.pty.Write(data)
}

// Subscribe returns a channel that receives PTY output. Caller must call
// Unsubscribe when done. Any buffered offline output is sent immediately.
// [REQ:P0-003b] Reconnect State Restoration
func (s *Session) Subscribe() chan []byte {
	ch := make(chan []byte, s.clientChannelBuffer)
	s.mu.Lock()
	// Send buffered offline output
	if len(s.offlineBuf) > 0 {
		ch <- s.offlineBuf
		s.offlineBuf = nil
	}
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

// Unsubscribe removes a client channel.
func (s *Session) Unsubscribe(ch chan []byte) {
	s.mu.Lock()
	delete(s.clients, ch)
	s.mu.Unlock()
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

// broadcast fans out PTY output to all connected WebSocket clients. When no
// clients are connected, output is buffered (up to offlineBufferMax) so it
// can be replayed on the next Subscribe call, enabling reconnect fidelity.
func (s *Session) broadcast(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.clients) == 0 {
		// Buffer offline output up to configured limit
		if len(s.offlineBuf)+len(data) <= s.offlineBufferMax {
			s.offlineBuf = append(s.offlineBuf, data...)
		} else if !s.bufferOverflowed {
			s.bufferOverflowed = true
			log.Printf("session %s: offline buffer full (%d bytes), dropping output", s.ID, s.offlineBufferMax)
		}
		return
	}
	// Copy to avoid data races since buf is reused
	cp := make([]byte, len(data))
	copy(cp, data)
	for ch := range s.clients {
		select {
		case ch <- cp:
		default:
			// Client is slow, drop frame
		}
	}
}

// readLoop continuously reads PTY output and broadcasts to subscribers.
// On PTY read error (including normal process exit), it:
//  1. Marks the session as exited
//  2. Closes all client channels (triggering "exit" messages in WS handlers)
//  3. Signals exitCh so the SessionManager can clean up
func (s *Session) readLoop() {
	buf := make([]byte, s.ptyReadBuffer)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			s.broadcast(buf[:n])
		}
		if err != nil {
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

// SessionManager tracks all active terminal sessions.
// [REQ:P0-002a] PTY Session Backend
type SessionManager struct {
	mu         sync.RWMutex
	sessions   map[string]*Session
	ptyFactory PTYFactory
	cfg        Config
}

// NewSessionManager creates a new session manager with the default PTY factory
// and configuration loaded from environment variables.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:   make(map[string]*Session),
		ptyFactory: defaultPTYFactory,
		cfg:        LoadConfig(),
	}
}

// NewSessionManagerWithFactory creates a session manager with a custom PTY factory.
// Use this in tests to substitute a fake PTY implementation.
func NewSessionManagerWithFactory(factory PTYFactory) *SessionManager {
	return &SessionManager{
		sessions:   make(map[string]*Session),
		ptyFactory: factory,
		cfg:        DefaultConfig(),
	}
}

// applySessionDefaults fills in zero-valued parameters with configured defaults.
// The convention is: zero/empty from the caller means "use server default".
func (sm *SessionManager) applySessionDefaults(shell string, cols, rows uint16) (string, uint16, uint16) {
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
func (sm *SessionManager) Create(shell string, cols, rows uint16) (*Session, error) {
	shell, cols, rows = sm.applySessionDefaults(shell, cols, rows)

	if sm.isSessionLimitReached() {
		return nil, fmt.Errorf("%w (%d)", ErrSessionLimitReached, sm.cfg.MaxSessions)
	}

	p, err := sm.ptyFactory(shell, cols, rows)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPTYSpawnFailed, err)
	}

	sess := &Session{
		ID:                  uuid.New().String(),
		Shell:               shell,
		CreatedAt:           time.Now(),
		Cols:                cols,
		Rows:                rows,
		pty:                 p,
		policy:              DefaultPolicy(),
		clients:             make(map[chan []byte]struct{}),
		exitCh:              make(chan struct{}),
		offlineBufferMax:    sm.cfg.OfflineBufferMax,
		ptyReadBuffer:       sm.cfg.PTYReadBuffer,
		clientChannelBuffer: sm.cfg.ClientChannelBuffer,
	}

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	// Start the PTY output reader; it will close exitCh when the process exits.
	go sess.readLoop()

	// Auto-remove: when the PTY exits, clean up the session map entry so
	// List()/Get() no longer return a terminated session.
	go func() {
		<-sess.Done()
		log.Printf("session %s: process exited", sess.ID)
		sm.mu.Lock()
		delete(sm.sessions, sess.ID)
		sm.mu.Unlock()
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
	return nil
}

// Resize changes the terminal dimensions of a session.
// [REQ:P0-002c] Terminal Resize Handling
func (sm *SessionManager) Resize(id string, cols, rows uint16) error {
	sm.mu.RLock()
	sess, ok := sm.sessions[id]
	sm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()
	sess.Cols = cols
	sess.Rows = rows
	return sess.pty.SetSize(cols, rows)
}
