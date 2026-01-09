package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// GraphService encapsulates business logic for graph operations.
// Separates domain logic from HTTP handling concerns.
type GraphService struct {
	db *sql.DB
}

// NewGraphService creates a new graph service instance.
func NewGraphService(database *sql.DB) *GraphService {
	return &GraphService{db: database}
}

// FetchDependencies retrieves all dependencies for a given tree.
func (s *GraphService) FetchDependencies(ctx context.Context, treeID string) ([]DependencyPayload, error) {
	if s.db == nil {
		return nil, errors.New("database connection is not initialized")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT sd.id, sd.dependent_stage_id, sd.prerequisite_stage_id,
			   sd.dependency_type, sd.dependency_strength, sd.description,
			   ps1.name AS dependent_stage_name, ps2.name AS prerequisite_stage_name
		FROM stage_dependencies sd
		JOIN progression_stages ps1 ON sd.dependent_stage_id = ps1.id
		JOIN progression_stages ps2 ON sd.prerequisite_stage_id = ps2.id
		JOIN sectors s1 ON ps1.sector_id = s1.id
		JOIN sectors s2 ON ps2.sector_id = s2.id
		WHERE s1.tree_id = $1 AND s2.tree_id = $1
		ORDER BY sd.dependency_strength DESC
	`, treeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dependencies []DependencyPayload
	for rows.Next() {
		var dep StageDependency
		var dependentName, prerequisiteName string
		err := rows.Scan(&dep.ID, &dep.DependentStageID, &dep.PrerequisiteStageID,
			&dep.DependencyType, &dep.DependencyStrength, &dep.Description,
			&dependentName, &prerequisiteName)
		if err != nil {
			return nil, err
		}

		dependencies = append(dependencies, DependencyPayload{
			Dependency:       dep,
			DependentName:    dependentName,
			PrerequisiteName: prerequisiteName,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dependencies, nil
}

// UpdateGraph performs a complete graph update with positions and dependencies.
func (s *GraphService) UpdateGraph(ctx context.Context, treeID string, request GraphUpdateRequest) ([]Sector, []DependencyPayload, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Get allowed stages for this tree (security check)
	allowedStages, err := s.getAllowedStages(ctx, tx, treeID)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to load stage catalog: %w", err)
	}

	// Update stage positions
	if len(request.StagePositions) > 0 {
		if err := s.updateStagePositions(ctx, tx, request.StagePositions, allowedStages); err != nil {
			tx.Rollback()
			return nil, nil, err
		}
	}

	// Update dependencies
	if request.Dependencies != nil {
		if err := s.updateDependencies(ctx, tx, treeID, request.Dependencies, allowedStages); err != nil {
			tx.Rollback()
			return nil, nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch updated data
	sectorSvc := NewSectorService(s.db)
	stageSvc := NewStageService(s.db)
	sectors, sectorErr := sectorSvc.FetchSectorsWithStages(ctx, treeID, stageSvc)
	if sectorErr != nil {
		return nil, nil, fmt.Errorf("graph saved but sectors could not be refreshed: %w", sectorErr)
	}

	dependencies, depErr := s.FetchDependencies(ctx, treeID)
	if depErr != nil {
		return nil, nil, fmt.Errorf("graph saved but dependencies could not be refreshed: %w", depErr)
	}

	return sectors, dependencies, nil
}

// getAllowedStages returns the set of stage IDs that belong to the given tree.
func (s *GraphService) getAllowedStages(ctx context.Context, tx *sql.Tx, treeID string) (map[string]struct{}, error) {
	stageRows, err := tx.QueryContext(ctx, `
		SELECT ps.id
		FROM progression_stages ps
		JOIN sectors s ON ps.sector_id = s.id
		WHERE s.tree_id = $1
	`, treeID)
	if err != nil {
		return nil, err
	}
	defer stageRows.Close()

	allowedStages := make(map[string]struct{})
	for stageRows.Next() {
		var stageID string
		if err := stageRows.Scan(&stageID); err != nil {
			return nil, err
		}
		allowedStages[stageID] = struct{}{}
	}

	return allowedStages, nil
}

// updateStagePositions updates the positions of stages in the graph.
func (s *GraphService) updateStagePositions(ctx context.Context, tx *sql.Tx, positions []StagePositionUpdate, allowedStages map[string]struct{}) error {
	stmt, err := tx.PrepareContext(ctx, `
		UPDATE progression_stages
		SET position_x = $1,
		    position_y = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare stage updates: %w", err)
	}
	defer stmt.Close()

	for _, stage := range positions {
		if strings.TrimSpace(stage.ID) == "" {
			continue
		}

		if _, ok := allowedStages[stage.ID]; !ok {
			continue
		}

		if _, execErr := stmt.ExecContext(ctx, stage.PositionX, stage.PositionY, stage.ID); execErr != nil {
			return fmt.Errorf("failed to update stage position: %w", execErr)
		}
	}

	return nil
}

