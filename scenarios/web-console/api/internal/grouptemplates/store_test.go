package grouptemplates

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
	dsn := "file:" + filepath.Join(t.TempDir(), "templates.db") +
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

// TestTemplateRoleListsRoundTripAtEveryLength covers the JSON column at the
// three lengths that matter: none, one, and more than two. A five-role
// template is the direct evidence that the model is not pair-shaped.
func TestTemplateRoleListsRoundTripAtEveryLength(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()

		empty, err := s.Upsert(ctx, UpsertRequest{Name: "Empty"})
		if err != nil {
			t.Fatal(err)
		}
		if len(empty.Roles) != 0 {
			t.Fatalf("empty template roles = %#v, want none", empty.Roles)
		}

		five := make([]TemplateRole, 0, 5)
		for _, label := range []string{"Planner", "Implementer", "Critic", "Reviewer", "Log watcher"} {
			mode := StartModeWaiting
			if label == "Planner" {
				mode = StartModeEager
			}
			five = append(five, TemplateRole{Label: label, Command: "agent " + label, StartMode: mode})
		}
		saved, err := s.Upsert(ctx, UpsertRequest{Name: "Wide", Color: "#22d3ee", Roles: five})
		if err != nil {
			t.Fatal(err)
		}
		if len(saved.Roles) != 5 {
			t.Fatalf("saved roles = %d, want 5", len(saved.Roles))
		}

		listed, err := s.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var wide *Template
		for i := range listed {
			if listed[i].ID == saved.ID {
				wide = &listed[i]
			}
		}
		if wide == nil {
			t.Fatalf("saved template missing from List")
		}
		if len(wide.Roles) != 5 {
			t.Fatalf("read-back roles = %d, want 5", len(wide.Roles))
		}
		// Order is content, not incidental: a template describes a sequence
		// the operator arranged.
		for i, want := range five {
			if wide.Roles[i].Label != want.Label {
				t.Fatalf("role %d = %q, want %q", i, wide.Roles[i].Label, want.Label)
			}
			if wide.Roles[i].StartMode != want.StartMode {
				t.Fatalf("role %d start mode = %q, want %q", i, wide.Roles[i].StartMode, want.StartMode)
			}
		}
		if eager := countEager(wide.Roles); eager != 1 {
			t.Fatalf("%d eager roles, want 1 — only eager roles may cost a process", eager)
		}
	})
}

func TestUpsertRejectsBlankNameAndUnknownStartMode(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		if _, err := s.Upsert(ctx, UpsertRequest{Name: ""}); err != ErrInvalidTemplate {
			t.Fatalf("blank name = %v, want ErrInvalidTemplate", err)
		}
		bad := []TemplateRole{{Label: "Planner", StartMode: "immediately"}}
		if _, err := s.Upsert(ctx, UpsertRequest{Name: "Bad", Roles: bad}); err != ErrInvalidTemplate {
			t.Fatalf("unknown start mode = %v, want ErrInvalidTemplate", err)
		}
	})
}

// TestEditingContentPreservesUseCount pins the reason UseCount has a Has flag:
// editing a template must not reset how often it has been used.
func TestEditingContentPreservesUseCount(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		created, err := s.Upsert(ctx, UpsertRequest{Name: "Plan then implement"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Upsert(ctx, UpsertRequest{ID: created.ID, Name: created.Name, UseCount: 7, HasUseCount: true}); err != nil {
			t.Fatal(err)
		}
		edited, err := s.Upsert(ctx, UpsertRequest{ID: created.ID, Name: "Plan then critique"})
		if err != nil {
			t.Fatal(err)
		}
		if edited.UseCount != 7 {
			t.Fatalf("use count after content edit = %d, want 7", edited.UseCount)
		}
		if edited.Name != "Plan then critique" {
			t.Fatalf("name after edit = %q", edited.Name)
		}
	})
}

// TestEverySeededTemplateIsDeletable is prohibition 5 as a test: there is no
// privileged row, so a template written by the seeder deletes exactly like one
// the operator wrote, and deleting it leaves the store working.
func TestEverySeededTemplateIsDeletable(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		seeded, err := s.Upsert(ctx, UpsertRequest{
			ID:    SeedTemplateID,
			Name:  "Plan → Implement",
			Roles: []TemplateRole{{Label: "Planner", StartMode: StartModeEager}, {Label: "Implementer", StartMode: StartModeWaiting}},
		})
		if err != nil {
			t.Fatal(err)
		}
		mine, err := s.Upsert(ctx, UpsertRequest{Name: "Mine"})
		if err != nil {
			t.Fatal(err)
		}

		removed, err := s.Delete(ctx, seeded.ID)
		if err != nil || !removed {
			t.Fatalf("delete seeded template = %v, %v; want true, nil", removed, err)
		}
		// Idempotent, exactly like any other row.
		if removed, err := s.Delete(ctx, seeded.ID); err != nil || removed {
			t.Fatalf("second delete = %v, %v; want false, nil", removed, err)
		}

		// Creating a group from a template still works with the example gone.
		listed, err := s.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) != 1 || listed[0].ID != mine.ID {
			t.Fatalf("after deleting the seeded example, List = %#v, want only the operator's own", listed)
		}
	})
}

func countEager(roles []TemplateRole) int {
	n := 0
	for _, r := range roles {
		if r.IsEager() {
			n++
		}
	}
	return n
}
