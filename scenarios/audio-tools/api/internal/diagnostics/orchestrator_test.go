package diagnostics_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	intaudio "audio-tools/internal/audio"
	"audio-tools/internal/diagnostics"
	"audio-tools/internal/diagnostics/mocks"
	"audio-tools/internal/logx"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	intsumm "audio-tools/internal/summarize"
	"audio-tools/internal/usagereport"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func okSTT() *mocks.STT {
	return &mocks.STT{Res: &sttchain.Result{Text: "hello", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: 12 * time.Millisecond}}
}

func okTTS() *mocks.TTS {
	return &mocks.TTS{Res: &ttschain.Result{Audio: []byte("RIFF"), ContentType: "audio/wav", Tier: ttschain.TierLocal, ProviderID: "kokoro", ModelID: "v1", Latency: 14 * time.Millisecond}}
}

func okSumm() *mocks.Summ {
	return &mocks.Summ{Res: &summarizechain.Result{Text: "tldr", Tier: summarizechain.TierLocal, ProviderID: "ollama", ModelID: "llama3", Latency: 17 * time.Millisecond}}
}

func okTranscode() *mocks.Transcode {
	return &mocks.Transcode{Out: make([]byte, 1024)}
}

func newOrch(s *mocks.STT, ts *mocks.TTS, su *mocks.Summ, tc *mocks.Transcode) *diagnostics.Orchestrator {
	return diagnostics.New(diagnostics.Deps{
		STT:       s,
		TTS:       ts,
		Summarize: su,
		Transcode: tc,
		NewRunID:  func() string { return "run-1" },
	})
}

func TestRunSuite_AllPass(t *testing.T) {
	o := newOrch(okSTT(), okTTS(), okSumm(), okTranscode())
	run, err := o.RunSuite(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if run.Overall != diagnostics.StatusPass {
		t.Fatalf("want PASS, got %s", run.Overall)
	}
	if run.PassCount != 4 || run.FailCount != 0 || run.TotalCount != 4 {
		t.Fatalf("counts wrong: %+v", run)
	}
	for _, s := range run.Steps {
		if !s.OK {
			t.Errorf("step %s should be ok, got %s/%s", s.Capability, s.ErrorCode, s.ErrorMessage)
		}
	}
}

func TestRunSuite_RecordsUsageForMappedCapabilities(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(usagereport.Schema)))
	recorder := usagereport.New(store.NewUsageStore(apidb.NewFromPrimary(d)), logx.Std{L: log.New(&bytes.Buffer{}, "", 0)})
	t.Cleanup(recorder.Close)
	o := diagnostics.New(diagnostics.Deps{
		STT: okSTT(), TTS: okTTS(), Summarize: okSumm(), Transcode: okTranscode(), Usage: recorder,
		NewRunID: func() string { return "diagnostic-usage" },
	})
	_, err := o.RunSuite(context.Background(), nil)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return recorder.Stats().EnqueuedTotal == 4 }, time.Second, 10*time.Millisecond)
}

func TestRunSuite_STTPassDoesNotClaimQualityMeasured(t *testing.T) {
	o := newOrch(okSTT(), okTTS(), okSumm(), okTranscode())
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := run.Steps[0]
	if !step.OK {
		t.Fatalf("STT step should pass: %+v", step)
	}
	if step.Details["diagnostic_scope"] != "asr_readiness" {
		t.Fatalf("diagnostic_scope = %q, want asr_readiness", step.Details["diagnostic_scope"])
	}
	if step.Details["quality_assessed"] != "false" {
		t.Fatalf("quality_assessed = %q, want false", step.Details["quality_assessed"])
	}
	if step.Details["quality_note"] == "" {
		t.Fatal("quality_note should explain that transcript accuracy is not assessed")
	}
}

