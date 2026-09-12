//go:build !liverepo

package memberflow

import "testing"

// realPromptManagerStore is deliberately unavailable in the default suite.
// Repository conformance is exercised explicitly with -tags liverepo; unit
// tests in files that also contain live canaries can retain their local cases.
func realPromptManagerStore(t *testing.T) (string, string) {
	t.Helper()
	t.Skip("live repository conformance requires: go test -tags liverepo ./memberflow/...")
	return "", ""
}
