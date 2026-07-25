package sttchain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// kyutaiMaxInFlightBatches bounds raw PCM accepted by the resource but not yet
// decoded. It preserves WebSocket control-frame liveness under deterministic
// replay instead of relying on socket-buffer growth as an implicit queue.
const kyutaiMaxInFlightBatches = 8

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
		ModelID: kyutaiModelID(),
		Doer:    httpc.DefaultDoer(),
		Clock:   clock.System{},
	}
}

// kyutaiModelID is explicit runtime provenance, not the engine id. Operators
// overriding the resource model must export AUDIO_KYUTAI_MODEL_ID alongside
// the resource configuration; promotion evidence then cannot cross that model
// boundary. The bundled resource's declared default is used otherwise.
func kyutaiModelID() string {
	if raw, configured := os.LookupEnv("AUDIO_KYUTAI_MODEL_ID"); configured && strings.TrimSpace(raw) != "" {
		model := strings.TrimSpace(raw)
		return model
	}
	if raw, configured := os.LookupEnv("KYUTAI_STT_HF_REPO"); configured && strings.TrimSpace(raw) != "" {
		model := strings.TrimSpace(raw)
		return model
	}
	return "kyutai/stt-1b-en_fr"
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

		// Keep request values but deliberately detach cancellation for this short
		// dial. A request can cancel after its first PCM chunk is accepted and
		// before the socket exists; we must still establish the connection long
		// enough to send its terminal drain marker and retain Kyutai's tail.
		dialCtx, dialCancel := context.WithTimeout(context.WithoutCancel(ctx), kyutaiDrainTimeout)
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
		// The resource accepts the start control frame immediately but may place
		// the session in FIFO admission. Never push PCM while it is queued: the
		// upstream session ledger/browser journal is the replay authority, not a
		// resource socket buffer. The reader closes this only on an explicit
		// `ready` status.
		readyForAudio := make(chan struct{})
		var readyOnce sync.Once

		// DEDICATED WRITER. gorilla forbids concurrent writers, but serializing
		// writes behind a mutex meant a blocked WriteMessage (a stalled consumer)
		// was held WHILE LOCKED, so the cancel path's end marker could never be
		// sent and teardown wedged forever. Instead a single writer goroutine owns
		// all conn writes, fed by a buffered channel. NOTHING is held across a
		// blocking write, so the pump and the cancel watcher always make progress.
		type writeReq struct {
			mt    int
			data  []byte
			audio bool
		}
		writeCh := make(chan writeReq, 16)
		credits := make(chan struct{}, kyutaiMaxInFlightBatches)
		for range kyutaiMaxInFlightBatches {
			credits <- struct{}{}
		}
		writerGone := make(chan struct{})
		send := func(mt int, data []byte, audio bool) bool {
			select {
			case writeCh <- writeReq{mt: mt, data: data, audio: audio}:
				return true
			case <-writerGone:
				return false
			case <-streamDone:
				return false
			}
		}
		var endOnce sync.Once
		sendEnd := func() {
			endOnce.Do(func() { send(websocket.TextMessage, []byte(`{"type":"end"}`), false) })
		}
		go func(done <-chan struct{}) {
			defer close(writerGone)
			for {
				select {
				case req := <-writeCh:
					if req.audio {
						select {
						case <-credits:
						case <-done:
							return
						}
					}
					if err := conn.WriteMessage(req.mt, req.data); err != nil {
						return
					}
				case <-done:
					return
				}
			}
		}(streamDone)

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
			select {
			case <-readyForAudio:
			case <-ctx.Done():
				return
			case <-streamDone:
				return
			}
			if !send(websocket.BinaryMessage, first.Audio, true) {
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
					if !send(websocket.BinaryMessage, ch.Audio, true) {
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
		segmentOrdinal := 0
		var processedBatches int64
		grantProcessedBatches := func(processed int64) {
			for processedBatches < processed {
				select {
				case credits <- struct{}{}:
					processedBatches++
				default:
					return
				}
			}
		}
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
				Type             string `json:"type"`
				Text             string `json:"text"`
				StartMs          int64  `json:"start_ms"`
				EndMs            int64  `json:"end_ms"`
				Message          string `json:"message"`
				Code             string `json:"code"`
				Position         int32  `json:"position"`
				ProcessedBatches int64  `json:"processed_batches"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case "processed":
				grantProcessedBatches(msg.ProcessedBatches)
			case "queued":
				events <- StreamEvent{Kind: StreamEventSessionStatus, SessionStatus: &SessionStatusEvent{
					SessionID: start.SessionID, Generation: start.Generation, State: "queued", QueuePosition: msg.Position,
					RecoveryGuidance: "Audio is retained while waiting for the local decoder.",
				}}
			case "ready":
				readyOnce.Do(func() { close(readyForAudio) })
				events <- StreamEvent{Kind: StreamEventSessionStatus, SessionStatus: &SessionStatusEvent{
					SessionID: start.SessionID, Generation: start.Generation, State: "ready",
				}}
			case "partial":
				if msg.Text != "" {
					// Partials are progress-only and explicitly droppable. Never let a
					// high-rate native stream fill the bounded downstream channel and
					// stop this reader from servicing resource ping/control frames or
					// receiving the following durable segment/done events.
					select {
					case events <- StreamEvent{Kind: StreamEventPartial, Partial: &PartialEvent{Text: msg.Text}}:
					default:
					}
				}
			case "segment":
				if msg.Text == "" {
					continue
				}
				finalText += " " + msg.Text
				segmentOrdinal++
				events <- StreamEvent{Kind: StreamEventSegment, Segment: &SegmentEvent{
					Text:             msg.Text,
					SegmentID:        fmt.Sprintf("%s:%d:kyutai:%d", start.SessionID, start.Generation, segmentOrdinal),
					Generation:       start.Generation,
					StartMs:          msg.StartMs,
					EndMs:            msg.EndMs,
					StartSample:      first.StartSample + msg.StartMs*16,
					EndSample:        first.StartSample + msg.EndMs*16,
					AlignmentQuality: "approximate",
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
			case "timed_out", "rejected":
				events <- StreamEvent{Kind: StreamEventSessionStatus, SessionStatus: &SessionStatusEvent{
					SessionID: start.SessionID, Generation: start.Generation, State: msg.Type,
					RecoveryGuidance: "Retained audio can be replayed through the configured recovery engine.",
				}}
				events <- StreamEvent{Kind: StreamEventError, Error: fmt.Errorf("kyutai: %s: %s", msg.Code, msg.Message)}
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
