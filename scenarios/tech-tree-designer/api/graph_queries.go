package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GraphQueryService provides graph traversal and query operations.
// Designed for agent/LLM consumption to understand tech tree structure.
type GraphQueryService struct {
	db *sql.DB
}

// NewGraphQueryService creates a new graph query service instance.
func NewGraphQueryService(database *sql.DB) *GraphQueryService {
	return &GraphQueryService{db: database}
}

// NeighborhoodOptions configures neighborhood query behavior.
type NeighborhoodOptions struct {
	StageID           string
	Depth             int
	Direction         string // "prerequisites", "dependents", "both"
	IncludeHierarchy  bool   // Include hierarchical children
	IncludeScenarios  bool   // Include scenario mappings
	MaxResults        int    // Limit results to prevent huge responses
}

// PathOptions configures path-finding queries.
type PathOptions struct {
	FromStageID string
	ToStageID   string
	MaxDepth    int // Prevent infinite loops
}

// AncestorOptions configures hierarchical ancestor queries.
type AncestorOptions struct {
	StageID         string
	Depth           int  // How many levels up
	IncludeChildren bool // Include siblings at each level
}

// StageWithContext represents a stage with its full context for agent consumption.
type StageWithContext struct {
	Stage            ProgressionStage  `json:"stage"`
	Sector           Sector            `json:"sector"`
	ScenarioMappings []ScenarioMapping `json:"scenario_mappings,omitempty"`
	Distance         int               `json:"distance,omitempty"` // Distance from query origin
}

// NeighborhoodResponse contains stages within N hops of a node.
type NeighborhoodResponse struct {
	Origin   StageWithContext   `json:"origin"`
	Stages   []StageWithContext `json:"stages"`
	EdgeCount int               `json:"edge_count"`
	MaxDepth int                `json:"max_depth_reached"`
}

// PathResponse contains the shortest path between two stages.
type PathResponse struct {
	Path     []StageWithContext `json:"path"`
	Length   int                `json:"length"`
	Found    bool               `json:"found"`
}

// AncestorResponse contains hierarchical ancestor chain.
type AncestorResponse struct {
	Origin    StageWithContext   `json:"origin"`
	Ancestors []StageWithContext `json:"ancestors"`
	Depth     int                `json:"depth_reached"`
}

// GetNeighborhood returns all stages within N hops using dependency relationships.
func (s *GraphQueryService) GetNeighborhood(ctx context.Context, treeID string, opts NeighborhoodOptions) (*NeighborhoodResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Validate inputs
	if opts.Depth < 1 {
		opts.Depth = 1
	}
	if opts.Depth > 5 {
		opts.Depth = 5 // Cap at 5 to prevent performance issues
	}
	if opts.MaxResults == 0 {
		opts.MaxResults = 100
	}
	if opts.Direction == "" {
		opts.Direction = "both"
	}

	// Get origin stage with context
	origin, err := s.getStageWithContext(ctx, opts.StageID, treeID)
	if err != nil {
		return nil, fmt.Errorf("failed to load origin stage: %w", err)
	}

	// BFS traversal using dependencies
	visited := make(map[string]bool)
	visited[opts.StageID] = true

	var stages []StageWithContext
	currentLevel := []string{opts.StageID}
	depth := 0

	for depth < opts.Depth && len(currentLevel) > 0 && len(stages) < opts.MaxResults {
		depth++
		var nextLevel []string

		for _, stageID := range currentLevel {
			neighbors, err := s.getNeighbors(ctx, stageID, treeID, opts.Direction)
			if err != nil {
				continue
			}

			for _, neighborID := range neighbors {
				if visited[neighborID] {
					continue
				}
				visited[neighborID] = true

				stageCtx, err := s.getStageWithContext(ctx, neighborID, treeID)
				if err != nil {
					continue
				}
				stageCtx.Distance = depth

				stages = append(stages, *stageCtx)
				nextLevel = append(nextLevel, neighborID)

				if len(stages) >= opts.MaxResults {
					break
				}
			}

			if len(stages) >= opts.MaxResults {
				break
			}
		}

		currentLevel = nextLevel
	}

	// Optionally include hierarchical children
	if opts.IncludeHierarchy {
		children, err := s.getHierarchicalDescendants(ctx, opts.StageID, treeID, opts.Depth)
		if err == nil {
			stages = append(stages, children...)
		}
	}

	return &NeighborhoodResponse{
		Origin:   *origin,
		Stages:   stages,
		EdgeCount: len(stages),
		MaxDepth: depth,
	}, nil
}

