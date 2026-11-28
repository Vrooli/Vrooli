// Package server wires the production HTTP handler stack: cross-cutting
// middleware + a slice of domain modules. Each domain returns a
// module.Module from its handlers package; main.go passes them in.
// There is no central routes.go and no per-domain field on Deps —
// adding a feature means creating files, not modifying this package.
//
// Constructed once in main.go and exposed via Handler() for both the
// production listener and the httpx test harness
// (internal/testutil/httpx.NewLiveServer).
package server

import (
	"log"
	"net/http"
	"strings"

	"react-component-library/internal/middleware"
	"react-component-library/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Deps holds the cross-cutting interfaces the Server depends on
// regardless of which modules are mounted. Production wires concrete
// implementations (schedule.System(), log.Default()) in main.go; tests
// wire fakes from internal/testutil/mocks.
//
// Per-domain dependencies (database handle, repository services,
// pingers) live inside each module's constructor — Deps is intentionally
// limited to what the middleware stack reads.
type Deps struct {
	Clock  schedule.Clock
	Logger *log.Logger
}

// Server is the wired HTTP application: cross-cutting deps + router
// with all module routes mounted. After New, every module's Mount has
// already been called and Handler() returns the production-ready
// http.Handler.
type Server struct {
	deps   Deps
	router *mux.Router
}

// New builds a Server with logging middleware applied and every module's
// Mount invoked. Logger defaults to log.Default() if nil; Clock has no
// default and is required so the logging middleware never hides its time seam.
//
// The handler test in handlers/health/handler_test.go reproduces a
// stripped-down version of the middleware composition; if you add
// cross-cutting middleware here, mirror it in the test or move the
// composition into a shared helper.
func New(d Deps, modules ...module.Module) *Server {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Clock == nil {
		panic("server.New requires Deps.Clock")
	}
	s := &Server{deps: d, router: mux.NewRouter()}
	s.router.Use(middleware.NewLoggingMiddleware(d.Clock, d.Logger))
	for _, m := range modules {
		m.Mount(s.router)
	}
	return s
}

// Handler returns the production HTTP handler wrapped with the recovery
// middleware. This is what main.go listens on and what
// internal/testutil/httpx.NewLiveServer wraps for handler tests.
func (s *Server) Handler() http.Handler {
	return securityHeaders(handlers.RecoveryHandler()(s.router))
}

// securityHeaders establishes the baseline browser-facing API policy once at
// the router boundary, before any domain handler can write a response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Preview harnesses are deliberately embedded by the scenario UI through
		// its same-origin /preview proxy. Keep every other API response
		// unframeable, while allowing that one trusted embedding relationship.
		if strings.HasPrefix(r.URL.Path, "/preview/") {
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			w.Header().Set("Content-Security-Policy", "frame-ancestors 'self'")
		} else {
			w.Header().Set("X-Frame-Options", "DENY")
		}
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}
