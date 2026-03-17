package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// sgrReset is an ANSI SGR reset sequence that clears all text attributes
// (color, bold, underline, etc.). Prepended to replayed history so that
// any dangling color state from a trimmed buffer doesn't bleed into the
// reconnecting client's terminal.
var sgrReset = []byte("\x1b[0m")

// historyChunkSize is the maximum bytes sent per channel message when
// replaying output history to a reconnecting client. Smaller chunks
// prevent browser UI freezes during large history replays.
// DOC: docs/concepts/ARCHITECTURE.md#history-replay-limitations
const historyChunkSize = 64 * 1024 // 64 KB

// Sentinel errors for session operations. Handlers use these to select the
// correct HTTP status code and user-facing message.
var (
	// ErrSessionLimitReached is returned when MaxSessions is configured and
	// the limit has been reached. Maps to HTTP 429.
	ErrSessionLimitReached = errors.New("session limit reached")

	// ErrPTYSpawnFailed wraps PTY creation failures. Maps to HTTP 500.
	ErrPTYSpawnFailed = errors.New("PTY spawn failed")
)

// ClientInfo tracks per-client broadcast flow control for a subscribed
// WebSocket connection. When the client's output channel is full, incoming
// frames are coalesced into a pending buffer instead of being dropped.
// The WebSocket output forwarder calls FlushPending after each successful
// write to drain coalesced data back into the channel.
type ClientInfo struct {
	pending         []byte   // coalesced data awaiting consumer drain
	pendingTrimmed  bool     // set when pending buffer was trimmed; triggers SIGWINCH after drain
	CoalescedFrames int      // count of coalesced frames (observability)
	NotifyCh        chan int // receives cumulative coalesced count when threshold crossed
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

	pty    PTY
	policy ExpirationPolicy // [REQ:P1-001a] per-session expiration policy

	// Output fan-out: the readLoop goroutine reads from the PTY and either
	// broadcasts to connected WebSocket clients while preserving bounded
	// output history for reconnect and reload replay.
	mu              sync.Mutex
	clients         map[chan []byte]*ClientInfo
	outputHistory   []byte
	processExited   bool // set by readLoop when the PTY read returns an error
	processExitCode int  // exit code from the PTY process (-1 if unknown)
	historyTrimmed  bool // set once when history cap is hit (log once)

	// utf8Buf holds an incomplete multi-byte UTF-8 sequence from the previous
	// PTY read. Prepended to the next read before broadcasting so that
	// string(data) + JSON encoding never sees partial codepoints.
	utf8Buf []byte

	// Config-driven limits for this session
	offlineBufferMax        int
	ptyReadBuffer           int
	clientChannelBuffer     int
	coalesceNotifyThreshold int

	// exitCh is closed when the PTY process exits, signaling the session owner.
	exitCh chan struct{}
}

// Write sends data to the PTY stdin.
func (s *Session) Write(data []byte) (int, error) {
	return s.pty.Write(data)
}

// Subscribe returns a channel that receives PTY output, a notification
// channel that fires when coalesced frames exceed the configured threshold,
// and a bool indicating whether buffered history was replayed.
//
// When hadHistory is true, the caller should expect a nil sentinel value
// on the output channel after all history chunks have been delivered. This
// sentinel tells the WebSocket forwarder to send a "history_end" message
// so the client can batch-render history in one pass.
//
// Caller must call Unsubscribe when done. Recent output history is replayed
// immediately, prefixed with an SGR reset to clear any dangling
// color/attribute state that may have been lost when the history buffer
// was trimmed.
// [REQ:P0-003b] Reconnect State Restoration
func (s *Session) Subscribe() (chan []byte, chan int, bool) {
	notifyCh := make(chan int, 1)
	s.mu.Lock()

	// Build history chunks (if any) before allocating the channel so we
	// can size it to hold all chunks plus the nil sentinel without blocking.
	var chunks [][]byte
	if len(s.outputHistory) > 0 {
		snapshot := make([]byte, 0, len(sgrReset)+len(s.outputHistory))
		snapshot = append(snapshot, sgrReset...)
		snapshot = append(snapshot, s.outputHistory...)
		for off := 0; off < len(snapshot); off += historyChunkSize {
			end := off + historyChunkSize
			if end > len(snapshot) {
				end = len(snapshot)
			}
			chunk := make([]byte, end-off)
			copy(chunk, snapshot[off:end])
			chunks = append(chunks, chunk)
		}
	}

	hadHistory := len(chunks) > 0

	// Ensure the channel can hold all history chunks + the nil sentinel
	// without blocking, while still respecting the configured buffer size
	// for live output delivery.
	bufSize := s.clientChannelBuffer
	if needed := len(chunks) + 1; needed > bufSize {
		bufSize = needed
	}
	ch := make(chan []byte, bufSize)

	for _, chunk := range chunks {
		ch <- chunk
	}
	if hadHistory {
		ch <- nil // sentinel: history replay complete
	}

	s.clients[ch] = &ClientInfo{NotifyCh: notifyCh}
	s.mu.Unlock()
	return ch, notifyCh, hadHistory
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
		return
	}
	s.Cols = cols
	s.Rows = rows
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

