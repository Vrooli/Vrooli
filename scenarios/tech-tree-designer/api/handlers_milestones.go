package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type milestonePayload struct {
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	MilestoneType           string   `json:"milestone_type"`
	CompletionPercentage    *float64 `json:"completion_percentage"`
	BusinessValueEstimate   *int64   `json:"business_value_estimate"`
	ConfidenceLevel         *float64 `json:"confidence_level"`
	EstimatedCompletionDate *string  `json:"estimated_completion_date"`
	TargetSectorIDs         []string `json:"target_sector_ids"`
	TargetStageIDs          []string `json:"target_stage_ids"`
}

func (p *milestonePayload) sanitize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.MilestoneType = strings.TrimSpace(p.MilestoneType)
}

func (p *milestonePayload) validate() error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.MilestoneType == "" {
		return errors.New("milestone_type is required")
	}
	if p.CompletionPercentage != nil {
		if *p.CompletionPercentage < 0 || *p.CompletionPercentage > 100 {
			return errors.New("completion_percentage must be between 0 and 100")
		}
	}
	if p.ConfidenceLevel != nil {
		if *p.ConfidenceLevel < 0 || *p.ConfidenceLevel > 1 {
			return errors.New("confidence_level must be between 0 and 1")
		}
	}
	if p.BusinessValueEstimate != nil && *p.BusinessValueEstimate < 0 {
		return errors.New("business_value_estimate cannot be negative")
	}
	if _, err := parseEstimatedDate(p.EstimatedCompletionDate); err != nil {
		return err
	}
	return nil
}

func parseEstimatedDate(input *string) (*time.Time, error) {
	if input == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*input)
	if trimmed == "" {
		return nil, nil
	}
	if ts, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return &ts, nil
	}
	if ts, err := time.Parse("2006-01-02", trimmed); err == nil {
		return &ts, nil
	}
	return nil, fmt.Errorf("invalid estimated_completion_date format")
}

func marshalIDList(values []string) json.RawMessage {
	cleaned := make([]string, 0, len(values))
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	data, _ := json.Marshal(cleaned)
	if len(data) == 0 {
		return json.RawMessage("[]")
	}
	return data
}

func scanStrategicMilestoneRow(scanner interface {
	Scan(dest ...interface{}) error
}) (StrategicMilestone, error) {
	var milestone StrategicMilestone
	err := scanner.Scan(
		&milestone.ID,
		&milestone.TreeID,
		&milestone.Name,
		&milestone.Description,
		&milestone.MilestoneType,
		&milestone.RequiredSectors,
		&milestone.RequiredStages,
		&milestone.CompletionPercentage,
		&milestone.EstimatedCompletionDate,
		&milestone.ConfidenceLevel,
		&milestone.BusinessValueEstimate,
		&milestone.CreatedAt,
		&milestone.UpdatedAt,
	)
	return milestone, err
}

func createStrategicMilestone(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	var payload milestonePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	payload.sanitize()
	if err := payload.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	milestone, err := persistMilestone(tree.ID, "", payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create milestone"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"milestone": milestone})
}

func updateStrategicMilestone(c *gin.Context) {
	milestoneID := strings.TrimSpace(c.Param("id"))
	if milestoneID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Milestone ID is required"})
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

	var payload milestonePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	payload.sanitize()
	if err := payload.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	milestone, err := persistMilestone(tree.ID, milestoneID, payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Milestone not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update milestone"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"milestone": milestone})
}

func persistMilestone(treeID, milestoneID string, payload milestonePayload) (StrategicMilestone, error) {
	estimated, err := parseEstimatedDate(payload.EstimatedCompletionDate)
	if err != nil {
		return StrategicMilestone{}, err
	}

	completion := 0.0
	if payload.CompletionPercentage != nil {
		completion = *payload.CompletionPercentage
	}

	confidence := 0.6
	if payload.ConfidenceLevel != nil {
		confidence = *payload.ConfidenceLevel
	}

	businessValue := int64(0)
	if payload.BusinessValueEstimate != nil {
		businessValue = *payload.BusinessValueEstimate
	}

	sectorsJSON := marshalIDList(payload.TargetSectorIDs)
	stagesJSON := marshalIDList(payload.TargetStageIDs)

	var row *sql.Row
	if milestoneID == "" {
		row = db.QueryRow(`
            INSERT INTO strategic_milestones (
                tree_id, name, description, milestone_type,
                required_sectors, required_stages, completion_percentage,
                estimated_completion_date, confidence_level, business_value_estimate
            ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
            RETURNING id, tree_id, name, description, milestone_type, required_sectors,
                      required_stages, completion_percentage, estimated_completion_date,
                      confidence_level, business_value_estimate, created_at, updated_at
        `, treeID, payload.Name, payload.Description, payload.MilestoneType,
			sectorsJSON, stagesJSON, completion, estimated, confidence, businessValue)
	} else {
		row = db.QueryRow(`
            UPDATE strategic_milestones
            SET name=$1, description=$2, milestone_type=$3,
                required_sectors=$4, required_stages=$5,
                completion_percentage=$6, estimated_completion_date=$7,
                confidence_level=$8, business_value_estimate=$9,
                updated_at=NOW()
            WHERE id=$10 AND tree_id=$11
            RETURNING id, tree_id, name, description, milestone_type, required_sectors,
                      required_stages, completion_percentage, estimated_completion_date,
                      confidence_level, business_value_estimate, created_at, updated_at
        `, payload.Name, payload.Description, payload.MilestoneType,
			sectorsJSON, stagesJSON, completion, estimated, confidence, businessValue, milestoneID, treeID)
	}

	milestone, err := scanStrategicMilestoneRow(row)
	if err != nil {
		return StrategicMilestone{}, err
	}
	return milestone, nil
}

func deleteStrategicMilestone(c *gin.Context) {
	milestoneID := strings.TrimSpace(c.Param("id"))
	if milestoneID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Milestone ID is required"})
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

	result, err := db.Exec(`DELETE FROM strategic_milestones WHERE id=$1 AND tree_id=$2`, milestoneID, tree.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete milestone"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete milestone"})
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Milestone not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
