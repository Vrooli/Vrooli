package session

// broadcast.go: Output fan-out, per-client delivery, and emulator resync when
// a slow client falls too far behind.
//
// This file owns:
//   - ClientInfo         (per-subscriber flow-control state)
//   - (*Session).broadcast, .deliver, .notifyIfThreshold
//   - (*Session).FlushPending
//   - (*Session).Resync, .CompleteResync
//
// PTY bytes flow through the per-session terminal.Emulator (the durable
// state) and then to subscribed clients as live frames. The emulator is
// the source of truth for replay; per-client coalesce buffering only
// absorbs short bursts when a client falls behind.

// pendingBufferMax is the maximum bytes of coalesced output retained per slow
// client. When exceeded, the unsafe fragment is discarded and the emulator
// snapshot is sent instead.
const pendingBufferMax = 1 << 20 // 1 MiB

// HistoryChunkSize is the maximum bytes per WebSocket frame when
// streaming the snapshot or draining the pending buffer. Smaller chunks
// prevent browser UI freezes on large initial replays.
const HistoryChunkSize = 64 * 1024

// OutputFrame is a replayable PTY frame. Cursors are UTF-8 byte offsets in
// the session output stream and always refer to frame boundaries.
type OutputFrame struct {
	Data        []byte
	StartCursor int64
	EndCursor   int64
}

// ClientInfo tracks per-client broadcast flow control for a subscribed
// WebSocket connection. When the client's output channel is full, incoming
// frames are coalesced into a pending buffer instead of being dropped.
// The WebSocket output forwarder calls FlushPending after each successful
// write to drain coalesced data back into the channel.
type ClientInfo struct {
	// AcceptedBase is the session-wide accepted count when this connection
	// subscribed. It is used only to expose the connection-relative wire
	// offset; each client also maintains its own accepted prefix below.
	AcceptedBase      int64
	acceptedThrough   int64
	acceptedInput     []byte
	pendingInputBytes int64
	PresenceCh        chan PresenceState
	FrameCh           chan OutputFrame
	pendingFrames     []OutputFrame
	pending           []byte   // coalesced data awaiting consumer drain
	resyncRequested   bool     // set when coalesced output is discarded
	resyncGeneration  uint64   // identifies the request being served
	CoalescedFrames   int      // count of coalesced frames (observability)
	NotifyCh          chan int // receives cumulative coalesced count when threshold crossed
	// SizeCh carries the authoritative terminal grid. It is deliberately
	// separate from PTY output so a slow output consumer cannot make a viewer
	// retain a stale terminal size.
	SizeCh       chan [2]uint16
	DeclaredCols uint16
	DeclaredRows uint16
	DeviceID     string
	DeviceLabel  string
	// DeviceClass is the connection's self-declared device family. It exists
	// so a follower can frame this client's session without inspecting the
	// terminal grid, which changes whenever a virtual keyboard opens. It is
	// display-only and never an authorization signal.
	DeviceClass string
	// KbOpen tracks whether this client's virtual keyboard currently covers
	// part of its viewport.
	KbOpen          bool
	SubscribedOrder uint64
}

// PresenceState changes independently of terminal dimensions and therefore
// travels on its own channel.
type PresenceState struct {
	Leader       string
	LeaderDevice string
	// LeaderClass and LeaderKbOpen let a follower present the leader's device
	// without deriving it from the shared grid.
	LeaderClass  string
	LeaderKbOpen bool
	HoldsLease   bool
	ViewerCount  int
}

// broadcast feeds PTY output into the durable emulator and fans out the
// frame to all connected WebSocket clients. Slow clients have frames
// coalesced into a pending buffer instead of being dropped.
func (s *Session) broadcast(data []byte) {
	if len(data) == 0 {
		return
	}
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	// Feed the durable emulator with RAW PTY bytes so its parser sees
	// every CSI query. The ANSI responder observes the emulator's
	// ControlEvent stream and answers only the server-owned query set;
	// if the emulator never saw the query, the reply never fires.
	_, _ = s.emu.Feed(data)
	s.snapshotCacheDirty = true
	bctrace("broadcast", s.ID, data, "clients=%d", len(s.clients))
	s.markFrame()
	s.outputCursor += int64(len(data))
	frame := OutputFrame{Data: cpBytes(data), StartCursor: s.outputCursor - int64(len(data)), EndCursor: s.outputCursor}
	s.outputFrames = append(s.outputFrames, frame)
	s.outputReplayBytes += len(data)
	for s.outputReplayBytes > outputReplayBytes && len(s.outputFrames) > 0 {
		s.outputReplayBytes -= len(s.outputFrames[0].Data)
		s.outputFrames = s.outputFrames[1:]
	}
	if len(s.clients) == 0 {
		return
	}
	// The emulator and every client receive the same bytes. xterm.js owns
	// synchronized-output capability state and must see the framing intact.
	cp := make([]byte, len(data))
	copy(cp, data)
	for ch, info := range s.clients {
		s.deliver(ch, info, cp)
		s.deliverFrame(info, frame)
	}
}

func cpBytes(data []byte) []byte {
	return append([]byte(nil), data...)
}

