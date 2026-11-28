package personas

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestPersonaStatusTransitionOwner(t *testing.T) {
	if err := TransitionStatus(StatusActive, StatusArchived); err != nil {
		t.Fatalf("active -> archived: %v", err)
	}
	for _, tc := range []struct {
		name string
		from Status
		to   Status
	}{
		{"terminal reopening", StatusArchived, StatusActive},
		{"unknown source", Status("future"), StatusArchived},
		{"unknown target", StatusActive, Status("future")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := TransitionStatus(tc.from, tc.to); !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("TransitionStatus(%q, %q) = %v, want ErrInvalidTransition", tc.from, tc.to, err)
			}
		})
	}
	persona := Persona{Status: StatusArchived}
	if err := persona.TransitionStatus(StatusActive); err == nil || persona.Status != StatusArchived {
		t.Fatalf("invalid reversal error=%v left status=%q", err, persona.Status)
	}
}

func TestSQLiteRejectsUnknownPersonaStatusOnCreateAndReload(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, nil)
	base := Persona{Kind: KindPersonal, LegalBasis: LegalBasis{SubjectID: "subject", SubjectName: "Ada", BasisType: "test"}, DisplayName: "Ada", Identifiers: []Identifier{{Type: "passport", Value: "P-1"}}, CreatedAt: time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)}
	invalid := base
	invalid.Status = Status("future")
	if _, err := repo.Create(context.Background(), invalid); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Create unknown state error=%v, want ErrInvalidTransition", err)
	}
	created, err := repo.Create(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	// The production CHECK constraint correctly rejects unknown statuses. This
	// test bypasses only that storage guard so it can verify the repository's
	// defensive read behavior for legacy/corrupt persisted rows.
	if _, err := db.ExecContext(context.Background(), `PRAGMA ignore_check_constraints = ON`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `UPDATE personas SET status='future' WHERE id=?`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(context.Background(), created.ID); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Get unknown persisted state error=%v, want ErrInvalidTransition", err)
	}
}
