// Package server wires the production HTTP handler stack: cross-cutting
// middleware + a slice of domain modules. Each domain returns a
// modulekit.Module from its handlers package; main.go passes them in.
// There is no central routes.go and no per-domain field on Deps —
// adding a feature means creating files, not modifying this package.
//
// Constructed once in main.go and exposed via Handler() for both the
// production listener and the httpx test harness
// (internal/testutil/httpx.NewLiveServer).
package server

import (
	"net/http"

	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
	"audio-tools/internal/middleware"
	"audio-tools/internal/modulekit"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Deps holds the cross-cutting interfaces the Server depends on
// regardless of which modules are mounted. Production wires concrete
// implementations (clock.System{}, logx.Std{...}) in main.go; tests
// wire fakes from internal/testutil/mocks. Both fields are required.
type Deps struct {
	Clock  clock.Clock
	Logger logx.Logger
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
// Mount invoked. Both Deps.Clock and Deps.Logger are required; nil
// panics so a forgotten wire-up surfaces at boot, not at request-time.
//
// The handler test in handlers/health/handler_test.go reproduces a
// stripped-down version of the middleware composition; if you add
// cross-cutting middleware here, mirror it in the test or move the
// composition into a shared helper.
func New(d Deps, modules ...modulekit.Module) *Server {
	if d.Clock == nil {
		panic("server.New requires Deps.Clock")
	}
	if d.Logger == nil {
		panic("server.New requires Deps.Logger")
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
	return handlers.RecoveryHandler()(s.router)
}
