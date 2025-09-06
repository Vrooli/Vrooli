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
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

type App struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	ScenarioName string                 `json:"scenario_name" db:"scenario_name"`
	Path         string                 `json:"path" db:"path"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	Status       string                 `json:"status" db:"status"`
	PortMappings map[string]interface{} `json:"port_mappings" db:"port_mappings"`
	Environment  map[string]interface{} `json:"environment" db:"environment"`
	Config       map[string]interface{} `json:"config" db:"config"`
}

type AppStatus struct {
	ID         string    `json:"id" db:"id"`
	AppID      string    `json:"app_id" db:"app_id"`
	Status     string    `json:"status" db:"status"`
	CPUUsage   float64   `json:"cpu_usage" db:"cpu_usage"`
	MemUsage   float64   `json:"memory_usage" db:"memory_usage"`
	DiskUsage  float64   `json:"disk_usage" db:"disk_usage"`
	NetworkIn  int64     `json:"network_in" db:"network_in"`
	NetworkOut int64     `json:"network_out" db:"network_out"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// OrchestratorResponse represents the response from vrooli scenario status --json
type OrchestratorResponse struct {
	Total   int                   `json:"total"`
	Running int                   `json:"running"`
	Apps    []OrchestratorApp     `json:"apps"`
}

type OrchestratorApp struct {
	Name           string            `json:"name"`
	Status         string            `json:"status"`
	ActualHealth   string            `json:"actual_health,omitempty"`
	AllocatedPorts map[string]int    `json:"allocated_ports,omitempty"`
	PID            int               `json:"pid,omitempty"`
	StartedAt      string            `json:"started_at,omitempty"`
	StoppedAt      string            `json:"stopped_at,omitempty"`
	RestartCount   int               `json:"restart_count,omitempty"`
}

// Resource represents a Vrooli resource
type Resource struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
}

// VrooliResource represents the response from vrooli resource status --json
type VrooliResource struct {
	Name    string `json:"Name"`
	Enabled bool   `json:"Enabled"`
	Running bool   `json:"Running"`
}

type Server struct {
	db          *sql.DB
	redis       *redis.Client
	docker      *client.Client
	upgrader    websocket.Upgrader
	port        string
	n8nBaseURL  string
	nodeRedURL  string
}

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatal("Failed to initialize server:", err)
	}
	defer server.Close()

	r := setupRoutes(server)
	
	log.Printf("🚀 App Monitor API server starting on port %s", server.port)
	if err := r.Run(":" + server.port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func NewServer() (*Server, error) {
	port := getEnv("API_PORT", getEnv("PORT", "8080"))
	postgresURL := getEnv("POSTGRES_URL", "")
	redisURL := getEnv("REDIS_URL", "")
	n8nBaseURL := getEnv("N8N_BASE_URL", "http://localhost:5678")
	nodeRedURL := getEnv("NODE_RED_BASE_URL", "http://localhost:1880")

	// Initialize database connection (optional for now since we use vrooli CLI)
	var db *sql.DB
	if postgresURL != "" {
		var err error
		db, err = sql.Open("postgres", postgresURL)
		if err != nil {
			log.Printf("Warning: failed to connect to database: %v (continuing without DB)", err)
			db = nil
		} else if err := db.Ping(); err != nil {
			log.Printf("Warning: failed to ping database: %v (continuing without DB)", err)
			db = nil
		}
	} else {
		log.Printf("Warning: POSTGRES_URL not provided (continuing without DB)")
		db = nil
	}

	// Initialize Redis connection (optional)
	var rdb *redis.Client
	if redisURL != "" {
		redisOpts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("Warning: failed to parse Redis URL: %v (continuing without Redis)", err)
			rdb = nil
		} else {
			rdb = redis.NewClient(redisOpts)
			if err := rdb.Ping(context.Background()).Err(); err != nil {
				log.Printf("Warning: failed to connect to Redis: %v (continuing without Redis)", err)
				rdb = nil
			}
		}
	} else {
		log.Printf("Warning: REDIS_URL not provided (continuing without Redis)")
		rdb = nil
	}

	// Initialize Docker client (optional)
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Warning: failed to create Docker client: %v (continuing without Docker)", err)
		dockerClient = nil
	}

	return &Server{
		db:          db,
		redis:       rdb,
		docker:      dockerClient,
		port:        port,
		n8nBaseURL:  n8nBaseURL,
		nodeRedURL:  nodeRedURL,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}, nil
}

