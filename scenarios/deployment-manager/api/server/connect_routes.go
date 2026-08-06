package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func registerBundleExportCompatibilityRoute(router *mux.Router, handler http.HandlerFunc) {
	if router != nil && handler != nil {
		router.HandleFunc("/api/v1/bundles/export", handler).Methods("POST")
	}
}

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
	// scenario-to-desktop still consumes the bundle exporter as a small
	// producer/consumer seam while the rest of deployment-manager is Connect.
	// Keep this compatibility route explicit and narrow; it does not re-open
	// the retired REST surface for product operations.
	if s.BundlesHandler != nil {
		registerBundleExportCompatibilityRoute(s.Router, s.BundlesHandler.ExportBundle)
	}
	s.Router.HandleFunc("/health", s.HealthHandler.Health).Methods("GET")
}
