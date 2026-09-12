package domain

import "strings"

// NormalizeImportHarness reduces a harness label to the identity it stands for.
//
// The same store is labelled differently by different import paths — the
// scheduled sweep writes "codex" while the corpus importer writes
// "resource:codex/sessions" — so the raw label describes where a transcript was
// read from, and only the normalized form says which harness it is. Comparing
// raw labels let the two paths each adopt the same session, duplicating runs on
// every sweep. Identity comparisons must go through this function.
func NormalizeImportHarness(harness string) string {
	normalized := strings.ToLower(strings.TrimSpace(harness))
	normalized = strings.TrimPrefix(normalized, "resource:")
	if index := strings.IndexByte(normalized, '/'); index >= 0 {
		normalized = normalized[:index]
	}
	return normalized
}
