package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-inbox/persistence"
)

func TestSearchChats_MultipleResultsPerChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create a chat with multiple messages containing the search term
	chat := createTestChat(t, ts)

	// Add messages with the search term "skill" appearing in different messages
	addTestMessage(t, ts, chat.ID, "user", "Tell me about the skill system")
	addTestMessage(t, ts, chat.ID, "assistant", "The skill system manages various skill definitions. Each skill has a name and description.")
	addTestMessage(t, ts, chat.ID, "user", "How do I create a new skill?")

	// Search with per_chat=5 — should find multiple matches
	req := httptest.NewRequest("GET", "/api/v1/search?q=skill&per_chat=5", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var results []persistence.SearchResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to parse results: %v", err)
	}

	// Count message_content matches (not chat_name)
	contentMatches := 0
	for _, r := range results {
		if r.MatchType == "message_content" {
			contentMatches++
			t.Logf("Content match: message_id=%s snippet=%q match_start=%d match_end=%d",
				r.MessageID, r.Snippet, r.MatchStart, r.MatchEnd)
		}
	}

	if contentMatches < 3 {
		t.Errorf("Expected at least 3 content matches (skill appears in 3 messages), got %d", contentMatches)
	}
}

func TestSearchChats_SnippetCentersOnMatch(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Create a long message where "skill" appears far from the beginning
	longPrefix := "This is a very long message that talks about many different things in great detail, " +
		"covering topics from A to Z and everything in between. It discusses architecture, " +
		"builds, configurations, deployments, environments, and frameworks. After all that, " +
		"it finally mentions the word skill in this part of the text. And then continues with more text after."

	addTestMessage(t, ts, chat.ID, "user", longPrefix)

	req := httptest.NewRequest("GET", "/api/v1/search?q=skill&per_chat=5", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var results []persistence.SearchResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to parse results: %v", err)
	}

	// Find the content match
	for _, r := range results {
		if r.MatchType == "message_content" {
			t.Logf("Snippet: %q (len=%d)", r.Snippet, len(r.Snippet))
			t.Logf("MatchStart=%d MatchEnd=%d", r.MatchStart, r.MatchEnd)

			// The snippet should contain the match term
			if r.MatchStart >= r.MatchEnd {
				t.Error("MatchStart >= MatchEnd, match range is invalid")
			}
			if r.MatchEnd > len(r.Snippet) {
				t.Errorf("MatchEnd (%d) > snippet length (%d)", r.MatchEnd, len(r.Snippet))
			}

			// Extract the highlighted portion
			if r.MatchStart < len(r.Snippet) && r.MatchEnd <= len(r.Snippet) {
				highlighted := r.Snippet[r.MatchStart:r.MatchEnd]
				t.Logf("Highlighted: %q", highlighted)
				if highlighted != "skill" {
					t.Errorf("Expected highlighted text to be 'skill', got %q", highlighted)
				}
			}

			// The snippet should NOT start with "This is a very long" (the beginning of the message)
			if len(r.Snippet) < len(longPrefix) && r.Snippet[:4] == "This" {
				t.Error("Snippet shows the beginning of the message instead of centering on the match")
			}

			// The snippet should be reasonably short (around 80-100 chars with ... prefix/suffix)
			if len(r.Snippet) > 120 {
				t.Errorf("Snippet too long (%d chars), expected ~80-100 chars", len(r.Snippet))
			}
		}
	}
}

func TestSearchChats_MatchStartEndCorrect(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	addTestMessage(t, ts, chat.ID, "user", "hello world skill test")

	req := httptest.NewRequest("GET", "/api/v1/search?q=skill", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var results []persistence.SearchResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to parse results: %v", err)
	}

	found := false
	for _, r := range results {
		if r.MatchType == "message_content" {
			found = true
			t.Logf("snippet=%q start=%d end=%d", r.Snippet, r.MatchStart, r.MatchEnd)

			// The extracted match should be "skill"
			if r.MatchStart >= 0 && r.MatchEnd <= len(r.Snippet) && r.MatchStart < r.MatchEnd {
				highlighted := r.Snippet[r.MatchStart:r.MatchEnd]
				if highlighted != "skill" {
					t.Errorf("Expected match text 'skill', got %q", highlighted)
				}
			} else {
				t.Errorf("Invalid match range: start=%d end=%d snippet_len=%d", r.MatchStart, r.MatchEnd, len(r.Snippet))
			}
		}
	}

	if !found {
		t.Error("No message_content match found")
		for i, r := range results {
			t.Logf("Result %d: type=%s snippet=%q", i, r.MatchType, r.Snippet)
		}
	}
}

func TestSearchChats_MultipleMatchesSameLine(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Single message with "skill" appearing multiple times on the same line (no newlines)
	addTestMessage(t, ts, chat.ID, "user",
		"The skill system is great. Every skill can be composed with other skill definitions to create advanced skill workflows.")

	req := httptest.NewRequest("GET", "/api/v1/search?q=skill&per_chat=5", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var results []persistence.SearchResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to parse results: %v", err)
	}

	// Count message_content matches
	contentMatches := 0
	for _, r := range results {
		if r.MatchType == "message_content" {
			contentMatches++
			t.Logf("Match %d: snippet=%q start=%d end=%d", contentMatches, r.Snippet, r.MatchStart, r.MatchEnd)
			// Verify match range is valid and extracts "skill"
			if r.MatchStart >= 0 && r.MatchEnd <= len(r.Snippet) && r.MatchStart < r.MatchEnd {
				highlighted := r.Snippet[r.MatchStart:r.MatchEnd]
				if highlighted != "skill" {
					t.Errorf("Match %d: expected 'skill', got %q", contentMatches, highlighted)
				}
			}
		}
	}

	// "skill" appears 4 times in the message, perChat=5, so we should get 4 matches
	if contentMatches != 4 {
		t.Errorf("Expected 4 content matches (skill appears 4 times), got %d", contentMatches)
	}
}

func TestSearchChats_MatchStartNeverOmitted(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Message where the match is at position 0
	addTestMessage(t, ts, chat.ID, "user", "skill is at the beginning")

	req := httptest.NewRequest("GET", "/api/v1/search?q=skill", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Parse raw JSON to check that match_start is present (not omitted)
	var rawResults []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &rawResults); err != nil {
		t.Fatalf("Failed to parse results: %v", err)
	}

	for _, r := range rawResults {
		if r["match_type"] == "message_content" {
			// match_start should be present in JSON even when 0
			if _, ok := r["match_start"]; !ok {
				t.Error("match_start is missing from JSON response (omitempty bug)")
			}
			if _, ok := r["match_end"]; !ok {
				t.Error("match_end is missing from JSON response (omitempty bug)")
			}
			matchStart := int(r["match_start"].(float64))
			matchEnd := int(r["match_end"].(float64))
			t.Logf("match_start=%d match_end=%d snippet=%v", matchStart, matchEnd, r["snippet"])
			if matchStart != 0 {
				t.Errorf("Expected match_start=0 for 'skill' at beginning, got %d", matchStart)
			}
		}
	}
}

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
