package sttchain_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"

	"audio-tools/internal/ai/sttchain"
)

func TestRumbleVLLMBackendUsesDocumentedOpenAIChatAudioContract(t *testing.T) {
	var sawHealth, sawModels, sawCompletion bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			sawHealth = true
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			sawModels = true
			_, _ = io.WriteString(w, `{"data":[{"id":"Reson8/Rumble-2-Flash"}]}`)
		case "/v1/chat/completions":
			sawCompletion = true
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			var body struct {
				Model       string `json:"model"`
				Temperature int    `json:"temperature"`
				ExtraBody   struct {
					Template struct {
						Language string `json:"language"`
					} `json:"chat_template_kwargs"`
				} `json:"extra_body"`
				Messages []struct {
					Role    string `json:"role"`
					Content []struct {
						Type     string `json:"type"`
						AudioURL struct {
							URL string `json:"url"`
						} `json:"audio_url"`
					} `json:"content"`
				} `json:"messages"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "Reson8/Rumble-2-Flash", body.Model)
			require.Zero(t, body.Temperature)
			require.Equal(t, "English", body.ExtraBody.Template.Language)
			require.Len(t, body.Messages, 1)
			require.Equal(t, "user", body.Messages[0].Role)
			require.Len(t, body.Messages[0].Content, 1)
			content := body.Messages[0].Content[0]
			require.Equal(t, "audio_url", content.Type)
			const prefix = "data:audio/wav;base64,"
			require.True(t, strings.HasPrefix(content.AudioURL.URL, prefix))
			payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(content.AudioURL.URL, prefix))
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(string(payload), "RIFF"), "raw PCM must be wrapped as WAV for the documented audio_url endpoint")
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"language English<asr_text>Hello from Rumble."}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	backend := sttchain.NewRumbleVLLMBackend(srv.URL+"/v1", srv.Client())
	require.True(t, backend.IsAvailable(context.Background()))
	provider := sttchain.NewBatchProvider("rumble-stt", backend, nil)
	require.True(t, provider.IsServiceable(context.Background()))
	result, err := provider.Transcribe(context.Background(), sttchain.Request{
		Audio: []byte{1, 0, 2, 0}, Format: "pcm_s16le", Language: "en",
	})
	require.NoError(t, err)
	require.Equal(t, "Hello from Rumble.", result.Text)
	require.Equal(t, "en", result.DetectedLanguage)
	require.Equal(t, sttchain.TierLocal, result.Tier)
	require.Equal(t, "rumble-stt", result.ProviderID)
	require.Equal(t, "Reson8/Rumble-2-Flash", result.ModelID)
	require.GreaterOrEqual(t, result.Latency.Nanoseconds(), int64(0))
	require.Empty(t, result.Words, "Rumble chat output has no declared word timestamps")
	require.Nil(t, result.Confidence, "Rumble chat output has no declared confidence signals")
	require.True(t, sawHealth && sawModels && sawCompletion)
	traits := provider.Traits()
	require.True(t, traits.Batch)
	require.False(t, traits.Stream)
	require.True(t, traits.Supports(sttchain.StrategyVADSegment))
}

func TestRumbleVLLMBackendRequiresConfiguredServedModel(t *testing.T) {
	backend := sttchain.NewRumbleVLLMBackend("", nil)
	require.False(t, backend.IsAvailable(context.Background()))
	_, err := backend.Transcribe(context.Background(), sttchain.Request{Audio: []byte{1}, Format: "wav"})
	require.ErrorContains(t, err, "endpoint is not configured")

	wrongModel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"some-other-model"}]}`)
	}))
	defer wrongModel.Close()
	require.False(t, sttchain.NewRumbleVLLMBackend(wrongModel.URL, wrongModel.Client()).IsAvailable(context.Background()))
}

