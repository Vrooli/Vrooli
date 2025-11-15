package main

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Delegation functions for backward compatibility with existing code
func fetchStageByID(ctx context.Context, stageID string) (*ProgressionStage, string, error) {
	return stageService.FetchStageByID(ctx, stageID)
}

func getStagesForSector(ctx context.Context, sectorID string) ([]ProgressionStage, error) {
	return stageService.GetStagesForSector(ctx, sectorID)
}

// getStage retrieves a single stage by ID
func getStage(c *gin.Context) {
	stageID := c.Param("id")

	stage, _, err := fetchStageByID(c.Request.Context(), stageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Stage not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stage"})
		}
		return
	}

	mappings, err := getScenarioMappingsForStage(stageID)
	if err == nil {
		stage.ScenarioMappings = mappings
	}

	c.JSON(http.StatusOK, stage)
}

// createStageHandler creates a new progression stage
func createStageHandler(c *gin.Context) {
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
		SectorID           string   `json:"sector_id"`
		ParentStageID      *string  `json:"parent_stage_id"` // Optional: for creating child stages
		StageType          string   `json:"stage_type"`
		StageOrder         int      `json:"stage_order"`
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		ProgressPercentage *float64 `json:"progress_percentage"`
		PositionX          *float64 `json:"position_x"`
		PositionY          *float64 `json:"position_y"`
		Examples           []string `json:"examples"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sectorID := strings.TrimSpace(payload.SectorID)
	if sectorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sector_id is required"})
		return
	}

	ctx := c.Request.Context()
	sector, err := fetchSectorByID(ctx, sectorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sector"})
		return
	}
	if sector.TreeID != tree.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sector does not belong to selected tech tree"})
		return
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	progress := 0.0
	if payload.ProgressPercentage != nil {
		progress = math.Max(0, math.Min(100, *payload.ProgressPercentage))
	}

	posX := 0.0
	if payload.PositionX != nil {
		posX = *payload.PositionX
	}
	posY := 0.0
	if payload.PositionY != nil {
		posY = *payload.PositionY
	}

	stageParams := CreateStageParams{
		SectorID:           sectorID,
		ParentStageID:      payload.ParentStageID,
		StageType:          payload.StageType,
		StageOrder:         payload.StageOrder,
		Name:               name,
		Description:        payload.Description,
		ProgressPercentage: progress,
		PositionX:          posX,
		PositionY:          posY,
		Examples:           payload.Examples,
	}

	stage, err := stageService.CreateStage(ctx, stageParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stage"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"stage": stage, "tree": tree})
}

// updateStageHandler updates an existing stage
func updateStageHandler(c *gin.Context) {
	stageID := c.Param("id")
	if strings.TrimSpace(stageID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stage id is required"})
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

	_, stageTreeID, err := fetchStageByID(c.Request.Context(), stageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Stage not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stage"})
		}
		return
	}
	if stageTreeID != tree.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stage does not belong to selected tech tree"})
		return
	}

	var payload struct {
		SectorID           *string  `json:"sector_id"`
		StageType          *string  `json:"stage_type"`
		StageOrder         *int     `json:"stage_order"`
		Name               *string  `json:"name"`
		Description        *string  `json:"description"`
		ProgressPercentage *float64 `json:"progress_percentage"`
		PositionX          *float64 `json:"position_x"`
		PositionY          *float64 `json:"position_y"`
		Examples           []string `json:"examples"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Validate sector_id if provided
	if payload.SectorID != nil {
		sectorID := strings.TrimSpace(*payload.SectorID)
		if sectorID != "" {
			sector, err := fetchSectorByID(ctx, sectorID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Target sector does not exist"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load target sector"})
				}
				return
			}
			if sector.TreeID != tree.ID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Target sector does not belong to tree"})
				return
			}
		}
	}

	updateParams := UpdateStageParams{
		SectorID:           payload.SectorID,
		StageType:          payload.StageType,
		StageOrder:         payload.StageOrder,
		Name:               payload.Name,
		Description:        payload.Description,
		ProgressPercentage: payload.ProgressPercentage,
		PositionX:          payload.PositionX,
		PositionY:          payload.PositionY,
		Examples:           payload.Examples,
	}

	updatedStage, err := stageService.UpdateStage(ctx, stageID, updateParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stage": updatedStage, "tree": tree})
}

// deleteStageHandler deletes a stage
func deleteStageHandler(c *gin.Context) {
	stageID := c.Param("id")
	if strings.TrimSpace(stageID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stage id is required"})
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

	if err := stageService.DeleteStage(c.Request.Context(), stageID, tree.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Stage not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete stage"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// getStageChildren returns all direct children of a stage (lazy loading endpoint)
func getStageChildren(c *gin.Context) {
	stageID := c.Param("id")
	if strings.TrimSpace(stageID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stage id is required"})
		return
	}

	children, err := stageService.FetchStageChildren(c.Request.Context(), stageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stage children"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"children": children, "count": len(children)})
}
