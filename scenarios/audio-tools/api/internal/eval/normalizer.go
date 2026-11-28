//go:build audio_lab

package eval

import "audio-tools/internal/transcriptmetrics"

// NormalizeOptions selects the shared transcript-normalization passes.
type NormalizeOptions = transcriptmetrics.NormalizeOptions

// DefaultNormalizeOptions is the evaluation harness' versioned v1 policy.
func DefaultNormalizeOptions() NormalizeOptions {
	return transcriptmetrics.DefaultNormalizeOptions()
}

// Normalize applies the shared normalization policy.
func Normalize(s string, opts NormalizeOptions) string {
	return transcriptmetrics.Normalize(s, opts)
}

// Tokenize normalizes and splits text into words.
func Tokenize(s string, opts NormalizeOptions) []string {
	return transcriptmetrics.Tokenize(s, opts)
}
