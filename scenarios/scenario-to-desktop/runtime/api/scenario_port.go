package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/peerlookup"
)

// handleScenarioPort is the native discovery boundary used by bundled callers.
// Peer selection stays with the runtime that owns the declared version ranges.
func (s *Server) handleScenarioPort(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scenario, key := r.URL.Query().Get("scenario"), r.URL.Query().Get("port_name")
	if !peerlookup.ValidScenarioName(scenario) || (key != "API_PORT" && key != "UI_PORT") {
		http.Error(w, "invalid scenario or port name", http.StatusBadRequest)
		return
	}
	home, err := os.UserHomeDir()
	if source, ok := s.runtime.(interface{ PeerHomeDir() (string, error) }); ok {
		home, err = source.PeerHomeDir()
	}
	if err != nil {
		http.Error(w, "peer registry unavailable", http.StatusServiceUnavailable)
		return
	}
	bundle := s.runtime.Manifest()
	port, _ := peerlookup.External(bundle, home, scenario, key)
	if port == 0 {
		service, name, err := peerlookup.Local(bundle, scenario, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		port = s.runtime.PortMap()[service][name]
	}
	if port < 1 || port > 65535 {
		http.Error(w, "no running runtime ports found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"scenario": scenario, "port_name": key, "port": port})
}
