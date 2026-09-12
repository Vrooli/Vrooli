package gates

func ValidateKitCompatibility(scope Scope) (Result, error) {
	root := scope.Root
	census, err := TokenCensus(root)
	if err != nil {
		return Result{}, err
	}
	return compatibilityGateResult(census, false), nil
}

// ValidateAffinityNotBroaderThanCompatibility keeps authored aesthetic fit
// inside the objectively renderable kit set.
