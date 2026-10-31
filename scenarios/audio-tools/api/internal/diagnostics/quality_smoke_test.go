package diagnostics_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/diagnostics"
	"audio-tools/internal/diagnostics/mocks"
)

// fixtureResult mirrors diagnostics.FixtureQualityResult for JSON decoding in
// tests without exporting a decode helper from the production package.
type fixtureResult struct {
	FixtureID             string  `json:"fixture_id"`
	ExpectedKind          string  `json:"expected_kind"`
	Status                string  `json:"status"`
	WER                   float64 `json:"wer"`
	WERThreshold          float64 `json:"wer_threshold"`
	Filtered              bool    `json:"transcript_filtered"`
	FilterReason          string  `json:"filter_reason"`
	HallucinationDetected bool    `json:"hallucination_detected"`
	Preview               string  `json:"preview"`
}

// qualityFixtures is the deterministic fixture set the contract tests drive.
// Audio bytes are synthetic markers — the fixture-aware fake STT keys on them
// rather than transcribing real audio, so these tests never touch Whisper.
var (
	silenceAudio = []byte("FIXTURE:silence")
	speechAudio  = []byte("FIXTURE:speech")
)

func qualityFixtures() []diagnostics.QualityFixture {
	return []diagnostics.QualityFixture{
		{ID: "silence", Kind: diagnostics.KindNoSpeech, WAV: silenceAudio},
		{ID: "clean-speech", Kind: diagnostics.KindSpeech, WAV: speechAudio, Expected: "the quick brown fox jumps over the lazy dog", WERThreshold: 0.2},
	}
}

// fixtureAwareSTT returns per-fixture transcripts keyed on request audio, so a
// single fake can model "silence hallucinates X, speech transcribes Y".
func fixtureAwareSTT(byAudio map[string]*sttchain.Result) *mocks.STT {
	return &mocks.STT{ResFunc: func(req sttchain.Request) (*sttchain.Result, error) {
		if r, ok := byAudio[string(req.Audio)]; ok {
			return r, nil
		}
		// Readiness probe (bundled tone) and any other audio: benign result.
		return &sttchain.Result{Text: "hello", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: 5 * time.Millisecond}, nil
	}}
}

func orchWithQuality(stt *mocks.STT) *diagnostics.Orchestrator {
	return diagnostics.New(diagnostics.Deps{
		STT:             stt,
		TTS:             okTTS(),
		Summarize:       okSumm(),
		Transcode:       okTranscode(),
		NewRunID:        func() string { return "run-q" },
		QualityFixtures: qualityFixtures(),
	})
}

func sttStep(t *testing.T, run diagnostics.Run) diagnostics.StepResult {
	t.Helper()
	for _, s := range run.Steps {
		if s.Capability == diagnostics.CapabilitySTT {
			return s
		}
	}
	t.Fatal("no STT step in run")
	return diagnostics.StepResult{}
}

func decodeFixtures(t *testing.T, step diagnostics.StepResult) []fixtureResult {
	t.Helper()
	blob := step.Details["quality_fixtures"]
	if blob == "" {
		t.Fatal("quality_fixtures detail is empty")
	}
	var out []fixtureResult
	if err := json.Unmarshal([]byte(blob), &out); err != nil {
		t.Fatalf("quality_fixtures not valid JSON: %v", err)
	}
	return out
}

