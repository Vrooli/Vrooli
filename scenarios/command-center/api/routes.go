package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/health"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
	integrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/integrations/integrations_v1connect"
)

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	healthBuilder := health.New(scenarioName).Version("0.0.1")
	for _, checker := range integrationHealthCheckers(s.integrationRegistry) {
		healthBuilder.Check(checker, health.Optional)
	}
	healthHandler := healthBuilder.Handler()
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

// integrationHealthCheckers projects the registry's cheap liveness tier into
// api-core's standard dependency map. It deliberately does not resolve full
// feature compatibility or metric payloads on the health path.
func integrationHealthCheckers(registry *capreg.Registry) []health.Checker {
	if registry == nil {
		return nil
	}
	definitions := registry.Definitions()
	checkers := make([]health.Checker, 0, len(definitions))
	for _, definition := range definitions {
		id := definition.ID
		checkers = append(checkers, health.CheckerFunc(func(ctx context.Context) health.CheckResult {
			started := time.Now()
			for _, state := range registry.ResolveLiveness(ctx) {
				if state.ID != id {
					continue
				}
				result := health.CheckResult{Name: id, Connected: state.Status == capreg.StatusAvailable, Latency: time.Since(started)}
				if !result.Connected {
					message := state.Message
					if message == "" {
						message = string(state.Status)
					}
					result.Error = fmt.Errorf("%s: %s", state.ReasonCode, message)
				}
				return result
			}
			return health.CheckResult{Name: id, Connected: false, Latency: time.Since(started), Error: fmt.Errorf("integration state was not returned")}
		}))
	}
	return checkers
}
