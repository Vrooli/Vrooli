package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type HealthResponse struct {
	Status    string                 `json:"status"`
	Service   string                 `json:"service"`
	Timestamp int64                  `json:"timestamp"`
	Uptime    float64                `json:"uptime"`
	Checks    map[string]interface{} `json:"checks"`
	Version   string                 `json:"version"`
}

type MetricsResponse struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	TCPConnections int     `json:"tcp_connections"`
	Timestamp      string  `json:"timestamp"`
}

// Enhanced metric structures for expandable cards
type ProcessInfo struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryMB    float64 `json:"memory_mb"`
	Connections int     `json:"connections"`
	Threads     int     `json:"threads"`
	FDs         int     `json:"file_descriptors"`
	Status      string  `json:"status"`
	Goroutines  int     `json:"goroutines,omitempty"`
}

type TCPConnectionStates struct {
	Established int `json:"established"`
	TimeWait    int `json:"time_wait"`
	CloseWait   int `json:"close_wait"`
	FinWait1    int `json:"fin_wait1"`
	FinWait2    int `json:"fin_wait2"`
	SynSent     int `json:"syn_sent"`
	SynRecv     int `json:"syn_recv"`
	Closing     int `json:"closing"`
	LastAck     int `json:"last_ack"`
	Listen      int `json:"listen"`
	Total       int `json:"total"`
}

type ConnectionPoolInfo struct {
	Name        string `json:"name"`
	Active      int    `json:"active"`
	Idle        int    `json:"idle"`
	MaxSize     int    `json:"max_size"`
	Waiting     int    `json:"waiting"`
	Healthy     bool   `json:"healthy"`
	LeakRisk    string `json:"leak_risk"`
}

type NetworkStats struct {
	BandwidthInMbps  float64 `json:"bandwidth_in_mbps"`
	BandwidthOutMbps float64 `json:"bandwidth_out_mbps"`
	PacketLoss       float64 `json:"packet_loss"`
	DNSSuccessRate   float64 `json:"dns_success_rate"`
	DNSLatencyMs     float64 `json:"dns_latency_ms"`
}

type SystemHealthDetails struct {
	FileDescriptors struct {
		Used     int `json:"used"`
		Max      int `json:"max"`
		Percent  float64 `json:"percent"`
	} `json:"file_descriptors"`
	ServiceDependencies []ServiceHealth `json:"service_dependencies"`
	Certificates        []CertificateInfo `json:"certificates"`
}

type ServiceHealth struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	LatencyMs  float64 `json:"latency_ms"`
	LastCheck  string  `json:"last_check"`
	Endpoint   string  `json:"endpoint"`
}

type CertificateInfo struct {
	Domain      string `json:"domain"`
	DaysToExpiry int   `json:"days_to_expiry"`
	Status      string `json:"status"`
}

// Detailed metric responses for expandable cards
type DetailedMetricsResponse struct {
	CPUDetails struct {
		Usage        float64       `json:"usage"`
		TopProcesses []ProcessInfo `json:"top_processes"`
		LoadAverage  []float64     `json:"load_average"`
		ContextSwitches int64      `json:"context_switches"`
		Goroutines   int           `json:"total_goroutines"`
	} `json:"cpu_details"`
	MemoryDetails struct {
		Usage        float64       `json:"usage"`
		TopProcesses []ProcessInfo `json:"top_processes"`
		GrowthPatterns []struct {
			Process   string  `json:"process"`
			GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
			RiskLevel string  `json:"risk_level"`
		} `json:"growth_patterns"`
		SwapUsage struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		} `json:"swap_usage"`
		DiskUsage struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		} `json:"disk_usage"`
	} `json:"memory_details"`
	NetworkDetails struct {
		TCPStates      TCPConnectionStates   `json:"tcp_states"`
		PortUsage      struct {
			Used  int `json:"used"`
			Total int `json:"total"`
		} `json:"port_usage"`
		NetworkStats   NetworkStats          `json:"network_stats"`
		ConnectionPools []ConnectionPoolInfo `json:"connection_pools"`
	} `json:"network_details"`
	SystemDetails  SystemHealthDetails   `json:"system_details"`
	Timestamp      string               `json:"timestamp"`
}

type ProcessMonitorResponse struct {
	ProcessHealth struct {
		TotalProcesses int           `json:"total_processes"`
		ZombieProcesses []ProcessInfo `json:"zombie_processes"`
		HighThreadCount []ProcessInfo `json:"high_thread_count"`
		LeakCandidates  []ProcessInfo `json:"leak_candidates"`
	} `json:"process_health"`
	ResourceMatrix []ProcessInfo `json:"resource_matrix"`
	Timestamp      string        `json:"timestamp"`
}

