package tts

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
	"time"
)

func testDeps() Deps {
	cfg := Config{Backend: "auto", KokoroVoice: "af_heart", KokoroSpeed: 1}
	sum := SummarizeConfig{Enabled: true, CharThreshold: 1200, Level: "light", Model: "test-model", TimeoutSeconds: 30}
	cache := map[CacheKey]SynthesizeResult{}
	return Deps{
		GetConfig: func() Config {
			return cfg
		},
		SetConfig: func(c Config) {
			cfg = c
		},
		GetSummarizeConfig: func() SummarizeConfig {
			return sum
		},
		SetSummarizeConfig: func(c SummarizeConfig) {
			sum = c
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
		Logger: log.New(io.Discard, "", 0),
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
