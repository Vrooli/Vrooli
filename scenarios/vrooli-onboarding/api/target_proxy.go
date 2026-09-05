package main

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/nodereach"
)

// targetProxyMiddleware keeps onboarding's handlers authoritative on the
// machine being configured. A browser may stay attached to the control-plane
// onboarding UI, while every /api/v2 read or write for target=<node> is
// executed by that node's onboarding process through Bridge.
func (s *Server) targetProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := strings.TrimSpace(r.URL.Query().Get("target"))
		if target == "" || target == "local" || !strings.HasPrefix(r.URL.Path, "/api/v2/") {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20+1))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target request body could not be read"})
			return
		}
		if len(body) > 8<<20 {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "target request exceeds byte limit"})
			return
		}
		methodPath := strings.TrimPrefix(r.URL.Path, "/api/")
		client := s.bridge
		if client == nil {
			client = nodereach.New(nodereach.Config{})
		}
		if token := strings.TrimSpace(r.Header.Get("Authorization")); token != "" {
			client = nodereach.New(nodereach.Config{Token: token})
		}
		response, err := client.CallScenario(r.Context(), nodereach.ScenarioRequest{
			NodeID: target, Scenario: "vrooli-onboarding", Service: "api", Method: methodPath,
			HTTPPath: methodPath, HTTPMethod: r.Method, Body: body, Timeout: 30 * time.Second, MaxResponse: 8 << 20,
		})
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error(), "target": target, "recovery": "Confirm the node is online, paired, and granted the vrooli-onboarding scope."})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	})
}
