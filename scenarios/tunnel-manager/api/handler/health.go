package handler

import (
	"context"
	"net/http"
	"time"

	"tunnel-manager/domain"
)

// HandleDetailedHealth returns composite tunnel health including per-route status.
// Designed for consumption by other Vrooli scenarios (e.g., vrooli-autoheal). [REQ:OBS-004]
func HandleDetailedHealth(tunnelHealth TunnelChecker, routeLister RouteLister, probeSvc ProbeRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Get tunnel health
		ts := tunnelHealth.Check(ctx)

		// Get routes
		routes, err := routeLister.List()
		if err != nil {
			writeError(w, err)
			return
		}

		// Build route health info from latest probe results
		routeHealths := make([]domain.RouteHealthInfo, 0, len(routes))
		for _, route := range routes {
			rh := domain.RouteHealthInfo{
				Subdomain:    route.Subdomain,
				ScenarioName: route.ScenarioName,
				Enabled:      route.Enabled,
			}
			routeHealths = append(routeHealths, rh)
		}

		// Determine overall status
		overallStatus := "healthy"
		if ts.Ready != "ok" {
			overallStatus = "degraded"
		}
		if ts.Systemd != "" && ts.Systemd != "active" {
			overallStatus = "unhealthy"
		}

		resp := domain.DetailedHealth{
			Status: overallStatus,
			Tunnel: domain.TunnelHealthInfo{
				Ready:        ts.Ready,
				Systemd:      ts.Systemd,
				Score:        ts.Score,
				ReadyLatency: ts.ReadyLatency,
			},
			Routes:    routeHealths,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
