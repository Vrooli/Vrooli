package handlers

import (
	"fmt"
	"net/http"
	"time"

	"metareasoning-api/internal/container"
	"metareasoning-api/internal/domain/ai"
	"metareasoning-api/internal/domain/execution"
	"metareasoning-api/internal/domain/workflow"
	"metareasoning-api/internal/pkg/errors"
	"metareasoning-api/internal/pkg/interfaces"
)

// SystemHandler handles system-level HTTP requests
type SystemHandler struct {
	container       *container.Container
	workflowService workflow.Service
	aiService       ai.GenerationService
	executionEngine execution.ExecutionEngine
	errorHandler    ErrorHandler
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(
	container *container.Container,
	workflowService workflow.Service,
	aiService ai.GenerationService,
	executionEngine execution.ExecutionEngine,
	errorHandler ErrorHandler,
) *SystemHandler {
	return &SystemHandler{
		container:       container,
		workflowService: workflowService,
		aiService:       aiService,
		executionEngine: executionEngine,
		errorHandler:    errorHandler,
	}
}

// HealthCheck handles GET /health
func (h *SystemHandler) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	start := time.Now()
	
	// Quick database ping
	dbHealthy := h.container.DB.Ping() == nil
	
	status := "healthy"
	if !dbHealthy {
		status = "degraded"
	}
	
	response := HealthResponse{
		Status:    status,
		Version:   "3.0.0",
		Timestamp: time.Now(),
		Database:  dbHealthy,
		Uptime:    formatUptime(time.Since(start)), // Placeholder - would use actual start time
		Metrics: map[string]interface{}{
			"response_time_ms": float64(time.Since(start).Nanoseconds()) / 1e6,
			"memory_usage":     getMemoryUsage(), // Placeholder
		},
	}
	
	statusCode := http.StatusOK
	if !dbHealthy {
		statusCode = http.StatusServiceUnavailable
	}
	
	return WriteJSON(w, statusCode, response)
}

// HealthCheckFull handles GET /health/full
func (h *SystemHandler) HealthCheckFull(w http.ResponseWriter, r *http.Request) error {
	start := time.Now()
	
	// Check database
	dbHealthy := h.container.DB.Ping() == nil
	
	// Check external services
	services := map[string]bool{
		"database":   dbHealthy,
		"n8n":        h.checkServiceHealth(h.container.Config.N8nBase),
		"windmill":   h.checkServiceHealth(h.container.Config.WindmillBase),
		"ollama":     h.checkServiceHealth(h.container.Config.OllamaBase),
	}
	
	// Get system stats
	stats, err := h.getSystemStats()
	if err != nil {
		// Log error but don't fail health check
		stats = map[string]interface{}{"error": "failed to retrieve stats"}
	}
	
	// Determine overall status
	status := "healthy"
	healthyServices := 0
	for _, healthy := range services {
		if healthy {
			healthyServices++
		}
	}
	
	if healthyServices == len(services) {
		status = "healthy"
	} else if healthyServices > len(services)/2 {
		status = "degraded"
	} else {
		status = "unhealthy"
	}
	
	response := HealthResponse{
		Status:    status,
		Version:   "3.0.0",
		Timestamp: time.Now(),
		Services:  services,
		Database:  dbHealthy,
		Uptime:    formatUptime(time.Since(start)), // Placeholder
		Metrics: map[string]interface{}{
			"response_time_ms": float64(time.Since(start).Nanoseconds()) / 1e6,
			"services_healthy": healthyServices,
			"services_total":   len(services),
			"system_stats":     stats,
		},
	}
	
	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	return WriteJSON(w, statusCode, response)
}

