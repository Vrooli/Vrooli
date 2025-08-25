package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type MetricsResponse struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	TCPConnections int     `json:"tcp_connections"`
	Timestamp      string  `json:"timestamp"`
}

type Investigation struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"` // queued, in_progress, completed, failed
	AnomalyID string                 `json:"anomaly_id"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time,omitempty"`
	Findings  string                 `json:"findings,omitempty"`
	Progress  int                    `json:"progress"`          // 0-100
	Details   map[string]interface{} `json:"details,omitempty"` // Structured data
	Steps     []InvestigationStep    `json:"steps,omitempty"`
}

type InvestigationStep struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Findings  string    `json:"findings,omitempty"`
}

// Global variables to store investigations
var (
	latestInvestigation *Investigation
	investigations      = make(map[string]*Investigation)
	investigationsMutex sync.RWMutex
)

type ReportRequest struct {
	Type string `json:"type"`
}

func main() {
	// Try SERVICE_PORT first (allocated by Vrooli), then PORT, then fallback
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Metrics endpoints
	r.HandleFunc("/api/metrics/current", getCurrentMetricsHandler).Methods("GET")

	// Investigation endpoints
	r.HandleFunc("/api/investigations/latest", getLatestInvestigationHandler).Methods("GET")
	r.HandleFunc("/api/investigations/trigger", triggerInvestigationHandler).Methods("POST")
	r.HandleFunc("/api/investigations/{id}", getInvestigationHandler).Methods("GET")
	r.HandleFunc("/api/investigations/{id}/status", updateInvestigationStatusHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/findings", updateInvestigationFindingsHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/progress", updateInvestigationProgressHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/step", addInvestigationStepHandler).Methods("POST")

	// Report endpoints
	r.HandleFunc("/api/reports/generate", generateReportHandler).Methods("POST")

	// Test endpoints for anomaly simulation
	r.HandleFunc("/api/test/anomaly/cpu", simulateHighCPUHandler).Methods("GET")

	// Enable CORS
	r.Use(corsMiddleware)

	log.Printf("System Monitor API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{Status: "healthy"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCurrentMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := MetricsResponse{
		CPUUsage:       getCPUUsage(),
		MemoryUsage:    getMemoryUsage(),
		TCPConnections: getTCPConnections(),
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func getLatestInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	if latestInvestigation == nil {
		// Default mock investigation if none triggered yet
		investigation := Investigation{
			ID:        "inv_default",
			Status:    "pending",
			AnomalyID: "none",
			StartTime: time.Now(),
			Findings:  "No investigations have been triggered yet. Click 'RUN ANOMALY CHECK' to start an investigation.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(investigation)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latestInvestigation)
}

func generateReportHandler(w http.ResponseWriter, r *http.Request) {
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Mock report generation
	response := map[string]interface{}{
		"status":       "success",
		"report_id":    "report_" + strconv.FormatInt(time.Now().Unix(), 10),
		"report_type":  req.Type,
		"generated_at": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func simulateHighCPUHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate high CPU detection
	response := map[string]interface{}{
		"anomaly_detected": true,
		"anomaly_type":     "high_cpu",
		"cpu_usage":        95.5,
		"threshold":        80.0,
		"detected_at":      time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCPUUsage() float64 {
	// Get CPU usage using system commands
	if runtime.GOOS == "linux" {
		cmd := exec.Command("bash", "-c", "grep 'cpu ' /proc/stat | awk '{usage=($2+$4)*100/($2+$3+$4+$5)} END {print usage}'")
		output, err := cmd.Output()
		if err != nil {
			return 0.0
		}

		usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err != nil {
			return 0.0
		}
		return usage
	}

	// Fallback for non-Linux systems
	return float64(15 + (time.Now().Second() % 30)) // Mock varying CPU usage
}

func getMemoryUsage() float64 {
	// Get memory usage using system commands
	if runtime.GOOS == "linux" {
		cmd := exec.Command("bash", "-c", "free | grep Mem | awk '{print ($3/$2) * 100.0}'")
		output, err := cmd.Output()
		if err != nil {
			return 0.0
		}

		usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err != nil {
			return 0.0
		}
		return usage
	}

	// Fallback for non-Linux systems
	return float64(45 + (time.Now().Second() % 20)) // Mock varying memory usage
}

func getTCPConnections() int {
	// Get TCP connection count using netstat
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | grep ESTABLISHED | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0
	}
	return count
}

func triggerInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	// Generate new investigation ID
	investigationID := "inv_" + strconv.FormatInt(time.Now().Unix(), 10)

	// Create investigation with queued status
	investigation := &Investigation{
		ID:        investigationID,
		Status:    "queued",
		AnomalyID: "ai_investigation_" + strconv.FormatInt(time.Now().Unix(), 10),
		StartTime: time.Now(),
		Findings:  "Investigation queued for processing...",
		Progress:  0,
		Details:   make(map[string]interface{}),
		Steps:     []InvestigationStep{},
	}

	// Store investigation
	investigationsMutex.Lock()
	investigations[investigationID] = investigation
	latestInvestigation = investigation
	investigationsMutex.Unlock()

	// Start investigation in background
	go func() {
		runClaudeInvestigation(investigationID)
	}()

	// Return immediate response with API info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "queued",
		"investigation_id": investigationID,
		"api_base_url":     "http://localhost:8080",
		"message":          "Investigation queued for Claude Code processing",
	})
}

func runClaudeInvestigation(investigationID string) {
	// Update status to in_progress
	updateInvestigationField(investigationID, "Status", "in_progress")

	// Get current system metrics for context
	cpuUsage := getCPUUsage()
	memoryUsage := getMemoryUsage()
	tcpConnections := getTCPConnections()
	timestamp := time.Now().Format(time.RFC3339)

	// Load prompt template from file (hot-reloadable) with investigation context
	prompt, err := loadAndProcessPromptWithInvestigation(cpuUsage, memoryUsage, tcpConnections, timestamp, investigationID)
	if err != nil {
		log.Printf("Failed to load prompt template: %v", err)
		// Fallback to basic prompt if file cannot be read
		prompt = fmt.Sprintf(`System Anomaly Investigation
Investigation ID: %s
API Base URL: http://localhost:8080
CPU: %.2f%%, Memory: %.2f%%, TCP Connections: %d
Analyze system for anomalies and provide findings.`,
			investigationID, cpuUsage, memoryUsage, tcpConnections)
	}

	// Execute Claude Code investigation with timeout
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`cd /home/matthalloran8/Vrooli && echo %q | timeout 300 vrooli resource claude-code`, prompt))
	output, err := cmd.Output()

	if err != nil {
		// Investigation failed
		updateInvestigationField(investigationID, "Status", "failed")
		updateInvestigationField(investigationID, "EndTime", time.Now())
		updateInvestigationField(investigationID, "Findings", fmt.Sprintf("Investigation failed: %v\n\nPartial output:\n%s", err, string(output)))
		return
	}

	// Store raw output as fallback
	updateInvestigationField(investigationID, "Findings", string(output))

	// Check if investigation was marked as completed via API
	investigationsMutex.RLock()
	inv, exists := investigations[investigationID]
	investigationsMutex.RUnlock()

	if exists && inv.Status != "completed" {
		// If not marked as completed via API, mark it now
		updateInvestigationField(investigationID, "Status", "completed")
		updateInvestigationField(investigationID, "EndTime", time.Now())
		updateInvestigationField(investigationID, "Progress", 100)
	}
}

func loadAndProcessPromptWithInvestigation(cpuUsage, memoryUsage float64, tcpConnections int, timestamp string, investigationID string) (string, error) {
	// Construct path to prompt file
	// Try both the local scenario path and the generated app path
	promptPaths := []string{
		"/home/matthalloran8/Vrooli/scenarios/system-monitor/initialization/claude-code/anomaly-check.md",
		"/home/matthalloran8/generated-apps/system-monitor/initialization/claude-code/anomaly-check.md",
		"initialization/claude-code/anomaly-check.md", // Relative path as fallback
	}

	var promptContent []byte
	var err error

	// Try each path until we find the file
	for _, path := range promptPaths {
		promptContent, err = ioutil.ReadFile(path)
		if err == nil {
			log.Printf("Loaded prompt from: %s", path)
			break
		}
	}

	if err != nil {
		return "", fmt.Errorf("could not read prompt file from any location: %v", err)
	}

	// Replace placeholders with actual values
	prompt := string(promptContent)
	prompt = strings.ReplaceAll(prompt, "{{CPU_USAGE}}", fmt.Sprintf("%.2f", cpuUsage))
	prompt = strings.ReplaceAll(prompt, "{{MEMORY_USAGE}}", fmt.Sprintf("%.2f", memoryUsage))
	prompt = strings.ReplaceAll(prompt, "{{TCP_CONNECTIONS}}", strconv.Itoa(tcpConnections))
	prompt = strings.ReplaceAll(prompt, "{{TIMESTAMP}}", timestamp)
	prompt = strings.ReplaceAll(prompt, "{{INVESTIGATION_ID}}", investigationID)
	prompt = strings.ReplaceAll(prompt, "{{API_BASE_URL}}", "http://localhost:8080")

	return prompt, nil
}

// Helper function to update investigation fields
func updateInvestigationField(id string, field string, value interface{}) {
	investigationsMutex.Lock()
	defer investigationsMutex.Unlock()

	if inv, exists := investigations[id]; exists {
		switch field {
		case "Status":
			inv.Status = value.(string)
		case "Findings":
			inv.Findings = value.(string)
		case "Progress":
			inv.Progress = value.(int)
		case "EndTime":
			inv.EndTime = value.(time.Time)
		}
	}
}

// New handler functions for investigation updates
func getInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	investigationsMutex.RLock()
	inv, exists := investigations[id]
	investigationsMutex.RUnlock()

	if !exists {
		http.Error(w, "Investigation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inv)
}

func updateInvestigationStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updateInvestigationField(id, "Status", req.Status)

	if req.Status == "completed" || req.Status == "failed" {
		updateInvestigationField(id, "EndTime", time.Now())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func updateInvestigationFindingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Findings string                 `json:"findings"`
		Details  map[string]interface{} `json:"details,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	investigationsMutex.Lock()
	if inv, exists := investigations[id]; exists {
		inv.Findings = req.Findings
		if req.Details != nil {
			for k, v := range req.Details {
				inv.Details[k] = v
			}
		}
	}
	investigationsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func updateInvestigationProgressHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Progress int `json:"progress"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updateInvestigationField(id, "Progress", req.Progress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func addInvestigationStepHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var step InvestigationStep
	if err := json.NewDecoder(r.Body).Decode(&step); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	step.StartTime = time.Now()

	investigationsMutex.Lock()
	if inv, exists := investigations[id]; exists {
		inv.Steps = append(inv.Steps, step)
	}
	investigationsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "step_added"})
}