type InfrastructureMonitorResponse struct {
	DatabasePools   []ConnectionPoolInfo `json:"database_pools"`
	HTTPClientPools []ConnectionPoolInfo `json:"http_client_pools"`
	MessageQueues struct {
		RedisPubSub struct {
			Subscribers int `json:"subscribers"`
			Channels    int `json:"channels"`
		} `json:"redis_pubsub"`
		BackgroundJobs struct {
			Pending   int `json:"pending"`
			Active    int `json:"active"`
			Failed    int `json:"failed"`
		} `json:"background_jobs"`
	} `json:"message_queues"`
	StorageIO struct {
		DiskQueueDepth float64 `json:"disk_queue_depth"`
		IOWaitPercent  float64 `json:"io_wait_percent"`
		ReadMBPerSec   float64 `json:"read_mb_per_sec"`
		WriteMBPerSec  float64 `json:"write_mb_per_sec"`
	} `json:"storage_io"`
	Timestamp       string               `json:"timestamp"`
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

// Global variables to store investigations and app state
var (
	latestInvestigation *Investigation
	investigations      = make(map[string]*Investigation)
	investigationsMutex sync.RWMutex
	startTime           = time.Now()
	monitoringProcessor *MonitoringProcessor
)

type ReportRequest struct {
	Type string `json:"type"`
}

func main() {
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT for compatibility
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}

	// Initialize database connection (optional - will use mock data if unavailable)
	var db *sql.DB
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		var err error
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
		} else {
			// Configure connection pool
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)

			// Implement exponential backoff for database connection
			maxRetries := 10
			baseDelay := 1 * time.Second
			maxDelay := 30 * time.Second

			log.Println("🔄 Attempting database connection with exponential backoff...")

			var pingErr error
			for attempt := 0; attempt < maxRetries; attempt++ {
				pingErr = db.Ping()
				if pingErr == nil {
					log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
					break
				}
				
				// Calculate exponential backoff delay
				delay := time.Duration(math.Min(
					float64(baseDelay) * math.Pow(2, float64(attempt)),
					float64(maxDelay),
				))
				
				// Add progressive jitter to prevent thundering herd
				jitterRange := float64(delay) * 0.25
				jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
				actualDelay := delay + jitter
				
				log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
				log.Printf("⏳ Waiting %v before next attempt", actualDelay)
				
				time.Sleep(actualDelay)
			}

			if pingErr != nil {
				log.Printf("⚠️ Database connection failed after %d attempts, continuing without database: %v", maxRetries, pingErr)
				db.Close()
				db = nil
			}
		}
	} else {
		log.Printf("DATABASE_URL not set, using mock data")
	}

	// Initialize MonitoringProcessor
	monitoringProcessor = NewMonitoringProcessor(db)

	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Metrics endpoints
	r.HandleFunc("/api/metrics/current", getCurrentMetricsHandler).Methods("GET")
	r.HandleFunc("/api/metrics/detailed", getDetailedMetricsHandler).Methods("GET")
	r.HandleFunc("/api/metrics/processes", getProcessMonitorHandler).Methods("GET")
	r.HandleFunc("/api/metrics/infrastructure", getInfrastructureMonitorHandler).Methods("GET")

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

	// New MonitoringProcessor endpoints
	r.HandleFunc("/api/monitoring/threshold-check", thresholdMonitorHandler).Methods("POST")
	r.HandleFunc("/api/monitoring/investigate-anomaly", anomalyInvestigationHandler).Methods("POST")
	r.HandleFunc("/api/monitoring/generate-report", systemReportHandler).Methods("POST")

	// Test endpoints for anomaly simulation
	r.HandleFunc("/api/test/anomaly/cpu", simulateHighCPUHandler).Methods("GET")
	
	// Debug logs endpoint for UI troubleshooting
	r.HandleFunc("/api/logs", getLogsHandler).Methods("GET")

	// Enable CORS
	r.Use(corsMiddleware)
	
	// Enable comprehensive request/response logging
	r.Use(loggingMiddleware)

	log.Printf("System Monitor API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// Removed getEnv function - no defaults allowed

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Log incoming request
		reqMsg := fmt.Sprintf(">>> INCOMING REQUEST: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		log.Printf(reqMsg)
		addLogEntry("INFO", reqMsg, "http-server")
		
		headerMsg := fmt.Sprintf("    Headers: %+v", r.Header)
		log.Printf(headerMsg)
		addLogEntry("DEBUG", headerMsg, "http-server")
		
		userAgentMsg := fmt.Sprintf("    User-Agent: %s", r.UserAgent())
		log.Printf(userAgentMsg)
		addLogEntry("DEBUG", userAgentMsg, "http-server")
		
		// Create a response writer that captures status and size
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		// Call the next handler
		next.ServeHTTP(rw, r)
		
		// Log response
		duration := time.Since(start)
		respMsg := fmt.Sprintf("<<< RESPONSE: %s %s - Status: %d - Duration: %v", 
			r.Method, r.URL.Path, rw.statusCode, duration)
		log.Printf(respMsg)
		addLogEntry("INFO", respMsg, "http-server")
		
		// Log any errors
		if rw.statusCode >= 400 {
			errorMsg := fmt.Sprintf("!!! ERROR RESPONSE: %s %s returned %d", r.Method, r.URL.Path, rw.statusCode)
			log.Printf(errorMsg)
			addLogEntry("ERROR", errorMsg, "http-server")
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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
	// Perform health checks
	checks := make(map[string]interface{})
	overallStatus := "healthy"
	
	// Check metrics collection capability
	cpuUsage := getCPUUsage()
	memUsage := getMemoryUsage()
	
	if cpuUsage >= 0 && memUsage >= 0 {
		checks["metrics_collection"] = map[string]interface{}{
			"status": "healthy",
			"message": fmt.Sprintf("Collecting metrics - CPU: %.1f%%, Memory: %.1f%%", cpuUsage, memUsage),
		}
	} else {
		checks["metrics_collection"] = map[string]interface{}{
			"status": "degraded",
			"message": "Unable to collect some metrics",
		}
		overallStatus = "degraded"
	}
	
	// Check investigation system
	investigationsMutex.RLock()
	investigationCount := len(investigations)
	investigationsMutex.RUnlock()
	
	checks["investigation_system"] = map[string]interface{}{
		"status": "healthy",
		"message": fmt.Sprintf("Investigation system operational - %d investigations tracked", investigationCount),
	}
	
	// Check filesystem access (for logs/reports)
	if _, err := os.Stat("/tmp"); err == nil {
		checks["filesystem"] = map[string]interface{}{
			"status": "healthy",
			"message": "Filesystem accessible",
		}
	} else {
		checks["filesystem"] = map[string]interface{}{
			"status": "unhealthy",
			"error": "Cannot access filesystem",
		}
		overallStatus = "unhealthy"
	}
	
	// Calculate uptime
	uptime := time.Since(startTime).Seconds()
	
	response := HealthResponse{
		Status:    overallStatus,
		Service:   "system-monitor",
		Timestamp: time.Now().Unix(),
		Uptime:    uptime,
		Checks:    checks,
		Version:   "2.0.0",
	}
	
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

// Get detailed TCP connection states
func getTCPConnectionStates() TCPConnectionStates {
	states := TCPConnectionStates{}
	
	// Parse netstat output for connection states
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | awk 'NR>2 {print $6}' | sort | uniq -c")
	output, err := cmd.Output()
	if err != nil {
		return states
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		
		count, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		
		state := strings.ToUpper(parts[1])
		switch state {
		case "ESTABLISHED":
			states.Established = count
		case "TIME_WAIT":
			states.TimeWait = count
		case "CLOSE_WAIT":
			states.CloseWait = count
		case "FIN_WAIT1":
			states.FinWait1 = count
		case "FIN_WAIT2":
			states.FinWait2 = count
		case "SYN_SENT":
			states.SynSent = count
		case "SYN_RECV":
			states.SynRecv = count
		case "CLOSING":
			states.Closing = count
		case "LAST_ACK":
			states.LastAck = count
		case "LISTEN":
			states.Listen = count
		}
		
		states.Total += count
	}
	
	return states
}

// Get top processes by CPU usage
func getTopProcessesByCPU(limit int) []ProcessInfo {
	var processes []ProcessInfo
	
	// Use ps to get process info sorted by CPU
	cmd := exec.Command("bash", "-c", 
		fmt.Sprintf("ps -eo pid,comm,%%cpu,%%mem,nlwp --sort=-%%cpu --no-headers | head -%d", limit))
	output, err := cmd.Output()
	if err != nil {
		return processes
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		
		pid, _ := strconv.Atoi(fields[0])
		cpuPercent, _ := strconv.ParseFloat(fields[2], 64)
		memoryPercent, _ := strconv.ParseFloat(fields[3], 64)
		threads, _ := strconv.Atoi(fields[4])
		
		// Get memory in MB (rough calculation)
		memoryMB := memoryPercent * getSystemMemoryGB() * 1024 / 100
		
		// Get file descriptors for this process
		fdCount := getProcessFileDescriptors(pid)
		
		processes = append(processes, ProcessInfo{
			PID:         pid,
			Name:        fields[1],
			CPUPercent:  cpuPercent,
			MemoryMB:    memoryMB,
			Threads:     threads,
			FDs:         fdCount,
			Status:      "running",
			Goroutines:  getProcessGoroutines(fields[1], pid),
		})
	}
	
	return processes
}

// Get top processes by memory usage
func getTopProcessesByMemory(limit int) []ProcessInfo {
	var processes []ProcessInfo
	
	// Use ps to get process info sorted by memory
	cmd := exec.Command("bash", "-c", 
		fmt.Sprintf("ps -eo pid,comm,%%cpu,%%mem,nlwp,rss --sort=-%%mem --no-headers | head -%d", limit))
	output, err := cmd.Output()
	if err != nil {
		return processes
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		
		pid, _ := strconv.Atoi(fields[0])
		cpuPercent, _ := strconv.ParseFloat(fields[2], 64)
		_ = fields[3] // ignore memory percent, we use RSS instead
		threads, _ := strconv.Atoi(fields[4])
		rssKB, _ := strconv.ParseFloat(fields[5], 64)
		
		// Convert RSS from KB to MB
		memoryMB := rssKB / 1024
		
		// Get file descriptors for this process
		fdCount := getProcessFileDescriptors(pid)
		
		processes = append(processes, ProcessInfo{
			PID:         pid,
			Name:        fields[1],
			CPUPercent:  cpuPercent,
			MemoryMB:    memoryMB,
			Threads:     threads,
			FDs:         fdCount,
			Status:      "running",
			Goroutines:  getProcessGoroutines(fields[1], pid),
		})
	}
	
	return processes
}

// Helper functions
func getSystemMemoryGB() float64 {
	cmd := exec.Command("bash", "-c", "free -g | grep Mem | awk '{print $2}'")
	output, err := cmd.Output()
	if err != nil {
		return 8.0 // fallback
	}
	
	memGB, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 8.0
	}
	return memGB
}

func getProcessFileDescriptors(pid int) int {
	// Count files in /proc/PID/fd/
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	files, err := os.ReadDir(fdDir)
	if err != nil {
		return 0
	}
	return len(files)
}

func getProcessGoroutines(processName string, pid int) int {
	// Only check for Go processes
	if !strings.Contains(processName, "api") && !strings.Contains(processName, "go") {
		return 0
	}
	
	// Try to get goroutine count from pprof endpoint (if available)
	// This is a mock implementation - in real world you'd query the actual pprof endpoint
	return 0
}

func getLoadAverage() []float64 {
	cmd := exec.Command("bash", "-c", "cat /proc/loadavg | awk '{print $1, $2, $3}'")
	output, err := cmd.Output()
	if err != nil {
		return []float64{0.0, 0.0, 0.0}
	}
	
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 3 {
		return []float64{0.0, 0.0, 0.0}
	}
	
	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)
	
	return []float64{load1, load5, load15}
}

func getContextSwitches() int64 {
	cmd := exec.Command("bash", "-c", "grep '^ctxt' /proc/stat | awk '{print $2}'")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	
	ctxt, _ := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	return ctxt
}

func getSystemFileDescriptors() (int, int) {
	// Get current FD count
	cmd1 := exec.Command("bash", "-c", "lsof 2>/dev/null | wc -l")
	output1, err1 := cmd1.Output()
	used := 0
	if err1 == nil {
		used, _ = strconv.Atoi(strings.TrimSpace(string(output1)))
	}
	
	// Get max FD limit
	cmd2 := exec.Command("bash", "-c", "cat /proc/sys/fs/file-max")
	output2, err2 := cmd2.Output()
	max := 65536 // fallback
	if err2 == nil {
		max, _ = strconv.Atoi(strings.TrimSpace(string(output2)))
	}
	
	return used, max
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
	updateInvestigationField(investigationID, "Progress", 10)

	// Get current system metrics for context
	cpuUsage := getCPUUsage()
	memoryUsage := getMemoryUsage()
	tcpConnections := getTCPConnections()
	timestamp := time.Now().Format(time.RFC3339)

	// Try Claude Code first, but have a good fallback
	var findings string
	var details map[string]interface{}
	
	// Load prompt template from file (hot-reloadable) with investigation context
	prompt, err := loadAndProcessPromptWithInvestigation(cpuUsage, memoryUsage, tcpConnections, timestamp, investigationID)
	if err != nil {
		log.Printf("Failed to load prompt template: %v", err)
		prompt = fmt.Sprintf(`System Anomaly Investigation
Investigation ID: %s
API Base URL: http://localhost:8080
CPU: %.2f%%, Memory: %.2f%%, TCP Connections: %d
Analyze system for anomalies and provide findings.`,
			investigationID, cpuUsage, memoryUsage, tcpConnections)
	}

	// Try Claude Code investigation with timeout
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`cd ${VROOLI_ROOT:-${HOME}/Vrooli} && echo %q | timeout 10 vrooli resource claude-code run 2>&1 || true`, prompt))
	output, err := cmd.Output()
	
	// Check if Claude Code actually worked (not just help output)
	claudeWorked := err == nil && !strings.Contains(string(output), "USAGE:") && !strings.Contains(string(output), "Failed to load library")
	
	if claudeWorked {
		findings = string(output)
		details = map[string]interface{}{
			"source": "claude_code",
			"risk_level": "low",
		}
	} else {
		// Fallback: Perform basic system analysis
		log.Printf("Claude Code unavailable, using fallback investigation")
		updateInvestigationField(investigationID, "Progress", 25)
		
		// Analyze system metrics
		riskLevel := "low"
		anomalies := []string{}
		recommendations := []string{}
		
		if cpuUsage > 80 {
			riskLevel = "high"
			anomalies = append(anomalies, fmt.Sprintf("High CPU usage: %.2f%%", cpuUsage))
			recommendations = append(recommendations, "Investigate top CPU-consuming processes with 'top' or 'htop'")
		} else if cpuUsage > 60 {
			riskLevel = "medium"
			anomalies = append(anomalies, fmt.Sprintf("Elevated CPU usage: %.2f%%", cpuUsage))
		}
		
		updateInvestigationField(investigationID, "Progress", 50)
		
		if memoryUsage > 90 {
			if riskLevel == "low" {
				riskLevel = "high"
			}
			anomalies = append(anomalies, fmt.Sprintf("Critical memory usage: %.2f%%", memoryUsage))
			recommendations = append(recommendations, "Check for memory leaks and consider increasing system RAM")
		} else if memoryUsage > 75 {
			if riskLevel == "low" {
				riskLevel = "medium"
			}
			anomalies = append(anomalies, fmt.Sprintf("High memory usage: %.2f%%", memoryUsage))
		}
		
		updateInvestigationField(investigationID, "Progress", 75)
		
		if tcpConnections > 500 {
			if riskLevel == "low" {
				riskLevel = "medium"
			}
			anomalies = append(anomalies, fmt.Sprintf("High number of TCP connections: %d", tcpConnections))
			recommendations = append(recommendations, "Review network connections with 'netstat -tuln'")
		}
		
		// Build findings report
		findings = fmt.Sprintf(`### Investigation Summary

**Status**: %s
**Investigation ID**: %s
**Timestamp**: %s

**Key Findings**:
- CPU Usage: %.2f%%
- Memory Usage: %.2f%%
- TCP Connections: %d

**Anomalies Detected**: %d
%s

**Risk Level**: %s

**Recommendations**:
%s

**Technical Details**:
System metrics are %s. %s`,
			func() string {
				if len(anomalies) > 0 {
					return "Warning"
				}
				return "Normal"
			}(),
			investigationID,
			timestamp,
			cpuUsage,
			memoryUsage,
			tcpConnections,
			len(anomalies),
			func() string {
				if len(anomalies) == 0 {
					return "- No anomalies detected"
				}
				result := ""
				for _, a := range anomalies {
					result += fmt.Sprintf("- %s\n", a)
				}
				return result
			}(),
			riskLevel,
			func() string {
				if len(recommendations) == 0 {
					return "- Continue normal monitoring"
				}
				result := ""
				for _, r := range recommendations {
					result += fmt.Sprintf("- %s\n", r)
				}
				return result
			}(),
			func() string {
				if len(anomalies) == 0 {
					return "within normal parameters"
				}
				return "showing some concerns"
			}(),
			func() string {
				if riskLevel == "high" {
					return "Immediate attention recommended."
				} else if riskLevel == "medium" {
					return "Monitor closely for escalation."
				}
				return "No immediate action required."
			}(),
		)
		
		details = map[string]interface{}{
			"source":               "fallback_analysis",
			"risk_level":          riskLevel,
			"anomalies_found":     len(anomalies),
			"recommendations_count": len(recommendations),
			"critical_issues":     riskLevel == "high",
		}
	}
	
	// Update investigation with findings
	investigationsMutex.Lock()
	if inv, exists := investigations[investigationID]; exists {
		inv.Findings = findings
		inv.Details = details
		inv.Status = "completed"
		inv.EndTime = time.Now()
		inv.Progress = 100
	}
	investigationsMutex.Unlock()
}

