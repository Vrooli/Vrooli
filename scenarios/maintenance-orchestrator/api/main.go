package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type MaintenanceScenario struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	DisplayName  string            `json:"displayName"`
	Description  string            `json:"description"`
	IsActive     bool              `json:"isActive"`
	Endpoint     string            `json:"endpoint"`
	Port         int               `json:"port"`
	Tags         []string          `json:"tags"`
	LastActive   *time.Time        `json:"lastActive,omitempty"`
	ResourceUsage map[string]float64 `json:"resourceUsage,omitempty"`
}

type Preset struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	States      map[string]bool   `json:"states"`
	Tags        []string          `json:"tags,omitempty"`
	Pattern     string            `json:"pattern,omitempty"`
	IsDefault   bool              `json:"isDefault"`
}

type Orchestrator struct {
	scenarios map[string]*MaintenanceScenario
	presets   map[string]*Preset
	mu        sync.RWMutex
	activityLog []ActivityEntry
}

type ActivityEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Scenario  string    `json:"scenario,omitempty"`
	Preset    string    `json:"preset,omitempty"`
	Success   bool      `json:"success"`
	Message   string    `json:"message,omitempty"`
}

var orchestrator *Orchestrator

func init() {
	orchestrator = &Orchestrator{
		scenarios:   make(map[string]*MaintenanceScenario),
		presets:     make(map[string]*Preset),
		activityLog: make([]ActivityEntry, 0),
	}
	initializeDefaultPresets()
}

func initializeDefaultPresets() {
	orchestrator.presets["full-maintenance"] = &Preset{
		ID:          "full-maintenance",
		Name:        "Full Maintenance",
		Description: "Activate all maintenance scenarios",
		Pattern:     "*",
		IsDefault:   true,
		States:      make(map[string]bool),
	}

	orchestrator.presets["security-only"] = &Preset{
		ID:          "security-only",
		Name:        "Security Only",
		Description: "Security-related maintenance only",
		Tags:        []string{"security"},
		IsDefault:   true,
		States:      make(map[string]bool),
	}

	orchestrator.presets["performance"] = &Preset{
		ID:          "performance",
		Name:        "Performance",
		Description: "Performance optimization scenarios",
		Tags:        []string{"performance", "optimization"},
		IsDefault:   true,
		States:      make(map[string]bool),
	}

	orchestrator.presets["off-hours"] = &Preset{
		ID:          "off-hours",
		Name:        "Off Hours",
		Description: "Heavy maintenance for quiet periods",
		Tags:        []string{"heavy", "backup", "cleanup"},
		IsDefault:   true,
		States:      make(map[string]bool),
	}

	orchestrator.presets["minimal"] = &Preset{
		ID:          "minimal",
		Name:        "Minimal",
		Description: "Essential maintenance only",
		Tags:        []string{"essential", "critical"},
		IsDefault:   true,
		States:      make(map[string]bool),
	}
}

func discoverScenarios() {
	scenariosPath := "/home/matthalloran8/Vrooli/scenarios"
	if envPath := os.Getenv("VROOLI_SCENARIOS_PATH"); envPath != "" {
		scenariosPath = envPath
	}

	entries, err := ioutil.ReadDir(scenariosPath)
	if err != nil {
		log.Printf("Error reading scenarios directory: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		serviceJsonPath := filepath.Join(scenariosPath, entry.Name(), ".vrooli", "service.json")
		data, err := ioutil.ReadFile(serviceJsonPath)
		if err != nil {
			continue
		}

		var service map[string]interface{}
		if err := json.Unmarshal(data, &service); err != nil {
			continue
		}

		tags, ok := service["tags"].([]interface{})
		if !ok {
			continue
		}

		hasMaintenance := false
		stringTags := make([]string, 0)
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				stringTags = append(stringTags, tagStr)
				if tagStr == "maintenance" {
					hasMaintenance = true
				}
			}
		}

		if !hasMaintenance {
			continue
		}

		scenario := &MaintenanceScenario{
			ID:          entry.Name(),
			Name:        entry.Name(),
			DisplayName: getStringField(service, "displayName", entry.Name()),
			Description: getStringField(service, "description", ""),
			IsActive:    false,
			Tags:        stringTags,
		}

		if ports, ok := service["ports"].(map[string]interface{}); ok {
			if apiPort, ok := ports["api"].(float64); ok {
				scenario.Port = int(apiPort)
				scenario.Endpoint = fmt.Sprintf("http://localhost:%d", int(apiPort))
			}
		}

		orchestrator.mu.Lock()
		orchestrator.scenarios[scenario.ID] = scenario
		orchestrator.mu.Unlock()

		log.Printf("Discovered maintenance scenario: %s", scenario.ID)
	}

	updatePresetStates()
}

func updatePresetStates() {
	orchestrator.mu.Lock()
	defer orchestrator.mu.Unlock()

	for _, preset := range orchestrator.presets {
		for scenarioID, scenario := range orchestrator.scenarios {
			if preset.Pattern == "*" {
				preset.States[scenarioID] = true
			} else if len(preset.Tags) > 0 {
				for _, presetTag := range preset.Tags {
					for _, scenarioTag := range scenario.Tags {
						if presetTag == scenarioTag {
							preset.States[scenarioID] = true
							break
						}
					}
				}
			}
		}
	}
}

func getStringField(m map[string]interface{}, field, defaultValue string) string {
	if val, ok := m[field].(string); ok {
		return val
	}
	return defaultValue
}

func handleGetScenarios(w http.ResponseWriter, r *http.Request) {
	orchestrator.mu.RLock()
	defer orchestrator.mu.RUnlock()

	scenarios := make([]*MaintenanceScenario, 0, len(orchestrator.scenarios))
	for _, scenario := range orchestrator.scenarios {
		scenarios = append(scenarios, scenario)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
	})
}