func TestRunSuite_STTFiltersHallucinationPreviewButReadinessPasses(t *testing.T) {
	stt := &mocks.STT{Res: &sttchain.Result{
		Text:       "Thanks for watching!",
		Tier:       sttchain.TierLocal,
		ProviderID: "whisper",
		ModelID:    "base",
		Latency:    12 * time.Millisecond,
		Confidence: &sttchain.Confidence{NoSpeechProb: 0.99, AvgLogProb: -2.5},
	}}
	o := newOrch(stt, okTTS(), okSumm(), okTranscode())
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := run.Steps[0]
	if !step.OK {
		t.Fatalf("STT readiness should pass: %+v", step)
	}
	if step.Details["transcript_filtered"] != "true" {
		t.Fatalf("transcript_filtered = %q, want true", step.Details["transcript_filtered"])
	}
	if step.Details["filter_reason"] == "" {
		t.Fatal("filter_reason should be populated")
	}
	if got := step.Details["transcript_preview"]; got != "" {
		t.Fatalf("transcript_preview = %q, want empty", got)
	}
}

func TestRunSuite_STTReadinessPropagatesVADFilter(t *testing.T) {
	stt := okSTT()
	o := diagnostics.New(diagnostics.Deps{
		STT:       stt,
		TTS:       okTTS(),
		Summarize: okSumm(),
		Transcode: okTranscode(),
		NewRunID:  func() string { return "run-1" },
		STTConfig: func(context.Context) sttpkg.StreamConfig {
			cfg := sttpkg.Defaults()
			cfg.VADFilterEnabled = true
			return cfg
		},
	})
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !run.Steps[0].OK {
		t.Fatalf("STT readiness should pass: %+v", run.Steps[0])
	}
	if !stt.LastReq.VADFilter {
		t.Fatal("diagnostics STT request should propagate VADFilter=true")
	}
}

func TestRunSuite_Partial_STTProviderUnavailable(t *testing.T) {
	stt := &mocks.STT{Err: sttchain.ErrAllProvidersFailed}
	o := newOrch(stt, okTTS(), okSumm(), okTranscode())
	run, err := o.RunSuite(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if run.Overall != diagnostics.StatusPartial {
		t.Fatalf("want PARTIAL, got %s", run.Overall)
	}
	if run.FailCount != 1 || run.PassCount != 3 {
		t.Fatalf("counts wrong: %+v", run)
	}
	var sttStep *diagnostics.StepResult
	for i := range run.Steps {
		if run.Steps[i].Capability == diagnostics.CapabilitySTT {
			sttStep = &run.Steps[i]
		}
	}
	if sttStep == nil || sttStep.ErrorCode != "provider_unavailable" {
		t.Fatalf("expected provider_unavailable on STT step, got %+v", sttStep)
	}
}

func TestRunSuite_AllFail(t *testing.T) {
	o := newOrch(
		&mocks.STT{Err: errors.New("stt boom")},
		&mocks.TTS{Err: errors.New("tts boom")},
		&mocks.Summ{Err: errors.New("summ boom")},
		&mocks.Transcode{Err: errors.New("ffmpeg boom")},
	)
	run, err := o.RunSuite(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if run.Overall != diagnostics.StatusFail {
		t.Fatalf("want FAIL, got %s", run.Overall)
	}
	if run.FailCount != 4 {
		t.Fatalf("FailCount want 4, got %d", run.FailCount)
	}
}

func TestRunSuite_CapabilityFilter(t *testing.T) {
	stt := okSTT()
	tts := okTTS()
	su := okSumm()
	tc := okTranscode()
	o := newOrch(stt, tts, su, tc)
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilityTTS})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(run.Steps) != 1 || run.Steps[0].Capability != diagnostics.CapabilityTTS {
		t.Fatalf("expected only TTS step, got %+v", run.Steps)
	}
	if stt.Calls != 0 {
		t.Errorf("STT should not have been called, but call count is %d", stt.Calls)
	}
}

