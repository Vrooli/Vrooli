package enrichment

import (
	"context"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// HierarchyResolver resolves parent/child status rollup.
type HierarchyResolver struct {
	// Cache for resolved statuses
	resolvedLive     map[string]types.LiveStatus
	resolvedDeclared map[string]types.DeclaredStatus
}

// NewHierarchyResolver creates a new HierarchyResolver.
func NewHierarchyResolver() *HierarchyResolver {
	return &HierarchyResolver{
		resolvedLive:     make(map[string]types.LiveStatus),
		resolvedDeclared: make(map[string]types.DeclaredStatus),
	}
}

// ResolveHierarchy resolves status rollup from children to parents.
func (h *HierarchyResolver) ResolveHierarchy(ctx context.Context, index *parsing.ModuleIndex) error {
	if index == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Reset caches
	h.resolvedLive = make(map[string]types.LiveStatus)
	h.resolvedDeclared = make(map[string]types.DeclaredStatus)

	// Process all requirements, starting from roots
	visited := make(map[string]bool)
	resolving := make(map[string]bool)

	for id := range index.ByID {
		if _, hasParent := index.ParentIndex[id]; !hasParent {
			// This is a root node
			h.resolveRequirement(ctx, id, index, visited, resolving)
		}
	}

	// Apply resolved statuses back to requirements
	for _, module := range index.Modules {
		for i := range module.Requirements {
			req := &module.Requirements[i]
			normalizedID := parsing.NormalizeID(req.ID)

			if live, ok := h.resolvedLive[normalizedID]; ok && live != "" {
				// Only update if no direct live status
				if req.LiveStatus == "" || req.LiveStatus == types.LiveUnknown {
					req.LiveStatus = live
				}
			}

			if declared, ok := h.resolvedDeclared[normalizedID]; ok && declared != "" {
				// Update aggregated status
				req.AggregatedStatus.DeclaredRollup = declared
			}
		}
	}

	return nil
}

// resolveRequirement recursively resolves status for a requirement and its children.
func (h *HierarchyResolver) resolveRequirement(
	ctx context.Context,
	id string,
	index *parsing.ModuleIndex,
	visited, resolving map[string]bool,
) (types.LiveStatus, types.DeclaredStatus, error) {
	select {
	case <-ctx.Done():
		return types.LiveUnknown, types.StatusPending, ctx.Err()
	default:
	}

	normalizedID := parsing.NormalizeID(id)

	// Check for cycle
	if resolving[normalizedID] {
		return types.LiveUnknown, types.StatusPending, types.ErrCycleDetected
	}

	// Check if already resolved
	if visited[normalizedID] {
		return h.resolvedLive[normalizedID], h.resolvedDeclared[normalizedID], nil
	}

	resolving[normalizedID] = true
	defer func() { delete(resolving, normalizedID) }()

	req := index.GetRequirement(normalizedID)
	if req == nil {
		visited[normalizedID] = true
		return types.LiveUnknown, types.StatusPending, nil
	}

	// Get children
	children := index.GetChildren(normalizedID)

	if len(children) == 0 {
		// Leaf node - use its own status
		live := req.LiveStatus
		if live == "" {
			live = types.LiveNotRun
		}
		declared := req.Status
		if declared == "" {
			declared = types.StatusPending
		}

		h.resolvedLive[normalizedID] = live
		h.resolvedDeclared[normalizedID] = declared
		visited[normalizedID] = true
		return live, declared, nil
	}

	// Resolve children first
	var childLiveStatuses []types.LiveStatus
	var childDeclaredStatuses []types.DeclaredStatus

	for _, childID := range children {
		childLive, childDeclared, err := h.resolveRequirement(ctx, childID, index, visited, resolving)
		if err != nil {
			continue // Skip cycles
		}
		childLiveStatuses = append(childLiveStatuses, childLive)
		childDeclaredStatuses = append(childDeclaredStatuses, childDeclared)
	}

	// Compute rollup
	var live types.LiveStatus
	var declared types.DeclaredStatus

	if len(childLiveStatuses) > 0 {
		live = types.DeriveLiveRollup(childLiveStatuses)
	} else if req.LiveStatus != "" {
		live = req.LiveStatus
	} else {
		live = types.LiveNotRun
	}

	if len(childDeclaredStatuses) > 0 {
		declared = types.DeriveDeclaredRollup(childDeclaredStatuses)
	} else {
		declared = req.Status
	}

	// Store resolved status
	h.resolvedLive[normalizedID] = live
	h.resolvedDeclared[normalizedID] = declared
	visited[normalizedID] = true

	return live, declared, nil
}

// DetectCycles finds cycles in the requirement hierarchy.
func (h *HierarchyResolver) DetectCycles(index *parsing.ModuleIndex) []CyclePath {
	if index == nil {
		return nil
	}

	var cycles []CyclePath
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for id := range index.ByID {
		if !visited[id] {
			h.detectCyclesDFS(id, index, visited, recStack, nil, &cycles)
		}
	}

	return cycles
}

// CyclePath represents a cycle in the requirement hierarchy.
type CyclePath struct {
	IDs []string
}

// detectCyclesDFS performs DFS to find cycles.
func (h *HierarchyResolver) detectCyclesDFS(
	id string,
	index *parsing.ModuleIndex,
	visited, recStack map[string]bool,
	path []string,
	cycles *[]CyclePath,
) {
	normalizedID := parsing.NormalizeID(id)
	visited[normalizedID] = true
	recStack[normalizedID] = true
	path = append(path, normalizedID)

	for _, childID := range index.GetChildren(normalizedID) {
		normalizedChild := parsing.NormalizeID(childID)
		if !visited[normalizedChild] {
			h.detectCyclesDFS(normalizedChild, index, visited, recStack, path, cycles)
		} else if recStack[normalizedChild] {
			// Found a cycle
			cyclePath := make([]string, 0)
			inCycle := false
			for _, p := range path {
				if p == normalizedChild {
					inCycle = true
				}
				if inCycle {
					cyclePath = append(cyclePath, p)
				}
			}
			cyclePath = append(cyclePath, normalizedChild)
			*cycles = append(*cycles, CyclePath{IDs: cyclePath})
		}
	}

	recStack[normalizedID] = false
}

// GetAncestors returns all ancestors of a requirement.
func (h *HierarchyResolver) GetAncestors(id string, index *parsing.ModuleIndex) []string {
	if index == nil {
		return nil
	}

	var ancestors []string
	current := parsing.NormalizeID(id)
	visited := make(map[string]bool)

	for {
		parent, ok := index.GetParent(current)
		if !ok || visited[parent] {
			break
		}
		visited[parent] = true
		ancestors = append(ancestors, parent)
		current = parent
	}

	return ancestors
}

// GetDescendants returns all descendants of a requirement.
func (h *HierarchyResolver) GetDescendants(id string, index *parsing.ModuleIndex) []string {
	if index == nil {
		return nil
	}

	var descendants []string
	queue := []string{parsing.NormalizeID(id)}
	visited := make(map[string]bool)
	visited[queue[0]] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, childID := range index.GetChildren(current) {
			normalizedChild := parsing.NormalizeID(childID)
			if !visited[normalizedChild] {
				visited[normalizedChild] = true
				descendants = append(descendants, normalizedChild)
				queue = append(queue, normalizedChild)
			}
		}
	}

	return descendants
}

// GetDepth returns the depth of a requirement in the hierarchy (0 for roots).
func (h *HierarchyResolver) GetDepth(id string, index *parsing.ModuleIndex) int {
	ancestors := h.GetAncestors(id, index)
	return len(ancestors)
}
