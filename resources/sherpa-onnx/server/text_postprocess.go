package main

import (
	"strings"
	"unicode"
)

func normalizePunctuation(text string) string {
	text = strings.NewReplacer("，", ",", "。", ".", "？", "?", "！", "!").Replace(text)
	runes := []rune(text)
	var normalized strings.Builder
	normalized.Grow(len(text))
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		normalized.WriteRune(r)
		if !isWordSeparator(r) {
			continue
		}

		// Punctuation models sometimes omit the space after separators in Latin
		// text. Periods are intentionally excluded because they also join
		// abbreviations, initials, and domain-like tokens.
		if i+1 < len(runes) && isWordSeparator(runes[i+1]) {
			continue
		}
		before := i - 1
		for before >= 0 && isWordSeparator(runes[before]) {
			before--
		}
		if before >= 0 && i+1 < len(runes) && isLatinLetter(runes[before]) && isLatinLetter(runes[i+1]) {
			normalized.WriteByte(' ')
		}
	}
	return normalized.String()
}

func isWordSeparator(r rune) bool {
	switch r {
	case ',', ';', ':', '?', '!':
		return true
	default:
		return false
	}
}

func isLatinLetter(r rune) bool {
	return unicode.IsLetter(r) && unicode.In(r, unicode.Latin)
}

// restoreCapitalization converts the all-caps form emitted by the bundled
// punctuation model into sentence case. Mixed-case output is preserved so a
// future model can retain names and acronyms without this adapter rewriting
// them.
func restoreCapitalization(text string) string {
	if text == "" {
		return text
	}
	hasLetter := false
	allUpper := true
	for _, r := range text {
		if unicode.IsLetter(r) {
			hasLetter = true
			if unicode.IsLower(r) {
				allUpper = false
			}
		}
	}
	if !hasLetter || !allUpper {
		return text
	}
	runes := []rune(strings.ToLower(text))
	capitalize := true
	for i, r := range runes {
		if unicode.IsLetter(r) {
			if capitalize {
				runes[i] = unicode.ToUpper(r)
			}
			capitalize = false
			continue
		}
		if strings.ContainsRune(".!?。！？\n", r) {
			capitalize = true
		}
	}
	return string(runes)
}
