package gates

func ValidateNoUtilityClasses(scope Scope) (Result, error) {
	return validateNoUtilityClasses(scope, "utility-class")
}
