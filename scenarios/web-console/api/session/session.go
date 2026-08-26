package session

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"time"

	"web-console/internal/backend"
	"web-console/internal/metrics"
	"web-console/internal/policy"
	"web-console/internal/pty"
	"web-console/terminal"
)

// EchoState is the server-owned authorization context for predictive input.
// Unknown is intentionally fail-closed: a client must never draw a secret
// when the backend cannot inspect terminal echo state.
type EchoState struct {
	Known           bool
	EchoEnabled     bool
	InAltBuffer     bool
	CursorAtLineEnd bool
}

var ErrEchoStateUnsupported = errors.New("terminal echo state is unsupported")

// ErrInputQueueFull is returned when a session is under sustained input load.
// Callers may retry after the ordered input lane drains.
var ErrInputQueueFull = errors.New("terminal input queue is full")

// ErrPTYClosed is shared by queued input callers and concrete backends when
// the session can no longer accept data.
var ErrPTYClosed = errors.New("pty is closed")

type echoStateProvider interface {
	TerminalEchoState() (EchoState, error)
}

type mouseModeController interface {
	SetMouseMode(bool) error
}

type mouseModeReader interface {
	MouseMode() (bool, error)
}

// SubscribeResult holds the channels and snapshot returned by Subscribe.
// DOC: docs/concepts/ARCHITECTURE.md#terminal-snapshot-replay
type SubscribeResult struct {
	// OutputCh receives PTY output frames after the snapshot has been
	// delivered. Closed when the PTY exits.
	OutputCh chan []byte
	// FrameCh is the cursor-bearing output stream used by protocol transports.
	// OutputCh is retained for package-local consumers that only need bytes.
	FrameCh chan OutputFrame
	// NotifyCh fires when coalesced frames exceed the configured threshold.
	NotifyCh chan int
	// SizeCh receives authoritative grid changes for this connection.
	SizeCh chan [2]uint16
	// PresenceCh receives lease and viewer changes independently of size.
	PresenceCh chan PresenceState
	// Snapshot is the self-contained ANSI byte stream reproducing the
	// current emulator state (screen + alt-buffer flag + scrollback).
	// Caller must write it before draining OutputCh so live frames are
	// applied on top of the restored state.
	Snapshot []byte
	// OutputCursor is the output-stream cursor at the snapshot boundary.
	OutputCursor int64
}

const outputReplayBytes = 8 << 20

// DOC: docs/concepts/ARCHITECTURE.md#data-flow
// DOC: docs/internal/SEAMS.md#3-domain-session-lifecycle
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
	// uploadRoot is captured when the session is registered. Cleanup must not
	// resolve it again after the PTY exits because test and routed environments
	// may legitimately change their process-level storage roots meanwhile.
	uploadRoot string

	// Output fan-out: the readLoop goroutine reads from the PTY, feeds the
	// decoded emulator, and broadcasts the live frame to connected
	// WebSocket clients. The emulator owns the durable replay state.
	emuMu sync.Mutex
	// clientsMu guards the client registry, per-client flow control, and the
	// size/presence lease. When both locks are needed, clientsMu is acquired
	// before emuMu. Neither lock is held across backend I/O.
	clientsMu sync.Mutex
	// ptyMu guards only ownership of the replaceable PTY pointer. It is never
	// held while calling a PTY method; re-attach can therefore swap the
	// pointer without blocking emulator or client state.
	ptyMu             sync.RWMutex
	inputQueue        chan queuedInput
	inputStopCh       chan struct{}
	inputStopOnce     sync.Once
	echoMu            sync.Mutex
	echoSampleMu      sync.Mutex
	backendEcho       EchoState
	backendEchoErr    error
	echoSampled       bool
	lastEchoSampleAt  time.Time
	clients           map[chan []byte]*ClientInfo
	outputCursor      int64
	outputFrames      []OutputFrame
	outputReplayBytes int
	// acceptedThrough is the cumulative UTF-8 byte offset of reliable stdin
	// accepted by this session. It survives WebSocket reconnects and is
	// intentionally independent of connection-local sequence numbers.
	acceptedThrough int64
	leaseOwner      chan []byte
	leaseReason     LeaseReason
	nextClientOrder uint64
	emu             *terminal.Emulator
	// snapshotCache is serialized only when emulator state changes. Subscribe
	// can then register a client and serve the last replay without walking the
	// entire scrollback while the PTY read loop holds s.emuMu.
	snapshotCache      []byte
	snapshotCacheDirty bool
	processExited      bool // set by readLoop when the PTY read returns an error
	processExitCode    int  // exit code from the PTY process (-1 if unknown)

	// utf8Buf holds an incomplete multi-byte UTF-8 sequence from the previous
	// PTY read. Prepended to the next read before broadcasting so that
	// string(data) + JSON encoding never sees partial codepoints.
	utf8Buf []byte

	// Config-driven limits for this session.
	ptyReadBuffer           int
	clientChannelBuffer     int
	coalesceNotifyThreshold int

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

	// keyMap is the active key-name → bytes mapping for SendInput when
	// the variant is InputKeys. The programmatic Connect adapter resolves
	// its default map before constructing the session input.
	keyMap KeyMap

	// lastFrameAt is the wall time of the most recent non-empty PTY
	// output frame broadcast. Updated by broadcast() under s.emuMu;
	// consumed by WaitIdle. Zero before the first frame.
	lastFrameAt time.Time

	// idleWaiters fans WaitIdle's "fresh frame arrived" notifications
	// out to in-flight callers. Each WaitIdle owns one buffered
	// channel; markFrame does a non-blocking send to every waiter.
	idleWaiters []chan struct{}
}

