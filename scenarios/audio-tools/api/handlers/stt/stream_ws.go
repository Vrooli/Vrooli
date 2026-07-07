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
		var writeMu sync.Mutex
		writeJSON := func(m wsMessage) {
			writeMu.Lock()
			defer writeMu.Unlock()
			_ = conn.WriteJSON(m)
		}
		writeJSON(wsMessage{Type: wsMsgStatus, Code: "stream_connected", Text: "Streaming transcription connected."})

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
		for ev := range events {
			switch ev.Kind {
			case sttchain.StreamEventPartial:
				if ev.Partial != nil {
					writeJSON(wsMessage{Type: wsMsgPartial, Text: ev.Partial.Text})
				}
			case sttchain.StreamEventSegment:
				if ev.Segment != nil {
					writeJSON(wsMessage{Type: wsMsgSegmentFinal, Text: ev.Segment.Text, SegmentIndex: segIdx})
					segIdx++
				}
			case sttchain.StreamEventVadState:
				if ev.VadState != nil {
					voiced := ev.VadState.Voiced
					elapsed := ev.VadState.SilenceElapsedMs
					timeout := ev.VadState.SilenceTimeoutMs
					seq := ev.VadState.TickSeq
					timedOut := ev.VadState.SilenceTimedOut
					writeJSON(wsMessage{
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
					writeJSON(wsMessage{
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
				writeJSON(wsMessage{Type: wsMsgError, Text: msg, Code: streamErrorCode(ev.Error)})
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
			writeJSON(wsMessage{Type: wsMsgFinal, Text: ""})
		} else {
			writeJSON(wsMessage{Type: wsMsgFinal, Text: finalText})
		}
		writeJSON(wsMessage{Type: wsMsgDone})

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
		emitStreamDeliveryTelemetry(d, segIdx, finalText != "", readerCloseErr)
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
func emitStreamDeliveryTelemetry(d Deps, segments int, tailFinalDelivered bool, readerCloseErr error) {
	outcome := streamCloseOutcome(readerCloseErr)
	graceful := outcome == "graceful"
	if d.Logger != nil {
		d.Logger.Printf("voice-ws: session closed outcome=%s segments=%d tailFinalDelivered=%t graceful=%t",
			outcome, segments, tailFinalDelivered, graceful)
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
