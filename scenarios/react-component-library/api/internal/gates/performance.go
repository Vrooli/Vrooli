package gates

func ValidatePerformance(scope Scope) (Result, error) {
	result, err := loadStoryEvidence(scope, "performance")
	if err != nil {
		return Result{}, err
	}
	return nonEmpty(result, "performance"), nil
}
