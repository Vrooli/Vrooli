//go:build ruletests
// +build ruletests

package api

import "testing"

func TestContentTypeBinaryDocCases(t *testing.T) {
	runDocTestsViolations(t, "content_type_binary.go", "api/download.go", CheckBinaryContentTypeHeaders)
}
