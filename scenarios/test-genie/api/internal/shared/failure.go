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
	case FailureClassMaturityContract:
		return FailureClassMaturityContract
	case FailureClassTestFailure:
		// Preserve provider-attributed product failures. Collapsing these into
		// system makes a red fleet look like a broken harness and defeats the
		// failure taxonomy used by self-health and focus. Revisit only if the
		// phase wire contract introduces a distinct, richer product-failure
		// category.
		return FailureClassTestFailure
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
