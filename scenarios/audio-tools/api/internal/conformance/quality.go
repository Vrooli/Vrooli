package conformance

import (
	"fmt"
	"strings"

	"audio-tools/internal/eval"
)

// QualityObservation is the transcript-quality portion of one conformance
// run. The caller supplies the same reference/hypothesis pair that the
// product-path composer observed; no synthetic quality default is invented.
type QualityObservation struct {
	Reference             string
	Hypothesis            string
	MaxWER                float64
	MinPunctuationRate    float64
	MinCapitalisationRate float64
}

// MeasureQuality turns a real transcript observation into explicit quality
// assertions. Presentation rates are recorded in the WER detail so a report
// cannot hide an unpunctuated or uncased engine behind a good recognition
// score.
func MeasureQuality(observation QualityObservation) []Assertion {
	options := eval.DefaultNormalizeOptions()
	referenceTokens := eval.Tokenize(observation.Reference, options)
	hypothesisTokens := eval.Tokenize(observation.Hypothesis, options)
	wer := eval.WER(referenceTokens, hypothesisTokens)
	presentation := eval.MeasurePresentation(observation.Hypothesis)
	detail := fmt.Sprintf("wer=%.4f ref_words=%d hyp_words=%d substitutions=%d insertions=%d deletions=%d punctuation_rate=%.4f capitalisation_rate=%.4f",
		wer.Rate(), len(referenceTokens), len(hypothesisTokens), wer.Substitutions, wer.Insertions, wer.Deletions, presentation.PunctuationRate, presentation.CapitalisationRate)
	return []Assertion{
		Measured("word_error_rate_stable", wer.Rate() <= observation.MaxWER, detail),
		Measured("punctuation_rate_recorded", presentation.PunctuationRate >= observation.MinPunctuationRate, detail),
		Measured("capitalisation_rate_recorded", presentation.CapitalisationRate >= observation.MinCapitalisationRate, detail),
	}
}

// QualityDetailIsMachineReadable is a small guard for consumers that persist
// assertion detail in line-oriented evidence stores.
func QualityDetailIsMachineReadable(assertion Assertion) bool {
	return strings.Contains(assertion.Detail, "wer=") && strings.Contains(assertion.Detail, "punctuation_rate=") && strings.Contains(assertion.Detail, "capitalisation_rate=")
}
