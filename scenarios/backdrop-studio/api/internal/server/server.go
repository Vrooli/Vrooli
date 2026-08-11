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

	"backdrop-studio/internal/clock"
	"backdrop-studio/internal/middleware"
	"backdrop-studio/internal/module"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Deps holds the cross-cutting interfaces the Server depends on
// regardless of which modules are mounted. Production wires concrete
// implementations (clock.System{}, log.Default()) in main.go; tests
// wire fakes from internal/testutil/mocks.
//
// Per-domain dependencies (database handle, repository services,
// pingers) live inside each module's constructor — Deps is intentionally
// limited to what the middleware stack reads.
type Deps struct {
	Clock  clock.Clock
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
	s.router.Use(middleware.NewSecurityHeadersMiddleware())
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
	return handlers.RecoveryHandler()(s.router)
}
