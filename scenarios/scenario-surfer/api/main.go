package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type ScenarioInfo struct {
	Name           string            `json:"name"`
	Status         string            `json:"status"`
	Health         string            `json:"health_status"`
	Ports          map[string]int    `json:"ports"`
	Tags           []string          `json:"tags,omitempty"`
	URL            string            `json:"url,omitempty"`
	DisplayName    string            `json:"display_name,omitempty"`
	Description    string            `json:"description,omitempty"`
	Runtime        string            `json:"runtime,omitempty"`
	Processes      int               `json:"processes,omitempty"`
}


type VrooliStatusResponse struct {
	Success   bool           `json:"success"`
	Scenarios []ScenarioInfo `json:"scenarios"`
	Summary   struct {
		Total   int `json:"total_scenarios"`
		Running int `json:"running"`
		Stopped int `json:"stopped"`
	} `json:"summary"`
	RawResponse json.RawMessage `json:"raw_response,omitempty"`
}

type IssueReport struct {
	Scenario    string `json:"scenario"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Screenshot  string `json:"screenshot,omitempty"`
}

type HealthyScenarioResponse struct {
	Scenarios  []ScenarioInfo `json:"scenarios"`
	Categories []string       `json:"categories"`
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-surfer

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "27000"
	}

	r := mux.NewRouter()
	
	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")
	
	// API endpoints
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/scenarios/healthy", getHealthyScenariosHandler).Methods("GET")
	api.HandleFunc("/scenarios/status", getAllScenariosHandler).Methods("GET")
	api.HandleFunc("/scenarios/debug", getDebugStatusHandler).Methods("GET")
	api.HandleFunc("/issues/report", reportIssueHandler).Methods("POST")
	
	// CORS middleware
	r.Use(corsMiddleware)

	log.Printf("🌊 Scenario Surfer API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "scenario-surfer",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getAllScenariosHandler(w http.ResponseWriter, r *http.Request) {
	scenarios, err := getScenarios()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
		"timestamp": time.Now().Unix(),
	})
}

func getHealthyScenariosHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Fetching healthy scenarios...")
	scenarios, err := getScenarios()
	if err != nil {
		log.Printf("Error getting scenarios: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Found %d total scenarios", len(scenarios))
	
	// Filter for healthy running scenarios with UI ports
	healthyScenarios := []ScenarioInfo{}
	categorySet := make(map[string]bool)
	
	for _, scenario := range scenarios {
		// Must be running and healthy with a UI port
		// health_status field is "healthy", "degraded", or "unhealthy"
		if scenario.Status == "running" && scenario.Health == "healthy" {
			// Check for UI port - could be "ui", "UI_PORT", or scenario-specific names
			uiPort := 0
			for key, port := range scenario.Ports {
				keyLower := strings.ToLower(key)
				if strings.Contains(keyLower, "ui") && port > 0 {
					uiPort = port
					break
				}
			}
			
			if uiPort > 0 {
				// Skip port check for now - assume healthy scenarios have responding ports
				scenario.URL = fmt.Sprintf("http://localhost:%d", uiPort)
				
				// Load tags from service.json if not already present
				if len(scenario.Tags) == 0 {
					tags := loadScenarioTags(scenario.Name)
					scenario.Tags = tags
				}
				for _, tag := range scenario.Tags {
					categorySet[tag] = true
				}
				
				healthyScenarios = append(healthyScenarios, scenario)
			}
		}
	}
	
	// Convert category set to slice
	categories := []string{}
	for category := range categorySet {
		categories = append(categories, category)
	}
	
	response := HealthyScenarioResponse{
		Scenarios:  healthyScenarios,
		Categories: categories,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func reportIssueHandler(w http.ResponseWriter, r *http.Request) {
	var report IssueReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Basic validation
	if report.Scenario == "" || report.Title == "" || report.Description == "" {
		http.Error(w, "Missing required fields: scenario, title, description", http.StatusBadRequest)
		return
	}
	
	// Try to capture screenshot if requested and browserless is available
	if report.Screenshot == "" {
		screenshot, err := captureScreenshot(report.Scenario)
		if err == nil {
			report.Screenshot = screenshot
		}
		// Don't fail if screenshot capture fails, just continue without it
	}
	
	// Submit to app-issue-tracker (simplified - in production would use proper API)
	err := submitIssueToTracker(report)
	if err != nil {
		log.Printf("Failed to submit issue to tracker: %v", err)
		http.Error(w, "Failed to submit issue", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Issue reported successfully",
		"timestamp": time.Now().Unix(),
	})
}

func getScenarios() ([]ScenarioInfo, error) {
	// Execute vrooli scenario status --json
	cmd := exec.Command("vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get scenario status: %v", err)
	}
	
	// Parse the full response structure
	var fullResponse struct {
		Success   bool `json:"success"`
		Summary   struct {
			Total   int `json:"total_scenarios"`
			Running int `json:"running"`
			Stopped int `json:"stopped"`
		} `json:"summary"`
		Scenarios []ScenarioInfo `json:"scenarios"`
	}
	
	if err := json.Unmarshal(output, &fullResponse); err != nil {
		return nil, fmt.Errorf("failed to parse scenario status: %v", err)
	}
	
	return fullResponse.Scenarios, nil
}

func loadScenarioTags(scenarioName string) []string {
	// Try to read service.json file for the scenario
	servicePath := fmt.Sprintf("../scenarios/%s/.vrooli/service.json", scenarioName)
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return []string{}
	}
	
	content, err := os.ReadFile(servicePath)
	if err != nil {
		return []string{}
	}
	
	var serviceConfig struct {
		Service struct {
			Tags []string `json:"tags"`
		} `json:"service"`
	}
	
	if err := json.Unmarshal(content, &serviceConfig); err != nil {
		return []string{}
	}
	
	return serviceConfig.Service.Tags
}

func captureScreenshot(scenarioURL string) (string, error) {
	// Use browserless to capture screenshot
	cmd := exec.Command("resource-browserless", "screenshot", scenarioURL, "--output", "/tmp/scenario-surfer-"+strconv.FormatInt(time.Now().Unix(), 10)+".png")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	// Return the path to the screenshot
	return strings.TrimSpace(string(output)), nil
}

func submitIssueToTracker(report IssueReport) error {
	// Try to find app-issue-tracker API endpoint
	scenarios, err := getScenarios()
	if err != nil {
		return err
	}
	
	var issueTrackerPort int
	for _, scenario := range scenarios {
		if scenario.Name == "app-issue-tracker" && scenario.Status == "running" {
			if apiPort, hasAPI := scenario.Ports["api"]; hasAPI {
				issueTrackerPort = apiPort
				break
			}
		}
	}
	
	if issueTrackerPort == 0 {
		return fmt.Errorf("app-issue-tracker not found or not running")
	}
	
	// Prepare issue data
	issueData := map[string]interface{}{
		"title":       fmt.Sprintf("[scenario-surfer] %s", report.Title),
		"description": fmt.Sprintf("**Scenario:** %s\n\n%s", report.Scenario, report.Description),
		"scenario":    report.Scenario,
		"source":      "scenario-surfer",
		"screenshot":  report.Screenshot,
	}
	
	// Submit to app-issue-tracker API
	jsonData, _ := json.Marshal(issueData)
	url := fmt.Sprintf("http://localhost:%d/api/v1/issues", issueTrackerPort)
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("issue tracker returned status: %d", resp.StatusCode)
	}
	
	return nil
}

// isPortResponding checks if a port is actually responding to HTTP requests
func isPortResponding(port int) bool {
	client := &http.Client{
		Timeout: 500 * time.Millisecond, // Reduced timeout for faster checks
	}
	
	url := fmt.Sprintf("http://localhost:%d", port)
	resp, err := client.Get(url)
	if err != nil {
		// Port might be listening but not HTTP, or not responding
		return false
	}
	defer resp.Body.Close()
	
	// Any HTTP response means the port is responding
	return true
}

// getDebugStatusHandler returns the raw vrooli scenario status output for debugging
func getDebugStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Execute vrooli scenario status --json
	cmd := exec.Command("vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	
	debugResponse := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"command":   "vrooli scenario status --json",
	}
	
	if err != nil {
		debugResponse["error"] = fmt.Sprintf("Command failed: %v", err)
		debugResponse["raw_output"] = string(output)
	} else {
		// Try to parse as JSON
		var parsed interface{}
		if err := json.Unmarshal(output, &parsed); err != nil {
			debugResponse["parse_error"] = fmt.Sprintf("Failed to parse JSON: %v", err)
			debugResponse["raw_output"] = string(output)
		} else {
			debugResponse["parsed_output"] = parsed
			debugResponse["raw_output"] = string(output)
		}
	}
	
	// Also get scenarios using our internal method
	scenarios, err := getScenarios()
	if err != nil {
		debugResponse["internal_error"] = fmt.Sprintf("Internal getScenarios failed: %v", err)
	} else {
		debugResponse["internal_scenarios"] = scenarios
		debugResponse["internal_count"] = len(scenarios)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(debugResponse)
}