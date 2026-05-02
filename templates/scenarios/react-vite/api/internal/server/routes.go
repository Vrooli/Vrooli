package server

import (
	"{{SCENARIO_ID}}/handlers/health"
	"{{SCENARIO_ID}}/internal/middleware"
)

// registerRoutes wires the production middleware stack and route table.
// The handler test in handlers/health/handler_test.go reproduces a
// stripped-down version of this composition; if you add cross-cutting
// middleware here, mirror it in the test or move the composition into
// a shared helper.
func (s *Server) registerRoutes() {
	s.router.Use(middleware.NewLoggingMiddleware(s.deps.Clock, s.deps.Logger))

	healthHandler := health.NewHandler(health.Deps{
		Pinger:  s.deps.Pinger,
		Service: s.deps.Service,
		Version: s.deps.Version,
	})
	// /health for infrastructure probes; /api/v1/health for clients.
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
}
