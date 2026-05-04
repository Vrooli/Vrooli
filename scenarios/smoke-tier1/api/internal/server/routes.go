package server

import (
	"smoke-tier1/handlers/health"
	"smoke-tier1/handlers/notes"
	"smoke-tier1/internal/middleware"
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

	// Notes CRUD reference. Mounted as a subrouter so the notes handler
	// owns its full path table (list, create, get-by-id) without
	// reaching back into the server's mux.
	notesHandler := notes.NewHandler(notes.Deps{
		Store:  s.deps.NoteStore,
		Logger: s.deps.Logger,
	})
	s.router.PathPrefix("/api/v1/notes").Handler(notesHandler)
}
