package collectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"scenario-completeness-scoring/pkg/scoring"
)

// loadRequirements loads requirements from index.json and imported modules
// [REQ:SCS-CORE-001] Requirements data loading
func loadRequirements(scenarioRoot string) []Requirement {
	requirementsDir := filepath.Join(scenarioRoot, "requirements")

	if _, err := os.Stat(requirementsDir); os.IsNotExist(err) {
		return []Requirement{}
	}

	requirements := []Requirement{}
	loadedFiles := make(map[string]bool)

	// Try to load from index.json first
	indexPath := filepath.Join(requirementsDir, "index.json")
	if _, err := os.Stat(indexPath); err == nil {
		data, err := os.ReadFile(indexPath)
		if err == nil {
			var indexData RequirementsData
			if json.Unmarshal(data, &indexData) == nil {
				requirements = append(requirements, indexData.Requirements...)
				loadedFiles[indexPath] = true

				// Load imports
				for _, importPath := range indexData.Imports {
					fullPath := filepath.Join(requirementsDir, importPath)
					reqs := loadFromFile(fullPath, loadedFiles)
					requirements = append(requirements, reqs...)
				}
			}
		}
	}

	// Also scan for module.json files
	moduleReqs := scanModules(requirementsDir, loadedFiles)
	requirements = append(requirements, moduleReqs...)

	return requirements
}

// loadFromFile loads requirements from a single JSON file
func loadFromFile(filePath string, loadedFiles map[string]bool) []Requirement {
	if loadedFiles[filePath] {
		return []Requirement{}
	}
	loadedFiles[filePath] = true

	data, err := os.ReadFile(filePath)
	if err != nil {
		return []Requirement{}
	}

	var reqData RequirementsData
	if err := json.Unmarshal(data, &reqData); err != nil {
		return []Requirement{}
	}

	return reqData.Requirements
}

// scanModules recursively scans for module.json files
func scanModules(dir string, loadedFiles map[string]bool) []Requirement {
	requirements := []Requirement{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return requirements
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			requirements = append(requirements, scanModules(fullPath, loadedFiles)...)
		} else if entry.Name() == "module.json" {
			reqs := loadFromFile(fullPath, loadedFiles)
			requirements = append(requirements, reqs...)
		}
	}

	return requirements
}

// loadSyncMetadata loads requirements sync metadata
func loadSyncMetadata(scenarioRoot string) *SyncMetadata {
	syncPath := filepath.Join(scenarioRoot, "coverage", "requirements-sync.json")

	data, err := os.ReadFile(syncPath)
	if err != nil {
		return nil
	}

	var syncData SyncMetadata
	if err := json.Unmarshal(data, &syncData); err != nil {
		return nil
	}

	return &syncData
}

// calculateRequirementPass calculates requirement pass rate
// [REQ:SCS-CORE-001A] Requirement pass rate calculation
func calculateRequirementPass(requirements []Requirement, syncData *SyncMetadata) PassMetrics {
	if len(requirements) == 0 {
		return PassMetrics{Total: 0, Passing: 0}
	}

	total := 0
	passing := 0

	for _, req := range requirements {
		// Skip requirements that only exist to group children
		if len(req.Children) > 0 && req.Status == "" {
			continue
		}

		total++

		// Check sync metadata first
		if syncData != nil {
			if synced, ok := syncData.Requirements[req.ID]; ok {
				if synced.Status == "passed" || synced.Status == "complete" {
					passing++
					continue
				}
			}
		}

		// Fall back to requirement status
		status := strings.ToLower(req.Status)
		if status == "passed" || status == "complete" || status == "done" {
			passing++
		}
	}

	return PassMetrics{Total: total, Passing: passing}
}

// calculateTargetPass calculates operational target pass rate
// [REQ:SCS-CORE-001A] Target pass rate calculation
func calculateTargetPass(requirements []Requirement, syncData *SyncMetadata) PassMetrics {
	// Build map of targets to their requirements
	targetReqs := make(map[string][]Requirement)

	for _, req := range requirements {
		targetID := req.OperationalTargetID
		if targetID == "" {
			// Try to infer from ID prefix (e.g., SCS-CORE -> OT-P0-001)
			continue
		}
		targetReqs[targetID] = append(targetReqs[targetID], req)
	}

	if len(targetReqs) == 0 {
		// Fall back to counting P0/P1 requirements as targets
		return countPriorityRequirements(requirements, syncData)
	}

	total := len(targetReqs)
	passing := 0

	for _, reqs := range targetReqs {
		allPassed := true
		for _, req := range reqs {
			status := strings.ToLower(req.Status)
			if status != "passed" && status != "complete" && status != "done" {
				if syncData != nil {
					if synced, ok := syncData.Requirements[req.ID]; ok {
						if synced.Status != "passed" && synced.Status != "complete" {
							allPassed = false
							break
						}
					} else {
						allPassed = false
						break
					}
				} else {
					allPassed = false
					break
				}
			}
		}
		if allPassed {
			passing++
		}
	}

	return PassMetrics{Total: total, Passing: passing}
}

// countPriorityRequirements counts P0/P1 requirements as proxy for targets
func countPriorityRequirements(requirements []Requirement, syncData *SyncMetadata) PassMetrics {
	total := 0
	passing := 0

	for _, req := range requirements {
		priority := strings.ToUpper(req.Priority)
		if priority == "P0" || priority == "P1" || priority == "" {
			total++
			status := strings.ToLower(req.Status)
			if status == "passed" || status == "complete" || status == "done" {
				passing++
			} else if syncData != nil {
				if synced, ok := syncData.Requirements[req.ID]; ok {
					if synced.Status == "passed" || synced.Status == "complete" {
						passing++
					}
				}
			}
		}
	}

	return PassMetrics{Total: total, Passing: passing}
}

// buildRequirementTrees builds requirement trees for depth calculation
// [REQ:SCS-CORE-001B] Requirement depth structure
func buildRequirementTrees(requirements []Requirement) []scoring.RequirementTree {
	// Build map of ID to children
	childMap := make(map[string][]string)
	for _, req := range requirements {
		for _, childID := range req.Children {
			childMap[req.ID] = append(childMap[req.ID], childID)
		}
	}

	// Find root requirements (no parent)
	hasParent := make(map[string]bool)
	for _, req := range requirements {
		for _, childID := range req.Children {
			hasParent[childID] = true
		}
	}

	var roots []scoring.RequirementTree
	for _, req := range requirements {
		if !hasParent[req.ID] {
			roots = append(roots, buildTree(req.ID, childMap))
		}
	}

	return roots
}

func buildTree(id string, childMap map[string][]string) scoring.RequirementTree {
	tree := scoring.RequirementTree{ID: id}
	for _, childID := range childMap[id] {
		tree.Children = append(tree.Children, buildTree(childID, childMap))
	}
	return tree
}
