package gates

// ValidateInteraction is the registry seam for the shared component-test
// report's interaction assertions.
func ValidateInteraction(Scope) (Result, error) {
	return delegatedAppearanceResult("interaction"), nil
}
