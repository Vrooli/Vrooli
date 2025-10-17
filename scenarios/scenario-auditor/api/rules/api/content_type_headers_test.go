//go:build ruletests
// +build ruletests

package api

import "testing"

func TestContentTypeHeadersDocCases(t *testing.T) {
	runDocTestsViolations(t, "content_type_headers.go", "api/handlers.go", CheckContentTypeHeaders)
}
