package segmenter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	afmmocks "audio-tools/internal/audioformat/mocks"
	trustfloor "audio-tools/internal/qualification"
	"audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter/testaudio"
)

type offlineFaultCase struct {
	ID       string `json:"id"`
	Test     string `json:"test"`
	Seam     string `json:"seam"`
	Expected string `json:"expected"`
}

type offlineFaultProductPathEvidence struct {
	Rows   []json.RawMessage `json:"rows"`
	Reason string            `json:"reason"`
}

type offlineFaultManifest struct {
	SchemaVersion       int                             `json:"schema_version"`
	Scope               string                          `json:"scope"`
	Promotable          bool                            `json:"promotable"`
	Cases               []offlineFaultCase              `json:"cases"`
	Unsupported         []string                        `json:"unsupported_declared_profiles"`
	ProductPathEvidence offlineFaultProductPathEvidence `json:"product_path_evidence"`
}

// TestOfflineFaultMatrix executes the small server-only slice of the larger
// qualification matrix. The JSON manifest is also consumed by operators when
// reviewing which fault profiles remain browser-only.
func TestOfflineFaultMatrix(t *testing.T) {
	data, err := os.ReadFile("testdata/offline-fault-matrix.json")
	require.NoError(t, err)
	var manifest offlineFaultManifest
	require.NoError(t, json.Unmarshal(data, &manifest))
	require.Equal(t, 1, manifest.SchemaVersion)
	require.Equal(t, "offline_server_stream_regressions", manifest.Scope)
	require.False(t, manifest.Promotable, "server-only evidence cannot qualify a browser product path")
	require.Empty(t, manifest.ProductPathEvidence.Rows, "server-only evidence cannot carry browser product-path rows")
	require.NotEmpty(t, manifest.ProductPathEvidence.Reason, "the product-path evidence gap must stay explicit")
	unsupportedRequired := make([]string, 0, len(trustfloor.RequiredFaults))
	covered := map[string]bool{"provider_busy": true, "delayed_ready": true, "dropped_connection": true, "backend_restart": true, "slow_consumer": true}
	for _, profile := range trustfloor.RequiredFaults {
		if !covered[profile] {
			unsupportedRequired = append(unsupportedRequired, profile)
		}
	}
	require.ElementsMatch(t, unsupportedRequired, manifest.Unsupported,
		"every declared browser fault without a server-level runner must remain visible as a gap")

	cases := map[string]func(*testing.T){
		"TestOfflineFaultProviderTimeoutIsBounded":                      TestOfflineFaultProviderTimeoutIsBounded,
		"TestOfflineFaultProviderBusyIsTerminal":                        TestOfflineFaultProviderBusyIsTerminal,
		"TestOfflineFaultDelayedReadyRecovers":                          TestOfflineFaultDelayedReadyRecovers,
		"TestOfflineFaultDroppedConnectionIsBounded":                    TestOfflineFaultDroppedConnectionIsBounded,
		"TestOfflineFaultBackendRestartRecovers":                        TestOfflineFaultBackendRestartRecovers,
		"TestOfflineFaultSlowConsumerCancellationIsBounded":             TestOfflineFaultSlowConsumerCancellationIsBounded,
		"TestOfflineFaultProviderCancellationDoesNotEmitLateTranscript": TestOfflineFaultProviderCancellationDoesNotEmitLateTranscript,
		"TestNormalizeChunksReportsDecoderFailure":                      TestNormalizeChunksReportsDecoderFailure,
		"TestNormalizeChunksCleanupWaitsForWorkers":                     TestNormalizeChunksCleanupWaitsForWorkers,
		"TestNormalizeChunksDecoderErrorDeliveryIsCancellable":          TestNormalizeChunksDecoderErrorDeliveryIsCancellable,
	}
	seen := map[string]bool{}
	for _, fault := range manifest.Cases {
		fault := fault
		require.NotEmptyf(t, fault.Seam, "matrix case %q must identify its injected seam", fault.ID)
		require.NotEmptyf(t, fault.Expected, "matrix case %q must state its expected behavior", fault.ID)
		run, exists := cases[fault.Test]
		require.Truef(t, exists, "matrix case %q references missing test %q", fault.ID, fault.Test)
		require.Falsef(t, seen[fault.ID], "duplicate fault profile %q", fault.ID)
		seen[fault.ID] = true
		t.Run(fault.ID, run)
	}
	require.ElementsMatch(t, []string{"provider_timeout", "provider_busy", "delayed_ready", "dropped_connection", "backend_restart", "slow_consumer", "provider_cancellation", "truncated_stream", "client_cancellation", "decoder_error_delivery_cancellation"}, keys(seen))
}

