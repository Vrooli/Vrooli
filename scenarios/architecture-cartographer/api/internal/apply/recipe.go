package apply

import "context"

// Recipe is the seam for refactor recipes (P1 — the registry is empty
// in v0.1). Each recipe knows how to extend a Plan with extra
// operations for a particular pattern (e.g., "extract glue file
// between domains A and B").
type Recipe interface {
	Name() string
	Description() string

	// Applies returns true when the recipe wants to run against the
	// given plan/scenario context.
	Applies(ctx context.Context, plan Plan) bool

	// Extend produces additional operations to append to the plan.
	Extend(ctx context.Context, plan Plan) ([]Operation, error)
}

// RecipeRegistry is the plug-in registry for recipes. v0.1 keeps it
// empty; the type exists so production wiring is stable.
type RecipeRegistry struct {
	recipes []Recipe
}

// NewRecipeRegistry constructs an empty registry. v0.1 does not register
// any recipes; the function exists so handler tests can stand up the
// service with the canonical wiring.
func NewRecipeRegistry(in ...Recipe) *RecipeRegistry {
	out := append([]Recipe(nil), in...)
	return &RecipeRegistry{recipes: out}
}

// All returns the registered recipes (empty in v0.1).
func (r *RecipeRegistry) All() []Recipe {
	out := make([]Recipe, len(r.recipes))
	copy(out, r.recipes)
	return out
}
