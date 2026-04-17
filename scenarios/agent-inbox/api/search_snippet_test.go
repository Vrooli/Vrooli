package main

import (
	"agent-inbox/persistence"
	"fmt"
	"testing"
)

func TestExtractSnippet_CentersOnMatch(t *testing.T) {
	// Test extractSnippet directly with a long line
	line := "This is a very long line that talks about many different things and covers topics from A to Z and finally mentions the word skill somewhere in the middle of this long text and then continues with more content after the match"

	// Find "skill" in the line
	matchStart := 0
	for i := 0; i+5 <= len(line); i++ {
		if line[i:i+5] == "skill" {
			matchStart = i
			break
		}
	}
	matchEnd := matchStart + 5

	t.Logf("Line length: %d, match at [%d:%d]", len(line), matchStart, matchEnd)

	snippet, snipStart, snipEnd := persistence.ExtractSnippet(line, matchStart, matchEnd)

	t.Logf("Snippet: %q (len=%d)", snippet, len(snippet))
	t.Logf("SnipStart=%d SnipEnd=%d", snipStart, snipEnd)

	// Snippet should contain "skill"
	if snipStart >= 0 && snipEnd <= len(snippet) && snipStart < snipEnd {
		highlighted := snippet[snipStart:snipEnd]
		if highlighted != "skill" {
			t.Errorf("Expected highlighted 'skill', got %q", highlighted)
		}
	} else {
		t.Errorf("Invalid snippet range: start=%d end=%d len=%d", snipStart, snipEnd, len(snippet))
	}

	// Snippet should NOT start with "This" (the beginning)
	if len(snippet) > 4 && snippet[:4] == "This" {
		t.Error("Snippet starts at line beginning instead of centering on match")
	}

	// Should have ... prefix since match isn't at start
	if snippet[:3] != "..." {
		t.Errorf("Expected ... prefix, got %q", snippet[:10])
	}

	fmt.Printf("extractSnippet test: snippet=%q start=%d end=%d\n", snippet, snipStart, snipEnd)
}

func TestExtractSnippet_ShortLine_MatchNearEnd(t *testing.T) {
	// Lines under the old threshold (100 chars) were returned in full.
	// Combined with CSS truncation in the sidebar (~30-40 visible chars),
	// the match highlight at the end was invisible to users.
	line := "This is a normal chat message about various topics ending with the keyword skill here"
	matchStart := 75 // "skill" starts at position 75
	matchEnd := 80

	// Verify our match position
	if line[matchStart:matchEnd] != "skill" {
		t.Fatalf("Test setup error: expected 'skill' at [%d:%d], got %q", matchStart, matchEnd, line[matchStart:matchEnd])
	}
	t.Logf("Line length: %d, match at [%d:%d]", len(line), matchStart, matchEnd)

	snippet, snipStart, snipEnd := persistence.ExtractSnippet(line, matchStart, matchEnd)
	t.Logf("Snippet: %q (len=%d), highlight=[%d:%d]", snippet, len(snippet), snipStart, snipEnd)

	// The snippet must be short enough to display in the sidebar without
	// CSS truncation hiding the match. Max ~60 chars.
	if len(snippet) > 60 {
		t.Errorf("Snippet too long for sidebar display: %d chars (max 60). Full snippet: %q", len(snippet), snippet)
	}

	// The match must be present in the snippet
	if snipStart < 0 || snipEnd > len(snippet) || snipStart >= snipEnd {
		t.Fatalf("Invalid snippet range: start=%d end=%d len=%d", snipStart, snipEnd, len(snippet))
	}
	highlighted := snippet[snipStart:snipEnd]
	if highlighted != "skill" {
		t.Errorf("Expected highlighted 'skill', got %q", highlighted)
	}
}
