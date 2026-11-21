package main

import (
	"strings"
	"unicode"
)

func normalizePunctuation(text string) string {
	return strings.NewReplacer("，", ",", "。", ".", "？", "?", "！", "!").Replace(text)
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
