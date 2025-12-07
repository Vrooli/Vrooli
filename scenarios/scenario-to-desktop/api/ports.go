package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type scenarioPortResponse struct {
	Scenario string `json:"scenario"`
	PortName string `json:"port_name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	URL      string `json:"url"`
}

// getScenarioPortHandler resolves a scenario port using the vrooli CLI.
// Ports are dynamic per lifecycle; avoid hardcoding or caching across runs.
func (s *Server) getScenarioPortHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := strings.TrimSpace(vars["scenario"])
	portName := strings.TrimSpace(vars["port_name"])

	if scenario == "" || portName == "" {
		http.Error(w, "scenario and port_name are required", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("vrooli", "scenario", "port", scenario, portName)
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve port via vrooli: %v", err), http.StatusBadGateway)
		return
	}

	portStr := strings.TrimSpace(string(output))
	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid port returned: %s", portStr), http.StatusBadGateway)
		return
	}

	resp := scenarioPortResponse{
		Scenario: scenario,
		PortName: portName,
		Host:     "127.0.0.1",
		Port:     port,
		URL:      fmt.Sprintf("http://127.0.0.1:%d", port),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
