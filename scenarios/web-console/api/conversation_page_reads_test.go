package main

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"web-console/internal/audioports"
)

// pageWindow is the reference semantics of ConversationPageQuery, computed
// from the whole session: the newest Limit events strictly between the bounds.
func pageWindow(all []ConversationEvent, page ConversationPageQuery) ([]int64, bool) {
	var window []int64
	for _, event := range all {
		if (page.BeforeSequence <= 0 || event.Sequence < page.BeforeSequence) && (page.SinceSequence <= 0 || event.Sequence > page.SinceSequence) {
			window = append(window, event.Sequence)
		}
	}
	if page.Limit > 0 && len(window) > page.Limit {
		return window[len(window)-page.Limit:], true
	}
	return window, false
}

func TestConversationPageQueryMatchesTheWholeSessionWindow(t *testing.T) {
	repos := map[string]ConversationRepository{"sql": searchTestRepo(t), "memory": NewInMemoryConversationRepository()}
	queries := []ConversationPageQuery{
		{}, {Limit: 10}, {Limit: 60}, {Limit: 10, BeforeSequence: 31}, {SinceSequence: 45}, {SinceSequence: 50},
		{Limit: 5, SinceSequence: 20}, {Limit: 50, SinceSequence: 20}, {Limit: 5, SinceSequence: 20, BeforeSequence: 30}, {BeforeSequence: 1},
	}
	for name, repo := range repos {
		seedConversationEvents(t, repo, "paged", 50)
		whole, err := repo.ListSession(context.Background(), "paged")
		if err != nil || len(whole.Events) != 50 {
			t.Fatalf("%s: seed = %d events, %v", name, len(whole.Events), err)
		}
		for _, query := range queries {
			page, hasMore, err := repo.ListSessionPage(context.Background(), "paged", query)
			if err != nil {
				t.Fatalf("%s %+v: %v", name, query, err)
			}
			got := make([]int64, 0, len(page.Events))
			for _, event := range page.Events {
				got = append(got, event.Sequence)
			}
			want, wantMore := pageWindow(whole.Events, query)
			if !slices.Equal(got, want) || hasMore != wantMore {
				t.Errorf("%s %+v = %v more=%v, want %v more=%v", name, query, got, hasMore, want, wantMore)
			}
		}
		if page, more, err := repo.ListSessionPage(context.Background(), "unknown", ConversationPageQuery{Limit: 5}); err != nil || len(page.Events) != 0 || more {
			t.Errorf("%s unknown session = %+v more=%v err=%v", name, page, more, err)
		}
	}
}

// fullLoadCounter fails the contract that request-path reads never load a
// whole session: every bounded read path must avoid ListSession.
type fullLoadCounter struct {
	ConversationRepository
	fullLoads int
}

func (c *fullLoadCounter) ListSession(ctx context.Context, sessionID string) (ConversationSessionState, error) {
	c.fullLoads++
	return c.ConversationRepository.ListSession(ctx, sessionID)
}

func TestConversationReadPathsNeverLoadTheWholeSession(t *testing.T) {
	ctx := context.Background()
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(ctx, "", 80, 24, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(ctx, sess.ID) })
	repo := &fullLoadCounter{ConversationRepository: searchTestRepo(t)}
	srv.conversations = NewConversationStoreWithRepository(repo)
	started := time.Now().UTC()
	var lastID string
	for i := 1; i <= 30; i++ {
		event, result := srv.conversations.AppendAssistantEvent(ctx, sess.ID, "test", fmt.Sprintf("reply %d", i))
		if !result.Appended {
			t.Fatalf("append %d: %+v", i, result)
		}
		lastID = event.ID
	}
	seen := int64(12)
	srv.conversations.UpdateCursor(ctx, sess.ID, conversationCursorPatch{seenSequence: &seen})
	adapter := newConversationAdapter(srv)

	first, err := adapter.Get(ctx, sess.ID, 0, 10, 0)
	if err != nil || len(first.Events) != 10 || !first.HasMore || first.OldestSequence != 21 || first.NewestSequence != 30 || first.TotalCount != 30 || first.Cursor.LastSeenSequence != 12 {
		t.Fatalf("first page = %+v err=%v", first, err)
	}
	poll, err := adapter.Get(ctx, sess.ID, 27, 0, 0)
	if err != nil || len(poll.Events) != 3 || poll.Events[0].Sequence != 28 || poll.HasMore || poll.TotalCount != 30 {
		t.Fatalf("since poll = %+v err=%v", poll, err)
	}
	empty, err := adapter.Get(ctx, sess.ID, 30, 0, 0)
	if err != nil || len(empty.Events) != 0 || empty.Cursor.LastSeenSequence != 12 {
		t.Fatalf("empty poll = %+v err=%v", empty, err)
	}
	ranged, err := adapter.GetRange(ctx, sess.ID, 5, 7)
	if err != nil || len(ranged.Events) != 3 || ranged.Cursor.LastSeenSequence != 12 {
		t.Fatalf("range = %+v err=%v", ranged, err)
	}
	srv.summarizer = testSummarizer{out: audioports.SummarizeOutput{Text: "short form"}}
	if summary, err := adapter.SummarizeEvent(ctx, sess.ID, lastID); err != nil || !summary.Summarized {
		t.Fatalf("summarize existing event = %+v, %v", summary, err)
	}
	if _, err := adapter.SummarizeEvent(ctx, sess.ID, "missing"); err == nil {
		t.Fatal("summarize of a missing event must fail")
	}
	if captured := adapter.lastCapturedAt(ctx, sess.ID); captured == "" {
		t.Fatal("lastCapturedAt must report the newest stored event")
	}
	if !srv.conversations.HasConversationAfter(ctx, sess.ID, started.Add(-time.Second)) {
		t.Fatal("events recorded after the instant must be reported")
	}
	if srv.conversations.HasConversationAfter(ctx, sess.ID, time.Now().Add(time.Hour)) {
		t.Fatal("no event is recorded after a future instant")
	}
	if repo.fullLoads != 0 {
		t.Fatalf("bounded read paths loaded the whole session %d times", repo.fullLoads)
	}
}
