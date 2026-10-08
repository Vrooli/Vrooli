package gates

func ValidateForeignTokenClasses(scope Scope) (Result, error) {
	result, err := validateNoUtilityClasses(scope, "foreign-token-classes")
	return result, err
}

type utilityClassAllowance struct {
	SchemaVersion int `json:"schemaVersion"`
	Expires       string
	Entries       []struct {
		Path     string
		Reason   string
		ClosedBy string
	}
}

// ValidateNoUtilityClasses enforces the package portability boundary across
// every released source file. The allowlist is explicit migration debt and is
// surfaced as informational evidence; any unlisted hit is blocking.
