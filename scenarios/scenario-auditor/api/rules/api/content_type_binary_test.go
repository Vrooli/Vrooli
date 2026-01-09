//go:build ruletests
// +build ruletests

package api

import "testing"

func TestContentTypeBinaryDocCases(t *testing.T) {
	runDocTestsViolations(t, "content_type_binary.go", "api/download.go", CheckBinaryContentTypeHeaders)
}

func TestContentTypeBinaryNosniffDoesNotSatisfy(t *testing.T) {
	code := []byte(`package main
import "net/http"
func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment; filename=\"data.bin\"")
	w.Write([]byte("hi"))
}`)
	violations := CheckBinaryContentTypeHeaders(code, "api/download.go")
	if len(violations) == 0 {
		t.Fatalf("expected violation when only X-Content-Type-Options is set")
	}
}
