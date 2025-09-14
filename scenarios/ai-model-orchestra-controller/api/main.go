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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "ai-model-orchestra-controller"
)

var (
	logger    *log.Logger
	db        *sql.DB
	rdb       *redis.Client
	dockerCli *client.Client
	mu        sync.RWMutex
)

// Models for AI orchestration
type ModelMetric struct {
	ID                  uuid.UUID              `json:"id"`
	ModelName           string                 `json:"model_name"`
	RequestCount        int                    `json:"request_count"`
	SuccessCount        int                    `json:"success_count"`
	ErrorCount          int                    `json:"error_count"`
	AvgResponseTimeMs   float64                `json:"avg_response_time_ms"`
	CurrentLoad         float64                `json:"current_load"`
	MemoryUsageMB       float64                `json:"memory_usage_mb"`
	LastUsed            *time.Time             `json:"last_used"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	Healthy             bool                   `json:"healthy"`
	Capabilities        []string               `json:"capabilities"`
	Speed               string                 `json:"speed"`
	QualityTier         string                 `json:"quality_tier"`
	CostPer1KTokens     float64                `json:"cost_per_1k_tokens"`
	RamRequiredGB       float64                `json:"ram_required_gb"`
}

type OrchestratorRequest struct {
	ID               uuid.UUID              `json:"id"`
	RequestID        string                 `json:"request_id"`
	TaskType         string                 `json:"task_type"`
	SelectedModel    string                 `json:"selected_model"`
	FallbackUsed     bool                   `json:"fallback_used"`
	ResponseTimeMs   int                    `json:"response_time_ms"`
	Success          bool                   `json:"success"`
	ErrorMessage     *string                `json:"error_message,omitempty"`
	ResourcePressure float64                `json:"resource_pressure"`
	CostEstimate     float64                `json:"cost_estimate"`
	CreatedAt        time.Time              `json:"created_at"`
}

type SystemResource struct {
	ID                  uuid.UUID `json:"id"`
	MemoryAvailableGB   float64   `json:"memory_available_gb"`
	MemoryFreeGB        float64   `json:"memory_free_gb"`
	MemoryTotalGB       float64   `json:"memory_total_gb"`
	CPUUsagePercent     float64   `json:"cpu_usage_percent"`
	SwapUsedPercent     float64   `json:"swap_used_percent"`
	RecordedAt          time.Time `json:"recorded_at"`
}

type ModelCapability struct {
	ModelName       string   `json:"model_name"`
	Capabilities    []string `json:"capabilities"`
	RamRequiredGB   float64  `json:"ram_required_gb"`
	Speed           string   `json:"speed"`
	CostPer1KTokens float64  `json:"cost_per_1k_tokens"`
	QualityTier     string   `json:"quality_tier"`
	BestFor         []string `json:"best_for"`
}

// Request/Response types
type ModelSelectRequest struct {
	TaskType     string                 `json:"taskType"`
	Requirements map[string]interface{} `json:"requirements,omitempty"`
}

type ModelSelectResponse struct {
	RequestID      string                 `json:"requestId"`
	SelectedModel  string                 `json:"selectedModel"`
	TaskType       string                 `json:"taskType"`
	FallbackUsed   bool                   `json:"fallbackUsed"`
	Alternatives   []string               `json:"alternatives"`
	SystemMetrics  map[string]interface{} `json:"systemMetrics"`
	ModelInfo      map[string]interface{} `json:"modelInfo"`
}

type RouteRequest struct {
	TaskType       string                 `json:"taskType"`
	Prompt         string                 `json:"prompt"`
	Requirements   map[string]interface{} `json:"requirements,omitempty"`
	RetryAttempts  *int                   `json:"retryAttempts,omitempty"`
}

type RouteResponse struct {
	RequestID      string                 `json:"requestId"`
	SelectedModel  string                 `json:"selectedModel"`
	Response       string                 `json:"response"`
	FallbackUsed   bool                   `json:"fallbackUsed"`
	Metrics        map[string]interface{} `json:"metrics"`
}

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]string      `json:"services"`
	System    map[string]interface{} `json:"system"`
	Version   string                 `json:"version"`
}

// Initialize database connection with exponential backoff
func initDatabase() error {
	host := getEnvOrDefault("ORCHESTRATOR_HOST", "localhost")
	port := getEnvOrDefault("RESOURCE_PORTS_POSTGRES", "5432")
	user := getEnvOrDefault("POSTGRES_USER", "postgres")
	password := getEnvOrDefault("POSTGRES_PASSWORD", "postgres")
	dbname := getEnvOrDefault("POSTGRES_DB", "orchestrator")
	
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	
	var err error
	maxRetries := 10
	backoffBase := time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sql.Open("postgres", psqlInfo)
		if err != nil {
			return fmt.Errorf("error opening database: %v", err)
		}
		
		// Test connection
		if err = db.Ping(); err == nil {
			logger.Printf("✅ Connected to PostgreSQL on attempt %d", attempt)
			
			// Set connection pool settings
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)
			
			return nil
		}
		
		db.Close()
		
		if attempt == maxRetries {
			return fmt.Errorf("failed to connect to database after %d attempts: %v", maxRetries, err)
		}
		
		// Exponential backoff with jitter
		backoff := time.Duration(math.Pow(2, float64(attempt-1))) * backoffBase
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		
		logger.Printf("⚠️  Database connection failed (attempt %d/%d), retrying in %v: %v", 
			attempt, maxRetries, backoff, err)
		time.Sleep(backoff)
	}
	
	return err
}

// Initialize Redis connection with exponential backoff
func initRedis() error {
	host := getEnvOrDefault("ORCHESTRATOR_HOST", "localhost")
	port := getEnvOrDefault("RESOURCE_PORTS_REDIS", "6379")
	
	rdb = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Password:     getEnvOrDefault("REDIS_PASSWORD", ""),
		DB:           0,
		MaxRetries:   3,
		RetryDelay:   time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})
	
	maxRetries := 10
	backoffBase := time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := rdb.Ping(ctx).Result()
		cancel()
		
		if err == nil {
			logger.Printf("✅ Connected to Redis on attempt %d", attempt)
			return nil
		}
		
		if attempt == maxRetries {
			return fmt.Errorf("failed to connect to Redis after %d attempts: %v", maxRetries, err)
		}
		
		// Exponential backoff
		backoff := time.Duration(math.Pow(2, float64(attempt-1))) * backoffBase
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		
		logger.Printf("⚠️  Redis connection failed (attempt %d/%d), retrying in %v: %v", 
			attempt, maxRetries, backoff, err)
		time.Sleep(backoff)
	}
	
	return nil
}

// Initialize Docker client
func initDocker() error {
	var err error
	dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Printf("⚠️  Docker client initialization failed: %v", err)
		return err
	}
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err = dockerCli.Info(ctx)
	if err != nil {
		logger.Printf("⚠️  Docker connection test failed: %v", err)
		return err
	}
	
	logger.Printf("✅ Connected to Docker daemon")
	return nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, map[string]interface{}{
		"error":     message,
		"timestamp": time.Now(),
		"service":   serviceName,
	})
}

// Get system resource metrics
func getSystemMetrics() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Convert bytes to GB
	totalMem := float64(m.Sys) / 1024 / 1024 / 1024
	freeMem := float64(m.Frees) / 1024 / 1024 / 1024
	usedMem := totalMem - freeMem
	
	memoryPressure := 0.0
	if totalMem > 0 {
		memoryPressure = usedMem / totalMem
	}
	
	return map[string]interface{}{
		"memoryPressure":     memoryPressure,
		"availableMemoryGb":  freeMem,
		"totalMemoryGb":      totalMem,
		"usedMemoryGb":       usedMem,
		"cpuUsage":           runtime.NumGoroutine(), // Simplified CPU metric
	}
}

// Model selection logic
func selectOptimalModel(taskType string, requirements map[string]interface{}) (*ModelSelectResponse, error) {
	requestID := uuid.New().String()
	
	// Get system metrics
	systemMetrics := getSystemMetrics()
	memoryPressure := systemMetrics["memoryPressure"].(float64)
	
	// Available models (in production, this would query Ollama or a model registry)
	availableModels := []ModelCapability{
		{
			ModelName:       "llama3.2:1b",
			Capabilities:    []string{"completion", "reasoning"},
			RamRequiredGB:   2.0,
			Speed:          "fast",
			CostPer1KTokens: 0.001,
			QualityTier:    "basic",
			BestFor:        []string{"simple-completion", "basic-reasoning"},
		},
		{
			ModelName:       "llama3.2:3b", 
			Capabilities:    []string{"completion", "reasoning", "code"},
			RamRequiredGB:   4.0,
			Speed:          "medium",
			CostPer1KTokens: 0.003,
			QualityTier:    "good",
			BestFor:        []string{"completion", "reasoning", "simple-code"},
		},
		{
			ModelName:       "llama3.2:8b",
			Capabilities:    []string{"completion", "reasoning", "code", "analysis"},
			RamRequiredGB:   8.0,
			Speed:          "slow",
			CostPer1KTokens: 0.008,
			QualityTier:    "high",
			BestFor:        []string{"complex-reasoning", "code-generation", "analysis"},
		},
		{
			ModelName:       "codellama:7b",
			Capabilities:    []string{"code", "completion"},
			RamRequiredGB:   7.0,
			Speed:          "medium",
			CostPer1KTokens: 0.007,
			QualityTier:    "high",
			BestFor:        []string{"code-generation", "code-analysis"},
		},
	}
	
	// Filter models by capabilities
	suitableModels := []ModelCapability{}
	for _, model := range availableModels {
		for _, capability := range model.Capabilities {
			if capability == taskType {
				suitableModels = append(suitableModels, model)
				break
			}
		}
	}
	
	if len(suitableModels) == 0 {
		return nil, fmt.Errorf("no models available for task type: %s", taskType)
	}
	
	// Apply selection logic based on requirements and system state
	selectedModel := suitableModels[0] // Default to first suitable
	fallbackUsed := false
	
	// Consider memory pressure
	if memoryPressure > 0.8 {
		// High memory pressure - prefer smaller models
		for _, model := range suitableModels {
			if model.RamRequiredGB < selectedModel.RamRequiredGB {
				selectedModel = model
				fallbackUsed = true
			}
		}
	} else {
		// Normal memory - prefer quality unless cost limit specified
		if costLimit, ok := requirements["costLimit"].(float64); ok {
			for _, model := range suitableModels {
				if model.CostPer1KTokens <= costLimit && model.QualityTier == "high" {
					selectedModel = model
					break
				}
			}
		} else {
			// No cost limit - prefer highest quality
			for _, model := range suitableModels {
				if model.QualityTier == "high" {
					selectedModel = model
					break
				}
			}
		}
	}
	
	// Build alternatives list
	alternatives := []string{}
	for _, model := range suitableModels {
		if model.ModelName != selectedModel.ModelName {
			alternatives = append(alternatives, model.ModelName)
		}
	}
	
	// Model info
	modelInfo := map[string]interface{}{
		"capabilities":  selectedModel.Capabilities,
		"speed":        selectedModel.Speed,
		"quality_tier": selectedModel.QualityTier,
		"ram_required": selectedModel.RamRequiredGB,
		"cost_per_1k":  selectedModel.CostPer1KTokens,
	}
	
	return &ModelSelectResponse{
		RequestID:     requestID,
		SelectedModel: selectedModel.ModelName,
		TaskType:      taskType,
		FallbackUsed:  fallbackUsed,
		Alternatives:  alternatives,
		SystemMetrics: systemMetrics,
		ModelInfo:     modelInfo,
	}, nil
}

// API Handlers
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	services := make(map[string]string)
	
	// Check database
	if db != nil {
		if err := db.Ping(); err != nil {
			services["database"] = "error"
			status = "degraded"
		} else {
			services["database"] = "ok"
		}
	} else {
		services["database"] = "not_connected"
		status = "unhealthy"
	}
	
	// Check Redis
	if rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := rdb.Ping(ctx).Result()
		cancel()
		
		if err != nil {
			services["redis"] = "error"
			if status != "unhealthy" {
				status = "degraded"
			}
		} else {
			services["redis"] = "ok"
		}
	} else {
		services["redis"] = "not_connected"
		status = "unhealthy"
	}
	
	// Check Ollama (simplified - would normally make HTTP request)
	services["ollama"] = "ok"
	
	systemMetrics := getSystemMetrics()
	systemMetrics["available_models"] = 4 // Simplified
	
	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Services:  services,
		System:    systemMetrics,
		Version:   apiVersion,
	}
	
	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if status == "degraded" {
		statusCode = http.StatusPartialContent
	}
	
	writeJSONResponse(w, statusCode, response)
}

func handleModelSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	var req ModelSelectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	
	if req.TaskType == "" {
		writeErrorResponse(w, http.StatusBadRequest, "taskType is required")
		return
	}
	
	response, err := selectOptimalModel(req.TaskType, req.Requirements)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

func handleRouteRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	var req RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	
	if req.TaskType == "" || req.Prompt == "" {
		writeErrorResponse(w, http.StatusBadRequest, "taskType and prompt are required")
		return
	}
	
	startTime := time.Now()
	
	// First select optimal model
	modelResponse, err := selectOptimalModel(req.TaskType, req.Requirements)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	// Simulate AI inference (in production, would call Ollama API)
	time.Sleep(100 * time.Millisecond) // Simulate processing time
	responseText := fmt.Sprintf("Simulated response from %s for task '%s': %s", 
		modelResponse.SelectedModel, req.TaskType, req.Prompt)
	
	responseTimeMs := int(time.Since(startTime).Milliseconds())
	
	metrics := map[string]interface{}{
		"responseTimeMs":  responseTimeMs,
		"memoryPressure":  modelResponse.SystemMetrics["memoryPressure"],
		"modelUsed":       modelResponse.SelectedModel,
		"tokensGenerated": len(strings.Split(responseText, " ")),
		"promptTokens":    len(strings.Split(req.Prompt, " ")),
	}
	
	response := RouteResponse{
		RequestID:     modelResponse.RequestID,
		SelectedModel: modelResponse.SelectedModel,
		Response:      responseText,
		FallbackUsed:  modelResponse.FallbackUsed,
		Metrics:       metrics,
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

func handleModelsStatus(w http.ResponseWriter, r *http.Request) {
	// Simulate model status data
	models := []ModelMetric{
		{
			ID:                uuid.New(),
			ModelName:        "llama3.2:1b",
			RequestCount:     150,
			SuccessCount:     147,
			ErrorCount:       3,
			AvgResponseTimeMs: 250.5,
			CurrentLoad:      0.3,
			MemoryUsageMB:    2048,
			Healthy:          true,
			Capabilities:     []string{"completion", "reasoning"},
			Speed:           "fast",
			QualityTier:     "basic",
			CostPer1KTokens: 0.001,
			RamRequiredGB:   2.0,
		},
		{
			ID:                uuid.New(),
			ModelName:        "llama3.2:8b",
			RequestCount:     89,
			SuccessCount:     88,
			ErrorCount:       1,
			AvgResponseTimeMs: 850.2,
			CurrentLoad:      0.7,
			MemoryUsageMB:    8192,
			Healthy:          true,
			Capabilities:     []string{"completion", "reasoning", "code", "analysis"},
			Speed:           "slow",
			QualityTier:     "high",
			CostPer1KTokens: 0.008,
			RamRequiredGB:   8.0,
		},
	}
	
	response := map[string]interface{}{
		"models":       models,
		"totalModels":  len(models),
		"healthyModels": len(models),
		"systemHealth": getSystemMetrics(),
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

func handleResourceMetrics(w http.ResponseWriter, r *http.Request) {
	hoursParam := r.URL.Query().Get("hours")
	hours := 1 // Default to 1 hour
	
	if hoursParam != "" {
		if h, err := strconv.Atoi(hoursParam); err == nil {
			hours = h
		}
	}
	
	current := getSystemMetrics()
	
	// Simulate historical data
	history := []map[string]interface{}{}
	for i := 0; i < hours; i++ {
		historyPoint := map[string]interface{}{
			"timestamp":      time.Now().Add(-time.Duration(i) * time.Hour),
			"memoryPressure": 0.4 + (float64(i%10)/25.0), // Simulate varying pressure
			"cpuUsage":       20.0 + float64(i%30),
			"availableMemoryGb": 8.0 - (float64(i%5) * 0.5),
		}
		history = append(history, historyPoint)
	}
	
	response := map[string]interface{}{
		"current":        current,
		"history":        history,
		"memoryPressure": current["memoryPressure"],
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Simple dashboard redirect
	http.Redirect(w, r, "/ui/dashboard.html", http.StatusFound)
}

// Initialize database schema
func initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS model_metrics (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		model_name VARCHAR(255) NOT NULL UNIQUE,
		request_count INTEGER DEFAULT 0,
		success_count INTEGER DEFAULT 0,
		error_count INTEGER DEFAULT 0,
		avg_response_time_ms FLOAT DEFAULT 0,
		current_load FLOAT DEFAULT 0,
		memory_usage_mb FLOAT DEFAULT 0,
		last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		healthy BOOLEAN DEFAULT TRUE
	);

	CREATE TABLE IF NOT EXISTS orchestrator_requests (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		request_id VARCHAR(255) UNIQUE NOT NULL,
		task_type VARCHAR(100) NOT NULL,
		selected_model VARCHAR(255) NOT NULL,
		fallback_used BOOLEAN DEFAULT FALSE,
		response_time_ms INTEGER,
		success BOOLEAN DEFAULT TRUE,
		error_message TEXT,
		resource_pressure FLOAT,
		cost_estimate FLOAT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS system_resources (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		memory_available_gb FLOAT,
		memory_free_gb FLOAT,
		memory_total_gb FLOAT,
		cpu_usage_percent FLOAT,
		swap_used_percent FLOAT,
		recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_model_metrics_name ON model_metrics(model_name);
	CREATE INDEX IF NOT EXISTS idx_orchestrator_requests_model ON orchestrator_requests(selected_model);
	CREATE INDEX IF NOT EXISTS idx_system_resources_recorded ON system_resources(recorded_at);
	`
	
	_, err := db.Exec(schema)
	return err
}

