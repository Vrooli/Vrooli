package gates

// ValidateUnit is the registry seam for the version-pinned browser runner.
// The catalog handler supplies the result because it owns BAS and the durable
// component-test report repository.
func ValidateUnit(Scope) (Result, error) {
	return delegatedAppearanceResult("unit"), nil
}
