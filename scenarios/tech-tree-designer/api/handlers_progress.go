package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func getScenarioMappingsForStage(stageID string) ([]ScenarioMapping, error) {
	rows, err := db.Query(`
		SELECT id, scenario_name, stage_id, contribution_weight, completion_status,
			   priority, estimated_impact, last_status_check, notes, created_at, updated_at
		FROM scenario_mappings
		WHERE stage_id = $1
		ORDER BY priority, estimated_impact DESC
	`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []ScenarioMapping
	for rows.Next() {
		var mapping ScenarioMapping
		err := rows.Scan(&mapping.ID, &mapping.ScenarioName, &mapping.StageID,
			&mapping.ContributionWeight, &mapping.CompletionStatus, &mapping.Priority,
			&mapping.EstimatedImpact, &mapping.LastStatusCheck, &mapping.Notes,
			&mapping.CreatedAt, &mapping.UpdatedAt)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// Get all scenario mappings
func getScenarioMappings(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	filters := []string{"s.tree_id = $1"}
	args := []interface{}{tree.ID}
	argIndex := 2

	if stageFilter := strings.TrimSpace(c.Query("stage_id")); stageFilter != "" {
		filters = append(filters, fmt.Sprintf("sm.stage_id = $%d", argIndex))
		args = append(args, stageFilter)
		argIndex++
	}

	if scenarioFilter := strings.TrimSpace(c.Query("scenario")); scenarioFilter != "" {
		filters = append(filters, fmt.Sprintf("sm.scenario_name = $%d", argIndex))
		args = append(args, scenarioFilter)
		argIndex++
	}

	query := `
		SELECT sm.id, sm.scenario_name, sm.stage_id, sm.contribution_weight,
		       sm.completion_status, sm.priority, sm.estimated_impact,
		       sm.last_status_check, sm.notes, sm.created_at, sm.updated_at,
		       ps.name as stage_name, s.name as sector_name
		FROM scenario_mappings sm
		JOIN progression_stages ps ON sm.stage_id = ps.id
		JOIN sectors s ON ps.sector_id = s.id
	`
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " ORDER BY sm.priority, sm.estimated_impact DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scenario mappings"})
		return
	}
	defer rows.Close()

	var mappings []gin.H
	for rows.Next() {
		var mapping ScenarioMapping
		var stageName, sectorName string
		err := rows.Scan(&mapping.ID, &mapping.ScenarioName, &mapping.StageID,
			&mapping.ContributionWeight, &mapping.CompletionStatus, &mapping.Priority,
			&mapping.EstimatedImpact, &mapping.LastStatusCheck, &mapping.Notes,
			&mapping.CreatedAt, &mapping.UpdatedAt, &stageName, &sectorName)
		if err != nil {
			continue
		}

		mappings = append(mappings, gin.H{
			"mapping":     mapping,
			"stage_name":  stageName,
			"sector_name": sectorName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"scenario_mappings": mappings, "tree": tree})
}

// Update scenario status
func updateScenarioStatus(c *gin.Context) {
	scenarioName := c.Param("scenario")
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	var request struct {
		CompletionStatus string  `json:"completion_status"`
		Notes            string  `json:"notes"`
		EstimatedImpact  float64 `json:"estimated_impact,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update scenario mapping
	result, err := db.Exec(`
		UPDATE scenario_mappings 
		SET completion_status = $1, notes = $2, last_status_check = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP,
		    estimated_impact = CASE WHEN $3 > 0 THEN $3 ELSE estimated_impact END
		WHERE scenario_name = $4
		  AND stage_id IN (
			SELECT ps.id
			FROM progression_stages ps
			JOIN sectors s ON ps.sector_id = s.id
			WHERE s.tree_id = $5
		  )
	`, request.CompletionStatus, request.Notes, request.EstimatedImpact, scenarioName, tree.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scenario status"})
		return
	}

	if rowsAffected, rowsErr := result.RowsAffected(); rowsErr == nil && rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scenario mapping not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Scenario status updated successfully",
		"scenario": scenarioName,
		"status":   request.CompletionStatus,
		"tree":     tree,
	})
}

// Strategic analysis endpoint

func updateScenarioMapping(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	var mapping ScenarioMapping
	if err := c.ShouldBindJSON(&mapping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var belongs bool
	checkErr := db.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM progression_stages ps
			JOIN sectors s ON ps.sector_id = s.id
			WHERE ps.id = $1 AND s.tree_id = $2
		)
	`, mapping.StageID, tree.ID).Scan(&belongs)
	if checkErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate stage"})
		return
	}
	if !belongs {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stage does not belong to the selected tech tree"})
		return
	}

	// Set ID and timestamps
	mapping.ID = uuid.New().String()
	mapping.CreatedAt = time.Now()
	mapping.UpdatedAt = time.Now()
	mapping.LastStatusCheck = time.Now()

	_, err = db.Exec(`
		INSERT INTO scenario_mappings (id, scenario_name, stage_id, contribution_weight,
			completion_status, priority, estimated_impact, last_status_check, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (scenario_name, stage_id) DO UPDATE SET
			contribution_weight = $4, completion_status = $5, priority = $6,
			estimated_impact = $7, last_status_check = $8, notes = $9, updated_at = $11
	`, mapping.ID, mapping.ScenarioName, mapping.StageID, mapping.ContributionWeight,
		mapping.CompletionStatus, mapping.Priority, mapping.EstimatedImpact,
		mapping.LastStatusCheck, mapping.Notes, mapping.CreatedAt, mapping.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scenario mapping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scenario mapping updated successfully",
		"mapping": mapping,
		"tree":    tree,
	})
}

// Delete scenario mapping (unlink scenario from stage)
func deleteScenarioMapping(c *gin.Context) {
	mappingID := c.Param("id")
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	// Verify the mapping belongs to this tree before deleting
	var stageID string
	err = db.QueryRow(`
		SELECT sm.stage_id
		FROM scenario_mappings sm
		JOIN progression_stages ps ON sm.stage_id = ps.id
		JOIN sectors s ON ps.sector_id = s.id
		WHERE sm.id = $1 AND s.tree_id = $2
	`, mappingID, tree.ID).Scan(&stageID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scenario mapping not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify scenario mapping"})
		}
		return
	}

	// Delete the mapping
	_, err = db.Exec(`DELETE FROM scenario_mappings WHERE id = $1`, mappingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete scenario mapping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scenario mapping deleted successfully",
		"id":      mappingID,
		"tree":    tree,
	})
}