func loadAndProcessPromptWithInvestigation(cpuUsage, memoryUsage float64, tcpConnections int, timestamp string, investigationID string) (string, error) {
	// Get VROOLI_ROOT with proper fallback
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/root" // fallback for containers
		}
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}
	
	// Construct scenarios directory path
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")

	// Construct path to prompt file
	// Try multiple locations where the prompt file might exist
	promptPaths := []string{
		filepath.Join(scenariosDir, "system-monitor", "initialization", "claude-code", "anomaly-check.md"),
		filepath.Join("initialization", "claude-code", "anomaly-check.md"), // Relative path as fallback
	}

	var promptContent []byte
	var err error

	// Try each path until we find the file
	for _, path := range promptPaths {
		promptContent, err = os.ReadFile(path)
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

// Log storage for debugging
var (
	logBuffer []LogEntry
	logMutex sync.RWMutex
	maxLogs = 1000 // Keep last 1000 log entries
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

func addLogEntry(level, message, source string) {
	logMutex.Lock()
	defer logMutex.Unlock()
	
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Source:    source,
	}
	
	logBuffer = append(logBuffer, entry)
	if len(logBuffer) > maxLogs {
		logBuffer = logBuffer[1:] // Remove oldest entry
	}
}

func getLogsHandler(w http.ResponseWriter, r *http.Request) {
	logMutex.RLock()
	defer logMutex.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logBuffer,
		"count": len(logBuffer),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// New handlers for enhanced metrics
func getDetailedMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	timestamp := time.Now().Format(time.RFC3339)
	
	// Gather all detailed metrics
	cpuUsage := getCPUUsage()
	memoryUsage := getMemoryUsage()
	tcpStates := getTCPConnectionStates()
	topCPUProcesses := getTopProcessesByCPU(5)
	topMemoryProcesses := getTopProcessesByMemory(5)
	loadAvg := getLoadAverage()
	contextSwitches := getContextSwitches()
	fdUsed, fdMax := getSystemFileDescriptors()
	
	response := DetailedMetricsResponse{
		CPUDetails: struct {
			Usage        float64       `json:"usage"`
			TopProcesses []ProcessInfo `json:"top_processes"`
			LoadAverage  []float64     `json:"load_average"`
			ContextSwitches int64      `json:"context_switches"`
			Goroutines   int           `json:"total_goroutines"`
		}{
			Usage:           cpuUsage,
			TopProcesses:    topCPUProcesses,
			LoadAverage:     loadAvg,
			ContextSwitches: contextSwitches,
			Goroutines:      getTotalGoroutines(),
		},
		MemoryDetails: struct {
			Usage        float64       `json:"usage"`
			TopProcesses []ProcessInfo `json:"top_processes"`
			GrowthPatterns []struct {
				Process   string  `json:"process"`
				GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
				RiskLevel string  `json:"risk_level"`
			} `json:"growth_patterns"`
			SwapUsage struct {
				Used    int64   `json:"used"`
				Total   int64   `json:"total"`
				Percent float64 `json:"percent"`
			} `json:"swap_usage"`
			DiskUsage struct {
				Used    int64   `json:"used"`
				Total   int64   `json:"total"`
				Percent float64 `json:"percent"`
			} `json:"disk_usage"`
		}{
			Usage:        memoryUsage,
			TopProcesses: topMemoryProcesses,
			GrowthPatterns: getMemoryGrowthPatterns(),
			SwapUsage:    getSwapUsage(),
			DiskUsage:    getDiskUsage(),
		},
		NetworkDetails: struct {
			TCPStates      TCPConnectionStates   `json:"tcp_states"`
			PortUsage      struct {
				Used  int `json:"used"`
				Total int `json:"total"`
			} `json:"port_usage"`
			NetworkStats   NetworkStats          `json:"network_stats"`
			ConnectionPools []ConnectionPoolInfo `json:"connection_pools"`
		}{
			TCPStates:      tcpStates,
			PortUsage:      getPortUsage(),
			NetworkStats:   getNetworkStats(),
			ConnectionPools: getHTTPConnectionPools(),
		},
		SystemDetails: SystemHealthDetails{
			FileDescriptors: struct {
				Used     int `json:"used"`
				Max      int `json:"max"`
				Percent  float64 `json:"percent"`
			}{
				Used:    fdUsed,
				Max:     fdMax,
				Percent: float64(fdUsed) / float64(fdMax) * 100,
			},
			ServiceDependencies: checkServiceDependencies(),
			Certificates:        checkCertificates(),
		},
		Timestamp: timestamp,
	}
	
	json.NewEncoder(w).Encode(response)
}

func getProcessMonitorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	zombieProcesses := getZombieProcesses()
	highThreadProcesses := getHighThreadCountProcesses()
	leakCandidates := getResourceLeakCandidates()
	resourceMatrix := getTopProcessesByCPU(10) // Top 10 for resource matrix
	
	response := ProcessMonitorResponse{
		ProcessHealth: struct {
			TotalProcesses int           `json:"total_processes"`
			ZombieProcesses []ProcessInfo `json:"zombie_processes"`
			HighThreadCount []ProcessInfo `json:"high_thread_count"`
			LeakCandidates  []ProcessInfo `json:"leak_candidates"`
		}{
			TotalProcesses: getTotalProcessCount(),
			ZombieProcesses: zombieProcesses,
			HighThreadCount: highThreadProcesses,
			LeakCandidates: leakCandidates,
		},
		ResourceMatrix: resourceMatrix,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	
	json.NewEncoder(w).Encode(response)
}

func getInfrastructureMonitorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := InfrastructureMonitorResponse{
		DatabasePools:   getDatabasePools(),
		HTTPClientPools: getHTTPConnectionPools(),
		MessageQueues: struct {
			RedisPubSub struct {
				Subscribers int `json:"subscribers"`
				Channels    int `json:"channels"`
			} `json:"redis_pubsub"`
			BackgroundJobs struct {
				Pending   int `json:"pending"`
				Active    int `json:"active"`
				Failed    int `json:"failed"`
			} `json:"background_jobs"`
		}{
			RedisPubSub:    getRedisPubSubStats(),
			BackgroundJobs: getBackgroundJobStats(),
		},
		StorageIO: getStorageIOStats(),
		Timestamp: time.Now().Format(time.RFC3339),
	}
	
	json.NewEncoder(w).Encode(response)
}

