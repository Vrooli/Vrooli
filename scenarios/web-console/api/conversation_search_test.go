package main

import (
	"context"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"
)

// searchCorpus seeds, when numbered, 2,500 numbered events (the first 100
// carry "oldest-only needle"), then a handful written for each search mode.
func searchCorpus(t *testing.T, repo ConversationRepository, sessionID string, numbered bool) {
	t.Helper()
	if numbered {
		seedConversationEvents(t, repo, sessionID, 2500)
	}
	for _, event := range []struct {
		id   string
		role ConversationRole
		text string
	}{
		{"whole", ConversationRoleAssistant, "The pinned settle loop kept moving the list"},
		{"substring", ConversationRoleAssistant, "unsettled loopholes everywhere"},
		{"caps", ConversationRoleAssistant, "Scroll Anchor rewritten"},
		{"user-turn", ConversationRoleUser, "please fix the settle loop"},
		{"wild", ConversationRoleAssistant, "literal 100%_done marker"},
		{"fuzzy", ConversationRoleAssistant, "virtualizer compensation applied"},
	} {
		if _, err := repo.AppendEvent(context.Background(), ConversationEvent{ID: sessionID + "-" + event.id, SessionID: sessionID, Role: event.role, Text: event.text}); err != nil {
			t.Fatalf("append %s: %v", event.id, err)
		}
	}
}

// searchTestRepo is a SQL repository with the full-text mirror in place, as
// the server runs it.
func searchTestRepo(t *testing.T) *SQLConversationRepository {
	t.Helper()
	db := setupTestDB(t)
	if err := ensureConversationFTS(context.Background(), db); err != nil {
		t.Fatalf("ensure FTS: %v", err)
	}
	return NewSQLConversationRepository(db)
}

func searchIDs(matches []ConversationSearchMatch, sessionID string) []string {
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match.EventID[len(sessionID)+1:])
	}
	return ids
}

// rangeText returns the excerpt text a range covers; ranges are UTF-16 offsets.
func rangeText(match ConversationSearchMatch, index int) string {
	units := utf16.Encode([]rune(match.Excerpt))
	r := match.Ranges[index]
	return string(utf16.Decode(units[r.Start:r.End]))
}

