package tts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"audio-tools/internal/logx"
)

func testDeps() Deps {
	cfg := Config{DefaultResponseFormat: "mp3", KokoroVoice: "af_heart", KokoroSpeed: 1}
	cache := map[CacheKey]SynthesizeResult{}
	return Deps{
		GetConfig: func() Config {
			return cfg
		},
		SetConfig: func(c Config) {
			cfg = c
		},
		GetHookStatus: func() (bool, string, string, string) {
			return true, "registered", "", "/tmp/settings.json"
		},
		GetLastRouting: func() (*AppendResult, time.Time) {
			return nil, time.Time{}
		},
		GetRoutingBySource: func(string) (*AppendResult, time.Time) {
			return nil, time.Time{}
		},
		GetLastAck: func() (*ClientAck, time.Time) {
			return nil, time.Time{}
		},
		GetAckBySource: func(string) (*ClientAck, time.Time) {
			return nil, time.Time{}
		},
		GetLastPlaybackEvent: func() (*PlaybackEvent, time.Time) {
			return nil, time.Time{}
		},
		RecordPlaybackEvent: func(PlaybackEvent) {},
		KokoroCapability: func(context.Context) (string, string) {
			return "available", "Kokoro available"
		},
		SynthesizeAudio: func(context.Context, SynthesizeInput) (io.ReadCloser, string, error) {
			return io.NopCloser(strings.NewReader("audio")), "audio/mpeg", nil
		},
		GetCache: func(key CacheKey) (SynthesizeResult, bool) {
			out, ok := cache[key]
			return out, ok
		},
		PutCache: func(key CacheKey, audio []byte, contentType string) {
			cache[key] = SynthesizeResult{Audio: audio, ContentType: contentType}
		},
		ListVoiceCatalog: func(context.Context) ([]Voice, error) {
			return []Voice{{ID: "af_heart", Name: "Heart"}}, nil
		},
		Logger: logx.Std{L: log.New(io.Discard, "", 0)},
	}
}

func TestReadBoundedSynthesisResponse(t *testing.T) {
	exact := strings.NewReader("12345678")
	data, err := readBoundedResponse(exact, 8)
	if err != nil || string(data) != "12345678" {
		t.Fatalf("exact-limit response = %q, %v; want full payload", data, err)
	}

	over := strings.NewReader("123456789")
	data, err = readBoundedResponse(over, 8)
	if !errors.Is(err, errResponseBodyTooLarge) {
		t.Fatalf("oversized response error = %v, want %v", err, errResponseBodyTooLarge)
	}
	if data != nil {
		t.Fatalf("oversized response returned %d bytes, want no payload", len(data))
	}
	if over.Len() != 0 {
		t.Fatalf("oversized response reader has %d unread bytes; expected read to stop at limit+1", over.Len())
	}
}

