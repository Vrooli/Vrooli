package stt

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/byok/envelope"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter"
)

// Voice streaming WS message types. These match the wire shape the
// browser embed (@audio-tools/embed) consumes; transport adapters
// must not change them without coordinating with that package.
const (
	wsMsgPartial         = "partial"
	wsMsgFinal           = "final"
	wsMsgSegmentFinal    = "segment-final"
	wsMsgSegmentRejected = "segment-rejected"
	wsMsgError           = "error"
	wsMsgDone            = "done"
)

type wsMessage struct {
	Type         string  `json:"type"`
	Text         string  `json:"text,omitempty"`
	SegmentIndex int     `json:"segmentIndex,omitempty"`
	Score        float64 `json:"score,omitempty"`
	Threshold    float64 `json:"threshold,omitempty"`
	ProfileID    string  `json:"profileId,omitempty"`
}

// StreamWSHandler is the browser-voice WebSocket transport. It opens
// the connection, hands off to the Segmenter, and translates the
// transport-free StreamEvent stream into the JSON wire shape the
// @audio-tools/embed package expects.
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
			log.Printf("voice-ws: upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()

		chunks := make(chan sttchain.AudioChunk, 16)
		events := make(chan sttchain.StreamEvent, 16)
		var writeMu sync.Mutex
		writeJSON := func(m wsMessage) {
			writeMu.Lock()
			defer writeMu.Unlock()
			_ = conn.WriteJSON(m)
		}

		env := envelope.FromHTTP(r.Header)
		start := sttchain.StreamStart{
			Language:     r.URL.Query().Get("language"),
			BYOKProvider: env.Provider,
			BYOKKey:      env.Key,
			LPBSToken:    env.LPBSToken,
			UserIdentity: env.UserIdentity,
		}
		cfg := resolveStreamPipelineConfigFromDeps(ctx, d)
		seg := segmenter.New(segmenter.Deps{Chain: d.Chain, Selector: d.Selector})

		// Reader: WS frames → chunks channel. Binary frames are raw
		// audio bytes; text frames (legacy VAD signals from the embed)
		// are ignored — the strategy does its own VAD or buffering.
		readerErr := make(chan error, 1)
		go func() {
			defer close(chunks)
			for {
				mt, data, err := conn.ReadMessage()
				if err != nil {
					readerErr <- err
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
			case sttchain.StreamEventSpeakerRejection:
				if ev.SpeakerRejection != nil {
					writeJSON(wsMessage{Type: wsMsgSegmentRejected})
				}
			case sttchain.StreamEventError:
				msg := ""
				if ev.Error != nil {
					msg = ev.Error.Error()
				}
				writeJSON(wsMessage{Type: wsMsgError, Text: msg})
			case sttchain.StreamEventDone:
				if ev.Done != nil {
					finalText = ev.Done.FinalText
				}
			}
		}
		writeJSON(wsMessage{Type: wsMsgFinal, Text: finalText})
		writeJSON(wsMessage{Type: wsMsgDone})

		_ = <-runErr
		select {
		case <-readerErr:
		default:
		}
	})
}

// resolveStreamPipelineConfigFromDeps reads the persisted operator
// lever set from the StreamConfig store via the same shape as
// admin_handlers.go::resolveStreamPipelineConfig. Duplicated here so
// the WS handler does not need a Connect handler instance.
func resolveStreamPipelineConfigFromDeps(ctx context.Context, d Deps) sttpkg.StreamConfig {
	h := &connectHandler{deps: d}
	return h.resolveStreamPipelineConfig(ctx)
}