func main() {
	logger = log.New(os.Stdout, "[ai-orchestrator] ", log.LstdFlags)
	
	logger.Printf("🚀 Starting AI Model Orchestra Controller v%s", apiVersion)
	
	// Initialize connections
	if err := initDatabase(); err != nil {
		logger.Printf("❌ Database initialization failed: %v", err)
		// Continue without database for development
	} else {
		if err := initSchema(); err != nil {
			logger.Printf("⚠️  Schema initialization failed: %v", err)
		}
	}
	
	if err := initRedis(); err != nil {
		logger.Printf("❌ Redis initialization failed: %v", err)
		// Continue without Redis for development
	}
	
	if err := initDocker(); err != nil {
		logger.Printf("❌ Docker initialization failed: %v", err)
		// Continue without Docker for development
	}
	
	// Setup routes
	r := mux.NewRouter()
	
	// Health check
	r.HandleFunc("/health", handleHealthCheck).Methods("GET")
	r.HandleFunc("/api/health", handleHealthCheck).Methods("GET")
	
	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/ai/select-model", handleModelSelect).Methods("POST")
	api.HandleFunc("/ai/route-request", handleRouteRequest).Methods("POST")
	api.HandleFunc("/ai/models/status", handleModelsStatus).Methods("GET")
	api.HandleFunc("/ai/resources/metrics", handleResourceMetrics).Methods("GET")
	api.HandleFunc("/health", handleHealthCheck).Methods("GET")
	
	// Dashboard
	r.HandleFunc("/dashboard", handleDashboard).Methods("GET")
	
	// Serve static files
	r.PathPrefix("/ui/").Handler(http.StripPrefix("/ui/", http.FileServer(http.Dir("../ui/"))))
	
	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
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
	})
	
	port := getEnvOrDefault("API_PORT", "8080")
	
	logger.Printf("🎛️  API server starting on port %s", port)
	logger.Printf("📊 Dashboard available at: http://localhost:%s/dashboard", port)
	logger.Printf("🔗 Health check: http://localhost:%s/health", port)
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatalf("❌ Server failed to start: %v", err)
	}
}