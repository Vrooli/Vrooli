package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/healthutil"
	"system-monitor-api/internal/httputil"
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

	httputil.JSON(w, healthResponse) //nolint:errcheck
}

// checkMetricsCollection tests the core system monitoring capability
func (h *HealthHandler) checkMetricsCollection(ctx context.Context) map[string]interface{} {
	result := healthutil.NewResult()

	// Test if we can collect system metrics
	metrics, err := h.monitorSvc.GetCurrentMetrics(ctx)
	if err != nil {
		return healthutil.WithError(result, "METRICS_COLLECTION_FAILED",
			fmt.Sprintf("Cannot collect system metrics: %v", err), "resource", true)
	}

	if metrics == nil {
		return healthutil.WithError(result, "METRICS_NULL",
			"Metrics collection returned null data", "internal", true)
	}

	result["latency_ms"] = nil // Could measure collection time if needed
	return healthutil.MarkConnected(result)
}

// checkInvestigationSystem tests the investigation file system access
func (h *HealthHandler) checkInvestigationSystem() map[string]interface{} {
	result := healthutil.NewResult()

	// Check if investigations directory exists and is accessible
	investigationsDir := "investigations"
	if _, err := os.Stat(investigationsDir); err != nil {
		return healthutil.WithError(result, "INVESTIGATION_DIR_ACCESS",
			fmt.Sprintf("Cannot access investigations directory: %v", err), "resource", false)
	}

	// Check if results directory exists and is writable
	resultsDir := "results"
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return healthutil.WithError(result, "RESULTS_DIR_WRITE",
			fmt.Sprintf("Cannot create/write results directory: %v", err), "resource", false)
	}

	// Test writing a small test file
	testFile := filepath.Join(resultsDir, ".health_check_test")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		return healthutil.WithError(result, "FILESYSTEM_WRITE_TEST",
			fmt.Sprintf("Cannot write test file: %v", err), "resource", true)
	}

	// Clean up test file
	os.Remove(testFile)

	return healthutil.MarkConnected(result)
}

// checkPostgreSQL tests PostgreSQL database connectivity
func (h *HealthHandler) checkPostgreSQL() map[string]interface{} {
	result := healthutil.NewResult()

	// This is a placeholder - you'd need to implement actual DB connection
	// For now, just check if URL is accessible
	if h.config.Resources.PostgresURL == "" {
		return healthutil.WithError(result, "POSTGRES_URL_MISSING",
			"PostgreSQL URL not configured", "configuration", false)
	}

	// TODO: Implement actual database ping when DB connection is available
	return healthutil.MarkConnected(result)
}

// checkRedis tests Redis connectivity
func (h *HealthHandler) checkRedis() map[string]interface{} {
	result := healthutil.NewResult()

	if h.config.Resources.RedisURL == "" {
		return healthutil.WithError(result, "REDIS_URL_MISSING",
			"Redis URL not configured", "configuration", false)
	}

	// TODO: Implement actual Redis ping
	return healthutil.MarkConnected(result)
}

// checkQuestDB tests QuestDB connectivity
func (h *HealthHandler) checkQuestDB() map[string]interface{} {
	result := healthutil.NewResult()

	if h.config.Resources.QuestDBURL == "" {
		return healthutil.WithError(result, "QUESTDB_URL_MISSING",
			"QuestDB URL not configured", "configuration", false)
	}

	// Test HTTP connection to QuestDB
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(h.config.Resources.QuestDBURL)
	if err != nil {
		return healthutil.WithError(result, "QUESTDB_CONNECTION_FAILED",
			fmt.Sprintf("Cannot connect to QuestDB: %v", err), "network", true)
	}
	defer resp.Body.Close()

	return healthutil.MarkConnected(result)
}

// checkNodeRed tests Node-RED connectivity
func (h *HealthHandler) checkNodeRed() map[string]interface{} {
	result := healthutil.NewResult()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(h.config.Resources.NodeRedURL)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return healthutil.WithError(result, "CONNECTION_REFUSED",
				"Node-RED service not running", "network", true)
		}
		return healthutil.WithError(result, "NODE_RED_CONNECTION_FAILED",
			fmt.Sprintf("Cannot connect to Node-RED: %v", err), "network", true)
	}
	defer resp.Body.Close()

	result["latency_ms"] = nil // Could measure if needed
	return healthutil.MarkConnected(result)
}

// checkOllama tests Ollama AI service connectivity
func (h *HealthHandler) checkOllama() map[string]interface{} {
	result := healthutil.NewResult()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(h.config.Resources.OllamaURL + "/api/tags")
	if err != nil {
		return healthutil.WithError(result, "OLLAMA_CONNECTION_FAILED",
			fmt.Sprintf("Cannot connect to Ollama: %v", err), "network", true)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return healthutil.WithError(result, fmt.Sprintf("HTTP_%d", resp.StatusCode),
			fmt.Sprintf("Ollama returned status %d", resp.StatusCode), "network", resp.StatusCode >= 500)
	}

	return healthutil.MarkConnected(result)
}

// checkWebhookEndpoint tests webhook URL accessibility
func (h *HealthHandler) checkWebhookEndpoint() map[string]interface{} {
	result := healthutil.NewResult()

	// For webhook, we just check if URL is reachable (don't actually send alert)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Head(h.config.Alerts.WebhookURL)
	if err != nil {
		return healthutil.WithError(result, "WEBHOOK_UNREACHABLE",
			fmt.Sprintf("Webhook URL unreachable: %v", err), "network", true)
	}
	defer resp.Body.Close()

	return healthutil.MarkConnected(result)
}
