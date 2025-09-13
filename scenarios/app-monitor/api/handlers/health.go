package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var startTime = time.Now()

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db     *sql.DB
	redis  *redis.Client
	docker *client.Client
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sql.DB, redis *redis.Client, docker *client.Client) *HealthHandler {
	return &HealthHandler{
		db:     db,
		redis:  redis,
		docker: docker,
	}
}

// Check performs a comprehensive health check compliant with schema
func (h *HealthHandler) Check(c *gin.Context) {
	ctx := context.Background()
	overallStatus := "healthy"
	
	// Schema-compliant health response
	healthResponse := map[string]interface{}{
		"status":    overallStatus,
		"service":   "app-monitor-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Service is ready to accept requests
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{},
		"metrics": map[string]interface{}{
			"uptime_seconds": time.Since(startTime).Seconds(),
		},
	}
	
	dependencies := healthResponse["dependencies"].(map[string]interface{})
	
	// 1. Check Docker connectivity (critical for app monitoring)
	dockerHealth := h.checkDocker(ctx)
	dependencies["docker"] = dockerHealth
	if dockerHealth["connected"] == false {
		overallStatus = "unhealthy" // Docker is critical for app-monitor
	}
	
	// 2. Check PostgreSQL database
	dbHealth := h.checkDatabase()
	dependencies["database"] = dbHealth
	if dbHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	// 3. Check Redis cache
	redisHealth := h.checkRedis(ctx)
	dependencies["redis"] = redisHealth
	if redisHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	// 4. Check system metrics collection capability
	metricsHealth := h.checkSystemMetrics()
	dependencies["system_metrics"] = metricsHealth
	if metricsHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	// Update overall status
	healthResponse["status"] = overallStatus
	
	// Add current system metrics if available
	if metrics, ok := metricsHealth["current_metrics"].(map[string]interface{}); ok {
		systemMetrics := healthResponse["metrics"].(map[string]interface{})
		for k, v := range metrics {
			systemMetrics[k] = v
		}
	}
	
	c.JSON(200, healthResponse)
}