func TestNormalizeChunksDecoderErrorDeliveryIsCancellable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		emitDecoderFailure(ctx, make(chan sttchain.StreamEvent), errors.New("decoder failed"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("decoder error delivery blocked worker cleanup after cancellation")
	}
}

func TestOfflineFaultDelayedReadyRecovers(t *testing.T) {
	defer goleak.VerifyNone(t, ignoreDependencyGoroutines())
	const readinessDelay = 100 * time.Millisecond
	var requests atomic.Int32
	var startReadiness sync.Once
	firstRequest := make(chan struct{}, 1)
	ready := make(chan struct{})
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			startReadiness.Do(func() { timer = time.AfterFunc(readinessDelay, func() { close(ready) }) })
			firstRequest <- struct{}{}
		}
		select {
		case <-ready:
			_ = json.NewEncoder(w).Encode(map[string]string{"text": "ready transcript"})
		default:
			http.Error(w, "provider not ready", http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	engine := audioformat.New(audioformat.WithProcessRunner(&afmmocks.FakeProcessRunner{}))
	chain, selector := localChainWithWhisper(t, engine, server.URL, client)
	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	earlyEvents := make(chan sttchain.StreamEvent, 64)
	go func() {
		_ = seg.Run(ctx, sttchain.StreamStart{InputFormat: "pcm_s16le"}, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), earlyEvents)
	}()
	select {
	case <-firstRequest:
	case <-time.After(time.Second):
		t.Fatal("provider was not reached before readiness")
	}
	segments, final, _ := collect(t, earlyEvents)
	require.Empty(t, segments, "unavailable provider must not claim a partial transcript")
	require.Empty(t, final, "unavailable provider must not claim a final transcript")

	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("provider did not become ready within the bounded delay")
	}
	recoveryCtx, recoveryCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer recoveryCancel()
	recoveryEvents := make(chan sttchain.StreamEvent, 64)
	go func() {
		_ = seg.Run(recoveryCtx, sttchain.StreamStart{InputFormat: "pcm_s16le"}, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), recoveryEvents)
	}()
	_, recovered, _ := collect(t, recoveryEvents)
	require.Equal(t, "ready transcript", recovered, "a later stream should succeed after provider readiness")
	require.GreaterOrEqual(t, requests.Load(), int32(2), "the recovery stream must make a fresh provider request")
}

func TestOfflineFaultSlowConsumerCancellationIsBounded(t *testing.T) {
	defer goleak.VerifyNone(t, ignoreDependencyGoroutines())
	requested := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requested <- struct{}{}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "consumer stalled"})
	}))
	defer server.Close()
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	engine := audioformat.New(audioformat.WithProcessRunner(&afmmocks.FakeProcessRunner{}))
	chain, selector := localChainWithWhisper(t, engine, server.URL, client)
	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan sttchain.StreamEvent) // deliberately no consumer
	done := make(chan error, 1)
	cfg := stt.Defaults()
	cfg.Mode = stt.ModeOff
	go func() {
		done <- seg.Run(ctx, sttchain.StreamStart{InputFormat: "pcm_s16le"}, cfg, feed(testaudio.SpeechTonePauseTone3s(), 3), events)
	}()
	select {
	case <-requested:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("provider was not reached")
	}
	// Let the provider response reach the output path while nobody receives it.
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream did not exit after a stalled consumer was cancelled")
	}
}

func TestOfflineFaultProviderCancellationDoesNotEmitLateTranscript(t *testing.T) {
	defer goleak.VerifyNone(t, ignoreDependencyGoroutines())
	requested := make(chan struct{}, 1)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested <- struct{}{}
		select {
		case <-release:
			_, _ = w.Write([]byte(`{"text":"late result"}`))
		case <-r.Context().Done():
			// Simulate a provider that does not promptly stop its underlying work.
			<-release
			_, _ = w.Write([]byte(`{"text":"late result"}`))
		}
	}))
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	engine := audioformat.New(audioformat.WithProcessRunner(&afmmocks.FakeProcessRunner{}))
	chain, selector := localChainWithWhisper(t, engine, server.URL, client)
	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan sttchain.StreamEvent, 64)
	done := make(chan error, 1)
	cfg := stt.Defaults()
	cfg.Mode = stt.ModeOff
	go func() {
		done <- seg.Run(ctx, sttchain.StreamStart{InputFormat: "pcm_s16le"}, cfg, feed(testaudio.SpeechTonePauseTone3s(), 3), events)
	}()
	select {
	case <-requested:
	case <-time.After(5 * time.Second):
		cancel()
		close(release)
		t.Fatal("provider request did not start")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("provider request did not stop after session cancellation")
	}
	close(release)
	for ev := range events {
		require.NotEqual(t, sttchain.StreamEventSegment, ev.Kind, "late provider response must not produce a partial transcript")
		if ev.Done != nil {
			require.Empty(t, ev.Done.FinalText, "late provider response must not produce a final transcript")
		}
	}
}

