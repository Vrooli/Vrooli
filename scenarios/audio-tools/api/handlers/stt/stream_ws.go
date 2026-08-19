package stt

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/protoint"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter"
	"audio-tools/internal/stt/session"
)

// Voice streaming WS message types. These match the wire shape the
// browser audio-integration consumes; transport adapters must not
// change them without coordinating with that copy-paste reference.
const (
	wsMsgPartial         = "partial"
	wsMsgFinal           = "final"
	wsMsgSegmentFinal    = "segment-final"
	wsMsgSegmentRejected = "segment-rejected"
	wsMsgError           = "error"
	wsMsgDone            = "done"
	wsMsgStatus          = "status"
	// wsMsgVadState mirrors sttchain.StreamEventVadState on the browser
	// transport. See VadStateEvent doc + plan §7 step 3.
	wsMsgVadState = "vad-state"
)

const wsV2AudioHeaderBytes = 60

// A browser PCM wire batch is normally 100 ms (3.2 KiB) and the qualification
// path uses at most one second (32 KiB). Keep a generous ceiling while making
// ReadMessage unable to materialize an unbounded or accidental whole-turn
// audio frame in the API heap.
const maxWSMessageBytes = 1 << 20

// decodeWSV2AudioFrame reads the versioned browser framing: ASCII ATV2,
// uint64 chunk sequence, int64 absolute start/end samples (big endian), a
// SHA-256 digest, then canonical audio bytes. It has no transport state so both the WebSocket
// adapter and its contract tests use exactly the same mapping.
func decodeWSV2AudioFrame(frame []byte) (sttchain.AudioChunk, error) {
	if len(frame) < wsV2AudioHeaderBytes {
		return sttchain.AudioChunk{}, fmt.Errorf("stt websocket v2: frame shorter than header")
	}
	if string(frame[:4]) != "ATV2" {
		return sttchain.AudioChunk{}, fmt.Errorf("stt websocket v2: invalid frame magic")
	}
	chunk := sttchain.AudioChunk{
		Sequence:    binary.BigEndian.Uint64(frame[4:12]),
		StartSample: protoint.FromUint64(binary.BigEndian.Uint64(frame[12:20])),
		EndSample:   protoint.FromUint64(binary.BigEndian.Uint64(frame[20:28])),
		Digest:      append([]byte(nil), frame[28:60]...),
		Audio:       append([]byte(nil), frame[wsV2AudioHeaderBytes:]...),
	}
	if len(chunk.Audio) == 0 || chunk.EndSample < chunk.StartSample {
		return sttchain.AudioChunk{}, fmt.Errorf("stt websocket v2: invalid audio range")
	}
	actual := sha256.Sum256(chunk.Audio)
	if !bytes.Equal(actual[:], chunk.Digest) {
		return sttchain.AudioChunk{}, fmt.Errorf("stt websocket v2: audio digest mismatch")
	}
	return chunk, nil
}

// wsWriterDrainTimeout bounds how long teardown waits for the coalescing
// writer to flush queued durable messages to a slow consumer before the socket
// is force-closed. Committed text is flushed within this window; a dead
// consumer cannot hang teardown past it.
const wsWriterDrainTimeout = 5 * time.Second

// wsCoalescingWriter owns every write to the browser socket. It applies the
// event-durability contract to the browser-facing egress so a slow or stalled
// consumer can never back-pressure the upstream relay — and therefore can never
// wedge the kyutai socket reader. Durable messages (status acknowledgements,
// segment-final, rejection, error, final, and done) are queued losslessly and
// in order. Partial, VAD, and ordinary status snapshots each have their own
// latest-value slot: they may be coalesced under backpressure, but a VAD tick
// can never erase the latest live text (or vice versa). The events loop enqueues
// without ever blocking on the socket; only this one goroutine blocks on a slow
// consumer.
// See docs/domains/stt/streaming-pipeline.md#event-durability-contract.
type wsCoalescingWriter struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	durable []wsMessage
	partial *wsMessage
	vad     *wsMessage
	status  *wsMessage
	closed  bool
	// finished records that run() has returned, so no goroutine is draining the
	// queue any more. Producers other than the events loop (notably the reader
	// goroutine's deterministic fault paths) can still enqueue after that, and
	// a durable queued into a dead writer used to disappear with no signal --
	// including the incomplete_coverage error that tells the operator their
	// audio was not fully accounted for.
	finished bool
	signal   chan struct{}
	done     chan struct{}
}

func newWSCoalescingWriter(conn *websocket.Conn) *wsCoalescingWriter {
	w := &wsCoalescingWriter{conn: conn, signal: make(chan struct{}, 1), done: make(chan struct{})}
	go w.run()
	return w
}

