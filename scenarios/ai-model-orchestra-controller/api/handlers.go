package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Handler struct to hold application state
type Handlers struct {
	app *AppState
}

// NewHandlers creates a new handlers instance
func NewHandlers(app *AppState) *Handlers {
	return &Handlers{app: app}
}

func (h *Handlers) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	services := make(map[string]string)
	
	// Check database
	if h.app.DB != nil {
		if err := h.app.DB.Ping(); err != nil {
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
	if h.app.Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := h.app.Redis.Ping(ctx).Result()
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
	if h.app.OllamaClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if h.app.OllamaClient.IsHealthy(ctx) {
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
	if h.app.OllamaClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if models, err := h.app.OllamaClient.GetModels(ctx); err == nil {
			modelCount = len(models.Models)
		}
	}
	systemMetrics["available_models"] = modelCount
	
	// Add circuit breaker status
	if h.app.OllamaClient != nil && h.app.OllamaClient.CircuitBreaker != nil {
		cbState := h.app.OllamaClient.CircuitBreaker.GetState()
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
		systemMetrics["circuit_breaker_failures"] = h.app.OllamaClient.CircuitBreaker.failures
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

func (h *Handlers) handleModelSelect(w http.ResponseWriter, r *http.Request) {
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
	
	response, err := selectOptimalModel(req.TaskType, req.Requirements, h.app.OllamaClient, h.app.Logger)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

func (h *Handlers) handleRouteRequest(w http.ResponseWriter, r *http.Request) {
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
	
	response, err := routeAIRequest(&req, h.app.DB, h.app.OllamaClient, h.app.Logger)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

func (h *Handlers) handleModelsStatus(w http.ResponseWriter, r *http.Request) {
	var models []ModelMetric
	healthyCount := 0
	
	// Get real model data from database
	if h.app.DB != nil {
		query := `
			SELECT model_name, request_count, success_count, error_count, 
				   avg_response_time_ms, current_load, memory_usage_mb, 
				   healthy, last_used, created_at, updated_at
			FROM model_metrics 
			ORDER BY last_used DESC`
		
		rows, err := h.app.DB.Query(query)
		if err != nil {
			h.app.Logger.Printf("⚠️  Failed to query model metrics: %v", err)
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
					h.app.Logger.Printf("⚠️  Failed to scan model metric: %v", err)
					continue
				}
				
				model.ID = uuid.New()
				if lastUsed.Valid {
					model.LastUsed = &lastUsed.Time
				}
				
				// Get model capabilities from Ollama or defaults
				if h.app.OllamaClient != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					
					if ollamaModels, err := h.app.OllamaClient.GetModels(ctx); err == nil {
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
	if len(models) == 0 && h.app.OllamaClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if ollamaModels, err := h.app.OllamaClient.GetModels(ctx); err == nil {
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

func (h *Handlers) handleResourceMetrics(w http.ResponseWriter, r *http.Request) {
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
	
	if h.app.DB != nil {
		query := `
			SELECT memory_available_gb, memory_free_gb, memory_total_gb, 
				   cpu_usage_percent, swap_used_percent, recorded_at
			FROM system_resources 
			WHERE recorded_at >= NOW() - INTERVAL '%d hours'
			ORDER BY recorded_at DESC`
		
		rows, err := h.app.DB.Query(fmt.Sprintf(query, hours))
		if err != nil {
			h.app.Logger.Printf("⚠️  Failed to query resource metrics: %v", err)
		} else {
			defer rows.Close()
			
			for rows.Next() {
				var memAvailable, memFree, memTotal, cpuUsage, swapUsed float64
				var recordedAt time.Time
				
				err := rows.Scan(&memAvailable, &memFree, &memTotal, &cpuUsage, &swapUsed, &recordedAt)
				if err != nil {
					h.app.Logger.Printf("⚠️  Failed to scan resource metric: %v", err)
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

func (h *Handlers) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Simple dashboard redirect
	http.Redirect(w, r, "/ui/dashboard.html", http.StatusFound)
}