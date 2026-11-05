// Package eval is audio-tools' offline speech-to-text evaluation harness.
//
// It measures the accuracy (WER/CER), compute cost (RTF, Whisper-call
// count, audio-seconds), and finalization latency of the streaming STT
// strategies against a corpus of real audio clips with known ground
// truth. The architecture mirrors the AI-search eval harness
// (packages/ai-go/search/grading.go GradeSuite -> SuiteReport): a corpus
// of cases, a transcriber seam, and a per-case + aggregate report.
//
// This file holds text normalization. WER is meaningless without a fixed
// normalization policy: Whisper jitters capitalization and punctuation
// across runs, and the reference transcript an operator types will differ
// cosmetically from any backend's output. Normalization collapses those
// cosmetic differences so the remaining edit distance reflects real
// recognition errors.
//
// NON-GOAL (documented contract, plan §8): this is NOT a reimplementation
// of OpenAI Whisper's Python text normalizer (number-word expansion,
// currency/unit spelling, contraction tables, locale rules). v1 uses a
// deliberately small local normalizer (lowercase, strip punctuation,
// collapse whitespace). Absolute WER values are therefore only comparable
// WITHIN this harness, not against published Whisper WER benchmarks — the
// harness exists to compare strategies against each other on the same
// normalized footing, which this satisfies.
package eval

import (
	"strings"
	"unicode"
)

// NormalizeOptions selects which normalization passes run. The zero value
// runs none; DefaultNormalizeOptions enables the v1 policy.
type NormalizeOptions struct {
	// Lowercase folds the text to lower case (Unicode-aware).
	Lowercase bool
	// StripPunctuation removes Unicode punctuation and symbol runes. Note:
	// this folds intra-word punctuation too ("don't" -> "dont",
	// "well-known" -> "wellknown"), which is acceptable for v1 because the
	// SAME rule applies to both reference and hypothesis, so it cannot
	// inflate the measured error between them.
	StripPunctuation bool
	// CollapseWhitespace trims and collapses runs of whitespace to a single
	// space.
	CollapseWhitespace bool
}

// DefaultNormalizeOptions is the v1 WER normalization policy: lowercase +
// strip punctuation + collapse whitespace.
func DefaultNormalizeOptions() NormalizeOptions {
	return NormalizeOptions{
		Lowercase:          true,
		StripPunctuation:   true,
		CollapseWhitespace: true,
	}
}

// Normalize applies the selected passes to s and returns the normalized
// string. Order is fixed: lowercase, then strip punctuation, then collapse
// whitespace — so punctuation removal never leaves double spaces behind.
func Normalize(s string, opts NormalizeOptions) string {
	if opts.Lowercase {
		s = strings.ToLower(s)
	}
	if opts.StripPunctuation {
		s = strings.Map(func(r rune) rune {
			if unicode.IsPunct(r) || unicode.IsSymbol(r) {
				return -1
			}
			return r
		}, s)
	}
	if opts.CollapseWhitespace {
		s = strings.Join(strings.Fields(s), " ")
	}
	return s
}

// Tokenize normalizes s and splits it into whitespace-delimited word
// tokens. It is the canonical input to WER.
func Tokenize(s string, opts NormalizeOptions) []string {
	return strings.Fields(Normalize(s, opts))
}
