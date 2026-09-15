package api

import "testing"

// A response-writing handler that sets no security headers normally trips the
// rule — but only in production source, never in a _test.go file.
const insecureHandlerSrc = `package x
import "net/http"
func H(w http.ResponseWriter, r *http.Request) { w.Write([]byte("hi")) }
`

func TestSecurityHeaders_SkipsTestFiles(t *testing.T) {
	rule := &SecurityHeadersRule{}

	prod, err := rule.Check(insecureHandlerSrc, "api/handlers/thing.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prod) == 0 {
		t.Fatal("expected production handler without security headers to be flagged")
	}

	for _, p := range []string{"api/handlers/thing_test.go", "api/internal/server/server_test.go"} {
		got, err := rule.Check(insecureHandlerSrc, p)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", p, err)
		}
		if len(got) != 0 {
			t.Errorf("test file %s must be skipped, got %d violations", p, len(got))
		}
	}
}