type queuedInput struct {
	data   []byte
	kind   pty.InputKind
	result chan error
}

// InputHeadFor returns the connection's accepted prefix plus payloads already
// submitted to the ordered input lane. It lets a WebSocket enqueue successive
// frames without waiting for backend latency between reads.
func (s *Session) InputHeadFor(client chan []byte) int64 {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	if info := s.clients[client]; info != nil {
		return info.acceptedThrough + info.pendingInputBytes
	}
	return 0
}

func (s *Session) ReserveInputFor(client chan []byte, bytes int64) bool {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	info := s.clients[client]
	if info == nil || bytes < 0 {
		return false
	}
	info.pendingInputBytes += bytes
	return true
}

func (s *Session) CompleteInputFor(client chan []byte, data []byte, writeErr error) int64 {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	info := s.clients[client]
	if info == nil {
		return 0
	}
	info.pendingInputBytes -= int64(len(data))
	if info.pendingInputBytes < 0 {
		info.pendingInputBytes = 0
	}
	if writeErr == nil {
		info.acceptedInput = append(info.acceptedInput, data...)
		info.acceptedThrough += int64(len(data))
		s.acceptedThrough += int64(len(data))
	}
	return info.acceptedThrough
}

const (
	echoSampleMinInterval = 250 * time.Millisecond
	echoSampleMaxInterval = 5 * time.Second
)

// EchoState returns the cached backend attributes plus emulator facts. It is
// deliberately read-only: backend probes may execute tmux/stty or block on a
// device and therefore never belong on a WebSocket output path.
func (s *Session) EchoState() (EchoState, error) {
	s.emuMu.Lock()
	inAltBuffer := s.emu.InAltBuffer()
	view := s.emu.View()
	cursorAtLineEnd := view.Cols > 0 && view.Cursor.X >= view.Cols-1
	s.emuMu.Unlock()

	s.echoMu.Lock()
	state, err := s.backendEcho, s.backendEchoErr
	s.echoMu.Unlock()
	state.InAltBuffer = inAltBuffer
	state.CursorAtLineEnd = cursorAtLineEnd
	return state, err
}

// RefreshEchoState samples the current PTY at a bounded cadence and updates
// the session cache. The sampling lock is separate from all session state, so
// concurrent WebSocket clients share one backend probe rather than multiplying
// tmux/stty work. force is reserved for explicit lifecycle/input events.
func (s *Session) RefreshEchoState(force bool) bool {
	s.echoSampleMu.Lock()
	defer s.echoSampleMu.Unlock()
	now := time.Now()
	if !force && !s.lastEchoSampleAt.IsZero() && now.Sub(s.lastEchoSampleAt) < echoSampleMinInterval {
		return false
	}
	s.lastEchoSampleAt = now

	s.ptyMu.RLock()
	provider, ok := s.pty.(echoStateProvider)
	s.ptyMu.RUnlock()
	if !ok {
		s.echoMu.Lock()
		changed := s.echoSampled || s.backendEchoErr != ErrEchoStateUnsupported
		s.backendEcho = EchoState{}
		s.backendEchoErr = ErrEchoStateUnsupported
		s.echoSampled = true
		s.echoMu.Unlock()
		return changed
	}

	state, err := provider.TerminalEchoState()
	s.echoMu.Lock()
	changed := !s.echoSampled || state != s.backendEcho || !sameError(s.backendEchoErr, err)
	s.backendEcho = state
	s.backendEchoErr = err
	s.echoSampled = true
	s.echoMu.Unlock()
	return changed
}

func sameError(a, b error) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Error() == b.Error())
}