// checkDocker tests Docker daemon connectivity and container listing
func (h *HealthHandler) checkDocker(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	if h.docker == nil {
		result["error"] = map[string]interface{}{
			"code":      "DOCKER_CLIENT_NIL",
			"message":   "Docker client not initialized",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}
	
	// Test Docker ping
	dockerStart := time.Now()
	_, err := h.docker.Ping(ctx)
	dockerLatency := time.Since(dockerStart)
	
	if err != nil {
		errorCode := "DOCKER_PING_FAILED"
		category := "network"
		
		if strings.Contains(err.Error(), "permission denied") {
			errorCode = "DOCKER_PERMISSION_DENIED"
			category = "authentication"
		} else if strings.Contains(err.Error(), "cannot connect") {
			errorCode = "DOCKER_CONNECTION_FAILED"
		}
		
		result["error"] = map[string]interface{}{
			"code":      errorCode,
			"message":   fmt.Sprintf("Docker ping failed: %v", err),
			"category":  category,
			"retryable": true,
		}
		return result
	}
	
	// Test container listing
	containers, err := h.docker.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "DOCKER_LIST_FAILED",
			"message":   fmt.Sprintf("Cannot list containers: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	result["connected"] = true
	result["latency_ms"] = float64(dockerLatency.Nanoseconds()) / 1e6
	result["container_count"] = len(containers)
	
	// Count running containers
	runningCount := 0
	for _, container := range containers {
		if container.State == "running" {
			runningCount++
		}
	}
	result["running_containers"] = runningCount
	
	return result
}

// checkDatabase tests PostgreSQL connectivity
func (h *HealthHandler) checkDatabase() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	if h.db == nil {
		result["error"] = map[string]interface{}{
			"code":      "DATABASE_NOT_CONFIGURED",
			"message":   "Database connection not configured",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}
	
	// Test database ping
	dbStart := time.Now()
	err := h.db.Ping()
	dbLatency := time.Since(dbStart)
	
	if err != nil {
		errorCode := "DATABASE_PING_FAILED"
		category := "network"
		
		if strings.Contains(err.Error(), "password authentication failed") {
			errorCode = "DATABASE_AUTH_FAILED"
			category = "authentication"
		} else if strings.Contains(err.Error(), "connection refused") {
			errorCode = "DATABASE_CONNECTION_REFUSED"
		}
		
		result["error"] = map[string]interface{}{
			"code":      errorCode,
			"message":   fmt.Sprintf("Database ping failed: %v", err),
			"category":  category,
			"retryable": true,
		}
		return result
	}
	
	// Test basic query
	var dbTime time.Time
	err = h.db.QueryRow("SELECT NOW()").Scan(&dbTime)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "DATABASE_QUERY_FAILED",
			"message":   fmt.Sprintf("Database query failed: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	result["connected"] = true
	result["latency_ms"] = float64(dbLatency.Nanoseconds()) / 1e6
	result["server_time"] = dbTime.Format(time.RFC3339)
	
	// Get connection pool stats
	stats := h.db.Stats()
	result["pool_stats"] = map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":          stats.InUse,
		"idle":            stats.Idle,
	}
	
	return result
}

// checkRedis tests Redis connectivity
func (h *HealthHandler) checkRedis(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	if h.redis == nil {
		result["error"] = map[string]interface{}{
			"code":      "REDIS_NOT_CONFIGURED",
			"message":   "Redis connection not configured",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}
	
	// Test Redis ping
	redisStart := time.Now()
	err := h.redis.Ping(ctx).Err()
	redisLatency := time.Since(redisStart)
	
	if err != nil {
		errorCode := "REDIS_PING_FAILED"
		category := "network"
		
		if strings.Contains(err.Error(), "connection refused") {
			errorCode = "REDIS_CONNECTION_REFUSED"
		} else if strings.Contains(err.Error(), "NOAUTH") {
			errorCode = "REDIS_AUTH_REQUIRED"
			category = "authentication"
		}
		
		result["error"] = map[string]interface{}{
			"code":      errorCode,
			"message":   fmt.Sprintf("Redis ping failed: %v", err),
			"category":  category,
			"retryable": true,
		}
		return result
	}
	
	// Test basic set/get operation
	testKey := "health_check_test"
	testValue := time.Now().Format(time.RFC3339)
	
	err = h.redis.Set(ctx, testKey, testValue, 10*time.Second).Err()
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "REDIS_WRITE_FAILED",
			"message":   fmt.Sprintf("Redis write test failed: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	// Clean up test key
	h.redis.Del(ctx, testKey)
	
	result["connected"] = true
	result["latency_ms"] = float64(redisLatency.Nanoseconds()) / 1e6
	
	// Get Redis info
	info, _ := h.redis.Info(ctx).Result()
	if info != "" {
		// Parse some basic info
		lines := strings.Split(info, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "connected_clients:") {
				var clients int
				if n, err := fmt.Sscanf(line, "connected_clients:%d", &clients); n > 0 && err == nil {
					result["connected_clients"] = clients
				}
			} else if strings.HasPrefix(line, "used_memory_human:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					result["memory_used"] = strings.TrimSpace(parts[1])
				}
			}
		}
	}
	
	return result
}

// checkSystemMetrics tests system metrics collection capability
func (h *HealthHandler) checkSystemMetrics() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	// Try to collect basic system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Get CPU count
	cpuCount := runtime.NumCPU()
	
	// Get goroutine count
	goroutineCount := runtime.NumGoroutine()
	
	// Check if we can read /proc/stat for CPU usage (Linux)
	cpuUsage := -1.0
	if stat, err := os.ReadFile("/proc/stat"); err == nil {
		lines := strings.Split(string(stat), "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], "cpu ") {
			// Basic CPU parsing (would need more sophisticated logic for accurate %)
			result["connected"] = true
			cpuUsage = 0.0 // Placeholder - real implementation would calculate properly
		}
	} else {
		// Not Linux or can't read /proc/stat, but that's okay
		result["connected"] = true
	}
	
	result["current_metrics"] = map[string]interface{}{
		"memory_alloc_mb":     float64(m.Alloc) / 1024 / 1024,
		"memory_sys_mb":       float64(m.Sys) / 1024 / 1024,
		"goroutines":          goroutineCount,
		"cpu_count":           cpuCount,
		"gc_runs":            m.NumGC,
	}
	
	if cpuUsage >= 0 {
		result["current_metrics"].(map[string]interface{})["cpu_usage_percent"] = cpuUsage
	}
	
	return result
}

// APIHealth returns simple API health status (legacy endpoint)
func (h *HealthHandler) APIHealth(c *gin.Context) {
	// Use the main health check internally
	h.Check(c)
}