package tts

import (
	"regexp"
	"strings"
	"unicode"
)

// TTSMaxChunkLength is the maximum character count per speech chunk. It must
// stay under the backend's maxSynthesizeInputLength (5000) with margin.
const TTSMaxChunkLength = 4500

// SplitIntoSpeechParagraphs normalizes text for speech and splits it into
// chunks that each fit within the synthesis character limit.
//
// This is the single entry point for the backend TTS text pipeline:
//
//	raw text → NormalizeTextForSpeech → split → filter non-speakable → enforce limits
func SplitIntoSpeechParagraphs(text string) []string {
	normalized := NormalizeTextForSpeech(text)
	if strings.TrimSpace(normalized) == "" {
		// Normalization removed all content (e.g. text was just "---").
		// Return original as fallback so callers always get something.
		return []string{text}
	}
	return splitIntoParagraphs(normalized)
}

// splitIntoParagraphs splits text into speakable paragraphs under the chunk
// length limit.
//
// Strategy (matching the original frontend logic):
//  1. Split on double-newline boundaries (paragraph breaks)
//  2. For blocks > 500 chars, split further on single newlines
//  3. Filter out non-speakable chunks
//  4. For chunks over the limit, split on sentence boundaries
//  5. As a last resort, hard-split at TTSMaxChunkLength
func splitIntoParagraphs(text string) []string {
	// Step 1: Split on paragraph breaks.
	raw := splitNonEmpty(text, "\n\n")

	// Step 2: For long blocks, split further on single newlines.
	afterNewlines := make([]string, 0, len(raw))
	for _, block := range raw {
		if len(block) > 500 {
			for _, line := range strings.Split(block, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					afterNewlines = append(afterNewlines, line)
				}
			}
		} else {
			afterNewlines = append(afterNewlines, block)
		}
	}
	if len(afterNewlines) == 0 {
		afterNewlines = []string{text}
	}

	// Step 3: Filter non-speakable + enforce max chunk length.
	result := make([]string, 0, len(afterNewlines))
	for _, chunk := range afterNewlines {
		if !IsSpeakable(chunk) {
			continue
		}
		if len(chunk) <= TTSMaxChunkLength {
			result = append(result, chunk)
		} else {
			result = append(result, splitLongChunk(chunk)...)
		}
	}

	// Fallback: if everything was filtered, return the original text.
	if len(result) == 0 {
		return []string{text}
	}
	return result
}

// IsSpeakable returns true if a chunk contains enough speakable content for
// TTS. Filters out markdown syntax, lone punctuation, code fences, etc. that
// cause the TTS engine to return 0-byte audio.
func IsSpeakable(text string) bool {
	stripped := text
	stripped = reCodeFenceLine.ReplaceAllString(stripped, "")
	stripped = reHeadingMarker.ReplaceAllString(stripped, "")
	stripped = reHRLine.ReplaceAllString(stripped, "")
	stripped = reLoneListMarker.ReplaceAllString(stripped, "")
	stripped = strings.TrimSpace(stripped)

	// Must have at least one word character (letter or digit).
	for _, r := range stripped {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// splitLongChunk breaks a chunk that exceeds TTSMaxChunkLength, preferring
// sentence boundaries.
func splitLongChunk(text string) []string {
	sentences := reSentenceBoundary.FindAllString(text, -1)
	if len(sentences) > 1 {
		var result []string
		var current strings.Builder
		for _, sentence := range sentences {
			if current.Len()+len(sentence) > TTSMaxChunkLength {
				if current.Len() > 0 {
					result = append(result, strings.TrimSpace(current.String()))
					current.Reset()
				}
				if len(sentence) > TTSMaxChunkLength {
					result = append(result, hardSplit(strings.TrimSpace(sentence))...)
					continue
				}
			}
			current.WriteString(sentence)
		}
		if current.Len() > 0 {
			final := strings.TrimSpace(current.String())
			if len(final) > TTSMaxChunkLength {
				result = append(result, hardSplit(final)...)
			} else {
				result = append(result, final)
			}
		}
		return result
	}

	return hardSplit(text)
}

// hardSplit cuts text at TTSMaxChunkLength, preferring word boundaries.
func hardSplit(text string) []string {
	var result []string
	remaining := text
	for len(remaining) > TTSMaxChunkLength {
		splitAt := strings.LastIndex(remaining[:TTSMaxChunkLength], " ")
		if splitAt <= 0 {
			splitAt = TTSMaxChunkLength
		}
		result = append(result, strings.TrimSpace(remaining[:splitAt]))
		remaining = strings.TrimSpace(remaining[splitAt:])
	}
	if trimmed := strings.TrimSpace(remaining); trimmed != "" {
		result = append(result, trimmed)
	}
	return result
}

// splitNonEmpty splits s by sep and returns only non-empty trimmed parts.
func splitNonEmpty(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Compiled patterns for speakability checking
// ---------------------------------------------------------------------------

var (
	reCodeFenceLine  = regexp.MustCompile(`(?m)^` + "```" + `\w*$`)
	reHeadingMarker  = regexp.MustCompile(`(?m)^#{1,6}\s*`)
	reHRLine         = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	reLoneListMarker = regexp.MustCompile(`(?m)^[\s*>+-]+$`)

	// Sentence boundary: text ending with . ! ? followed by whitespace or end.
	reSentenceBoundary = regexp.MustCompile(`[^.!?]*[.!?]+(?:\s+|$)|[^.!?]+$`)
)
