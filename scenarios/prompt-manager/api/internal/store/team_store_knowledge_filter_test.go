package store

import (
	"context"
	"testing"
)

func TestListTeamCorpus_TopicFilters(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	entries := []KnowledgeEntry{
		{ID: "k1", At: "2026-05-01T00:00:00Z", Caller: "vision-walk", Topic: "research-inbox/audience/foo", Content: "a"},
		{ID: "k2", At: "2026-05-01T00:00:01Z", Caller: "vision-walk", Topic: "research-inbox/hook/bar", Content: "b"},
		{ID: "k3", At: "2026-05-01T00:00:02Z", Caller: "researcher", Topic: "audience-scan/foo", Content: "c"},
		{ID: "k4", At: "2026-05-01T00:00:03Z", Caller: "researcher", Topic: "research-inbox", Content: "d"},
	}
	for i := range entries {
		if err := s.AppendTeamCorpus(ctx, "team-1", &entries[i]); err != nil {
			t.Fatalf("AppendTeamCorpus[%d]: %v", i, err)
		}
	}

	t.Run("no filter returns all", func(t *testing.T) {
		got, err := s.ListTeamCorpus(ctx, "team-1", "", "", 0)
		if err != nil {
			t.Fatalf("ListTeamCorpus: %v", err)
		}
		if len(got) != 4 {
			t.Errorf("want 4 entries, got %d", len(got))
		}
	})

	t.Run("exact topic matches only the literal topic", func(t *testing.T) {
		got, err := s.ListTeamCorpus(ctx, "team-1", "research-inbox", "", 0)
		if err != nil {
			t.Fatalf("ListTeamCorpus: %v", err)
		}
		if len(got) != 1 || got[0].ID != "k4" {
			t.Errorf("want exactly k4, got %+v", got)
		}
	})

	t.Run("topic prefix matches hierarchical entries under the prefix", func(t *testing.T) {
		got, err := s.ListTeamCorpus(ctx, "team-1", "", "research-inbox/", 0)
		if err != nil {
			t.Fatalf("ListTeamCorpus: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 entries (k1, k2), got %d: %+v", len(got), got)
		}
		ids := map[string]bool{got[0].ID: true, got[1].ID: true}
		if !ids["k1"] || !ids["k2"] {
			t.Errorf("want k1 and k2, got %+v", got)
		}
	})

	t.Run("topic prefix without trailing slash also matches the bare topic", func(t *testing.T) {
		got, err := s.ListTeamCorpus(ctx, "team-1", "", "research-inbox", 0)
		if err != nil {
			t.Fatalf("ListTeamCorpus: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("want 3 entries (k1, k2, k4), got %d", len(got))
		}
	})

	t.Run("last caps results to most recent N", func(t *testing.T) {
		got, err := s.ListTeamCorpus(ctx, "team-1", "", "research-inbox/", 1)
		if err != nil {
			t.Fatalf("ListTeamCorpus: %v", err)
		}
		if len(got) != 1 || got[0].ID != "k2" {
			t.Errorf("want most recent prefix-matching entry k2, got %+v", got)
		}
	})
}
