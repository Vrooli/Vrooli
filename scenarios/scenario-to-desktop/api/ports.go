package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
)

type scenarioPortResponse struct {
	Scenario string `json:"scenario"`
	PortName string `json:"port_name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	URL      string `json:"url"`
}

// getScenarioPortHandler resolves a scenario port using discovery.
// Ports are dynamic per lifecycle; avoid hardcoding or caching across runs.
func (s *Server) getScenarioPortHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := strings.TrimSpace(vars["scenario"])
	portName := strings.TrimSpace(vars["port_name"])

	if scenario == "" || portName == "" {
		http.Error(w, "scenario and port_name are required", http.StatusBadRequest)
		return
	}

	port, err := discovery.ResolveScenarioPort(r.Context(), scenario, portName)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve port: %v", err), http.StatusBadGateway)
		return
	}

	resp := scenarioPortResponse{
		Scenario: scenario,
		PortName: portName,
		Host:     "127.0.0.1",
		Port:     port,
		URL:      fmt.Sprintf("http://127.0.0.1:%d", port),
	}

	writeJSONResponse(w, http.StatusOK, resp)
}
