package shared

// StandardizeFailureClass converts any failure class to a standard phase failure class.
// This provides a single point of truth for failure class normalization.
func StandardizeFailureClass(fc FailureClass) FailureClass {
	switch fc {
	case FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case FailureClassMissingDependency:
		return FailureClassMissingDependency
	case FailureClassTimeout:
		return FailureClassTimeout
	case FailureClassTestFailure:
		// Map test failures to system for phase reporting
		return FailureClassSystem
	case FailureClassExecution:
		// Map execution failures to system for phase reporting
		return FailureClassSystem
	case FailureClassSystem:
		return FailureClassSystem
	case FailureClassNone:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}

// StandardizeFailureClassString converts any failure class string to a standard FailureClass.
// This is useful for converting from legacy string-based failure classes.
func StandardizeFailureClassString(fc string) FailureClass {
	return StandardizeFailureClass(FailureClass(fc))
}