func handleActivateScenario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["id"]

	orchestrator.mu.Lock()
	defer orchestrator.mu.Unlock()

	scenario, exists := orchestrator.scenarios[scenarioID]
	if !exists {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	}

	scenario.IsActive = true
	now := time.Now()
	scenario.LastActive = &now

	orchestrator.activityLog = append(orchestrator.activityLog, ActivityEntry{
		Timestamp: now,
		Action:    "activate",
		Scenario:  scenarioID,
		Success:   true,
	})

	if scenario.Endpoint != "" {
		go notifyScenarioStateChange(scenario.Endpoint, "active")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"scenario": scenarioID,
		"newState": "active",
	})
}

func handleDeactivateScenario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["id"]

	orchestrator.mu.Lock()
	defer orchestrator.mu.Unlock()

	scenario, exists := orchestrator.scenarios[scenarioID]
	if !exists {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	}

	scenario.IsActive = false

	orchestrator.activityLog = append(orchestrator.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    "deactivate",
		Scenario:  scenarioID,
		Success:   true,
	})

	if scenario.Endpoint != "" {
		go notifyScenarioStateChange(scenario.Endpoint, "inactive")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"scenario": scenarioID,
		"newState": "inactive",
	})
}

func handleGetPresets(w http.ResponseWriter, r *http.Request) {
	orchestrator.mu.RLock()
	defer orchestrator.mu.RUnlock()

	presets := make([]*Preset, 0, len(orchestrator.presets))
	for _, preset := range orchestrator.presets {
		presets = append(presets, preset)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"presets": presets,
	})
}

func handleApplyPreset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	presetID := vars["id"]

	orchestrator.mu.Lock()
	defer orchestrator.mu.Unlock()

	preset, exists := orchestrator.presets[presetID]
	if !exists {
		http.Error(w, "Preset not found", http.StatusNotFound)
		return
	}

	activated := make([]string, 0)
	deactivated := make([]string, 0)

	for scenarioID, shouldBeActive := range preset.States {
		scenario, exists := orchestrator.scenarios[scenarioID]
		if !exists {
			continue
		}

		if shouldBeActive && !scenario.IsActive {
			scenario.IsActive = true
			now := time.Now()
			scenario.LastActive = &now
			activated = append(activated, scenarioID)
			
			if scenario.Endpoint != "" {
				go notifyScenarioStateChange(scenario.Endpoint, "active")
			}
		} else if !shouldBeActive && scenario.IsActive {
			scenario.IsActive = false
			deactivated = append(deactivated, scenarioID)
			
			if scenario.Endpoint != "" {
				go notifyScenarioStateChange(scenario.Endpoint, "inactive")
			}
		}
	}

	orchestrator.activityLog = append(orchestrator.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    "apply-preset",
		Preset:    presetID,
		Success:   true,
		Message:   fmt.Sprintf("Activated: %d, Deactivated: %d", len(activated), len(deactivated)),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"preset":      presetID,
		"activated":   activated,
		"deactivated": deactivated,
	})
}

func handleGetStatus(w http.ResponseWriter, r *http.Request) {
	orchestrator.mu.RLock()
	defer orchestrator.mu.RUnlock()

	activeCount := 0
	for _, scenario := range orchestrator.scenarios {
		if scenario.IsActive {
			activeCount++
		}
	}

	recentActivity := orchestrator.activityLog
	if len(recentActivity) > 10 {
		recentActivity = orchestrator.activityLog[len(orchestrator.activityLog)-10:]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"health":            "healthy",
		"maintenanceState":  "active",
		"totalScenarios":    len(orchestrator.scenarios),
		"activeScenarios":   activeCount,
		"inactiveScenarios": len(orchestrator.scenarios) - activeCount,
		"recentActivity":    recentActivity,
		"uptime":            time.Since(startTime).Seconds(),
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Unix(),
	})
}

func handleStopAll(w http.ResponseWriter, r *http.Request) {
	orchestrator.mu.Lock()
	defer orchestrator.mu.Unlock()

	deactivated := make([]string, 0)
	for scenarioID, scenario := range orchestrator.scenarios {
		if scenario.IsActive {
			scenario.IsActive = false
			deactivated = append(deactivated, scenarioID)
			
			if scenario.Endpoint != "" {
				go notifyScenarioStateChange(scenario.Endpoint, "inactive")
			}
		}
	}

	orchestrator.activityLog = append(orchestrator.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    "stop-all",
		Success:   true,
		Message:   fmt.Sprintf("Deactivated %d scenarios", len(deactivated)),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"deactivated": deactivated,
	})
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

var startTime = time.Now()

func main() {
	port := os.Getenv("MAINTENANCE_PORT")
	if port == "" {
		port = "3250"
	}

	go func() {
		for {
			discoverScenarios()
			time.Sleep(60 * time.Second)
		}
	}()

	r := mux.NewRouter()
	
	r.HandleFunc("/api/v1/scenarios", handleGetScenarios).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{id}/activate", handleActivateScenario).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{id}/deactivate", handleDeactivateScenario).Methods("POST")
	r.HandleFunc("/api/v1/presets", handleGetPresets).Methods("GET")
	r.HandleFunc("/api/v1/presets/{id}/apply", handleApplyPreset).Methods("POST")
	r.HandleFunc("/api/v1/status", handleGetStatus).Methods("GET")
	r.HandleFunc("/api/v1/stop-all", handleStopAll).Methods("POST")
	r.HandleFunc("/api/health", handleHealth).Methods("GET")

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(r)

	log.Printf("Maintenance Orchestrator API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}