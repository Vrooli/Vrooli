package themes

import "context"

// Repository is the persistence seam the themes service depends on.
// Built-in themes live here; scenario-derived themes are not persisted.
type Repository interface {
	UpsertBuiltin(ctx context.Context, t Theme) error
	ReplaceBuiltins(ctx context.Context, themes []Theme) error
	GetBuiltin(ctx context.Context, id string) (Theme, error)
	ListBuiltins(ctx context.Context) ([]Theme, error)
	CountBuiltins(ctx context.Context) (int, error)
}