// Additional helper functions for detailed metrics
func getTotalGoroutines() int {
	// Mock implementation - in reality you'd sum up goroutines from all Go processes
	return 0
}

func getMemoryGrowthPatterns() []struct {
	Process   string  `json:"process"`
	GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
	RiskLevel string  `json:"risk_level"`
} {
	// Mock implementation - in reality you'd track memory usage over time
	return []struct {
		Process   string  `json:"process"`
		GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
		RiskLevel string  `json:"risk_level"`
	}{
		{Process: "scenario-api-1", GrowthMBPerHour: 15.0, RiskLevel: "medium"},
		{Process: "postgres", GrowthMBPerHour: 2.0, RiskLevel: "low"},
	}
}

func getSwapUsage() struct {
	Used    int64   `json:"used"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
} {
	cmd := exec.Command("bash", "-c", "free -b | grep Swap | awk '{print $2, $3}'")
	output, err := cmd.Output()
	if err != nil {
		return struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		}{0, 0, 0.0}
	}
	
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 2 {
		return struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		}{0, 0, 0.0}
	}
	
	total, _ := strconv.ParseInt(fields[0], 10, 64)
	used, _ := strconv.ParseInt(fields[1], 10, 64)
	percent := 0.0
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}
	
	return struct {
		Used    int64   `json:"used"`
		Total   int64   `json:"total"`
		Percent float64 `json:"percent"`
	}{used, total, percent}
}

func getDiskUsage() struct {
	Used    int64   `json:"used"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
} {
	cmd := exec.Command("bash", "-c", "df -B1 / | tail -1 | awk '{print $2, $3}'")
	output, err := cmd.Output()
	if err != nil {
		return struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		}{0, 0, 0.0}
	}
	
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 2 {
		return struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		}{0, 0, 0.0}
	}
	
	total, _ := strconv.ParseInt(fields[0], 10, 64)
	used, _ := strconv.ParseInt(fields[1], 10, 64)
	percent := 0.0
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}
	
	return struct {
		Used    int64   `json:"used"`
		Total   int64   `json:"total"`
		Percent float64 `json:"percent"`
	}{used, total, percent}
}