func (w *wsCoalescingWriter) wake() {
	select {
	case w.signal <- struct{}{}:
	default:
	}
}

// enqueue classifies a message per the durability contract and never blocks on
// the socket. Each progress stream coalesces independently to its latest value;
// durable messages go to the ordered queue. A committed/terminal durable
// (segment-final / final) supersedes every unsent progress snapshot so stale
// text or VAD state is never re-shown after a commit.
func (w *wsCoalescingWriter) enqueue(m wsMessage) {
	w.mu.Lock()
	if wsMessageDeliveryClass(m) == sttchain.DeliveryProgress {
		mm := m
		switch m.Type {
		case wsMsgPartial:
			w.partial = &mm
		case wsMsgVadState:
			w.vad = &mm
		case wsMsgStatus:
			w.status = &mm
		default:
			// Keep the classifier and the slot mapping fail-safe if a new
			// progress message is introduced without a dedicated slot.
			w.status = &mm
		}
	} else {
		if m.Type == wsMsgSegmentFinal || m.Type == wsMsgFinal {
			w.partial = nil
			w.vad = nil
			w.status = nil
		}
		if w.finished {
			// run() is gone, so nothing will ever drain this. Write it here
			// instead: with the drain goroutine returned there is no concurrent
			// writer, and losing a durable silently is the one outcome the
			// durability contract exists to prevent. A dead consumer just makes
			// this fail fast -- the handler's deferred conn.Close() follows.
			w.mu.Unlock()
			_ = w.conn.WriteJSON(m)
			return
		}
		w.durable = append(w.durable, m)
	}
	w.mu.Unlock()
	w.wake()
}

// wsMessageDeliveryClass mirrors sttchain.StreamEvent.DeliveryClass. VAD and
// transient status updates are snapshots, not lossless commitments; keeping
// them durable would let a high-rate status source grow an unbounded queue.
func wsMessageDeliveryClass(m wsMessage) sttchain.DeliveryClass {
	// A processed acknowledgement advances the browser's durable recovery
	// cursor. Provider identity is also durable qualification evidence: it must
	// survive terminal final/done delivery or the browser cannot prove which
	// provider cell actually handled the turn.
	if m.Type == wsMsgStatus && (m.Code == "processed_acknowledgement" || m.Code == "provider_identity") {
		return sttchain.DeliveryDurable
	}
	switch m.Type {
	case wsMsgPartial, wsMsgVadState, wsMsgStatus:
		return sttchain.DeliveryProgress
	default:
		return sttchain.DeliveryDurable
	}
}

func (w *wsCoalescingWriter) run() {
	defer close(w.done)
	for range w.signal {
		for {
			w.mu.Lock()
			var next wsMessage
			has := false
			switch {
			case len(w.durable) > 0:
				next, w.durable = w.durable[0], w.durable[1:]
				has = true
			case w.partial != nil:
				next, w.partial = *w.partial, nil
				has = true
			case w.vad != nil:
				next, w.vad = *w.vad, nil
				has = true
			case w.status != nil:
				next, w.status = *w.status, nil
				has = true
			}
			finished := w.closed && !has
			if finished {
				w.finished = true
			}
			w.mu.Unlock()
			if !has {
				if finished {
					return
				}
				break // wait for the next wake
			}
			_ = w.conn.WriteJSON(next) // may block on a slow consumer — only here
		}
	}
}

// close signals end-of-events and drains queued durables within the bounded
// window; a dead consumer is force-unblocked by closing the conn.
func (w *wsCoalescingWriter) close(timeout time.Duration) {
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()
	w.wake()
	select {
	case <-w.done:
	case <-time.After(timeout):
		_ = w.conn.Close()
		<-w.done
	}
}

type wsMessage struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// SegmentID and Generation make durable segment delivery idempotent across
	// a reconnect. They are absent on legacy (v1) messages.
	SegmentID  string `json:"segmentId,omitempty"`
	Generation uint64 `json:"generation,omitempty"`
	// Code carries a machine-readable status/error class. On Type==wsMsgStatus
	// it is a progress marker such as "stream_connected"; on Type==wsMsgError
	// it is an error class such as "backend_starting", "backend_unavailable",
	// or "provider_failure" so the UI can distinguish retryable backend states.
	Code              string  `json:"code,omitempty"`
	SegmentIndex      int     `json:"segmentIndex,omitempty"`
	ReceivedSequence  int64   `json:"receivedSequence,omitempty"`
	ProcessedSequence int64   `json:"processedSequence,omitempty"`
	ProviderID        string  `json:"providerId,omitempty"`
	ModelID           string  `json:"modelId,omitempty"`
	Score             float64 `json:"score,omitempty"`
	Threshold         float64 `json:"threshold,omitempty"`
	ProfileID         string  `json:"profileId,omitempty"`
	// VAD-state fields (only populated when Type == wsMsgVadState). All
	// are pointer-typed so omitempty drops them from non-VAD messages.
	Voiced           *bool   `json:"voiced,omitempty"`
	SilenceElapsedMs *int64  `json:"silenceElapsedMs,omitempty"`
	SilenceTimeoutMs *int64  `json:"silenceTimeoutMs,omitempty"`
	TickSeq          *uint64 `json:"tickSeq,omitempty"`
	SilenceTimedOut  *bool   `json:"silenceTimedOut,omitempty"`
}

