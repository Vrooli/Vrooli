package handlers

import (
	"encoding/json"
	"net/http"
	"scenario-authenticator/db"
)

// HealthResponse represents health check response
type HealthResponse struct {
	Status   string `json:"status"`
	Database bool   `json:"database"`
	Redis    bool   `json:"redis"`
	Version  string `json:"version"`
}

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	dbHealthy := false
	if db.DB != nil {
		err := db.DB.Ping()
		dbHealthy = (err == nil)
	}
	
	// Check Redis connection
	redisHealthy := false
	if db.RedisClient != nil {
		err := db.RedisClient.Ping(db.Ctx).Err()
		redisHealthy = (err == nil)
	}
	
	status := "healthy"
	if !dbHealthy || !redisHealthy {
		status = "degraded"
	}
	
	response := HealthResponse{
		Status:   status,
		Database: dbHealthy,
		Redis:    redisHealthy,
		Version:  "1.0.0",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}