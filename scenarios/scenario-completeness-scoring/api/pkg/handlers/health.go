package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"scenario-completeness-scoring/pkg/health"

	"github.com/gorilla/mux"
)

// HandleHealth returns basic service health status
func (ctx *Context) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "Scenario Completeness Scoring API",
		"version":   "1.0.0",
		"readiness": true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetCollectorHealth returns health status of all collectors
// [REQ:SCS-HEALTH-001] Health status API endpoint
func (ctx *Context) HandleGetCollectorHealth(w http.ResponseWriter, r *http.Request) {
	overall := ctx.HealthTracker.GetOverallHealth()

	response := map[string]interface{}{
		"status":     overall.Status,
		"collectors": overall.Collectors,
		"summary": map[string]int{
			"healthy":  overall.Healthy,
			"degraded": overall.Degraded,
			"failed":   overall.Failed,
			"total":    overall.Total,
		},
		"checked_at": overall.CheckedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleTestCollector tests a specific collector on demand
// [REQ:SCS-HEALTH-003] Collector test endpoint
func (ctx *Context) HandleTestCollector(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectorName := vars["name"]

	// Define test functions for each collector type
	var testFn health.CollectorTestFunc
	switch collectorName {
	case "requirements":
		testFn = func() error {
			_, err := ctx.Collector.Collect("scenario-completeness-scoring")
			return err
		}
	case "tests", "ui", "service":
		testFn = func() error {
			// These are part of the overall collect, so test via a sample scenario
			_, err := ctx.Collector.Collect("scenario-completeness-scoring")
			return err
		}
	default:
		http.Error(w, fmt.Sprintf("Unknown collector: %s", collectorName), http.StatusNotFound)
		return
	}

	result := ctx.HealthTracker.TestCollector(collectorName, testFn)

	// Also record in circuit breaker
	cb := ctx.CBRegistry.Get(collectorName)
	if result.Success {
		cb.RecordSuccess()
	} else {
		cb.RecordFailure()
	}

	response := map[string]interface{}{
		"name":        result.Name,
		"success":     result.Success,
		"duration_ms": result.Duration.Milliseconds(),
		"status":      result.Status,
	}
	if result.Error != "" {
		response["error"] = result.Error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetCircuitBreakers returns status of all circuit breakers
// [REQ:SCS-CB-004] View auto-disabled components
func (ctx *Context) HandleGetCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	statuses := ctx.CBRegistry.ListStatus()
	stats := ctx.CBRegistry.Stats()
	openBreakers := ctx.CBRegistry.OpenBreakers()

	response := map[string]interface{}{
		"breakers": statuses,
		"stats": map[string]int{
			"total":     stats.Total,
			"closed":    stats.Closed,
			"open":      stats.Open,
			"half_open": stats.HalfOpen,
		},
		"tripped": openBreakers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleResetAllCircuitBreakers resets all circuit breakers
// [REQ:SCS-CB-004] Reset all components
func (ctx *Context) HandleResetAllCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	count := ctx.CBRegistry.ResetAll()

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Reset %d circuit breakers", count),
		"count":   count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleResetCircuitBreaker resets a specific circuit breaker
// [REQ:SCS-CB-004] Reset specific component
func (ctx *Context) HandleResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectorName := vars["collector"]

	success := ctx.CBRegistry.Reset(collectorName)
	if !success {
		http.Error(w, fmt.Sprintf("Circuit breaker not found: %s", collectorName), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"collector": collectorName,
		"message":   fmt.Sprintf("Circuit breaker '%s' has been reset", collectorName),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
