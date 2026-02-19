package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// DetailedHealth is the composite health response for cross-scenario consumption. [REQ:OBS-004]
type DetailedHealth struct {
	Status    string            `json:"status"`
	Tunnel    TunnelHealthInfo  `json:"tunnel"`
	Routes    []RouteHealthInfo `json:"routes"`
	Timestamp string            `json:"timestamp"`
}

// TunnelHealthInfo summarizes the tunnel's health state.
type TunnelHealthInfo struct {
	Ready        string `json:"ready"`
	Systemd      string `json:"systemd"`
	Score        int    `json:"score"`
	ReadyLatency int    `json:"ready_latency_ms,omitempty"`
}

// RouteHealthInfo summarizes a route's recent probe status.
type RouteHealthInfo struct {
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	Enabled      bool   `json:"enabled"`
	InternalUp   *bool  `json:"internal_up,omitempty"`
	ExternalUp   *bool  `json:"external_up,omitempty"`
}

// handleDetailedHealth returns composite tunnel health including per-route status.
// Designed for consumption by other Vrooli scenarios (e.g., vrooli-autoheal). [REQ:OBS-004]
func handleDetailedHealth(tunnelHealth *TunnelHealthChecker, routeSvc *RouteService, probeSvc *ProbeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Get tunnel health
		ts := tunnelHealth.Check(ctx)

		// Get routes
		routes, err := routeSvc.List()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("list routes: %v", err))
			return
		}

		// Build route health info from latest probe results
		routeHealths := make([]RouteHealthInfo, 0, len(routes))
		for _, route := range routes {
			rh := RouteHealthInfo{
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

		resp := DetailedHealth{
			Status: overallStatus,
			Tunnel: TunnelHealthInfo{
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
