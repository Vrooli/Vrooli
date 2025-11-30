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
// Tries multiple locations to match JS behavior
func loadSyncMetadata(scenarioRoot string) *SyncMetadata {
	// Try primary location: coverage/requirements-sync/latest.json
	syncPaths := []string{
		filepath.Join(scenarioRoot, "coverage", "requirements-sync", "latest.json"),
		filepath.Join(scenarioRoot, "coverage", "requirements-sync.json"),
		filepath.Join(scenarioRoot, "coverage", "sync", "latest.json"),
	}

	for _, syncPath := range syncPaths {
		data, err := os.ReadFile(syncPath)
		if err != nil {
			continue
		}

		var syncData SyncMetadata
		if err := json.Unmarshal(data, &syncData); err != nil {
			continue
		}

		return &syncData
	}

	return nil
}

// calculateRequirementPass calculates requirement pass rate
// [REQ:SCS-CORE-001A] Requirement pass rate calculation
func calculateRequirementPass(requirements []Requirement, syncData *SyncMetadata) scoring.MetricCounts {
	if len(requirements) == 0 {
		return scoring.MetricCounts{Total: 0, Passing: 0}
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

	return scoring.MetricCounts{Total: total, Passing: passing}
}

// calculateTargetPass calculates operational target pass rate
// [REQ:SCS-CORE-001A] Target pass rate calculation
// Uses sync metadata operational_targets when available (matches JS behavior)
func calculateTargetPass(requirements []Requirement, syncData *SyncMetadata) scoring.MetricCounts {
	// First, check if sync metadata has operational_targets (preferred source)
	if syncData != nil && len(syncData.OperationalTargets) > 0 {
		return calculateTargetPassFromSyncMetadata(syncData)
	}

	// Fall back to building targets from requirements
	targetReqs := make(map[string][]Requirement)

	for _, req := range requirements {
		targetID := req.OperationalTargetID
		if targetID == "" {
			// Also try prd_ref field (matches JS behavior)
			targetID = extractTargetFromPrdRef(req.PRDRef)
		}
		if targetID == "" {
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
		// Target passes if at least one linked requirement passes (matches JS behavior)
		anyPassed := false
		for _, req := range reqs {
			status := strings.ToLower(req.Status)
			if status == "passed" || status == "complete" || status == "done" {
				anyPassed = true
				break
			}
			if syncData != nil {
				if synced, ok := syncData.Requirements[req.ID]; ok {
					if synced.Status == "passed" || synced.Status == "complete" {
						anyPassed = true
						break
					}
				}
			}
		}
		if anyPassed {
			passing++
		}
	}

	return scoring.MetricCounts{Total: total, Passing: passing}
}

// calculateTargetPassFromSyncMetadata uses pre-computed operational targets from sync metadata
// This matches the JS behavior which prioritizes sync metadata operational_targets
func calculateTargetPassFromSyncMetadata(syncData *SyncMetadata) scoring.MetricCounts {
	total := len(syncData.OperationalTargets)
	passing := 0

	// TARGET_PASS_THRESHOLD from JS constants.js: 0.5 (50%)
	const targetPassThreshold = 0.5

	for _, target := range syncData.OperationalTargets {
		// Check if target status is explicitly complete
		status := strings.ToLower(target.Status)
		if status == "complete" || status == "passed" {
			passing++
			continue
		}

		// Check counts if available (matches JS isFolderTargetPassing)
		if target.Counts != nil && target.Counts.Total > 0 {
			completeRatio := float64(target.Counts.Complete) / float64(target.Counts.Total)
			if completeRatio > targetPassThreshold {
				passing++
			}
		}
	}

	return scoring.MetricCounts{Total: total, Passing: passing}
}

// extractTargetFromPrdRef extracts OT-P0-001 style target IDs from prd_ref field
func extractTargetFromPrdRef(prdRef string) string {
	if prdRef == "" {
		return ""
	}
	// Match OT-P0-001, OT-P1-002, etc. pattern
	prdRef = strings.ToUpper(prdRef)
	if strings.HasPrefix(prdRef, "OT-P") {
		return prdRef
	}
	return ""
}

// countPriorityRequirements counts P0/P1 requirements as proxy for targets
func countPriorityRequirements(requirements []Requirement, syncData *SyncMetadata) scoring.MetricCounts {
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

	return scoring.MetricCounts{Total: total, Passing: passing}
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
