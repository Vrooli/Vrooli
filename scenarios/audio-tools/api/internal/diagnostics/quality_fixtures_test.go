package diagnostics_test

import (
	"context"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/diagnostics"
	"audio-tools/internal/diagnostics/mocks"
)

// TestDefaultQualityFixtures_BundleIntegrity locks the shape and embed budget
// of the bundled quality-smoke fixtures so a regenerated or corrupted asset
// fails loudly instead of silently degrading the smoke check.
func TestDefaultQualityFixtures_BundleIntegrity(t *testing.T) {
	fixtures := diagnostics.DefaultQualityFixtures()
	if len(fixtures) < 2 {
		t.Fatalf("want at least a no-speech and a speech fixture, got %d", len(fixtures))
	}
	var haveNoSpeech, haveSpeech bool
	for _, f := range fixtures {
		if len(f.WAV) == 0 {
			t.Errorf("fixture %s has empty WAV", f.ID)
			continue
		}
		if string(f.WAV[:4]) != "RIFF" {
			t.Errorf("fixture %s missing RIFF header: %x", f.ID, f.WAV[:4])
		}
		// Diagnostics must stay lean: no bundled fixture should exceed 128 KB
		// (decision D2 — tiny deterministic fixtures, not corpus audio).
		if len(f.WAV) > 128*1024 {
			t.Errorf("fixture %s too large: %d bytes (>128KB)", f.ID, len(f.WAV))
		}
		switch f.Kind {
		case diagnostics.KindNoSpeech:
			haveNoSpeech = true
			if f.Expected != "" {
				t.Errorf("no-speech fixture %s must not carry an expected transcript", f.ID)
			}
		case diagnostics.KindSpeech:
			haveSpeech = true
			if f.Expected == "" {
				t.Errorf("speech fixture %s must carry an expected transcript", f.ID)
			}
			if f.WERThreshold <= 0 || f.WERThreshold >= 1 {
				t.Errorf("speech fixture %s WER threshold %.3f out of (0,1)", f.ID, f.WERThreshold)
			}
		default:
			t.Errorf("fixture %s has unknown kind %q", f.ID, f.Kind)
		}
	}
	if !haveNoSpeech || !haveSpeech {
		t.Fatalf("fixture set must include both no-speech and speech kinds (noSpeech=%v speech=%v)", haveNoSpeech, haveSpeech)
	}
}

// bundledQualityOrch drives the REAL bundled fixtures through a fixture-aware
// fake keyed on the actual embedded WAV bytes, so these tests exercise the
// production fixture metadata (reference text, threshold) without live Whisper.
func bundledQualityOrch(byAudio map[string]*sttchain.Result) *diagnostics.Orchestrator {
	return diagnostics.New(diagnostics.Deps{
		STT: &mocks.STT{ResFunc: func(req sttchain.Request) (*sttchain.Result, error) {
			if r, ok := byAudio[string(req.Audio)]; ok {
				return r, nil
			}
			return &sttchain.Result{Text: "hello", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: 3 * time.Millisecond}, nil
		}},
		TTS:             okTTS(),
		Summarize:       okSumm(),
		Transcode:       okTranscode(),
		NewRunID:        func() string { return "run-bundle" },
		QualityFixtures: diagnostics.DefaultQualityFixtures(),
	})
}

// TestQualitySmoke_CaseAndPunctuationNormalized: a transcript that differs
// from the reference only in case and punctuation must normalize to WER 0 and
// pass, proving diagnostics reuse the eval normalizer's semantics.
func TestQualitySmoke_CaseAndPunctuationNormalized(t *testing.T) {
	fixtures := diagnostics.DefaultQualityFixtures()
	var speechWAV, silenceWAV []byte
	for _, f := range fixtures {
		switch f.Kind {
		case diagnostics.KindSpeech:
			speechWAV = f.WAV
		case diagnostics.KindNoSpeech:
			silenceWAV = f.WAV
		}
	}
	o := bundledQualityOrch(map[string]*sttchain.Result{
		string(silenceWAV): {Text: "", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
		// Reference is "the quick brown fox jumps"; jitter case + punctuation.
		string(speechWAV): {Text: "The Quick, Brown Fox JUMPS!", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"},
	})
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := sttStep(t, run)
	if step.Details["quality_status"] != "pass" {
		t.Fatalf("quality_status = %q, want pass (case/punct should normalize)", step.Details["quality_status"])
	}
	for _, f := range decodeFixtures(t, step) {
		if f.ExpectedKind == "speech" && f.WER != 0 {
			t.Fatalf("normalized WER should be 0, got %.3f", f.WER)
		}
	}
}

// TestQualitySmoke_ProviderErrorWarnsNotFails: a provider error during the
// quality probe must not hard-fail the STT step — readiness owns reachability.
func TestQualitySmoke_ProviderErrorWarnsNotFails(t *testing.T) {
	fixtures := diagnostics.DefaultQualityFixtures()
	var speechWAV []byte
	for _, f := range fixtures {
		if f.Kind == diagnostics.KindSpeech {
			speechWAV = f.WAV
		}
	}
	o := diagnostics.New(diagnostics.Deps{
		STT: &mocks.STT{ResFunc: func(req sttchain.Request) (*sttchain.Result, error) {
			if string(req.Audio) == string(speechWAV) {
				return nil, sttchain.ErrAllProvidersFailed
			}
			return &sttchain.Result{Text: "", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"}, nil
		}},
		TTS:             okTTS(),
		Summarize:       okSumm(),
		Transcode:       okTranscode(),
		NewRunID:        func() string { return "run-err" },
		QualityFixtures: fixtures,
	})
	run, err := o.RunSuite(context.Background(), []diagnostics.Capability{diagnostics.CapabilitySTT})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	step := sttStep(t, run)
	if !step.OK {
		t.Fatalf("provider error in quality probe should not fail the step: %+v", step)
	}
	if step.Details["quality_status"] != "warn" {
		t.Fatalf("quality_status = %q, want warn", step.Details["quality_status"])
	}
}
