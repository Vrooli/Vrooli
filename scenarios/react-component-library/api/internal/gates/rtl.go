package gates

func ValidateRTL(scope Scope) (Result, error) {
	return validateDifferentialGate(scope, "rtl")
}
