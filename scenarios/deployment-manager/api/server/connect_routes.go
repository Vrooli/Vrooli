package server

import "net/http"

// setupRoutes mounts the generated Connect services and the single retained
// REST health probe. Product operations are intentionally absent from the
// Gorilla/mux surface: the proto descriptors and generated handlers are the
// source of truth for every domain operation.
func (s *Server) setupRoutes() {
	s.Router.Use(func(next http.Handler) http.Handler {
		return LoggingMiddlewareWithLogger(s.Logger, next)
	})
	for _, route := range s.ConnectRoutes {
		s.Router.PathPrefix(route.Path).Handler(route.Handler)
	}
	if s.EvidenceHandler != nil && s.EvidencePath != "" {
		s.Router.PathPrefix(s.EvidencePath).Handler(s.EvidenceHandler)
	}
	if s.ProfilesConnectHandler != nil && s.ProfilesConnectPath != "" {
		s.Router.PathPrefix(s.ProfilesConnectPath).Handler(s.ProfilesConnectHandler)
	}
	s.Router.HandleFunc("/health", s.HealthHandler.Health).Methods("GET")
}
