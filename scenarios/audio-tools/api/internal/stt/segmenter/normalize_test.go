package segmenter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	afmocks "audio-tools/internal/audioformat/mocks"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/stt"
	pipeline "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/stt/segmenter/testaudio"
)

// fakeChecker forces a capability status without probing a real resource.
type fakeChecker struct{ status capabilities.Status }

func (f fakeChecker) Check(context.Context) (capabilities.Status, string) { return f.status, "" }

// whisperStub returns an httptest server that answers the Whisper /asr
// endpoint with a fixed transcript. The caller MUST defer srv.Close()
// before the deferred goleak check so the server goroutines are reaped
// first (t.Cleanup runs after defers, which goleak would flag).
func whisperStub(t *testing.T, text string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
	}))
}

// noKeepAliveClient avoids lingering idle HTTP connections that goleak
// would otherwise flag after the test returns.
func noKeepAliveClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
}

// availableWhisperRegistry builds a registry reporting whisper-stt
// available so the LocalProvider is eligible.
func availableWhisperRegistry() *capabilities.Registry {
	return capabilities.NewRegistry(
		[]capabilities.Def{{ID: "whisper-stt", DependencyKind: capabilities.DependencyResource, DependencySlug: "whisper"}},
		map[string]capabilities.Checker{"whisper-stt": fakeChecker{capabilities.StatusAvailable}},
		time.Minute,
	)
}

// localChainWithWhisper wires a Chain whose Local tier transcribes via the
// stub Whisper, sharing one audioformat engine for decode + WAV wrap.
func localChainWithWhisper(t *testing.T, engine *audioformat.Engine, whisperURL string, client *http.Client) (*sttchain.Chain, *stt.Selector) {
	t.Helper()
	svc := pipeline.NewService(
		pipeline.Config{}, "", nil, "",
		pipeline.SpeakerConfig{}, "", nil,
		availableWhisperRegistry(),
		&atomic.Int64{},
		whisperURL+"/asr?output=json",
		client,
		engine,
	)
	local := sttchain.NewLocalProvider(svc)
	chain := sttchain.NewChain(sttchain.Options{Local: local, EnableLocal: true})
	return chain, stt.NewSelectorWith(chain, engine)
}

func collect(t *testing.T, events <-chan sttchain.StreamEvent) (segments []string, finalText string, fellBack bool) {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			switch ev.Kind {
			case sttchain.StreamEventSegment:
				if ev.Segment != nil {
					segments = append(segments, ev.Segment.Text)
				}
			case sttchain.StreamEventDone:
				if ev.Done != nil {
					finalText = ev.Done.FinalText
					fellBack = ev.Done.FellBackToUnary
				}
			}
		case <-timeout:
			t.Fatal("timed out collecting events")
			return
		}
	}
}

func feed(audio []byte, n int) chan sttchain.AudioChunk {
	ch := make(chan sttchain.AudioChunk, n+1)
	step := len(audio) / n
	for i := 0; i < n; i++ {
		lo := i * step
		hi := lo + step
		if i == n-1 {
			hi = len(audio)
		}
		ch <- sttchain.AudioChunk{Audio: audio[lo:hi]}
	}
	close(ch)
	return ch
}

// TestSegmenterNormalizesWebMThroughVAD feeds a stream DECLARED as webm
// through the VAD strategy. The fake decode process (identity transform)
// stands in for ffmpeg, so the PCM fixture round-trips; the assertion is
// that a decode process was started (normalization ran) and the strategy
// produced a transcript from the stub Whisper.
func TestSegmenterNormalizesWebMThroughVAD(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws := whisperStub(t, "hello")
	defer ws.Close()
	proc := &afmocks.FakeProcessRunner{} // identity transform
	engine := audioformat.New(audioformat.WithProcessRunner(proc), audioformat.WithFfmpegProbe(func() bool { return true }))
	chain, selector := localChainWithWhisper(t, engine, ws.URL, noKeepAliveClient())

	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})
	events := make(chan sttchain.StreamEvent, 64)
	start := sttchain.StreamStart{InputFormat: "webm"}
	go func() { _ = seg.Run(ctx, start, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), events) }()

	segments, final, _ := collect(t, events)
	require.Len(t, proc.Calls, 1, "a decode process must be started for webm input")
	require.NotEmpty(t, segments, "expected at least one transcribed segment")
	require.Equal(t, "hello", final)
}

// TestSegmenterPCMFastPathNoDecodeProcess declares PCM, so no ffmpeg
// process is started — the browser fast-path.
func TestSegmenterPCMFastPathNoDecodeProcess(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws := whisperStub(t, "hello")
	defer ws.Close()
	proc := &afmocks.FakeProcessRunner{}
	engine := audioformat.New(audioformat.WithProcessRunner(proc), audioformat.WithFfmpegProbe(func() bool { return true }))
	chain, selector := localChainWithWhisper(t, engine, ws.URL, noKeepAliveClient())

	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})
	events := make(chan sttchain.StreamEvent, 64)
	start := sttchain.StreamStart{InputFormat: "pcm_s16le"}
	go func() { _ = seg.Run(ctx, start, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), events) }()

	_, final, _ := collect(t, events)
	require.Empty(t, proc.Calls, "PCM fast-path must not start a decode process")
	require.Equal(t, "hello", final)
}

// TestSegmenterNoFfmpegFallsBackToBuffered confirms the selector gate: a
// non-PCM stream with no ffmpeg runs BufferedFallback (whole file →
// Whisper) and starts no decode process.
func TestSegmenterNoFfmpegFallsBackToBuffered(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws := whisperStub(t, "hello")
	defer ws.Close()
	proc := &afmocks.FakeProcessRunner{}
	engine := audioformat.New(audioformat.WithProcessRunner(proc), audioformat.WithFfmpegProbe(func() bool { return false }))
	chain, selector := localChainWithWhisper(t, engine, ws.URL, noKeepAliveClient())

	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})
	events := make(chan sttchain.StreamEvent, 64)
	start := sttchain.StreamStart{InputFormat: "webm"}
	go func() { _ = seg.Run(ctx, start, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), events) }()

	_, final, fellBack := collect(t, events)
	require.Empty(t, proc.Calls, "no decode process when ffmpeg is absent")
	require.Equal(t, "hello", final)
	require.True(t, fellBack, "buffered fallback must mark FellBackToUnary")
}
