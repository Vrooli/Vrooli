package operatordecision

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestClipStaysWithinTheByteLimitAndCutsAtAWord(t *testing.T) {
	for _, text := range []string{strings.Repeat("🧭", 300), strings.Repeat("word ", 300), strings.Repeat("x", 300)} {
		got := Clip(text, 200)
		if len(got) > 200 || !utf8.ValidString(got) || !strings.HasSuffix(got, "…") {
			t.Errorf("Clip gave %d bytes, valid=%v: %q", len(got), utf8.ValidString(got), got)
		}
	}
	if got := Clip("  keep the whole sentence ", 200); got != "keep the whole sentence" {
		t.Errorf("short text changed: %q", got)
	}
	if got := Clip("Add the backlog item, then ship it", 26); got != "Add the backlog item…" {
		t.Errorf("cut inside a word or kept the comma: %q", got)
	}
	if got := Clip("abcdef", 2); got != "ab" {
		t.Errorf("a limit below the ellipsis keeps the bytes that fit: %q", got)
	}
}

func TestPluralAndCount(t *testing.T) {
	if Plural(1, "it", "them") != "it" || Plural(0, "it", "them") != "them" {
		t.Error("Plural picks the wrong word")
	}
	if Count(1, "file", "files") != "1 file" || Count(3, "file", "files") != "3 files" {
		t.Error("Count writes the wrong words")
	}
}

func TestFitShrinksAPageUntilItFitsADecisionEvent(t *testing.T) {
	var budgets []int
	page := Fit(1<<20, func(budget int) string {
		budgets = append(budgets, budget)
		return strings.Repeat("<", budget)
	})
	if len(budgets) != 3 || page != "" {
		t.Fatalf("budgets tried %v, page %d bytes: '<' encodes as six bytes, so only the empty page fits", budgets, len(page))
	}
	if page := Fit(100, func(b int) string { return strings.Repeat("x", b) }); len(page) != 100 {
		t.Fatalf("a page that fits was shrunk to %d bytes", len(page))
	}
	if EncodedLen("") != 0 || EncodedLen("<") != 8 {
		t.Fatal("EncodedLen does not measure the payload bytes (\"<\" is sent as a six-byte escape in quotes)")
	}
}