func TestRumbleVLLMBackendBoundsAndCancelsBatchCalls(t *testing.T) {
	backend := sttchain.NewRumbleVLLMBackend("http://127.0.0.1:1", nil)
	var nilContext context.Context
	require.False(t, backend.IsAvailable(nilContext), "read-only readiness probes must fail closed for a nil context")
	_, err := backend.Transcribe(nilContext, sttchain.Request{Audio: []byte{1}, Format: "wav"})
	require.ErrorContains(t, err, "context is nil")
	backend.MaxAudio = 2
	_, err = backend.Transcribe(context.Background(), sttchain.Request{Audio: []byte{1, 2, 3}, Format: "wav"})
	require.ErrorContains(t, err, "byte limit")
	_, err = backend.Transcribe(context.Background(), sttchain.Request{Audio: []byte{1}, Format: "wav", Language: "xx"})
	require.ErrorContains(t, err, "unsupported language code")
	_, err = backend.Transcribe(context.Background(), sttchain.Request{Audio: []byte{1}, Format: "", Language: "xx"})
	require.ErrorContains(t, err, "unsupported language code", "invalid language must be rejected before audio detection and conversion")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = backend.Transcribe(ctx, sttchain.Request{Audio: []byte{1}, Format: "wav", Language: "en"})
	require.ErrorIs(t, err, context.Canceled)

	var canceledRequestCount atomic.Int64
	canceledServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		canceledRequestCount.Add(1)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"should not be requested"}}]}`)
	}))
	defer canceledServer.Close()
	canceledBackend := sttchain.NewRumbleVLLMBackend(canceledServer.URL, canceledServer.Client())
	preCanceled, cancelBeforeCall := context.WithCancel(context.Background())
	cancelBeforeCall()
	_, err = canceledBackend.Transcribe(preCanceled, sttchain.Request{Audio: []byte{1, 0}, Format: "pcm_s16le", Language: "en"})
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, canceledRequestCount.Load(), "an already-canceled batch must not contact the endpoint")
}

func TestRumbleVLLMBackendCancelsInFlightRequestAndBoundsResponse(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		close(started)
		<-release
	}))
	backend := sttchain.NewRumbleVLLMBackend(srv.URL, srv.Client())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := backend.Transcribe(ctx, sttchain.Request{Audio: []byte{1, 0}, Format: "pcm_s16le", Language: "en"})
		done <- err
	}()
	select {
	case <-started:
	case err := <-done:
		t.Fatalf("request failed before reaching test endpoint: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("request did not reach test endpoint")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	close(release)
	srv.Close()

	largeResponse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"this response is larger than the configured cap"}}]}`)
	}))
	defer largeResponse.Close()
	bounded := sttchain.NewRumbleVLLMBackend(largeResponse.URL, largeResponse.Client())
	bounded.MaxResponse = 8
	_, err := bounded.Transcribe(context.Background(), sttchain.Request{Audio: []byte{1, 0}, Format: "pcm_s16le", Language: "en"})
	require.ErrorContains(t, err, "response exceeds 8 byte limit")
}

func TestRumbleVLLMBackendDoesNotFollowEndpointRedirects(t *testing.T) {
	var redirectedRequests atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirectedRequests.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"data":[{"id":"Reson8/Rumble-2-Flash"}]}`)
	}))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/capture", http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()

	backend := sttchain.NewRumbleVLLMBackend(redirector.URL, nil)
	require.False(t, backend.IsAvailable(context.Background()), "health redirects must not be followed")
	_, err := backend.Transcribe(context.Background(), sttchain.Request{Audio: []byte{1, 0}, Format: "pcm_s16le", Language: "en"})
	require.ErrorContains(t, err, "HTTP 307")
	require.Zero(t, redirectedRequests.Load(), "audio must not be forwarded to a redirect destination")
}

func TestProductUnaryRouteSelectsRumbleOnlyWhenExplicit(t *testing.T) {
	var completions atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"Reson8/Rumble-2-Flash"}]}`)
		case "/v1/chat/completions":
			completions.Add(1)
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"language English<asr_text>Routed by explicit engine."}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	rumble := sttchain.NewBatchProvider("rumble-stt", sttchain.NewRumbleVLLMBackend(srv.URL, srv.Client()), nil)
	chain := sttchain.NewChain(sttchain.Options{
		Local:       sttchain.NewLocalProvider(nil),
		BatchModels: map[string]sttchain.Provider{"rumble-2-flash": rumble},
		EnableLocal: true,
	})
	req := sttchain.Request{Audio: []byte{1, 0}, Format: "pcm_s16le", Language: "en"}

	_, err := chain.Execute(context.Background(), req)
	require.ErrorContains(t, err, "local provider not configured", "empty engine selection must preserve the legacy Whisper local route")
	require.Zero(t, completions.Load(), "default request must not contact Rumble")

	req.Model = "rumble-2-flash"
	result, err := chain.Execute(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "Routed by explicit engine.", result.Text)
	require.Equal(t, "rumble-stt", result.ProviderID)
	require.Equal(t, "Reson8/Rumble-2-Flash", result.ModelID)
	require.EqualValues(t, 1, completions.Load())
}

