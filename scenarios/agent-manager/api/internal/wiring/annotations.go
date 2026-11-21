package wiring

import (
	"context"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/runannotations"
	"agent-manager/internal/sqlcompat"
	"agent-manager/internal/toolfacts"
)

// NewRunAnnotations composes the run-annotation service from the scenario's
// canonical pack and its durable state root (where the database lives, and so
// where gold lives). Resolving that root is wiring's job: the engine receives
// it, it never looks it up.
func NewRunAnnotations(ctx context.Context, db sqlcompat.DB, getenv func(string) string, parsers toolfacts.ParserResolver) (*runannotations.Service, error) {
	packPath, err := runannotations.DefaultPackPath(getenv)
	if err != nil {
		return nil, err
	}
	return runannotations.NewDefaultService(ctx, db, runannotations.DefaultOptions{
		PackPath: packPath, StateRoot: database.DataDir(), Parsers: parsers,
	})
}
