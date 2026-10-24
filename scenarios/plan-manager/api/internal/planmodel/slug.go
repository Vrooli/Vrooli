package planmodel

import "strings"

// MaxSlugLength caps newly DERIVED slugs so handles stay typeable in every
// hinted command. Existing longer slugs are untouched and keep resolving —
// the cap applies at derivation time (plan/session creation), never when
// reading or path-mapping stored slugs.
const MaxSlugLength = 60

// TruncateSlug shortens an already-slugified value to maxLen at a word (dash)
// boundary — never mid-word — and trims trailing dashes. Collision suffixes
// (-2, -3, …) are applied AFTER truncation by the callers and may exceed the
// cap by their own length only.
func TruncateSlug(slug string, maxLen int) string {
	if maxLen <= 0 || len(slug) <= maxLen {
		return slug
	}
	cut := slug[:maxLen]
	if idx := strings.LastIndexByte(cut, '-'); idx > 0 {
		cut = cut[:idx]
	}
	return strings.Trim(cut, "-")
}
