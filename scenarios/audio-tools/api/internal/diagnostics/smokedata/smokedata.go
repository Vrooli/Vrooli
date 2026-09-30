// Package smokedata embeds the canned inputs the diagnostics suite
// exercises every capability against. Smoke data lives server-side so the
// UI bundle stays slim and every operator runs the same payload.
package smokedata

import _ "embed"

//go:embed smoke.wav
var smokeWAV []byte

//go:embed smoke_text.txt
var smokeText string

//go:embed quality_speech.wav
var qualitySpeechWAV []byte

//go:embed quality_silence.wav
var qualitySilenceWAV []byte

// SmokeWAV returns the bundled ~1s 16 kHz mono PCM tone used by the
// STT and Transcode steps.
func SmokeWAV() []byte { return smokeWAV }

// SmokeText returns the canned summarization input used by the
// Summarize and TTS steps.
func SmokeText() string { return smokeText }

// QualitySpeechWAV returns the bundled ~2s 16 kHz mono clean-speech clip
// ("The quick brown fox jumps.") the quality-smoke layer grades by WER. It
// is TTS-generated and trimmed; the reference transcript and WER threshold
// live with the fixture definition in the diagnostics package.
func QualitySpeechWAV() []byte { return qualitySpeechWAV }

// QualitySilenceWAV returns the bundled ~1.2s 16 kHz mono digital-silence
// clip the quality-smoke layer uses for the no-speech hallucination safety
// gate: the user-facing transcript must be empty after the egress policy.
func QualitySilenceWAV() []byte { return qualitySilenceWAV }
