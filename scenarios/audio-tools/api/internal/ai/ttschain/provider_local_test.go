package ttschain_test

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/logx"
	"audio-tools/internal/tts"
)

func newLocalTTSService(t *testing.T, capability string, synthesizeErr error) (*tts.Service, *tts.SynthesizeInput) {
	t.Helper()
	cfg := tts.Config{DefaultResponseFormat: "mp3", KokoroVoice: "af_heart", KokoroSpeed: 1}
	seen := &tts.SynthesizeInput{}
	return tts.NewService(tts.Deps{
		GetConfig: func() tts.Config { return cfg },
		SetConfig: func(tts.Config) {},
		KokoroCapability: func(context.Context) (string, string) {
			return capability, "probe"
		},
		SynthesizeAudio: func(_ context.Context, in tts.SynthesizeInput) (io.ReadCloser, string, error) {
			*seen = in
			if synthesizeErr != nil {
				return nil, "", synthesizeErr
			}
			return io.NopCloser(strings.NewReader("audio-bytes")), "audio/mpeg", nil
		},
		Logger: logx.Std{L: log.New(io.Discard, "", 0)},
	}), seen
}

func TestLocalProvider_SynthesizeUsesServiceAndMapsVoice(t *testing.T) {
	svc, seen := newLocalTTSService(t, "available", nil)
	p := ttschain.NewLocalProvider(svc)

	require.True(t, p.IsAvailable(context.Background()))
	out, err := p.Synthesize(context.Background(), ttschain.Request{
		Text: "hello", Voice: "voice.feminine.warm", ResponseFormat: "mp3", Speed: 1.25,
	})
	require.NoError(t, err)
	require.Equal(t, "audio-bytes", string(out.Audio))
	require.Equal(t, "audio/mpeg", out.ContentType)
	require.Equal(t, ttschain.TierLocal, out.Tier)
	require.Equal(t, "kokoro-local", out.ProviderID)
	require.Equal(t, "kokoro-82m-v1.0", out.ModelID)
	require.Equal(t, "af_bella", out.VoiceUsed)
	require.Equal(t, "af_bella", seen.Voice)
	require.Equal(t, 1.25, seen.Speed)
}

func TestLocalProvider_UsesConfiguredDefaultVoice(t *testing.T) {
	svc, seen := newLocalTTSService(t, "available", nil)
	p := ttschain.NewLocalProvider(svc)

	_, err := p.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.NoError(t, err)
	require.Equal(t, "af_heart", seen.Voice)
}

func TestLocalProvider_VoiceOverrideWins(t *testing.T) {
	svc, seen := newLocalTTSService(t, "available", nil)
	p := ttschain.NewLocalProvider(svc)

	_, err := p.Synthesize(context.Background(), ttschain.Request{
		Text:           "hello",
		Voice:          "voice.feminine.warm",
		VoiceOverrides: map[string]string{"local:kokoro-local": "am_override"},
	})
	require.NoError(t, err)
	require.Equal(t, "am_override", seen.Voice)
}

func TestLocalProviderLegacyKokoroOverrideIsKokoroOnly(t *testing.T) {
	legacyOverrides := map[string]string{tts.LegacyAdapterLocalKokoro: "legacy-voice"}
	request := ttschain.Request{Text: "hello", Voice: "voice.feminine.warm", VoiceOverrides: legacyOverrides}

	kaleSvc, kaleSeen := newLocalTTSService(t, "available", nil)
	kale := ttschain.NewConfiguredLocalProvider(kaleSvc, nil, "kale-tts", "kale-tts-1-nano", tts.AdapterLocalKale, tts.LocalKaleMapping)
	kaleOut, err := kale.Synthesize(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "kenzie", kaleSeen.Voice, "Kale must use its own mapping, not the legacy Kokoro override")
	require.Equal(t, "kenzie", kaleOut.VoiceUsed)

	kokoroSvc, kokoroSeen := newLocalTTSService(t, "available", nil)
	kokoro := ttschain.NewLocalProvider(kokoroSvc)
	kokoroOut, err := kokoro.Synthesize(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "legacy-voice", kokoroSeen.Voice, "Kokoro must continue honoring stored legacy overrides")
	require.Equal(t, "legacy-voice", kokoroOut.VoiceUsed)
}

