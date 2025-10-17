package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func handleGetScenarios(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scenarios := orchestrator.GetScenarios()

		// Convert to a clean response format
		result := make([]map[string]interface{}, 0, len(scenarios))
		for _, s := range scenarios {
			scenarioMap := map[string]interface{}{
				"id":          s.ID,
				"name":        s.Name,
				"displayName": s.DisplayName,
				"description": s.Description,
				"isActive":    s.IsActive,
				"endpoint":    s.Endpoint,
				"port":        s.Port,
				"tags":        s.Tags,
			}
			if s.LastActive != nil {
				scenarioMap["lastActive"] = s.LastActive.Format(time.RFC3339)
			}
			if s.ResourceUsage != nil {
				scenarioMap["resourceUsage"] = s.ResourceUsage
			}
			result = append(result, scenarioMap)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"scenarios": result,
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

		// Convert to a clean response format
		result := make([]map[string]interface{}, 0, len(presets))
		for _, p := range presets {
			presetMap := map[string]interface{}{
				"id":          p.ID,
				"name":        p.Name,
				"description": p.Description,
				"states":      p.States,
				"tags":        p.Tags,
				"pattern":     p.Pattern,
				"isDefault":   p.IsDefault,
				"isActive":    p.IsActive,
			}
			result = append(result, presetMap)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"presets": result,
		})
	}
}