func (s *Session) currentPTY() pty.PTY {
	s.ptyMu.RLock()
	defer s.ptyMu.RUnlock()
	return s.pty
}

func (s *Session) replacePTY(next pty.PTY) pty.PTY {
	s.ptyMu.Lock()
	defer s.ptyMu.Unlock()
	previous := s.pty
	s.pty = next
	return previous
}

// SetMouseMode changes mouse capture for the current backend when supported.
// Standard and bridged PTYs do not own a tmux session, so they return the
// shared unsupported sentinel rather than pretending the setting applied.
func (s *Session) SetMouseMode(enabled bool) error {
	controller, ok := s.currentPTY().(mouseModeController)
	if !ok {
		return pty.ErrUnsupported
	}
	return controller.SetMouseMode(enabled)
}

// MouseMode reports the current backend-owned mouse capture state. The
// boolean is meaningful only when the error is nil; unsupported backends
// return the shared sentinel so callers can hide the pane control.
func (s *Session) MouseMode() (bool, error) {
	reader, ok := s.currentPTY().(mouseModeReader)
	if !ok {
		return false, pty.ErrUnsupported
	}
	return reader.MouseMode()
}

// LastFrameAt returns the most recent non-empty terminal-output frame time.
func (s *Session) LastFrameAt() time.Time {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.lastFrameAt
}

// AcceptedThrough returns the cumulative reliable-stdin offset accepted by
// this session. The value is monotonic for the lifetime of the PTY session.
func (s *Session) AcceptedThrough() int64 {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.acceptedThrough
}

// OutputCursor returns the end of the retained PTY output stream.
func (s *Session) OutputCursor() int64 {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.outputCursor
}

// AcceptedThroughFor returns the session's accepted count relative to the
// connection that owns client. A newly subscribed client therefore starts at
// zero even when the PTY session has already received input.
func (s *Session) AcceptedThroughFor(client chan []byte) int64 {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	info := s.clients[client]
	if info == nil {
		return 0
	}
	return s.acceptedThrough - info.AcceptedBase
}

// ClientAcceptedThrough returns the reliable-input prefix belonging to one
// WebSocket connection. It is independent of input accepted from other
// viewers sharing the same PTY session.
func (s *Session) ClientAcceptedThrough(client chan []byte) int64 {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	if info := s.clients[client]; info != nil {
		return info.acceptedThrough
	}
	return 0
}

// HasAcceptedInput reports whether data exactly duplicates bytes this
// connection already delivered. A stale offset with different bytes is a
// gap, not a safe replay.
func (s *Session) HasAcceptedInput(client chan []byte, offset int64, data []byte) bool {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	info := s.clients[client]
	if info == nil || offset < 0 || offset+int64(len(data)) > int64(len(info.acceptedInput)) {
		return false
	}
	return bytes.Equal(info.acceptedInput[int(offset):int(offset)+len(data)], data)
}

// AdvanceAcceptedThroughFor records a successful write for one connection
// and advances the session-wide bookkeeping used to seed future bases.
func (s *Session) AdvanceAcceptedThroughFor(client chan []byte, data []byte) int64 {
	if len(data) == 0 {
		return s.ClientAcceptedThrough(client)
	}
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	info := s.clients[client]
	if info == nil {
		return 0
	}
	info.acceptedInput = append(info.acceptedInput, data...)
	info.acceptedThrough += int64(len(data))
	s.acceptedThrough += int64(len(data))
	return info.acceptedThrough
}

// AdvanceAcceptedThrough advances the reliable-stdin offset after a
// successful PTY write. Callers must pass the number of bytes written.
func (s *Session) AdvanceAcceptedThrough(bytes int64) int64 {
	if bytes <= 0 {
		return s.AcceptedThrough()
	}
	s.emuMu.Lock()
	s.acceptedThrough += bytes
	accepted := s.acceptedThrough
	s.emuMu.Unlock()
	return accepted
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
	_, err := s.SendInputCount(in)
	return err
}

// SendInputCount resolves and writes an input, returning the resolved payload
// length. The count is useful to structured transports that expose a byte
// count; ordinary callers should use SendInput.
func (s *Session) SendInputCount(in SessionInput) (int, error) {
	return s.sendInputCount(in, s.keyMap)
}

// SendInputCountWithKeyMap resolves a named-key input with an explicit map
// before writing it through the same PTY seam as every other input. It is
// used by structured adapters whose key vocabulary lives outside package
// session, while ordinary session callers use the configured map above.
func (s *Session) SendInputCountWithKeyMap(in SessionInput, km KeyMap) (int, error) {
	return s.sendInputCount(in, km)
}