func TestLocalProvider_ReportsBackendReadiness(t *testing.T) {
	down, _ := newLocalTTSService(t, "unavailable", nil)
	p := ttschain.NewLocalProvider(down)
	require.False(t, p.IsAvailable(context.Background()))
	_, err := p.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.ErrorIs(t, err, tts.ErrUnavailable)
}

func TestLocalProvider_PropagatesSynthesisFailure(t *testing.T) {
	svc, _ := newLocalTTSService(t, "available", errors.New("kokoro child exited"))
	p := ttschain.NewLocalProvider(svc)
	_, err := p.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.ErrorIs(t, err, tts.ErrInternal)
}

func TestFailoverLocalProviderDropsToKokoroAndUsesRecoveredKale(t *testing.T) {
	var kaleLive atomic.Bool
	kaleLive.Store(true)
	var kaleSyntheses atomic.Int32
	var kokoroSyntheses atomic.Int32
	var invalidations atomic.Int32
	cfg := tts.Config{DefaultResponseFormat: "mp3", KokoroVoice: "af_heart", KokoroSpeed: 1}
	makeService := func(capability func() bool, synth func(tts.SynthesizeInput) (string, error), invalidate func()) *tts.Service {
		return tts.NewService(tts.Deps{
			GetConfig: func() tts.Config { return cfg }, SetConfig: func(tts.Config) {},
			KokoroCapability: func(context.Context) (string, string) {
				if capability() {
					return "available", "live"
				}
				return "unavailable", "down"
			},
			InvalidateBackendLiveness: invalidate,
			SynthesizeAudio: func(_ context.Context, in tts.SynthesizeInput) (io.ReadCloser, string, error) {
				payload, err := synth(in)
				if err != nil {
					return nil, "", err
				}
				return io.NopCloser(strings.NewReader(payload)), "audio/mpeg", nil
			},
			Logger: logx.Std{L: log.New(io.Discard, "", 0)},
		})
	}
	kaleSvc := makeService(kaleLive.Load, func(in tts.SynthesizeInput) (string, error) {
		kaleSyntheses.Add(1)
		if kaleSyntheses.Load() == 1 {
			return "", tts.ErrUnavailable
		}
		return "kale", nil
	}, func() { invalidations.Add(1) })
	kokoroSvc := makeService(func() bool { return true }, func(in tts.SynthesizeInput) (string, error) {
		kokoroSyntheses.Add(1)
		return "kokoro", nil
	}, func() {})
	kale := ttschain.NewConfiguredLocalProvider(kaleSvc, nil, "kale-tts", "kale-tts-1-nano", tts.AdapterLocalKale, tts.LocalKaleMapping)
	kale.SetCacheBackend("kale-tts-1-nano")
	kokoro := ttschain.NewConfiguredLocalProvider(kokoroSvc, nil, "kokoro-local", "kokoro-82m-v1.0", tts.AdapterLocalKokoro, tts.LocalKokoroMapping)
	kokoro.SetCacheBackend("kokoro")
	provider := ttschain.NewFailoverLocalProvider(kale, kokoro)
	req := ttschain.Request{Text: "hello", Voice: "voice.feminine.warm", ResponseFormat: "mp3", Speed: 1}

	first, err := provider.Synthesize(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "kokoro", string(first.Audio))
	require.Equal(t, "kokoro-local", first.ProviderID)
	require.Equal(t, "kokoro-82m-v1.0", first.ModelID)
	require.Equal(t, "af_bella", first.VoiceUsed)
	require.Equal(t, "kokoro", first.CacheBackend)
	require.EqualValues(t, 1, invalidations.Load())

	// The service can recover after a transient drop; the next request again
	// prefers Kale and reports Kale's actual voice/model/cache identity.
	kaleLive.Store(true)
	second, err := provider.Synthesize(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "kale", string(second.Audio))
	require.Equal(t, "kale-tts", second.ProviderID)
	require.Equal(t, "kale-tts-1-nano", second.ModelID)
	require.Equal(t, "kenzie", second.VoiceUsed)
	require.Equal(t, "kale-tts-1-nano", second.CacheBackend)
	require.EqualValues(t, 2, kaleSyntheses.Load())
	require.EqualValues(t, 1, kokoroSyntheses.Load())
}

