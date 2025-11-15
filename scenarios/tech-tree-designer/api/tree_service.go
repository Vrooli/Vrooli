package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TreeService encapsulates business logic for tech tree operations.
// Separates domain logic from HTTP handling concerns.
type TreeService struct {
	db *sql.DB
}

// NewTreeService creates a new tree service instance.
func NewTreeService(database *sql.DB) *TreeService {
	return &TreeService{db: database}
}

// CreateTreeParams contains parameters for creating a new tech tree.
type CreateTreeParams struct {
	Name         string
	Slug         string
	Description  string
	TreeType     string
	Status       string
	Version      string
	ParentTreeID string
	IsActive     *bool
}

// UpdateTreeParams contains optional fields for updating a tech tree.
type UpdateTreeParams struct {
	Name        *string
	Description *string
	TreeType    *string
	Status      *string
	Version     *string
	Slug        *string
	IsActive    *bool
}

// CloneTreeParams contains parameters for cloning a tech tree.
type CloneTreeParams struct {
	Name        string
	Slug        string
	Description string
	TreeType    string
	Status      string
	IsActive    *bool
}

// ListTreesFilters contains filters for listing tech trees.
type ListTreesFilters struct {
	TreeType        string
	Status          string
	IncludeArchived bool
	TreeID          string
	Slug            string
}

// CreateTree creates a new tech tree with the given parameters.
func (s *TreeService) CreateTree(ctx context.Context, params CreateTreeParams) (*TechTree, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Validate and sanitize inputs
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}

	slugInput := params.Slug
	if slugInput == "" {
		slugInput = name
	}
	slug := normalizeSlug(slugInput)

	description := strings.TrimSpace(params.Description)
	treeType := strings.TrimSpace(params.TreeType)
	if treeType == "" {
		treeType = "experimental"
	}
	status := strings.TrimSpace(params.Status)
	if status == "" {
		status = "active"
	}
	version := strings.TrimSpace(params.Version)
	if version == "" {
		version = "1.0.0"
	}
	parentID := strings.TrimSpace(params.ParentTreeID)

	isActive := false
	if params.IsActive != nil {
		isActive = *params.IsActive
	} else if treeType == "official" && status == "active" {
		isActive = true
	}

	// Insert new tree
	newID := uuid.New().String()
	log.Printf("DEBUG: Creating tree with ID=%s, slug=%s, name=%s, type=%s, status=%s, active=%v, parentID=%s",
		newID, slug, name, treeType, status, isActive, parentID)

	// Handle parent_tree_id - must be NULL if empty, not empty string
	var parentTreeID interface{}
	if parentID != "" {
		parentTreeID = parentID
	} else {
		parentTreeID = nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tech_trees (id, slug, name, description, version, tree_type, status, is_active, parent_tree_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, newID, slug, name, description, version, treeType, status, isActive, parentTreeID)
	if err != nil {
		log.Printf("ERROR: Failed to insert tech tree: %v", err)
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key value") {
			return nil, fmt.Errorf("slug '%s' already exists", slug)
		}
		return nil, fmt.Errorf("failed to insert tech tree: %w", err)
	}

	log.Printf("DEBUG: Tree created successfully, fetching details for ID=%s", newID)
	return fetchTechTreeByID(ctx, newID)
}