func (s *Session) sendInputCount(in SessionInput, km KeyMap) (int, error) {
	data, err := in.resolveBytes(km)
	if err != nil {
		return 0, err
	}
	if err := s.applyInput(data, in.ptyKind()); err != nil {
		return 0, err
	}
	return len(data), nil
}

// applyInput is the single PTY-write seam. All client-origin and
// server-origin input flows through here so a future observer or
// auditor only needs to wrap this one method.
func (s *Session) applyInput(data []byte, kind pty.InputKind) error {
	result, err := s.EnqueueInput(data, kind)
	if err != nil {
		return err
	}
	select {
	case writeErr := <-result:
		return writeErr
	case <-s.inputStopCh:
		return ErrPTYClosed
	}
}

// EnqueueInput submits one payload to the session's ordered input lane and
// returns a completion channel. The send itself is non-blocking; callers that
// need synchronous semantics can wait on the returned channel.
func (s *Session) EnqueueInput(data []byte, kind pty.InputKind) (<-chan error, error) {
	result := make(chan error, 1)
	request := queuedInput{data: append([]byte(nil), data...), kind: kind, result: result}
	if s.inputQueue == nil {
		go func() { result <- s.writeInput(request.data, request.kind) }()
		return result, nil
	}
	select {
	case s.inputQueue <- request:
		return result, nil
	case <-s.inputStopCh:
		return nil, ErrPTYClosed
	default:
		return nil, ErrInputQueueFull
	}
}

func (s *Session) writeInput(data []byte, kind pty.InputKind) error {
	p := s.currentPTY()
	if p == nil {
		return ErrPTYClosed
	}
	err := p.WriteInput(data, kind)
	if err == nil {
		s.RefreshEchoState(false)
	}
	return err
}

func (s *Session) startInputWriter() {
	if s.inputQueue == nil || s.inputStopCh == nil {
		return
	}
	go func() {
		for {
			select {
			case request := <-s.inputQueue:
				request.result <- s.writeInput(request.data, request.kind)
			case <-s.inputStopCh:
				// Do not strand callers waiting for acknowledgements when a
				// session exits. Anything still buffered was never delivered.
				for {
					select {
					case request := <-s.inputQueue:
						request.result <- ErrPTYClosed
					default:
						return
					}
				}
			}
		}
	}()
}

func (s *Session) stopInputWriter() {
	if s.inputStopCh != nil {
		s.inputStopOnce.Do(func() { close(s.inputStopCh) })
	}
}

// Screen returns a structured deep-copy of the active screen: cell
// grid, cursor, dimensions, alt-buffer flag, and scrollback line count.
// Independent of the emulator; callers may retain or mutate it freely.
func (s *Session) Screen() terminal.ScreenView {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.emu.View()
}

// ScreenWithText returns both projections from one emulator moment. Callers
// that expose a structured screen and its plain-text companion must not take
// two independent locks around those reads.
func (s *Session) ScreenWithText(includeScrollback bool) (terminal.ScreenView, string) {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.emu.View(), s.emu.PlainText(includeScrollback)
}

// PlainText returns the active screen as UTF-8 text (trailing blanks
// stripped, rows joined with '\n'). When includeScrollback is true,
// scrollback lines (oldest first) are prepended.
func (s *Session) PlainText(includeScrollback bool) string {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.emu.PlainText(includeScrollback)
}

// markFrame is called by broadcast (under s.emuMu) every time a non-empty
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

	s.emuMu.Lock()
	notify := s.addIdleWaiter()
	s.emuMu.Unlock()
	defer func() {
		s.emuMu.Lock()
		s.removeIdleWaiter(notify)
		s.emuMu.Unlock()
	}()

	for {
		s.emuMu.Lock()
		exited := s.processExited
		ref := s.lastFrameAt
		s.emuMu.Unlock()

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
	p := s.currentPTY()
	return p.ProbeReady(ctx)
}

// CurrentDir returns the session's best-known working directory.
func (s *Session) CurrentDir(ctx context.Context) (string, error) {
	p := s.currentPTY()
	return p.CurrentDir(ctx)
}

