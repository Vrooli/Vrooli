package gates

func ValidateConsoleClean(scope Scope) (Result, error) {
	result, err := loadStoryEvidence(scope, "console")
	if err != nil {
		return Result{}, err
	}
	if result.Inspected == 0 {
		return unmeasuredStoryGate(scope), nil
	}
	return nonEmpty(result, "console-clean"), nil
}