func TestFailoverLocalProviderDoesNotMaskSynthesisErrors(t *testing.T) {
	kaleSvc, _ := newLocalTTSService(t, "available", errors.New("model rejected synthesis"))
	kokoroSvc, _ := newLocalTTSService(t, "available", nil)
	kale := ttschain.NewConfiguredLocalProvider(kaleSvc, nil, "kale-tts", "kale-tts-1-nano", tts.AdapterLocalKale, tts.LocalKaleMapping)
	kokoro := ttschain.NewConfiguredLocalProvider(kokoroSvc, nil, "kokoro-local", "kokoro", tts.AdapterLocalKokoro, tts.LocalKokoroMapping)
	_, err := ttschain.NewFailoverLocalProvider(kale, kokoro).Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.ErrorIs(t, err, tts.ErrInternal)
}

func TestFailoverStartsKaleLazilyThenStartsKokoroOnFallback(t *testing.T) {
	var kaleReady, kokoroReady atomic.Bool
	var starts atomic.Int32
	var order []string
	var mu sync.Mutex
	makeService := func(ready *atomic.Bool, value string) *tts.Service {
		cfg := tts.Config{KokoroVoice: "af_heart", KokoroSpeed: 1}
		return tts.NewService(tts.Deps{
			GetConfig: func() tts.Config { return cfg }, SetConfig: func(tts.Config) {},
			Logger: logx.Std{L: log.New(io.Discard, "", 0)},
			KokoroCapability: func(context.Context) (string, string) {
				if ready.Load() {
					return "available", "ready"
				}
				return "unavailable", "down"
			},
			SynthesizeAudio: func(context.Context, tts.SynthesizeInput) (io.ReadCloser, string, error) {
				return io.NopCloser(strings.NewReader(value)), "audio/mpeg", nil
			},
		})
	}
	start := func(_ context.Context, resource string) error {
		starts.Add(1)
		mu.Lock()
		order = append(order, resource)
		mu.Unlock()
		if resource == "kokoro" {
			kokoroReady.Store(true)
		}
		return nil // Kale's managed runtime is unavailable after its start attempt.
	}
	kale := ttschain.NewConfiguredLocalProvider(makeService(&kaleReady, "kale"), nil, "kale-tts", "kale-tts-1-nano", tts.AdapterLocalKale, tts.LocalKaleMapping)
	kokoro := ttschain.NewConfiguredLocalProvider(makeService(&kokoroReady, "kokoro"), nil, "kokoro-local", "kokoro", tts.AdapterLocalKokoro, tts.LocalKokoroMapping)
	ensurer := ttschain.NewBackendEnsurerForTest(start, "kale-tts", "kokoro")
	kale.SetBackendEnsurer(ensurer, "kale-tts")
	kokoro.SetBackendEnsurer(ensurer, "kokoro")
	provider := ttschain.NewFailoverLocalProvider(kale, kokoro)
	require.EqualValues(t, 0, starts.Load(), "composition must not start either runtime")
	out, err := provider.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.NoError(t, err)
	require.Equal(t, "kokoro-local", out.ProviderID)
	require.EqualValues(t, 2, starts.Load())
	require.Equal(t, []string{"kale-tts", "kokoro"}, order)
}

