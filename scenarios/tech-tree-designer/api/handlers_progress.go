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

// Update stage maturity
func updateStageMaturity(c *gin.Context) {
	stageID := c.Param("id")

	var request struct {
		Maturity  string `json:"maturity" binding:"required"`
		ChangedBy string `json:"changed_by,omitempty"`
		Notes     string `json:"notes,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate maturity value
	validMaturityLevels := map[string]bool{
		"planned":  true,
		"building": true,
		"live":     true,
		"scaled":   true,
	}
	if !validMaturityLevels[request.Maturity] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Invalid maturity level",
			"valid_levels":   []string{"planned", "building", "live", "scaled"},
			"provided_value": request.Maturity,
		})
		return
	}

	// Get current maturity for audit trail
	var oldMaturity string
	err := db.QueryRowContext(c.Request.Context(), `
		SELECT maturity FROM progression_stages WHERE id = $1
	`, stageID).Scan(&oldMaturity)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Stage not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stage"})
		}
		return
	}

	// If maturity hasn't changed, return early
	if oldMaturity == request.Maturity {
		c.JSON(http.StatusOK, gin.H{
			"message":      "Maturity unchanged",
			"stage_id":     stageID,
			"maturity":     request.Maturity,
			"no_change":    true,
		})
		return
	}

	// Begin transaction for atomic update + audit
	tx, err := db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	// Update stage maturity
	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE progression_stages
		SET maturity = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, request.Maturity, stageID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update maturity"})
		return
	}

	// Insert audit event
	changedBy := request.ChangedBy
	if changedBy == "" {
		changedBy = "system"
	}

	eventID := uuid.New().String()
	_, err = tx.ExecContext(c.Request.Context(), `
		INSERT INTO maturity_events (id, stage_id, old_maturity, new_maturity, changed_by, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, eventID, stageID, oldMaturity, request.Maturity, changedBy, request.Notes)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record maturity event"})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit maturity update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Stage maturity updated successfully",
		"stage_id":     stageID,
		"old_maturity": oldMaturity,
		"new_maturity": request.Maturity,
		"event_id":     eventID,
	})
}

// Get maturity events for a stage
func getStageMaturityEvents(c *gin.Context) {
	stageID := c.Param("id")

	rows, err := db.QueryContext(c.Request.Context(), `
		SELECT id, old_maturity, new_maturity, changed_at, changed_by, notes
		FROM maturity_events
		WHERE stage_id = $1
		ORDER BY changed_at DESC
		LIMIT 50
	`, stageID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch maturity events"})
		return
	}
	defer rows.Close()

	type MaturityEvent struct {
		ID          string    `json:"id"`
		OldMaturity *string   `json:"old_maturity"`
		NewMaturity string    `json:"new_maturity"`
		ChangedAt   time.Time `json:"changed_at"`
		ChangedBy   string    `json:"changed_by"`
		Notes       string    `json:"notes"`
	}

	var events []MaturityEvent
	for rows.Next() {
		var event MaturityEvent
		err := rows.Scan(&event.ID, &event.OldMaturity, &event.NewMaturity,
			&event.ChangedAt, &event.ChangedBy, &event.Notes)
		if err != nil {
			continue
		}
		events = append(events, event)
	}

	c.JSON(http.StatusOK, gin.H{
		"stage_id": stageID,
		"events":   events,
		"count":    len(events),
	})
}