func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
	if s.redis != nil {
		s.redis.Close()
	}
	if s.docker != nil {
		s.docker.Close()
	}
}

func setupRoutes(s *Server) *gin.Engine {
	r := gin.Default()

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health endpoints
	r.GET("/health", s.healthCheck)
	r.GET("/api/health", s.apiHealth)

	// System endpoints
	r.GET("/api/system/info", s.getSystemInfo)
	r.GET("/api/system/metrics", s.getSystemMetrics)

	// App management endpoints
	r.GET("/api/apps", s.getApps)
	r.GET("/api/apps/:id", s.getApp)
	r.POST("/api/apps/:id/start", s.startApp)
	r.POST("/api/apps/:id/stop", s.stopApp)
	r.GET("/api/apps/:id/logs", s.getAppLogs)
	r.GET("/api/apps/:id/logs/lifecycle", s.getAppLifecycleLogs)
	r.GET("/api/apps/:id/logs/background", s.getAppBackgroundLogs)
	r.GET("/api/apps/:id/metrics", s.getAppMetrics)
	
	// Log endpoints for scenarios using app name
	r.GET("/api/logs/:appName", s.getScenarioLogs)
	
	// Resource endpoints
	r.GET("/api/resources", s.getResources)

	// Docker integration endpoints
	r.GET("/api/docker/info", s.getDockerInfo)
	r.GET("/api/docker/containers", s.getContainers)

	// WebSocket endpoint for real-time updates
	r.GET("/ws", s.handleWebSocket)

	return r
}

func (s *Server) healthCheck(c *gin.Context) {
	ctx := context.Background()
	services := make(map[string]string)

	// Check database
	if s.db == nil {
		services["database"] = "disabled"
	} else if err := s.db.Ping(); err != nil {
		services["database"] = "error"
	} else {
		services["database"] = "ok"
	}

	// Check Redis
	if s.redis == nil {
		services["redis"] = "disabled"
	} else if err := s.redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "error"
	} else {
		services["redis"] = "ok"
	}

	// Check Docker
	if s.docker == nil {
		services["docker"] = "disabled"
	} else if _, err := s.docker.Ping(ctx); err != nil {
		services["docker"] = "error"
	} else {
		services["docker"] = "ok"
	}

	status := "healthy"
	for _, service := range services {
		if service == "error" {
			status = "unhealthy"
			break
		}
	}

	c.JSON(http.StatusOK, HealthStatus{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
		Services:  services,
	})
}