func TestLocalProviderPropagatesCanceledLazyStart(t *testing.T) {
	svc, _ := newLocalTTSService(t, "unavailable", nil)
	p := ttschain.NewConfiguredLocalProvider(svc, nil, "kale-tts", "kale-tts-1-nano", tts.AdapterLocalKale, tts.LocalKaleMapping)
	p.SetBackendEnsurer(ttschain.NewBackendEnsurerForTest(func(ctx context.Context, _ string) error {
		return ctx.Err()
	}, "kale-tts"), "kale-tts")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := p.Synthesize(ctx, ttschain.Request{Text: "hello"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestConcurrentSynthesisRequestsShareOneLazyStart(t *testing.T) {
	var ready atomic.Bool
	var starts, synths atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	cfg := tts.Config{KokoroVoice: "af_heart", KokoroSpeed: 1}
	svc := tts.NewService(tts.Deps{
		GetConfig: func() tts.Config { return cfg }, SetConfig: func(tts.Config) {},
		Logger: logx.Std{L: log.New(io.Discard, "", 0)},
		KokoroCapability: func(context.Context) (string, string) {
			if ready.Load() {
				return "available", "ready"
			}
			return "unavailable", "down"
		},
		SynthesizeAudio: func(context.Context, tts.SynthesizeInput) (io.ReadCloser, string, error) {
			synths.Add(1)
			return io.NopCloser(strings.NewReader("audio")), "audio/mpeg", nil
		},
	})
	p := ttschain.NewConfiguredLocalProvider(svc, nil, "kale-tts", "kale-tts-1-nano", tts.AdapterLocalKale, tts.LocalKaleMapping)
	p.SetBackendEnsurer(ttschain.NewBackendEnsurerForTest(func(ctx context.Context, resource string) error {
		if resource != "kale-tts" {
			t.Errorf("resource = %q", resource)
		}
		if starts.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
			ready.Store(true)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, "kale-tts"), "kale-tts")
	chain := ttschain.NewChain(ttschain.Options{Local: p, LocalFirst: true, EnableLocal: true})
	require.False(t, p.IsAvailable(context.Background()), "health surface must remain truthful while stopped")
	const n = 16
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := chain.Execute(context.Background(), ttschain.Request{Text: "hello"}); err != nil {
				t.Errorf("synthesize: %v", err)
			}
		}()
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("resource start did not begin")
	}
	close(release)
	wg.Wait()
	require.EqualValues(t, 1, starts.Load())
	require.EqualValues(t, n, synths.Load())
}

func TestFailoverLocalProviderHandlesMissingBackendServices(t *testing.T) {
	fallbackSvc, _ := newLocalTTSService(t, "available", nil)
	fallback := ttschain.NewConfiguredLocalProvider(fallbackSvc, nil, "kokoro-local", "kokoro", tts.AdapterLocalKokoro, tts.LocalKokoroMapping)
	preferred := ttschain.NewConfiguredLocalProvider(nil, nil, "kale-tts", "kale-tts-1-nano", tts.AdapterLocalKale, tts.LocalKaleMapping)
	preferred.SetBackendEnsurer(ttschain.NewBackendEnsurerForTest(func(context.Context, string) error {
		t.Fatal("provider without a service must not attempt startup")
		return nil
	}, "kale-tts"), "kale-tts")
	provider := ttschain.NewFailoverLocalProvider(preferred, fallback)

	require.True(t, provider.IsAvailable(context.Background()))
	out, err := provider.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.NoError(t, err)
	require.Equal(t, "kokoro-local", out.ProviderID)

	missingFallback := ttschain.NewFailoverLocalProvider(preferred, &ttschain.LocalProvider{})
	require.False(t, missingFallback.IsAvailable(context.Background()))
	_, err = missingFallback.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.Error(t, err)
}

func TestFailoverLocalProviderTraversesThreeEntriesWithProviderIdentity(t *testing.T) {
	var ready [3]atomic.Bool
	var starts []string
	var mu sync.Mutex
	makeService := func(i int) *tts.Service {
		cfg := tts.Config{KokoroVoice: "fallback-default", KokoroSpeed: 1}
		return tts.NewService(tts.Deps{
			GetConfig: func() tts.Config { return cfg }, SetConfig: func(tts.Config) {},
			Logger: logx.Std{L: log.New(io.Discard, "", 0)},
			KokoroCapability: func(context.Context) (string, string) {
				if ready[i].Load() {
					return "available", "ready"
				}
				return "unavailable", "down"
			},
			SynthesizeAudio: func(_ context.Context, in tts.SynthesizeInput) (io.ReadCloser, string, error) {
				return io.NopCloser(strings.NewReader(in.Voice)), "audio/mpeg", nil
			},
		})
	}
	providers := make([]*ttschain.LocalProvider, 3)
	ids := []string{"first", "second", "third"}
	models := []string{"model-first", "model-second", "model-third"}
	voices := []string{"voice-first", "voice-second", "voice-third"}
	for i := range providers {
		index := i
		providers[i] = ttschain.NewConfiguredLocalProvider(makeService(i), nil, ids[i], models[i], "adapter-"+ids[i], tts.AdapterVoiceMap{"voice.feminine.warm": voices[i]})
		providers[i].SetCacheBackend("cache-" + ids[i])
		providers[i].SetBackendEnsurer(ttschain.NewBackendEnsurerForTest(func(context.Context, string) error {
			mu.Lock()
			starts = append(starts, ids[index])
			mu.Unlock()
			if index == 2 {
				ready[index].Store(true)
			}
			return nil
		}, "kale-tts"), "kale-tts")
	}
	chain := ttschain.NewFailoverLocalProviders(providers...)
	require.False(t, chain.IsAvailable(context.Background()))
	out, err := chain.Synthesize(context.Background(), ttschain.Request{Text: "hello", Voice: "voice.feminine.warm"})
	require.NoError(t, err)
	require.Equal(t, "third", out.ProviderID)
	require.Equal(t, "model-third", out.ModelID)
	require.Equal(t, "voice-third", out.VoiceUsed)
	require.Equal(t, "cache-third", out.CacheBackend)
	require.Equal(t, []string{"first", "second", "third"}, starts)
	require.True(t, chain.IsAvailable(context.Background()), "availability must include the terminal provider")
}

func TestLocalProvider_NilReceiverContracts(t *testing.T) {
	var p *ttschain.LocalProvider
	require.Equal(t, ttschain.TierLocal, p.Type())
	require.False(t, p.IsAvailable(context.Background()))
	require.Equal(t, "kokoro", p.Model())
	require.False(t, p.StreamingCapability())
	_, err := p.Synthesize(context.Background(), ttschain.Request{})
	require.Error(t, err)
}

func TestLocalCacheBackendsFollowConfiguredFailover(t *testing.T) {
	primary := ttschain.NewConfiguredLocalProvider(nil, nil, "primary", "primary-model", "primary", nil)
	primary.SetCacheBackend("primary-cache")
	fallback := ttschain.NewConfiguredLocalProvider(nil, nil, "fallback", "fallback-model", "fallback", nil)
	fallback.SetCacheBackend("fallback-cache")
	chain := ttschain.NewChain(ttschain.Options{Local: ttschain.NewFailoverLocalProvider(primary, fallback)})
	require.Equal(t, []string{"primary-cache", "fallback-cache"}, chain.LocalCacheBackends())
}

func TestLocalCacheBackendsUseModelDefaultAndDeduplicate(t *testing.T) {
	primary := ttschain.NewConfiguredLocalProvider(nil, nil, "primary", "same-cache", "primary", nil)
	fallback := ttschain.NewConfiguredLocalProvider(nil, nil, "fallback", "same-cache", "fallback", nil)
	chain := ttschain.NewChain(ttschain.Options{Local: ttschain.NewFailoverLocalProvider(primary, fallback)})
	require.Equal(t, []string{"same-cache"}, chain.LocalCacheBackends())
}

func TestLocalCacheBackendsHandleAbsentChainAndIdentity(t *testing.T) {
	var nilChain *ttschain.Chain
	require.Nil(t, nilChain.LocalCacheBackends())
	require.Nil(t, (&ttschain.Chain{}).LocalCacheBackends())
	chain := ttschain.NewChain(ttschain.Options{Local: &ttschain.LocalProvider{}})
	require.Empty(t, chain.LocalCacheBackends())
}