func TestOfflineFaultBackendRestartRecovers(t *testing.T) {
	defer goleak.VerifyNone(t, ignoreDependencyGoroutines())
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "backend restarting", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "recovered"})
	}))
	defer server.Close()
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	engine := audioformat.New(audioformat.WithProcessRunner(&afmmocks.FakeProcessRunner{}))
	chain, selector := localChainWithWhisper(t, engine, server.URL, client)
	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	events := make(chan sttchain.StreamEvent, 64)
	go func() {
		_ = seg.Run(ctx, sttchain.StreamStart{InputFormat: "pcm_s16le"}, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), events)
	}()
	_, _, _ = collect(t, events)
	require.GreaterOrEqual(t, requests.Load(), int32(2), "provider should be retried after its restart response")
	// Partial and final requests may be concurrent, so the one-shot 503 can
	// land on either. A subsequent stream must complete after recovery.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	events2 := make(chan sttchain.StreamEvent, 64)
	go func() {
		_ = seg.Run(ctx2, sttchain.StreamStart{InputFormat: "pcm_s16le"}, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), events2)
	}()
	_, recovered, _ := collect(t, events2)
	require.Equal(t, "recovered", recovered, "a subsequent stream should succeed after the backend restart")
}

// These server-level diagnostics exercise provider failures through the actual
// local STT HTTP seam. They are intentionally not browser/device evidence.
func runOfflineProviderFault(t *testing.T, handler http.Handler) (time.Duration, []string, string) {
	t.Helper()
	defer goleak.VerifyNone(t, ignoreDependencyGoroutines())
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &http.Client{Timeout: 250 * time.Millisecond, Transport: &http.Transport{DisableKeepAlives: true}}
	engine := audioformat.New(audioformat.WithProcessRunner(&afmmocks.FakeProcessRunner{}))
	chain, selector := localChainWithWhisper(t, engine, server.URL, client)
	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	events := make(chan sttchain.StreamEvent, 64)
	started := time.Now()
	go func() {
		_ = seg.Run(ctx, sttchain.StreamStart{InputFormat: "pcm_s16le"}, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), events)
	}()
	segments, final, _ := collect(t, events)
	return time.Since(started), segments, final
}

func TestOfflineFaultProviderBusyIsTerminal(t *testing.T) {
	elapsed, segments, final := runOfflineProviderFault(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "provider busy", http.StatusServiceUnavailable)
	}))
	require.Less(t, elapsed, 500*time.Millisecond, "busy provider response must not stall the stream")
	require.Empty(t, segments)
	require.Empty(t, final, "busy provider response must not claim a transcript")
}

func TestOfflineFaultDroppedConnectionIsBounded(t *testing.T) {
	elapsed, segments, final := runOfflineProviderFault(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, err := hijacker.Hijack()
		if err == nil {
			_ = conn.Close()
		}
	}))
	require.Less(t, elapsed, 500*time.Millisecond, "dropped provider connection must terminate promptly")
	require.Empty(t, segments)
	require.Empty(t, final, "dropped connection must not claim a transcript")
}

func TestOfflineFaultProviderTimeoutIsBounded(t *testing.T) {
	defer goleak.VerifyNone(t, ignoreDependencyGoroutines())
	started := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
			_, _ = w.Write([]byte(`{"text":"late transcript"}`))
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: 40 * time.Millisecond, Transport: &http.Transport{DisableKeepAlives: true}}
	engine := audioformat.New(audioformat.WithProcessRunner(&afmmocks.FakeProcessRunner{}))
	chain, selector := localChainWithWhisper(t, engine, server.URL, client)
	seg := New(Deps{Chain: chain, Selector: selector, Engine: engine})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	events := make(chan sttchain.StreamEvent, 64)
	go func() {
		_ = seg.Run(ctx, sttchain.StreamStart{InputFormat: "pcm_s16le"}, stt.Defaults(), feed(testaudio.SpeechTonePauseTone3s(), 3), events)
	}()
	segments, final, _ := collect(t, events)
	require.Empty(t, final, "timed-out provider must not claim a successful final transcript")
	require.Empty(t, segments, "timed-out provider must not leak a late partial transcript")
	require.Less(t, time.Since(started), 500*time.Millisecond, "client timeout should bound provider wait")
}

func keys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result
}
