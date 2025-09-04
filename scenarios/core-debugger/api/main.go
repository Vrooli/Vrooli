package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Types
type Component struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	HealthCheck string `json:"health_check"`
	Critical    bool   `json:"critical"`
	TimeoutMs   int    `json:"timeout_ms"`
}

type ComponentHealth struct {
	Component      string    `json:"component"`
	Status         string    `json:"status"`
	LastCheck      time.Time `json:"last_check"`
	ResponseTimeMs int       `json:"response_time_ms"`
	ErrorCount     int       `json:"error_count"`
	Details        map[string]interface{} `json:"details"`
}

type CoreIssue struct {
	ID             string                 `json:"id"`
	Component      string                 `json:"component"`
	Severity       string                 `json:"severity"`
	Status         string                 `json:"status"`
	Description    string                 `json:"description"`
	ErrorSignature string                 `json:"error_signature"`
	FirstSeen      time.Time              `json:"first_seen"`
	LastSeen       time.Time              `json:"last_seen"`
	OccurrenceCount int                   `json:"occurrence_count"`
	Workarounds    []Workaround           `json:"workarounds"`
	FixAttempts    []FixAttempt           `json:"fix_attempts"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type Workaround struct {
	ID          string    `json:"id"`
	IssueID     string    `json:"issue_id"`
	Description string    `json:"description"`
	Commands    []string  `json:"commands"`
	SuccessRate float64   `json:"success_rate"`
	CreatedAt   time.Time `json:"created_at"`
	Validated   bool      `json:"validated"`
}

type FixAttempt struct {
	Timestamp  time.Time `json:"timestamp"`
	Analysis   string    `json:"analysis"`
	Success    bool      `json:"success"`
	Output     string    `json:"output"`
}

type HealthResponse struct {
	Status      string            `json:"status"`
	Components  []ComponentHealth `json:"components"`
	ActiveIssues int              `json:"active_issues"`
	LastCheck   time.Time         `json:"last_check"`
}

// Global variables
var (
	dataDir    string
	components []Component
)

func init() {
	// Get data directory from environment or use default
	dataDir = os.Getenv("DATA_DIR")
	if dataDir == "" {
		execPath, _ := os.Executable()
		scenarioRoot := filepath.Dir(filepath.Dir(execPath))
		dataDir = filepath.Join(scenarioRoot, "data")
	}

	// Load components
	loadComponents()
}

func loadComponents() {
	componentsFile := filepath.Join(dataDir, "components.json")
	data, err := ioutil.ReadFile(componentsFile)
	if err != nil {
		log.Printf("Warning: Could not load components: %v", err)
		return
	}

	var componentData struct {
		Components []Component `json:"components"`
	}
	if err := json.Unmarshal(data, &componentData); err != nil {
		log.Printf("Warning: Could not parse components: %v", err)
		return
	}

	components = componentData.Components
}

// Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := checkAllComponents()
	
	// Count active issues
	issuesDir := filepath.Join(dataDir, "issues")
	issues, _ := ioutil.ReadDir(issuesDir)
	activeCount := 0
	for _, file := range issues {
		if strings.HasSuffix(file.Name(), ".json") {
			activeCount++
		}
	}

	response := HealthResponse{
		Status:       determineOverallStatus(health),
		Components:   health,
		ActiveIssues: activeCount,
		LastCheck:    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func checkAllComponents() []ComponentHealth {
	var results []ComponentHealth

	for _, comp := range components {
		start := time.Now()
		cmd := exec.Command("bash", "-c", comp.HealthCheck)
		cmd.Timeout = time.Duration(comp.TimeoutMs) * time.Millisecond
		
		status := "healthy"
		err := cmd.Run()
		if err != nil {
			status = "unhealthy"
		}
		
		elapsed := time.Since(start)
		
		health := ComponentHealth{
			Component:      comp.ID,
			Status:         status,
			LastCheck:      time.Now(),
			ResponseTimeMs: int(elapsed.Milliseconds()),
			ErrorCount:     0,
			Details:        make(map[string]interface{}),
		}
		
		results = append(results, health)
		
		// Save to health log
		saveHealthState(health)
	}

	return results
}

func determineOverallStatus(health []ComponentHealth) string {
	criticalDown := false
	anyDown := false

	for i, h := range health {
		if h.Status != "healthy" {
			anyDown = true
			// Check if this component is critical
			if i < len(components) && components[i].Critical {
				criticalDown = true
			}
		}
	}

	if criticalDown {
		return "critical"
	} else if anyDown {
		return "degraded"
	}
	return "healthy"
}

func saveHealthState(health ComponentHealth) {
	healthFile := filepath.Join(dataDir, "health", "latest.jsonl")
	file, err := os.OpenFile(healthFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error saving health state: %v", err)
		return
	}
	defer file.Close()

	data, _ := json.Marshal(health)
	file.WriteString(string(data) + "\n")
}

func listIssuesHandler(w http.ResponseWriter, r *http.Request) {
	component := r.URL.Query().Get("component")
	severity := r.URL.Query().Get("severity")
	status := r.URL.Query().Get("status")
	
	issuesDir := filepath.Join(dataDir, "issues")
	files, err := ioutil.ReadDir(issuesDir)
	if err != nil {
		http.Error(w, "Error reading issues", http.StatusInternalServerError)
		return
	}

	var issues []CoreIssue
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(issuesDir, file.Name()))
		if err != nil {
			continue
		}

		var issue CoreIssue
		if err := json.Unmarshal(data, &issue); err != nil {
			continue
		}

		// Apply filters
		if component != "" && issue.Component != component {
			continue
		}
		if severity != "" && issue.Severity != severity {
			continue
		}
		if status != "" && issue.Status != status {
			continue
		}

		issues = append(issues, issue)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"issues": issues,
	})
}

func getWorkaroundsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := vars["id"]

	// First try to get issue-specific workarounds
	issueFile := filepath.Join(dataDir, "issues", issueID+".json")
	if data, err := ioutil.ReadFile(issueFile); err == nil {
		var issue CoreIssue
		if err := json.Unmarshal(data, &issue); err == nil && len(issue.Workarounds) > 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"workarounds": issue.Workarounds,
			})
			return
		}
	}

	// Load common workarounds
	commonFile := filepath.Join(dataDir, "workarounds", "common.json")
	data, err := ioutil.ReadFile(commonFile)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"workarounds": []Workaround{},
		})
		return
	}

	var workaroundData struct {
		Workarounds []Workaround `json:"workarounds"`
	}
	json.Unmarshal(data, &workaroundData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workarounds": workaroundData.Workarounds,
	})
}

func analyzeIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := vars["id"]

	// Load issue
	issueFile := filepath.Join(dataDir, "issues", issueID+".json")
	data, err := ioutil.ReadFile(issueFile)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	var issue CoreIssue
	if err := json.Unmarshal(data, &issue); err != nil {
		http.Error(w, "Error parsing issue", http.StatusInternalServerError)
		return
	}

	// Call Claude for analysis (simulated for now)
	analysis := fmt.Sprintf("Analysis for %s issue in %s component: %s\n\nSuggested fix:\n1. Check component configuration\n2. Restart the service\n3. Verify dependencies are running",
		issue.Severity, issue.Component, issue.Description)

	// Add to fix attempts
	attempt := FixAttempt{
		Timestamp: time.Now(),
		Analysis:  analysis,
		Success:   false,
		Output:    "",
	}
	issue.FixAttempts = append(issue.FixAttempts, attempt)

	// Save updated issue
	updatedData, _ := json.MarshalIndent(issue, "", "  ")
	ioutil.WriteFile(issueFile, updatedData, 0644)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"analysis":       analysis,
		"suggested_fix":  "See analysis for detailed steps",
		"confidence":     0.75,
	})
}

func reportIssueHandler(w http.ResponseWriter, r *http.Request) {
	var issue CoreIssue
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	if issue.ID == "" {
		issue.ID = fmt.Sprintf("%s-%d", issue.Component, time.Now().Unix())
	}

	// Set timestamps
	if issue.FirstSeen.IsZero() {
		issue.FirstSeen = time.Now()
	}
	issue.LastSeen = time.Now()

	// Set defaults
	if issue.Status == "" {
		issue.Status = "active"
	}
	if issue.Severity == "" {
		issue.Severity = "medium"
	}
	if issue.OccurrenceCount == 0 {
		issue.OccurrenceCount = 1
	}

	// Save issue
	issueFile := filepath.Join(dataDir, "issues", issue.ID+".json")
	data, _ := json.MarshalIndent(issue, "", "  ")
	if err := ioutil.WriteFile(issueFile, data, 0644); err != nil {
		http.Error(w, "Error saving issue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(issue)
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthHandler).Methods("GET")
	api.HandleFunc("/issues", listIssuesHandler).Methods("GET")
	api.HandleFunc("/issues", reportIssueHandler).Methods("POST")
	api.HandleFunc("/issues/{id}/workarounds", getWorkaroundsHandler).Methods("GET")
	api.HandleFunc("/issues/{id}/analyze", analyzeIssueHandler).Methods("POST")

	// Health check at root
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Apply CORS
	handler := corsMiddleware(router)

	// Get port from environment
	port := getEnv("API_PORT", getEnv("PORT", ""))

	log.Printf("Core Debugger API starting on port %s", port)
	log.Printf("Data directory: %s", dataDir)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
