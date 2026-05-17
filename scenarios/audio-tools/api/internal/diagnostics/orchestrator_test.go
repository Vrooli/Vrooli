package diagnostics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/diagnostics"
)

type stubSTT struct {
	res  *sttchain.Result
	err  error
	call int
}

func (s *stubSTT) Execute(_ context.Context, _ sttchain.Request) (*sttchain.Result, error) {
	s.call++
	return s.res, s.err
}

type stubTTS struct {
	res *ttschain.Result
	err error
}

func (s *stubTTS) Execute(_ context.Context, _ ttschain.Request) (*ttschain.Result, error) {
	return s.res, s.err
}

type stubSumm struct {
	res *summarizechain.Result
	err error
}

func (s *stubSumm) Execute(_ context.Context, _ summarizechain.Request) (*summarizechain.Result, error) {
	return s.res, s.err
}

type stubTranscode struct {
	out []byte
	err error
}

func (s *stubTranscode) Transcode(_ context.Context, _ []byte, _ string) ([]byte, error) {
	return s.out, s.err
}

func okSTT() *stubSTT {
	return &stubSTT{res: &sttchain.Result{Text: "hello", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: 12 * time.Millisecond}}
}

func okTTS() *stubTTS {
	return &stubTTS{res: &ttschain.Result{Audio: []byte("RIFF"), ContentType: "audio/wav", Tier: ttschain.TierLocal, ProviderID: "kokoro", ModelID: "v1", Latency: 14 * time.Millisecond}}
}

func okSumm() *stubSumm {
	return &stubSumm{res: &summarizechain.Result{Text: "tldr", Tier: summarizechain.TierLocal, ProviderID: "ollama", ModelID: "llama3", Latency: 17 * time.Millisecond}}
}

func okTranscode() *stubTranscode {
	return &stubTranscode{out: make([]byte, 1024)}
}

func newOrch(s *stubSTT, ts *stubTTS, su *stubSumm, tc *stubTranscode) *diagnostics.Orchestrator {
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

func TestRunSuite_Partial_STTProviderUnavailable(t *testing.T) {
	stt := &stubSTT{err: sttchain.ErrAllProvidersFailed}
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
		&stubSTT{err: errors.New("stt boom")},
		&stubTTS{err: errors.New("tts boom")},
		&stubSumm{err: errors.New("summ boom")},
		&stubTranscode{err: errors.New("ffmpeg boom")},
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
	if stt.call != 0 {
		t.Errorf("STT should not have been called, but call count is %d", stt.call)
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
