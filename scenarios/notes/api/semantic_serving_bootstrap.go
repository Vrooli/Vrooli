package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	schema "notes-api/internal/notes"
)

// NotesSemanticServingFactory is the production composition seam for the
// admitted embedding runtime, active-generation catalog, Qdrant serving owner,
// and operation-admission owner. The factory must bind only facts established
// by those owners; it must not use collection, role, or endpoint defaults.
type NotesSemanticServingFactory interface {
	Build(context.Context, *sql.DB) (*schema.SemanticServing, error)
}

// notesSemanticServingFactory is populated only by a concrete owner adapter
// integration. It intentionally remains unset until the shared adapter and
// exact owner/recipe/admission identities are available through governed paths.
var notesSemanticServingFactory NotesSemanticServingFactory

func initializeSemanticServing(ctx context.Context, database *sql.DB, factory NotesSemanticServingFactory) error {
	setSemanticServing(nil)
	if database == nil {
		return errors.New("Notes semantic serving requires its database")
	}
	if factory == nil {
		return errors.New("Notes semantic serving remains disabled: no admitted owner factory supplies active collection, alias, recipe, dimensions, runtime, and operation admission")
	}
	service, err := factory.Build(ctx, database)
	if err != nil {
		return fmt.Errorf("build Notes semantic serving owner: %w", err)
	}
	if service == nil {
		return errors.New("Notes semantic serving factory returned no owner")
	}
	if _, err := service.ValidateActiveGeneration(ctx); err != nil {
		return fmt.Errorf("active Notes semantic generation is not qualified: %w", err)
	}
	setSemanticServing(service)
	return nil
}