// TestQualitySmoke_CleanRun: readiness passes, no-speech is safely empty, and
// clean speech transcribes under threshold — the all-green two-layer path.
func TestQualitySmoke_CleanRun(t *testing.T) {
	stt := fixtureAwareSTT(map[string]*sttchain.Result{
		string(silenceAudio): {Text: "", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
		string(speechAudio):  {Text: "The quick brown fox jumps over the lazy dog.", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
	})
	o := orchWithQuality(stt)
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := sttStep(t, run)
	if !step.OK {
		t.Fatalf("STT step should pass: %+v", step)
	}
	// Readiness stays its own signal.
	if step.Details["diagnostic_scope"] != "asr_readiness" {
		t.Fatalf("readiness scope missing: %q", step.Details["diagnostic_scope"])
	}
	if step.Details["quality_assessed"] != "true" {
		t.Fatalf("quality_assessed = %q, want true", step.Details["quality_assessed"])
	}
	if step.Details["quality_status"] != "pass" {
		t.Fatalf("quality_status = %q, want pass", step.Details["quality_status"])
	}
	fx := decodeFixtures(t, step)
	if len(fx) != 2 {
		t.Fatalf("want 2 fixtures, got %d", len(fx))
	}
	for _, f := range fx {
		if f.Status != "pass" {
			t.Errorf("fixture %s status = %s, want pass (%+v)", f.FixtureID, f.Status, f)
		}
	}
}

// TestQualitySmoke_NoSpeechHallucinationFails: the readiness provider is
// reachable (readiness PASS), but a no-speech fixture leaks a surviving
// transcript the egress policy did not suppress — a hard quality FAIL that
// flips the STT step (decision D4).
func TestQualitySmoke_NoSpeechHallucinationFails(t *testing.T) {
	stt := fixtureAwareSTT(map[string]*sttchain.Result{
		// No confidence signal → confidence stage cannot drop it; hallucination
		// stage only catches the known phrase list, so "you know what i mean"
		// survives the gate and represents a real leak.
		string(silenceAudio): {Text: "you know what i mean", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
		string(speechAudio):  {Text: "The quick brown fox jumps over the lazy dog.", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
	})
	o := orchWithQuality(stt)
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := sttStep(t, run)
	if step.OK {
		t.Fatal("STT step should fail when a no-speech fixture leaks a transcript")
	}
	if step.ErrorCode != "quality_smoke_failed" {
		t.Fatalf("error_code = %q, want quality_smoke_failed", step.ErrorCode)
	}
	// Readiness must still read as reachable — the provider answered.
	if step.Details["diagnostic_scope"] != "asr_readiness" {
		t.Fatal("readiness scope should remain present even on quality fail")
	}
	if step.Details["quality_status"] != "fail" {
		t.Fatalf("quality_status = %q, want fail", step.Details["quality_status"])
	}
	if step.Details["quality_hallucination_detected"] != "true" {
		t.Fatalf("quality_hallucination_detected = %q, want true", step.Details["quality_hallucination_detected"])
	}
	fx := decodeFixtures(t, step)
	var silence fixtureResult
	for _, f := range fx {
		if f.FixtureID == "silence" {
			silence = f
		}
	}
	if silence.Status != "fail" || !silence.HallucinationDetected {
		t.Fatalf("silence fixture should fail with hallucination detected: %+v", silence)
	}
	if silence.Preview != "" {
		t.Fatalf("no-speech leak must never be previewed, got %q", silence.Preview)
	}
}

// TestQualitySmoke_KnownHallucinationFilteredPasses: the classic "Thanks for
// watching!" appears on the no-speech fixture but the shared egress policy
// suppresses it — quality PASSES and the phrase never surfaces as a preview.
func TestQualitySmoke_KnownHallucinationFilteredPasses(t *testing.T) {
	stt := fixtureAwareSTT(map[string]*sttchain.Result{
		string(silenceAudio): {
			Text:       "Thanks for watching!",
			Tier:       sttchain.TierLocal,
			ProviderID: "whisper",
			ModelID:    "base",
			Confidence: &sttchain.Confidence{NoSpeechProb: 0.99, AvgLogProb: -2.5},
		},
		string(speechAudio): {Text: "The quick brown fox jumps over the lazy dog.", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
	})
	o := orchWithQuality(stt)
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := sttStep(t, run)
	if !step.OK {
		t.Fatalf("STT step should pass when the hallucination is filtered: %+v", step)
	}
	if step.Details["quality_status"] != "pass" {
		t.Fatalf("quality_status = %q, want pass", step.Details["quality_status"])
	}
	fx := decodeFixtures(t, step)
	var silence fixtureResult
	for _, f := range fx {
		if f.FixtureID == "silence" {
			silence = f
		}
	}
	if silence.Status != "pass" {
		t.Fatalf("filtered silence fixture should pass: %+v", silence)
	}
	if !silence.Filtered || silence.FilterReason == "" {
		t.Fatalf("silence fixture should record filter metadata: %+v", silence)
	}
	if !silence.HallucinationDetected {
		t.Fatal("a filtered known hallucination should still record hallucination_detected=true")
	}
	if silence.Preview != "" {
		t.Fatalf("filtered hallucination must never be previewed, got %q", silence.Preview)
	}
}

// TestQualitySmoke_CleanSpeechDriftWarns: clean speech that transcribes badly
// (WER over threshold) warns without flipping the suite — model drift is not a
// substrate safety failure.
func TestQualitySmoke_CleanSpeechDriftWarns(t *testing.T) {
	stt := fixtureAwareSTT(map[string]*sttchain.Result{
		string(silenceAudio): {Text: "", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
		string(speechAudio):  {Text: "completely different words entirely", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
	})
	o := orchWithQuality(stt)
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := sttStep(t, run)
	if !step.OK {
		t.Fatalf("STT step should stay OK on WER drift (warn, not fail): %+v", step)
	}
	if step.Details["quality_status"] != "warn" {
		t.Fatalf("quality_status = %q, want warn", step.Details["quality_status"])
	}
	fx := decodeFixtures(t, step)
	var speech fixtureResult
	for _, f := range fx {
		if f.FixtureID == "clean-speech" {
			speech = f
		}
	}
	if speech.Status != "warn" {
		t.Fatalf("clean-speech drift should warn: %+v", speech)
	}
	if speech.WER <= speech.WERThreshold {
		t.Fatalf("expected WER above threshold, got wer=%.3f threshold=%.3f", speech.WER, speech.WERThreshold)
	}
}

// TestQualitySmoke_NotAssessedWithoutFixtures: with no fixtures configured the
// STT step reports quality as not assessed and never claims a quality status.
func TestQualitySmoke_NotAssessedWithoutFixtures(t *testing.T) {
	o := diagnostics.New(diagnostics.Deps{
		STT:       okSTT(),
		TTS:       okTTS(),
		Summarize: okSumm(),
		Transcode: okTranscode(),
		NewRunID:  func() string { return "run-noq" },
	})
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := sttStep(t, run)
	if step.Details["quality_assessed"] != "false" {
		t.Fatalf("quality_assessed = %q, want false", step.Details["quality_assessed"])
	}
	if _, ok := step.Details["quality_status"]; ok {
		t.Fatal("quality_status should be absent when quality is not assessed")
	}
}
