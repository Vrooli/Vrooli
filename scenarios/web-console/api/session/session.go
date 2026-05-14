package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"web-console/internal/backend"
	"web-console/internal/metrics"
	"web-console/internal/policy"
	"web-console/internal/pty"
	"web-console/terminal"
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
	ID        string     `json:"id"`
	Shell     string     `json:"shell"`
	CreatedAt time.Time  `json:"created_at"`
	Cols      uint16     `json:"cols"`
	Rows      uint16     `json:"rows"`
	Backend   backend.ID `json:"backend"`

	pty    pty.PTY
	policy policy.Policy // [REQ:P1-001a] per-session expiration policy

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

	// sessionPrefix is the tmux session-name prefix used to map this
	// Session's ID to its tmux session. Set by Manager at construction.
	sessionPrefix string

	// metrics is optional; when set, readLoop increments re-attach counters.
	metrics *metrics.Metrics

	// Observer seam: non-client consumers (idle/prompt detectors, ANSI
	// responder, adapter dispatchers) tap the PTY output stream here.
	// Lazily initialized by RegisterObserver to keep construction sites
	// untouched. See session_observer.go.
	observersOnce sync.Once
	observers     *observerRegistry
}

// WriteInput delivers client-origin bytes to the PTY. Thread-safe —
// the PTY reference may be swapped during tmux re-attach. The kind
// parameter selects the delivery mechanism; see PTY.WriteInput and
// pty.InputKind for details.
func (s *Session) WriteInput(data []byte, kind pty.InputKind) error {
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

// CurrentDir returns the session's best-known working directory.
func (s *Session) CurrentDir(ctx context.Context) (string, error) {
	s.mu.Lock()
	p := s.pty
	s.mu.Unlock()
	return p.CurrentDir(ctx)
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
func (s *Session) GetPolicy() policy.Policy {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policy
}

// SetPolicy updates the session's expiration policy.
// [REQ:P1-001a] Expiration Policy Engine
func (s *Session) SetPolicy(p policy.Policy) {
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

// EffectiveSize returns the current PTY dimensions. Thread-safe.
func (s *Session) EffectiveSize() (uint16, uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Cols, s.Rows
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

// Recovered reports whether this session was restored from a surviving tmux
// session on server startup. The flag is consumed by handlers to inform
// clients that a snapshot may differ from the in-progress live state.
func (s *Session) Recovered() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recovered
}
