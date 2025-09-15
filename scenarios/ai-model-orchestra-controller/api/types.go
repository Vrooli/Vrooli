package main

import (
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

// Application state
type AppState struct {
	Logger       *log.Logger
	DB           *sql.DB
	Redis        *redis.Client
	DockerClient *client.Client
	OllamaClient *OllamaClient
	Mutex        sync.RWMutex
}