// deliverFrame mirrors deliver for the cursor-bearing transport used by the
// WebSocket bridge. The byte channel remains for package-local compatibility;
// new transports consume FrameCh so replay boundaries cannot be inferred from
// payload bytes.
func (s *Session) deliverFrame(info *ClientInfo, frame OutputFrame) {
	if info.FrameCh == nil || info.resyncRequested {
		return
	}
	if len(info.pendingFrames) > 0 {
		info.pendingFrames = append(info.pendingFrames, frame)
		if pendingFrameBytes(info.pendingFrames) > pendingBufferMax {
			info.pendingFrames = nil
			info.resyncRequested = true
			info.resyncGeneration++
		}
		return
	}
	select {
	case info.FrameCh <- frame:
	default:
		info.pendingFrames = append(info.pendingFrames, frame)
		info.CoalescedFrames++
		s.notifyIfThreshold(info)
	}
}

func pendingFrameBytes(frames []OutputFrame) int {
	total := 0
	for _, frame := range frames {
		total += len(frame.Data)
	}
	return total
}

// FlushPendingFrame drains cursor-bearing frames after a successful socket
// write. It returns true when the subscriber needs an emulator resync.
func (s *Session) FlushPendingFrame(ch chan OutputFrame) bool {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	var info *ClientInfo
	for _, candidate := range s.clients {
		if candidate.FrameCh == ch {
			info = candidate
			break
		}
	}
	if info == nil || len(info.pendingFrames) == 0 {
		return info != nil && info.resyncRequested
	}
	for len(info.pendingFrames) > 0 {
		select {
		case ch <- info.pendingFrames[0]:
			info.pendingFrames = info.pendingFrames[1:]
		default:
			return info.resyncRequested
		}
	}
	info.CoalescedFrames = 0
	return info.resyncRequested
}

// ReplayFrom returns the retained output suffix after cursor. A false result
// means the cursor predates the replay ring or is not on a frame boundary.
func (s *Session) ReplayFrom(cursor int64) ([]OutputFrame, int64, bool) {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	if cursor == s.outputCursor {
		return nil, s.outputCursor, s.outputCursor > 0
	}
	if cursor < 0 || len(s.outputFrames) == 0 {
		return nil, s.outputCursor, false
	}
	for i, frame := range s.outputFrames {
		if frame.StartCursor == cursor {
			frames := make([]OutputFrame, len(s.outputFrames)-i)
			for j := range frames {
				frames[j] = OutputFrame{Data: cpBytes(s.outputFrames[i+j].Data), StartCursor: s.outputFrames[i+j].StartCursor, EndCursor: s.outputFrames[i+j].EndCursor}
			}
			return frames, s.outputCursor, true
		}
	}
	return nil, s.outputCursor, false
}

// DOC: docs/internal/ERROR_SEMANTICS.md#sync-warning-data-loss-notification
// deliver sends data to a client channel, coalescing into the pending buffer
// when the channel is full. Must be called with s.emuMu held.
func (s *Session) deliver(ch chan []byte, info *ClientInfo, data []byte) {
	if info.resyncRequested {
		info.CoalescedFrames++
		s.notifyIfThreshold(info)
		return
	}
	if len(info.pending) > 0 {
		info.pending = append(info.pending, data...)
		info.CoalescedFrames++
		if len(info.pending) > pendingBufferMax {
			info.pending = nil
			info.resyncRequested = true
			info.resyncGeneration++
		}
		s.notifyIfThreshold(info)
		return
	}
	select {
	case ch <- data:
	default:
		info.pending = append([]byte(nil), data...)
		info.CoalescedFrames++
		s.notifyIfThreshold(info)
	}
}

// notifyIfThreshold sends a coalescing notification when the cumulative
// count crosses the configured threshold. Must be called with s.emuMu held.
func (s *Session) notifyIfThreshold(info *ClientInfo) {
	if s.coalesceNotifyThreshold > 0 && info.CoalescedFrames%s.coalesceNotifyThreshold == 0 {
		select {
		case info.NotifyCh <- info.CoalescedFrames:
		default:
		}
	}
}

// FlushPending drains any coalesced output for the given client channel.
// The WebSocket output forwarder calls this after each successful write
// to resume normal per-frame delivery. Data is chunked at HistoryChunkSize
// to prevent browser UI freezes from single large WebSocket messages.
// DOC: docs/internal/SEAMS.md#3-domain-session-lifecycle
func (s *Session) FlushPending(ch chan []byte) bool {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	info, ok := s.clients[ch]
	if !ok || len(info.pending) == 0 {
		return ok && info.resyncRequested
	}
	for len(info.pending) > 0 {
		end := HistoryChunkSize
		if end > len(info.pending) {
			end = len(info.pending)
		}
		chunk := make([]byte, end)
		copy(chunk, info.pending[:end])
		select {
		case ch <- chunk:
			info.pending = info.pending[end:]
		default:
			return info.resyncRequested
		}
	}
	info.pending = nil
	info.CoalescedFrames = 0
	return info.resyncRequested
}

// Resync returns the authoritative emulator snapshot requested for a slow
// client. The generation lets completion avoid clearing a newer request that
// arrived while this snapshot was being written.
func (s *Session) Resync(ch chan []byte) ([]byte, uint64, bool) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	info, ok := s.clients[ch]
	if !ok || !info.resyncRequested {
		return nil, 0, false
	}
	if s.snapshotCacheDirty || s.snapshotCache == nil {
		s.snapshotCache = s.emu.Snapshot()
		s.snapshotCacheDirty = false
	}
	return append([]byte(nil), s.snapshotCache...), info.resyncGeneration, true
}

// CompleteResync clears exactly the generation that was written. A newer
// overflow request remains pending for the next flush.
func (s *Session) CompleteResync(ch chan []byte, generation uint64) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	if info, ok := s.clients[ch]; ok && info.resyncGeneration == generation {
		info.resyncRequested = false
	}
}
