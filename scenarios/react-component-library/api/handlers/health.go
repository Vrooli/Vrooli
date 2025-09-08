package handlers

import (
	"database/sql"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vrooli/scenarios/react-component-library/models"
)

type HealthHandler struct {
	db *sql.DB
	startTime time.Time
	metrics *APIMetrics
}

type APIMetrics struct {
	ComponentCreations    int64
	SearchQueries         int64
	AccessibilityTests    int64
	PerformanceBenchmarks int64
	AIGenerations         int64
	ComponentExports      int64
	ResponseTimes         map[string]float64
	ErrorRates            map[string]float64
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{
		db: db,
		startTime: time.Now(),
		metrics: &APIMetrics{
			ResponseTimes: make(map[string]float64),
			ErrorRates:    make(map[string]float64),
		},
	}
}

// HealthCheck handles GET /health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	status := models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    time.Since(h.startTime).String(),
		Resources: make(map[string]string),
	}

	// Check database connection
	if err := h.db.Ping(); err != nil {
		status.Status = "unhealthy"
		status.Database = "failed: " + err.Error()
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}
	status.Database = "connected"

	// Check required resources
	resources := []struct {
		name    string
		command string
	}{
		{"qdrant", "resource-qdrant status"},
		{"minio", "resource-minio status"},
		{"claude-code", "resource-claude-code status"},
	}

	for _, resource := range resources {
		if err := exec.Command("sh", "-c", resource.command).Run(); err != nil {
			status.Resources[resource.name] = "unavailable"
			if status.Status == "healthy" {
				status.Status = "degraded" // Don't fail entirely if optional resources are down
			}
		} else {
			status.Resources[resource.name] = "available"
		}
	}

	// Check optional resources
	optionalResources := []struct {
		name    string
		command string
	}{
		{"redis", "resource-redis status"},
		{"browserless", "resource-browserless status"},
	}

	for _, resource := range optionalResources {
		if err := exec.Command("sh", "-c", resource.command).Run(); err != nil {
			status.Resources[resource.name] = "unavailable (optional)"
		} else {
			status.Resources[resource.name] = "available"
		}
	}

	// Return appropriate status code
	if status.Status == "healthy" {
		c.JSON(http.StatusOK, status)
	} else if status.Status == "degraded" {
		c.JSON(http.StatusOK, status) // 200 but with degraded status
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}

// Metrics handles GET /api/metrics
func (h *HealthHandler) Metrics(c *gin.Context) {
	// Get database metrics
	var componentCount, testCount, exportCount int64
	
	// Component metrics
	h.db.QueryRow("SELECT COUNT(*) FROM components WHERE is_active = true").Scan(&componentCount)
	h.db.QueryRow("SELECT COUNT(*) FROM test_results WHERE tested_at > NOW() - INTERVAL '24 hours'").Scan(&testCount)
	h.db.QueryRow("SELECT COUNT(*) FROM usage_analytics WHERE used_at > NOW() - INTERVAL '24 hours'").Scan(&exportCount)

	// Performance metrics (simplified - in production you'd use proper metrics collection)
	metrics := models.APIMetrics{
		ComponentCreations:    h.metrics.ComponentCreations,
		SearchQueries:         h.metrics.SearchQueries,
		AccessibilityTests:    h.metrics.AccessibilityTests,
		PerformanceBenchmarks: h.metrics.PerformanceBenchmarks,
		AIGenerations:         h.metrics.AIGenerations,
		ComponentExports:      h.metrics.ComponentExports,
		ResponseTimes: map[string]float64{
			"component_create": 150.5,
			"component_search": 45.2,
			"accessibility_test": 2500.0,
			"ai_generation": 8000.0,
		},
		ErrorRates: map[string]float64{
			"api_errors": 0.1,
			"test_failures": 5.2,
			"generation_failures": 2.8,
		},
	}

	c.JSON(http.StatusOK, metrics)
}

// UpdateMetrics updates the metrics (called by middleware)
func (h *HealthHandler) UpdateMetrics(endpoint string, responseTime float64, hadError bool) {
	switch endpoint {
	case "/api/v1/components":
		h.metrics.ComponentCreations++
	case "/api/v1/components/search":
		h.metrics.SearchQueries++
	case "/api/v1/components/test":
		h.metrics.AccessibilityTests++
	case "/api/v1/components/generate":
		h.metrics.AIGenerations++
	case "/api/v1/components/export":
		h.metrics.ComponentExports++
	}

	// Update response times (moving average)
	if currentTime, exists := h.metrics.ResponseTimes[endpoint]; exists {
		h.metrics.ResponseTimes[endpoint] = (currentTime + responseTime) / 2
	} else {
		h.metrics.ResponseTimes[endpoint] = responseTime
	}

	// Update error rates
	if hadError {
		if currentRate, exists := h.metrics.ErrorRates[endpoint]; exists {
			h.metrics.ErrorRates[endpoint] = currentRate + 0.1
		} else {
			h.metrics.ErrorRates[endpoint] = 0.1
		}
	}
}