func TestServiceUpdateConfigValidatesResponseFormat(t *testing.T) {
	svc := NewService(testDeps())
	format := "aiff"

	_, err := svc.UpdateConfig(context.Background(), ConfigPatch{DefaultResponseFormat: &format})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestServiceConcurrentConfigUpdatesMergeWithoutRaces(t *testing.T) {
	deps := testDeps()
	var storeMu sync.RWMutex
	cfg := Config{DefaultResponseFormat: "mp3", KokoroVoice: "af_heart", KokoroSpeed: 1}
	deps.GetConfig = func() Config {
		storeMu.RLock()
		defer storeMu.RUnlock()
		return cfg
	}
	deps.SetConfig = func(next Config) {
		storeMu.Lock()
		defer storeMu.Unlock()
		cfg = next
	}
	svc := NewService(deps)
	voice, speed := "Kenzie", 1.5
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, patch := range []ConfigPatch{{KokoroVoice: &voice}, {KokoroSpeed: &speed}} {
		patch := patch
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 500; i++ {
				if _, err := svc.UpdateConfig(context.Background(), patch); err != nil {
					t.Errorf("UpdateConfig: %v", err)
					return
				}
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 500; j++ {
				if _, err := svc.GetConfig(context.Background()); err != nil {
					t.Errorf("GetConfig: %v", err)
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	got, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.KokoroVoice != voice || got.KokoroSpeed != speed {
		t.Fatalf("concurrent partial updates lost a field: voice=%q speed=%v", got.KokoroVoice, got.KokoroSpeed)
	}
}

func TestServiceSynthesizeUsesConfiguredDefaultResponseFormat(t *testing.T) {
	deps := testDeps()
	var seen SynthesizeInput
	deps.GetConfig = func() Config { return Config{DefaultResponseFormat: "wav", KokoroVoice: "af_heart"} }
	deps.SynthesizeAudio = func(_ context.Context, in SynthesizeInput) (io.ReadCloser, string, error) {
		seen = in
		return io.NopCloser(strings.NewReader("audio")), "audio/wav", nil
	}
	if _, err := NewService(deps).Synthesize(context.Background(), SynthesizeInput{Input: "hello"}); err != nil {
		t.Fatal(err)
	}
	if seen.ResponseFormat != "wav" {
		t.Fatalf("response format = %q, want wav", seen.ResponseFormat)
	}
}

func TestServiceSynthesizeNormalizesAndCaches(t *testing.T) {
	deps := testDeps()
	var seen SynthesizeInput
	deps.SynthesizeAudio = func(_ context.Context, in SynthesizeInput) (io.ReadCloser, string, error) {
		seen = in
		return io.NopCloser(strings.NewReader("audio")), "audio/mpeg", nil
	}
	svc := NewService(deps)

	out, err := svc.Synthesize(context.Background(), SynthesizeInput{
		Input:   "  hello  ",
		EventID: "evt-1",
	})
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if string(out.Audio) != "audio" || out.ContentType != "audio/mpeg" {
		t.Fatalf("unexpected synth result: %#v", out)
	}
	if seen.Input != "hello" || seen.Voice != "af_heart" || seen.ResponseFormat != "mp3" || seen.Speed != 1 {
		t.Fatalf("request was not normalized: %#v", seen)
	}

	cached, err := svc.GetCache(context.Background(), CacheLookup{EventID: "evt-1"})
	if err != nil {
		t.Fatalf("cache lookup: %v", err)
	}
	if string(cached.Audio) != "audio" || cached.ContentType != "audio/mpeg" {
		t.Fatalf("unexpected cached result: %#v", cached)
	}
}

func TestServiceGetCacheUsesSynthesisNormalization(t *testing.T) {
	deps := testDeps()
	var observed CacheKey
	deps.GetCache = func(key CacheKey) (SynthesizeResult, bool) {
		observed = key
		return SynthesizeResult{Audio: []byte("cached")}, true
	}
	svc := NewService(deps)

	if _, err := svc.GetCache(context.Background(), CacheLookup{EventID: "evt", Speed: 99, ResponseFormat: "wav"}); err != nil {
		t.Fatalf("GetCache: %v", err)
	}
	if observed.Speed != maxTTSSpeed || observed.ResponseFormat != "wav" {
		t.Fatalf("cache key not normalized like synthesis: speed=%v format=%q", observed.Speed, observed.ResponseFormat)
	}
	if _, err := svc.GetCache(context.Background(), CacheLookup{EventID: "evt", ResponseFormat: "invalid"}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid format error = %v, want ErrInvalidArgument", err)
	}
}

func TestServiceRejectsNonFiniteSynthesisSpeedsBeforeBackendOrCache(t *testing.T) {
	for _, speed := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		t.Run(fmt.Sprint(speed), func(t *testing.T) {
			deps := testDeps()
			backendCalls, cacheCalls := 0, 0
			deps.SynthesizeAudio = func(context.Context, SynthesizeInput) (io.ReadCloser, string, error) {
				backendCalls++
				return io.NopCloser(strings.NewReader("audio")), "audio/mpeg", nil
			}
			deps.GetCache = func(CacheKey) (SynthesizeResult, bool) {
				cacheCalls++
				return SynthesizeResult{Audio: []byte("cached")}, true
			}
			svc := NewService(deps)
			if _, err := svc.Synthesize(context.Background(), SynthesizeInput{Input: "hello", Speed: speed}); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("Synthesize error = %v, want ErrInvalidArgument", err)
			}
			if _, err := svc.GetCache(context.Background(), CacheLookup{EventID: "evt", Speed: speed}); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("GetCache error = %v, want ErrInvalidArgument", err)
			}
			if backendCalls != 0 || cacheCalls != 0 {
				t.Fatalf("non-finite speed reached backend/cache: backend=%d cache=%d", backendCalls, cacheCalls)
			}

			configSpeed := speed
			if _, err := svc.UpdateConfig(context.Background(), ConfigPatch{KokoroSpeed: &configSpeed}); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("UpdateConfig error = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestServiceSynthesizePreservesCancellationAndDoesNotCache(t *testing.T) {
	for _, tc := range []struct {
		name string
		want error
	}{
		{name: "canceled", want: context.Canceled},
		{name: "deadline", want: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps := testDeps()
			cacheWrites := 0
			deps.PutCache = func(CacheKey, []byte, string) { cacheWrites++ }
			deps.SynthesizeAudio = func(ctx context.Context, _ SynthesizeInput) (io.ReadCloser, string, error) {
				return nil, "", tc.want
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.want == context.DeadlineExceeded {
				deadlineCtx, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer deadlineCancel()
				ctx = deadlineCtx
			} else {
				cancel()
			}

			_, err := NewService(deps).Synthesize(ctx, SynthesizeInput{Input: "hello", EventID: "event"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("Synthesize error = %v, want %v", err, tc.want)
			}
			if cacheWrites != 0 {
				t.Fatalf("canceled synthesis wrote cache %d times", cacheWrites)
			}
		})
	}
}

type cancelOnRead struct{ cancel context.CancelFunc }

func (r cancelOnRead) Read([]byte) (int, error) {
	r.cancel()
	return 0, errors.New("request canceled while reading response")
}

func (cancelOnRead) Close() error { return nil }

type cancelAfterBytesRead struct {
	cancel context.CancelFunc
	done   bool
}

func (r *cancelAfterBytesRead) Read(dst []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(dst, "audio")
	r.cancel()
	return n, nil
}

func (*cancelAfterBytesRead) Close() error { return nil }

func TestServiceSynthesizePreservesCancellationWhileReadingAndDoesNotCache(t *testing.T) {
	deps := testDeps()
	cacheWrites := 0
	deps.PutCache = func(CacheKey, []byte, string) { cacheWrites++ }
	ctx, cancel := context.WithCancel(context.Background())
	deps.SynthesizeAudio = func(context.Context, SynthesizeInput) (io.ReadCloser, string, error) {
		return cancelOnRead{cancel: cancel}, "audio/mpeg", nil
	}

	_, err := NewService(deps).Synthesize(ctx, SynthesizeInput{Input: "hello", EventID: "event"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Synthesize error = %v, want context.Canceled", err)
	}
	if cacheWrites != 0 {
		t.Fatalf("canceled response read wrote cache %d times", cacheWrites)
	}
}

func TestServiceSynthesizeDoesNotCacheSuccessfulReadAfterCancellation(t *testing.T) {
	deps := testDeps()
	cacheWrites := 0
	deps.PutCache = func(CacheKey, []byte, string) { cacheWrites++ }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps.SynthesizeAudio = func(context.Context, SynthesizeInput) (io.ReadCloser, string, error) {
		return &cancelAfterBytesRead{cancel: cancel}, "audio/mpeg", nil
	}

	_, err := NewService(deps).Synthesize(ctx, SynthesizeInput{Input: "hello", EventID: "event"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Synthesize error = %v, want context.Canceled", err)
	}
	if cacheWrites != 0 {
		t.Fatalf("canceled synthesis wrote cache %d times", cacheWrites)
	}
}

func TestSynthesizeDoesNotSweepCatalogue(t *testing.T) {
	deps := testDeps()
	var capabilityCalls int
	var synthesisCalls int
	deps.KokoroCapability = func(context.Context) (string, string) {
		capabilityCalls++
		return "available", "Kokoro available"
	}
	deps.SynthesizeAudio = func(context.Context, SynthesizeInput) (io.ReadCloser, string, error) {
		synthesisCalls++
		return io.NopCloser(strings.NewReader("audio")), "audio/mpeg", nil
	}

	if _, err := NewService(deps).Synthesize(context.Background(), SynthesizeInput{Input: "one request"}); err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if capabilityCalls != 1 {
		t.Fatalf("Kokoro capability calls = %d, want one single-provider gate", capabilityCalls)
	}
	if synthesisCalls != 1 {
		t.Fatalf("synthesis calls = %d, want 1", synthesisCalls)
	}
}

func TestServiceSynthesizeFormatValidation(t *testing.T) {
	deps := testDeps()
	deps.SynthesizeAudio = func(_ context.Context, in SynthesizeInput) (io.ReadCloser, string, error) {
		return io.NopCloser(strings.NewReader("audio")), "audio/x", nil
	}
	svc := NewService(deps)

	// Every advertised format is accepted (vocabulary owned by the
	// audioformat substrate — flac included, no drift).
	for _, f := range []string{"mp3", "wav", "opus", "flac"} {
		_, err := svc.Synthesize(context.Background(), SynthesizeInput{Input: "hi", ResponseFormat: f})
		if err != nil {
			t.Fatalf("format %q should be accepted: %v", f, err)
		}
	}

	// An unsupported format is a typed invalid-argument error.
	_, err := svc.Synthesize(context.Background(), SynthesizeInput{Input: "hi", ResponseFormat: "aiff"})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument for bad format, got %v", err)
	}
}

func TestServiceSynthesizePreservesBackendInputRejection(t *testing.T) {
	deps := testDeps()
	deps.SynthesizeAudio = func(context.Context, SynthesizeInput) (io.ReadCloser, string, error) {
		return nil, "", fmt.Errorf("speech backend rejected voice: %w", ErrInvalidArgument)
	}
	_, err := NewService(deps).Synthesize(context.Background(), SynthesizeInput{Input: "hello"})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
	if errors.Is(err, ErrInternal) {
		t.Fatalf("error = %v, must not classify a rejected request as internal", err)
	}
}

func TestServiceListVoicesRequiresAvailableKokoro(t *testing.T) {
	deps := testDeps()
	deps.KokoroCapability = func(context.Context) (string, string) {
		return "unavailable", "Kokoro unavailable"
	}
	svc := NewService(deps)

	_, err := svc.ListVoices(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

// The local TTS tier used to answer "available" from a hardcoded constant, so
// `audio-tools settings providers` reported Kokoro up with its port closed and
// no process running. Readiness is now one predicate that Synthesize and the
// provider chain both consult, so an availability answer cannot disagree with
// what the next request would actually do.
func TestLocalBackendReadyFollowsTheCapabilityProbe(t *testing.T) {
	deps := testDeps()
	capability := "available"
	deps.KokoroCapability = func(context.Context) (string, string) { return capability, "Kokoro (Local)" }
	svc := NewService(deps)

	if !svc.LocalBackendReady(context.Background()) {
		t.Fatal("expected ready while the capability probe reports available")
	}

	capability = "unavailable"
	if svc.LocalBackendReady(context.Background()) {
		t.Fatal("expected not ready once the capability probe reports unavailable")
	}
	if _, err := svc.Synthesize(context.Background(), SynthesizeInput{Input: "hello"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable from Synthesize, got %v", err)
	}
}

func TestLocalBackendNotReadyWithoutSynthesisWiring(t *testing.T) {
	deps := testDeps()
	deps.SynthesizeAudio = nil
	if NewService(deps).LocalBackendReady(context.Background()) {
		t.Fatal("expected not ready when synthesis is not wired")
	}
}
