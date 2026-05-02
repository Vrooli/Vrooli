// Package server wires the production HTTP handler stack: middleware,
// routes, and dependencies. Constructed once in main.go and exposed via
// Handler() for both the production listener and the httpx test harness
// (internal/testutil/httpx.NewLiveServer).
package server

import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"{{SCENARIO_ID}}/internal/clock"
	"{{SCENARIO_ID}}/internal/store"
)

// Deps holds the interfaces the Server depends on. Production wires
// concrete implementations (clock.System{}, *sql.DB) in main.go; tests
// wire fakes from internal/testutil/mocks.
type Deps struct {
	Pinger  store.Pinger
	Clock   clock.Clock
	Logger  *log.Logger
	Service string
	Version string
}

// Server is the wired HTTP application: dependencies + router. After
// New, registerRoutes has already been called and Handler() returns the
// production-ready http.Handler.
type Server struct {
	deps   Deps
	router *mux.Router
}

// New builds a Server with routes registered. Logger defaults to
// log.Default() if nil; the rest of Deps must be set explicitly
// (greenfield rule — no hidden seams).
func New(d Deps) *Server {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	s := &Server{deps: d, router: mux.NewRouter()}
	s.registerRoutes()
	return s
}

// Handler returns the production HTTP handler wrapped with the recovery
// middleware. This is what main.go listens on and what
// internal/testutil/httpx.NewLiveServer wraps for handler tests.
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}
