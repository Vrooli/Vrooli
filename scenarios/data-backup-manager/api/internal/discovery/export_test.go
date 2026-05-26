package discovery

// TargetSuggestionIDForTest exposes the unexported stable-id derivation to the
// external _test package so tests can pre-seed a dismissal by the exact id the
// service will compute.
func TargetSuggestionIDForTest(locator string) string { return targetSuggestionID(locator) }
