package discovery

// TargetSuggestionIDForTest exposes the stable-id derivation to external
// package tests without making the production identifier part of the API.
func TargetSuggestionIDForTest(locator string) string { return targetSuggestionID(locator) }
