//go:build ruletests
// +build ruletests

package api

import "testing"

func TestHTTPStatusCodesDocCases(t *testing.T) {
	runDocTestsViolations(t, "http_status_codes.go", "api/main.go", CheckHTTPStatusCodes)
}
