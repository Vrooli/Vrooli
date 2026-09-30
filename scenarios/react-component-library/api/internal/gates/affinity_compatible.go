package gates

func ValidateAffinityNotBroaderThanCompatibility(scope Scope) (Result, error) {
	root := scope.Root
	census, err := TokenCensus(root)
	if err != nil {
		return Result{}, err
	}
	return compatibilityGateResult(census, true), nil
}