// updateDependencies manages the full set of dependencies for a tree.
func (s *GraphService) updateDependencies(ctx context.Context, tx *sql.Tx, treeID string, dependencies []GraphDependencyInput, allowedStages map[string]struct{}) error {
	// Load existing dependencies
	existingByID, existingByPair, err := s.loadExistingDependencies(ctx, tx, treeID)
	if err != nil {
		return err
	}

	keepIDs := make(map[string]struct{})
	seenPairs := make(map[string]struct{})

	// Process each dependency input
	for _, dep := range dependencies {
		dependentID := strings.TrimSpace(dep.DependentStageID)
		prerequisiteID := strings.TrimSpace(dep.PrerequisiteStageID)

		// Validate dependency
		if dependentID == "" || prerequisiteID == "" || dependentID == prerequisiteID {
			continue
		}

		if _, ok := allowedStages[dependentID]; !ok {
			continue
		}
		if _, ok := allowedStages[prerequisiteID]; !ok {
			continue
		}

		pairKey := dependentID + "|" + prerequisiteID
		if _, duplicate := seenPairs[pairKey]; duplicate {
			continue
		}
		seenPairs[pairKey] = struct{}{}

		// Normalize values
		depType := strings.TrimSpace(dep.DependencyType)
		if depType == "" {
			depType = "required"
		}

		strength := dep.DependencyStrength
		if strength < 0 {
			strength = 0
		}
		if strength > 1 {
			strength = 1
		}
		if strength == 0 {
			strength = 1
		}

		desc := strings.TrimSpace(dep.Description)
		description := sql.NullString{String: desc, Valid: desc != ""}

		// Update existing or create new
		if id := strings.TrimSpace(dep.ID); id != "" {
			if err := s.updateDependency(ctx, tx, id, dependentID, prerequisiteID, depType, strength, description); err != nil {
				return err
			}
			keepIDs[id] = struct{}{}
		} else if existing, ok := existingByPair[pairKey]; ok {
			if err := s.updateDependencyByID(ctx, tx, existing.ID, depType, strength, description); err != nil {
				return err
			}
			keepIDs[existing.ID] = struct{}{}
		} else {
			newID, err := s.createDependency(ctx, tx, dependentID, prerequisiteID, depType, strength, description)
			if err != nil {
				return err
			}
			keepIDs[newID] = struct{}{}
		}
	}

	// Remove dependencies that are no longer in the request
	for id := range existingByID {
		if _, ok := keepIDs[id]; !ok {
			if err := s.deleteDependency(ctx, tx, id); err != nil {
				return err
			}
		}
	}

	return nil
}

// loadExistingDependencies loads the current dependencies for a tree.
func (s *GraphService) loadExistingDependencies(ctx context.Context, tx *sql.Tx, treeID string) (map[string]StageDependency, map[string]StageDependency, error) {
	existingRows, err := tx.QueryContext(ctx, `
		SELECT sd.id, sd.dependent_stage_id, sd.prerequisite_stage_id
		FROM stage_dependencies sd
		JOIN progression_stages psd ON sd.dependent_stage_id = psd.id
		JOIN progression_stages psp ON sd.prerequisite_stage_id = psp.id
		JOIN sectors s ON psd.sector_id = s.id
		WHERE s.tree_id = $1
	`, treeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load current dependencies: %w", err)
	}
	defer existingRows.Close()

	existingByID := make(map[string]StageDependency)
	existingByPair := make(map[string]StageDependency)

	for existingRows.Next() {
		var dep StageDependency
		if err := existingRows.Scan(&dep.ID, &dep.DependentStageID, &dep.PrerequisiteStageID); err != nil {
			return nil, nil, fmt.Errorf("failed to read existing dependency: %w", err)
		}

		existingByID[dep.ID] = dep
		existingByPair[dep.DependentStageID+"|"+dep.PrerequisiteStageID] = dep
	}

	if err := existingRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to inspect existing dependencies: %w", err)
	}

	return existingByID, existingByPair, nil
}