func (s *Session) appendHistory(data []byte) {
	if s.offlineBufferMax <= 0 || len(data) == 0 {
		return
	}

	if len(data) >= s.offlineBufferMax {
		trimmed := data[len(data)-s.offlineBufferMax:]
		s.outputHistory = append([]byte(nil), snapToCleanBoundary(trimmed)...)
		if !s.historyTrimmed {
			s.historyTrimmed = true
			log.Printf("session %s: output history trimmed to %d bytes", s.ID, s.offlineBufferMax)
		}
		return
	}

	combinedLen := len(s.outputHistory) + len(data)
	if combinedLen <= s.offlineBufferMax {
		s.outputHistory = append(s.outputHistory, data...)
		return
	}

	trim := combinedLen - s.offlineBufferMax
	remainder := append(append([]byte(nil), s.outputHistory[trim:]...), data...)
	s.outputHistory = snapToCleanBoundary(remainder)
	if !s.historyTrimmed {
		s.historyTrimmed = true
		log.Printf("session %s: output history trimmed to %d bytes", s.ID, s.offlineBufferMax)
	}
}

// snapToCleanBoundary advances past any partial ANSI escape sequence at the
// start of buf and, when possible, snaps forward to the first newline so
// replayed history starts on a line boundary. This prevents reconnecting
// clients from seeing garbage bytes from a mid-sequence trim.
func snapToCleanBoundary(buf []byte) []byte {
	if len(buf) == 0 {
		return buf
	}

	// If the first byte is ESC, the sequence is intact (starts fresh).
	// If it's NOT ESC but looks like mid-CSI-sequence parameter/intermediate/
	// final bytes, we're inside a truncated sequence.
	start := 0
	if buf[0] != 0x1b && looksLikeMidSequence(buf) {
		// Skip the CSI introducer '[' if present (it follows the truncated ESC).
		if buf[0] == '[' {
			start = 1
		}
		// Scan past parameter bytes (0x30-0x3F) and intermediate bytes (0x20-0x2F)
		// until we hit the final byte (0x40-0x7E) that terminates the sequence.
		for start < len(buf) {
			b := buf[start]
			start++
			if b >= 0x40 && b <= 0x7E {
				break
			}
		}
	}

	// Try to advance to the first newline for a clean line boundary,
	// but only if the newline is within the first 256 bytes to avoid
	// discarding too much history.
	const maxNewlineScan = 256
	scanLimit := start + maxNewlineScan
	if scanLimit > len(buf) {
		scanLimit = len(buf)
	}
	if nlIdx := bytes.IndexByte(buf[start:scanLimit], '\n'); nlIdx >= 0 {
		start += nlIdx + 1
	}

	if start >= len(buf) {
		return nil
	}
	return buf[start:]
}

// looksLikeMidSequence heuristically detects whether buf starts inside a
// truncated ANSI CSI escape sequence. It requires evidence of a real CSI
// sequence (a final byte 0x40-0x7E within a short window) to avoid false
// positives on normal text starting with digits, spaces, or punctuation.
func looksLikeMidSequence(buf []byte) bool {
	if len(buf) == 0 {
		return false
	}
	// '[' following a trimmed ESC is the clear CSI indicator.
	if buf[0] == '[' {
		return true
	}
	// Parameter bytes (0x30-0x3F: digits, semicolons) are only mid-sequence
	// if followed by a CSI final byte (0x40-0x7E) within a short window.
	// This prevents false positives on lines starting with numbers or punctuation.
	if buf[0] >= 0x30 && buf[0] <= 0x3F {
		limit := 8
		if limit > len(buf) {
			limit = len(buf)
		}
		for i := 0; i < limit; i++ {
			b := buf[i]
			if b >= 0x40 && b <= 0x7E {
				return true // Found CSI final byte — this is mid-sequence
			}
			if b < 0x20 || b > 0x3F {
				return false // Non-parameter byte before final — not mid-sequence
			}
		}
		return false // No final byte found in window — not mid-sequence
	}
	// Space (0x20) and intermediate bytes (0x20-0x2F) alone are NOT
	// treated as mid-sequence — they are too common as regular text.
	return false
}

// broadcast fans out PTY output to all connected WebSocket clients while
// preserving bounded output history for reconnect/reload replay.
// Slow clients that can't keep up have frames coalesced into a pending buffer
// instead of being dropped. The WebSocket output forwarder calls FlushPending
// after each successful write to drain coalesced data back into the channel.
func (s *Session) broadcast(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appendHistory(data)
	if len(s.clients) == 0 {
		return
	}
	// Copy to avoid data races since buf is reused by readLoop.
	cp := make([]byte, len(data))
	copy(cp, data)
	for ch, info := range s.clients {
		s.deliver(ch, info, cp)
	}
}

