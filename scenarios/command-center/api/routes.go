package main

import (
	"github.com/vrooli/api-core/health"
)

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	healthHandler := health.New(scenarioName).Version("0.0.1").Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	s.registerDashboardRoutes()
	s.registerGapRoutes()
	s.registerDebugRoutes()
}
