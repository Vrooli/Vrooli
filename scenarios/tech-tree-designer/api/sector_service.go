package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
)

// SectorService encapsulates business logic for sector and stage operations.
// Separates domain logic from HTTP handling concerns.
type SectorService struct {
	db *sql.DB
}

// NewSectorService creates a new sector service instance.
func NewSectorService(database *sql.DB) *SectorService {
	return &SectorService{db: database}
}

// CreateSectorParams contains parameters for creating a new sector.
type CreateSectorParams struct {
	TreeID      string
	Name        string
	Category    string
	Description string
	Color       string
	PositionX   float64
	PositionY   float64
}

// UpdateSectorParams contains optional fields for updating a sector.
type UpdateSectorParams struct {
	Name        *string
	Category    *string
	Description *string
	Progress    *float64
	PositionX   *float64
	PositionY   *float64
	Color       *string
}

// CreateStageParams contains parameters for creating a new stage.
type CreateStageParams struct {
	SectorID           string
	ParentStageID      *string
	StageType          string
	StageOrder         int
	Name               string
	Description        string
	ProgressPercentage float64
	PositionX          float64
	PositionY          float64
	Examples           []string
}

// UpdateStageParams contains optional fields for updating a stage.
type UpdateStageParams struct {
	SectorID           *string
	StageType          *string
	StageOrder         *int
	Name               *string
	Description        *string
	ProgressPercentage *float64
	PositionX          *float64
	PositionY          *float64
	Examples           []string
}

