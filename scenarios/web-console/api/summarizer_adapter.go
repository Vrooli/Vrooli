package main

import (
	"context"

	"web-console/internal/audioports"
	inttts "web-console/internal/tts"
)

// remoteSummarizerAdapter wraps audioports.RemoteSummarizer so it satisfies
// inttts.SummarizerBackend. The audio-tools Summarize proto returns a
// flat summary + provider trace; the legacy diagnostic fields (DoneReason,
// EvalCount, RawContent) are not modeled remotely and surface as zero
// values — SummarizationService.classifyEmptySummary degrades to
// ErrSummarizeTrulyEmpty when content is empty, which matches the
// caller-visible behaviour for a remote provider that returned nothing.
type remoteSummarizerAdapter struct {
	remote *audioports.RemoteSummarizer
}

func (a *remoteSummarizerAdapter) Summarize(ctx context.Context, text, model, level string) (inttts.SummarizerResponse, error) {
	if a == nil || a.remote == nil {
		return inttts.SummarizerResponse{}, audioportsUnavailable()
	}
	out, err := a.remote.Summarize(ctx, audioports.SummarizeInput{
		Text:  text,
		Level: level,
		Model: model,
	})
	if err != nil {
		return inttts.SummarizerResponse{}, err
	}
	return inttts.SummarizerResponse{
		Content:    out.Text,
		RawContent: out.Text,
		DoneReason: "stop",
	}, nil
}

func audioportsUnavailable() error {
	return audioportsUnavailableErr
}

var audioportsUnavailableErr = errAudioportsUnavailable{}

type errAudioportsUnavailable struct{}

func (errAudioportsUnavailable) Error() string {
	return "audio-tools summarizer adapter not wired"
}