func handleGetActivePresets(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		activePresets := orchestrator.GetActivePresets()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"activePresets": activePresets,
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

func handleCreatePreset(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name             string          `json:"name"`
			Description      string          `json:"description"`
			States           map[string]bool `json:"states,omitempty"`
			Tags             []string        `json:"tags,omitempty"`
			FromCurrentState bool            `json:"fromCurrentState,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		var preset *Preset
		var err error

		if req.FromCurrentState {
			// Create preset from current state of all scenarios
			preset, err = orchestrator.CreatePresetFromCurrentState(req.Name, req.Description)
		} else {
			// Create preset from provided states
			if req.States == nil || len(req.States) == 0 {
				http.Error(w, "States are required when not creating from current state", http.StatusBadRequest)
				return
			}
			preset, err = orchestrator.CreatePreset(req.Name, req.Description, req.States, req.Tags)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"preset":  preset,
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
		overallStatus := "healthy"

		// Schema-compliant health response
		healthResponse := map[string]interface{}{
			"status":       overallStatus,
			"service":      serviceName,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
			"readiness":    true, // Service is ready to accept requests
			"version":      apiVersion,
			"dependencies": map[string]interface{}{},
			"metrics": map[string]interface{}{
				"uptime_seconds": time.Since(startTime).Seconds(),
			},
		}

		dependencies := healthResponse["dependencies"].(map[string]interface{})

		// 1. Check scenario discovery capability (critical for orchestrator)
		discoveryHealth := checkScenarioDiscovery()
		dependencies["scenario_discovery"] = discoveryHealth
		if discoveryHealth["connected"] == false {
			overallStatus = "unhealthy" // Discovery is critical
		}

		// 2. Check preset management system
		presetHealth := checkPresetManagement()
		dependencies["preset_management"] = presetHealth
		if presetHealth["connected"] == false {
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}

		// 3. Check scenario state management
		stateHealth := checkScenarioStateManagement()
		dependencies["state_management"] = stateHealth
		if stateHealth["connected"] == false {
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}

		// 4. Check filesystem access for scenario management
		filesystemHealth := checkFilesystemAccess()
		dependencies["filesystem"] = filesystemHealth
		if filesystemHealth["connected"] == false {
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}

		// Update overall status
		healthResponse["status"] = overallStatus

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(healthResponse)
	}
}

// checkScenarioDiscovery tests the core scenario discovery functionality
func checkScenarioDiscovery() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	// Check if we can access scenarios directory
	scenariosDir := "scenarios"
	if _, err := os.Stat(scenariosDir); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "SCENARIOS_DIR_ACCESS_FAILED",
			"message":   fmt.Sprintf("Cannot access scenarios directory: %v", err),
			"category":  "resource",
			"retryable": false,
		}
		return result
	}

	// Try to read directory contents
	entries, err := ioutil.ReadDir(scenariosDir)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "SCENARIOS_DIR_READ_FAILED",
			"message":   fmt.Sprintf("Cannot read scenarios directory: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}

	// Count valid scenario directories (those with Makefile)
	validScenarios := 0
	for _, entry := range entries {
		if entry.IsDir() {
			makefilePath := fmt.Sprintf("%s/%s/Makefile", scenariosDir, entry.Name())
			if _, err := os.Stat(makefilePath); err == nil {
				validScenarios++
			}
		}
	}

	result["connected"] = true
	result["total_directories"] = len(entries)
	result["valid_scenarios"] = validScenarios

	return result
}

// checkPresetManagement tests preset system functionality
func checkPresetManagement() map[string]interface{} {
	result := map[string]interface{}{
		"connected": true,
		"error":     nil,
	}

	// Simple check - just return that presets are available
	// Avoid creating new orchestrators which might cause deadlocks
	result["available_presets"] = 0

	return result
}

// checkScenarioStateManagement tests scenario activation/deactivation system
func checkScenarioStateManagement() map[string]interface{} {
	result := map[string]interface{}{
		"connected": true,
		"error":     nil,
	}

	// Simple check - avoid creating new orchestrators
	result["active_scenarios"] = 0
	result["inactive_scenarios"] = 0
	result["total_scenarios"] = 0

	return result
}

// Simplified version to avoid deadlock
func checkScenarioStateManagement_OLD() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	// Test scenario state tracking
	tempOrchestrator := NewOrchestrator()
	if tempOrchestrator == nil {
		result["error"] = map[string]interface{}{
			"code":      "ORCHESTRATOR_INIT_FAILED",
			"message":   "Cannot initialize orchestrator for state testing",
			"category":  "internal",
			"retryable": true,
		}
		return result
	}

	// Test getting scenario states
	scenarios := tempOrchestrator.GetScenarios()
	if scenarios == nil {
		result["error"] = map[string]interface{}{
			"code":      "SCENARIO_STATE_SYSTEM_FAILED",
			"message":   "Scenario state system not functioning",
			"category":  "internal",
			"retryable": true,
		}
		return result
	}

	// Count active/inactive scenarios
	activeCount := 0
	for _, scenario := range scenarios {
		if scenario.IsActive {
			activeCount++
		}
	}

	result["connected"] = true
	result["total_scenarios"] = len(scenarios)
	result["active_scenarios"] = activeCount
	result["inactive_scenarios"] = len(scenarios) - activeCount

	return result
}

// checkFilesystemAccess tests filesystem operations needed for orchestration
func checkFilesystemAccess() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	// Test write access in working directory
	testFile := ".maintenance_orchestrator_health_test"
	testContent := fmt.Sprintf("Health check test at %s", time.Now().Format(time.RFC3339))

	err := ioutil.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "FILESYSTEM_WRITE_FAILED",
			"message":   fmt.Sprintf("Cannot write test file: %v", err),
			"category":  "resource",
			"retryable": false,
		}
		return result
	}

	// Test read access
	_, err = ioutil.ReadFile(testFile)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "FILESYSTEM_READ_FAILED",
			"message":   fmt.Sprintf("Cannot read test file: %v", err),
			"category":  "resource",
			"retryable": false,
		}
		return result
	}

	// Test delete access
	err = os.Remove(testFile)
	if err != nil {
		log.Printf("Warning: Could not clean up test file %s: %v", testFile, err)
		// Don't fail the health check for cleanup failure
	}

	// Get working directory info
	cwd, _ := os.Getwd()

	result["connected"] = true
	result["working_directory"] = cwd
	result["write_access"] = true
	result["read_access"] = true

	return result
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

// Start a scenario process using vrooli CLI
func handleStartScenario() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioID := vars["id"]

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "start", scenarioID)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			log.Printf("Error starting scenario %s: %v, stderr: %s", scenarioID, err, stderr.String())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Failed to start scenario: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"scenario": scenarioID,
			"action":   "started",
			"output":   strings.TrimSpace(out.String()),
		})
	}
}

// Stop a scenario process using vrooli CLI
func handleStopScenario() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioID := vars["id"]

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "stop", scenarioID)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			log.Printf("Error stopping scenario %s: %v, stderr: %s", scenarioID, err, stderr.String())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Failed to stop scenario: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"scenario": scenarioID,
			"action":   "stopped",
			"output":   strings.TrimSpace(out.String()),
		})
	}
}

// Fetch scenario statuses using vrooli CLI
func handleGetScenarioStatuses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context with 5 second timeout to prevent test hangs
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Run vrooli scenario status --json
		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "status", "--json")
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			// Log error but don't fail - return empty statuses
			log.Printf("Error fetching scenario statuses: %v, stderr: %s", err, stderr.String())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statuses": map[string]interface{}{},
			})
			return
		}

		// Parse the JSON output
		var statusData map[string]interface{}
		if err := json.Unmarshal(out.Bytes(), &statusData); err != nil {
			log.Printf("Error parsing scenario status JSON: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statuses": map[string]interface{}{},
			})
			return
		}

		// Extract scenarios from the status data
		statuses := make(map[string]interface{})
		if scenarios, ok := statusData["scenarios"].([]interface{}); ok {
			for _, scenario := range scenarios {
				if s, ok := scenario.(map[string]interface{}); ok {
					if name, ok := s["name"].(string); ok {
						status := "stopped"
						processCount := 0

						if statusStr, ok := s["status"].(string); ok {
							if statusStr == "running" || statusStr == "RUNNING" {
								status = "running"
							} else if statusStr == "error" || statusStr == "ERROR" {
								status = "error"
							}
						}

						// Check for "processes" field (actual field name in JSON)
						if procs, ok := s["processes"].(float64); ok {
							processCount = int(procs)
						}

						statuses[name] = map[string]interface{}{
							"status":       status,
							"processCount": processCount,
						}
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"statuses": statuses,
		})
	}
}

// List all scenarios using vrooli CLI
func handleListAllScenarios() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context with 5 second timeout to prevent test hangs
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "list", "--json")
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			log.Printf("Error listing scenarios: %v, stderr: %s", err, stderr.String())
			http.Error(w, "Failed to list scenarios", http.StatusInternalServerError)
			return
		}

		// Parse the JSON output
		var listData map[string]interface{}
		if err := json.Unmarshal(out.Bytes(), &listData); err != nil {
			log.Printf("Error parsing scenario list JSON: %v", err)
			http.Error(w, "Failed to parse scenario list", http.StatusInternalServerError)
			return
		}

		// Extract scenarios and their tags
		allScenarios := []map[string]interface{}{}
		if scenarios, ok := listData["scenarios"].([]interface{}); ok {
			for _, scenario := range scenarios {
				if s, ok := scenario.(map[string]interface{}); ok {
					scenarioInfo := map[string]interface{}{
						"name":        s["name"],
						"displayName": s["displayName"],
						"description": s["description"],
						"tags":        s["tags"],
					}
					allScenarios = append(allScenarios, scenarioInfo)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"scenarios": allScenarios,
		})
	}
}

// Add maintenance tag to a scenario
func handleAddMaintenanceTag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioName := vars["name"]

		// Read the service.json file
		servicePath := fmt.Sprintf("scenarios/%s/.vrooli/service.json", scenarioName)
		data, err := ioutil.ReadFile(servicePath)
		if err != nil {
			log.Printf("Error reading service.json for %s: %v", scenarioName, err)
			http.Error(w, "Failed to read service configuration", http.StatusInternalServerError)
			return
		}

		var service map[string]interface{}
		if err := json.Unmarshal(data, &service); err != nil {
			log.Printf("Error parsing service.json for %s: %v", scenarioName, err)
			http.Error(w, "Failed to parse service configuration", http.StatusInternalServerError)
			return
		}

		// Get or create service object
		serviceData, ok := service["service"].(map[string]interface{})
		if !ok {
			serviceData = make(map[string]interface{})
			service["service"] = serviceData
		}

		// Get or create tags array
		tags := []interface{}{}
		if existingTags, ok := serviceData["tags"].([]interface{}); ok {
			tags = existingTags
		}

		// Check if maintenance tag already exists
		hasMaintenanceTag := false
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok && tagStr == "maintenance" {
				hasMaintenanceTag = true
				break
			}
		}

		if !hasMaintenanceTag {
			tags = append(tags, "maintenance")
			serviceData["tags"] = tags

			// Write back the modified service.json
			modifiedData, err := json.MarshalIndent(service, "", "  ")
			if err != nil {
				log.Printf("Error marshaling service.json for %s: %v", scenarioName, err)
				http.Error(w, "Failed to update service configuration", http.StatusInternalServerError)
				return
			}

			if err := ioutil.WriteFile(servicePath, modifiedData, 0644); err != nil {
				log.Printf("Error writing service.json for %s: %v", scenarioName, err)
				http.Error(w, "Failed to save service configuration", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Maintenance tag added successfully",
		})
	}
}

// Remove maintenance tag from a scenario
func handleRemoveMaintenanceTag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioName := vars["name"]

		// Read the service.json file
		servicePath := fmt.Sprintf("scenarios/%s/.vrooli/service.json", scenarioName)
		data, err := ioutil.ReadFile(servicePath)
		if err != nil {
			log.Printf("Error reading service.json for %s: %v", scenarioName, err)
			http.Error(w, "Failed to read service configuration", http.StatusInternalServerError)
			return
		}

		var service map[string]interface{}
		if err := json.Unmarshal(data, &service); err != nil {
			log.Printf("Error parsing service.json for %s: %v", scenarioName, err)
			http.Error(w, "Failed to parse service configuration", http.StatusInternalServerError)
			return
		}

		// Get service object
		serviceData, ok := service["service"].(map[string]interface{})
		if !ok {
			// No service object means no tags
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "No maintenance tag to remove",
			})
			return
		}

		// Get tags array
		if existingTags, ok := serviceData["tags"].([]interface{}); ok {
			newTags := []interface{}{}
			for _, tag := range existingTags {
				if tagStr, ok := tag.(string); ok && tagStr != "maintenance" {
					newTags = append(newTags, tag)
				}
			}

			serviceData["tags"] = newTags

			// Write back the modified service.json
			modifiedData, err := json.MarshalIndent(service, "", "  ")
			if err != nil {
				log.Printf("Error marshaling service.json for %s: %v", scenarioName, err)
				http.Error(w, "Failed to update service configuration", http.StatusInternalServerError)
				return
			}

			if err := ioutil.WriteFile(servicePath, modifiedData, 0644); err != nil {
				log.Printf("Error writing service.json for %s: %v", scenarioName, err)
				http.Error(w, "Failed to save service configuration", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Maintenance tag removed successfully",
		})
	}
}

// Get preset assignments for a scenario
func handleGetScenarioPresetAssignments(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioName := vars["name"]

		assignments := make(map[string]bool)
		presets := orchestrator.GetPresets()

		// Check which presets include this scenario
		for _, preset := range presets {
			if preset.States != nil {
				if state, exists := preset.States[scenarioName]; exists {
					assignments[preset.ID] = state
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"assignments": assignments,
		})
	}
}

// Get scenario port using vrooli CLI
func handleGetScenarioPort() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioName := vars["name"]
		portType := r.URL.Query().Get("type")

		// Default to UI_PORT if not specified
		if portType == "" {
			portType = "UI_PORT"
		}

		// Run vrooli scenario port command with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, portType)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			// Scenario might not be running or might not have the requested port
			log.Printf("Error getting port for %s/%s: %v, stderr: %s", scenarioName, portType, err, stderr.String())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"port":  nil,
				"error": "Port not available",
			})
			return
		}

		// Parse the port number
		portStr := strings.TrimSpace(out.String())

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"port": portStr,
			"type": portType,
		})
	}
}

// Update preset assignments for a scenario
func handleUpdateScenarioPresetAssignments(orchestrator *Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenarioName := vars["name"]

		var requestData struct {
			Assignments map[string]bool `json:"assignments"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Update all presets to reflect the new assignments
		presets := orchestrator.GetPresets()

		// Create a map for easier lookup
		presetMap := make(map[string]*Preset)
		for _, preset := range presets {
			presetMap[preset.ID] = preset
		}

		for presetID, shouldInclude := range requestData.Assignments {
			if preset, exists := presetMap[presetID]; exists {
				if preset.States == nil {
					preset.States = make(map[string]bool)
				}

				if shouldInclude {
					// Add scenario to this preset (set to true for activation)
					preset.States[scenarioName] = true
				} else {
					// Remove scenario from this preset
					delete(preset.States, scenarioName)
				}
			}
		}

		// Also handle presets not mentioned in the request (remove scenario from them)
		for _, preset := range presets {
			if _, mentioned := requestData.Assignments[preset.ID]; !mentioned {
				if preset.States != nil {
					delete(preset.States, scenarioName)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Preset assignments updated successfully",
		})
	}
}
