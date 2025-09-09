package handlers

import (
	"context"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

// DockerHandler handles Docker-related endpoints
type DockerHandler struct {
	docker *client.Client
}

// NewDockerHandler creates a new Docker handler
func NewDockerHandler(docker *client.Client) *DockerHandler {
	return &DockerHandler{
		docker: docker,
	}
}

// GetDockerInfo returns Docker daemon information
func (h *DockerHandler) GetDockerInfo(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Docker not available",
		})
		return
	}

	ctx := context.Background()
	info, err := h.docker.Info(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get Docker info",
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

// GetContainers returns list of Docker containers
func (h *DockerHandler) GetContainers(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Docker not available",
		})
		return
	}

	ctx := context.Background()
	containers, err := h.docker.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list containers",
		})
		return
	}

	c.JSON(http.StatusOK, containers)
}