package sttchain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"audio-tools/internal/clock"
	"audio-tools/internal/httpc"
)

// kyutaiDrainTimeout bounds how long the reader is given to receive the
// server's flush/done after an end marker is sent on a cancelled session. The
// measured kyutai flush lag is ~0.07s (RTF 0.13), so this is generous; it only
// exists so a wedged backend cannot hang teardown forever.
const kyutaiDrainTimeout = 5 * time.Second

// KyutaiProvider is the Local-tier adapter for the Kyutai streaming STT
// engine (resources/kyutai-stt). Unlike the Whisper LocalProvider (batch),
// Kyutai is NATIVELY streaming: it declares Traits{Stream:true,
// Strategies:[passthrough]} so the selector pairs it with the Passthrough
// strategy and the Segmenter skips its own VAD chunking. Both engines are
// Local-tier; the chain distinguishes them via StreamStart.EngineID.
//
// The adapter speaks the resource's STABLE WebSocket contract (documented in
// resources/kyutai-stt/docs/API.md), NOT the upstream moshi-server protocol —
// the resource owns that translation (wrap-not-use). Contract:
//   - client TEXT  {"type":"start","sample_rate":16000,"language":"en"}
//   - client BINARY frames: canonical PCM s16le, 16 kHz, mono
//   - client TEXT  {"type":"end"}
//   - server TEXT  {"type":"partial","text":...}
//     {"type":"segment","text":...,"start_ms":..,"end_ms":..}
//     {"type":"done"} | {"type":"error","message":...}
//
// seam: KyutaiProvider is a sttchain.Provider (SEAMS.md row
// "sttchain.Provider"). Production wires it from bootstrap with the resource
// base URL resolved via the env discovery exports; tests inject StreamEndpoint
// pointed at internal/testutil/vendorws.NewKyutaiServer.
type KyutaiProvider struct {
	// BaseURL is the resource HTTP base (e.g. http://localhost:8092),
	// resolved from the KYUTAI_STT_URL discovery export. /health and /v1/info
	// are probed on it; the WS stream URL is derived (http->ws + /v1/stream).
	BaseURL string
	// StreamEndpoint overrides the derived ws:// stream URL — tests point it
	// at a vendorws fake (rewritten ws://). Empty uses the derived URL.
	StreamEndpoint string
	// ModelID is the engine model identifier reported by Model(); resolved at
	// construction from the manifest/env. Empty falls back to "kyutai".
	ModelID string
	Doer    httpc.Doer
	Clock   clock.Clock
}

// NewKyutaiProvider constructs the Kyutai adapter for the given resource base
// URL. baseURL is the http(s) base; the WS stream URL is derived from it.
func NewKyutaiProvider(baseURL string) *KyutaiProvider {
	return &KyutaiProvider{
		BaseURL: baseURL,
		ModelID: "kyutai",
		Doer:    httpc.DefaultDoer(),
		Clock:   clock.System{},
	}
}

func (p *KyutaiProvider) Type() ProviderTier { return TierLocal }

func (p *KyutaiProvider) Model() string {
	if p == nil || p.ModelID == "" {
		return "kyutai"
	}
	return p.ModelID
}

