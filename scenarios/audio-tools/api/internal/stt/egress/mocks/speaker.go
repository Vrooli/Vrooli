package mocks

import (
	"context"

	"audio-tools/internal/stt/egress"
)

// FakeSpeakerIsolation is a programmable egress.SpeakerIsolation. Tests set
// Verdict (or a Decide func over the audio) and record every audio slice
// Evaluate observed.
type FakeSpeakerIsolation struct {
	// Verdict is returned when Decide is nil.
	Verdict egress.SpeakerVerdict
	// Decide, when set, computes the verdict from the audio bytes.
	Decide func(audio []byte) egress.SpeakerVerdict
	// Seen records every audio slice Evaluate observed, in order.
	Seen [][]byte
}

func (f *FakeSpeakerIsolation) Evaluate(_ context.Context, audio []byte) egress.SpeakerVerdict {
	f.Seen = append(f.Seen, audio)
	if f.Decide != nil {
		return f.Decide(audio)
	}
	return f.Verdict
}

var _ egress.SpeakerIsolation = (*FakeSpeakerIsolation)(nil)