// DOC: docs/internal/ERROR-SEMANTICS.md#sync-warning-coalescing-notification
// deliver sends data to a client channel, coalescing into the pending buffer
// when the channel is full. Must be called with s.mu held.
func (s *Session) deliver(ch chan []byte, info *ClientInfo, data []byte) {
	if len(info.pending) > 0 {
		// Already coalescing — append to pending buffer.
		info.pending = append(info.pending, data...)
		info.CoalescedFrames++
		// Cap pending at offlineBufferMax to prevent unbounded growth.
		// Snap to a clean ANSI boundary so partial escape sequences don't
		// corrupt the terminal when the coalesced data is flushed.
		// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
		if s.offlineBufferMax > 0 && len(info.pending) > s.offlineBufferMax {
			trimmed := info.pending[len(info.pending)-s.offlineBufferMax:]
			trimmedClean := snapToCleanBoundary(trimmed)
			// Prepend SGR reset so trimmed color-setting sequences don't
			// bleed stale attributes into the client's terminal.
			info.pending = make([]byte, 0, len(sgrReset)+len(trimmedClean))
			info.pending = append(info.pending, sgrReset...)
			info.pending = append(info.pending, trimmedClean...)
			info.pendingTrimmed = true
		}
		s.notifyIfThreshold(info)
		return
	}
	select {
	case ch <- data:
		// Sent immediately.
	default:
		// Channel full — start coalescing instead of dropping.
		info.pending = append([]byte(nil), data...)
		info.CoalescedFrames++
		s.notifyIfThreshold(info)
	}
}

// notifyIfThreshold sends a coalescing notification when the cumulative
// count crosses the configured threshold. Must be called with s.mu held.
func (s *Session) notifyIfThreshold(info *ClientInfo) {
	if s.coalesceNotifyThreshold > 0 && info.CoalescedFrames%s.coalesceNotifyThreshold == 0 {
		select {
		case info.NotifyCh <- info.CoalescedFrames:
		default:
			// Notification already pending.
		}
	}
}

// FlushPending drains any coalesced output for the given client channel.
// The WebSocket output forwarder calls this after each successful write
// to resume normal per-frame delivery. Data is chunked at historyChunkSize
// to prevent browser UI freezes from single large WebSocket messages.
// DOC: docs/internal/SEAMS.md#3-domain--session-lifecycle
func (s *Session) FlushPending(ch chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.clients[ch]
	if !ok || len(info.pending) == 0 {
		return
	}
	for len(info.pending) > 0 {
		end := historyChunkSize
		if end > len(info.pending) {
			end = len(info.pending)
		}
		chunk := make([]byte, end)
		copy(chunk, info.pending[:end])
		select {
		case ch <- chunk:
			info.pending = info.pending[end:]
		default:
			return // Channel full mid-flush — keep remainder for next cycle
		}
	}
	info.pending = nil
	info.CoalescedFrames = 0
	if info.pendingTrimmed {
		info.pendingTrimmed = false
		// Trigger SIGWINCH so the shell redraws its screen, recovering
		// structural state (cursor position, scroll region, alternate
		// screen buffer) that was lost when the coalesced buffer was trimmed.
		// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
		_ = s.pty.SetSize(s.Cols, s.Rows)
	}
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
	buf := make([]byte, s.ptyReadBuffer)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			data := buf[:n]
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
		ID:                      uuid.New().String(),
		Shell:                   shell,
		CreatedAt:               time.Now(),
		Cols:                    cols,
		Rows:                    rows,
		pty:                     p,
		policy:                  DefaultPolicy(),
		clients:                 make(map[chan []byte]*ClientInfo),
		exitCh:                  make(chan struct{}),
		offlineBufferMax:        sm.cfg.OfflineBufferMax,
		ptyReadBuffer:           sm.cfg.PTYReadBuffer,
		clientChannelBuffer:     sm.cfg.ClientChannelBuffer,
		coalesceNotifyThreshold: sm.cfg.CoalesceNotifyThreshold,
	}

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	// Start the PTY output reader; it will close exitCh when the process exits.
	go sess.readLoop()

	// Auto-remove: when the PTY exits, clean up the session map entry and
	// any upload temp directory so List()/Get() no longer return a terminated session.
	go func() {
		<-sess.Done()
		log.Printf("session %s: process exited", sess.ID)
		sm.mu.Lock()
		delete(sm.sessions, sess.ID)
		sm.mu.Unlock()
		// Clean up session upload directory
		uploadDir := filepath.Join(uploadBaseDir, sess.ID)
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
	// Clean up session upload directory
	uploadDir := filepath.Join(uploadBaseDir, id)
	if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
		log.Printf("session %s: failed to clean up upload dir on delete: %v", id, err)
	}
	return nil
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