// updateDependency updates a dependency by ID with full field replacement.
func (s *GraphService) updateDependency(ctx context.Context, tx *sql.Tx, id, dependentID, prerequisiteID, depType string, strength float64, description sql.NullString) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE stage_dependencies
		SET dependent_stage_id = $1,
		    prerequisite_stage_id = $2,
		    dependency_type = $3,
		    dependency_strength = $4,
		    description = $5
		WHERE id = $6
	`, dependentID, prerequisiteID, depType, strength, description, id)
	if err != nil {
		return fmt.Errorf("failed to update dependency: %w", err)
	}
	return nil
}

// updateDependencyByID updates only the metadata of a dependency (keeps endpoints the same).
func (s *GraphService) updateDependencyByID(ctx context.Context, tx *sql.Tx, id, depType string, strength float64, description sql.NullString) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE stage_dependencies
		SET dependency_type = $1,
		    dependency_strength = $2,
		    description = $3
		WHERE id = $4
	`, depType, strength, description, id)
	if err != nil {
		return fmt.Errorf("failed to refresh dependency: %w", err)
	}
	return nil
}

// createDependency creates a new dependency and returns its ID.
func (s *GraphService) createDependency(ctx context.Context, tx *sql.Tx, dependentID, prerequisiteID, depType string, strength float64, description sql.NullString) (string, error) {
	var newID string
	err := tx.QueryRowContext(ctx, `
		INSERT INTO stage_dependencies (dependent_stage_id, prerequisite_stage_id, dependency_type, dependency_strength, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, dependentID, prerequisiteID, depType, strength, description).Scan(&newID)
	if err != nil {
		return "", fmt.Errorf("failed to create dependency: %w", err)
	}
	return newID, nil
}

// deleteDependency removes a dependency by ID.
func (s *GraphService) deleteDependency(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM stage_dependencies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to remove dependency: %w", err)
	}
	return nil
}

// ExportGraphAsDOT generates a DOT (Graphviz) representation of the tech tree.
func (s *GraphService) ExportGraphAsDOT(ctx context.Context, tree *TechTree, sectors []Sector, dependencies []DependencyPayload) string {
	var builder strings.Builder
	builder.WriteString("digraph TechTree {\n")
	builder.WriteString("  rankdir=LR;\n")
	builder.WriteString("  labelloc=\"t\";\n")
	builder.WriteString(fmt.Sprintf("  label=\"%s (%s)\";\n", escapeDOTLabel(tree.Name), escapeDOTLabel(tree.TreeType)))
	builder.WriteString("  node [shape=box, style=\"rounded,filled\", fillcolor=\"#0f172a\", fontcolor=white, fontname=\"Inter\"];\n")

	for _, sector := range sectors {
		for _, stage := range sector.Stages {
			label := fmt.Sprintf("%s\\n%s\\n%.0f%%%%", escapeDOTLabel(stage.Name), escapeDOTLabel(prettifyStageType(stage.StageType)), stage.ProgressPercentage)
			builder.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", fillcolor=\"%s\"];\n", stage.ID, label, sector.Color))
		}
	}

	for _, dep := range dependencies {
		strength := dep.Dependency.DependencyStrength
		if strength <= 0 {
			strength = 0.35
		}
		penWidth := 1.0 + (strength * 2)
		builder.WriteString(fmt.Sprintf(
			"  \"%s\" -> \"%s\" [label=\"%s\", penwidth=%.2f];\n",
			dep.Dependency.PrerequisiteStageID,
			dep.Dependency.DependentStageID,
			escapeDOTLabel(dep.Dependency.DependencyType),
			penWidth,
		))
	}

	builder.WriteString("}\n")
	return builder.String()
}

func escapeDOTLabel(value string) string {
	replaced := strings.ReplaceAll(value, "\\", "\\\\")
	replaced = strings.ReplaceAll(replaced, "\"", "\\\"")
	replaced = strings.ReplaceAll(replaced, "\n", "\\n")
	return replaced
}
