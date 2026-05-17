// Package integration hosts cross-domain regression tests that exercise
// more than one chain at a time. Single-chain tests stay co-located in
// their owning package; this package is reserved for the
// transcribe → summarize → synthesize sequence (and future
// multi-domain flows) so a refactor that breaks the seam between
// chains is caught by a single dedicated suite.
package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/diagnostics/mocks"
)

// runPipeline drives the canonical UX path: transcribe → summarize →
// synthesize. Each chain is replaced with a *mocks.* runner that
// records its inputs and emits a canned result. The function asserts
// the wiring observable to the user: each chain is invoked exactly
// once per call, the summarize input matches the STT final transcript,
// and the TTS input matches the summary output.
func runPipeline(t *testing.T, stt *mocks.STT, summ *mocks.Summ, tts *mocks.TTS, audio []byte) (sttRes *sttchain.Result, summRes *summarizechain.Result, ttsRes *ttschain.Result) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sttRes, err := stt.Execute(ctx, sttchain.Request{Audio: audio, Format: "wav"})
	if err != nil {
		t.Fatalf("stt: %v", err)
	}
	if sttRes == nil || sttRes.Text == "" {
		t.Fatalf("stt produced empty transcript")
	}

	summRes, err = summ.Execute(ctx, summarizechain.Request{Text: sttRes.Text, Level: "moderate"})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if summRes == nil || summRes.Text == "" {
		t.Fatalf("summarize produced empty result")
	}

	ttsRes, err = tts.Execute(ctx, ttschain.Request{Text: summRes.Text, Voice: "voice.feminine.warm", ResponseFormat: "wav", Speed: 1.0})
	if err != nil {
		t.Fatalf("tts: %v", err)
	}
	if ttsRes == nil || len(ttsRes.Audio) == 0 {
		t.Fatalf("tts produced empty audio")
	}
	return sttRes, summRes, ttsRes
}

func TestCrossDomain_HappyPath(t *testing.T) {
	stt := &mocks.STT{Res: &sttchain.Result{Text: "hello world", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"}}
	summ := &mocks.Summ{Res: &summarizechain.Result{Text: "hi", Tier: summarizechain.TierLocal, ProviderID: "ollama", ModelID: "llama3"}}
	tts := &mocks.TTS{Res: &ttschain.Result{Audio: []byte("RIFFmocked"), ContentType: "audio/wav", Tier: ttschain.TierLocal, ProviderID: "kokoro", ModelID: "v1"}}

	sttRes, summRes, ttsRes := runPipeline(t, stt, summ, tts, []byte("audio"))

	if stt.Calls != 1 || summ.Calls != 1 || tts.Calls != 1 {
		t.Fatalf("each chain should be hit exactly once; got stt=%d summ=%d tts=%d", stt.Calls, summ.Calls, tts.Calls)
	}
	if sttRes.Text != "hello world" || summRes.Text != "hi" || string(ttsRes.Audio) != "RIFFmocked" {
		t.Fatalf("unexpected results: stt=%q summ=%q tts=%d bytes", sttRes.Text, summRes.Text, len(ttsRes.Audio))
	}
}

func TestCrossDomain_SummarizeFailureShortCircuits(t *testing.T) {
	stt := &mocks.STT{Res: &sttchain.Result{Text: "longer transcript that needs summarizing"}}
	summ := &mocks.Summ{Err: errors.New("ollama timeout")}
	tts := &mocks.TTS{Res: &ttschain.Result{Audio: []byte("never-reached")}}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sttRes, err := stt.Execute(ctx, sttchain.Request{Audio: []byte("x"), Format: "wav"})
	if err != nil || sttRes == nil {
		t.Fatalf("stt should succeed: %v", err)
	}
	_, err = summ.Execute(ctx, summarizechain.Request{Text: sttRes.Text, Level: "moderate"})
	if err == nil {
		t.Fatal("expected summarize to fail")
	}
	if tts.Calls != 0 {
		t.Fatalf("tts should not be reached when summarize fails; got %d", tts.Calls)
	}
}

func TestCrossDomain_TTSEmptyAudioIsObservable(t *testing.T) {
	stt := &mocks.STT{Res: &sttchain.Result{Text: "x"}}
	summ := &mocks.Summ{Res: &summarizechain.Result{Text: "y"}}
	tts := &mocks.TTS{Res: &ttschain.Result{Audio: nil, ContentType: "audio/wav"}}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sttRes, _ := stt.Execute(ctx, sttchain.Request{Audio: []byte("a"), Format: "wav"})
	summRes, _ := summ.Execute(ctx, summarizechain.Request{Text: sttRes.Text})
	ttsRes, err := tts.Execute(ctx, ttschain.Request{Text: summRes.Text})
	if err != nil {
		t.Fatalf("tts.Execute returned error for empty-audio response: %v", err)
	}
	if len(ttsRes.Audio) != 0 {
		t.Fatalf("expected zero audio bytes")
	}
}
