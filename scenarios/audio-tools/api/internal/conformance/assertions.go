package conformance

// PassedAssertions reports whether every named assertion exists and passed.
// Missing assertions fail closed so callers cannot treat unmeasured evidence
// as a successful conformance result.
func PassedAssertions(run Run, names ...string) bool {
	byName := make(map[string]Assertion, len(run.Assertions))
	for _, assertion := range run.Assertions {
		byName[assertion.Name] = assertion
	}
	for _, name := range names {
		assertion, ok := byName[name]
		if !ok || assertion.Outcome != OutcomePassed {
			return false
		}
	}
	return true
}