func (s *Server) apiHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"api":       "app-monitor",
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) getSystemMetrics(c *gin.Context) {
	// Get system metrics using standard Unix commands
	metrics := make(map[string]interface{})
	
	// CPU usage - get idle percentage and calculate usage
	cpuCmd := exec.Command("bash", "-c", "top -b -n1 | grep '%Cpu' | awk '{print 100-$8}'")
	if cpuOutput, err := cpuCmd.Output(); err == nil {
		cpuStr := strings.TrimSpace(string(cpuOutput))
		// Remove any non-numeric characters
		cpuStr = strings.TrimSuffix(cpuStr, "%")
		if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
			metrics["cpu"] = cpu
		}
	}
	
	// Fallback CPU calculation using /proc/stat
	if _, ok := metrics["cpu"]; !ok {
		cpuCmd2 := exec.Command("bash", "-c", "grep 'cpu ' /proc/stat | head -1 | awk '{idle=$5; total=0; for(i=2;i<=NF;i++){total+=$i}; print 100*(1-idle/total)}'")
		if cpuOutput2, err := cpuCmd2.Output(); err == nil {
			if cpu2, err := strconv.ParseFloat(strings.TrimSpace(string(cpuOutput2)), 64); err == nil {
				metrics["cpu"] = cpu2
			}
		}
	}
	
	// Memory usage - correct column mapping for modern free output
	// $1: "Mem:", $2: total, $3: used
	memCmd := exec.Command("bash", "-c", "free -m | awk 'NR==2{printf \"%.1f\", $3*100/$2}'")
	memOutput, err := memCmd.Output()
	if err == nil {
		memStr := strings.TrimSpace(string(memOutput))
		if mem, err := strconv.ParseFloat(memStr, 64); err == nil && mem <= 100 {
			metrics["memory"] = mem
		}
	}
	
	// Another memory fallback using /proc/meminfo
	if _, ok := metrics["memory"]; !ok {
		memCmd2 := exec.Command("bash", "-c", "awk '/MemTotal:/ {total=$2} /MemAvailable:/ {available=$2} END {printf \"%.1f\", 100*(1-available/total)}' /proc/meminfo")
		if memOutput2, err := memCmd2.Output(); err == nil {
			if mem2, err := strconv.ParseFloat(strings.TrimSpace(string(memOutput2)), 64); err == nil && mem2 >= 0 && mem2 <= 100 {
				metrics["memory"] = mem2
			}
		}
	}
	
	// Disk usage
	diskCmd := exec.Command("bash", "-c", "df / | awk 'NR==2 {print $5}' | sed 's/%//'")
	if diskOutput, err := diskCmd.Output(); err == nil {
		diskStr := strings.TrimSpace(string(diskOutput))
		if disk, err := strconv.ParseFloat(diskStr, 64); err == nil && disk >= 0 && disk <= 100 {
			metrics["disk"] = disk
		}
	}
	
	// Network I/O rate using vnstat if available, fallback to simple calculation
	netCmd := exec.Command("bash", "-c", "command -v vnstat >/dev/null && vnstat -i eth0 --json | jq -r '.interfaces[0].traffic.total.rx + .interfaces[0].traffic.total.tx' 2>/dev/null || echo '0'")
	if netOutput, err := netCmd.Output(); err == nil {
		netStr := strings.TrimSpace(string(netOutput))
		if net, err := strconv.ParseFloat(netStr, 64); err == nil && net > 0 {
			// Convert from bytes to KB/s (approximation since vnstat gives totals)
			kbPerSec := net / 1024 / 60 // Rough estimate assuming 1-minute average
			metrics["network"] = math.Min(math.Round(kbPerSec*100)/100, 10000) // Cap at reasonable max
		} else {
			// Fallback: use ss command to get current connection bandwidth usage
			ssCmd := exec.Command("bash", "-c", "ss -i | grep -E 'bytes_sent|bytes_received' | awk '{sum+=$2} END {print sum/1024}' 2>/dev/null || echo '0'")
			if ssOutput, err := ssCmd.Output(); err == nil {
				ssStr := strings.TrimSpace(string(ssOutput))
				if ssNet, err := strconv.ParseFloat(ssStr, 64); err == nil {
					metrics["network"] = math.Round(ssNet*100) / 100
				}
			}
		}
	}
	
	// Add default values if metrics are missing or invalid
	if cpu, ok := metrics["cpu"].(float64); !ok || cpu < 0 || cpu > 100 {
		metrics["cpu"] = 0
	}
	if mem, ok := metrics["memory"].(float64); !ok || mem < 0 || mem > 100 {
		metrics["memory"] = 0
	}
	if disk, ok := metrics["disk"].(float64); !ok || disk < 0 || disk > 100 {
		metrics["disk"] = 0
	}
	if _, ok := metrics["network"]; !ok {
		metrics["network"] = 0
	}
	
	c.JSON(http.StatusOK, metrics)
}

