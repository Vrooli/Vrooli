// Package transcriptmetrics provides the normalization and alignment rules
// shared by user-facing smoke diagnostics and the offline evaluation harness.
package transcriptmetrics

import (
	"strings"
	"unicode"
)

// NormalizeOptions selects which normalization passes run. The zero value
// runs none; DefaultNormalizeOptions enables the v1 policy.
type NormalizeOptions struct {
	Lowercase          bool
	StripPunctuation   bool
	CollapseWhitespace bool
}

// DefaultNormalizeOptions is the v1 WER policy: lowercase, strip punctuation,
// and collapse whitespace.
func DefaultNormalizeOptions() NormalizeOptions {
	return NormalizeOptions{Lowercase: true, StripPunctuation: true, CollapseWhitespace: true}
}

// Normalize applies lowercase, punctuation removal, then whitespace collapse.
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

// Tokenize normalizes text and splits it into whitespace-delimited tokens.
func Tokenize(s string, opts NormalizeOptions) []string {
	return strings.Fields(Normalize(s, opts))
}
