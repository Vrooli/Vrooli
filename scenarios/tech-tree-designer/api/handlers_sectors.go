package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Delegation functions for backward compatibility with existing code
func fetchSectorByID(ctx context.Context, sectorID string) (*Sector, error) {
	return sectorService.FetchSectorByID(ctx, sectorID)
}

func fetchSectorsWithStages(ctx context.Context, treeID string) ([]Sector, error) {
	return sectorService.FetchSectorsWithStages(ctx, treeID)
}

func fetchStageByID(ctx context.Context, stageID string) (*ProgressionStage, string, error) {
	return sectorService.FetchStageByID(ctx, stageID)
}

func getStagesForSector(sectorID string) ([]ProgressionStage, error) {
	return sectorService.GetStagesForSector(nil, sectorID)
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

	stages, err := getStagesForSector(sectorID)
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

	params := CreateSectorParams{
		TreeID:      tree.ID,
		Name:        payload.Name,
		Category:    payload.Category,
		Description: payload.Description,
		Color:       payload.Color,
		PositionX:   payload.PositionX,
		PositionY:   payload.PositionY,
	}

	sector, err := sectorService.CreateSector(c.Request.Context(), params)
	if err != nil {
		if strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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

	sector, err := fetchSectorByID(c.Request.Context(), sectorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sector"})
		}
		return
	}
	if sector.TreeID != tree.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
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

	updated, err := sectorService.UpdateSector(c.Request.Context(), sectorID, params)
	if err != nil {
		if strings.Contains(err.Error(), "no updatable fields") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sector"})
		return
	}

	stages, err := getStagesForSector(sectorID)
	if err == nil {
		updated.Stages = stages
	}

	c.JSON(http.StatusOK, gin.H{"sector": updated, "tree": tree})
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

	err = sectorService.DeleteSector(c.Request.Context(), sectorID, tree.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sector"})
		return
	}

	c.Status(http.StatusNoContent)
}

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

	stageType := normalizeStageType(payload.StageType)
	stageOrder := payload.StageOrder
	if stageOrder <= 0 {
		if stageOrder, err = computeNextStageOrder(ctx, sectorID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute stage order"})
			return
		}
	}

	examplesJSON, err := encodeExamplesArray(payload.Examples)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid examples payload"})
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

	stageID := uuid.New().String()
	if _, err := db.ExecContext(ctx, `
        INSERT INTO progression_stages (id, sector_id, parent_stage_id, stage_type, stage_order, name, description,
            progress_percentage, examples, position_x, position_y)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `, stageID, sectorID, payload.ParentStageID, stageType, stageOrder, name, payload.Description, progress, examplesJSON, posX, posY); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stage"})
		return
	}

	stage, _, err := fetchStageByID(ctx, stageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stage created but retrieval failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"stage": stage, "tree": tree})
}

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
	updates := []string{}
	args := []interface{}{}
	idx := 1

	appendClause := func(value interface{}, template string) {
		updates = append(updates, fmt.Sprintf(template, idx))
		args = append(args, value)
		idx++
	}

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
			appendClause(sectorID, "sector_id = $%d")
		}
	}

	if payload.StageType != nil {
		appendClause(normalizeStageType(*payload.StageType), "stage_type = $%d")
	}
	if payload.StageOrder != nil {
		appendClause(*payload.StageOrder, "stage_order = $%d")
	}
	if payload.Name != nil {
		appendClause(strings.TrimSpace(*payload.Name), "name = $%d")
	}
	if payload.Description != nil {
		appendClause(strings.TrimSpace(*payload.Description), "description = $%d")
	}
	if payload.ProgressPercentage != nil {
		progress := math.Max(0, math.Min(100, *payload.ProgressPercentage))
		appendClause(progress, "progress_percentage = $%d")
	}
	if payload.PositionX != nil {
		appendClause(*payload.PositionX, "position_x = $%d")
	}
	if payload.PositionY != nil {
		appendClause(*payload.PositionY, "position_y = $%d")
	}
	if payload.Examples != nil {
		examplesJSON, err := encodeExamplesArray(payload.Examples)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid examples payload"})
			return
		}
		appendClause(examplesJSON, "examples = $%d")
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No updatable fields supplied"})
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, stageID)

	query := fmt.Sprintf("UPDATE progression_stages SET %s WHERE id = $%d", strings.Join(updates, ", "), idx)
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stage"})
		return
	}

	updatedStage, _, err := fetchStageByID(ctx, stageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stage updated but retrieval failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stage": updatedStage, "tree": tree})
}

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

	result, err := db.ExecContext(c.Request.Context(), `
        DELETE FROM progression_stages
        WHERE id = $1
          AND sector_id IN (
                SELECT id FROM sectors WHERE tree_id = $2
          )
    `, stageID, tree.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete stage"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stage not found"})
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

	children, err := sectorService.FetchStageChildren(c.Request.Context(), stageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stage children"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"children": children, "count": len(children)})
}