func TestRunSuite_UnknownCapabilityRejected(t *testing.T) {
	o := newOrch(okSTT(), okTTS(), okSumm(), okTranscode())
	_, err := o.RunSuite(context.Background(), []diagnostics.Capability{"banana"})
	if err == nil {
		t.Fatal("expected error for unknown capability")
	}
}

func TestRunSuite_NotConfigured(t *testing.T) {
	o := diagnostics.New(diagnostics.Deps{
		STT: okSTT(),
		// TTS, Summarize, Transcode intentionally nil.
		NewRunID: func() string { return "run-x" },
	})
	run, err := o.RunSuite(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	var notConfigured int
	for _, s := range run.Steps {
		if !s.OK && s.ErrorCode == "not_configured" {
			notConfigured++
		}
	}
	if notConfigured != 3 {
		t.Fatalf("want 3 not_configured steps, got %d (%+v)", notConfigured, run.Steps)
	}
}

// TestRunSuite_HonestErrorCodes locks in the honest per-capability error
// classification: a summarize model that is not installed and an ffmpeg
// rejection must surface distinct, actionable codes — not the opaque
// "internal" the suite used to flatten them to.
func TestRunSuite_HonestErrorCodes(t *testing.T) {
	modelMissing := fmt.Errorf("chain failed: %w",
		fmt.Errorf("%w: model qwen3.5:4b not pulled", intsumm.ErrSummarizeModelNotInstalled))
	ffmpegRejected := fmt.Errorf("%w: %w", intaudio.ErrFfmpegExec, errors.New("ffmpeg: exit 1"))
	ffmpegMissing := fmt.Errorf("wrap: %w", intaudio.ErrFFmpegMissing)

	o := newOrch(
		okSTT(),
		okTTS(),
		&mocks.Summ{Err: modelMissing},
		&mocks.Transcode{Err: ffmpegRejected},
	)
	run, err := o.RunSuite(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	codes := map[diagnostics.Capability]string{}
	for _, s := range run.Steps {
		codes[s.Capability] = s.ErrorCode
	}
	if codes[diagnostics.CapabilitySummarize] != "model_not_installed" {
		t.Errorf("summarize code = %q, want model_not_installed", codes[diagnostics.CapabilitySummarize])
	}
	if codes[diagnostics.CapabilityTranscode] != "invalid_input" {
		t.Errorf("transcode code = %q, want invalid_input", codes[diagnostics.CapabilityTranscode])
	}

	// ffmpeg missing is a distinct, operator-fixable class.
	o2 := newOrch(okSTT(), okTTS(), okSumm(), &mocks.Transcode{Err: ffmpegMissing})
	run2, err := o2.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilityTranscode})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if run2.Steps[0].ErrorCode != "provider_unavailable" {
		t.Errorf("ffmpeg-missing code = %q, want provider_unavailable", run2.Steps[0].ErrorCode)
	}
}

func TestLast_NeverThenAfterRun(t *testing.T) {
	o := newOrch(okSTT(), okTTS(), okSumm(), okTranscode())
	if got := o.Last(); got.ID != "" {
		t.Fatalf("expected empty last before any run, got %+v", got)
	}
	_, _ = o.RunSuite(context.Background(), nil)
	if got := o.Last(); got.ID == "" {
		t.Fatal("expected non-empty last after run")
	}
}

func TestFixtures_EmbeddedSizeUnderBudget(t *testing.T) {
	// Phase B contract: fixture WAV must stay under 100 KB so the embed
	// budget doesn't quietly bloat the API binary.
	wav := loadSmokeWAV()
	if len(wav) == 0 {
		t.Fatal("smoke.wav is empty")
	}
	if len(wav) > 100*1024 {
		t.Fatalf("smoke.wav too large: %d bytes (>100KB)", len(wav))
	}
	if string(wav[:4]) != "RIFF" {
		t.Fatalf("smoke.wav missing RIFF header: %x", wav[:4])
	}
}
