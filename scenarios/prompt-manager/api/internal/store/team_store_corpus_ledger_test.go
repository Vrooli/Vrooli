package store

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"prompt-manager/internal/sourceledger"

	"connectrpc.com/connect"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
)

// fakeJournal mirrors source-ledger ListEntries paging: oldest first, at most
// 500 rows per page, and a next_cursor (the last row id) only when a page is
// full.
type fakeJournal struct {
	journalconnect.UnimplementedJournalServiceHandler
	mu      sync.Mutex
	entries map[string][]*journalv1.Entry
	lists   int
}

func (f *fakeJournal) AppendEntry(_ context.Context, req *connect.Request[journalv1.AppendEntryRequest]) (*connect.Response[journalv1.AppendEntryResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	scope := req.Msg.GetScope()
	entry := &journalv1.Entry{Id: fmt.Sprintf("e%06d", len(f.entries[scope])+1), Body: req.Msg.GetBody(), Kind: req.Msg.GetKind()}
	f.entries[scope] = append(f.entries[scope], entry)
	return connect.NewResponse(&journalv1.AppendEntryResponse{Entry: entry}), nil
}

func (f *fakeJournal) ListEntries(_ context.Context, req *connect.Request[journalv1.ListEntriesRequest]) (*connect.Response[journalv1.ListEntriesResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists++
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("limit must be at most 500"))
	}
	all := f.entries[req.Msg.GetScope()]
	start := 0
	if cursor := req.Msg.GetCursor(); cursor != "" {
		for i, entry := range all {
			if entry.GetId() == cursor {
				start = i + 1
				break
			}
		}
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	page := all[start:end]
	resp := &journalv1.ListEntriesResponse{Entries: page}
	if len(page) == limit {
		resp.NextCursor = page[len(page)-1].GetId()
	}
	return connect.NewResponse(resp), nil
}

func newLedgerBackedTestStore(t *testing.T) (*FileTeamStore, *fakeJournal) {
	t.Helper()
	journal := &fakeJournal{entries: map[string][]*journalv1.Entry{}}
	mux := http.NewServeMux()
	path, handler := journalconnect.NewJournalServiceHandler(journal)
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	s := setupStateTestStore(t)
	s.SetSourceLedger(sourceledger.NewAt(server.URL))
	return s, journal
}

func TestQueryTeamCorpusFollowsLedgerCursorsPastOnePage(t *testing.T) {
	s, journal := newLedgerBackedTestStore(t)
	ctx := context.Background()
	const inbox = 606
	// Interleave non-knowledge rows so the knowledge set spans several
	// 500-row ledger pages, as in the live scenario-qa scope.
	for i := 0; i < inbox; i++ {
		entry := KnowledgeEntry{ID: "knw-" + strconv.Itoa(i), Topic: "bug-inbox/code-defect/b" + strconv.Itoa(i), Content: "c"}
		if err := s.AppendTeamCorpus(ctx, "team-1", &entry); err != nil {
			t.Fatal(err)
		}
		other := KnowledgeEntry{ID: "other-" + strconv.Itoa(i), Topic: "bug-investigation-report/" + strconv.Itoa(i), Content: "r"}
		if err := s.AppendTeamCorpus(ctx, "team-1", &other); err != nil {
			t.Fatal(err)
		}
	}

	all, err := s.ListTeamCorpus(ctx, "team-1", "", "bug-inbox/", 0)
	if err != nil {
		t.Fatalf("ListTeamCorpus: %v", err)
	}
	if len(all) != inbox {
		t.Fatalf("want all %d inbox rows across ledger pages, got %d", inbox, len(all))
	}
	if all[0].ID != "knw-0" || all[len(all)-1].ID != "knw-"+strconv.Itoa(inbox-1) {
		t.Fatalf("want oldest-first order, got first=%s last=%s", all[0].ID, all[len(all)-1].ID)
	}
	if journal.lists < 3 {
		t.Fatalf("want the listing to follow cursors over >=3 pages, got %d list calls", journal.lists)
	}

	limited, err := s.QueryTeamCorpus(ctx, "team-1", KnowledgeListOptions{TopicPrefix: "bug-inbox/", Last: 20})
	if err != nil {
		t.Fatalf("QueryTeamCorpus: %v", err)
	}
	if limited.Total != inbox || len(limited.Entries) != 20 {
		t.Fatalf("want total=%d with 20 most recent entries, got total=%d len=%d", inbox, limited.Total, len(limited.Entries))
	}
	if limited.Entries[19].ID != "knw-"+strconv.Itoa(inbox-1) {
		t.Fatalf("want the most recent entries, got last=%s", limited.Entries[19].ID)
	}
}

func TestQueryTeamCorpusDedupesSupersedingCopies(t *testing.T) {
	s, _ := newLedgerBackedTestStore(t)
	ctx := context.Background()
	for _, entry := range []KnowledgeEntry{
		{ID: "k1", Topic: "bug-inbox/code-defect/a", Content: "a"},
		{ID: "k2", Topic: "bug-inbox/code-defect/b", Content: "b"},
	} {
		entry := entry
		if err := s.AppendTeamCorpus(ctx, "team-1", &entry); err != nil {
			t.Fatal(err)
		}
	}
	// Retag appends a copy with the same id; the inbox view must reflect the
	// latest version only, so the retagged entry leaves the inbox.
	if err := s.UpdateTeamCorpus(ctx, "team-1", "k1", func(k *KnowledgeEntry) { k.Topic = "low-signal/a" }); err != nil {
		t.Fatal(err)
	}
	inbox, err := s.ListTeamCorpus(ctx, "team-1", "", "bug-inbox/", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(inbox) != 1 || inbox[0].ID != "k2" {
		t.Fatalf("want only k2 in inbox after retag, got %+v", inbox)
	}
	all, err := s.ListTeamCorpus(ctx, "team-1", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 distinct entries, got %d: %+v", len(all), all)
	}
}

func TestCloseTeamCorpusHidesEntryUnlessIncludeClosed(t *testing.T) {
	for name, ledger := range map[string]bool{"source-ledger": true, "fixture": false} {
		t.Run(name, func(t *testing.T) {
			var s *FileTeamStore
			if ledger {
				s, _ = newLedgerBackedTestStore(t)
			} else {
				s = setupStateTestStore(t)
			}
			ctx := context.Background()
			for _, entry := range []KnowledgeEntry{
				{ID: "k1", Topic: "bug-inbox/code-defect/a", Content: "a"},
				{ID: "k2", Topic: "bug-inbox/code-defect/b", Content: "b"},
			} {
				entry := entry
				if err := s.AppendTeamCorpus(ctx, "team-1", &entry); err != nil {
					t.Fatal(err)
				}
			}
			closed, err := s.CloseTeamCorpus(ctx, "team-1", KnowledgeClosure{KnowledgeID: "k1", Reason: "routed-to", Ref: "fix/some-item", Actor: "scenario-qa/bug-investigator"})
			if err != nil {
				t.Fatalf("CloseTeamCorpus: %v", err)
			}
			if closed.Closure == nil || closed.Closure.Reason != "routed-to" || closed.Closure.At == "" {
				t.Fatalf("want closure on returned entry, got %+v", closed.Closure)
			}

			open, err := s.QueryTeamCorpus(ctx, "team-1", KnowledgeListOptions{TopicPrefix: "bug-inbox/"})
			if err != nil {
				t.Fatal(err)
			}
			if open.Total != 1 || len(open.Entries) != 1 || open.Entries[0].ID != "k2" {
				t.Fatalf("want closed k1 hidden from default inbox, got %+v", open)
			}

			withClosed, err := s.QueryTeamCorpus(ctx, "team-1", KnowledgeListOptions{TopicPrefix: "bug-inbox/", IncludeClosed: true})
			if err != nil {
				t.Fatal(err)
			}
			if withClosed.Total != 2 {
				t.Fatalf("want 2 entries with include-closed, got %+v", withClosed)
			}
			var k1 *KnowledgeEntry
			for i := range withClosed.Entries {
				if withClosed.Entries[i].ID == "k1" {
					k1 = &withClosed.Entries[i]
				}
			}
			if k1 == nil || k1.Closure == nil || k1.Closure.Ref != "fix/some-item" || k1.Closure.Actor != "scenario-qa/bug-investigator" {
				t.Fatalf("want k1 with its closure when include-closed, got %+v", k1)
			}
			if open.Entries[0].Closure != nil {
				t.Fatalf("open entry must carry no closure")
			}
		})
	}
}

func TestCloseTeamCorpusValidation(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()
	entry := KnowledgeEntry{ID: "k1", Topic: "bug-inbox/code-defect/a", Content: "a"}
	if err := s.AppendTeamCorpus(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}
	cases := []KnowledgeClosure{
		{KnowledgeID: "k1", Reason: "done"},
		{KnowledgeID: "k1", Reason: "duplicate-of"},
		{KnowledgeID: "k1", Reason: "duplicate-of", Ref: "k1"},
		{KnowledgeID: "k1", Reason: "routed-to"},
		{KnowledgeID: "missing", Reason: "fixed"},
	}
	for _, closure := range cases {
		if _, err := s.CloseTeamCorpus(ctx, "team-1", closure); err == nil {
			t.Errorf("want error for %+v", closure)
		}
	}
	if _, err := s.CloseTeamCorpus(ctx, "team-1", KnowledgeClosure{KnowledgeID: "k1", Reason: "Fixed"}); err != nil {
		t.Fatalf("want case-insensitive fixed accepted: %v", err)
	}
}
