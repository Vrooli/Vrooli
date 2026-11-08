// Package apihttp provides shared HTTP middleware used across scenario APIs.
//
// The middleware in this package layers on top of any Go http.Handler — it is
// agnostic to whether a handler is hand-rolled REST, gorilla/mux, or a
// Connect-RPC handler.
package apihttp

import (
	"net/http"
	"net/url"
	"os"
	"strings"

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
	// TestModeWebSocketQuery carries the same development-only routing marker
	// for browser WebSockets, whose constructor cannot set request headers.
	TestModeWebSocketQuery = "__vrooli_test_mode"
)

// TestModeForceEnableEnv is a documented escape hatch: when set to "1",
// TestModeMiddleware behaves as if the project were in development mode
// regardless of service.json. Intended for CI scenarios where the mode flag
// can't easily be flipped.
const TestModeForceEnableEnv = "VROOLI_TEST_MODE_FORCE_ENABLE"

// TestModeMiddleware wraps next so that requests carrying X-Vrooli-Test-Mode: 1
// are marked as test-mode in the request context. RoutedDB picks up the mark
// and routes the call to the installed test pool (if any).
// This low-level marker does not pin a lease generation or refuse missing
// storage. SQL/file APIs that retain work or notifications beyond a request
// use the same devrouting.Service for routing RPCs and its
// PinnedTestModeMiddleware to bind both storage legs before dispatch.
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
		if r.Header.Get(TestModeHeader) == TestModeValue || webSocketTestMode(r) {
			r = r.WithContext(database.WithTestMode(r.Context()))
		}
		next.ServeHTTP(w, r)
	})
}

func webSocketTestMode(r *http.Request) bool {
	if r.Method != http.MethodGet || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	upgrade := false
	for _, value := range r.Header.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			upgrade = upgrade || strings.EqualFold(strings.TrimSpace(token), "upgrade")
		}
	}
	if !upgrade {
		return false
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	values := query[TestModeWebSocketQuery]
	return err == nil && len(values) == 1 && values[0] == TestModeValue
}
