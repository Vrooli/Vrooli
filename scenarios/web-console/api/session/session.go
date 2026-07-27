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
	// SizeCh receives authoritative grid changes for this connection.
	SizeCh chan [2]uint16
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
	leaseOwner      chan []byte
	leaseReason     LeaseReason
	nextClientOrder uint64
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

	// keyMap is the active key-name → bytes mapping for SendInput when
	// the variant is InputKeys. Nil means use DefaultKeyMap. Phase 4
	// will let BackendDescriptor override this per-backend.
	keyMap KeyMap

	// lastFrameAt is the wall time of the most recent non-empty PTY
	// output frame broadcast. Updated by broadcast() under s.mu;
	// consumed by WaitIdle. Zero before the first frame.
	lastFrameAt time.Time

	// idleWaiters fans WaitIdle's "fresh frame arrived" notifications
	// out to in-flight callers. Each WaitIdle owns one buffered
	// channel; markFrame does a non-blocking send to every waiter.
	idleWaiters []chan struct{}
}

// LastFrameAt returns the most recent non-empty terminal-output frame time.
func (s *Session) LastFrameAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastFrameAt
}

// SendInput delivers a typed SessionInput to the PTY through the single
// applyInput seam. Thread-safe — the PTY reference may be swapped during
// tmux re-attach.
//
// SessionInput's payload (text / keys / raw bytes) is resolved to bytes
// using the session's active KeyMap, then written through PTY.WriteInput
// with the kind selected by InputMeta.IsPaste (paste → pty.KindPaste,
// otherwise pty.KindKeystroke).
func (s *Session) SendInput(in SessionInput) error {
	data, err := in.resolveBytes(s.keyMap)
	if err != nil {
		return err
	}
	return s.applyInput(data, in.ptyKind())
}

// applyInput is the single PTY-write seam. All client-origin and
// server-origin input flows through here so a future observer or
// auditor only needs to wrap this one method.
func (s *Session) applyInput(data []byte, kind pty.InputKind) error {
	s.mu.Lock()
	p := s.pty
	s.mu.Unlock()
	return p.WriteInput(data, kind)
}

// Screen returns a structured deep-copy of the active screen: cell
// grid, cursor, dimensions, alt-buffer flag, and scrollback line count.
// Independent of the emulator; callers may retain or mutate it freely.
func (s *Session) Screen() terminal.ScreenView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emu.View()
}

// PlainText returns the active screen as UTF-8 text (trailing blanks
// stripped, rows joined with '\n'). When includeScrollback is true,
// scrollback lines (oldest first) are prepended.
func (s *Session) PlainText(includeScrollback bool) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emu.PlainText(includeScrollback)
}

// markFrame is called by broadcast (under s.mu) every time a non-empty
// frame is delivered. It updates the last-frame timestamp and wakes any
// goroutines waiting in WaitIdle so they can reset their quiet-window
// timer.
func (s *Session) markFrame() {
	s.lastFrameAt = time.Now()
	// Non-blocking nudge to any single waiter. Multiple concurrent
	// WaitIdle callers each have their own buffered channel; we use a
	// fan-out slice to wake them all.
	for _, w := range s.idleWaiters {
		select {
		case w <- struct{}{}:
		default:
		}
	}
}

func (s *Session) addIdleWaiter() chan struct{} {
	ch := make(chan struct{}, 1)
	s.idleWaiters = append(s.idleWaiters, ch)
	return ch
}

func (s *Session) removeIdleWaiter(ch chan struct{}) {
	for i, w := range s.idleWaiters {
		if w == ch {
			s.idleWaiters = append(s.idleWaiters[:i], s.idleWaiters[i+1:]...)
			return
		}
	}
}

