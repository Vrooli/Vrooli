package main

import (
	"context"
	"os"
	"path/filepath"
	"prompt-manager/memberflow"
	"prompt-manager/store"
	"testing"
	"time"
)

func TestTeamKnowledgeQuery_StripsWildcardAndDelegates(t *testing.T) {
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")
	teamID := "team-1"
	if err := os.MkdirAll(filepath.Join(storeDir, "teams", teamID, "shared"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	fs := store.NewFileTeamStore(storeDir, storeDir, store.NewFileRelationStore(storeDir))
	now := time.Now().UTC()
	entries := []store.KnowledgeEntry{
		{ID: "1", At: now.Add(-10 * 24 * time.Hour).Format(time.RFC3339), Topic: "research-inbox/audience/foo"},
		{ID: "2", At: now.Format(time.RFC3339), Topic: "research-inbox/audience/bar"},
		{ID: "3", At: now.Format(time.RFC3339), Topic: "audience-scan/baz"}, // routed; should be filtered out
	}
	for i := range entries {
		if err := fs.AppendKnowledge(context.Background(), teamID, &entries[i]); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	q := newTeamKnowledgeQuery(fs)
	if q == nil {
		t.Fatal("expected non-nil query")
	}
	got, err := q.ListUnrouted(teamID, "research-inbox/*")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 unrouted entries, got %d (%+v)", len(got), got)
	}
	for _, e := range got {
		if e.At.IsZero() {
			t.Fatalf("expected non-zero At for entry %s", e.ID)
		}
	}
}

func TestTeamKnowledgeQuery_NilStoreReturnsNil(t *testing.T) {
	q := newTeamKnowledgeQuery(nil)
	if q != nil {
		t.Fatalf("expected nil query for nil store")
	}
}

// Verifies the adapter satisfies the memberflow.KnowledgeQuery interface at
// compile time. The plain assignment fails build if the contract drifts.
var _ memberflow.KnowledgeQuery = (*teamKnowledgeQuery)(nil)
