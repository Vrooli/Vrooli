package tts

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"audio-tools/internal/logx"
)

func testDeps() Deps {
	cfg := Config{Backend: "auto", KokoroVoice: "af_heart", KokoroSpeed: 1}
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

func TestServiceUpdateConfigValidatesBackend(t *testing.T) {
	svc := NewService(testDeps())
	backend := "remote"

	_, err := svc.UpdateConfig(context.Background(), ConfigPatch{Backend: &backend})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
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