func TestExplicitRumbleUnaryRouteFailsClosedWhenEndpointUnavailable(t *testing.T) {
	rumble := sttchain.NewBatchProvider("rumble-stt", sttchain.NewRumbleVLLMBackend("", nil), nil)
	chain := sttchain.NewChain(sttchain.Options{
		Local:       sttchain.NewLocalProvider(nil),
		BatchModels: map[string]sttchain.Provider{"rumble-2-flash": rumble},
		EnableLocal: true,
	})
	_, err := chain.Execute(context.Background(), sttchain.Request{Model: "rumble-2-flash", Audio: []byte{1}, Format: "wav"})
	require.ErrorContains(t, err, "not serviceable")
}

func TestBatchProviderRejectsMissingConfiguration(t *testing.T) {
	provider := sttchain.NewBatchProvider("", nil, nil)
	_, err := provider.Transcribe(context.Background(), sttchain.Request{})
	require.ErrorContains(t, err, "not configured")
	require.False(t, provider.IsAvailable(context.Background()))
}

type batchBackendStub struct {
	available   bool
	serviceable bool
	model       string
	result      sttchain.BatchResult
	err         error
	calls       int
	clock       *scheduletest.FakeClock
}

type rumbleBackendWithoutServiceability struct{}

func (*rumbleBackendWithoutServiceability) IsAvailable(context.Context) bool { return true }
func (*rumbleBackendWithoutServiceability) Model() string                    { return "model" }
func (*rumbleBackendWithoutServiceability) Transcribe(context.Context, sttchain.Request) (sttchain.BatchResult, error) {
	return sttchain.BatchResult{}, nil
}

func (b *batchBackendStub) IsAvailable(context.Context) bool   { return b.available }
func (b *batchBackendStub) IsServiceable(context.Context) bool { return b.serviceable }
func (b *batchBackendStub) Model() string                      { return b.model }
func (b *batchBackendStub) Transcribe(_ context.Context, _ sttchain.Request) (sttchain.BatchResult, error) {
	b.calls++
	if b.clock != nil {
		b.clock.Advance(40 * time.Millisecond)
	}
	return b.result, b.err
}

func TestBatchProviderReportsLocalIdentityAndCapabilities(t *testing.T) {
	backend := &batchBackendStub{available: true, serviceable: true, model: "test/model"}
	provider := sttchain.NewBatchProvider(" test-provider ", backend, nil)
	require.Equal(t, sttchain.TierLocal, provider.Type())
	require.Equal(t, "test/model", provider.Model())
	require.True(t, provider.IsAvailable(context.Background()))
	require.True(t, provider.IsServiceable(context.Background()))
	backend.available = false
	backend.serviceable = false
	require.False(t, provider.IsAvailable(context.Background()))
	require.False(t, provider.IsServiceable(context.Background()))
	noObserver := sttchain.NewBatchProvider("other", &rumbleBackendWithoutServiceability{}, nil)
	require.False(t, noObserver.IsServiceable(context.Background()), "read-only health fails closed when a backend has no serviceability contract")

	traits := provider.Traits()
	require.True(t, traits.Batch)
	require.False(t, traits.Stream)
	require.False(t, traits.BackendAcknowledgements)
	require.ElementsMatch(t, []sttchain.StrategyKind{
		sttchain.StrategyVADSegment,
		sttchain.StrategyOverlapAgree,
		sttchain.StrategyBuffered,
	}, traits.Strategies)
	require.False(t, traits.Supports(sttchain.StrategyPassthrough))
	stream, err := provider.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, nil)
	require.NoError(t, err)
	require.Nil(t, stream, "batch-only provider must decline native streaming")
}

