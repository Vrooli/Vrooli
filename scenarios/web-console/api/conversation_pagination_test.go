package main

import (
	"context"
	"fmt"
	"testing"
)

func seedConversationEvents(t *testing.T, repo ConversationRepository, sessionID string, count int) {
	t.Helper()
	for i := 1; i <= count; i++ {
		text := fmt.Sprintf("event %d", i)
		if i <= 100 {
			text += " oldest-only needle"
		}
		if _, err := repo.AppendEvent(context.Background(), ConversationEvent{ID: fmt.Sprintf("%s-%d", sessionID, i), SessionID: sessionID, Role: ConversationRoleAssistant, Text: text}); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}
}

func TestSQLConversationRepositoryPagesAndSearchesWholeHistory(t *testing.T) {
	repo := searchTestRepo(t)
	const sessionID = "large-history"
	seedConversationEvents(t, repo, sessionID, 2500)

	page, more, err := repo.ListSessionPage(context.Background(), sessionID, 500, 0)
	if err != nil || len(page.Events) != 500 || !more || page.Events[0].Sequence != 2001 || page.Events[499].Sequence != 2500 {
		t.Fatalf("newest page = %d events [%d..%d], more=%v, err=%v", len(page.Events), page.Events[0].Sequence, page.Events[len(page.Events)-1].Sequence, more, err)
	}

	seen := map[int64]bool{}
	before := int64(0)
	for {
		current, hasMore, pageErr := repo.ListSessionPage(context.Background(), sessionID, 500, before)
		if pageErr != nil {
			t.Fatal(pageErr)
		}
		for _, event := range current.Events {
			if seen[event.Sequence] {
				t.Fatalf("duplicate sequence %d", event.Sequence)
			}
			seen[event.Sequence] = true
		}
		if !hasMore {
			break
		}
		before = current.Events[0].Sequence
	}
	if len(seen) != 2500 || !seen[1] || !seen[2500] {
		t.Fatalf("backward paging saw %d events; first=%v last=%v", len(seen), seen[1], seen[2500])
	}

	result, err := repo.SearchSession(context.Background(), sessionID, ConversationSearchQuery{Query: "oldest-only needle", Limit: 500})
	if err != nil || result.Truncated || result.Total != 100 || len(result.Matches) != 100 {
		t.Fatalf("search = %d matches total=%d truncated=%v err=%v", len(result.Matches), result.Total, result.Truncated, err)
	}
	if result.Matches[0].Sequence != 1 {
		t.Fatalf("first search result sequence = %d, want 1", result.Matches[0].Sequence)
	}
	whole, err := repo.ListSession(context.Background(), sessionID)
	if err != nil || len(whole.Events) != 2500 {
		t.Fatalf("legacy full history = %d, err=%v", len(whole.Events), err)
	}
}

func TestSQLConversationSearchEscapesLikeWildcards(t *testing.T) {
	repo := searchTestRepo(t)
	const sessionID = "literal-wildcards"
	seedConversationEvents(t, repo, sessionID, 1)
	if _, err := repo.AppendEvent(context.Background(), ConversationEvent{ID: "literal", SessionID: sessionID, Role: ConversationRoleAssistant, Text: "literal 100%_done"}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.SearchSession(context.Background(), sessionID, ConversationSearchQuery{Query: "100%_done", Limit: 10})
	if err != nil || len(result.Matches) != 1 || result.Matches[0].EventID != "literal" {
		t.Fatalf("literal wildcard search matched %#v, err=%v", result.Matches, err)
	}
}