// UpdateTree updates an existing tech tree with the given parameters.
func (s *TreeService) UpdateTree(ctx context.Context, treeID string, params UpdateTreeParams) (*TechTree, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if strings.TrimSpace(treeID) == "" {
		return nil, errors.New("tree ID is required")
	}

	setClauses := []string{}
	args := []interface{}{}
	paramIndex := 1

	appendClause := func(value interface{}, clause string) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf(clause, paramIndex))
		paramIndex++
	}

	if params.Name != nil {
		name := strings.TrimSpace(*params.Name)
		if name != "" {
			appendClause(name, "name = $%d")
		}
	}

	if params.Description != nil {
		appendClause(strings.TrimSpace(*params.Description), "description = $%d")
	}

	if params.TreeType != nil {
		treeType := strings.TrimSpace(*params.TreeType)
		if treeType != "" {
			appendClause(treeType, "tree_type = $%d")
		}
	}

	if params.Status != nil {
		status := strings.TrimSpace(*params.Status)
		if status != "" {
			appendClause(status, "status = $%d")
		}
	}

	if params.Version != nil {
		version := strings.TrimSpace(*params.Version)
		if version != "" {
			appendClause(version, "version = $%d")
		}
	}

	if params.Slug != nil {
		slugInput := strings.TrimSpace(*params.Slug)
		if slugInput != "" {
			appendClause(normalizeSlug(slugInput), "slug = $%d")
		}
	}

	if params.IsActive != nil {
		appendClause(*params.IsActive, "is_active = $%d")
	}

	if len(setClauses) == 0 {
		return nil, errors.New("no updatable fields provided")
	}

	args = append(args, treeID)
	updateQuery := fmt.Sprintf("UPDATE tech_trees SET %s, updated_at = CURRENT_TIMESTAMP WHERE id = $%d", strings.Join(setClauses, ", "), len(args))
	result, err := s.db.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key value") {
			return nil, errors.New("slug already exists")
		}
		return nil, fmt.Errorf("failed to update tech tree: %w", err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil, sql.ErrNoRows
	}

	tree, err := fetchTechTreeByID(ctx, treeID)
	if err != nil {
		return nil, err
	}

	// Ensure only one official tree is active
	if tree.TreeType == "official" && tree.IsActive {
		_, _ = s.db.ExecContext(ctx, `UPDATE tech_trees SET is_active = false WHERE id <> $1 AND tree_type = 'official'`, tree.ID)
	}

	return tree, nil
}

// CloneTree creates a deep copy of an existing tech tree with all its sectors, stages, and dependencies.
func (s *TreeService) CloneTree(ctx context.Context, sourceTreeID string, params CloneTreeParams) (*TechTree, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if strings.TrimSpace(sourceTreeID) == "" {
		return nil, errors.New("source tree ID is required")
	}

	// Fetch source tree
	sourceTree, err := fetchTechTreeByID(ctx, sourceTreeID)
	if err != nil {
		return nil, fmt.Errorf("failed to load source tree: %w", err)
	}

	// Prepare cloned tree metadata
	name := strings.TrimSpace(params.Name)
	if name == "" {
		name = fmt.Sprintf("%s Draft", sourceTree.Name)
	}

	description := strings.TrimSpace(params.Description)
	if description == "" {
		description = sourceTree.Description
	}

	slugInput := strings.TrimSpace(params.Slug)
	if slugInput == "" {
		slugInput = fmt.Sprintf("%s-%d", name, time.Now().Unix())
	}
	slug := normalizeSlug(slugInput)

	treeType := strings.TrimSpace(params.TreeType)
	if treeType == "" {
		treeType = "draft"
	}
	status := strings.TrimSpace(params.Status)
	if status == "" {
		status = "active"
	}
	isActive := false
	if params.IsActive != nil {
		isActive = *params.IsActive
	}

	newTreeID := uuid.New().String()

	// Start transaction for deep clone
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start clone transaction: %w", err)
	}

	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}

	// Clone tech tree
	log.Printf("DEBUG: Cloning tree sourceID=%s to newID=%s, slug=%s, name=%s", sourceTreeID, newTreeID, slug, name)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tech_trees (id, slug, name, description, version, tree_type, status, is_active, parent_tree_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, newTreeID, slug, name, description, sourceTree.Version, treeType, status, isActive, sourceTree.ID); err != nil {
		log.Printf("ERROR: Failed to clone tech tree: %v", err)
		return nil, rollback(fmt.Errorf("failed to create cloned tech tree: %w", err))
	}

	// Clone sectors
	sectorMap, err := s.cloneSectors(ctx, tx, sourceTreeID, newTreeID)
	if err != nil {
		return nil, rollback(err)
	}

	// Clone stages
	stageMap, err := s.cloneStages(ctx, tx, sourceTreeID, sectorMap)
	if err != nil {
		return nil, rollback(err)
	}

	// Clone dependencies
	if err := s.cloneDependencies(ctx, tx, sourceTreeID, stageMap); err != nil {
		return nil, rollback(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit clone transaction: %w", err)
	}

	return fetchTechTreeByID(ctx, newTreeID)
}

