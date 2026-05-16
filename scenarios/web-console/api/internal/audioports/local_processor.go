package audioports

import (
	inttts "web-console/internal/tts"
)

// LocalSpeechTextProcessor is the production SpeechTextProcessor backed by
// the in-process internal/tts text pipeline. The future audio-tools client
// will replace this with a remote-call adapter; orchestration code is
// unchanged when that swap happens.
type LocalSpeechTextProcessor struct{}

func (LocalSpeechTextProcessor) NormalizeForSpeech(text string) string {
	return inttts.NormalizeTextForSpeech(text)
}

func (LocalSpeechTextProcessor) SplitIntoParagraphs(text string) []string {
	return inttts.SplitIntoSpeechParagraphs(text)
}

// Compile-time assertion.
var _ SpeechTextProcessor = LocalSpeechTextProcessor{}
