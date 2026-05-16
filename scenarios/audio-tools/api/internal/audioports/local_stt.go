package audioports

import (
	"context"

	intvoice "audio-tools/internal/voice"
)

// LocalSpeechToText is the production SpeechToText adapter backed by the
// in-process internal/voice.Service. The future audio-tools client will
// replace this with a remote-call adapter; orchestration code is unchanged
// when that swap happens.
//
// Speaker verification, hallucination filtering, and Whisper-availability
// gating remain inside the voice Adapter/Backend layer reached through the
// Connect handler; this port exposes only the raw transcribe call shape
// orchestration callers need.
type LocalSpeechToText struct {
	Backend *intvoice.Service
}

// Transcribe delegates to the underlying voice service. STTOptions.Language
// maps directly. SkipSpeakerVerification increments the bypass metric on the
// backend when set; InitialPrompt is currently ignored by the voice service
// (the Whisper /asr path does not accept it through Service.Transcribe yet).
func (l LocalSpeechToText) Transcribe(ctx context.Context, audio []byte, opts STTOptions) (STTResult, error) {
	if l.Backend == nil {
		return STTResult{}, nil
	}
	if opts.SkipSpeakerVerification {
		l.Backend.IncrSkipVerification()
	}
	text, err := l.Backend.Transcribe(ctx, audio, opts.Language)
	if err != nil {
		return STTResult{}, err
	}
	if l.Backend.IsWhisperHallucination(text) {
		text = ""
	}
	return STTResult{Text: text}, nil
}

// Compile-time assertion.
var _ SpeechToText = LocalSpeechToText{}