// StreamWSHandler is the browser-voice WebSocket transport. It opens
// the connection, hands off to the Segmenter, and translates the
// transport-free StreamEvent stream into the JSON wire shape the
// audio-integration UI module expects.
//
// Construction takes the same dependency bundle as the Connect handler
// so both transports share one Chain + Selector and cannot drift.
// Replaces the legacy voice.Service.HandleStreamWS path; the
// streaming-pipeline doc explains the architecture and the
// PROBLEMS.md entry "browser WebM partial-decoding regression after
// HandleStreamWS deletion" documents the user-facing behavior change.
func StreamWSHandler(d Deps) http.Handler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  64 * 1024,
		WriteBufferSize: 16 * 1024,
		CheckOrigin:     func(*http.Request) bool { return true },
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.Chain == nil || d.Selector == nil {
			http.Error(w, "stt streaming pipeline not configured", http.StatusServiceUnavailable)
			return
		}
		fault, err := streamTestFaultFromRequest(r, streamTestFaultsAuthorized(d))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if fault.enabled() && d.Logger != nil {
			d.Logger.Printf("voice-ws: deterministic qualification fault armed profile=%s providerBusy=%t closeAfterChunks=%d recoverableCloseAfterChunks=%d closeAfterCommits=%d terminateAfterChunks=%d pauseAfterChunks=%d pauseReadsFor=%s delayProcessedAckFor=%s suppressProcessedAck=%t", fault.profile, fault.providerBusy, fault.closeAfterChunks, fault.closeAfterChunksRecoverable, fault.closeAfterCommits, fault.terminateAfterChunks, fault.pauseAfterChunks, fault.pauseReadsFor, fault.delayProcessedAckFor, fault.suppressProcessedAck)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			d.Logger.Printf("voice-ws: upgrade failed: %v", err)
			return
		}
		defer conn.Close()
		conn.SetReadLimit(maxWSMessageBytes)

		// No absolute wall-clock cap: bound the session by INACTIVITY instead.
		// The old context.WithTimeout(..., 5*time.Minute) fired on active
		// dictation and cold-closed the socket, dropping the tail. The reader
		// below enforces an idle deadline that resets on every audio frame, so
		// a continuously-speaking user is never cut; the session ends only on
		// real silence, a client stop, or request cancellation — all of which
		// drain-then-close via `chunks` closing.
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		chunks := make(chan sttchain.AudioChunk, 16)
		events := make(chan sttchain.StreamEvent, 16)
		// The browser-facing egress is decoupled through a coalescing writer:
		// the events loop enqueues without ever blocking on the socket, so a
		// slow/stalled consumer can never back-pressure the relay and wedge the
		// kyutai reader. Partials coalesce/drop; durables are ordered/lossless.
		writer := newWSCoalescingWriter(conn)
		writer.enqueue(wsMessage{Type: wsMsgStatus, Code: "stream_connected", Text: "Streaming transcription connected."})
		if fault.providerBusy {
			writer.enqueue(wsMessage{Type: wsMsgError, Code: "stt_busy", Text: "STT provider is busy (deterministic qualification fault)."})
			writer.enqueue(wsMessage{Type: wsMsgDone})
			writer.close(wsWriterDrainTimeout)
			return
		}

		start := buildStreamStart(r)
		var ledger *session.Ledger
		var ledgers *session.Registry
		resumed := false
		// Reaching a terminal state does NOT release the session. The server
		// cannot know whether the browser received the terminal, so every
		// outcome keeps the two things a reconnect needs: the committed
		// segments, so replay re-emits the same segment identity instead of
		// re-transcribing into a duplicate, and the terminal reason, so the
		// server's own evidence of a lost turn is distinguishable from a clean
		// one. Eager cleanup here destroyed both — a reconnecting browser got a
		// fresh empty session, and a soak reading the ledger saw no terminal
		// reason at all. The registry's bounded recovery expiry
		// (defaultSessionRecoveryTTL) is what reclaims these, and it already
		// re-arms on every open.
		if start.ProtocolVersion == 2 {
			ledgers = d.Sessions
			if ledgers == nil {
				ledgers = session.NewRegistry(0)
			}
			ledger, resumed, err = ledgers.OpenContext(ctx, start.SessionID, start.ResumeToken)
			if err != nil {
				writer.enqueue(wsMessage{Type: wsMsgError, Code: "invalid_session", Text: err.Error()})
				writer.enqueue(wsMessage{Type: wsMsgDone})
				writer.close(wsWriterDrainTimeout)
				return
			}
			// A deterministic transport fault belongs to one logical browser
			// session, not to every reconnect handshake. Re-arming a close fault
			// on a resumed session creates an artificial infinite recovery loop
			// and prevents the product from ever surfacing a terminal outcome.
			if resumed && fault.enabled() {
				fault = streamTestFault{}
			}
		}
		cfg := resolveStreamPipelineConfigFromDeps(ctx, d)
		virtualReplayAuthorized := streamVirtualReplayAuthorized(d, r)
		// An explicit engine_id on the WebSocket request is the per-session
		// selection used by the browser product path and by the long-form soak.
		// The persisted stream config supplies defaults and policy levers, but it
		// must not overwrite a request-scoped engine choice; doing so made a soak
		// labelled whisper-local actually run Kyutai and invalidated its cell
		// evidence. Reject unknown ids instead of silently falling back.
		if start.EngineID != "" {
			if d.Registry != nil {
				// virtual-replay is intentionally absent from the production
				// manifest. It is valid only when the request carries the
				// explicit virtual-capture pair and the server qualification
				// gate is active; all other unknown engine ids remain rejected.
				virtualReplayRequest := start.EngineID == "virtual-replay" && virtualReplayAuthorized
				if _, ok := d.Registry.Engine(start.EngineID); !ok && !virtualReplayRequest {
					writer.enqueue(wsMessage{Type: wsMsgError, Code: "invalid_engine", Text: fmt.Sprintf("unknown engine_id %q", start.EngineID)})
					writer.enqueue(wsMessage{Type: wsMsgDone})
					writer.close(wsWriterDrainTimeout)
					return
				}
			}
			cfg.EngineID = start.EngineID
		}
		// The accelerated browser soak uses a virtual capture source. It must
		// cross the explicit test-mode query pair plus either the server-owned
		// isolation lease or the operator's server environment gate. The provider
		// is absent from the production engine manifest, so this cannot silently
		// become a user-facing fallback.
		if virtualReplayAuthorized {
			cfg.EngineID = "virtual-replay"
		}
		speakerIsolation := currentSpeakerIsolation(d)
		if virtualReplayAuthorized {
			// The replay provider qualifies browser/ledger durability and native
			// streaming semantics. Speaker verification is an independent,
			// operator-persisted policy and would turn this deterministic test
			// into retained-audio fallback when no enrolled profile is present.
			// The production and realtime paths retain the live speaker policy.
			speakerIsolation = nil
		}
		seg := segmenter.New(segmenter.Deps{Chain: d.Chain, Selector: d.Selector, Engine: d.Engine, Registry: d.Registry, SpeakerIsolation: speakerIsolation, SpeakerExtraction: currentSpeakerExtraction(d)})

		idle := time.Duration(cfg.SessionIdleTimeoutMs) * time.Millisecond
		if idle <= 0 {
			idle = time.Duration(sttpkg.DefaultSessionIdleTimeoutMs) * time.Millisecond
		}

		// Reader: WS frames → chunks channel. Binary frames are raw
		// audio bytes; text frames (legacy VAD signals from the embed)
		// are ignored — the strategy does its own VAD or buffering.
		//
		// Every read arms an idle deadline: if no frame arrives within `idle`,
		// ReadMessage returns a timeout, which we treat as a GRACEFUL end (like
		// an explicit done) rather than an error, so closing `chunks` (defer)
		// routes it through the same drain-then-close path that flushes the
		// trailing segment.
		readerErr := make(chan error, 1)
		go func() {
			defer close(chunks)
			receivedChunks := 0
			for {
				_ = conn.SetReadDeadline(time.Now().Add(idle))
				mt, data, err := conn.ReadMessage()
				if err != nil {
					if ne, ok := err.(net.Error); ok && ne.Timeout() {
						readerErr <- nil // idle timeout: graceful end, drain the tail
					} else {
						readerErr <- err
					}
					return
				}
				if mt == websocket.BinaryMessage {
					chunk := sttchain.AudioChunk{Audio: data}
					if start.ProtocolVersion == 2 {
						parsed, parseErr := decodeWSV2AudioFrame(data)
						if parseErr != nil {
							ledger.Fail(session.TerminalReason("malformed_frame"))
							_ = ledgers.PersistContext(ctx, ledger)
							readerErr <- parseErr
							return
						}
						result, receiveErr := ledger.Receive(session.Chunk{Sequence: parsed.Sequence, StartSample: parsed.StartSample, EndSample: parsed.EndSample, Audio: parsed.Audio})
						if receiveErr != nil {
							ledger.Fail(session.TerminalReason("receive_failed"))
							_ = ledgers.PersistContext(ctx, ledger)
							readerErr <- receiveErr
							return
						}
						if err := ledgers.PersistContext(ctx, ledger); err != nil {
							readerErr <- err
							return
						}
						if shouldSkipReceivedDuplicate(result, resumed) {
							continue
						}
						chunk = parsed
					}
					select {
					case chunks <- chunk:
					case <-ctx.Done():
						readerErr <- ctx.Err()
						return
					}
					receivedChunks++
					if fault.terminateAfterChunks > 0 && receivedChunks >= fault.terminateAfterChunks {
						if ledger != nil {
							ledger.Fail(session.TerminalReason(fault.terminateCode))
							_ = ledgers.PersistContext(ctx, ledger)
						}
						writer.enqueue(wsMessage{Type: wsMsgError, Code: fault.terminateCode, Text: fault.terminateText})
						cancel()
						readerErr <- fmt.Errorf("deterministic stream fault: %s", fault.profile)
						return
					}
					if fault.closeAfterChunks > 0 && receivedChunks >= fault.closeAfterChunks {
						// Close after forwarding the selected chunk, so the pipeline sees
						// a real in-flight transport interruption rather than a synthetic
						// pre-connect error. Deliver a typed durable error before closing:
						// otherwise the client sees only a transport close and can mistake
						// a recoverable missing range for a generic empty transcript.
						if ledger != nil {
							ledger.Fail(session.TerminalReason("test_fault_close_after_chunk"))
							_ = ledgers.PersistContext(ctx, ledger)
						}
						writer.enqueue(wsMessage{Type: wsMsgError, Code: "incomplete_coverage", Text: "Streaming connection closed after deterministic qualification chunk limit."})
						writer.close(wsWriterDrainTimeout)
						cancel()
						readerErr <- errors.New("deterministic stream fault: closed after configured chunk")
						return
					}
					if !resumed && fault.closeAfterChunksRecoverable > 0 && receivedChunks >= fault.closeAfterChunksRecoverable {
						// This is a one-shot transport interruption after the server has
						// durably received the chunk. The resumed handler feeds the retained
						// duplicate back into its fresh segmenter, while the browser replays
						// the same journal entry against the same session.
						if fault.profile == "backend_restart" {
							// A backend restart is a recoverable transport interruption,
							// not a terminal ledger outcome. Keep the session open so the
							// resumed handler can accept the replayed/duplicate chunk.
							writer.enqueue(wsMessage{Type: wsMsgError, Code: "backend_restart", Text: "The speech backend restarted during deterministic qualification."})
						}
						writer.close(wsWriterDrainTimeout)
						cancel()
						readerErr <- errors.New("deterministic stream fault: recoverable close after configured chunk")
						return
					}
					if fault.pauseAfterChunks > 0 && receivedChunks == fault.pauseAfterChunks {
						// Pause at the reader seam after accepting the configured chunk.
						// The next deadline is armed only after the pause, so this models
						// a stalled server consumer rather than an accidental idle timeout.
						if err := waitStreamTestFaultDelay(ctx, fault.pauseReadsFor); err != nil {
							readerErr <- err
							return
						}
					}
					continue
				}
				if mt == websocket.TextMessage {
					var m wsMessage
					if err := json.Unmarshal(data, &m); err == nil && m.Type == "done" {
						readerErr <- nil
						return
					}
				}
			}
		}()

		// Segmenter goroutine. The Segmenter closes events when its
		// strategy returns; the main loop below drains until then.
		runErr := make(chan error, 1)
		go func() {
			runErr <- seg.Run(ctx, start, cfg, chunks, events)
		}()

		segIdx := 0
		if ledger != nil && resumed {
			// A reconnect may have missed a durable segment after the server
			// committed it but before the browser received it. Replay the ledger's
			// stable IDs in timeline order; clients use SegmentID to ignore any
			// copy they already rendered.
			committed := ledger.Snapshot().Committed
			sort.Slice(committed, func(i, j int) bool {
				if committed[i].StartSample != committed[j].StartSample {
					return committed[i].StartSample < committed[j].StartSample
				}
				return committed[i].ID < committed[j].ID
			})
			for _, segment := range committed {
				writer.enqueue(wsMessage{Type: wsMsgSegmentFinal, Text: segment.Text, SegmentID: segment.ID, Generation: start.Generation, SegmentIndex: segIdx})
				segIdx++
			}
		}
		var finalText string
		var providerID string
		var modelID string
		providerCloseReason := "provider_done"
		for ev := range events {
			switch ev.Kind {
			case sttchain.StreamEventAcknowledgement:
				if fault.suppressProcessedAck || ledger == nil || ev.Acknowledgement == nil {
					continue
				}
				ack := ev.Acknowledgement
				if ack.ProcessedSequence < 0 {
					continue
				}
				if err := ledger.AcknowledgeProcessed(uint64(ack.ProcessedSequence)); err != nil {
					ledger.Fail(session.TerminalReason("processed_ack_failed"))
					_ = ledgers.PersistContext(ctx, ledger)
					writer.enqueue(wsMessage{Type: wsMsgError, Code: "processed_ack_failed", Text: "Unable to preserve processed audio coverage."})
					continue
				}
				if err := ledgers.PersistContext(ctx, ledger); err != nil {
					writer.enqueue(wsMessage{Type: wsMsgError, Code: "persistence_failed", Text: "Unable to preserve audio recovery state."})
					continue
				}
				state := ledger.Snapshot()
				writer.enqueue(wsMessage{
					Type: wsMsgStatus, Code: "processed_acknowledgement",
					ReceivedSequence: state.ReceivedSequence, ProcessedSequence: state.ProcessedSequence,
					Text: "Captured audio processing coverage updated.",
				})
			case sttchain.StreamEventPartial:
				if ev.Partial != nil {
					writer.enqueue(wsMessage{Type: wsMsgPartial, Text: ev.Partial.Text})
				}
			case sttchain.StreamEventSegment:
				if ev.Segment != nil {
					if virtualReplayAuthorized {
						d.Logger.Printf("voice-ws: virtual replay segment event start_sample=%d end_sample=%d", ev.Segment.StartSample, ev.Segment.EndSample)
					}
					segmentID := ev.Segment.SegmentID
					if segmentID == "" {
						segmentID = fmt.Sprintf("%s:%d:%d:%d", start.SessionID, start.Generation, ev.Segment.StartSample, ev.Segment.EndSample)
					}
					if ledger != nil {
						isNew, commitErr := ledger.Commit(session.Segment{ID: segmentID, Text: ev.Segment.Text, StartSample: ev.Segment.StartSample, EndSample: ev.Segment.EndSample})
						if commitErr != nil {
							ledger.Fail(session.TerminalReason("commit_conflict"))
							_ = ledgers.PersistContext(ctx, ledger)
							providerCloseReason = "commit_conflict"
							writer.enqueue(wsMessage{Type: wsMsgError, Code: "commit_conflict", Text: "Unable to preserve a durable transcript segment."})
							continue
						}
						if err := ledgers.PersistContext(ctx, ledger); err != nil {
							ledger.Fail(session.TerminalReason("persistence_failed"))
							providerCloseReason = "persistence_failed"
							writer.enqueue(wsMessage{Type: wsMsgError, Code: "persistence_failed", Text: "Unable to preserve transcript recovery state."})
							continue
						}
						if !isNew {
							// The segment was already replayed from the ledger above.
							continue
						}
					}
					writer.enqueue(wsMessage{Type: wsMsgSegmentFinal, Text: ev.Segment.Text, SegmentID: segmentID, Generation: start.Generation, SegmentIndex: segIdx})
					segIdx++
					if fault.closeAfterCommits > 0 && segIdx >= fault.closeAfterCommits {
						// This is deliberately after the durable commit has been written
						// to the session ledger but before final/done. It exercises the
						// browser's resume boundary rather than fabricating a pre-connect
						// failure. The retained audio is left recoverable and the terminal
						// reason remains explicit for diagnostics.
						if ledger != nil {
							ledger.Fail(session.TerminalReason("test_fault_close_before_done"))
							_ = ledgers.PersistContext(ctx, ledger)
						}
						writer.enqueue(wsMessage{Type: wsMsgError, Code: "incomplete_coverage", Text: "Streaming connection closed after a committed segment (deterministic qualification fault)."})
						writer.close(wsWriterDrainTimeout)
						cancel()
						return
					}
				}
			case sttchain.StreamEventVadState:
				if ev.VadState != nil {
					voiced := ev.VadState.Voiced
					elapsed := ev.VadState.SilenceElapsedMs
					timeout := ev.VadState.SilenceTimeoutMs
					seq := ev.VadState.TickSeq
					timedOut := ev.VadState.SilenceTimedOut
					writer.enqueue(wsMessage{
						Type:             wsMsgVadState,
						Voiced:           &voiced,
						SilenceElapsedMs: &elapsed,
						SilenceTimeoutMs: &timeout,
						TickSeq:          &seq,
						SilenceTimedOut:  &timedOut,
					})
				}
			case sttchain.StreamEventSpeakerRejection:
				if ev.SpeakerRejection != nil {
					writer.enqueue(wsMessage{
						Type:      wsMsgSegmentRejected,
						Score:     ev.SpeakerRejection.Score,
						Threshold: ev.SpeakerRejection.Threshold,
					})
				}
			case sttchain.StreamEventError:
				msg := ""
				if ev.Error != nil {
					msg = ev.Error.Error()
				}
				code := streamErrorCode(ev.Error)
				providerCloseReason = "provider_error:" + code
				writer.enqueue(wsMessage{Type: wsMsgError, Text: msg, Code: code})
			case sttchain.StreamEventDone:
				if ev.Done != nil {
					finalText = ev.Done.FinalText
					providerID = ev.Done.ProviderID
					modelID = ev.Done.ModelID
				}
			}
		}
		if fault.closeAfterChunksRecoverable > 0 && !resumed {
			// A recoverable qualification close is a transport boundary, not a
			// terminal turn outcome. The browser will reconnect with the same
			// session and replay token; finalising this first handler would mark
			// the ledger terminal before that resume could occur and surface a
			// false "Recording failed." state to the composer.
			<-runErr
			var readerCloseErr error
			select {
			case readerCloseErr = <-readerErr:
			default:
			}
			emitStreamDeliveryTelemetry(d, segIdx, false, readerCloseErr, "recoverable_transport_boundary")
			return
		}
		// The Done event carries `committed` — the full concatenation of
		// every segment-final emitted this session. Re-sending it as
		// wsMsgFinal duplicates text the client already appended (the
		// client's onResult callback contract is "deliver only the
		// un-segmented tail"). When at least one segment-final was emitted
		// the final text is fully redundant; send empty so the client can
		// transition state without re-appending. Plain non-segmenting
		// strategies (passthrough, buffered fallback) still need finalText.
		var terminalReason session.TerminalReason
		if ledger != nil {
			state := ledger.Snapshot()
			if state.ReceivedSequence >= 0 {
				if fault.suppressProcessedAck {
					// A done outcome with unacknowledged audio must never look like
					// success. Keep the replay tail, persist the terminal reason, and
					// give the browser an actionable durable error instead of silently
					// compacting its journal.
					ledger.Fail(session.TerminalIncompleteCoverage)
					writer.enqueue(wsMessage{Type: wsMsgError, Code: "incomplete_coverage", Text: "Audio processing acknowledgement was intentionally withheld (deterministic qualification fault)."})
				} else if fault.delayProcessedAckFor > 0 {
					if err := waitStreamTestFaultDelay(ctx, fault.delayProcessedAckFor); err != nil {
						ledger.Fail(session.TerminalReason("test_fault_ack_delay_cancelled"))
					}
				}
				if state = ledger.Snapshot(); state.TerminalReason == "" {
					if err := ledger.AcknowledgeProcessed(uint64(state.ReceivedSequence)); err != nil {
						ledger.Fail(session.TerminalReason("processed_ack_failed"))
					} else if err := ledger.Complete(); err != nil && !errors.Is(err, session.ErrTerminal) {
						ledger.Fail(session.TerminalIncompleteCoverage)
					}
				}
			}
			if err := ledgers.PersistContext(ctx, ledger); err != nil {
				writer.enqueue(wsMessage{Type: wsMsgError, Code: "persistence_failed", Text: "Unable to preserve audio recovery state."})
			}
			state = ledger.Snapshot()
			terminalReason = state.TerminalReason
			if !fault.suppressProcessedAck {
				writer.enqueue(wsMessage{
					Type: wsMsgStatus, Code: "processed_acknowledgement",
					ReceivedSequence: state.ReceivedSequence, ProcessedSequence: state.ProcessedSequence,
					Text: "Captured audio processing coverage updated.",
				})
			}
		}
		if providerID != "" || modelID != "" {
			writer.enqueue(wsMessage{Type: wsMsgStatus, Code: "provider_identity", ProviderID: providerID, ModelID: modelID, Text: "Streaming provider identity observed."})
		}
		if segIdx > 0 {
			writer.enqueue(wsMessage{Type: wsMsgFinal, Text: ""})
		} else {
			writer.enqueue(wsMessage{Type: wsMsgFinal, Text: finalText})
		}
		writer.enqueue(wsMessage{Type: wsMsgDone, Code: string(terminalReason)})

		// Drain-then-close the browser egress: flush queued durables (including
		// the terminal final + done) to the consumer within the bounded window,
		// then stop the writer. A dead consumer is force-unblocked by closing
		// the conn so teardown never hangs.
		writer.close(wsWriterDrainTimeout)

		<-runErr
		// The reader goroutine always sends exactly once on readerErr before it
		// returns (and before its deferred close(chunks) lets the segmenter
		// finish), so by the time the events loop has drained the receive is
		// already buffered — block for it to classify the close accurately.
		var readerCloseErr error
		select {
		case readerCloseErr = <-readerErr:
		default:
		}
		emitStreamDeliveryTelemetry(d, segIdx, finalText != "", readerCloseErr, providerCloseReason)
	})
}

