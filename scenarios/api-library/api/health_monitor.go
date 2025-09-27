package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// HealthCheckResult represents the result of an API health check
type HealthCheckResult struct {
	ID           string    `json:"id"`
	APIID        string    `json:"api_id"`
	CheckTime    time.Time `json:"check_time"`
	ResponseTime int       `json:"response_time_ms"`
	StatusCode   int       `json:"status_code"`
	Healthy      bool      `json:"healthy"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// APIHealthMetrics represents aggregated health metrics for an API
type APIHealthMetrics struct {
	APIID               string     `json:"api_id"`
	LastCheck           time.Time  `json:"last_check"`
	UptimePercentage    float64    `json:"uptime_percentage"`
	AverageResponseTime int        `json:"avg_response_time_ms"`
	TotalChecks         int        `json:"total_checks"`
	SuccessfulChecks    int        `json:"successful_checks"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastHealthyCheck    *time.Time `json:"last_healthy_check,omitempty"`
	LastUnhealthyCheck  *time.Time `json:"last_unhealthy_check,omitempty"`
	CurrentStatus       string     `json:"current_status"` // "healthy", "degraded", "unhealthy", "unknown"
}

// HealthMonitor manages periodic health checks for APIs
type HealthMonitor struct {
	db            *sql.DB
	checkInterval time.Duration
	mu            sync.RWMutex
	activeChecks  map[string]bool
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(db *sql.DB) *HealthMonitor {
	hm := &HealthMonitor{
		db:            db,
		checkInterval: 5 * time.Minute, // Check every 5 minutes
		activeChecks:  make(map[string]bool),
		stopCh:        make(chan struct{}),
	}

	// Start the monitoring routine
	go hm.monitoringLoop()

	return hm
}

// monitoringLoop periodically checks API health
func (hm *HealthMonitor) monitoringLoop() {
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	// Run initial check immediately
	hm.checkAllAPIs()

	for {
		select {
		case <-ticker.C:
			hm.checkAllAPIs()
		case <-hm.stopCh:
			return
		}
	}
}

// checkAllAPIs checks the health of all APIs
func (hm *HealthMonitor) checkAllAPIs() {
	rows, err := hm.db.Query(`
		SELECT id, name, base_url, status 
		FROM apis 
		WHERE status IN ('active', 'beta') AND base_url IS NOT NULL AND base_url != ''
	`)
	if err != nil {
		log.Printf("Error fetching APIs for health check: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var apiID, name, baseURL, status string
		if err := rows.Scan(&apiID, &name, &baseURL, &status); err != nil {
			continue
		}

		// Skip if already checking this API
		hm.mu.Lock()
		if hm.activeChecks[apiID] {
			hm.mu.Unlock()
			continue
		}
		hm.activeChecks[apiID] = true
		hm.mu.Unlock()

		hm.wg.Add(1)
		go func(id, apiName, url string) {
			defer hm.wg.Done()
			defer func() {
				hm.mu.Lock()
				delete(hm.activeChecks, id)
				hm.mu.Unlock()
			}()

			hm.checkAPIHealth(id, apiName, url)
		}(apiID, name, baseURL)
	}
}

// checkAPIHealth performs a health check on a single API
func (hm *HealthMonitor) checkAPIHealth(apiID, apiName, baseURL string) {
	startTime := time.Now()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try to fetch the base URL
	resp, err := client.Get(baseURL)

	responseTime := int(time.Since(startTime).Milliseconds())
	statusCode := 0
	healthy := false
	errorMessage := ""

	if err != nil {
		errorMessage = fmt.Sprintf("Failed to reach API: %v", err)
		log.Printf("Health check failed for %s: %v", apiName, err)
	} else {
		statusCode = resp.StatusCode
		healthy = statusCode >= 200 && statusCode < 400
		resp.Body.Close()

		if !healthy {
			errorMessage = fmt.Sprintf("API returned status %d", statusCode)
		}
	}

	// Store the health check result
	resultID := uuid.New().String()
	_, err = hm.db.Exec(`
		INSERT INTO api_health_checks (id, api_id, check_time, response_time_ms, status_code, healthy, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, resultID, apiID, time.Now(), responseTime, statusCode, healthy, errorMessage)

	if err != nil {
		log.Printf("Failed to store health check result: %v", err)
	}

	// Update API health metrics
	hm.updateAPIHealthMetrics(apiID)

	// Trigger webhook if status changed
	hm.notifyHealthChange(apiID, apiName, healthy)
}

// updateAPIHealthMetrics updates the aggregated health metrics for an API
func (hm *HealthMonitor) updateAPIHealthMetrics(apiID string) {
	// Get recent health check data (last 7 days)
	var totalChecks, successfulChecks, totalResponseTime, consecutiveFailures int
	var lastHealthy, lastUnhealthy sql.NullTime

	err := hm.db.QueryRow(`
		WITH recent_checks AS (
			SELECT * FROM api_health_checks 
			WHERE api_id = $1 AND check_time > NOW() - INTERVAL '7 days'
			ORDER BY check_time DESC
		),
		consecutive_failures AS (
			SELECT COUNT(*) AS failures
			FROM api_health_checks
			WHERE api_id = $1 
			AND check_time > (
				SELECT COALESCE(MAX(check_time), '1970-01-01'::timestamp)
				FROM api_health_checks
				WHERE api_id = $1 AND healthy = true
			)
			AND healthy = false
		)
		SELECT 
			COUNT(*) AS total_checks,
			COUNT(*) FILTER (WHERE healthy = true) AS successful_checks,
			COALESCE(SUM(response_time_ms), 0) AS total_response_time,
			(SELECT failures FROM consecutive_failures),
			MAX(check_time) FILTER (WHERE healthy = true) AS last_healthy,
			MAX(check_time) FILTER (WHERE healthy = false) AS last_unhealthy
		FROM recent_checks
	`, apiID).Scan(&totalChecks, &successfulChecks, &totalResponseTime, &consecutiveFailures, &lastHealthy, &lastUnhealthy)

	if err != nil {
		log.Printf("Failed to calculate health metrics: %v", err)
		return
	}

	// Calculate metrics
	uptimePercentage := 0.0
	avgResponseTime := 0
	if totalChecks > 0 {
		uptimePercentage = (float64(successfulChecks) / float64(totalChecks)) * 100
		avgResponseTime = totalResponseTime / totalChecks
	}

	// Determine current status
	currentStatus := "unknown"
	if totalChecks > 0 {
		if consecutiveFailures == 0 {
			currentStatus = "healthy"
		} else if consecutiveFailures < 3 {
			currentStatus = "degraded"
		} else {
			currentStatus = "unhealthy"
		}
	}

	// Store or update metrics
	_, err = hm.db.Exec(`
		INSERT INTO api_health_metrics (id, api_id, uptime_percentage, avg_response_time_ms, 
			total_checks, successful_checks, consecutive_failures, current_status, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (api_id) DO UPDATE SET
			uptime_percentage = $3,
			avg_response_time_ms = $4,
			total_checks = $5,
			successful_checks = $6,
			consecutive_failures = $7,
			current_status = $8,
			last_updated = $9
	`, uuid.New().String(), apiID, uptimePercentage, avgResponseTime,
		totalChecks, successfulChecks, consecutiveFailures, currentStatus, time.Now())

	if err != nil {
		log.Printf("Failed to update health metrics: %v", err)
	}
}

// notifyHealthChange triggers a webhook if the health status changed
func (hm *HealthMonitor) notifyHealthChange(apiID, apiName string, currentHealthy bool) {
	// Check previous status
	var previousHealthy sql.NullBool
	err := hm.db.QueryRow(`
		SELECT healthy FROM api_health_checks 
		WHERE api_id = $1 AND check_time < NOW()
		ORDER BY check_time DESC 
		LIMIT 1 OFFSET 1
	`, apiID).Scan(&previousHealthy)

	if err == nil && previousHealthy.Valid && previousHealthy.Bool != currentHealthy {
		// Status changed, trigger webhook
		eventType := "api.health.recovered"
		if !currentHealthy {
			eventType = "api.health.degraded"
		}

		if webhookManager != nil {
			webhookManager.TriggerEvent(eventType, map[string]interface{}{
				"api_id":    apiID,
				"api_name":  apiName,
				"healthy":   currentHealthy,
				"timestamp": time.Now(),
			})
		}
	}
}

// Stop gracefully stops the health monitor
func (hm *HealthMonitor) Stop() {
	close(hm.stopCh)
	hm.wg.Wait()
}

// Handler functions for health monitoring

// getAPIHealthHandler returns health metrics for a specific API
func getAPIHealthHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var metrics APIHealthMetrics
	var lastHealthy, lastUnhealthy sql.NullTime

	err := db.QueryRow(`
		SELECT api_id, uptime_percentage, avg_response_time_ms, total_checks, 
			successful_checks, consecutive_failures, current_status, last_updated
		FROM api_health_metrics
		WHERE api_id = $1
	`, apiID).Scan(&metrics.APIID, &metrics.UptimePercentage, &metrics.AverageResponseTime,
		&metrics.TotalChecks, &metrics.SuccessfulChecks, &metrics.ConsecutiveFailures,
		&metrics.CurrentStatus, &metrics.LastCheck)

	if err == sql.ErrNoRows {
		// No health data yet
		metrics = APIHealthMetrics{
			APIID:         apiID,
			CurrentStatus: "unknown",
		}
	} else if err != nil {
		http.Error(w, "Failed to fetch health metrics", http.StatusInternalServerError)
		return
	}

	// Get last healthy/unhealthy times
	db.QueryRow(`
		SELECT 
			MAX(check_time) FILTER (WHERE healthy = true),
			MAX(check_time) FILTER (WHERE healthy = false)
		FROM api_health_checks
		WHERE api_id = $1
	`, apiID).Scan(&lastHealthy, &lastUnhealthy)

	if lastHealthy.Valid {
		metrics.LastHealthyCheck = &lastHealthy.Time
	}
	if lastUnhealthy.Valid {
		metrics.LastUnhealthyCheck = &lastUnhealthy.Time
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// getHealthHistoryHandler returns health check history for an API
func getHealthHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	// Parse query parameters
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit > 1000 {
			limit = 1000
		}
	}

	rows, err := db.Query(`
		SELECT id, api_id, check_time, response_time_ms, status_code, healthy, error_message
		FROM api_health_checks
		WHERE api_id = $1
		ORDER BY check_time DESC
		LIMIT $2
	`, apiID, limit)

	if err != nil {
		http.Error(w, "Failed to fetch health history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []HealthCheckResult
	for rows.Next() {
		var result HealthCheckResult
		var errorMsg sql.NullString

		err := rows.Scan(&result.ID, &result.APIID, &result.CheckTime,
			&result.ResponseTime, &result.StatusCode, &result.Healthy, &errorMsg)
		if err != nil {
			continue
		}

		if errorMsg.Valid {
			result.ErrorMessage = errorMsg.String
		}

		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// triggerHealthCheckHandler manually triggers a health check for an API
func triggerHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	// Get API details
	var name, baseURL string
	err := db.QueryRow("SELECT name, base_url FROM apis WHERE id = $1", apiID).Scan(&name, &baseURL)
	if err != nil {
		http.Error(w, "API not found", http.StatusNotFound)
		return
	}

	if baseURL == "" {
		http.Error(w, "API has no base URL configured", http.StatusBadRequest)
		return
	}

	// Trigger health check
	if healthMonitor != nil {
		go healthMonitor.checkAPIHealth(apiID, name, baseURL)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Health check triggered",
		"api_id":  apiID,
	})
}
