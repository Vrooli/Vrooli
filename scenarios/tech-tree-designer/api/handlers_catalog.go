package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func getScenarioCatalogHandler(c *gin.Context) {
	if catalogManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Scenario catalog unavailable"})
		return
	}

	scenarios, edges, hidden, lastSynced := catalogManager.Snapshot()
	c.JSON(http.StatusOK, gin.H{
		"scenarios":   scenarios,
		"edges":       edges,
		"hidden":      hidden,
		"last_synced": lastSynced,
	})
}

func refreshScenarioCatalogHandler(c *gin.Context) {
	if catalogManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Scenario catalog unavailable"})
		return
	}

	if err := catalogManager.Refresh(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh scenario catalog"})
		return
	}

	_, _, _, lastSynced := catalogManager.Snapshot()
	c.JSON(http.StatusOK, gin.H{"message": "Scenario catalog refreshed", "last_synced": lastSynced})
}

func updateScenarioVisibilityHandler(c *gin.Context) {
	if catalogManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Scenario catalog unavailable"})
		return
	}

	var payload struct {
		Scenario string `json:"scenario"`
		Hidden   bool   `json:"hidden"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if strings.TrimSpace(payload.Scenario) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scenario is required"})
		return
	}

	if err := catalogManager.UpdateVisibility(payload.Scenario, payload.Hidden); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scenarios, _, hidden, lastSynced := catalogManager.Snapshot()
	c.JSON(http.StatusOK, gin.H{
		"message":     "Scenario visibility updated",
		"scenarios":   scenarios,
		"hidden":      hidden,
		"last_synced": lastSynced,
	})
}
