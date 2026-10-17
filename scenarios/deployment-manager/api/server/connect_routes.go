package server

import (
	"net/http"

	"deployment-manager/readiness"

	"github.com/gorilla/mux"
)

func registerBundleExportCompatibilityRoute(router *mux.Router, handler http.HandlerFunc) {
	if router != nil && handler != nil {
		router.HandleFunc("/api/v1/bundles/export", handler).Methods("POST")
	}
}

// registerReadinessStateCompatibilityRoute mounts the release-side readiness
// projection consumed by Offer Desk's release ladder. It stays a narrow,
// read-only compatibility seam; the generated Connect ReadinessService
// remains the product operation surface.
func registerReadinessStateCompatibilityRoute(router *mux.Router, handler http.Handler) {
	if router != nil && handler != nil {
		router.Handle("/api/v1/readiness/state", handler).Methods(http.MethodGet)
	}
}

// setupRoutes mounts the generated Connect services and the narrow REST
// compatibility routes that still have external owner consumers. New product
// operations belong in the proto descriptors and generated handlers.
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
	if s.ReleasesHandler != nil {
		s.Router.HandleFunc("/api/v1/release-operations/{operation_id}", s.ReleasesHandler.GetOperation).Methods(http.MethodGet)
		s.Router.HandleFunc("/api/v1/releases/{release_id}/recover", s.ReleasesHandler.Recover).Methods(http.MethodPost)
		s.Router.HandleFunc("/api/v1/releases/{release_id}/reconcile", s.ReleasesHandler.Reconcile).Methods(http.MethodPost)
		s.Router.HandleFunc("/api/v1/releases/{release_id}/dossier", s.ReleasesHandler.Dossier).Methods(http.MethodGet)
		s.Router.HandleFunc("/api/v1/releases/{release_id}/health", s.ReleasesHandler.Health).Methods(http.MethodGet)
	}
	// Offer Desk's release ladder reads the latest release-side readiness
	// projection through this seam. Mount it when the release repository is
	// wired so the consumer no longer receives a 404.
	if s.ReleasesRepo != nil {
		registerReadinessStateCompatibilityRoute(s.Router, readiness.StateHandler(s.ReleasesRepo))
	}
	s.Router.HandleFunc("/health", s.HealthHandler.Health).Methods("GET")
}