func (s *Server) getSystemInfo(c *gin.Context) {
	// Get orchestrator PID first
	pidCmd := exec.Command("bash", "-c", "ps aux | grep 'enhanced_orchestrator.py' | grep -v grep | awk '{print $2}' | head -1")
	pidOutput, err := pidCmd.Output()
	if err != nil {
		log.Printf("Failed to get orchestrator PID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orchestrator process"})
		return
	}

	pid := strings.TrimSpace(string(pidOutput))
	if pid == "" {
		c.JSON(http.StatusOK, gin.H{
			"orchestrator_running": false,
			"uptime": "00:00:00",
			"uptime_seconds": 0,
		})
		return
	}

	// Get uptime using the PID
	uptimeCmd := exec.Command("ps", "-p", pid, "-o", "etime=")
	uptimeOutput, err := uptimeCmd.Output()
	if err != nil {
		log.Printf("Failed to get orchestrator uptime: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orchestrator uptime"})
		return
	}

	uptime := strings.TrimSpace(string(uptimeOutput))
	
	// Parse uptime to seconds for easier use
	uptimeSeconds := parseUptimeToSeconds(uptime)

	// Get orchestrator status from API
	orchStatus := make(map[string]interface{})
	resp, err := http.Get("http://localhost:9500/status")
	if err == nil {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&orchStatus)
	}

	c.JSON(http.StatusOK, gin.H{
		"orchestrator_running": true,
		"orchestrator_pid": pid,
		"uptime": uptime,
		"uptime_seconds": uptimeSeconds,
		"orchestrator_status": orchStatus,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Helper function to parse uptime string to seconds
func parseUptimeToSeconds(uptime string) int {
	uptime = strings.TrimSpace(uptime)
	parts := strings.Split(uptime, ":")
	
	switch len(parts) {
	case 2: // MM:SS
		minutes, _ := strconv.Atoi(parts[0])
		seconds, _ := strconv.Atoi(parts[1])
		return minutes*60 + seconds
	case 3: // HH:MM:SS or DD-HH:MM
		if strings.Contains(parts[0], "-") {
			// DD-HH:MM format
			dayHour := strings.Split(parts[0], "-")
			days, _ := strconv.Atoi(dayHour[0])
			hours, _ := strconv.Atoi(dayHour[1])
			minutes, _ := strconv.Atoi(parts[1])
			seconds, _ := strconv.Atoi(parts[2])
			return days*86400 + hours*3600 + minutes*60 + seconds
		} else {
			// HH:MM:SS format
			hours, _ := strconv.Atoi(parts[0])
			minutes, _ := strconv.Atoi(parts[1])
			seconds, _ := strconv.Atoi(parts[2])
			return hours*3600 + minutes*60 + seconds
		}
	default:
		return 0
	}
}

func (s *Server) getApps(c *gin.Context) {
	// Execute vrooli scenario status --json to get real-time app status
	cmd := exec.Command("vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to execute vrooli scenario status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch apps from orchestrator"})
		return
	}

	// Parse the orchestrator response
	var orchestratorResp OrchestratorResponse
	if err := json.Unmarshal(output, &orchestratorResp); err != nil {
		log.Printf("Failed to parse orchestrator response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse orchestrator response"})
		return
	}

	// Convert orchestrator apps to our App format
	apps := make([]App, 0, len(orchestratorResp.Apps))
	for _, orchApp := range orchestratorResp.Apps {
		// Map orchestrator status to our status format
		status := orchApp.Status
		if orchApp.Status == "running" && orchApp.ActualHealth != "" {
			// Use actual health for running apps
			if orchApp.ActualHealth == "degraded" || orchApp.ActualHealth == "unhealthy" {
				status = orchApp.ActualHealth
			}
		}

		// Format ports for display
		portMappings := make(map[string]interface{})
		for name, port := range orchApp.AllocatedPorts {
			portMappings[name] = port
		}

		app := App{
			ID:           orchApp.Name, // Use name as ID for now
			Name:         orchApp.Name,
			ScenarioName: orchApp.Name,
			Status:       status,
			PortMappings: portMappings,
			Environment:  make(map[string]interface{}),
			Config:       make(map[string]interface{}),
			CreatedAt:    time.Now(), // We'll improve this later
			UpdatedAt:    time.Now(),
		}
		
		// Parse started_at if available
		if orchApp.StartedAt != "" && orchApp.StartedAt != "never" {
			if t, err := time.Parse(time.RFC3339, orchApp.StartedAt); err == nil {
				app.CreatedAt = t
				app.UpdatedAt = t
			}
		}

		apps = append(apps, app)
	}

	c.JSON(http.StatusOK, apps)
}

