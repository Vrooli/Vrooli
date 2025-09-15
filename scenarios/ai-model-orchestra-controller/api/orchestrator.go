package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
func storeOrchestratorRequest(db *sql.DB, requestID, taskType, selectedModel string, fallbackUsed bool, responseTimeMs int, success bool, errorMessage *string, resourcePressure, costEstimate float64) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	query := `
		INSERT INTO orchestrator_requests 
		(request_id, task_type, selected_model, fallback_used, response_time_ms, success, error_message, resource_pressure, cost_estimate)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := db.Exec(query, requestID, taskType, selectedModel, fallbackUsed, responseTimeMs, success, errorMessage, resourcePressure, costEstimate)
	if err != nil {
		return err
	}
	
	return nil
}

func storeSystemResources(db *sql.DB, memoryAvailableGB, memoryFreeGB, memoryTotalGB, cpuUsagePercent, swapUsedPercent float64) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	query := `
		INSERT INTO system_resources 
		(memory_available_gb, memory_free_gb, memory_total_gb, cpu_usage_percent, swap_used_percent)
		VALUES ($1, $2, $3, $4, $5)`
	
	_, err := db.Exec(query, memoryAvailableGB, memoryFreeGB, memoryTotalGB, cpuUsagePercent, swapUsedPercent)
	if err != nil {
		return err
	}
	
	return nil
}

func updateModelMetrics(db *sql.DB, modelName string, requestCount, successCount, errorCount int, avgResponseTime, currentLoad, memoryUsage float64, healthy bool) error {
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
		return err
	}
	
	return nil
}

// Model selection logic
func selectOptimalModel(taskType string, requirements map[string]interface{}, ollamaClient *OllamaClient, logger *log.Logger) (*ModelSelectResponse, error) {
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

// Route AI request with orchestration
func routeAIRequest(req *RouteRequest, db *sql.DB, ollamaClient *OllamaClient, logger *log.Logger) (*RouteResponse, error) {
	startTime := time.Now()
	
	// First select optimal model
	modelResponse, err := selectOptimalModel(req.TaskType, req.Requirements, ollamaClient, logger)
	if err != nil {
		return nil, err
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
			
			if db != nil {
				if dbErr := storeOrchestratorRequest(
					db,
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
					db,
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
			}
			
			return nil, fmt.Errorf("AI generation failed: %v", err)
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
	costEstimate := 0.0
	
	// Calculate cost estimate based on tokens and model pricing
	if tokensGenerated > 0 && ollamaClient != nil {
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
	
	if db != nil {
		if err := storeOrchestratorRequest(
			db,
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
			db,
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
	}
	
	metrics := map[string]interface{}{
		"responseTimeMs":  responseTimeMs,
		"memoryPressure":  modelResponse.SystemMetrics["memoryPressure"],
		"modelUsed":       modelResponse.SelectedModel,
		"tokensGenerated": tokensGenerated,
		"promptTokens":    promptTokens,
		"costEstimate":    costEstimate,
	}
	
	return &RouteResponse{
		RequestID:     modelResponse.RequestID,
		SelectedModel: modelResponse.SelectedModel,
		Response:      responseText,
		FallbackUsed:  modelResponse.FallbackUsed,
		Metrics:       metrics,
	}, nil
}