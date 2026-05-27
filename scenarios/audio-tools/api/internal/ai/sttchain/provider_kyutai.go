package sttchain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"audio-tools/internal/clock"
	"audio-tools/internal/httpc"
)

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

// TranscribeStreaming opens the resource WS, sends the start header, pumps
// canonical-PCM chunks as binary frames, sends the end marker when chunks
// close, and translates the resource's JSON event frames into StreamEvents.
// The returned channel is closed after the terminal Done (emitted on a
// {"type":"done"} frame or when the WS closes).
func (p *KyutaiProvider) TranscribeStreaming(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error) {
	if p == nil || p.BaseURL == "" {
		return nil, fmt.Errorf("audio-tools/sttchain: kyutai provider not configured")
	}
	streamURL := p.StreamEndpoint
	if streamURL == "" {
		streamURL = p.streamURL()
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, streamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("kyutai: ws dial: %w", err)
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
		return nil, fmt.Errorf("kyutai: write start: %w", err)
	}

	events := make(chan StreamEvent, 16)
	model := p.Model()

	// streamDone is closed by the reader when it returns; the ctx watcher
	// selects on it so it never outlives the session.
	streamDone := make(chan struct{})

	// ctx watcher: cancelling the session closes the conn, which unblocks both
	// the pump's WriteMessage and the reader's blocking ReadMessage so they exit
	// promptly (a blocking ReadMessage can't select on ctx itself).
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-streamDone:
		}
	}()

	// Pump: inbound PCM chunks -> WS binary frames; on close send {"type":"end"}.
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case ch, ok := <-chunks:
				if !ok {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"end"}`))
					return
				}
				if err := conn.WriteMessage(websocket.BinaryMessage, ch.Audio); err != nil {
					return
				}
			}
		}
	}(ctx)

	// Reader: WS JSON frames -> StreamEvents.
	go func(ctx context.Context) {
		defer close(events)
		defer close(streamDone)
		defer conn.Close()
		var finalText string
		for {
			if ctx.Err() != nil {
				return
			}
			_, data, err := conn.ReadMessage()
			if err != nil {
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
				events <- StreamEvent{Kind: StreamEventError, Error: fmt.Errorf("kyutai: %s", msg.Message)}
			case "done":
				events <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
					FinalText: strings.TrimSpace(finalText), LockedTier: TierLocal,
					ProviderID: "kyutai", ModelID: model,
				}}
				return
			}
		}
	}(ctx)
	return events, nil
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
