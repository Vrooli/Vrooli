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
	Health         string            `json:"actual_health"`
	AllocatedPorts map[string]int    `json:"allocated_ports"`
	PortStatus     map[string]PortInfo `json:"port_status"`
	Tags           []string          `json:"tags,omitempty"`
	URL            string            `json:"url,omitempty"`
	DisplayName    string            `json:"display_name,omitempty"`
}

type PortInfo struct {
	Port       int  `json:"port"`
	Listening  bool `json:"listening"`
	Responding bool `json:"responding"`
}

type VrooliStatusResponse struct {
	Apps []ScenarioInfo `json:"apps"`
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
	api.HandleFunc("/issues/report", reportIssueHandler).Methods("POST")
	
	// CORS middleware
	r.Use(corsMiddleware)

	log.Printf("🌊 Scenario Surfer API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
	scenarios, err := getScenarios()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Filter for healthy running scenarios with UI ports
	healthyScenarios := []ScenarioInfo{}
	categorySet := make(map[string]bool)
	
	for _, scenario := range scenarios {
		// Must be running and healthy with a UI port
		if scenario.Status == "running" && scenario.Health == "healthy" {
			if uiPort, hasUI := scenario.AllocatedPorts["ui"]; hasUI && uiPort > 0 {
				// Check if UI port is actually responding
				if portInfo, exists := scenario.PortStatus["ui"]; exists && portInfo.Responding {
					scenario.URL = fmt.Sprintf("http://localhost:%d", uiPort)
					
					// Load tags from service.json if available
					tags := loadScenarioTags(scenario.Name)
					scenario.Tags = tags
					for _, tag := range tags {
						categorySet[tag] = true
					}
					
					healthyScenarios = append(healthyScenarios, scenario)
				}
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
	
	var response VrooliStatusResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse scenario status: %v", err)
	}
	
	return response.Apps, nil
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
			if apiPort, hasAPI := scenario.AllocatedPorts["api"]; hasAPI {
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