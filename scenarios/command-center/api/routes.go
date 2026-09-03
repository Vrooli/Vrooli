package main

import (
	"net/http"

	"github.com/vrooli/api-core/health"
	integrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/integrations/integrations_v1connect"
)

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	healthHandler := health.New(scenarioName).Version("0.0.1").Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	s.registerDashboardRoutes()
	s.registerGapRoutes()
	s.registerBoardRoutes()
	s.registerDebugRoutes()
	s.registerIntegrationRoutes()
	path, handler := integrationconnect.NewIntegrationsServiceHandler(integrationsConnectService{server: s})
	s.router.PathPrefix(path).Handler(http.HandlerFunc(handler.ServeHTTP))
}