func getPortUsage() struct {
	Used  int `json:"used"`
	Total int `json:"total"`
} {
	// Count used ephemeral ports
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | grep -E ':([3-6][0-9]{4})' | wc -l")
	output, err := cmd.Output()
	used := 0
	if err == nil {
		used, _ = strconv.Atoi(strings.TrimSpace(string(output)))
	}
	
	// Typical ephemeral port range is 32768-65535
	return struct {
		Used  int `json:"used"`
		Total int `json:"total"`
	}{used, 32767}
}

func getNetworkStats() NetworkStats {
	// Mock implementation - in reality you'd read from /proc/net/dev
	return NetworkStats{
		BandwidthInMbps:  12.5,
		BandwidthOutMbps: 8.2,
		PacketLoss:       0.1,
		DNSSuccessRate:   99.2,
		DNSLatencyMs:     15.0,
	}
}

func getHTTPConnectionPools() []ConnectionPoolInfo {
	// Mock implementation - in reality you'd query application metrics
	return []ConnectionPoolInfo{
		{
			Name:     "scenario-api-1->ollama",
			Active:   3,
			Idle:     7,
			MaxSize:  10,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
		{
			Name:     "scenario-api-2->n8n",
			Active:   10,
			Idle:     0,
			MaxSize:  10,
			Waiting:  5,
			Healthy:  false,
			LeakRisk: "high",
		},
	}
}

func checkServiceDependencies() []ServiceHealth {
	// Check common services
	services := []struct {
		name     string
		endpoint string
	}{
		{"postgres", "localhost:5432"},
		{"redis", "localhost:6379"},
		{"n8n", "localhost:5678"},
		{"ollama", "localhost:11434"},
	}
	
	var results []ServiceHealth
	for _, service := range services {
		status := "healthy"
		latency := 0.0
		
		// Quick TCP connection test
		start := time.Now()
		cmd := exec.Command("timeout", "2", "bash", "-c", 
			fmt.Sprintf("echo > /dev/tcp/%s", service.endpoint))
		err := cmd.Run()
		latency = float64(time.Since(start).Milliseconds())
		
		if err != nil {
			status = "unhealthy"
			latency = -1
		}
		
		results = append(results, ServiceHealth{
			Name:      service.name,
			Status:    status,
			LatencyMs: latency,
			LastCheck: time.Now().Format(time.RFC3339),
			Endpoint:  service.endpoint,
		})
	}
	
	return results
}

func checkCertificates() []CertificateInfo {
	// Mock implementation - in reality you'd check actual certificates
	return []CertificateInfo{
		{
			Domain:       "api.local",
			DaysToExpiry: 89,
			Status:       "valid",
		},
	}
}

func getZombieProcesses() []ProcessInfo {
	// Find zombie processes
	cmd := exec.Command("bash", "-c", "ps -eo pid,comm,stat | grep ' Z' | head -10")
	output, err := cmd.Output()
	if err != nil {
		return []ProcessInfo{}
	}
	
	var zombies []ProcessInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid, _ := strconv.Atoi(fields[0])
			zombies = append(zombies, ProcessInfo{
				PID:    pid,
				Name:   fields[1],
				Status: "zombie",
			})
		}
	}
	
	return zombies
}

