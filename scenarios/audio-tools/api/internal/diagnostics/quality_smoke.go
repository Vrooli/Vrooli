package diagnostics

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/diagnostics/smokedata"
	"audio-tools/internal/eval"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/pipeline"
	"audio-tools/internal/stt/quality"
)

// cleanSpeechReference is the ground-truth transcript of the bundled
// quality_speech.wav clip. Kept beside the fixture so WER grading and the
// fixture audio can never drift apart silently.
const cleanSpeechReference = "the quick brown fox jumps"

// cleanSpeechWERThreshold is the pass ceiling for the clean-speech fixture's
// normalized WER. It is deliberately lenient: diagnostics grade a live model
// on a tiny clip, so the goal is catching gross transcription breakage
// (empty/garbage output), not benchmarking accuracy — full corpus grading is
// the Dictation Studio eval harness's job (decision D2). Documented in
// docs/reference/eval-harness.md.
const cleanSpeechWERThreshold = 0.34

// DefaultQualityFixtures is the production quality-smoke fixture set: a
// no-speech silence clip for the hallucination safety gate and a short
// clean-speech clip for WER grading. Both are bundled, tiny, and
// deterministic — no corpus data, microphone, or long sleeps (decision D2).
func DefaultQualityFixtures() []QualityFixture {
	return []QualityFixture{
		{
			ID:   "no_speech_silence",
			Kind: KindNoSpeech,
			WAV:  smokedata.QualitySilenceWAV(),
		},
		{
			ID:           "clean_speech",
			Kind:         KindSpeech,
			WAV:          smokedata.QualitySpeechWAV(),
			Expected:     cleanSpeechReference,
			WERThreshold: cleanSpeechWERThreshold,
		},
	}
}

// QualityStatus is the tri-state verdict of one quality-smoke fixture or
// the aggregate over the fixture set. Unlike readiness (a boolean provider
// reachability check), quality smoke distinguishes a soft WARN — quality
// drift worth an operator's attention — from a hard FAIL that flips the
// suite because a safety invariant was violated.
type QualityStatus string

const (
	// QualityPass — the fixture met its expectation (empty transcript for
	// no-speech, WER under threshold for clean speech).
	QualityPass QualityStatus = "pass"
	// QualityWarn — soft quality drift (WER over threshold but bounded, or
	// clean speech unexpectedly filtered). Surfaced, but does not fail the
	// diagnostics suite.
	QualityWarn QualityStatus = "warn"
	// QualityFail — a hard safety violation: a no-speech fixture produced a
	// surviving user-facing transcript. This is the exact "Thanks for
	// watching!" hallucination class and flips the STT step.
	QualityFail QualityStatus = "fail"
)

// severity orders the statuses so the aggregate can take the worst.
func (q QualityStatus) severity() int {
	switch q {
	case QualityFail:
		return 2
	case QualityWarn:
		return 1
	default:
		return 0
	}
}

// QualityKind labels what a fixture is meant to prove.
type QualityKind string

const (
	// KindNoSpeech — silence / no-speech audio. The user-facing transcript
	// MUST be empty after the shared egress policy runs; anything surviving
	// is a hallucination leak.
	KindNoSpeech QualityKind = "no_speech"
	// KindSpeech — clean speech with a known reference transcript. Graded by
	// normalized WER against WERThreshold.
	KindSpeech QualityKind = "speech"
)

// QualityFixture is one bundled deterministic quality-smoke input. The
// diagnostics runner drives each fixture through the SAME STT chain and
// egress policy that user-facing transcription uses (decision D3), so the
// smoke check exercises the real quality path rather than a shadow copy.
type QualityFixture struct {
	// ID is the stable fixture identifier surfaced to CLI/UI consumers.
	ID string
	// Kind selects the grading rule.
	Kind QualityKind
	// WAV is the fixture audio (RIFF/WAV container).
	WAV []byte
	// Expected is the reference transcript for KindSpeech fixtures; unused
	// for KindNoSpeech.
	Expected string
	// WERThreshold is the pass ceiling for KindSpeech normalized WER. WER at
	// or below it passes; up to 2× warns; beyond that warns (never a hard
	// fail — model drift is not a substrate safety failure).
	WERThreshold float64
}

// FixtureQualityResult is one fixture's structured outcome. Every field is
// wire-serializable so CLI/UI can render quality evidence without parsing
// prose (plan Phase 1 step 4).
type FixtureQualityResult struct {
	FixtureID                string        `json:"fixture_id"`
	ExpectedKind             QualityKind   `json:"expected_kind"`
	Status                   QualityStatus `json:"status"`
	WER                      float64       `json:"wer"`
	WERThreshold             float64       `json:"wer_threshold"`
	Filtered                 bool          `json:"transcript_filtered"`
	FilterReason             string        `json:"filter_reason"`
	HallucinationDetected    bool          `json:"hallucination_detected"`
	RawTranscriptLength      int           `json:"raw_transcript_length"`
	FilteredTranscriptLength int           `json:"filtered_transcript_length"`
	// Preview is a bounded, SAFE excerpt: it is only ever the surviving
	// (post-policy) transcript, and is always empty for no-speech fixtures so
	// filtered hallucination text can never render as successful output.
	Preview string `json:"preview"`
	Note    string `json:"note"`
	Error   string `json:"error,omitempty"`
}

