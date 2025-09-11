package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MetricsCollector handles system metrics and analytics
type MetricsCollector struct {
	enabled bool
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		enabled: true,
	}
}

// LogEvent logs an analytics event to the database
func (mc *MetricsCollector) LogEvent(db *sql.DB, eventType, userID, graphID, pluginID string, metadata map[string]interface{}, durationMs int) {
	if !mc.enabled {
		return
	}
	
	// Prepare metadata JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("Failed to marshal event metadata: %v", err)
		metadataJSON = []byte("{}")
	}
	
	// Insert analytics record (non-blocking)
	go func() {
		_, err := db.Exec(`
			INSERT INTO graph_analytics (
				id, graph_id, plugin_id, event_type, user_id, 
				metadata, duration_ms, timestamp
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, uuid.New().String(), graphID, pluginID, eventType, userID, 
			metadataJSON, durationMs, time.Now())
		
		if err != nil {
			log.Printf("Failed to log analytics event: %v", err)
		}
	}()
}

// LogAPIRequest logs an API request for monitoring
func (mc *MetricsCollector) LogAPIRequest(c *gin.Context, startTime time.Time, statusCode int, responseSize int64) {
	if !mc.enabled {
		return
	}
	
	duration := time.Since(startTime)
	userID := getUserID(c)
	requestID := c.GetString("request_id")
	
	// Structured logging
	log.Printf("[API] %s %s %d %v | User: %s | ReqID: %s | Size: %d",
		c.Request.Method,
		c.Request.URL.Path,
		statusCode,
		duration,
		userID,
		requestID,
		responseSize,
	)
	
	// Log to database for analytics
	if db, exists := c.Get("db"); exists && db != nil {
		metadata := map[string]interface{}{
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"status_code":  statusCode,
			"response_size": responseSize,
			"user_agent":   c.Request.UserAgent(),
			"ip_address":   c.ClientIP(),
		}
		
		mc.LogEvent(db.(*sql.DB), "api_request", userID, "", "", metadata, int(duration.Milliseconds()))
	}
}

// LogGraphOperation logs a graph-specific operation
func (mc *MetricsCollector) LogGraphOperation(db *sql.DB, operation, userID, graphID, pluginID string, duration time.Duration, success bool, metadata map[string]interface{}) {
	if !mc.enabled {
		return
	}
	
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["success"] = success
	metadata["operation"] = operation
	
	eventType := operation
	if !success {
		eventType = operation + "_failed"
	}
	
	mc.LogEvent(db, eventType, userID, graphID, pluginID, metadata, int(duration.Milliseconds()))
}

// LogConversion logs a graph conversion operation
func (mc *MetricsCollector) LogConversion(db *sql.DB, userID, sourceGraphID, targetGraphID, fromFormat, toFormat string, duration time.Duration, success bool, errorMsg string) {
	if !mc.enabled {
		return
	}
	
	metadata := map[string]interface{}{
		"from_format":      fromFormat,
		"to_format":        toFormat,
		"source_graph_id":  sourceGraphID,
		"target_graph_id":  targetGraphID,
		"success":          success,
	}
	
	if errorMsg != "" {
		metadata["error"] = errorMsg
	}
	
	eventType := "converted"
	if !success {
		eventType = "conversion_failed"
	}
	
	mc.LogEvent(db, eventType, userID, sourceGraphID, "", metadata, int(duration.Milliseconds()))
}

// GetSystemMetrics returns current system metrics
func (mc *MetricsCollector) GetSystemMetrics(db *sql.DB) map[string]interface{} {
	metrics := make(map[string]interface{})
	
	// Total graphs
	var totalGraphs int
	db.QueryRow("SELECT COUNT(*) FROM graphs").Scan(&totalGraphs)
	metrics["total_graphs"] = totalGraphs
	
	// Graphs by type
	graphsByType := make(map[string]int)
	rows, err := db.Query("SELECT type, COUNT(*) FROM graphs GROUP BY type")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var graphType string
			var count int
			if rows.Scan(&graphType, &count) == nil {
				graphsByType[graphType] = count
			}
		}
	}
	metrics["graphs_by_type"] = graphsByType
	
	// Recent activity (last 24 hours)
	var recentActivity int
	db.QueryRow(`
		SELECT COUNT(*) FROM graph_analytics 
		WHERE timestamp > NOW() - INTERVAL '24 hours'
	`).Scan(&recentActivity)
	metrics["recent_activity_24h"] = recentActivity
	
	// Active users (last 7 days)
	var activeUsers int
	db.QueryRow(`
		SELECT COUNT(DISTINCT user_id) FROM graph_analytics 
		WHERE timestamp > NOW() - INTERVAL '7 days'
	`).Scan(&activeUsers)
	metrics["active_users_7d"] = activeUsers
	
	// Conversion stats
	conversionStats := make(map[string]int)
	convRows, err := db.Query(`
		SELECT plugin_id, COUNT(*) FROM graph_analytics 
		WHERE event_type = 'converted' 
		AND timestamp > NOW() - INTERVAL '30 days'
		GROUP BY plugin_id
	`)
	if err == nil {
		defer convRows.Close()
		for convRows.Next() {
			var pluginID string
			var count int
			if convRows.Scan(&pluginID, &count) == nil {
				conversionStats[pluginID] = count
			}
		}
	}
	metrics["conversions_30d"] = conversionStats
	
	// Performance metrics
	var avgResponseTime float64
	db.QueryRow(`
		SELECT AVG(duration_ms) FROM graph_analytics 
		WHERE event_type = 'api_request' 
		AND timestamp > NOW() - INTERVAL '1 hour'
	`).Scan(&avgResponseTime)
	metrics["avg_response_time_1h"] = avgResponseTime
	
	// Error rates
	var errorRate float64
	db.QueryRow(`
		SELECT 
			CASE 
				WHEN total.count = 0 THEN 0 
				ELSE (errors.count::float / total.count::float) * 100 
			END as error_rate
		FROM 
			(SELECT COUNT(*) as count FROM graph_analytics WHERE timestamp > NOW() - INTERVAL '1 hour') total,
			(SELECT COUNT(*) as count FROM graph_analytics WHERE event_type LIKE '%_failed' AND timestamp > NOW() - INTERVAL '1 hour') errors
	`).Scan(&errorRate)
	metrics["error_rate_1h"] = errorRate
	
	return metrics
}

// GetHealthStatus returns detailed health status
func (mc *MetricsCollector) GetHealthStatus(db *sql.DB) map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"checks":    make(map[string]interface{}),
	}
	
	checks := health["checks"].(map[string]interface{})
	
	// Database health
	dbHealth := map[string]interface{}{
		"status": "healthy",
	}
	
	if err := db.Ping(); err != nil {
		dbHealth["status"] = "unhealthy"
		dbHealth["error"] = err.Error()
		health["status"] = "degraded"
	}
	
	// Database performance
	start := time.Now()
	db.QueryRow("SELECT 1")
	dbHealth["response_time_ms"] = time.Since(start).Milliseconds()
	
	checks["database"] = dbHealth
	
	// Plugin health
	pluginHealth := map[string]interface{}{
		"status": "healthy",
	}
	
	var pluginCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM plugins WHERE enabled = true").Scan(&pluginCount); err != nil {
		pluginHealth["status"] = "degraded"
		pluginHealth["error"] = err.Error()
	} else {
		pluginHealth["enabled_count"] = pluginCount
	}
	
	checks["plugins"] = pluginHealth
	
	// System resources
	resourceHealth := map[string]interface{}{
		"status": "healthy",
		"memory": getMemoryUsage(),
		"uptime": getUptime(),
	}
	
	checks["resources"] = resourceHealth
	
	return health
}

// Performance monitoring middleware
func (mc *MetricsCollector) PerformanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		// Calculate response size
		var responseSize int64
		if size := c.Writer.Size(); size > 0 {
			responseSize = int64(size)
		}
		
		// Log the request
		mc.LogAPIRequest(c, start, c.Writer.Status(), responseSize)
	}
}

// Error tracking middleware
func (mc *MetricsCollector) ErrorTrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		
		// Check for errors
		if len(c.Errors) > 0 {
			userID := getUserID(c)
			requestID := c.GetString("request_id")
			
			for _, err := range c.Errors {
				log.Printf("[ERROR] %s | User: %s | ReqID: %s | Error: %v",
					c.Request.URL.Path,
					userID,
					requestID,
					err.Error(),
				)
				
				// Log to database
				if db, exists := c.Get("db"); exists && db != nil {
					metadata := map[string]interface{}{
						"path":       c.Request.URL.Path,
						"method":     c.Request.Method,
						"error_type": err.Type,
						"error_msg":  err.Error(),
					}
					
					mc.LogEvent(db.(*sql.DB), "error", userID, "", "", metadata, 0)
				}
			}
		}
	}
}

// Security event logging
func (mc *MetricsCollector) LogSecurityEvent(db *sql.DB, eventType, userID, details string, severity string) {
	if !mc.enabled {
		return
	}
	
	metadata := map[string]interface{}{
		"severity":    severity,
		"details":     details,
		"ip_address":  "", // Would be filled by middleware
		"user_agent":  "", // Would be filled by middleware
	}
	
	log.Printf("[SECURITY] %s | User: %s | Severity: %s | Details: %s",
		eventType, userID, severity, details)
	
	mc.LogEvent(db, "security_"+eventType, userID, "", "", metadata, 0)
}

// Rate limiting helper
func (mc *MetricsCollector) CheckRateLimit(db *sql.DB, userID string, operation string, windowMinutes int, maxRequests int) bool {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM graph_analytics 
		WHERE user_id = $1 
		AND event_type = $2 
		AND timestamp > NOW() - INTERVAL '%d minutes'
	`, userID, operation, windowMinutes).Scan(&count)
	
	if err != nil {
		log.Printf("Error checking rate limit: %v", err)
		return true // Allow on error
	}
	
	return count < maxRequests
}

