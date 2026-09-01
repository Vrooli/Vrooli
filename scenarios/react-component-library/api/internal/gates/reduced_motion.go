package gates

func ValidateReducedMotion(scope Scope) (Result, error) {
	return validateDifferentialGate(scope, "reduced-motion")
}
