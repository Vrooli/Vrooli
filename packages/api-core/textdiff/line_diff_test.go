package textdiff_test

import (
	"strings"
	"testing"

	"github.com/vrooli/api-core/textdiff"
)

func TestLineDiffFocusesOnChange(t *testing.T) {
	diff := textdiff.LineDiff("a\nb\nc\nd\ne", "a\nb\nX\nd\ne")
	for _, want := range []string{"- c", "+ X", "  b", "  d"} {
		if !strings.Contains(diff, want) {
			t.Errorf("diff %q does not contain %q", diff, want)
		}
	}
	if strings.Contains(diff, "- a") || strings.Contains(diff, "+ a") {
		t.Fatalf("common prefix leaked into diff body: %q", diff)
	}
}

func TestLineDiffBoundsChangedLinesPerSide(t *testing.T) {
	expected := make([]string, MaxTestLines)
	actual := make([]string, MaxTestLines)
	for i := range expected {
		expected[i] = "old"
		actual[i] = "new"
	}
	diff := textdiff.LineDiff(strings.Join(expected, "\n"), strings.Join(actual, "\n"))
	if got, want := strings.Count(diff, "- old"), textdiff.MaxLineChanges; got != want {
		t.Fatalf("removed lines = %d, want bounded count %d", got, want)
	}
	if got, want := strings.Count(diff, "+ new"), textdiff.MaxLineChanges; got != want {
		t.Fatalf("added lines = %d, want bounded count %d", got, want)
	}
}

const MaxTestLines = 2 * textdiff.MaxLineChanges
