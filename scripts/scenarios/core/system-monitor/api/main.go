package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
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
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	AnomalyID string    `json:"anomaly_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Findings  string    `json:"findings,omitempty"`
}

type ReportRequest struct {
	Type string `json:"type"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Metrics endpoints
	r.HandleFunc("/api/metrics/current", getCurrentMetricsHandler).Methods("GET")

	// Investigation endpoints
	r.HandleFunc("/api/investigations/latest", getLatestInvestigationHandler).Methods("GET")

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
	// Mock investigation for testing
	investigation := Investigation{
		ID:        "inv_" + strconv.FormatInt(time.Now().Unix(), 10),
		Status:    "completed",
		AnomalyID: "cpu_spike_001",
		StartTime: time.Now().Add(-5 * time.Minute),
		EndTime:   time.Now(),
		Findings:  "High CPU usage detected due to intensive background process. Recommended: Monitor process ID 1234 for resource optimization.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(investigation)
}

func generateReportHandler(w http.ResponseWriter, r *http.Request) {
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Mock report generation
	response := map[string]interface{}{
		"status":      "success",
		"report_id":   "report_" + strconv.FormatInt(time.Now().Unix(), 10),
		"report_type": req.Type,
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
	return float64(15.5 + (time.Now().Second() % 30)) // Mock varying CPU usage
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
	return float64(45.2 + (time.Now().Second() % 20)) // Mock varying memory usage
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