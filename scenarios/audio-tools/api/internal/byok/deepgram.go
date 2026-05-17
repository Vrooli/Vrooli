package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/clock"
	"audio-tools/internal/httpc"
)

// DeepgramSTT calls Deepgram's /v1/listen endpoint.
type DeepgramSTT struct {
	Endpoint string
	Doer     httpc.Doer
	Clock    clock.Clock
}

func NewDeepgramSTT() *DeepgramSTT {
	return &DeepgramSTT{
		Endpoint: "https://api.deepgram.com/v1/listen",
		Doer:     httpc.DefaultDoer(),
		Clock:    clock.System{},
	}
}

func (a *DeepgramSTT) ID() string    { return "deepgram" }
func (a *DeepgramSTT) Model() string { return "nova-2" }

func (a *DeepgramSTT) IsAvailable(ctx context.Context, key string) bool { return key != "" }

// StreamingCapability — Deepgram natively supports streaming via WSS.
// The streaming TranscribeStreaming implementation below opens a WS to
// wss://api.deepgram.com/v1/listen and translates each "Results"
// message to a StreamEvent (Partial for is_final=false, Segment for
// is_final=true). A terminal Done event is emitted when the WS closes.
func (a *DeepgramSTT) StreamingCapability() bool { return true }

// deepgramStreamURL builds the wss URL for Deepgram's streaming API.
// Defaults: linear16 PCM @ 16 kHz mono (matches Whisper's preferred
// input shape and the audio-tools internal contract).
func deepgramStreamURL(language string) string {
	u := url.URL{Scheme: "wss", Host: "api.deepgram.com", Path: "/v1/listen"}
	q := u.Query()
	q.Set("model", "nova-2")
	q.Set("smart_format", "true")
	q.Set("encoding", "linear16")
	q.Set("sample_rate", "16000")
	q.Set("channels", "1")
	if language != "" {
		q.Set("language", language)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// TranscribeStreaming opens a WS to Deepgram, forwards inbound chunks
// as binary frames, and translates "Results" JSON messages back to
// StreamEvents. The returned channel is closed when the vendor WS
// closes or ctx fires.
func (a *DeepgramSTT) TranscribeStreaming(ctx context.Context, key string, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	if key == "" {
		return nil, fmt.Errorf("deepgram: missing API key")
	}
	hdr := http.Header{}
	hdr.Set("Authorization", "Token "+key)
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, deepgramStreamURL(start.Language), hdr)
	if err != nil {
		return nil, fmt.Errorf("deepgram: ws dial: %w", err)
	}

	events := make(chan sttchain.StreamEvent, 16)
	// Pump chunks → vendor WS as binary frames.
	go func() {
		defer func() {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"CloseStream"}`))
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case ch, ok := <-chunks:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.BinaryMessage, ch.Audio); err != nil {
					return
				}
			}
		}
	}()
	// Reader: vendor WS → events channel.
	go func() {
		defer close(events)
		defer conn.Close()
		var finalText string
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{
					FinalText: finalText, ProviderID: "deepgram", ModelID: "nova-2",
				}}
				return
			}
			var msg struct {
				Type    string `json:"type"`
				IsFinal bool   `json:"is_final"`
				Channel struct {
					Alternatives []struct {
						Transcript string  `json:"transcript"`
						Confidence float64 `json:"confidence"`
					} `json:"alternatives"`
				} `json:"channel"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			if msg.Type != "Results" || len(msg.Channel.Alternatives) == 0 {
				continue
			}
			text := msg.Channel.Alternatives[0].Transcript
			if text == "" {
				continue
			}
			if msg.IsFinal {
				finalText += " " + text
				events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
					Text:         text,
					ProviderTier: sttchain.TierBYOK,
					ProviderID:   "deepgram",
					ModelID:      "nova-2",
				}}
			} else {
				events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: text}}
			}
		}
	}()
	return events, nil
}

func (a *DeepgramSTT) Transcribe(ctx context.Context, key string, req sttchain.Request) (*sttchain.Result, error) {
	if key == "" {
		return nil, fmt.Errorf("deepgram: missing API key")
	}
	u, err := url.Parse(a.Endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("model", "nova-2")
	q.Set("smart_format", "true")
	if req.Language != "" {
		q.Set("language", req.Language)
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(req.Audio))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Token "+key)
	httpReq.Header.Set("Content-Type", contentTypeFor(req.Format))

	clk := a.Clock
	if clk == nil {
		clk = clock.System{}
	}
	start := clk.Now()
	resp, err := a.Doer.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("deepgram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepgram: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 256))
	}

	var out struct {
		Results struct {
			Channels []struct {
				Alternatives []struct {
					Transcript string  `json:"transcript"`
					Confidence float64 `json:"confidence"`
				} `json:"alternatives"`
			} `json:"channels"`
		} `json:"results"`
		Metadata struct {
			Duration float64 `json:"duration"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("deepgram: decode response: %w", err)
	}
	text := ""
	if len(out.Results.Channels) > 0 && len(out.Results.Channels[0].Alternatives) > 0 {
		text = out.Results.Channels[0].Alternatives[0].Transcript
	}
	return &sttchain.Result{
		Text:             text,
		DetectedLanguage: req.Language,
		DurationSeconds:  out.Metadata.Duration,
		ModelID:          "nova-2",
		Latency:          clk.Now().Sub(start),
	}, nil
}

func contentTypeFor(format string) string {
	switch format {
	case "wav":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "ogg":
		return "audio/ogg"
	case "webm":
		return "audio/webm"
	default:
		return "application/octet-stream"
	}
}
