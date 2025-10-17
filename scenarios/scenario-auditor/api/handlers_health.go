package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// getHealthMetricsHandler returns detailed health metrics for a specific scenario
func getHealthMetricsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")
	logger.Info(fmt.Sprintf("Getting health metrics for scenario: %s", scenarioName))

	// Get scenario health details
	var scenarioID string
	var lastScanned *time.Time
	err := db.QueryRow("SELECT id, last_scanned FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID, &lastScanned)
	if err != nil {
		HTTPError(w, "Scenario not found", http.StatusNotFound, err)
		return
	}

	// Count vulnerabilities by severity
	var criticalCount, highCount, mediumCount, lowCount int
	db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN severity = 'CRITICAL' THEN 1 END),
			COUNT(CASE WHEN severity = 'HIGH' THEN 1 END),
			COUNT(CASE WHEN severity = 'MEDIUM' THEN 1 END),
			COUNT(CASE WHEN severity = 'LOW' THEN 1 END)
		FROM vulnerability_scans 
		WHERE scenario_id = $1 AND status = 'open'
	`, scenarioID).Scan(&criticalCount, &highCount, &mediumCount, &lowCount)

	// Get performance metrics
	var avgResponseTime, errorRate float64
	db.QueryRow(`
		SELECT 
			AVG(response_time_ms),
			AVG(error_rate)
		FROM performance_metrics 
		WHERE scenario_id = $1 AND created_at > NOW() - INTERVAL '1 hour'
	`, scenarioID).Scan(&avgResponseTime, &errorRate)

	// Build metrics response
	metrics := map[string]interface{}{
		"scenario":     scenarioName,
		"timestamp":    time.Now().UTC(),
		"health_score": calculateHealthScore(criticalCount, highCount, mediumCount, lowCount),
		"vulnerabilities": map[string]interface{}{
			"critical": criticalCount,
			"high":     highCount,
			"medium":   mediumCount,
			"low":      lowCount,
			"total":    criticalCount + highCount + mediumCount + lowCount,
		},
		"performance": map[string]interface{}{
			"avg_response_time_ms": avgResponseTime,
			"error_rate":           errorRate,
		},
		"last_scanned": lastScanned,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
