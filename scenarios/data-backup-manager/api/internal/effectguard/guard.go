// Package effectguard prevents production process adapters from being used by
// Go tests or test-mode HTTP requests. There is intentionally no environment
// variable override: tests must inject the existing command/engine seams.
package effectguard

import (
	"context"
	"errors"
	"github.com/vrooli/api-core/database"
	"net/http"
	"testing"
)

func Check(ctx context.Context) error {
	if testing.Testing() || database.IsTestMode(ctx) {
		return errors.New("production backup/source commands are disabled in tests; inject a fake adapter")
	}
	return ctx.Err()
}

// ProtectProduction is installed only on the production composition root.
// Routed databases alone cannot isolate credentials, engine destinations, or
// background jobs. Test RPCs must use a dedicated fixture server with fakes.
func ProtectProduction(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && (database.IsTestMode(r.Context()) || r.Header.Get("X-Vrooli-Test-Mode") == "1") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":"permission_denied","message":"DBM test RPCs require an isolated fixture server with fake backup and source adapters"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
