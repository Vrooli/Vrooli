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

// ClientInfo tracks per-client broadcast flow control for a subscribed
// WebSocket connection. When the client's output channel is full, incoming
// frames are coalesced into a pending buffer instead of being dropped.
// The WebSocket output forwarder calls FlushPending after each successful
// write to drain coalesced data back into the channel.
type ClientInfo struct {
	pending          []byte   // coalesced data awaiting consumer drain
	resyncRequested  bool     // set when coalesced output is discarded
	resyncGeneration uint64   // identifies the request being served
	CoalescedFrames  int      // count of coalesced frames (observability)
	NotifyCh         chan int // receives cumulative coalesced count when threshold crossed
	// SizeCh carries the authoritative terminal grid. It is deliberately
	// separate from PTY output so a slow output consumer cannot make a viewer
	// retain a stale terminal size.
	SizeCh          chan [2]uint16
	DeclaredCols    uint16
	DeclaredRows    uint16
	DeviceID        string
	DeviceLabel     string
	SubscribedOrder uint64
}

// broadcast feeds PTY output into the durable emulator and fans out the
// frame to all connected WebSocket clients. Slow clients have frames
// coalesced into a pending buffer instead of being dropped.
func (s *Session) broadcast(data []byte) {
	if len(data) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Feed the durable emulator with RAW PTY bytes so its parser sees
	// every CSI query. The ANSI responder observes the emulator's
	// ControlEvent stream and answers only the server-owned query set;
	// if the emulator never saw the query, the reply never fires.
	_, _ = s.emu.Feed(data)
	s.snapshotCacheDirty = true
	bctrace("broadcast", s.ID, data, "clients=%d", len(s.clients))
	s.markFrame()
	if len(s.clients) == 0 {
		return
	}
	// The emulator and every client receive the same bytes. xterm.js owns
	// synchronized-output capability state and must see the framing intact.
	cp := make([]byte, len(data))
	copy(cp, data)
	for ch, info := range s.clients {
		s.deliver(ch, info, cp)
	}
}

// DOC: docs/internal/ERROR_SEMANTICS.md#sync-warning-data-loss-notification
// deliver sends data to a client channel, coalescing into the pending buffer
// when the channel is full. Must be called with s.mu held.
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
// count crosses the configured threshold. Must be called with s.mu held.
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
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if info, ok := s.clients[ch]; ok && info.resyncGeneration == generation {
		info.resyncRequested = false
	}
}
