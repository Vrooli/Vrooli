package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	port := getEnv("PORT", "8090")
	postgresURL := getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/app_monitor?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	n8nBaseURL := getEnv("N8N_BASE_URL", "http://localhost:5678")
	nodeRedURL := getEnv("NODE_RED_BASE_URL", "http://localhost:1880")

	// Initialize database connection
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize Redis connection
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	rdb := redis.NewClient(redisOpts)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
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

	// App management endpoints
	r.GET("/api/apps", s.getApps)
	r.GET("/api/apps/:id", s.getApp)
	r.POST("/api/apps/:id/start", s.startApp)
	r.POST("/api/apps/:id/stop", s.stopApp)
	r.GET("/api/apps/:id/logs", s.getAppLogs)
	r.GET("/api/apps/:id/metrics", s.getAppMetrics)

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
	if err := s.db.Ping(); err != nil {
		services["database"] = "error"
	} else {
		services["database"] = "ok"
	}

	// Check Redis
	if err := s.redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "error"
	} else {
		services["redis"] = "ok"
	}

	// Check Docker
	if _, err := s.docker.Ping(ctx); err != nil {
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

func (s *Server) getApps(c *gin.Context) {
	query := `
		SELECT a.id, a.name, a.scenario_name, a.path, a.created_at, a.updated_at, a.status,
		       COALESCE(a.port_mappings, '{}') as port_mappings,
		       COALESCE(a.environment, '{}') as environment,
		       COALESCE(a.config, '{}') as config
		FROM apps a
		ORDER BY a.name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch apps"})
		return
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		var portMappingsJSON, environmentJSON, configJSON string

		err := rows.Scan(
			&app.ID, &app.Name, &app.ScenarioName, &app.Path,
			&app.CreatedAt, &app.UpdatedAt, &app.Status,
			&portMappingsJSON, &environmentJSON, &configJSON,
		)
		if err != nil {
			log.Printf("Error scanning app row: %v", err)
			continue
		}

		// Parse JSON fields
		json.Unmarshal([]byte(portMappingsJSON), &app.PortMappings)
		json.Unmarshal([]byte(environmentJSON), &app.Environment)
		json.Unmarshal([]byte(configJSON), &app.Config)

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