func (s *TreeService) cloneSectors(ctx context.Context, tx *sql.Tx, sourceTreeID, newTreeID string) (map[string]string, error) {
	sectorMap := make(map[string]string)

	// First, read all sectors into memory
	type SectorData struct {
		ID          string
		Name        string
		Category    string
		Description string
		Progress    float64
		PositionX   float64
		PositionY   float64
		Color       string
	}

	sectorRows, err := tx.QueryContext(ctx, `
		SELECT id, name, category, description, progress_percentage, position_x, position_y, color
		FROM sectors
		WHERE tree_id = $1
	`, sourceTreeID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sectors for cloning: %w", err)
	}

	var sectors []SectorData
	for sectorRows.Next() {
		var s SectorData
		if err := sectorRows.Scan(&s.ID, &s.Name, &s.Category, &s.Description, &s.Progress, &s.PositionX, &s.PositionY, &s.Color); err != nil {
			sectorRows.Close()
			return nil, fmt.Errorf("failed to scan sector: %w", err)
		}
		sectors = append(sectors, s)
	}
	sectorRows.Close()

	if err := sectorRows.Err(); err != nil {
		return nil, fmt.Errorf("error reading sector rows: %w", err)
	}

	// Now insert the sectors
	for _, s := range sectors {
		newID := uuid.New().String()
		sectorMap[s.ID] = newID
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sectors (id, tree_id, name, category, description, progress_percentage, position_x, position_y, color, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, newID, newTreeID, s.Name, s.Category, s.Description, s.Progress, s.PositionX, s.PositionY, s.Color); err != nil {
			return nil, fmt.Errorf("failed to clone sector: %w", err)
		}
	}

	return sectorMap, nil
}

func (s *TreeService) cloneStages(ctx context.Context, tx *sql.Tx, sourceTreeID string, sectorMap map[string]string) (map[string]string, error) {
	stageMap := make(map[string]string)

	// First, read all stages into memory
	type StageData struct {
		ID         string
		SectorID   string
		StageType  string
		StageOrder int
		Name       string
		Desc       string
		Progress   float64
		Examples   json.RawMessage
		PositionX  float64
		PositionY  float64
	}

	stageRows, err := tx.QueryContext(ctx, `
		SELECT ps.id, ps.sector_id, ps.stage_type, ps.stage_order, ps.name, ps.description,
		       ps.progress_percentage, ps.examples, ps.position_x, ps.position_y
		FROM progression_stages ps
		JOIN sectors s ON ps.sector_id = s.id
		WHERE s.tree_id = $1
	`, sourceTreeID)
	if err != nil {
		return nil, fmt.Errorf("failed to load stages for cloning: %w", err)
	}

	var stages []StageData
	for stageRows.Next() {
		var s StageData
		if err := stageRows.Scan(&s.ID, &s.SectorID, &s.StageType, &s.StageOrder, &s.Name, &s.Desc, &s.Progress, &s.Examples, &s.PositionX, &s.PositionY); err != nil {
			stageRows.Close()
			return nil, fmt.Errorf("failed to scan stage: %w", err)
		}
		stages = append(stages, s)
	}
	stageRows.Close()

	if err := stageRows.Err(); err != nil {
		return nil, fmt.Errorf("error reading stage rows: %w", err)
	}

	// Now insert the stages
	for _, s := range stages {
		newSectorID, ok := sectorMap[s.SectorID]
		if !ok {
			log.Printf("warning: stage %s references unknown sector %s", s.ID, s.SectorID)
			continue
		}

		newStageID := uuid.New().String()
		stageMap[s.ID] = newStageID
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description, progress_percentage, examples, position_x, position_y, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, newStageID, newSectorID, s.StageType, s.StageOrder, s.Name, s.Desc, s.Progress, s.Examples, s.PositionX, s.PositionY); err != nil {
			return nil, fmt.Errorf("failed to clone stage: %w", err)
		}
	}

	return stageMap, nil
}