// Subscribe registers a new client and returns a self-contained ANSI
// snapshot of the current emulator state plus channels for receiving live
// PTY output and coalescing notifications.
//
// The snapshot is generated under s.emuMu so no live frame can be broadcast
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
	presenceCh := make(chan PresenceState, 1)
	s.clientsMu.Lock()
	s.emuMu.Lock()
	if s.snapshotCacheDirty || s.snapshotCache == nil {
		s.snapshotCache = s.emu.Snapshot()
		s.snapshotCacheDirty = false
	}
	snap := s.snapshotCache
	outputCursor := s.outputCursor
	ch := make(chan []byte, s.clientChannelBuffer)
	frameCh := make(chan OutputFrame, s.clientChannelBuffer)
	s.nextClientOrder++
	s.clients[ch] = &ClientInfo{
		AcceptedBase:    s.acceptedThrough,
		NotifyCh:        notifyCh,
		SizeCh:          sizeCh,
		PresenceCh:      presenceCh,
		FrameCh:         frameCh,
		SubscribedOrder: s.nextClientOrder,
	}
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
	s.publishAllPresenceLocked()
	bctrace("subscribe", s.ID, nil, "snapshot_bytes=%d", len(snap))
	s.emuMu.Unlock()
	s.clientsMu.Unlock()

	return SubscribeResult{
		OutputCh:     ch,
		FrameCh:      frameCh,
		NotifyCh:     notifyCh,
		SizeCh:       sizeCh,
		PresenceCh:   presenceCh,
		Snapshot:     snap,
		OutputCursor: outputCursor,
	}
}

// Unsubscribe removes a client channel. The PTY size is unchanged.
func (s *Session) Unsubscribe(ch chan []byte) {
	s.clientsMu.Lock()
	s.emuMu.Lock()
	info, exists := s.clients[ch]
	if !exists {
		s.clientsMu.Unlock()
		s.emuMu.Unlock()
		return
	}
	delete(s.clients, ch)
	close(info.SizeCh)
	close(info.PresenceCh)
	if info.FrameCh != nil {
		close(info.FrameCh)
	}
	var resizePTY pty.PTY
	var resizeCols, resizeRows uint16
	if s.leaseOwner == ch {
		s.leaseOwner = s.oldestClientLocked()
		s.leaseReason = LeaseReasonLeaderDisconnect
		if s.leaseOwner != nil {
			resizePTY, resizeCols, resizeRows = s.applyDeclaredSizeLocked(s.leaseOwner)
			s.publishAllSizesLocked()
		}
	}
	s.publishAllSizesLocked()
	s.publishAllPresenceLocked()
	s.emuMu.Unlock()
	s.clientsMu.Unlock()
	if resizePTY != nil {
		_ = resizePTY.SetSize(resizeCols, resizeRows)
	}
}

// Resize applies a declared terminal size only when the caller holds the
// session's size lease. A session has one PTY winsize, so followers may
// declare their preferred dimensions but cannot silently reflow the leader.
// [REQ:P0-002c] Terminal Resize Handling
func (s *Session) Resize(owner chan []byte, cols, rows uint16) error {
	s.clientsMu.Lock()
	s.emuMu.Lock()
	if owner == nil || owner != s.leaseOwner {
		s.clientsMu.Unlock()
		s.emuMu.Unlock()
		return ErrLeaseNotHeld
	}
	p := s.applyResizeLocked(cols, rows)
	s.emuMu.Unlock()
	s.clientsMu.Unlock()
	if p == nil {
		return nil
	}
	return p.SetSize(cols, rows)
}

func (s *Session) applyResizeLocked(cols, rows uint16) pty.PTY {
	if cols == s.Cols && rows == s.Rows {
		bctrace("resize_noop", s.ID, nil, "cols=%d rows=%d", cols, rows)
		return nil
	}
	bctrace("resize", s.ID, nil, "cols=%d->%d rows=%d", s.Cols, cols, rows)
	s.Cols = cols
	s.Rows = rows
	s.emu.Resize(int(cols), int(rows))
	s.snapshotCacheDirty = true
	p := s.currentPTY()
	s.publishAllSizesLocked()
	return p
}

// GetPolicy returns the session's expiration policy.
// [REQ:P1-001a] Expiration Policy Engine
func (s *Session) GetPolicy() policy.Policy {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.policy
}

// SetPolicy updates the session's expiration policy.
// [REQ:P1-001a] Expiration Policy Engine
func (s *Session) SetPolicy(p policy.Policy) {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	s.policy = p
}

// IsDead reports whether the underlying PTY process has exited.
// A dead session cannot accept new input; callers should open a new session.
func (s *Session) IsDead() bool {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
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
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.processExitCode
}

// EffectiveSize returns the current PTY dimensions. Thread-safe.
func (s *Session) EffectiveSize() (uint16, uint16) {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.Cols, s.Rows
}

// Recovered reports whether this session was restored from a surviving tmux
// session on server startup. The flag is consumed by handlers to inform
// clients that a snapshot may differ from the in-progress live state.
func (s *Session) Recovered() bool {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return s.recovered
}
