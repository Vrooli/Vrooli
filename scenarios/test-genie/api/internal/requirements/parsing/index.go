package parsing

import (
	"fmt"

	"test-genie/internal/requirements/types"
)

// ModuleIndex provides fast lookup of requirements by various keys.
type ModuleIndex struct {
	// All parsed modules
	Modules []*types.RequirementModule

	// Lookup maps
	ByID     map[string]*types.Requirement       // requirementID -> Requirement
	ByFile   map[string]*types.RequirementModule // filePath -> Module
	ByModule map[string]*types.RequirementModule // moduleName -> Module

	// Hierarchy maps
	ParentIndex  map[string]string   // childID -> parentID
	ChildrenMap  map[string][]string // parentID -> []childID
	DependsOnMap map[string][]string // reqID -> []dependencyID
	BlocksMap    map[string][]string // reqID -> []blockedID

	// Validation reference index
	ValidationsByRef map[string][]*ValidationRef // ref -> [(requirement, validation)]

	// Parse errors encountered
	Errors []error
}

// ValidationRef links a validation to its parent requirement.
type ValidationRef struct {
	RequirementID string
	Validation    *types.Validation
}

// NewModuleIndex creates an empty ModuleIndex.
func NewModuleIndex() *ModuleIndex {
	return &ModuleIndex{
		Modules:          make([]*types.RequirementModule, 0),
		ByID:             make(map[string]*types.Requirement),
		ByFile:           make(map[string]*types.RequirementModule),
		ByModule:         make(map[string]*types.RequirementModule),
		ParentIndex:      make(map[string]string),
		ChildrenMap:      make(map[string][]string),
		DependsOnMap:     make(map[string][]string),
		BlocksMap:        make(map[string][]string),
		ValidationsByRef: make(map[string][]*ValidationRef),
		Errors:           make([]error, 0),
	}
}

// AddModule adds a module to the index and indexes all its requirements.
func (idx *ModuleIndex) AddModule(module *types.RequirementModule) error {
	if module == nil {
		return nil
	}

	idx.Modules = append(idx.Modules, module)
	idx.ByFile[module.FilePath] = module

	if module.ModuleName != "" {
		idx.ByModule[module.ModuleName] = module
	}

	// Index requirements
	for i := range module.Requirements {
		req := &module.Requirements[i]
		if req.ID == "" {
			idx.Errors = append(idx.Errors, types.NewParseError(
				module.FilePath,
				fmt.Errorf("%w: requirement at index %d", types.ErrMissingID, i),
			))
			continue
		}

		normalizedID := NormalizeID(req.ID)
		if existing, ok := idx.ByID[normalizedID]; ok {
			idx.Errors = append(idx.Errors, fmt.Errorf(
				"%w: '%s' found in both %s and %s",
				types.ErrDuplicateID, req.ID, existing.SourceFile, module.FilePath,
			))
			continue
		}

		idx.ByID[normalizedID] = req

		// Index validations by ref
		for j := range req.Validations {
			val := &req.Validations[j]
			ref := val.Key()
			if ref != "" {
				idx.ValidationsByRef[ref] = append(idx.ValidationsByRef[ref], &ValidationRef{
					RequirementID: req.ID,
					Validation:    val,
				})
			}
		}
	}

	return nil
}

// BuildHierarchy builds parent-child and dependency relationships.
func (idx *ModuleIndex) BuildHierarchy() {
	for _, module := range idx.Modules {
		for _, req := range module.Requirements {
			reqID := NormalizeID(req.ID)

			// Build children map and parent index
			for _, childID := range req.Children {
				normalizedChildID := NormalizeID(childID)
				idx.ChildrenMap[reqID] = append(idx.ChildrenMap[reqID], normalizedChildID)
				idx.ParentIndex[normalizedChildID] = reqID
			}

			// Build depends_on map
			for _, depID := range req.DependsOn {
				normalizedDepID := NormalizeID(depID)
				idx.DependsOnMap[reqID] = append(idx.DependsOnMap[reqID], normalizedDepID)
			}

			// Build blocks map
			for _, blockedID := range req.Blocks {
				normalizedBlockedID := NormalizeID(blockedID)
				idx.BlocksMap[reqID] = append(idx.BlocksMap[reqID], normalizedBlockedID)
			}
		}
	}
}

// GetRequirement retrieves a requirement by ID.
func (idx *ModuleIndex) GetRequirement(id string) *types.Requirement {
	if idx == nil {
		return nil
	}
	return idx.ByID[NormalizeID(id)]
}

// GetModule retrieves a module by file path.
func (idx *ModuleIndex) GetModule(filePath string) *types.RequirementModule {
	if idx == nil {
		return nil
	}
	return idx.ByFile[filePath]
}

// GetParent returns the parent requirement ID for a given requirement.
func (idx *ModuleIndex) GetParent(id string) (string, bool) {
	if idx == nil {
		return "", false
	}
	parent, ok := idx.ParentIndex[NormalizeID(id)]
	return parent, ok
}

// GetChildren returns child requirement IDs for a given requirement.
func (idx *ModuleIndex) GetChildren(id string) []string {
	if idx == nil {
		return nil
	}
	return idx.ChildrenMap[NormalizeID(id)]
}

