package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"web-console/internal/dbx"
)

// searchCapture records the session search statement so its plan can be read
// with the arguments it ran with.
type searchCapture struct {
	dbx.Handle
	query string
	args  []any
}

func (c *searchCapture) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.Contains(query, "conversation_events_fts") {
		c.query, c.args = query, args
	}
	return c.Handle.QueryContext(ctx, query, args...)
}

// searchOracle applies the authoritative Go matcher to every event of the
// session: what an index-free search would return.
func searchOracle(t *testing.T, events []ConversationEvent, q ConversationSearchQuery) []int64 {
	t.Helper()
	plan, queryError := planSearch(q)
	if queryError != "" {
		t.Fatalf("plan %q: %s", q.Query, queryError)
	}
	var out []int64
	for _, event := range events {
		if (q.Role == "" || string(event.Role) == q.Role) && len(plan.match(event.Text)) > 0 {
			out = append(out, event.Sequence)
		}
	}
	return out
}

// The session search reads one session's rows through its own index and uses
// FTS only as a membership filter. Driving the read from the FTS side probed
// the mirror once per row (4 s for a 5,000-row session) or, once statistics
// exist, read every matching row in every session.
func TestSessionSearchIsDrivenByTheSessionIndexAndMatchesTheOracle(t *testing.T) {
	db := setupTestDB(t)
	if err := ensureConversationFTS(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	capture := &searchCapture{Handle: db}
	repo := NewSQLConversationRepository(capture)
	words := []string{"deploy pipeline", "Deployment notes", "rollback plan", "the deploy-step failed", "unrelated chatter"}
	for _, session := range []string{"target", "neighbour-a", "neighbour-b"} {
		for i := range 60 {
			role := ConversationRoleAssistant
			if i%3 == 0 {
				role = ConversationRoleUser
			}
			text := fmt.Sprintf("%s %d", words[i%len(words)], i)
			if _, err := repo.AppendEvent(context.Background(), ConversationEvent{ID: fmt.Sprintf("%s-%d", session, i), SessionID: session, Role: role, Text: text}); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Statistics change what the planner prefers; the plan must hold with them.
	if _, err := db.Exec(`ANALYZE`); err != nil {
		t.Fatal(err)
	}
	whole, err := repo.ListSession(context.Background(), "target")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []ConversationSearchQuery{
		{Query: "deploy"}, {Query: "deploy", WholeWord: true}, {Query: "Deploy", CaseSensitive: true},
		{Query: "deploy-step"}, {Query: "rollback plan", Role: "assistant"},
		{Query: `deploy\w+`, Mode: ConversationSearchRegex}, {Query: "deploy notes", Mode: ConversationSearchFuzzy},
	} {
		q.Limit = 500
		result, err := repo.SearchSession(context.Background(), "target", q)
		if err != nil || result.Error != "" {
			t.Fatalf("%+v: %v %s", q, err, result.Error)
		}
		got := make([]int64, 0, len(result.Matches))
		for _, match := range result.Matches {
			got = append(got, match.Sequence)
		}
		if want := searchOracle(t, whole.Events, q); !slices.Equal(got, want) || len(want) == 0 {
			t.Fatalf("%+v = %v, want oracle %v", q, got, want)
		}
		rows, err := db.Query("EXPLAIN QUERY PLAN "+capture.query, capture.args...)
		if err != nil {
			t.Fatal(err)
		}
		var plan []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatal(err)
			}
			plan = append(plan, detail)
		}
		rows.Close()
		joined := strings.Join(plan, "\n")
		if !strings.Contains(joined, "SEARCH e USING INDEX idx_conversation_events_session_sequence (session_id=?") || strings.Contains(joined, "SEARCH e USING INTEGER PRIMARY KEY") {
			t.Fatalf("%+v: session search is not driven by the session index:\n%s", q, joined)
		}
	}
}

func TestSearchRangesPerHitAreBounded(t *testing.T) {
	repo := searchTestRepo(t)
	text := strings.Repeat("abcdefghij ", 400)
	if _, err := repo.AppendEvent(context.Background(), ConversationEvent{ID: "long", SessionID: "ranges", Role: ConversationRoleAssistant, Text: text}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []ConversationSearchQuery{
		{Query: ".", Mode: ConversationSearchRegex}, {Query: "a"}, {Query: "abcdefghij", WholeWord: true}, {Query: "abc", Mode: ConversationSearchFuzzy},
	} {
		q.Limit = 10
		result, err := repo.SearchSession(context.Background(), "ranges", q)
		if err != nil || len(result.Matches) != 1 {
			t.Fatalf("%+v = %+v, %v", q, result, err)
		}
		if ranges := result.Matches[0].Ranges; len(ranges) == 0 || len(ranges) > maxSearchRangesPerHit {
			t.Fatalf("%+v: %d ranges %v", q, len(ranges), ranges)
		}
	}
	// Whole-word filtering keeps reading past rejected matches: 50 embedded
	// "abc" precede the only whole word.
	buried := strings.Repeat("xabc ", 50) + "abc"
	if got := regexMatcher(regexp.MustCompile(`(?i)ABC`), true)(buried); !slices.Equal(got, [][2]int{{250, 253}}) {
		t.Fatalf("whole-word match after rejected ones = %v", got)
	}
}
