package gates

// ValidateVisual is the registry seam for ui-health's analyzed capture result.
func ValidateVisual(Scope) (Result, error) {
	return delegatedAppearanceResult("visual"), nil
}