// GetDependencies returns dependency IDs for a given requirement.
func (idx *ModuleIndex) GetDependencies(id string) []string {
	if idx == nil {
		return nil
	}
	return idx.DependsOnMap[NormalizeID(id)]
}

// GetBlockedBy returns IDs of requirements that this one blocks.
func (idx *ModuleIndex) GetBlockedBy(id string) []string {
	if idx == nil {
		return nil
	}
	return idx.BlocksMap[NormalizeID(id)]
}

// FindValidationsByRef finds all validations that reference a given path.
func (idx *ModuleIndex) FindValidationsByRef(ref string) []*ValidationRef {
	if idx == nil {
		return nil
	}
	return idx.ValidationsByRef[NormalizeRef(ref)]
}

// AllRequirements returns all requirements across all modules.
func (idx *ModuleIndex) AllRequirements() []*types.Requirement {
	if idx == nil {
		return nil
	}

	result := make([]*types.Requirement, 0, len(idx.ByID))
	for _, req := range idx.ByID {
		result = append(result, req)
	}
	return result
}

// AllRequirementIDs returns all requirement IDs.
func (idx *ModuleIndex) AllRequirementIDs() []string {
	if idx == nil {
		return nil
	}

	ids := make([]string, 0, len(idx.ByID))
	for id := range idx.ByID {
		ids = append(ids, id)
	}
	return ids
}

// RequirementCount returns the total number of requirements.
func (idx *ModuleIndex) RequirementCount() int {
	if idx == nil {
		return 0
	}
	return len(idx.ByID)
}

// ModuleCount returns the number of parsed modules.
func (idx *ModuleIndex) ModuleCount() int {
	if idx == nil {
		return 0
	}
	return len(idx.Modules)
}

// HasErrors returns true if any parsing errors occurred.
func (idx *ModuleIndex) HasErrors() bool {
	return idx != nil && len(idx.Errors) > 0
}

// IsAncestor checks if ancestorID is an ancestor of descendantID.
func (idx *ModuleIndex) IsAncestor(ancestorID, descendantID string) bool {
	if idx == nil {
		return false
	}

	normalizedAncestor := NormalizeID(ancestorID)
	current := NormalizeID(descendantID)
	visited := make(map[string]bool)

	for {
		if current == normalizedAncestor {
			return true
		}
		if visited[current] {
			return false // Cycle detected
		}
		visited[current] = true

		parent, ok := idx.ParentIndex[current]
		if !ok {
			return false
		}
		current = parent
	}
}

// GetRootRequirements returns requirements that have no parent.
func (idx *ModuleIndex) GetRootRequirements() []*types.Requirement {
	if idx == nil {
		return nil
	}

	var roots []*types.Requirement
	for id, req := range idx.ByID {
		if _, hasParent := idx.ParentIndex[id]; !hasParent {
			roots = append(roots, req)
		}
	}
	return roots
}

// GetLeafRequirements returns requirements that have no children.
func (idx *ModuleIndex) GetLeafRequirements() []*types.Requirement {
	if idx == nil {
		return nil
	}

	var leaves []*types.Requirement
	for id, req := range idx.ByID {
		if children := idx.ChildrenMap[id]; len(children) == 0 {
			leaves = append(leaves, req)
		}
	}
	return leaves
}

// DetectCycles finds any cycles in the requirement hierarchy.
// Returns a list of cycle paths if found.
func (idx *ModuleIndex) DetectCycles() [][]string {
	if idx == nil {
		return nil
	}

	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var detectCycle func(id string, path []string) bool
	detectCycle = func(id string, path []string) bool {
		visited[id] = true
		recStack[id] = true
		path = append(path, id)

		for _, childID := range idx.ChildrenMap[id] {
			if !visited[childID] {
				if detectCycle(childID, path) {
					return true
				}
			} else if recStack[childID] {
				// Found a cycle
				cyclePath := append(path, childID)
				cycles = append(cycles, cyclePath)
				return true
			}
		}

		recStack[id] = false
		return false
	}

	for id := range idx.ByID {
		if !visited[id] {
			detectCycle(id, nil)
		}
	}

	return cycles
}

// FilterByStatus returns requirements matching the given status.
func (idx *ModuleIndex) FilterByStatus(status types.DeclaredStatus) []*types.Requirement {
	if idx == nil {
		return nil
	}

	var result []*types.Requirement
	for _, req := range idx.ByID {
		if req.Status == status {
			result = append(result, req)
		}
	}
	return result
}

// FilterByCriticality returns requirements matching the given criticality.
func (idx *ModuleIndex) FilterByCriticality(criticality types.Criticality) []*types.Requirement {
	if idx == nil {
		return nil
	}

	var result []*types.Requirement
	for _, req := range idx.ByID {
		if req.Criticality == criticality {
			result = append(result, req)
		}
	}
	return result
}

// FilterCritical returns P0 and P1 requirements.
func (idx *ModuleIndex) FilterCritical() []*types.Requirement {
	if idx == nil {
		return nil
	}

	var result []*types.Requirement
	for _, req := range idx.ByID {
		if req.IsCriticalReq() {
			result = append(result, req)
		}
	}
	return result
}
