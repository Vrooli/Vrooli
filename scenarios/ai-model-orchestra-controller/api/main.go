package main

import (
	"bytes"
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
	ollamaClient *OllamaClient
	mu        sync.RWMutex
)

// Circuit breaker states
type CircuitState int

const (
	Closed CircuitState = iota
	Open
	HalfOpen
)

// Circuit breaker for Ollama
type CircuitBreaker struct {
	maxFailures   int
	resetTimeout  time.Duration
	state         CircuitState
	failures      int
	lastFailTime  time.Time
	mutex         sync.RWMutex
}

// Ollama API structures  
type OllamaClient struct {
	BaseURL        string
	Client         *http.Client
	CircuitBreaker *CircuitBreaker
}

type OllamaGenerateRequest struct {
	Model     string  `json:"model"`
	Prompt    string  `json:"prompt"`
	Stream    bool    `json:"stream"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

type OllamaGenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

type OllamaModel struct {
	Name       string            `json:"name"`
	Size       int64             `json:"size"`
	Digest     string            `json:"digest"`
	ModifiedAt time.Time         `json:"modified_at"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

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
	port := os.Getenv("RESOURCE_PORTS_POSTGRES")
	if port == "" {
		return fmt.Errorf("RESOURCE_PORTS_POSTGRES environment variable is required")
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		return fmt.Errorf("POSTGRES_USER environment variable is required")
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD environment variable is required")
	}
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
	port := os.Getenv("RESOURCE_PORTS_REDIS")
	if port == "" {
		return fmt.Errorf("RESOURCE_PORTS_REDIS environment variable is required")
	}
	
	rdb = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Password:     os.Getenv("REDIS_PASSWORD"), // No fallback - empty string if not set
		DB:           0,
		MaxRetries:   3,
		MinRetryBackoff: time.Second,
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

// Initialize Ollama client
func initOllama() error {
	host := getEnvOrDefault("ORCHESTRATOR_HOST", "localhost")
	port := os.Getenv("RESOURCE_PORTS_OLLAMA")
	if port == "" {
		return fmt.Errorf("RESOURCE_PORTS_OLLAMA environment variable is required")
	}
	
	baseURL := fmt.Sprintf("http://%s:%s", host, port)
	
	ollamaClient = &OllamaClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		CircuitBreaker: &CircuitBreaker{
			maxFailures:  5,                // Open circuit after 5 consecutive failures
			resetTimeout: 60 * time.Second, // Try again after 60 seconds
			state:        Closed,
		},
	}
	
	// Test connection by fetching models
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := ollamaClient.GetModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s: %v", baseURL, err)
	}
	
	logger.Printf("✅ Connected to Ollama at %s", baseURL)
	return nil
}

// Ollama client methods
func (c *OllamaClient) GetModels(ctx context.Context) (*OllamaModelsResponse, error) {
	// Check circuit breaker
	if err := c.CircuitBreaker.CanRequest(); err != nil {
		return nil, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		c.CircuitBreaker.RecordFailure()
		return nil, err
	}
	
	resp, err := c.Client.Do(req)
	if err != nil {
		c.CircuitBreaker.RecordFailure()
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		c.CircuitBreaker.RecordFailure()
		return nil, fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}
	
	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		c.CircuitBreaker.RecordFailure()
		return nil, err
	}
	
	c.CircuitBreaker.RecordSuccess()
	return &modelsResp, nil
}

func (c *OllamaClient) Generate(ctx context.Context, req *OllamaGenerateRequest) (*OllamaGenerateResponse, error) {
	// Check circuit breaker
	if err := c.CircuitBreaker.CanRequest(); err != nil {
		return nil, fmt.Errorf("circuit breaker open: %v", err)
	}
	
	reqBody, err := json.Marshal(req)
	if err != nil {
		c.CircuitBreaker.RecordFailure()
		return nil, err
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		c.CircuitBreaker.RecordFailure()
		return nil, err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.Client.Do(httpReq)
	if err != nil {
		c.CircuitBreaker.RecordFailure()
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		c.CircuitBreaker.RecordFailure()
		return nil, fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}
	
	var genResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		c.CircuitBreaker.RecordFailure()
		return nil, err
	}
	
	c.CircuitBreaker.RecordSuccess()
	return &genResp, nil
}

func (c *OllamaClient) IsHealthy(ctx context.Context) bool {
	_, err := c.GetModels(ctx)
	return err == nil
}

