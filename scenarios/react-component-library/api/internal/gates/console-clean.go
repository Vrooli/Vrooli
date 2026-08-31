package gates

func ValidateConsoleClean(scope Scope) (Result, error) {
	root := scope.Root
	result, err := loadStoryEvidence(scope, "console")
	if err != nil {
		return Result{}, err
	}
	if result.Inspected == 0 {
		return unmeasuredStoryGate(root), nil
	}
	return nonEmpty(result, "console-clean"), nil
}
