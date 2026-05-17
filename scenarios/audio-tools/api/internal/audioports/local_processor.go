package audioports

import (
	"audio-tools/internal/text/normalizer"
)

// LocalSpeechTextProcessor is the production SpeechTextProcessor backed
// by the in-process internal/text/normalizer pipeline. The future
// audio-tools client will replace this with a remote-call adapter;
// orchestration code is unchanged when that swap happens.
type LocalSpeechTextProcessor struct{}

func (LocalSpeechTextProcessor) NormalizeForSpeech(text string) string {
	return normalizer.NormalizeTextForSpeech(text)
}

func (LocalSpeechTextProcessor) SplitIntoParagraphs(text string) []string {
	return normalizer.SplitIntoSpeechParagraphs(text)
}

// Compile-time assertion.
var _ SpeechTextProcessor = LocalSpeechTextProcessor{}
