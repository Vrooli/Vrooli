// Package fixtures embeds the canned inputs the diagnostics suite
// exercises every capability against. Fixtures live server-side so the
// UI bundle stays slim and every operator runs the same payload.
package fixtures

import _ "embed"

//go:embed smoke.wav
var smokeWAV []byte

//go:embed smoke_text.txt
var smokeText string

// SmokeWAV returns the bundled ~1s 16 kHz mono PCM tone used by the
// STT and Transcode steps.
func SmokeWAV() []byte { return smokeWAV }

// SmokeText returns the canned summarization input used by the
// Summarize and TTS steps.
func SmokeText() string { return smokeText }
