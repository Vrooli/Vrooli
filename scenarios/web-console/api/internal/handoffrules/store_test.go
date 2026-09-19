package handoffrules

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestSQLStore(t *testing.T) *SQLStore {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "rules.db") +
		"?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile(filepath.Join("..", "sessions", "schema.sql"))
	if err != nil {
		t.Fatalf("read schema.sql: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("exec schema.sql: %v", err)
	}
	return NewSQLStore(db)
}

func eachStore(t *testing.T, run func(t *testing.T, s Store)) {
	t.Helper()
	t.Run("mem", func(t *testing.T) { run(t, NewMemStore()) })
	t.Run("sql", func(t *testing.T) { run(t, newTestSQLStore(t)) })
}

func TestRuleRoundTrip(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		saved, err := s.Upsert(ctx, UpsertRequest{
			Name:     "Plan file",
			Enabled:  true,
			Source:   SourceFilePath,
			Pattern:  "**/.vrooli/plans/*.md",
			Surfaces: []string{"messages"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if saved.ID == "" {
			t.Fatal("a blank id should be assigned by the store")
		}

		listed, err := s.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) != 1 {
			t.Fatalf("List = %d rules, want 1", len(listed))
		}
		got := listed[0]
		if !got.Enabled || got.Source != SourceFilePath || got.Pattern != "**/.vrooli/plans/*.md" {
			t.Fatalf("round-tripped rule = %#v", got)
		}
		if len(got.Surfaces) != 1 || got.Surfaces[0] != "messages" {
			t.Fatalf("surfaces = %#v, want [messages]", got.Surfaces)
		}

		// Disabling is an ordinary update, and it must actually persist —
		// a rule that cannot be turned off is a rule the operator cannot
		// escape.
		off, err := s.Upsert(ctx, UpsertRequest{ID: got.ID, Name: got.Name, Source: got.Source, Pattern: got.Pattern, Enabled: false})
		if err != nil {
			t.Fatal(err)
		}
		if off.Enabled {
			t.Fatal("rule stayed enabled after being turned off")
		}
	})
}

func TestUpsertRejectsInvalidRules(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		cases := map[string]UpsertRequest{
			"blank name":     {Name: "", Source: SourceFilePath, Pattern: "*.md"},
			"blank pattern":  {Name: "Anything", Source: SourceFilePath, Pattern: ""},
			"unknown source": {Name: "Anything", Source: "terminal_output", Pattern: "*.md"},
		}
		for label, req := range cases {
			if _, err := s.Upsert(ctx, req); err != ErrInvalidRule {
				t.Fatalf("%s = %v, want ErrInvalidRule", label, err)
			}
		}
	})
}

// TestEverySeededRuleIsDeletable is prohibition 5 as a test, and it also
// covers the capability claim: with every rule gone the store still works,
// which is what the UI's remaining handoff entry points depend on.
func TestEverySeededRuleIsDeletable(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		if err := SeedExamples(ctx, s); err != nil {
			t.Fatal(err)
		}
		listed, err := s.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) != 1 || listed[0].ID != SeedRuleID {
			t.Fatalf("after seeding, List = %#v, want the one example", listed)
		}

		removed, err := s.Delete(ctx, SeedRuleID)
		if err != nil || !removed {
			t.Fatalf("delete seeded rule = %v, %v; want true, nil", removed, err)
		}
		if removed, err := s.Delete(ctx, SeedRuleID); err != nil || removed {
			t.Fatalf("second delete = %v, %v; want false, nil", removed, err)
		}

		// An empty rule list is a valid, working state — it is what an
		// operator who wants no suggestions gets.
		listed, err = s.List(ctx)
		if err != nil {
			t.Fatalf("List after deleting every rule failed: %v", err)
		}
		if len(listed) != 0 {
			t.Fatalf("List = %#v, want empty", listed)
		}
	})
}

// TestSeedingSkipsANonEmptyStore is what makes the deletion permanent: a later
// boot must not resurrect an example the operator removed.
func TestSeedingSkipsANonEmptyStore(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		if _, err := s.Upsert(ctx, UpsertRequest{Name: "Mine", Source: SourceMessageText, Pattern: "TODO: (.+)"}); err != nil {
			t.Fatal(err)
		}
		if err := SeedExamples(ctx, s); err != nil {
			t.Fatal(err)
		}
		listed, err := s.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) != 1 || listed[0].Name != "Mine" {
			t.Fatalf("seeding into a non-empty store added rows: %#v", listed)
		}
	})
}
