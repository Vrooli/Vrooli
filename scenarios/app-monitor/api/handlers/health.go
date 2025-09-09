package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/docker/docker/client"
	"database/sql"
)

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

// HealthStatus represents the health status response
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// Check performs a comprehensive health check
func (h *HealthHandler) Check(c *gin.Context) {
	ctx := context.Background()
	services := make(map[string]string)

	// Check database
	if h.db == nil {
		services["database"] = "disabled"
	} else if err := h.db.Ping(); err != nil {
		services["database"] = "error"
	} else {
		services["database"] = "ok"
	}

	// Check Redis
	if h.redis == nil {
		services["redis"] = "disabled"
	} else if err := h.redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "error"
	} else {
		services["redis"] = "ok"
	}

	// Check Docker
	if h.docker == nil {
		services["docker"] = "disabled"
	} else if _, err := h.docker.Ping(ctx); err != nil {
		services["docker"] = "error"
	} else {
		services["docker"] = "ok"
	}

	// Determine overall status
	status := "healthy"
	for _, service := range services {
		if service == "error" {
			status = "unhealthy"
			break
		}
	}

	c.JSON(http.StatusOK, HealthStatus{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
		Services:  services,
	})
}

// APIHealth returns simple API health status
func (h *HealthHandler) APIHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"api":       "app-monitor",
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}