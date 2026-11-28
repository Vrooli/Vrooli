package strategy

import (
	"strings"

	"audio-tools/internal/ai/sttchain"
	voice "audio-tools/internal/stt/pipeline"
)

// timedWordsForCommittedTail maps a provider's request-relative word intervals
// onto the newly committed transcript suffix. VAD pre-roll intentionally
// repeats text across requests, so publishing every provider word would make
// timing rows longer than the final transcript and shift lexical alignment.
// If provider words do not exactly describe the emitted tail, omit timings
// rather than attach them to the wrong text.
func timedWordsForCommittedTail(tail string, providerWords []sttchain.TimedWord) []sttchain.TimedWord {
	tailTokens := strings.Fields(tail)
	if len(tailTokens) == 0 || len(tailTokens) > len(providerWords) {
		return nil
	}
	start := len(providerWords) - len(tailTokens)
	for i, token := range tailTokens {
		normalized := voice.NormalizeToken(token)
		if normalized == "" || normalized != voice.NormalizeToken(providerWords[start+i].Word) {
			return nil
		}
	}
	return append([]sttchain.TimedWord(nil), providerWords[start:]...)
}

// timedWordsForUniqueSpan maps a committed text span to provider timings when
// the span is not necessarily the hypothesis suffix (LocalAgreement may commit
// only its agreed prefix). Ambiguous repeated phrases are omitted so timing
// evidence cannot be attached to the wrong occurrence.
func timedWordsForUniqueSpan(text string, providerWords []sttchain.TimedWord) []sttchain.TimedWord {
	target := strings.Fields(text)
	if len(target) == 0 || len(target) > len(providerWords) {
		return nil
	}
	for _, token := range target {
		if voice.NormalizeToken(token) == "" {
			return nil
		}
	}
	matchStart := -1
	for start := 0; start+len(target) <= len(providerWords); start++ {
		matches := true
		for index, token := range target {
			if voice.NormalizeToken(token) != voice.NormalizeToken(providerWords[start+index].Word) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		if matchStart >= 0 {
			return nil
		}
		matchStart = start
	}
	if matchStart < 0 {
		return nil
	}
	return append([]sttchain.TimedWord(nil), providerWords[matchStart:matchStart+len(target)]...)
}
