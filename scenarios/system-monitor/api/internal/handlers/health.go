package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/services"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	config      *config.Config
	monitorSvc  *services.MonitorService
	settingsMgr *services.SettingsManager
	startTime   time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(cfg *config.Config, monitorSvc *services.MonitorService, settingsMgr *services.SettingsManager) *HealthHandler {
	return &HealthHandler{
		config:      cfg,
		monitorSvc:  monitorSvc,
		settingsMgr: settingsMgr,
		startTime:   time.Now(),
	}
}

// Handle processes health check requests
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	overallStatus := "healthy"

	// Schema-compliant health response
	healthResponse := map[string]interface{}{
		"status":    overallStatus,
		"service":   h.config.Server.ServiceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Service is ready to accept requests
		"version":   h.config.Server.Version,
		"processor_active": func() bool {
			if h.settingsMgr == nil {
				return true
			}
			return h.settingsMgr.IsActive()
		}(),
		"maintenance_state": func() string {
			if h.settingsMgr == nil {
				return "unknown"
			}
			return h.settingsMgr.GetMaintenanceState()
		}(),
		"dependencies": map[string]interface{}{},
		"metrics": map[string]interface{}{
			"uptime_seconds": time.Since(h.startTime).Seconds(),
		},
	}

	dependencies := healthResponse["dependencies"].(map[string]interface{})

	// 1. Check system metrics collection capability (core functionality)
	metricsHealth := h.checkMetricsCollection(ctx)
	dependencies["metrics_collection"] = metricsHealth
	if metricsHealth["connected"] == false {
		overallStatus = "degraded"
	}

	// 2. Check investigation system (file I/O for investigations and results)
	investigationHealth := h.checkInvestigationSystem()
	dependencies["investigation_system"] = investigationHealth
	if investigationHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}

	// 3. Check configured databases
	if h.config.Resources.PostgresURL != "" {
		pgHealth := h.checkPostgreSQL()
		dependencies["postgresql"] = pgHealth
		if pgHealth["connected"] == false {
			overallStatus = "degraded"
		}
	}

	if h.config.Resources.RedisURL != "" {
		redisHealth := h.checkRedis()
		dependencies["redis"] = redisHealth
		if redisHealth["connected"] == false {
			overallStatus = "degraded"
		}
	}

	if h.config.Resources.QuestDBURL != "" {
		questdbHealth := h.checkQuestDB()
		dependencies["questdb"] = questdbHealth
		if questdbHealth["connected"] == false {
			overallStatus = "degraded"
		}
	}

	// 4. Check external services (if configured)
	if h.config.Resources.NodeRedURL != "" {
		nodeRedHealth := h.checkNodeRed()
		dependencies["node_red"] = nodeRedHealth
		if nodeRedHealth["connected"] == false {
			// Node-RED is optional, so only degrade if we were healthy
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}
	}

	if h.config.Resources.OllamaURL != "" {
		ollamaHealth := h.checkOllama()
		dependencies["ollama"] = ollamaHealth
		if ollamaHealth["connected"] == false {
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}
	}

	// 5. Check alert systems (if enabled)
	if h.config.Alerts.EnableWebhooks && h.config.Alerts.WebhookURL != "" {
		webhookHealth := h.checkWebhookEndpoint()
		dependencies["webhooks"] = webhookHealth
		if webhookHealth["connected"] == false {
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}
	}

	// Update overall status and metrics
	healthResponse["status"] = overallStatus

	// Add system resource metrics if available
	if metrics, err := h.monitorSvc.GetCurrentMetrics(ctx); err == nil && metrics != nil {
		systemMetrics := healthResponse["metrics"].(map[string]interface{})
		systemMetrics["cpu_usage_percent"] = metrics.CPUUsage
		systemMetrics["memory_usage_percent"] = metrics.MemoryUsage
		if h.settingsMgr != nil {
			systemMetrics["active_monitoring"] = h.settingsMgr.IsActive()
		} else {
			systemMetrics["active_monitoring"] = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse)
}

