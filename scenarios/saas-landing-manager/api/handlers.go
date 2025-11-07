package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gorilla/mux"
)

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// writeJSONError writes a standardized JSON error response
func writeJSONError(w http.ResponseWriter, statusCode int, errorMsg string, details ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error: errorMsg,
	}

	if len(details) > 0 {
		response.Message = details[0]
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// createHealthError constructs a standardized error structure for health checks
func createHealthError(code, message, category string, retryable bool) map[string]interface{} {
	return map[string]interface{}{
		"code":      code,
		"message":   message,
		"category":  category,
		"retryable": retryable,
	}
}

// HTTP Handlers (as Server methods)
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := "healthy"
	ready := true
	latency := interface{}(nil)
	dbErr := interface{}(nil)
	connected := false

	if s.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), HealthCheckTimeout)
		defer cancel()
		start := time.Now()
		err := s.db.PingContext(ctx)
		if err != nil {
			status = "degraded"
			ready = false
			dbErr = createHealthError("DB_PING_FAILED", err.Error(), "resource", true)
		} else {
			connected = true
			latency = float64(time.Since(start).Milliseconds())
		}
	} else {
		status = "degraded"
		ready = false
		dbErr = createHealthError("DB_UNINITIALIZED", "database connection not initialized", "configuration", true)
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   ServiceName,
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": ready,
		"version":   ServiceVersion,
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  connected,
				"latency_ms": latency,
				"error":      dbErr,
			},
		},
		"metrics": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}

func (s *Server) scanScenariosHandler(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Get scenarios path from environment or use default
	scenariosPath := os.Getenv("SCENARIOS_PATH")
	if scenariosPath == "" {
		scenariosPath = DefaultScenariosPath
	}

	detectionService := NewSaaSDetectionService(scenariosPath, s.dbService)

	response, err := detectionService.ScanScenarios(req.ForceRescan, req.ScenarioFilter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Scan failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode scan response: %v", err)
	}
}

func (s *Server) generateLandingPageHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	response, err := s.landingPageService.GenerateLandingPage(&req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Generation failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode generate response: %v", err)
	}
}

func (s *Server) deployLandingPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	landingPageID := vars["id"]

	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	response, err := s.claudeService.DeployLandingPage(landingPageID, req.TargetScenario, &req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Deployment failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode deploy response: %v", err)
	}
}

func (s *Server) getTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	saasType := r.URL.Query().Get("saas_type")

	templates, err := s.dbService.GetTemplates(category, saasType)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get templates", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
	}); err != nil {
		log.Printf("Failed to encode templates response: %v", err)
	}
}

func (s *Server) getDashboardHandler(w http.ResponseWriter, r *http.Request) {
	scenarios, err := s.dbService.GetSaaSScenarios()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get dashboard data", err.Error())
		return
	}

	// Count active A/B tests (landing pages with multiple variants)
	activeABTests, err := s.dbService.GetActiveABTestsCount()
	if err != nil {
		// Log but don't fail - this is non-critical metric
		activeABTests = 0
	}

	// Calculate average conversion rate from analytics
	avgConversionRate, err := s.dbService.GetAverageConversionRate()
	if err != nil {
		// Log but don't fail - this is non-critical metric
		avgConversionRate = 0.0
	}

	dashboard := map[string]interface{}{
		"total_pages":             len(scenarios),
		"active_ab_tests":         activeABTests,
		"average_conversion_rate": avgConversionRate,
		"scenarios":               scenarios,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dashboard); err != nil {
		log.Printf("Failed to encode dashboard response: %v", err)
	}
}