// Circuit breaker methods
func (cb *CircuitBreaker) CanRequest() error {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	switch cb.state {
	case Open:
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			// Transition to half-open
			cb.mutex.RUnlock()
			cb.mutex.Lock()
			cb.state = HalfOpen
			cb.mutex.Unlock()
			cb.mutex.RLock()
			return nil
		}
		return fmt.Errorf("circuit breaker is open")
	case HalfOpen, Closed:
		return nil
	default:
		return nil
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.failures = 0
	cb.state = Closed
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.failures++
	cb.lastFailTime = time.Now()
	
	if cb.failures >= cb.maxFailures {
		cb.state = Open
	}
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Database operations
func storeOrchestratorRequest(requestID, taskType, selectedModel string, fallbackUsed bool, responseTimeMs int, success bool, errorMessage *string, resourcePressure, costEstimate float64) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	query := `
		INSERT INTO orchestrator_requests 
		(request_id, task_type, selected_model, fallback_used, response_time_ms, success, error_message, resource_pressure, cost_estimate)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := db.Exec(query, requestID, taskType, selectedModel, fallbackUsed, responseTimeMs, success, errorMessage, resourcePressure, costEstimate)
	if err != nil {
		logger.Printf("⚠️  Failed to store orchestrator request: %v", err)
		return err
	}
	
	return nil
}

func storeSystemResources(memoryAvailableGB, memoryFreeGB, memoryTotalGB, cpuUsagePercent, swapUsedPercent float64) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	query := `
		INSERT INTO system_resources 
		(memory_available_gb, memory_free_gb, memory_total_gb, cpu_usage_percent, swap_used_percent)
		VALUES ($1, $2, $3, $4, $5)`
	
	_, err := db.Exec(query, memoryAvailableGB, memoryFreeGB, memoryTotalGB, cpuUsagePercent, swapUsedPercent)
	if err != nil {
		logger.Printf("⚠️  Failed to store system resources: %v", err)
		return err
	}
	
	return nil
}

func updateModelMetrics(modelName string, requestCount, successCount, errorCount int, avgResponseTime, currentLoad, memoryUsage float64, healthy bool) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	query := `
		INSERT INTO model_metrics 
		(model_name, request_count, success_count, error_count, avg_response_time_ms, current_load, memory_usage_mb, healthy, last_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)
		ON CONFLICT (model_name) DO UPDATE SET
			request_count = model_metrics.request_count + $2,
			success_count = model_metrics.success_count + $3,
			error_count = model_metrics.error_count + $4,
			avg_response_time_ms = (model_metrics.avg_response_time_ms + $5) / 2,
			current_load = $6,
			memory_usage_mb = $7,
			healthy = $8,
			last_used = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP`
	
	_, err := db.Exec(query, modelName, requestCount, successCount, errorCount, avgResponseTime, currentLoad, memoryUsage, healthy)
	if err != nil {
		logger.Printf("⚠️  Failed to update model metrics: %v", err)
		return err
	}
	
	return nil
}

// Background monitoring
func startSystemMonitoring() {
	ticker := time.NewTicker(30 * time.Second) // Store metrics every 30 seconds
	defer ticker.Stop()
	
	logger.Printf("🔄 Started background system monitoring")
	
	for {
		select {
		case <-ticker.C:
			if db != nil {
				metrics := getSystemMetrics()
				
				if err := storeSystemResources(
					metrics["availableMemoryGb"].(float64),
					metrics["availableMemoryGb"].(float64), // Using available as free for simplicity
					metrics["totalMemoryGb"].(float64),
					metrics["cpuUsage"].(float64),
					0.0, // swap usage - would need to implement proper swap monitoring
				); err != nil {
					logger.Printf("⚠️  Failed to store system resources: %v", err)
				}
			}
		}
	}
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

// Get default model capabilities as fallback
func getDefaultModelCapabilities() []ModelCapability {
	return []ModelCapability{
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
}

// Convert Ollama models to model capabilities
func convertOllamaModelsToCapabilities(ollamaModels []OllamaModel) []ModelCapability {
	capabilities := make([]ModelCapability, 0, len(ollamaModels))
	
	for _, model := range ollamaModels {
		// Determine capabilities based on model name patterns
		var modelCaps []string
		var speed string
		var qualityTier string
		var costPer1K float64
		var bestFor []string
		
		modelName := strings.ToLower(model.Name)
		sizeGB := float64(model.Size) / (1024 * 1024 * 1024)
		
		// Classify model capabilities based on name patterns
		if strings.Contains(modelName, "code") || strings.Contains(modelName, "llama") {
			modelCaps = []string{"completion", "reasoning", "code"}
			bestFor = []string{"code-generation", "completion", "reasoning"}
		} else if strings.Contains(modelName, "embed") {
			modelCaps = []string{"embedding"}
			bestFor = []string{"text-embedding", "similarity"}
		} else {
			modelCaps = []string{"completion", "reasoning"}
			bestFor = []string{"completion", "reasoning"}
		}
		
		// Determine quality and speed based on size
		if sizeGB < 3 {
			speed = "fast"
			qualityTier = "basic"
			costPer1K = 0.001
		} else if sizeGB < 6 {
			speed = "medium"
			qualityTier = "good"
			costPer1K = 0.003
		} else {
			speed = "slow"
			qualityTier = "high"
			costPer1K = 0.008
		}
		
		// Add analysis capability for larger models
		if sizeGB > 6 {
			modelCaps = append(modelCaps, "analysis")
			bestFor = append(bestFor, "complex-reasoning", "analysis")
		}
		
		capability := ModelCapability{
			ModelName:       model.Name,
			Capabilities:    modelCaps,
			RamRequiredGB:   sizeGB,
			Speed:          speed,
			CostPer1KTokens: costPer1K,
			QualityTier:    qualityTier,
			BestFor:        bestFor,
		}
		
		capabilities = append(capabilities, capability)
	}
	
	return capabilities
}

// Model selection logic
func selectOptimalModel(taskType string, requirements map[string]interface{}) (*ModelSelectResponse, error) {
	requestID := uuid.New().String()
	
	// Get system metrics
	systemMetrics := getSystemMetrics()
	memoryPressure := systemMetrics["memoryPressure"].(float64)
	
	// Get available models from Ollama
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var availableModels []ModelCapability
	
	if ollamaClient != nil {
		ollamaModels, err := ollamaClient.GetModels(ctx)
		if err != nil {
			logger.Printf("⚠️  Failed to fetch models from Ollama: %v", err)
			// Fallback to basic model list
			availableModels = getDefaultModelCapabilities()
		} else {
			availableModels = convertOllamaModelsToCapabilities(ollamaModels.Models)
		}
	} else {
		logger.Printf("⚠️  Ollama client not initialized, using default models")
		availableModels = getDefaultModelCapabilities()
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
	
	// Check Ollama health
	if ollamaClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if ollamaClient.IsHealthy(ctx) {
			services["ollama"] = "ok"
		} else {
			services["ollama"] = "error"
			if status != "unhealthy" {
				status = "degraded"
			}
		}
	} else {
		services["ollama"] = "not_connected"
		status = "unhealthy"
	}
	
	systemMetrics := getSystemMetrics()
	
	// Get actual model count from Ollama
	modelCount := 0
	if ollamaClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if models, err := ollamaClient.GetModels(ctx); err == nil {
			modelCount = len(models.Models)
		}
	}
	systemMetrics["available_models"] = modelCount
	
	// Add circuit breaker status
	if ollamaClient != nil && ollamaClient.CircuitBreaker != nil {
		cbState := ollamaClient.CircuitBreaker.GetState()
		var stateStr string
		switch cbState {
		case Closed:
			stateStr = "closed"
		case Open:
			stateStr = "open"
		case HalfOpen:
			stateStr = "half-open"
		}
		systemMetrics["circuit_breaker_state"] = stateStr
		systemMetrics["circuit_breaker_failures"] = ollamaClient.CircuitBreaker.failures
	}
	
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
	
	// Generate actual AI response using Ollama
	var responseText string
	var tokensGenerated int
	var promptTokens int
	
	if ollamaClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		// Prepare generation options
		options := make(map[string]interface{})
		if maxTokens, ok := req.Requirements["maxTokens"].(float64); ok {
			options["num_predict"] = int(maxTokens)
		}
		if temperature, ok := req.Requirements["temperature"].(float64); ok {
			options["temperature"] = temperature
		}
		
		ollamaReq := &OllamaGenerateRequest{
			Model:   modelResponse.SelectedModel,
			Prompt:  req.Prompt,
			Stream:  false,
			Options: options,
		}
		
		ollamaResp, err := ollamaClient.Generate(ctx, ollamaReq)
		if err != nil {
			logger.Printf("❌ Ollama generation failed: %v", err)
			
			// Store failed request in database
			responseTimeMs := int(time.Since(startTime).Milliseconds())
			errorMsg := err.Error()
			resourcePressure := modelResponse.SystemMetrics["memoryPressure"].(float64)
			
			if dbErr := storeOrchestratorRequest(
				modelResponse.RequestID,
				req.TaskType,
				modelResponse.SelectedModel,
				modelResponse.FallbackUsed,
				responseTimeMs,
				false, // success = false
				&errorMsg,
				resourcePressure,
				0.0, // no cost for failed request
			); dbErr != nil {
				logger.Printf("⚠️  Failed to store failed request log: %v", dbErr)
			}
			
			// Update model metrics with error count
			if dbErr := updateModelMetrics(
				modelResponse.SelectedModel,
				1, // request count increment
				0, // success count increment  
				1, // error count increment
				float64(responseTimeMs),
				0.0, // current load
				0.0, // memory usage
				false, // healthy = false due to error
			); dbErr != nil {
				logger.Printf("⚠️  Failed to update model metrics: %v", dbErr)
			}
			
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("AI generation failed: %v", err))
			return
		}
		
		responseText = ollamaResp.Response
		tokensGenerated = ollamaResp.EvalCount
		promptTokens = ollamaResp.PromptEvalCount
	} else {
		// Fallback to simulation if Ollama is not available
		logger.Printf("⚠️  Ollama client not available, falling back to simulation")
		responseText = fmt.Sprintf("Simulated response from %s for task '%s': %s", 
			modelResponse.SelectedModel, req.TaskType, req.Prompt)
		tokensGenerated = len(strings.Split(responseText, " "))
		promptTokens = len(strings.Split(req.Prompt, " "))
	}
	
	responseTimeMs := int(time.Since(startTime).Milliseconds())
	
	// Store request in database
	success := true
	var errorMessage *string
	resourcePressure := modelResponse.SystemMetrics["memoryPressure"].(float64)
	costEstimate := 0.0 // TODO: Calculate actual cost based on tokens and model pricing
	
	// Calculate cost estimate based on tokens and model pricing
	if tokensGenerated > 0 {
		// Find the model capability to get cost per 1k tokens
		availableModels, _ := ollamaClient.GetModels(context.Background())
		if availableModels != nil {
			capabilities := convertOllamaModelsToCapabilities(availableModels.Models)
			for _, cap := range capabilities {
				if cap.ModelName == modelResponse.SelectedModel {
					costEstimate = (float64(tokensGenerated) / 1000.0) * cap.CostPer1KTokens
					break
				}
			}
		}
	}
	
	if err := storeOrchestratorRequest(
		modelResponse.RequestID,
		req.TaskType,
		modelResponse.SelectedModel,
		modelResponse.FallbackUsed,
		responseTimeMs,
		success,
		errorMessage,
		resourcePressure,
		costEstimate,
	); err != nil {
		logger.Printf("⚠️  Failed to store request log: %v", err)
	}
	
	// Update model metrics
	if err := updateModelMetrics(
		modelResponse.SelectedModel,
		1,    // request count increment
		1,    // success count increment  
		0,    // error count increment
		float64(responseTimeMs),
		0.0,  // current load (would need to calculate)
		0.0,  // memory usage (would need to measure)
		true, // healthy
	); err != nil {
		logger.Printf("⚠️  Failed to update model metrics: %v", err)
	}
	
	metrics := map[string]interface{}{
		"responseTimeMs":  responseTimeMs,
		"memoryPressure":  modelResponse.SystemMetrics["memoryPressure"],
		"modelUsed":       modelResponse.SelectedModel,
		"tokensGenerated": tokensGenerated,
		"promptTokens":    promptTokens,
		"costEstimate":    costEstimate,
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
	var models []ModelMetric
	healthyCount := 0
	
	// Get real model data from database
	if db != nil {
		query := `
			SELECT model_name, request_count, success_count, error_count, 
				   avg_response_time_ms, current_load, memory_usage_mb, 
				   healthy, last_used, created_at, updated_at
			FROM model_metrics 
			ORDER BY last_used DESC`
		
		rows, err := db.Query(query)
		if err != nil {
			logger.Printf("⚠️  Failed to query model metrics: %v", err)
		} else {
			defer rows.Close()
			
			for rows.Next() {
				var model ModelMetric
				var lastUsed sql.NullTime
				
				err := rows.Scan(
					&model.ModelName,
					&model.RequestCount,
					&model.SuccessCount,
					&model.ErrorCount,
					&model.AvgResponseTimeMs,
					&model.CurrentLoad,
					&model.MemoryUsageMB,
					&model.Healthy,
					&lastUsed,
					&model.CreatedAt,
					&model.UpdatedAt,
				)
				if err != nil {
					logger.Printf("⚠️  Failed to scan model metric: %v", err)
					continue
				}
				
				model.ID = uuid.New()
				if lastUsed.Valid {
					model.LastUsed = &lastUsed.Time
				}
				
				// Get model capabilities from Ollama or defaults
				if ollamaClient != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					
					if ollamaModels, err := ollamaClient.GetModels(ctx); err == nil {
						capabilities := convertOllamaModelsToCapabilities(ollamaModels.Models)
						for _, cap := range capabilities {
							if cap.ModelName == model.ModelName {
								model.Capabilities = cap.Capabilities
								model.Speed = cap.Speed
								model.QualityTier = cap.QualityTier
								model.CostPer1KTokens = cap.CostPer1KTokens
								model.RamRequiredGB = cap.RamRequiredGB
								break
							}
						}
					}
				}
				
				// Set defaults if not found
				if len(model.Capabilities) == 0 {
					model.Capabilities = []string{"completion", "reasoning"}
					model.Speed = "medium"
					model.QualityTier = "good"
					model.CostPer1KTokens = 0.005
					model.RamRequiredGB = 4.0
				}
				
				models = append(models, model)
				if model.Healthy {
					healthyCount++
				}
			}
		}
	}
	
	// If no models in database, get them from Ollama
	if len(models) == 0 && ollamaClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if ollamaModels, err := ollamaClient.GetModels(ctx); err == nil {
			capabilities := convertOllamaModelsToCapabilities(ollamaModels.Models)
			for _, cap := range capabilities {
				model := ModelMetric{
					ID:                uuid.New(),
					ModelName:        cap.ModelName,
					RequestCount:     0,
					SuccessCount:     0,
					ErrorCount:       0,
					AvgResponseTimeMs: 0,
					CurrentLoad:      0,
					MemoryUsageMB:    cap.RamRequiredGB * 1024,
					Healthy:          true,
					Capabilities:     cap.Capabilities,
					Speed:           cap.Speed,
					QualityTier:     cap.QualityTier,
					CostPer1KTokens: cap.CostPer1KTokens,
					RamRequiredGB:   cap.RamRequiredGB,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				models = append(models, model)
				healthyCount++
			}
		}
	}
	
	response := map[string]interface{}{
		"models":       models,
		"totalModels":  len(models),
		"healthyModels": healthyCount,
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
	
	// Get historical data from database
	var history []map[string]interface{}
	
	if db != nil {
		query := `
			SELECT memory_available_gb, memory_free_gb, memory_total_gb, 
				   cpu_usage_percent, swap_used_percent, recorded_at
			FROM system_resources 
			WHERE recorded_at >= NOW() - INTERVAL '%d hours'
			ORDER BY recorded_at DESC`
		
		rows, err := db.Query(fmt.Sprintf(query, hours))
		if err != nil {
			logger.Printf("⚠️  Failed to query resource metrics: %v", err)
		} else {
			defer rows.Close()
			
			for rows.Next() {
				var memAvailable, memFree, memTotal, cpuUsage, swapUsed float64
				var recordedAt time.Time
				
				err := rows.Scan(&memAvailable, &memFree, &memTotal, &cpuUsage, &swapUsed, &recordedAt)
				if err != nil {
					logger.Printf("⚠️  Failed to scan resource metric: %v", err)
					continue
				}
				
				memoryPressure := 0.0
				if memTotal > 0 {
					memoryPressure = 1.0 - (memAvailable / memTotal)
				}
				
				historyPoint := map[string]interface{}{
					"timestamp":         recordedAt,
					"memoryPressure":    memoryPressure,
					"cpuUsage":          cpuUsage,
					"availableMemoryGb": memAvailable,
					"totalMemoryGb":     memTotal,
					"swapUsedPercent":   swapUsed,
				}
				history = append(history, historyPoint)
			}
		}
	}
	
	// If no historical data, create a single point with current data
	if len(history) == 0 {
		historyPoint := map[string]interface{}{
			"timestamp":         time.Now(),
			"memoryPressure":    current["memoryPressure"],
			"cpuUsage":          current["cpuUsage"],
			"availableMemoryGb": current["availableMemoryGb"],
			"totalMemoryGb":     current["totalMemoryGb"],
			"swapUsedPercent":   0.0,
		}
		history = []map[string]interface{}{historyPoint}
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
	
	if err := initOllama(); err != nil {
		logger.Printf("❌ Ollama initialization failed: %v", err)
		// Continue without Ollama for development (will fall back to simulation)
	}
	
	if err := initDocker(); err != nil {
		logger.Printf("❌ Docker initialization failed: %v", err)
		// Continue without Docker for development
	}
	
	// Start background system monitoring
	go startSystemMonitoring()
	
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