// checkMetricsCollection tests the core system monitoring capability
func (h *HealthHandler) checkMetricsCollection(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	// Test if we can collect system metrics
	metrics, err := h.monitorSvc.GetCurrentMetrics(ctx)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "METRICS_COLLECTION_FAILED",
			"message":   fmt.Sprintf("Cannot collect system metrics: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}

	if metrics == nil {
		result["error"] = map[string]interface{}{
			"code":      "METRICS_NULL",
			"message":   "Metrics collection returned null data",
			"category":  "internal",
			"retryable": true,
		}
		return result
	}

	result["connected"] = true
	result["latency_ms"] = nil // Could measure collection time if needed
	return result
}

// checkInvestigationSystem tests the investigation file system access
func (h *HealthHandler) checkInvestigationSystem() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	// Check if investigations directory exists and is accessible
	investigationsDir := "investigations"
	if _, err := os.Stat(investigationsDir); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "INVESTIGATION_DIR_ACCESS",
			"message":   fmt.Sprintf("Cannot access investigations directory: %v", err),
			"category":  "resource",
			"retryable": false,
		}
		return result
	}

	// Check if results directory exists and is writable
	resultsDir := "results"
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "RESULTS_DIR_WRITE",
			"message":   fmt.Sprintf("Cannot create/write results directory: %v", err),
			"category":  "resource",
			"retryable": false,
		}
		return result
	}

	// Test writing a small test file
	testFile := filepath.Join(resultsDir, ".health_check_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "FILESYSTEM_WRITE_TEST",
			"message":   fmt.Sprintf("Cannot write test file: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}

	// Clean up test file
	os.Remove(testFile)

	result["connected"] = true
	return result
}

// checkPostgreSQL tests PostgreSQL database connectivity
func (h *HealthHandler) checkPostgreSQL() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	// This is a placeholder - you'd need to implement actual DB connection
	// For now, just check if URL is accessible
	if h.config.Resources.PostgresURL == "" {
		result["error"] = map[string]interface{}{
			"code":      "POSTGRES_URL_MISSING",
			"message":   "PostgreSQL URL not configured",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}

	// TODO: Implement actual database ping when DB connection is available
	result["connected"] = true
	return result
}

// checkRedis tests Redis connectivity
func (h *HealthHandler) checkRedis() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	if h.config.Resources.RedisURL == "" {
		result["error"] = map[string]interface{}{
			"code":      "REDIS_URL_MISSING",
			"message":   "Redis URL not configured",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}

	// TODO: Implement actual Redis ping
	result["connected"] = true
	return result
}

// checkQuestDB tests QuestDB connectivity
func (h *HealthHandler) checkQuestDB() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	if h.config.Resources.QuestDBURL == "" {
		result["error"] = map[string]interface{}{
			"code":      "QUESTDB_URL_MISSING",
			"message":   "QuestDB URL not configured",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}

	// Test HTTP connection to QuestDB
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(h.config.Resources.QuestDBURL)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "QUESTDB_CONNECTION_FAILED",
			"message":   fmt.Sprintf("Cannot connect to QuestDB: %v", err),
			"category":  "network",
			"retryable": true,
		}
		return result
	}
	defer resp.Body.Close()

	result["connected"] = true
	return result
}

// checkNodeRed tests Node-RED connectivity
func (h *HealthHandler) checkNodeRed() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(h.config.Resources.NodeRedURL)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			result["error"] = map[string]interface{}{
				"code":      "CONNECTION_REFUSED",
				"message":   "Node-RED service not running",
				"category":  "network",
				"retryable": true,
			}
		} else {
			result["error"] = map[string]interface{}{
				"code":      "NODE_RED_CONNECTION_FAILED",
				"message":   fmt.Sprintf("Cannot connect to Node-RED: %v", err),
				"category":  "network",
				"retryable": true,
			}
		}
		return result
	}
	defer resp.Body.Close()

	result["connected"] = true
	result["latency_ms"] = nil // Could measure if needed
	return result
}

// checkOllama tests Ollama AI service connectivity
func (h *HealthHandler) checkOllama() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(h.config.Resources.OllamaURL + "/api/tags")
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "OLLAMA_CONNECTION_FAILED",
			"message":   fmt.Sprintf("Cannot connect to Ollama: %v", err),
			"category":  "network",
			"retryable": true,
		}
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result["error"] = map[string]interface{}{
			"code":      fmt.Sprintf("HTTP_%d", resp.StatusCode),
			"message":   fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
			"category":  "network",
			"retryable": resp.StatusCode >= 500,
		}
		return result
	}

	result["connected"] = true
	return result
}

// checkWebhookEndpoint tests webhook URL accessibility
func (h *HealthHandler) checkWebhookEndpoint() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}

	// For webhook, we just check if URL is reachable (don't actually send alert)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Head(h.config.Alerts.WebhookURL)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "WEBHOOK_UNREACHABLE",
			"message":   fmt.Sprintf("Webhook URL unreachable: %v", err),
			"category":  "network",
			"retryable": true,
		}
		return result
	}
	defer resp.Body.Close()

	result["connected"] = true
	return result
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, code int, err error) {
	respondWithJSON(w, code, map[string]string{
		"error":  err.Error(),
		"status": "error",
	})
}
