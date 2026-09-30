package testutil

import "testing"

func TestRequireContainsChecksEveryExpectation(t *testing.T) {
	RequireContains(t, "alpha beta", "alpha", "beta")
}
