// Package mocks provides hoisted fakes for the diagnostics runner
// seams (SttRunner, TtsRunner, SummaryRunner, Transcoder). They live
// here so test files never declare inline fakes (see TESTING.md L4
// rule). Each fake carries a compile-time `var _ Iface = ...` check.
package mocks

import (
	"context"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/diagnostics"
)

// STT satisfies diagnostics.SttRunner. Set Res/Err to control the
// response; Calls counts invocations for assertion.
type STT struct {
	Res   *sttchain.Result
	Err   error
	Calls int
}

func (s *STT) Execute(_ context.Context, _ sttchain.Request) (*sttchain.Result, error) {
	s.Calls++
	return s.Res, s.Err
}

// TTS satisfies diagnostics.TtsRunner.
type TTS struct {
	Res   *ttschain.Result
	Err   error
	Calls int
}

func (t *TTS) Execute(_ context.Context, _ ttschain.Request) (*ttschain.Result, error) {
	t.Calls++
	return t.Res, t.Err
}

// Summ satisfies diagnostics.SummaryRunner.
type Summ struct {
	Res   *summarizechain.Result
	Err   error
	Calls int
}

func (s *Summ) Execute(_ context.Context, _ summarizechain.Request) (*summarizechain.Result, error) {
	s.Calls++
	return s.Res, s.Err
}

// Transcode satisfies diagnostics.Transcoder.
type Transcode struct {
	Out   []byte
	Err   error
	Calls int
}

func (t *Transcode) Transcode(_ context.Context, _ []byte, _ string) ([]byte, error) {
	t.Calls++
	return t.Out, t.Err
}

var (
	_ diagnostics.SttRunner     = (*STT)(nil)
	_ diagnostics.TtsRunner     = (*TTS)(nil)
	_ diagnostics.SummaryRunner = (*Summ)(nil)
	_ diagnostics.Transcoder    = (*Transcode)(nil)
)