// GetShortestPath finds the shortest dependency path between two stages.
func (s *GraphQueryService) GetShortestPath(ctx context.Context, treeID string, opts PathOptions) (*PathResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if opts.MaxDepth == 0 {
		opts.MaxDepth = 10
	}
	if opts.MaxDepth > 20 {
		opts.MaxDepth = 20
	}

	// BFS for shortest path
	visited := make(map[string]bool)
	parent := make(map[string]string)
	queue := []string{opts.FromStageID}
	visited[opts.FromStageID] = true

	found := false
	depth := 0

	for len(queue) > 0 && depth < opts.MaxDepth && !found {
		depth++
		levelSize := len(queue)

		for i := 0; i < levelSize && !found; i++ {
			current := queue[0]
			queue = queue[1:]

			neighbors, err := s.getNeighbors(ctx, current, treeID, "both")
			if err != nil {
				continue
			}

			for _, neighbor := range neighbors {
				if visited[neighbor] {
					continue
				}

				visited[neighbor] = true
				parent[neighbor] = current

				if neighbor == opts.ToStageID {
					found = true
					break
				}

				queue = append(queue, neighbor)
			}
		}
	}

	if !found {
		return &PathResponse{
			Path:   []StageWithContext{},
			Length: 0,
			Found:  false,
		}, nil
	}

	// Reconstruct path
	var pathIDs []string
	current := opts.ToStageID
	for current != "" {
		pathIDs = append([]string{current}, pathIDs...)
		current = parent[current]
		if current == opts.FromStageID {
			pathIDs = append([]string{current}, pathIDs...)
			break
		}
	}

	// Load full context for each stage in path
	var path []StageWithContext
	for i, stageID := range pathIDs {
		stageCtx, err := s.getStageWithContext(ctx, stageID, treeID)
		if err != nil {
			continue
		}
		stageCtx.Distance = i
		path = append(path, *stageCtx)
	}

	return &PathResponse{
		Path:   path,
		Length: len(path) - 1,
		Found:  true,
	}, nil
}

// GetAncestors returns the hierarchical parent chain.
func (s *GraphQueryService) GetAncestors(ctx context.Context, treeID string, opts AncestorOptions) (*AncestorResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if opts.Depth < 1 {
		opts.Depth = 10 // Default to full chain
	}
	if opts.Depth > 50 {
		opts.Depth = 50
	}

	origin, err := s.getStageWithContext(ctx, opts.StageID, treeID)
	if err != nil {
		return nil, fmt.Errorf("failed to load origin stage: %w", err)
	}

	var ancestors []StageWithContext
	currentID := origin.Stage.ParentStageID
	depth := 0

	for currentID != nil && depth < opts.Depth {
		depth++
		parentCtx, err := s.getStageWithContext(ctx, *currentID, treeID)
		if err != nil {
			break
		}
		parentCtx.Distance = depth
		ancestors = append(ancestors, *parentCtx)

		currentID = parentCtx.Stage.ParentStageID
	}

	return &AncestorResponse{
		Origin:    *origin,
		Ancestors: ancestors,
		Depth:     depth,
	}, nil
}