// streamCloseOutcome classifies why a streaming session ended, for the
// per-session delivery telemetry. A nil reader error is a graceful end
// (idle-timeout or an explicit client "done") — the drain-then-close path
// ran and the trailing segment was flushed. A context.Canceled is the
// request/context cancel path; anything else is an abrupt read error. In the
// non-graceful paths the final flush may have been lost, so they are the
// signal Phase 8 exists to surface.
func streamCloseOutcome(readerCloseErr error) string {
	switch {
	case readerCloseErr == nil:
		return "graceful"
	case errors.Is(readerCloseErr, context.Canceled):
		return "cancel"
	default:
		return "read_error"
	}
}

// emitStreamDeliveryTelemetry records a per-session STT delivery summary so a
// silent tail-drop stops being invisible (Phase 8). It emits an always-on
// structured log line and, when a usage recorder is wired, a durable
// `stream_session` usage row whose FallbackReason carries the close outcome
// and whose Error is set on a non-graceful close (the drop metric). Both are
// best-effort and must never affect the live session.
func emitStreamDeliveryTelemetry(d Deps, segments int, tailFinalDelivered bool, readerCloseErr error, providerCloseReason string) {
	outcome := streamCloseOutcome(readerCloseErr)
	graceful := outcome == "graceful"
	if d.Logger != nil {
		d.Logger.Printf("voice-ws: session closed outcome=%s providerCloseReason=%s segments=%d tailFinalDelivered=%t graceful=%t",
			outcome, providerCloseReason, segments, tailFinalDelivered, graceful)
	}
	if d.Usage == nil {
		return
	}
	now := time.Now()
	if d.Clock != nil {
		now = d.Clock.Now()
	}
	row := store.UsageRow{
		OperationID:    uuid.NewString(),
		EmittedAt:      now,
		Capability:     "stt",
		Operation:      "stream_session",
		OutputTokens:   protoint.FromInt(segments), // segments committed this session
		FallbackReason: outcome,
	}
	if !graceful {
		// A non-graceful close is the drop signal: the trailing segment may
		// not have been flushed. Populating Error makes it count in the usage
		// summary's ErrorCount aggregate.
		row.Error = "tail_drain_" + outcome
	}
	d.Usage.Enqueue(row)
}

