package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func handleGetScenarios(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scenarios := orchestrator.GetScenarios()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"scenarios": scenarios,
		})
	}
}

func handleActivateScenario(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioID := vars["id"]

		scenario, exists := orchestrator.GetScenario(scenarioID)
		if !exists {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}

		if orchestrator.ActivateScenario(scenarioID) {
			if scenario.Endpoint != "" {
				go notifyScenarioStateChange(scenario.Endpoint, "active")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":  true,
				"scenario": scenarioID,
				"newState": "active",
			})
		} else {
			http.Error(w, "Failed to activate scenario", http.StatusInternalServerError)
		}
	}
}

func handleDeactivateScenario(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioID := vars["id"]

		scenario, exists := orchestrator.GetScenario(scenarioID)
		if !exists {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}

		if orchestrator.DeactivateScenario(scenarioID) {
			if scenario.Endpoint != "" {
				go notifyScenarioStateChange(scenario.Endpoint, "inactive")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":  true,
				"scenario": scenarioID,
				"newState": "inactive",
			})
		} else {
			http.Error(w, "Failed to deactivate scenario", http.StatusInternalServerError)
		}
	}
}

func handleGetPresets(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		presets := orchestrator.GetPresets()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"presets": presets,
		})
	}
}

func handleApplyPreset(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		presetID := vars["id"]

		activated, deactivated, success := orchestrator.ApplyPreset(presetID)
		if !success {
			http.Error(w, "Preset not found", http.StatusNotFound)
			return
		}

		// Notify scenarios of state changes
		scenarios := orchestrator.GetScenarios()
		scenarioMap := make(map[string]*MaintenanceScenario)
		for _, s := range scenarios {
			scenarioMap[s.ID] = s
		}

		for _, scenarioID := range activated {
			if scenario, exists := scenarioMap[scenarioID]; exists && scenario.Endpoint != "" {
				go notifyScenarioStateChange(scenario.Endpoint, "active")
			}
		}

		for _, scenarioID := range deactivated {
			if scenario, exists := scenarioMap[scenarioID]; exists && scenario.Endpoint != "" {
				go notifyScenarioStateChange(scenario.Endpoint, "inactive")
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"preset":      presetID,
			"activated":   activated,
			"deactivated": deactivated,
		})
	}
}

func handleGetStatus(orchestrator *Orchestrator, startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		totalScenarios, activeCount, recentActivity := orchestrator.GetStatus()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"health":            "healthy",
			"maintenanceState":  "active",
			"totalScenarios":    totalScenarios,
			"activeScenarios":   activeCount,
			"inactiveScenarios": totalScenarios - activeCount,
			"recentActivity":    recentActivity,
			"uptime":            time.Since(startTime).Seconds(),
		})
	}
}

func healthHandler(startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"service":   serviceName,
			"version":   apiVersion,
			"timestamp": time.Now().UTC(),
			"uptime":    time.Since(startTime).Seconds(),
		})
	}
}

func handleStopAll(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deactivated := orchestrator.StopAll()

		// Notify scenarios of state changes
		scenarios := orchestrator.GetScenarios()
		scenarioMap := make(map[string]*MaintenanceScenario)
		for _, s := range scenarios {
			scenarioMap[s.ID] = s
		}

		for _, scenarioID := range deactivated {
			if scenario, exists := scenarioMap[scenarioID]; exists && scenario.Endpoint != "" {
				go notifyScenarioStateChange(scenario.Endpoint, "inactive")
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"deactivated": deactivated,
		})
	}
}

func notifyScenarioStateChange(endpoint, state string) {
	client := &http.Client{Timeout: 5 * time.Second}
	
	payload := map[string]string{"maintenanceState": state}
	data, _ := json.Marshal(payload)
	
	req, err := http.NewRequest("POST", endpoint+"/api/maintenance/state", strings.NewReader(string(data)))
	if err != nil {
		log.Printf("Error creating request for %s: %v", endpoint, err)
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error notifying %s of state change: %v", endpoint, err)
		return
	}
	defer resp.Body.Close()
}