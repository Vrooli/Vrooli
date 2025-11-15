package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Delegation functions for backward compatibility with existing code
func fetchSectorByID(ctx context.Context, sectorID string) (*Sector, error) {
	return sectorService.FetchSectorByID(ctx, sectorID)
}

func fetchSectorsWithStages(ctx context.Context, treeID string) ([]Sector, error) {
	return sectorService.FetchSectorsWithStages(ctx, treeID, stageService)
}

func getSectors(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	sectors, err := fetchSectorsWithStages(c.Request.Context(), tree.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sectors"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sectors": sectors, "tree": tree})
}

func getSector(c *gin.Context) {
	sectorID := c.Param("id")
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	sector, err := fetchSectorByID(c.Request.Context(), sectorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sector"})
		}
		return
	}

	if sector.TreeID != tree.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
		return
	}

	stages, err := getStagesForSector(c.Request.Context(), sectorID)
	if err == nil {
		sector.Stages = stages
	}

	c.JSON(http.StatusOK, gin.H{"sector": sector, "tree": tree})
}

func createSectorHandler(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	var payload struct {
		Name        string  `json:"name"`
		Category    string  `json:"category"`
		Description string  `json:"description"`
		Color       string  `json:"color"`
		PositionX   float64 `json:"position_x"`
		PositionY   float64 `json:"position_y"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	params := CreateSectorParams{
		TreeID:      tree.ID,
		Name:        name,
		Category:    payload.Category,
		Description: payload.Description,
		Color:       payload.Color,
		PositionX:   payload.PositionX,
		PositionY:   payload.PositionY,
	}

	sector, err := sectorService.CreateSector(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sector"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"sector": sector, "tree": tree})
}

func updateSectorHandler(c *gin.Context) {
	sectorID := c.Param("id")
	if strings.TrimSpace(sectorID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sector id is required"})
		return
	}

	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	ctx := c.Request.Context()
	existing, err := fetchSectorByID(ctx, sectorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sector"})
		}
		return
	}

	if existing.TreeID != tree.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sector does not belong to selected tech tree"})
		return
	}

	var payload struct {
		Name        *string  `json:"name"`
		Category    *string  `json:"category"`
		Description *string  `json:"description"`
		Progress    *float64 `json:"progress_percentage"`
		PositionX   *float64 `json:"position_x"`
		PositionY   *float64 `json:"position_y"`
		Color       *string  `json:"color"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := UpdateSectorParams{
		Name:        payload.Name,
		Category:    payload.Category,
		Description: payload.Description,
		Progress:    payload.Progress,
		PositionX:   payload.PositionX,
		PositionY:   payload.PositionY,
		Color:       payload.Color,
	}

	sector, err := sectorService.UpdateSector(ctx, sectorID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sector"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sector": sector, "tree": tree})
}

func deleteSectorHandler(c *gin.Context) {
	sectorID := c.Param("id")
	if strings.TrimSpace(sectorID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sector id is required"})
		return
	}

	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	if err := sectorService.DeleteSector(c.Request.Context(), sectorID, tree.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sector"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
