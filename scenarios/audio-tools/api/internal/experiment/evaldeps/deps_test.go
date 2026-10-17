package evaldeps

import (
	"context"
	"strings"
	"testing"

	"audio-tools/internal/ai/sttchain"
	sttpkg "audio-tools/internal/stt"
)

func TestNewProvidesExplicitUnavailableCorpusAndSpeakerDefaults(t *testing.T) {
	deps := New(nil, nil, func(string) sttchain.Provider { return nil }, sttpkg.Defaults(), nil)
	if deps.NewProvider() != nil || deps.NewProviderForEngine("whisper-local") != nil {
		t.Fatal("provider factory should preserve its configured unavailable result")
	}
	if deps.NewSpeakerIsolation() != nil || deps.NewSpeakerExtraction() != nil {
		t.Fatal("default speaker adapters must remain disabled")
	}
	if _, err := deps.LoadClips(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("LoadClips without corpus = %v; want configuration error", err)
	}
}