func (s *Server) getApp(c *gin.Context) {
	id := c.Param("id")
	
	query := `
		SELECT a.id, a.name, a.scenario_name, a.path, a.created_at, a.updated_at, a.status,
		       COALESCE(a.port_mappings, '{}') as port_mappings,
		       COALESCE(a.environment, '{}') as environment,
		       COALESCE(a.config, '{}') as config
		FROM apps a
		WHERE a.id = $1
	`

	var app App
	var portMappingsJSON, environmentJSON, configJSON string

	err := s.db.QueryRow(query, id).Scan(
		&app.ID, &app.Name, &app.ScenarioName, &app.Path,
		&app.CreatedAt, &app.UpdatedAt, &app.Status,
		&portMappingsJSON, &environmentJSON, &configJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch app"})
		}
		return
	}

	// Parse JSON fields
	json.Unmarshal([]byte(portMappingsJSON), &app.PortMappings)
	json.Unmarshal([]byte(environmentJSON), &app.Environment)
	json.Unmarshal([]byte(configJSON), &app.Config)

	c.JSON(http.StatusOK, app)
}

func (s *Server) startApp(c *gin.Context) {
	id := c.Param("id")
	
	// Get app details
	var app App
	var configJSON string

	query := `SELECT id, name, config FROM apps WHERE id = $1`
	err := s.db.QueryRow(query, id).Scan(&app.ID, &app.Name, &configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch app"})
		}
		return
	}

	json.Unmarshal([]byte(configJSON), &app.Config)

	// Get container name
	containerName := app.Name
	if containerConfig, ok := app.Config["container_name"].(string); ok {
		containerName = containerConfig
	}

	// Start container
	ctx := context.Background()
	err = s.docker.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
	if err != nil {
		log.Printf("Error starting container %s: %v", containerName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start app"})
		return
	}

	// Update app status
	_, err = s.db.Exec("UPDATE apps SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", "running", id)
	if err != nil {
		log.Printf("Error updating app status: %v", err)
	}

	// Publish event to Redis
	event := map[string]interface{}{
		"type":      "app_started",
		"app_id":    id,
		"app_name":  app.Name,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	eventJSON, _ := json.Marshal(event)
	s.redis.Publish(ctx, "app-events", eventJSON)

	c.JSON(http.StatusOK, gin.H{"message": "App started successfully"})
}

func (s *Server) stopApp(c *gin.Context) {
	id := c.Param("id")
	
	// Get app details
	var app App
	var configJSON string

	query := `SELECT id, name, config FROM apps WHERE id = $1`
	err := s.db.QueryRow(query, id).Scan(&app.ID, &app.Name, &configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch app"})
		}
		return
	}

	json.Unmarshal([]byte(configJSON), &app.Config)

	// Get container name
	containerName := app.Name
	if containerConfig, ok := app.Config["container_name"].(string); ok {
		containerName = containerConfig
	}

	// Stop container
	ctx := context.Background()
	timeout := 10
	err = s.docker.ContainerStop(ctx, containerName, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		log.Printf("Error stopping container %s: %v", containerName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop app"})
		return
	}

	// Update app status
	_, err = s.db.Exec("UPDATE apps SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", "stopped", id)
	if err != nil {
		log.Printf("Error updating app status: %v", err)
	}

	// Publish event to Redis
	event := map[string]interface{}{
		"type":      "app_stopped",
		"app_id":    id,
		"app_name":  app.Name,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	eventJSON, _ := json.Marshal(event)
	s.redis.Publish(ctx, "app-events", eventJSON)

	c.JSON(http.StatusOK, gin.H{"message": "App stopped successfully"})
}

func (s *Server) getAppLogs(c *gin.Context) {
	id := c.Param("id")
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	query := `
		SELECT id, app_id, level, message, source, timestamp
		FROM app_logs
		WHERE app_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs"})
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var logID, appID, level, message, source string
		var timestamp time.Time

		err := rows.Scan(&logID, &appID, &level, &message, &source, &timestamp)
		if err != nil {
			continue
		}

		logs = append(logs, map[string]interface{}{
			"id":        logID,
			"app_id":    appID,
			"level":     level,
			"message":   message,
			"source":    source,
			"timestamp": timestamp,
		})
	}

	c.JSON(http.StatusOK, logs)
}

func (s *Server) getAppLifecycleLogs(c *gin.Context) {
	id := c.Param("id")
	
	// Execute vrooli scenario logs command to get lifecycle logs
	cmd := exec.Command("vrooli", "scenario", "logs", id)
	cmd.Stderr = cmd.Stdout // Capture stderr as well
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log but don't fail - there might just be no logs yet
		log.Printf("Note: vrooli scenario logs returned error (may be normal): %v, output: %s", err, string(output))
		// Check if it's a "scenario not found" error
		if strings.Contains(string(output), "not found") || strings.Contains(string(output), "No such") {
			c.String(http.StatusOK, "Scenario not started or no lifecycle logs available yet")
			return
		}
	}
	
	// Return the raw logs as text
	if len(output) == 0 {
		c.String(http.StatusOK, "No lifecycle logs available")
	} else {
		c.String(http.StatusOK, string(output))
	}
}

func (s *Server) getAppBackgroundLogs(c *gin.Context) {
	id := c.Param("id")
	
	// Try different possible log locations
	logs := []map[string]interface{}{}
	logsFound := false
	
	// Try to get logs from the scenario's run directory
	// First check the standard Vrooli scenario log location
	vrooliLogDirs := []string{
		fmt.Sprintf("/home/%s/.vrooli/scenarios/%s/logs", os.Getenv("USER"), id),
		fmt.Sprintf("/tmp/vrooli/scenarios/%s/logs", id),
		fmt.Sprintf("/var/log/vrooli/scenarios/%s", id),
		fmt.Sprintf("./scenarios/%s/logs", id), // Relative path from current dir
		fmt.Sprintf("/home/%s/Vrooli/scenarios/%s/logs", os.Getenv("USER"), id),
	}
	
	for _, logDir := range vrooliLogDirs {
		if files, err := os.ReadDir(logDir); err == nil {
			log.Printf("Found log directory: %s", logDir)
			for _, file := range files {
				if !file.IsDir() && (strings.HasSuffix(file.Name(), ".log") || strings.Contains(file.Name(), "api") || strings.Contains(file.Name(), "ui")) {
					logPath := fmt.Sprintf("%s/%s", logDir, file.Name())
					if content, err := os.ReadFile(logPath); err == nil {
						// Read last 100 lines to avoid huge responses
						lines := strings.Split(string(content), "\n")
						startIdx := 0
						if len(lines) > 100 {
							startIdx = len(lines) - 100
						}
						for i := startIdx; i < len(lines); i++ {
							if lines[i] != "" {
								logsFound = true
								// Try to determine log level from content
								level := "log"
								lowerLine := strings.ToLower(lines[i])
								if strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "err") {
									level = "error"
								} else if strings.Contains(lowerLine, "warn") {
									level = "warning"
								} else if strings.Contains(lowerLine, "info") {
									level = "info"
								} else if strings.Contains(lowerLine, "debug") {
									level = "debug"
								}
								logs = append(logs, map[string]interface{}{
									"level":   level,
									"message": lines[i],
									"source":  file.Name(),
								})
							}
						}
					}
				}
			}
			if logsFound {
				break // Stop after finding the first valid log directory
			}
		}
	}
	
	// If still no logs, try to get process output using docker logs or systemd journal
	if !logsFound && s.docker != nil {
		ctx := context.Background()
		containerName := id // Scenarios often use their name as container name
		
		// Try to get Docker container logs
		if containerLogs, err := s.docker.ContainerLogs(ctx, containerName, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "100",
			Timestamps: true,
		}); err == nil {
			defer containerLogs.Close()
			
			buf := make([]byte, 4096)
			for {
				n, err := containerLogs.Read(buf)
				if n > 0 {
					lines := strings.Split(string(buf[:n]), "\n")
					for _, line := range lines {
						if line != "" {
							logsFound = true
							logs = append(logs, map[string]interface{}{
								"level":   "log",
								"message": line,
								"source":  "docker",
							})
						}
					}
				}
				if err != nil {
					break
				}
			}
		}
	}
	
	if !logsFound {
		logs = append(logs, map[string]interface{}{
			"level":   "info",
			"message": fmt.Sprintf("No background process logs found for %s. The scenario may not be running or may not produce background logs.", id),
			"source":  "system",
		})
	}
	
	c.JSON(http.StatusOK, logs)
}

func (s *Server) getScenarioLogs(c *gin.Context) {
	appName := c.Param("appName")
	logType := c.DefaultQuery("type", "both")
	
	// Response structure expected by frontend
	type LogResponse struct {
		Logs  []string `json:"logs"`
		Error string   `json:"error,omitempty"`
	}
	
	var logs []string
	
	// Helper function to execute vrooli command and get output
	executeVrooliLogs := func(args []string) (string, error) {
		cmd := exec.Command("vrooli", args...)
		cmd.Stderr = cmd.Stdout
		output, err := cmd.CombinedOutput()
		return string(output), err
	}
	
	// Get lifecycle logs if requested
	if logType == "lifecycle" || logType == "both" {
		output, err := executeVrooliLogs([]string{"scenario", "logs", appName})
		if err == nil && len(output) > 0 {
			// Add header for lifecycle logs
			logs = append(logs, "=== LIFECYCLE LOGS ===")
			logs = append(logs, strings.Split(strings.TrimSpace(output), "\n")...)
			logs = append(logs, "") // Empty line for separation
		} else if err != nil && !strings.Contains(output, "not found") && !strings.Contains(output, "No such") {
			// Log error but continue
			log.Printf("Error getting lifecycle logs for %s: %v", appName, err)
		}
	}
	
	// Get background logs if requested
	if logType == "background" || logType == "both" {
		// Try to get logs from various background steps
		steps := []string{"api", "ui", "server", "main", "worker", "background"}
		
		for _, step := range steps {
			output, err := executeVrooliLogs([]string{"scenario", "logs", appName, "--step", step})
			if err == nil && len(output) > 0 && !strings.Contains(output, "not found") && !strings.Contains(output, "No logs available") {
				// Add header for this step
				logs = append(logs, fmt.Sprintf("=== BACKGROUND LOGS (%s) ===", strings.ToUpper(step)))
				logs = append(logs, strings.Split(strings.TrimSpace(output), "\n")...)
				logs = append(logs, "") // Empty line for separation
			}
		}
		
		// If no background logs found but we're looking for them, try alternative locations
		if (logType == "background" && len(logs) == 0) || (logType == "both" && !strings.Contains(strings.Join(logs, "\n"), "BACKGROUND")) {
			// Try reading from the scenario's local log files
			scenarioDirs := []string{
				fmt.Sprintf("/home/%s/Vrooli/scenarios/%s", os.Getenv("USER"), appName),
				fmt.Sprintf("/home/%s/.vrooli/scenarios/%s", os.Getenv("USER"), appName),
				fmt.Sprintf("./scenarios/%s", appName),
			}
			
			for _, dir := range scenarioDirs {
				logDir := fmt.Sprintf("%s/logs", dir)
				if files, err := os.ReadDir(logDir); err == nil {
					for _, file := range files {
						if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
							logPath := fmt.Sprintf("%s/%s", logDir, file.Name())
							if content, err := os.ReadFile(logPath); err == nil {
								logs = append(logs, fmt.Sprintf("=== LOG FILE: %s ===", file.Name()))
								// Get last 50 lines to avoid huge responses
								lines := strings.Split(string(content), "\n")
								startIdx := 0
								if len(lines) > 50 {
									startIdx = len(lines) - 50
								}
								logs = append(logs, lines[startIdx:]...)
								logs = append(logs, "")
							}
						}
					}
					break // Found logs directory
				}
			}
		}
	}
	
	// If no logs found at all
	if len(logs) == 0 {
		logs = append(logs, fmt.Sprintf("No logs available for scenario '%s'. The scenario may not be running or hasn't produced logs yet.", appName))
	}
	
	// Return the response in the format expected by frontend
	response := LogResponse{
		Logs: logs,
	}
	
	c.JSON(http.StatusOK, response)
}

func (s *Server) getAppMetrics(c *gin.Context) {
	id := c.Param("id")
	hoursStr := c.DefaultQuery("hours", "24")

	hours, _ := strconv.Atoi(hoursStr)

	query := `
		SELECT id, app_id, status, cpu_usage, memory_usage, disk_usage, network_in, network_out, timestamp
		FROM app_status
		WHERE app_id = $1 AND timestamp > NOW() - INTERVAL '%d hours'
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(fmt.Sprintf(query, hours), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch metrics"})
		return
	}
	defer rows.Close()

	var metrics []AppStatus
	for rows.Next() {
		var status AppStatus
		err := rows.Scan(
			&status.ID, &status.AppID, &status.Status,
			&status.CPUUsage, &status.MemUsage, &status.DiskUsage,
			&status.NetworkIn, &status.NetworkOut, &status.Timestamp,
		)
		if err != nil {
			continue
		}
		metrics = append(metrics, status)
	}

	c.JSON(http.StatusOK, metrics)
}

func (s *Server) getDockerInfo(c *gin.Context) {
	ctx := context.Background()
	info, err := s.docker.Info(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Docker info"})
		return
	}

	c.JSON(http.StatusOK, info)
}

func (s *Server) getContainers(c *gin.Context) {
	ctx := context.Background()
	containers, err := s.docker.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list containers"})
		return
	}

	c.JSON(http.StatusOK, containers)
}

func (s *Server) getResources(c *gin.Context) {
	// Execute vrooli resource status --json to get real-time resource status
	cmd := exec.Command("vrooli", "resource", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to execute vrooli resource status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch resources"})
		return
	}

	// Parse the JSON response
	var vrooliResources []VrooliResource
	if err := json.Unmarshal(output, &vrooliResources); err != nil {
		log.Printf("Failed to parse resource response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse resource response"})
		return
	}

	// Convert to our Resource format
	resources := make([]Resource, 0, len(vrooliResources))
	for _, vr := range vrooliResources {
		// Determine status based on enabled/running state
		status := "offline"
		if vr.Running {
			status = "online"
		} else if vr.Enabled {
			status = "stopped"
		}

		// Map resource type from name
		resourceType := vr.Name
		switch vr.Name {
		case "postgres", "postgresql":
			resourceType = "postgres"
		case "redis":
			resourceType = "redis"
		case "n8n":
			resourceType = "n8n"
		case "ollama":
			resourceType = "ollama"
		case "qdrant":
			resourceType = "qdrant"
		case "minio":
			resourceType = "minio"
		case "windmill":
			resourceType = "windmill"
		case "node-red", "nodered":
			resourceType = "node-red"
		}

		resource := Resource{
			ID:      vr.Name,
			Name:    vr.Name,
			Type:    resourceType,
			Status:  status,
			Enabled: vr.Enabled,
			Running: vr.Running,
		}
		resources = append(resources, resource)
	}

	c.JSON(http.StatusOK, resources)
}

func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Subscribe to Redis events
	ctx := context.Background()
	pubsub := s.redis.Subscribe(ctx, "app-events")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
		if err != nil {
			log.Printf("Failed to write WebSocket message: %v", err)
			break
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
