// Package testutil contains helpers shared by web-console CLI tests.
package testutil

// ErrorMessage formats a contextual test error consistently. Keeping this
// formatting in the test utility package gives command tests one shared seam
// for setup and handler errors without adding helpers to production code.
func ErrorMessage(err error, context string) string {
	if err == nil {
		return ""
	}
	return context + ": " + err.Error()
}