func (s *TreeService) cloneDependencies(ctx context.Context, tx *sql.Tx, sourceTreeID string, stageMap map[string]string) error {
	// First, read all dependencies into memory
	type DependencyData struct {
		DependentID    string
		PrerequisiteID string
		DepType        string
		Strength       float64
		Description    string
	}

	depRows, err := tx.QueryContext(ctx, `
		SELECT sd.dependent_stage_id, sd.prerequisite_stage_id, sd.dependency_type, sd.dependency_strength, sd.description
		FROM stage_dependencies sd
		JOIN progression_stages ps1 ON sd.dependent_stage_id = ps1.id
		JOIN sectors s1 ON ps1.sector_id = s1.id
		WHERE s1.tree_id = $1
	`, sourceTreeID)
	if err != nil {
		return fmt.Errorf("failed to load dependencies for cloning: %w", err)
	}

	var dependencies []DependencyData
	for depRows.Next() {
		var d DependencyData
		if err := depRows.Scan(&d.DependentID, &d.PrerequisiteID, &d.DepType, &d.Strength, &d.Description); err != nil {
			depRows.Close()
			return fmt.Errorf("failed to scan dependency: %w", err)
		}
		dependencies = append(dependencies, d)
	}
	depRows.Close()

	if err := depRows.Err(); err != nil {
		return fmt.Errorf("error reading dependency rows: %w", err)
	}

	// Now insert the dependencies
	for _, d := range dependencies {
		newDependentID, ok1 := stageMap[d.DependentID]
		newPrerequisiteID, ok2 := stageMap[d.PrerequisiteID]
		if !ok1 || !ok2 {
			log.Printf("warning: skipping dependency %s->%s (missing mapped stage)", d.PrerequisiteID, d.DependentID)
			continue
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO stage_dependencies (dependent_stage_id, prerequisite_stage_id, dependency_type, dependency_strength, description)
			VALUES ($1, $2, $3, $4, $5)
		`, newDependentID, newPrerequisiteID, d.DepType, d.Strength, d.Description); err != nil {
			return fmt.Errorf("failed to clone dependency: %w", err)
		}
	}

	return nil
}

// ListTrees retrieves tech trees with optional filtering and includes statistics.
func (s *TreeService) ListTrees(ctx context.Context, filters ListTreesFilters) ([]TechTreeSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	listFilters := []string{}
	args := []interface{}{}
	index := 1

	if filters.TreeType != "" {
		listFilters = append(listFilters, fmt.Sprintf("tree_type = $%d", index))
		args = append(args, filters.TreeType)
		index++
	}

	if filters.Status != "" {
		listFilters = append(listFilters, fmt.Sprintf("status = $%d", index))
		args = append(args, filters.Status)
		index++
	}

	if !filters.IncludeArchived {
		listFilters = append(listFilters, "status <> 'archived'")
	}

	if filters.TreeID != "" {
		listFilters = append(listFilters, fmt.Sprintf("id = $%d", index))
		args = append(args, filters.TreeID)
		index++
	}

	if filters.Slug != "" {
		listFilters = append(listFilters, fmt.Sprintf("slug = $%d", index))
		args = append(args, filters.Slug)
		index++
	}

	baseQuery := `
		SELECT id, slug, name, description, version, tree_type, status, is_active, parent_tree_id,
		       created_at, updated_at,
		       (SELECT COUNT(1) FROM sectors WHERE tree_id = tech_trees.id) AS sector_count,
		       (SELECT COUNT(1)
		        FROM progression_stages ps
		        JOIN sectors s ON ps.sector_id = s.id
		        WHERE s.tree_id = tech_trees.id) AS stage_count,
		       (SELECT COUNT(1)
		        FROM scenario_mappings sm
		        JOIN progression_stages ps ON sm.stage_id = ps.id
		        JOIN sectors s ON ps.sector_id = s.id
		        WHERE s.tree_id = tech_trees.id) AS mapping_count
		FROM tech_trees
	`

	if len(listFilters) > 0 {
		baseQuery += " WHERE " + strings.Join(listFilters, " AND ")
	}

	baseQuery += " ORDER BY CASE WHEN tree_type = 'official' THEN 0 ELSE 1 END, updated_at DESC"

	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tech trees: %w", err)
	}
	defer rows.Close()

	var summaries []TechTreeSummary
	for rows.Next() {
		var tree TechTree
		var parent sql.NullString
		var summary TechTreeSummary
		err := rows.Scan(
			&tree.ID,
			&tree.Slug,
			&tree.Name,
			&tree.Description,
			&tree.Version,
			&tree.TreeType,
			&tree.Status,
			&tree.IsActive,
			&parent,
			&tree.CreatedAt,
			&tree.UpdatedAt,
			&summary.SectorCount,
			&summary.StageCount,
			&summary.ScenarioMappings,
		)
		if err != nil {
			continue
		}
		if parent.Valid {
			tree.ParentTree = &parent.String
		}
		summary.Tree = tree
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to stream tech trees: %w", err)
	}

	return summaries, nil
}