// IsAvailable probes the resource health endpoint. Kyutai is unavailable
// (engine hidden from the picker / not selectable) when the resource is down
// or its model has not finished loading — graceful degradation, never an error.
func (p *KyutaiProvider) IsAvailable(ctx context.Context) bool {
	if p == nil || p.BaseURL == "" || p.Doer == nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint("/health"), nil)
	if err != nil {
		return false
	}
	resp, err := p.Doer.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var body struct {
		Status      string `json:"status"`
		ModelLoaded bool   `json:"model_loaded"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false
	}
	return body.ModelLoaded
}

// Transcribe is unsupported: Kyutai is a streaming-only engine. The unary
// chain never reaches it (the Local unary slot is always Whisper); this guards
// against a future caller wiring Kyutai into a batch path by mistake.
func (p *KyutaiProvider) Transcribe(_ context.Context, _ Request) (*Result, error) {
	return nil, fmt.Errorf("audio-tools/sttchain: kyutai is a streaming-only engine; use TranscribeStreaming")
}

// Traits declares Kyutai as native-streaming, Passthrough-only. The manifest
// (internal/sttengine) is the authority on strategy eligibility; this mirrors
// it so the selector's provider-traits fallback path stays correct when no
// registry is wired (tests).
func (p *KyutaiProvider) Traits() ProviderTraits {
	return ProviderTraits{
		Batch:      false,
		Stream:     true,
		Strategies: []StrategyKind{StrategyPassthrough},
	}
}

// TranscribeStreaming waits for the first canonical-PCM chunk before opening
// the resource WS, then sends the start header, pumps chunks as binary frames,
// sends the end marker when chunks close, and translates the resource's JSON
// event frames into StreamEvents. Zero-audio sessions complete without dialing
// kyutai, so browser pre-connects never contend for the single-session model
// lock.
func (p *KyutaiProvider) TranscribeStreaming(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error) {
	if p == nil || p.BaseURL == "" {
		return nil, fmt.Errorf("audio-tools/sttchain: kyutai provider not configured")
	}
	streamURL := p.StreamEndpoint
	if streamURL == "" {
		streamURL = p.streamURL()
	}
	events := make(chan StreamEvent, 16)
	model := p.Model()
	go func() {
		first, ok := awaitFirstAudioChunk(ctx, chunks)
		if !ok {
			events <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
				LockedTier: TierLocal, ProviderID: "kyutai", ModelID: model,
			}}
			close(events)
			return
		}

		dialCtx, dialCancel := context.WithTimeout(context.Background(), kyutaiDrainTimeout)
		conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, streamURL, nil)
		dialCancel()
		if err != nil {
			events <- StreamEvent{Kind: StreamEventError, Error: fmt.Errorf("kyutai: ws dial: %w", err)}
			events <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
				LockedTier: TierLocal, ProviderID: "kyutai", ModelID: model,
			}}
			close(events)
			return
		}

		// The resource contract is fixed at canonical 16 kHz mono s16le and rejects
		// any other rate. Because Kyutai declares requires.pcm16kMono, the Segmenter
		// has already normalized the inbound chunks to canonical PCM before they
		// reach this adapter, so the wire rate is always 16000 regardless of the
		// session's original input rate hint.
		startMsg, _ := json.Marshal(map[string]any{
			"type":        "start",
			"sample_rate": 16000,
			"language":    start.Language,
		})
		if err := conn.WriteMessage(websocket.TextMessage, startMsg); err != nil {
			_ = conn.Close()
			events <- StreamEvent{Kind: StreamEventError, Error: fmt.Errorf("kyutai: write start: %w", err)}
			events <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
				LockedTier: TierLocal, ProviderID: "kyutai", ModelID: model,
			}}
			close(events)
			return
		}

		// streamDone is closed by the reader when it returns; the ctx watcher and
		// the writer select on it so neither outlives the session.
		streamDone := make(chan struct{})

		// DEDICATED WRITER. gorilla forbids concurrent writers, but serializing
		// writes behind a mutex meant a blocked WriteMessage (a stalled consumer)
		// was held WHILE LOCKED, so the cancel path's end marker could never be
		// sent and teardown wedged forever. Instead a single writer goroutine owns
		// all conn writes, fed by a buffered channel. NOTHING is held across a
		// blocking write, so the pump and the cancel watcher always make progress.
		type writeReq struct {
			mt   int
			data []byte
		}
		writeCh := make(chan writeReq, 16)
		writerGone := make(chan struct{})
		send := func(mt int, data []byte) bool {
			select {
			case writeCh <- writeReq{mt: mt, data: data}:
				return true
			case <-writerGone:
				return false
			case <-streamDone:
				return false
			}
		}
		var endOnce sync.Once
		sendEnd := func() {
			endOnce.Do(func() { send(websocket.TextMessage, []byte(`{"type":"end"}`)) })
		}
		go func() {
			defer close(writerGone)
			for {
				select {
				case req := <-writeCh:
					if err := conn.WriteMessage(req.mt, req.data); err != nil {
						return
					}
				case <-streamDone:
					return
				}
			}
		}()

		// ctx watcher: DRAIN-THEN-CLOSE. Cancelling the session must not cold-close
		// the socket (that drops kyutai's flush and the trailing segment). Instead
		// enqueue the end marker so the server flushes, give the reader a bounded
		// window to receive that flush/done, and only then close the conn.
		go func() {
			select {
			case <-ctx.Done():
				sendEnd()
				select {
				case <-streamDone:
				case <-time.After(kyutaiDrainTimeout):
				}
				_ = conn.Close()
			case <-streamDone:
			}
		}()

		// Pump: first chunk (which triggered the lazy dial), then remaining inbound
		// PCM chunks -> WS binary frames. On clean close enqueue end so the server
		// flushes its tail. On ctx cancel the watcher owns end + bounded drain.
		go func(ctx context.Context) {
			if !send(websocket.BinaryMessage, first.Audio) {
				return
			}
			for {
				select {
				case <-ctx.Done():
					return
				case ch, ok := <-chunks:
					if !ok {
						sendEnd()
						return
					}
					if !send(websocket.BinaryMessage, ch.Audio) {
						return
					}
				}
			}
		}(ctx)

		// Reader: WS JSON frames -> StreamEvents. It intentionally does NOT bail on
		// ctx cancel: after a cancel the watcher has sent the end marker and the
		// server is flushing, so the reader must keep reading to deliver that
		// trailing segment + done. The watcher's bounded drain unblocks this read
		// if the backend wedges.
		defer close(events)
		defer close(streamDone)
		defer conn.Close()
		var finalText string
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				events <- StreamEvent{Kind: StreamEventError, Error: fmt.Errorf("kyutai: unexpected websocket close before done: %w", err)}
				events <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
					FinalText: strings.TrimSpace(finalText), LockedTier: TierLocal,
					ProviderID: "kyutai", ModelID: model,
				}}
				return
			}
			var msg struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				StartMs int64  `json:"start_ms"`
				EndMs   int64  `json:"end_ms"`
				Message string `json:"message"`
				Code    string `json:"code"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case "partial":
				if msg.Text != "" {
					events <- StreamEvent{Kind: StreamEventPartial, Partial: &PartialEvent{Text: msg.Text}}
				}
			case "segment":
				if msg.Text == "" {
					continue
				}
				finalText += " " + msg.Text
				events <- StreamEvent{Kind: StreamEventSegment, Segment: &SegmentEvent{
					Text:             msg.Text,
					StartMs:          msg.StartMs,
					EndMs:            msg.EndMs,
					DetectedLanguage: start.Language,
					ProviderTier:     TierLocal,
					ProviderID:       "kyutai",
					ModelID:          model,
				}}
			case "error":
				if msg.Code != "" {
					events <- StreamEvent{Kind: StreamEventError, Error: fmt.Errorf("kyutai: %s: %s", msg.Code, msg.Message)}
				} else {
					events <- StreamEvent{Kind: StreamEventError, Error: fmt.Errorf("kyutai: %s", msg.Message)}
				}
			case "done":
				events <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
					FinalText: strings.TrimSpace(finalText), LockedTier: TierLocal,
					ProviderID: "kyutai", ModelID: model,
				}}
				return
			}
		}
	}()
	return events, nil
}

func awaitFirstAudioChunk(ctx context.Context, chunks <-chan AudioChunk) (AudioChunk, bool) {
	select {
	case <-ctx.Done():
		return AudioChunk{}, false
	case ch, ok := <-chunks:
		if !ok || len(ch.Audio) == 0 {
			return AudioChunk{}, false
		}
		return ch, true
	}
}

func (p *KyutaiProvider) endpoint(path string) string {
	return strings.TrimRight(p.BaseURL, "/") + path
}

// streamURL derives the ws:// stream URL from the http(s) base URL.
func (p *KyutaiProvider) streamURL() string {
	base := strings.TrimRight(p.BaseURL, "/")
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	return base + "/v1/stream"
}