// [REQ:P0-017g] One search path over the full-text mirror: text, regex, and
// fuzzy modes with case, whole-word, and role options, for sessions and the
// archive alike. One seeded database serves every case.
func TestConversationSearch(t *testing.T) {
	repo := searchTestRepo(t)
	searchCorpus(t, repo, "s", true)
	searchCorpus(t, repo, "gone", false)
	search := func(t *testing.T, query ConversationSearchQuery) ConversationSearchResult {
		t.Helper()
		result, err := repo.SearchSession(context.Background(), "s", query)
		if err != nil {
			t.Fatalf("search %+v: %v", query, err)
		}
		return result
	}

	t.Run("text finds a phrase anywhere in the history, oldest first", func(t *testing.T) {
		result := search(t, ConversationSearchQuery{Query: "oldest-only needle"})
		if result.Error != "" || result.Total != 100 || len(result.Matches) != 100 || result.Truncated {
			t.Fatalf("result = total %d, %d matches, truncated %v, error %q", result.Total, len(result.Matches), result.Truncated, result.Error)
		}
		if result.Matches[0].Sequence != 1 || rangeText(result.Matches[0], 0) != "oldest-only needle" {
			t.Fatalf("first match = seq %d ranges %v in %q", result.Matches[0].Sequence, result.Matches[0].Ranges, result.Matches[0].Excerpt)
		}
		for index := 1; index < len(result.Matches); index++ {
			if result.Matches[index].Sequence <= result.Matches[index-1].Sequence {
				t.Fatalf("matches are not in sequence order at %d", index)
			}
		}
	})

	t.Run("text matches word prefixes; whole word keeps whole words", func(t *testing.T) {
		if got := searchIDs(search(t, ConversationSearchQuery{Query: "loop"}).Matches, "s"); !slices.Equal(got, []string{"whole", "substring", "user-turn"}) {
			t.Fatalf("text 'loop' = %v, want whole, substring, user-turn", got)
		}
		if got := searchIDs(search(t, ConversationSearchQuery{Query: "loop", WholeWord: true}).Matches, "s"); !slices.Equal(got, []string{"whole", "user-turn"}) {
			t.Fatalf("whole-word 'loop' = %v, want whole, user-turn", got)
		}
	})

	t.Run("case-sensitive search post-filters the case-folded index", func(t *testing.T) {
		if got := searchIDs(search(t, ConversationSearchQuery{Query: "scroll anchor"}).Matches, "s"); !slices.Equal(got, []string{"caps"}) {
			t.Fatalf("insensitive = %v", got)
		}
		if got := search(t, ConversationSearchQuery{Query: "scroll anchor", CaseSensitive: true}); len(got.Matches) != 0 {
			t.Fatalf("case-sensitive lower-case query matched %v", searchIDs(got.Matches, "s"))
		}
		if got := searchIDs(search(t, ConversationSearchQuery{Query: "Scroll Anchor", CaseSensitive: true}).Matches, "s"); !slices.Equal(got, []string{"caps"}) {
			t.Fatalf("case-sensitive exact = %v", got)
		}
	})

	t.Run("regex mode matches and reports ranges", func(t *testing.T) {
		result := search(t, ConversationSearchQuery{Query: `settle\s+loop`, Mode: ConversationSearchRegex})
		if got := searchIDs(result.Matches, "s"); !slices.Equal(got, []string{"whole", "user-turn"}) {
			t.Fatalf("regex = %v, want whole, user-turn", got)
		}
		if rangeText(result.Matches[0], 0) != "settle loop" {
			t.Fatalf("regex range covers %q", rangeText(result.Matches[0], 0))
		}
	})

	t.Run("an invalid regex is a readable answer, not a transport error", func(t *testing.T) {
		result, err := repo.SearchSession(context.Background(), "s", ConversationSearchQuery{Query: `(`, Mode: ConversationSearchRegex})
		if err != nil || result.Error == "" || len(result.Matches) != 0 {
			t.Fatalf("invalid regex = %+v, err %v; want an error message and no matches", result, err)
		}
		// Failing value before the fix, seen live: "invalid regular
		// expression: missing closing ): `(?im)(`" — our flags, not the query.
		if strings.Contains(result.Error, "(?") || !strings.Contains(result.Error, "`(`") {
			t.Fatalf("error %q should quote the user's pattern only", result.Error)
		}
	})

	t.Run("a regex with no literal to narrow by scans a bounded window", func(t *testing.T) {
		restore := searchScanCap
		searchScanCap = 1000
		t.Cleanup(func() { searchScanCap = restore })
		result := search(t, ConversationSearchQuery{Query: `\d{4}`, Mode: ConversationSearchRegex})
		if !result.Truncated {
			t.Fatal("regex over 2,506 rows with a 1,000-row cap was not truncated")
		}
		for _, match := range result.Matches {
			if match.Sequence <= 2506-1000 {
				t.Fatalf("scan reached sequence %d, beyond the newest 1,000 rows", match.Sequence)
			}
		}
	})

	t.Run("fuzzy mode matches word prefixes near each other", func(t *testing.T) {
		if got := searchIDs(search(t, ConversationSearchQuery{Query: "virtual comp", Mode: ConversationSearchFuzzy}).Matches, "s"); !slices.Equal(got, []string{"fuzzy"}) {
			t.Fatalf("fuzzy = %v, want fuzzy", got)
		}
	})

	t.Run("the role filter narrows to one speaker", func(t *testing.T) {
		if got := searchIDs(search(t, ConversationSearchQuery{Query: "settle loop", Role: string(ConversationRoleUser)}).Matches, "s"); !slices.Equal(got, []string{"user-turn"}) {
			t.Fatalf("user-only = %v", got)
		}
	})

	t.Run("punctuation is matched literally, even a punctuation-only query", func(t *testing.T) {
		if got := searchIDs(search(t, ConversationSearchQuery{Query: "100%_done"}).Matches, "s"); !slices.Equal(got, []string{"wild"}) {
			t.Fatalf("'100%%_done' = %v", got)
		}
		if got := searchIDs(search(t, ConversationSearchQuery{Query: "%_"}).Matches, "s"); !slices.Equal(got, []string{"wild"}) {
			t.Fatalf("punctuation-only '%%_' = %v", got)
		}
	})

	t.Run("archived search runs the same matching", func(t *testing.T) {
		query := ConversationSearchQuery{Query: "settle loop", WholeWord: true}
		live, err := repo.SearchSession(context.Background(), "gone", query)
		if err != nil {
			t.Fatal(err)
		}
		// No session row: an orphaned projection is archive-searchable.
		archived, err := repo.SearchArchived(context.Background(), ArchivedConversationSearchFilter{ConversationSearchQuery: query})
		if err != nil || archived.Error != "" {
			t.Fatalf("archived search: %v %q", err, archived.Error)
		}
		var got []string
		for _, match := range archived.Matches {
			if match.SessionID == "gone" {
				got = append(got, match.EventID[len("gone")+1:])
			}
		}
		want := searchIDs(live.Matches, "gone")
		slices.Sort(got)
		slices.Sort(want)
		if len(want) == 0 || !slices.Equal(got, want) {
			t.Fatalf("archived = %v, live = %v; want the same matches", got, want)
		}
	})
}