func TestBatchProviderMapsBackendResultAndPreservesTiming(t *testing.T) {
	clock := scheduletest.New(time.Unix(1000, 0))
	backend := &batchBackendStub{
		available: true,
		model:     "vendor/model-revision",
		result: sttchain.BatchResult{
			Text: "transcribed words", DetectedLanguage: "fr",
			Words:      []sttchain.TimedWord{{Word: "transcribed", Start: 0.1, End: 0.7}},
			Confidence: &sttchain.Confidence{NoSpeechProb: 0.02, AvgLogProb: -0.1},
		},
		clock: clock,
	}
	provider := sttchain.NewBatchProvider("batch-provider", backend, clock)
	result, err := provider.Transcribe(context.Background(), sttchain.Request{Language: "en"})
	require.NoError(t, err)
	require.Equal(t, "transcribed words", result.Text)
	require.Equal(t, "fr", result.DetectedLanguage, "backend language detection takes precedence over the request hint")
	require.Equal(t, sttchain.TierLocal, result.Tier)
	require.Equal(t, "batch-provider", result.ProviderID)
	require.Equal(t, "vendor/model-revision", result.ModelID)
	require.Equal(t, 40*time.Millisecond, result.Latency)
	require.Equal(t, backend.result.Words, result.Words)
	require.Equal(t, backend.result.Confidence, result.Confidence)
	backend.result.Words[0].Word = "mutated"
	require.Equal(t, "transcribed", result.Words[0].Word, "word slices must not alias backend-owned buffers")

	backend.result.DetectedLanguage = ""
	backend.result.Words = nil
	fallback, err := provider.Transcribe(context.Background(), sttchain.Request{Language: "de"})
	require.NoError(t, err)
	require.Equal(t, "de", fallback.DetectedLanguage, "request language is retained when backend detection is absent")
	require.Nil(t, fallback.Words)
	require.Equal(t, 2, backend.calls)
}

func TestBatchProviderReturnsBackendErrorsAndRejectsInvalidConfiguration(t *testing.T) {
	wantErr := errors.New("backend failure")
	backend := &batchBackendStub{model: "model", err: wantErr}
	provider := sttchain.NewBatchProvider("provider", backend, nil)
	_, err := provider.Transcribe(context.Background(), sttchain.Request{})
	require.ErrorIs(t, err, wantErr)

	missingBackend := sttchain.NewBatchProvider("provider", nil, nil)
	_, err = missingBackend.Transcribe(context.Background(), sttchain.Request{})
	require.ErrorContains(t, err, "not configured")
	missingID := sttchain.NewBatchProvider("  ", &batchBackendStub{}, nil)
	_, err = missingID.Transcribe(context.Background(), sttchain.Request{})
	require.ErrorContains(t, err, "not configured")

	var nilProvider *sttchain.BatchProvider
	require.Equal(t, sttchain.TierLocal, nilProvider.Type())
	require.Empty(t, nilProvider.Model())
	require.False(t, nilProvider.IsAvailable(context.Background()))
	require.False(t, nilProvider.IsServiceable(context.Background()))
	_, err = nilProvider.Transcribe(context.Background(), sttchain.Request{})
	require.ErrorContains(t, err, "not configured")
	stream, err := nilProvider.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, nil)
	require.NoError(t, err)
	require.Nil(t, stream)
}