// Helper functions for system metrics

func getMemoryUsage() map[string]interface{} {
	// This would integrate with system monitoring in production
	return map[string]interface{}{
		"allocated_mb": 0,
		"heap_mb":      0,
		"stack_mb":     0,
	}
}

func getUptime() int64 {
	// This would track actual uptime in production
	return time.Now().Unix()
}

// Metrics export for external monitoring systems
func (mc *MetricsCollector) ExportPrometheusMetrics() string {
	// This would export metrics in Prometheus format
	return `# HELP graph_studio_requests_total Total number of requests
# TYPE graph_studio_requests_total counter
graph_studio_requests_total 0

# HELP graph_studio_graphs_total Total number of graphs
# TYPE graph_studio_graphs_total gauge
graph_studio_graphs_total 0

# HELP graph_studio_response_time_seconds Response time in seconds
# TYPE graph_studio_response_time_seconds histogram
graph_studio_response_time_seconds_bucket{le="0.1"} 0
graph_studio_response_time_seconds_bucket{le="0.5"} 0
graph_studio_response_time_seconds_bucket{le="1.0"} 0
graph_studio_response_time_seconds_bucket{le="+Inf"} 0
graph_studio_response_time_seconds_sum 0
graph_studio_response_time_seconds_count 0
`
}

// Cleanup old analytics data
func (mc *MetricsCollector) CleanupOldData(db *sql.DB, retentionDays int) error {
	_, err := db.Exec(`
		DELETE FROM graph_analytics 
		WHERE timestamp < NOW() - INTERVAL '%d days'
	`, retentionDays)
	
	if err != nil {
		log.Printf("Failed to cleanup old analytics data: %v", err)
		return err
	}
	
	log.Printf("Cleaned up analytics data older than %d days", retentionDays)
	return nil
}