// QualitySmokeReport aggregates the fixture results the STT step surfaces.
type QualitySmokeReport struct {
	Status                QualityStatus          `json:"status"`
	HallucinationDetected bool                   `json:"hallucination_detected"`
	Fixtures              []FixtureQualityResult `json:"fixtures"`
}

// qualityPreviewMax bounds transcript previews so a fixture cannot dump an
// unbounded transcript into diagnostics details.
const qualityPreviewMax = 80

// runQualitySmoke drives each fixture through the STT runner and the shared
// egress policy, grading the surviving transcript. It never calls into a
// physical microphone or the ExperimentService corpus (decision D2): the
// fixtures are tiny and deterministic. A nil/empty fixture set yields a
// zero report the caller treats as "quality not assessed".
func runQualitySmoke(
	ctx context.Context,
	stt SttRunner,
	pol quality.Policy,
	cfg sttpkg.StreamConfig,
	fixtures []QualityFixture,
) QualitySmokeReport {
	report := QualitySmokeReport{Status: QualityPass, Fixtures: make([]FixtureQualityResult, 0, len(fixtures))}
	for _, f := range fixtures {
		fr := evaluateFixture(ctx, stt, pol, cfg, f)
		if fr.HallucinationDetected {
			report.HallucinationDetected = true
		}
		if fr.Status.severity() > report.Status.severity() {
			report.Status = fr.Status
		}
		report.Fixtures = append(report.Fixtures, fr)
	}
	return report
}

// evaluateFixture transcribes one fixture and grades it. It is pure with
// respect to the injected seams so unit tests can exercise every branch
// with a fixture-aware fake STT runner (no live Whisper).
func evaluateFixture(
	ctx context.Context,
	stt SttRunner,
	pol quality.Policy,
	cfg sttpkg.StreamConfig,
	f QualityFixture,
) FixtureQualityResult {
	fr := FixtureQualityResult{
		FixtureID:    f.ID,
		ExpectedKind: f.Kind,
		WERThreshold: f.WERThreshold,
	}
	out, err := stt.Execute(ctx, sttchain.Request{Audio: f.WAV, Format: "wav", VADFilter: cfg.VADFilterEnabled})
	if err != nil {
		// A provider error during the smoke probe is a soft WARN, not a hard
		// fail: readiness owns provider reachability, and we must not turn a
		// transient provider blip into a suite-reddening quality failure.
		fr.Status = QualityWarn
		fr.Error = err.Error()
		fr.Note = "quality smoke could not run: STT chain returned an error"
		return fr
	}
	raw := ""
	if out != nil {
		raw = out.Text
	}
	decision := pol.ApplyResult(ctx, out, f.WAV)
	fr.Filtered = decision.Filtered
	fr.FilterReason = decision.FilterReason
	fr.RawTranscriptLength = len(raw)
	fr.FilteredTranscriptLength = len(decision.Text)

	switch f.Kind {
	case KindNoSpeech:
		if decision.Text != "" {
			// The egress policy failed to suppress a transcript on no-speech
			// audio — the exact hallucination leak the plan exists to catch
			// (decision D4, zero tolerance). Never preview the surviving text.
			fr.Status = QualityFail
			fr.HallucinationDetected = true
			fr.Note = "no-speech fixture produced a surviving transcript; egress policy did not suppress it"
			return fr
		}
		fr.Status = QualityPass
		switch {
		case pipeline.IsWhisperHallucination(raw) && raw != "":
			fr.HallucinationDetected = true
			fr.Note = "known no-speech hallucination detected and suppressed by egress policy"
		case raw != "":
			fr.Note = "no-speech fixture transcript suppressed by egress policy"
		default:
			fr.Note = "no-speech fixture produced no transcript"
		}
		return fr

	case KindSpeech:
		if decision.Filtered {
			// Clean speech should never be filtered. This is quality drift, not
			// a safety leak, so it warns rather than fails.
			fr.Status = QualityWarn
			fr.Note = fmt.Sprintf("clean-speech fixture was filtered by egress policy (%s)", decision.FilterReason)
			return fr
		}
		norm := eval.DefaultNormalizeOptions()
		wr := eval.WER(eval.Tokenize(f.Expected, norm), eval.Tokenize(decision.Text, norm))
		fr.WER = wr.Rate()
		fr.Preview = previewString(decision.Text, qualityPreviewMax)
		switch {
		case fr.WER <= f.WERThreshold:
			fr.Status = QualityPass
			fr.Note = fmt.Sprintf("clean-speech WER %.3f within threshold %.3f", fr.WER, f.WERThreshold)
		default:
			// Over threshold is model quality drift; warn so a health check
			// stays green while still flagging the regression for follow-up in
			// Dictation Studio (decision D2).
			fr.Status = QualityWarn
			fr.Note = fmt.Sprintf("clean-speech WER %.3f exceeds threshold %.3f", fr.WER, f.WERThreshold)
		}
		return fr

	default:
		fr.Status = QualityWarn
		fr.Note = fmt.Sprintf("unknown fixture kind %q", f.Kind)
		return fr
	}
}
