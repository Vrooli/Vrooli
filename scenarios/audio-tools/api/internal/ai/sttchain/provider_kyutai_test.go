package sttchain_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/testutil/vendorws"
)

// wsURL rewrites an httptest http:// URL to ws:// so the gorilla dialer accepts it.
func wsURL(httpURL string) string {
	return "ws://" + strings.TrimPrefix(httpURL, "http://")
}

func TestKyutaiProvider_TranslatesEventStream(t *testing.T) {
	srv := vendorws.NewKyutaiServer(vendorws.Options{
		Script: []vendorws.Frame{
			{Text: vendorws.EncodeJSON(map[string]any{"type": "partial", "text": "hel"})},
			{Text: vendorws.EncodeJSON(map[string]any{"type": "segment", "text": "hello", "start_ms": 0, "end_ms": 500})},
			{Text: vendorws.EncodeJSON(map[string]any{"type": "segment", "text": "world", "start_ms": 500, "end_ms": 900})},
			{Text: vendorws.EncodeJSON(map[string]any{"type": "done"})},
		},
	})
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x01, 0x02, 0x03, 0x04}}
	close(chunks)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "en"}, chunks)
	require.NoError(t, err)

	var partials, segments []string
	var done *sttchain.DoneEvent
	for ev := range events {
		switch ev.Kind {
		case sttchain.StreamEventPartial:
			partials = append(partials, ev.Partial.Text)
		case sttchain.StreamEventSegment:
			segments = append(segments, ev.Segment.Text)
			require.Equal(t, sttchain.TierLocal, ev.Segment.ProviderTier)
			require.Equal(t, "kyutai", ev.Segment.ProviderID)
		case sttchain.StreamEventDone:
			done = ev.Done
		}
	}
	require.Equal(t, []string{"hel"}, partials)
	require.Equal(t, []string{"hello", "world"}, segments)
	require.NotNil(t, done)
	require.Equal(t, "hello world", done.FinalText)
	require.Equal(t, sttchain.TierLocal, done.LockedTier)
}

func TestKyutaiProvider_SendsStartHeaderThenPCM(t *testing.T) {
	var mu sync.Mutex
	var gotStart string
	var binaryFrames int
	var gotEnd bool
	srv := vendorws.NewKyutaiServer(vendorws.Options{
		Script:        []vendorws.Frame{{Text: vendorws.EncodeJSON(map[string]any{"type": "done"})}},
		WaitForFrames: 4, // start header + 2 PCM frames + end marker
		OnMessage: func(mt int, payload []byte) {
			mu.Lock()
			defer mu.Unlock()
			switch {
			case strings.Contains(string(payload), `"start"`):
				gotStart = string(payload)
			case strings.Contains(string(payload), `"end"`):
				gotEnd = true
			default:
				binaryFrames++
			}
		},
	})
	defer srv.Close()

	p := sttchain.NewKyutaiProvider("http://example.invalid")
	p.StreamEndpoint = wsURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk, 2)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x10, 0x20}}
	chunks <- sttchain.AudioChunk{Audio: []byte{0x30, 0x40}}
	close(chunks)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, err := p.TranscribeStreaming(ctx, sttchain.StreamStart{Language: "fr", SampleRate: 16000}, chunks)
	require.NoError(t, err)
	for range events { //nolint:revive // drain
	}

	mu.Lock()
	defer mu.Unlock()
	require.Contains(t, gotStart, `"sample_rate":16000`)
	require.Contains(t, gotStart, `"language":"fr"`)
	require.Equal(t, 2, binaryFrames, "both PCM chunks forwarded as binary frames")
	require.True(t, gotEnd, "end marker sent after chunks close")
}

func TestKyutaiProvider_TraitsAndBatch(t *testing.T) {
	p := sttchain.NewKyutaiProvider("http://example.invalid")
	tr := p.Traits()
	require.True(t, tr.Stream)
	require.False(t, tr.Batch)
	require.Equal(t, []sttchain.StrategyKind{sttchain.StrategyPassthrough}, tr.Strategies)
	require.Equal(t, sttchain.TierLocal, p.Type())

	_, err := p.Transcribe(context.Background(), sttchain.Request{})
	require.Error(t, err, "kyutai must refuse unary/batch transcription")
}