func getHighThreadCountProcesses() []ProcessInfo {
	// Find processes with high thread counts
	cmd := exec.Command("bash", "-c", "ps -eo pid,comm,nlwp --sort=-nlwp --no-headers | head -5")
	output, err := cmd.Output()
	if err != nil {
		return []ProcessInfo{}
	}
	
	var processes []ProcessInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid, _ := strconv.Atoi(fields[0])
			threads, _ := strconv.Atoi(fields[2])
			
			if threads > 20 { // Only include processes with >20 threads
				processes = append(processes, ProcessInfo{
					PID:     pid,
					Name:    fields[1],
					Threads: threads,
					Status:  "high_threads",
				})
			}
		}
	}
	
	return processes
}

func getResourceLeakCandidates() []ProcessInfo {
	// Mock implementation - in reality you'd analyze FD growth, memory growth, etc.
	return []ProcessInfo{
		{
			PID:    1234,
			Name:   "scenario-api-1",
			Status: "fd_leak_risk",
			FDs:    512,
		},
	}
}

func getTotalProcessCount() int {
	cmd := exec.Command("bash", "-c", "ps -e --no-headers | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	
	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count
}

func getDatabasePools() []ConnectionPoolInfo {
	// Mock implementation - in reality you'd query database metrics
	return []ConnectionPoolInfo{
		{
			Name:     "postgres-main",
			Active:   8,
			Idle:     2,
			MaxSize:  10,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
		{
			Name:     "redis-main",
			Active:   45,
			Idle:     55,
			MaxSize:  100,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
	}
}

func getRedisPubSubStats() struct {
	Subscribers int `json:"subscribers"`
	Channels    int `json:"channels"`
} {
	// Mock implementation
	return struct {
		Subscribers int `json:"subscribers"`
		Channels    int `json:"channels"`
	}{12, 5}
}

func getBackgroundJobStats() struct {
	Pending   int `json:"pending"`
	Active    int `json:"active"`
	Failed    int `json:"failed"`
} {
	// Mock implementation
	return struct {
		Pending   int `json:"pending"`
		Active    int `json:"active"`
		Failed    int `json:"failed"`
	}{3, 1, 0}
}

func getStorageIOStats() struct {
	DiskQueueDepth float64 `json:"disk_queue_depth"`
	IOWaitPercent  float64 `json:"io_wait_percent"`
	ReadMBPerSec   float64 `json:"read_mb_per_sec"`
	WriteMBPerSec  float64 `json:"write_mb_per_sec"`
} {
	// Get I/O wait from /proc/stat
	cmd := exec.Command("bash", "-c", "grep '^cpu ' /proc/stat | awk '{print ($5/($2+$3+$4+$5+$6+$7+$8))*100}'")
	output, err := cmd.Output()
	iowait := 0.0
	if err == nil {
		iowait, _ = strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	}
	
	return struct {
		DiskQueueDepth float64 `json:"disk_queue_depth"`
		IOWaitPercent  float64 `json:"io_wait_percent"`
		ReadMBPerSec   float64 `json:"read_mb_per_sec"`
		WriteMBPerSec  float64 `json:"write_mb_per_sec"`
	}{0.2, iowait, 15.0, 8.0}
}

// New handler functions using MonitoringProcessor

func thresholdMonitorHandler(w http.ResponseWriter, r *http.Request) {
	var req ThresholdMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	result, err := monitoringProcessor.MonitorThresholds(ctx, req)
	if err != nil {
		log.Printf("Error monitoring thresholds: %v", err)
		http.Error(w, "Failed to monitor thresholds", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func anomalyInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	var req AnomalyInvestigationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	result, err := monitoringProcessor.InvestigateAnomaly(ctx, req)
	if err != nil {
		log.Printf("Error investigating anomaly: %v", err)
		http.Error(w, "Failed to investigate anomaly", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func systemReportHandler(w http.ResponseWriter, r *http.Request) {
	var req ReportGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	result, err := monitoringProcessor.GenerateReport(ctx, req)
	if err != nil {
		log.Printf("Error generating report: %v", err)
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
