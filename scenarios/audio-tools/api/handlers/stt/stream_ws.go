package stt

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter"
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

// wsWriterDrainTimeout bounds how long teardown waits for the coalescing
// writer to flush queued durable messages to a slow consumer before the socket
// is force-closed. Committed text is flushed within this window; a dead
// consumer cannot hang teardown past it.
const wsWriterDrainTimeout = 5 * time.Second

// wsCoalescingWriter owns every write to the browser socket. It applies the
// event-durability contract to the browser-facing egress so a slow or stalled
// consumer can never back-pressure the upstream relay — and therefore can never
// wedge the kyutai socket reader. Durable messages (status, segment-final,
// rejection, error, final, done, vad-state) are queued losslessly and in
// order; partial messages coalesce into a single latest slot and may be dropped
// under backpressure. The events loop enqueues without ever blocking on the
// socket; only this one goroutine blocks on a slow consumer.
// See docs/domains/stt/streaming-pipeline.md#event-durability-contract.
type wsCoalescingWriter struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	durable []wsMessage
	partial *wsMessage
	closed  bool
	signal  chan struct{}
	done    chan struct{}
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
// the socket. Partial → coalesce-to-latest (droppable); everything else →
// durable ordered queue. A committed/terminal durable (segment-final / final)
// supersedes any unsent interim partial so stale text is never re-shown.
func (w *wsCoalescingWriter) enqueue(m wsMessage) {
	w.mu.Lock()
	if m.Type == wsMsgPartial {
		mm := m
		w.partial = &mm
	} else {
		if m.Type == wsMsgSegmentFinal || m.Type == wsMsgFinal {
			w.partial = nil
		}
		w.durable = append(w.durable, m)
	}
	w.mu.Unlock()
	w.wake()
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
			}
			finished := w.closed && !has
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
	// Code carries a machine-readable status/error class. On Type==wsMsgStatus
	// it is a progress marker such as "stream_connected"; on Type==wsMsgError
	// it is an error class such as "backend_starting", "backend_unavailable",
	// or "provider_failure" so the UI can distinguish retryable backend states.
	Code         string  `json:"code,omitempty"`
	SegmentIndex int     `json:"segmentIndex,omitempty"`
	Score        float64 `json:"score,omitempty"`
	Threshold    float64 `json:"threshold,omitempty"`
	ProfileID    string  `json:"profileId,omitempty"`
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
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			d.Logger.Printf("voice-ws: upgrade failed: %v", err)
			return
		}
		defer conn.Close()

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

		start := buildStreamStart(r)
		cfg := resolveStreamPipelineConfigFromDeps(ctx, d)
		seg := segmenter.New(segmenter.Deps{Chain: d.Chain, Selector: d.Selector, Engine: d.Engine, Registry: d.Registry, SpeakerIsolation: currentSpeakerIsolation(d), SpeakerExtraction: currentSpeakerExtraction(d)})

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
					select {
					case chunks <- sttchain.AudioChunk{Audio: data}:
					case <-ctx.Done():
						readerErr <- ctx.Err()
						return
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
		var finalText string
		providerCloseReason := "provider_done"
		for ev := range events {
			switch ev.Kind {
			case sttchain.StreamEventPartial:
				if ev.Partial != nil {
					writer.enqueue(wsMessage{Type: wsMsgPartial, Text: ev.Partial.Text})
				}
			case sttchain.StreamEventSegment:
				if ev.Segment != nil {
					writer.enqueue(wsMessage{Type: wsMsgSegmentFinal, Text: ev.Segment.Text, SegmentIndex: segIdx})
					segIdx++
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
				}
			}
		}
		// The Done event carries `committed` — the full concatenation of
		// every segment-final emitted this session. Re-sending it as
		// wsMsgFinal duplicates text the client already appended (the
		// client's onResult callback contract is "deliver only the
		// un-segmented tail"). When at least one segment-final was emitted
		// the final text is fully redundant; send empty so the client can
		// transition state without re-appending. Plain non-segmenting
		// strategies (passthrough, buffered fallback) still need finalText.
		if segIdx > 0 {
			writer.enqueue(wsMessage{Type: wsMsgFinal, Text: ""})
		} else {
			writer.enqueue(wsMessage{Type: wsMsgFinal, Text: finalText})
		}
		writer.enqueue(wsMessage{Type: wsMsgDone})

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
		OutputTokens:   int32(segments), // segments committed this session
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
