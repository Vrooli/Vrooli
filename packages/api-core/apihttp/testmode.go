// Package apihttp provides shared HTTP middleware used across scenario APIs.
//
// The middleware in this package layers on top of any Go http.Handler — it is
// agnostic to whether a handler is hand-rolled REST, gorilla/mux, or a
// Connect-RPC handler.
package apihttp

import (
	"net/http"
	"os"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/projectmeta"
)

// TestModeHeader is the single recognized request header that opts an
// in-flight request into the test-mode routing path. Only the exact value
// "1" turns it on; any other value (including "true", "yes", empty, missing)
// is ignored.
const (
	TestModeHeader = "X-Vrooli-Test-Mode"
	TestModeValue  = "1"
)

// TestModeForceEnableEnv is a documented escape hatch: when set to "1",
// TestModeMiddleware behaves as if the project were in development mode
// regardless of service.json. Intended for CI scenarios where the mode flag
// can't easily be flipped.
const TestModeForceEnableEnv = "VROOLI_TEST_MODE_FORCE_ENABLE"

// TestModeMiddleware wraps next so that requests carrying X-Vrooli-Test-Mode: 1
// are marked as test-mode in the request context. RoutedDB picks up the mark
// and routes the call to the installed test pool (if any).
//
// In production mode, TestModeMiddleware is a no-op pass-through and adds
// zero per-request overhead.
//
// The dev/prod decision is evaluated once at the time of wrapping and frozen
// thereafter. A single per-process VROOLI_TEST_MODE_FORCE_ENABLE override is
// honored for CI use.
func TestModeMiddleware(next http.Handler) http.Handler {
	enabled := projectmeta.IsDevelopment() || os.Getenv(TestModeForceEnableEnv) == "1"
	if !enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(TestModeHeader) == TestModeValue {
			r = r.WithContext(database.WithTestMode(r.Context()))
		}
		next.ServeHTTP(w, r)
	})
}
