package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/vrooli/api-core/database"
	"swarm-manager/internal/development"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/transitions"
)

// [REQ:SWM-P0-002]
func TestDevelopmentWorkShapeCannotFallThroughToPlanExecution(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	ctx := context.Background()
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(development.Schema)); err != nil {
		t.Fatal(err)
	}
	repo := development.NewSQLiteRepository(database.NewFromPrimary(db))
	if err := repo.Commit(ctx, 0, development.Engagement{WorkItem: "execute/contract-item", Version: 1, WorkShape: "contract-development", Status: "approved"}, nil); err != nil {
		t.Fatal(err)
	}
	s := &Server{developmentSvc: development.NewService(repo, development.Reviewer{}, nil, nil)}
	definition := transitions.Definition{Key: "plan.execute", Subject: "backlog-item"}
	if err := s.guardDevelopmentWorkShape(ctx, definition, "execute/contract-item"); !errors.Is(err, development.ErrDenied) {
		t.Fatalf("contract item dispatched through a plan: %v", err)
	}
	if err := s.guardDevelopmentWorkShape(ctx, definition, "execute/ordinary-plan"); err != nil {
		t.Fatalf("ordinary plan regressed: %v", err)
	}
	if err := s.guardDevelopmentWorkShape(ctx, transitions.Definition{Key: transitionrunner.ContractDevelopmentTransition, Subject: "backlog-item"}, "execute/contract-item"); err != nil {
		t.Fatalf("contract transition was blocked by the plan guard: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.guardDevelopmentWorkShape(ctx, definition, "execute/contract-item"); err == nil {
		t.Fatal("unreadable authority allowed plan dispatch")
	}
}