// Helper: getStageWithContext loads a stage with full context (sector, scenarios).
func (s *GraphQueryService) getStageWithContext(ctx context.Context, stageID, treeID string) (*StageWithContext, error) {
	var stage ProgressionStage
	var sector Sector

	// Load stage with sector
	err := s.db.QueryRowContext(ctx, `
		SELECT ps.id, ps.sector_id, ps.parent_stage_id, ps.stage_type, ps.stage_order,
		       ps.name, ps.description, ps.progress_percentage, ps.maturity, ps.examples,
		       ps.position_x, ps.position_y, ps.has_children, ps.children_loaded,
		       ps.created_at, ps.updated_at,
		       sec.id, sec.tree_id, sec.name, sec.category, sec.description,
		       sec.progress_percentage, sec.position_x, sec.position_y, sec.color,
		       sec.created_at, sec.updated_at
		FROM progression_stages ps
		JOIN sectors sec ON ps.sector_id = sec.id
		WHERE ps.id = $1 AND sec.tree_id = $2
	`, stageID, treeID).Scan(
		&stage.ID, &stage.SectorID, &stage.ParentStageID, &stage.StageType, &stage.StageOrder,
		&stage.Name, &stage.Description, &stage.ProgressPercentage, &stage.Maturity, &stage.Examples,
		&stage.PositionX, &stage.PositionY, &stage.HasChildren, &stage.ChildrenLoaded,
		&stage.CreatedAt, &stage.UpdatedAt,
		&sector.ID, &sector.TreeID, &sector.Name, &sector.Category, &sector.Description,
		&sector.ProgressPercentage, &sector.PositionX, &sector.PositionY, &sector.Color,
		&sector.CreatedAt, &sector.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("stage not found: %s", stageID)
		}
		return nil, err
	}

	// Load scenario mappings
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, scenario_name, stage_id, contribution_weight, completion_status,
		       priority, estimated_impact, last_status_check, notes, created_at, updated_at
		FROM scenario_mappings
		WHERE stage_id = $1
	`, stageID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var mapping ScenarioMapping
			if err := rows.Scan(&mapping.ID, &mapping.ScenarioName, &mapping.StageID,
				&mapping.ContributionWeight, &mapping.CompletionStatus, &mapping.Priority,
				&mapping.EstimatedImpact, &mapping.LastStatusCheck, &mapping.Notes,
				&mapping.CreatedAt, &mapping.UpdatedAt); err == nil {
				stage.ScenarioMappings = append(stage.ScenarioMappings, mapping)
			}
		}
	}

	return &StageWithContext{
		Stage:            stage,
		Sector:           sector,
		ScenarioMappings: stage.ScenarioMappings,
	}, nil
}

// Helper: getNeighbors returns stage IDs connected via dependencies.
func (s *GraphQueryService) getNeighbors(ctx context.Context, stageID, treeID, direction string) ([]string, error) {
	var query string
	switch direction {
	case "prerequisites":
		query = `
			SELECT DISTINCT sd.prerequisite_stage_id
			FROM stage_dependencies sd
			JOIN progression_stages ps ON sd.prerequisite_stage_id = ps.id
			JOIN sectors sec ON ps.sector_id = sec.id
			WHERE sd.dependent_stage_id = $1 AND sec.tree_id = $2
		`
	case "dependents":
		query = `
			SELECT DISTINCT sd.dependent_stage_id
			FROM stage_dependencies sd
			JOIN progression_stages ps ON sd.dependent_stage_id = ps.id
			JOIN sectors sec ON ps.sector_id = sec.id
			WHERE sd.prerequisite_stage_id = $1 AND sec.tree_id = $2
		`
	default: // "both"
		query = `
			SELECT DISTINCT stage_id FROM (
				SELECT sd.prerequisite_stage_id AS stage_id
				FROM stage_dependencies sd
				JOIN progression_stages ps ON sd.prerequisite_stage_id = ps.id
				JOIN sectors sec ON ps.sector_id = sec.id
				WHERE sd.dependent_stage_id = $1 AND sec.tree_id = $2
				UNION
				SELECT sd.dependent_stage_id AS stage_id
				FROM stage_dependencies sd
				JOIN progression_stages ps ON sd.dependent_stage_id = ps.id
				JOIN sectors sec ON ps.sector_id = sec.id
				WHERE sd.prerequisite_stage_id = $1 AND sec.tree_id = $2
			) AS neighbors
		`
	}

	rows, err := s.db.QueryContext(ctx, query, stageID, treeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var neighbors []string
	for rows.Next() {
		var neighborID string
		if err := rows.Scan(&neighborID); err == nil {
			neighbors = append(neighbors, neighborID)
		}
	}

	return neighbors, nil
}

// Helper: getHierarchicalDescendants returns child stages recursively.
func (s *GraphQueryService) getHierarchicalDescendants(ctx context.Context, stageID, treeID string, maxDepth int) ([]StageWithContext, error) {
	var results []StageWithContext
	visited := make(map[string]bool)

	var traverse func(parentID string, depth int) error
	traverse = func(parentID string, depth int) error {
		if depth >= maxDepth || visited[parentID] {
			return nil
		}
		visited[parentID] = true

		rows, err := s.db.QueryContext(ctx, `
			SELECT ps.id
			FROM progression_stages ps
			JOIN sectors sec ON ps.sector_id = sec.id
			WHERE ps.parent_stage_id = $1 AND sec.tree_id = $2
		`, parentID, treeID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var childID string
			if err := rows.Scan(&childID); err != nil {
				continue
			}

			childCtx, err := s.getStageWithContext(ctx, childID, treeID)
			if err != nil {
				continue
			}
			childCtx.Distance = depth + 1
			results = append(results, *childCtx)

			// Recurse
			if err := traverse(childID, depth+1); err != nil {
				return err
			}
		}

		return nil
	}

	if err := traverse(stageID, 0); err != nil {
		return nil, err
	}

	return results, nil
}