// FetchSectorByID retrieves a sector by ID.
func (s *SectorService) FetchSectorByID(ctx context.Context, sectorID string) (*Sector, error) {
	if s.db == nil {
		return nil, errors.New("database connection is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var sector Sector
	err := s.db.QueryRowContext(ctx, `
        SELECT id, tree_id, name, category, description, progress_percentage,
               position_x, position_y, color, created_at, updated_at
        FROM sectors
        WHERE id = $1
    `, sectorID).Scan(
		&sector.ID,
		&sector.TreeID,
		&sector.Name,
		&sector.Category,
		&sector.Description,
		&sector.ProgressPercentage,
		&sector.PositionX,
		&sector.PositionY,
		&sector.Color,
		&sector.CreatedAt,
		&sector.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &sector, nil
}

// FetchSectorsWithStages retrieves all sectors for a tree with their stages.
func (s *SectorService) FetchSectorsWithStages(ctx context.Context, treeID string) ([]Sector, error) {
	if s.db == nil {
		return nil, errors.New("database connection is not initialized")
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, tree_id, name, category, description, progress_percentage,
               position_x, position_y, color, created_at, updated_at
        FROM sectors
        WHERE tree_id = $1
        ORDER BY category, name
    `, treeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sectors []Sector
	for rows.Next() {
		var sector Sector
		err := rows.Scan(&sector.ID, &sector.TreeID, &sector.Name, &sector.Category,
			&sector.Description, &sector.ProgressPercentage, &sector.PositionX,
			&sector.PositionY, &sector.Color, &sector.CreatedAt, &sector.UpdatedAt)
		if err != nil {
			return nil, err
		}

		stages, stageErr := s.GetStagesForSector(ctx, sector.ID)
		if stageErr == nil {
			sector.Stages = stages
		}

		sectors = append(sectors, sector)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sectors, nil
}

// CreateSector creates a new sector in the specified tech tree.
func (s *SectorService) CreateSector(ctx context.Context, params CreateSectorParams) (*Sector, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}

	category := strings.TrimSpace(params.Category)
	if category == "" {
		category = "software"
	}

	color := strings.TrimSpace(params.Color)
	if color == "" {
		color = "#3B82F6"
	}

	sectorID := uuid.New().String()
	if _, err := s.db.ExecContext(ctx, `
        INSERT INTO sectors (id, tree_id, name, category, description,
            progress_percentage, position_x, position_y, color)
        VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8)
    `, sectorID, params.TreeID, name, category, strings.TrimSpace(params.Description), params.PositionX, params.PositionY, color); err != nil {
		return nil, err
	}

	return s.FetchSectorByID(ctx, sectorID)
}

// UpdateSector updates an existing sector.
func (s *SectorService) UpdateSector(ctx context.Context, sectorID string, params UpdateSectorParams) (*Sector, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	updates := []string{}
	args := []interface{}{}
	idx := 1

	appendClause := func(value interface{}, template string) {
		updates = append(updates, fmt.Sprintf(template, idx))
		args = append(args, value)
		idx++
	}

	if params.Name != nil {
		if trimmed := strings.TrimSpace(*params.Name); trimmed != "" {
			appendClause(trimmed, "name = $%d")
		}
	}
	if params.Category != nil {
		if trimmed := strings.TrimSpace(*params.Category); trimmed != "" {
			appendClause(trimmed, "category = $%d")
		}
	}
	if params.Description != nil {
		appendClause(strings.TrimSpace(*params.Description), "description = $%d")
	}
	if params.Progress != nil {
		progress := math.Max(0, math.Min(100, *params.Progress))
		appendClause(progress, "progress_percentage = $%d")
	}
	if params.PositionX != nil {
		appendClause(*params.PositionX, "position_x = $%d")
	}
	if params.PositionY != nil {
		appendClause(*params.PositionY, "position_y = $%d")
	}
	if params.Color != nil {
		appendClause(strings.TrimSpace(*params.Color), "color = $%d")
	}

	if len(updates) == 0 {
		return nil, errors.New("no updatable fields supplied")
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, sectorID)

	query := fmt.Sprintf("UPDATE sectors SET %s WHERE id = $%d", strings.Join(updates, ", "), idx)
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}

	return s.FetchSectorByID(ctx, sectorID)
}

// DeleteSector deletes a sector from a specific tree.
func (s *SectorService) DeleteSector(ctx context.Context, sectorID, treeID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := s.db.ExecContext(ctx, `
        DELETE FROM sectors
        WHERE id = $1 AND tree_id = $2
    `, sectorID, treeID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// FetchStageByID retrieves a stage by ID along with its tree ID.
func (s *SectorService) FetchStageByID(ctx context.Context, stageID string) (*ProgressionStage, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var stage ProgressionStage
	var sectorID, treeID string
	var examples json.RawMessage
	var parentID sql.NullString
	err := s.db.QueryRowContext(ctx, `
        SELECT ps.id, ps.sector_id, s.tree_id, ps.parent_stage_id, ps.stage_type, ps.stage_order, ps.name, ps.description,
               ps.progress_percentage, ps.examples, ps.position_x, ps.position_y, ps.has_children, ps.children_loaded,
               ps.created_at, ps.updated_at
        FROM progression_stages ps
        JOIN sectors s ON ps.sector_id = s.id
        WHERE ps.id = $1
    `, stageID).Scan(
		&stage.ID,
		&sectorID,
		&treeID,
		&parentID,
		&stage.StageType,
		&stage.StageOrder,
		&stage.Name,
		&stage.Description,
		&stage.ProgressPercentage,
		&examples,
		&stage.PositionX,
		&stage.PositionY,
		&stage.HasChildren,
		&stage.ChildrenLoaded,
		&stage.CreatedAt,
		&stage.UpdatedAt,
	)
	if err != nil {
		return nil, "", err
	}
	stage.SectorID = sectorID
	if parentID.Valid {
		stage.ParentStageID = &parentID.String
	}
	stage.Examples = examples
	mappings, mapErr := getScenarioMappingsForStage(stageID)
	if mapErr == nil {
		stage.ScenarioMappings = mappings
	}
	return &stage, treeID, nil
}

// GetStagesForSector retrieves all stages for a given sector.
func (s *SectorService) GetStagesForSector(ctx context.Context, sectorID string) ([]ProgressionStage, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	rows, err := s.db.QueryContext(ctx, `
        SELECT id, sector_id, stage_type, stage_order, name, description,
               progress_percentage, examples, position_x, position_y, created_at, updated_at
        FROM progression_stages
        WHERE sector_id = $1
        ORDER BY stage_order
    `, sectorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []ProgressionStage
	for rows.Next() {
		var stage ProgressionStage
		err := rows.Scan(&stage.ID, &stage.SectorID, &stage.StageType, &stage.StageOrder,
			&stage.Name, &stage.Description, &stage.ProgressPercentage, &stage.Examples,
			&stage.PositionX, &stage.PositionY, &stage.CreatedAt, &stage.UpdatedAt)
		if err != nil {
			return nil, err
		}

		mappings, err := getScenarioMappingsForStage(stage.ID)
		if err == nil {
			stage.ScenarioMappings = mappings
		}

		stages = append(stages, stage)
	}

	return stages, nil
}

// CreateStage creates a new progression stage.
func (s *SectorService) CreateStage(ctx context.Context, params CreateStageParams) (*ProgressionStage, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}

	stageType := normalizeStageType(params.StageType)
	stageOrder := params.StageOrder
	if stageOrder <= 0 {
		order, err := s.computeNextStageOrder(ctx, params.SectorID)
		if err != nil {
			return nil, err
		}
		stageOrder = order
	}

	examplesJSON, err := encodeExamplesArray(params.Examples)
	if err != nil {
		return nil, errors.New("invalid examples payload")
	}

	progress := math.Max(0, math.Min(100, params.ProgressPercentage))

	stageID := uuid.New().String()
	if _, err := s.db.ExecContext(ctx, `
        INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description,
            progress_percentage, examples, position_x, position_y)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `, stageID, params.SectorID, stageType, stageOrder, name, params.Description, progress, examplesJSON, params.PositionX, params.PositionY); err != nil {
		return nil, err
	}

	stage, _, err := s.FetchStageByID(ctx, stageID)
	return stage, err
}

// UpdateStage updates an existing stage.
func (s *SectorService) UpdateStage(ctx context.Context, stageID string, params UpdateStageParams) (*ProgressionStage, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	updates := []string{}
	args := []interface{}{}
	idx := 1

	appendClause := func(value interface{}, template string) {
		updates = append(updates, fmt.Sprintf(template, idx))
		args = append(args, value)
		idx++
	}

	if params.SectorID != nil {
		sectorID := strings.TrimSpace(*params.SectorID)
		if sectorID != "" {
			appendClause(sectorID, "sector_id = $%d")
		}
	}

	if params.StageType != nil {
		appendClause(normalizeStageType(*params.StageType), "stage_type = $%d")
	}
	if params.StageOrder != nil {
		appendClause(*params.StageOrder, "stage_order = $%d")
	}
	if params.Name != nil {
		appendClause(strings.TrimSpace(*params.Name), "name = $%d")
	}
	if params.Description != nil {
		appendClause(strings.TrimSpace(*params.Description), "description = $%d")
	}
	if params.ProgressPercentage != nil {
		progress := math.Max(0, math.Min(100, *params.ProgressPercentage))
		appendClause(progress, "progress_percentage = $%d")
	}
	if params.PositionX != nil {
		appendClause(*params.PositionX, "position_x = $%d")
	}
	if params.PositionY != nil {
		appendClause(*params.PositionY, "position_y = $%d")
	}
	if params.Examples != nil {
		examplesJSON, err := encodeExamplesArray(params.Examples)
		if err != nil {
			return nil, errors.New("invalid examples payload")
		}
		appendClause(examplesJSON, "examples = $%d")
	}

	if len(updates) == 0 {
		return nil, errors.New("no updatable fields supplied")
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, stageID)

	query := fmt.Sprintf("UPDATE progression_stages SET %s WHERE id = $%d", strings.Join(updates, ", "), idx)
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}

	stage, _, err := s.FetchStageByID(ctx, stageID)
	return stage, err
}

// DeleteStage deletes a stage from a specific tree.
func (s *SectorService) DeleteStage(ctx context.Context, stageID, treeID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := s.db.ExecContext(ctx, `
        DELETE FROM progression_stages
        WHERE id = $1
          AND sector_id IN (
                SELECT id FROM sectors WHERE tree_id = $2
          )
    `, stageID, treeID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// computeNextStageOrder calculates the next available stage order for a sector.
func (s *SectorService) computeNextStageOrder(ctx context.Context, sectorID string) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var maxOrder sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(stage_order) FROM progression_stages WHERE sector_id = $1
	`, sectorID).Scan(&maxOrder)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	if maxOrder.Valid {
		return int(maxOrder.Int64) + 1, nil
	}
	return 1, nil
}

// FetchStageChildren retrieves all direct children of a stage (lazy loading).
func (s *SectorService) FetchStageChildren(ctx context.Context, parentStageID string) ([]ProgressionStage, error) {
	if s.db == nil {
		return nil, errors.New("database connection is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	query := `
		SELECT id, sector_id, parent_stage_id, stage_type, stage_order,
		       name, description, progress_percentage, examples,
		       position_x, position_y, has_children, children_loaded,
		       created_at, updated_at
		FROM progression_stages
		WHERE parent_stage_id = $1
		ORDER BY stage_order ASC, name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, parentStageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []ProgressionStage
	for rows.Next() {
		var child ProgressionStage
		var parentID sql.NullString

		err := rows.Scan(
			&child.ID,
			&child.SectorID,
			&parentID,
			&child.StageType,
			&child.StageOrder,
			&child.Name,
			&child.Description,
			&child.ProgressPercentage,
			&child.Examples,
			&child.PositionX,
			&child.PositionY,
			&child.HasChildren,
			&child.ChildrenLoaded,
			&child.CreatedAt,
			&child.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			child.ParentStageID = &parentID.String
		}

		children = append(children, child)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return children, nil
}

// FetchRootStagesForSector retrieves all root-level stages (parent_stage_id = NULL) for a sector.
func (s *SectorService) FetchRootStagesForSector(ctx context.Context, sectorID string) ([]ProgressionStage, error) {
	if s.db == nil {
		return nil, errors.New("database connection is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	query := `
		SELECT id, sector_id, parent_stage_id, stage_type, stage_order,
		       name, description, progress_percentage, examples,
		       position_x, position_y, has_children, children_loaded,
		       created_at, updated_at
		FROM progression_stages
		WHERE sector_id = $1 AND parent_stage_id IS NULL
		ORDER BY stage_order ASC, name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, sectorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []ProgressionStage
	for rows.Next() {
		var stage ProgressionStage
		var parentID sql.NullString

		err := rows.Scan(
			&stage.ID,
			&stage.SectorID,
			&parentID,
			&stage.StageType,
			&stage.StageOrder,
			&stage.Name,
			&stage.Description,
			&stage.ProgressPercentage,
			&stage.Examples,
			&stage.PositionX,
			&stage.PositionY,
			&stage.HasChildren,
			&stage.ChildrenLoaded,
			&stage.CreatedAt,
			&stage.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			stage.ParentStageID = &parentID.String
		}

		stages = append(stages, stage)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return stages, nil
}
