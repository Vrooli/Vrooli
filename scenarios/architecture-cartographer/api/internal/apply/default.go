package apply

// DefaultRecipeRegistry returns the day-one recipe set, which is empty
// in v0.1. The registry shape exists so v0.2's recipes drop in
// without a wire change.
func DefaultRecipeRegistry() *RecipeRegistry {
	return NewRecipeRegistry()
}