// WaitIdle blocks until the PTY produces no output for quietWindow, or
// the timeout elapses, or the session exits, or ctx is done. The
// returned (reason, waited) pair describes how the wait ended.
//
// reason is one of:
//
//	"idle"    — quietWindow elapsed with no output.
//	"timeout" — overall deadline reached before reaching quietWindow.
//	"exited"  — the underlying PTY exited.
func (s *Session) WaitIdle(ctx context.Context, quietWindow, timeout time.Duration) (string, time.Duration, error) {
	if quietWindow <= 0 {
		quietWindow = 200 * time.Millisecond
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	start := time.Now()
	deadline := start.Add(timeout)

	s.mu.Lock()
	notify := s.addIdleWaiter()
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.removeIdleWaiter(notify)
		s.mu.Unlock()
	}()

	for {
		s.mu.Lock()
		exited := s.processExited
		ref := s.lastFrameAt
		s.mu.Unlock()

		if exited {
			return "exited", time.Since(start), nil
		}
		now := time.Now()
		if now.After(deadline) {
			return "timeout", time.Since(start), nil
		}
		if ref.IsZero() {
			ref = start
		}
		quietElapsed := now.Sub(ref)
		if quietElapsed >= quietWindow {
			return "idle", time.Since(start), nil
		}

		// Sleep until the soonest of: quietWindow expiry, overall
		// deadline, ctx cancellation, exit, or a fresh frame arrival.
		wakeIn := quietWindow - quietElapsed
		if d := time.Until(deadline); d < wakeIn {
			wakeIn = d
		}
		if wakeIn <= 0 {
			continue
		}
		timer := time.NewTimer(wakeIn)
		select {
		case <-timer.C:
		case <-notify:
			timer.Stop()
		case <-ctx.Done():
			timer.Stop()
			return "", time.Since(start), ctx.Err()
		case <-s.exitCh:
			timer.Stop()
			return "exited", time.Since(start), nil
		}
	}
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
	sizeCh := make(chan [2]uint16, 1)
	s.mu.Lock()
	snap := s.emu.Snapshot()
	ch := make(chan []byte, s.clientChannelBuffer)
	s.nextClientOrder++
	s.clients[ch] = &ClientInfo{NotifyCh: notifyCh, SizeCh: sizeCh, SubscribedOrder: s.nextClientOrder}
	if s.leaseOwner == nil {
		s.leaseOwner = ch
		s.leaseReason = LeaseReasonFirstClient
	}
	// The new connection gets its initial state directly from the WebSocket
	// forwarder; notify only pre-existing viewers that presence changed.
	for existing, info := range s.clients {
		if existing != ch {
			s.publishSizeLocked(info)
		}
	}
	bctrace("subscribe", s.ID, fmt.Sprintf("snapshot_bytes=%d alt=%v", len(snap), s.inAltBuffer), nil)
	s.mu.Unlock()

	return SubscribeResult{
		OutputCh: ch,
		NotifyCh: notifyCh,
		SizeCh:   sizeCh,
		Snapshot: snap,
	}
}

// Unsubscribe removes a client channel. The PTY size is unchanged.
func (s *Session) Unsubscribe(ch chan []byte) {
	s.mu.Lock()
	info, exists := s.clients[ch]
	if !exists {
		s.mu.Unlock()
		return
	}
	delete(s.clients, ch)
	close(info.SizeCh)
	if s.leaseOwner == ch {
		s.leaseOwner = s.oldestClientLocked()
		s.leaseReason = LeaseReasonLeaderDisconnect
		if s.leaseOwner != nil {
			s.applyDeclaredSizeLocked(s.leaseOwner)
			s.publishAllSizesLocked()
		}
	}
	s.publishAllSizesLocked()
	s.mu.Unlock()
}

// Resize applies a declared terminal size only when the caller holds the
// session's size lease. A session has one PTY winsize, so followers may
// declare their preferred dimensions but cannot silently reflow the leader.
// [REQ:P0-002c] Terminal Resize Handling
func (s *Session) Resize(owner chan []byte, cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if owner == nil || owner != s.leaseOwner {
		return ErrLeaseNotHeld
	}
	s.applyResizeLocked(cols, rows)
	return nil
}

func (s *Session) applyResizeLocked(cols, rows uint16) {
	if cols == s.Cols && rows == s.Rows {
		bctrace("resize_noop", s.ID, fmt.Sprintf("cols=%d rows=%d", cols, rows), nil)
		return
	}
	bctrace("resize", s.ID, fmt.Sprintf("cols=%d->%d rows=%d->%d alt=%v", s.Cols, cols, s.Rows, rows, s.inAltBuffer), nil)
	s.Cols = cols
	s.Rows = rows
	s.emu.Resize(int(cols), int(rows))
	_ = s.pty.SetSize(cols, rows)
	s.publishAllSizesLocked()
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
