package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"

	"scenario-authenticator/db"
)

type healthError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Category  string                 `json:"category"`
	Retryable bool                   `json:"retryable"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type dependencyStatus struct {
	Connected bool         `json:"connected"`
	LatencyMs *float64     `json:"latency_ms,omitempty"`
	Error     *healthError `json:"error,omitempty"`
}

type healthDependencies struct {
	Database dependencyStatus `json:"database"`
	Redis    dependencyStatus `json:"redis"`
}

type healthMetrics struct {
	Goroutines int `json:"goroutines"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status       string             `json:"status"`
	Service      string             `json:"service"`
	Timestamp    string             `json:"timestamp"`
	Readiness    bool               `json:"readiness"`
	Version      string             `json:"version"`
	Dependencies healthDependencies `json:"dependencies"`
	Metrics      *healthMetrics     `json:"metrics,omitempty"`
}

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		return
	}

	ctx := r.Context()

	dbStatus := checkDatabase(ctx)
	redisStatus := checkRedis(ctx)

	status := "healthy"
	if !dbStatus.Connected || !redisStatus.Connected {
		status = "degraded"
	}
	if !dbStatus.Connected && !redisStatus.Connected {
		status = "unhealthy"
	}
	readiness := dbStatus.Connected && redisStatus.Connected

	version := os.Getenv("API_VERSION")
	if version == "" {
		version = "1.0.0"
	}

	response := HealthResponse{
		Status:    status,
		Service:   "scenario-authenticator-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Readiness: readiness,
		Version:   version,
		Dependencies: healthDependencies{
			Database: dbStatus,
			Redis:    redisStatus,
		},
		Metrics: &healthMetrics{
			Goroutines: runtime.NumGoroutine(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(response)
}

func checkDatabase(ctx context.Context) dependencyStatus {
	status := dependencyStatus{}
	if db.DB == nil {
		status.Error = &healthError{
			Code:      "DB_CONNECTION_UNINITIALIZED",
			Message:   "database connection pool is not initialized",
			Category:  "configuration",
			Retryable: false,
		}
		return status
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	start := time.Now()
	if err := db.DB.PingContext(pingCtx); err != nil {
		status.Error = &healthError{
			Code:      "DB_PING_FAILED",
			Message:   err.Error(),
			Category:  "resource",
			Retryable: true,
			Details: map[string]interface{}{
				"operation": "ping",
			},
		}
		return status
	}

	elapsed := time.Since(start)
	latency := float64(elapsed) / float64(time.Millisecond)
	status.Connected = true
	status.LatencyMs = &latency
	return status
}

func checkRedis(ctx context.Context) dependencyStatus {
	status := dependencyStatus{}
	if db.RedisClient == nil {
		status.Error = &healthError{
			Code:      "REDIS_CONNECTION_UNINITIALIZED",
			Message:   "redis client is not initialized",
			Category:  "configuration",
			Retryable: false,
		}
		return status
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	start := time.Now()
	if err := db.RedisClient.Ping(pingCtx).Err(); err != nil {
		status.Error = &healthError{
			Code:      "REDIS_PING_FAILED",
			Message:   err.Error(),
			Category:  "resource",
			Retryable: true,
			Details: map[string]interface{}{
				"operation": "ping",
			},
		}
		return status
	}

	elapsed := time.Since(start)
	latency := float64(elapsed) / float64(time.Millisecond)
	status.Connected = true
	status.LatencyMs = &latency
	return status
}