func shouldSkipReceivedDuplicate(result session.ReceiveResult, resumed bool) bool {
	return result == session.ReceivedDuplicate && !resumed
}

// buildStreamStart maps the WS request's query params + auth envelope to
// the transport-free StreamStart. The `format` query param declares the
// inbound codec (audioformat vocabulary, e.g. "webm", "pcm_s16le"); an
// empty value leaves the Segmenter to sniff the first chunk
// (declare-or-sniff). This mirrors the Connect transport's mapping of
// StreamStart.input_format so both transports declare the codec the same
// way — see the parity test.
func buildStreamStart(r *http.Request) sttchain.StreamStart {
	q := r.URL.Query()
	env := envelope.FromHTTP(r.Header)
	return sttchain.StreamStart{
		ProtocolVersion: func() int32 {
			if q.Get("protocol_version") == "2" {
				return 2
			}
			return 0
		}(),
		SessionID:    q.Get("session_id"),
		ResumeToken:  q.Get("resume_token"),
		EngineID:     q.Get("engine_id"),
		Language:     q.Get("language"),
		InputFormat:  q.Get("format"),
		BYOKProvider: env.Provider,
		BYOKKey:      env.Key,
		LPBSToken:    env.LPBSToken,
		UserIdentity: env.UserIdentity,
	}
}

// resolveStreamPipelineConfigFromDeps reads the persisted operator
// lever set from the StreamConfig store via the same shape as
// admin_handlers.go::resolveStreamPipelineConfig. Duplicated here so
// the WS handler does not need a Connect handler instance.
func resolveStreamPipelineConfigFromDeps(ctx context.Context, d Deps) sttpkg.StreamConfig {
	h := &connectHandler{deps: d}
	return h.resolveStreamPipelineConfig(ctx)
}