// GetPlatforms handles GET /platforms
func (h *SystemHandler) GetPlatforms(w http.ResponseWriter, r *http.Request) error {
	// Get supported platforms from execution engine
	supportedPlatforms := h.executionEngine.ListPlatforms()
	
	platforms := make([]map[string]interface{}, 0, len(supportedPlatforms))
	
	for _, platform := range supportedPlatforms {
		var baseURL string
		var description string
		
		switch platform.String() {
		case "n8n":
			baseURL = h.container.Config.N8nBase
			description = "n8n workflow automation platform"
		case "windmill":
			baseURL = h.container.Config.WindmillBase
			description = "Windmill script execution platform"
		default:
			baseURL = "unknown"
			description = "Unknown platform"
		}
		
		platformInfo := map[string]interface{}{
			"name":        platform.String(),
			"description": description,
			"status":      h.checkServiceHealth(baseURL),
			"url":         baseURL,
			"capabilities": []string{"execute", "validate", "import", "export"},
		}
		
		platforms = append(platforms, platformInfo)
	}
	
	response := map[string]interface{}{
		"platforms": platforms,
		"count":     len(platforms),
		"timestamp": time.Now(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// GetModels handles GET /models
func (h *SystemHandler) GetModels(w http.ResponseWriter, r *http.Request) error {
	// Get models from AI service
	models, err := h.aiService.ListModels()
	if err != nil {
		return errors.WithOperation(err, "list_models")
	}
	
	response := map[string]interface{}{
		"models":    models,
		"count":     len(models),
		"timestamp": time.Now(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// GetStats handles GET /stats
func (h *SystemHandler) GetStats(w http.ResponseWriter, r *http.Request) error {
	stats, err := h.getSystemStats()
	if err != nil {
		return errors.WithOperation(err, "get_system_stats")
	}
	
	return WriteJSON(w, http.StatusOK, stats)
}

// GetCircuitBreakerStats handles GET /circuit-breakers
func (h *SystemHandler) GetCircuitBreakerStats(w http.ResponseWriter, r *http.Request) error {
	stats := h.container.CircuitBreakers.GetAllStats()
	
	response := map[string]interface{}{
		"circuit_breakers": stats,
		"timestamp":       time.Now(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// GetCacheStats handles GET /cache
func (h *SystemHandler) GetCacheStats(w http.ResponseWriter, r *http.Request) error {
	stats := h.container.Cache.Stats()
	
	response := map[string]interface{}{
		"cache_stats": stats,
		"timestamp":   time.Now(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// GetVersion handles GET /version
func (h *SystemHandler) GetVersion(w http.ResponseWriter, r *http.Request) error {
	response := map[string]interface{}{
		"version":     "3.0.0",
		"api_version": "v1",
		"build_date":  "2025-08-12",
		"commit_hash": "dev-build",
		"go_version":  "1.21+",
		"timestamp":   time.Now(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// GetOpenAPISpec handles GET /openapi.json
func (h *SystemHandler) GetOpenAPISpec(w http.ResponseWriter, r *http.Request) error {
	// Placeholder for OpenAPI specification
	// In a real implementation, this would return the full OpenAPI 3.0 spec
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "Metareasoning API",
			"version":     "3.0.0",
			"description": "AI-powered workflow orchestration and metareasoning API",
			"contact": map[string]interface{}{
				"name": "Vrooli Team",
			},
		},
		"paths": map[string]interface{}{
			"/health":           "Health check endpoints",
			"/workflows":        "Workflow management endpoints",
			"/workflows/*/execute": "Workflow execution endpoints",
			"/platforms":        "Platform information endpoints",
			"/models":           "AI model information endpoints",
		},
		"message": "Full OpenAPI specification would be available here",
	}
	
	return WriteJSON(w, http.StatusOK, spec)
}

// Helper methods

func (h *SystemHandler) checkServiceHealth(baseURL string) bool {
	if baseURL == "" {
		return false
	}
	
	client := h.container.HTTPClients.GetClient(interfaces.HealthCheckClient)
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

func (h *SystemHandler) getSystemStats() (map[string]interface{}, error) {
	// Get workflow count
	var workflowCount int
	err := h.container.DB.QueryRow("SELECT COUNT(*) FROM workflows WHERE is_active = true").Scan(&workflowCount)
	if err != nil {
		workflowCount = 0
	}
	
	// Get execution count
	var executionCount int
	err = h.container.DB.QueryRow("SELECT COUNT(*) FROM execution_history").Scan(&executionCount)
	if err != nil {
		executionCount = 0
	}
	
	stats := map[string]interface{}{
		"workflows": map[string]interface{}{
			"total_active": workflowCount,
		},
		"executions": map[string]interface{}{
			"total_count": executionCount,
		},
		"cache": h.container.Cache.Stats(),
		"circuit_breakers": map[string]interface{}{
			"total_breakers": len(h.container.CircuitBreakers.GetAllStats()),
		},
		"timestamp": time.Now(),
	}
	
	return stats, nil
}

func formatUptime(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func getMemoryUsage() map[string]interface{} {
	// Placeholder - would use runtime.MemStats in real implementation
	return map[string]interface{}{
		"allocated_mb":   42,
		"heap_mb":        38,
		"stack_mb":       4,
		"gc_collections": 15,
	}
}

