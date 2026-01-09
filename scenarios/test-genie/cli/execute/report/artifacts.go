package report

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	execTypes "test-genie/cli/internal/execute"
	"test-genie/cli/internal/repo"
)

// CollectArtifactRoots gathers unique artifact directories from phase logs.
func CollectArtifactRoots(phases []execTypes.Phase) []string {
	roots := make(map[string]struct{})
	for _, phase := range phases {
		if phase.LogPath == "" {
			continue
		}
		dir := filepath.Dir(phase.LogPath)
		if dir == "." || dir == "/" {
			continue
		}
		roots[dir] = struct{}{}
	}
	result := make([]string, 0, len(roots))
	for dir := range roots {
		result = append(result, dir)
	}
	sort.Strings(result)
	return result
}

// DescribeArtifacts generates artifact summary lines.
func DescribeArtifacts(phases []execTypes.Phase) []string {
	roots := CollectArtifactRoots(phases)
	if len(roots) == 0 {
		return nil
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("artifact roots: %s", strings.Join(roots, ", ")))

	totalLogs := 0
	presentLogs := 0
	emptyLogs := 0
	warningCount := 0
	for _, phase := range phases {
		if phase.LogPath == "" {
			continue
		}
		totalLogs++
		exists, empty := repo.FileState(phase.LogPath)
		if exists {
			presentLogs++
		}
		if empty {
			emptyLogs++
		}
		if exists && !empty {
			warningCount += CountLinesContaining(phase.LogPath, "[WARNING")
		}
	}
	if totalLogs > 0 {
		lines = append(lines, fmt.Sprintf("logs: %d total • %d present • %d empty • %d warnings", totalLogs, presentLogs, emptyLogs, warningCount))
	}
	return lines
}

// phaseRan checks if a phase name was executed.
func phaseRan(phases []execTypes.Phase, name string) bool {
	for _, p := range phases {
		if strings.EqualFold(p.Name, name) {
			return true
		}
	}
	return false
}

// DescribeCoverage finds and describes coverage artifacts for phases that ran.
// Only shows artifacts relevant to the phases that were actually executed.
func DescribeCoverage(scenario string, phases []execTypes.Phase) []string {
	paths := repo.DiscoverScenarioPaths(scenario)
	if paths.ScenarioDir == "" {
		return nil
	}
	var lines []string

	// Unit coverage - only show if unit phase ran
	if phaseRan(phases, "unit") {
		coverageDirs := []string{
			filepath.Join(paths.ScenarioDir, "coverage", "unit", "go"),
			filepath.Join(paths.ScenarioDir, "coverage", "integration"),
			filepath.Join(paths.ScenarioDir, "coverage", "test-genie"),
		}
		for _, dir := range coverageDirs {
			if repo.Exists(dir) {
				lines = append(lines, fmt.Sprintf("coverage: %s", dir))
			}
		}
		goHTML := filepath.Join(paths.ScenarioDir, "coverage", "unit", "go", "coverage.html")
		if repo.Exists(goHTML) {
			lines = append(lines, fmt.Sprintf("go coverage html: %s", goHTML))
		}
		nodeHTML := filepath.Join(paths.ScenarioDir, "ui", "coverage", "lcov-report", "index.html")
		if repo.Exists(nodeHTML) {
			lines = append(lines, fmt.Sprintf("node coverage html: %s", nodeHTML))
		}
	}

	// Lighthouse - only show if performance phase ran
	if phaseRan(phases, "performance") {
		lighthouse := filepath.Join(paths.ScenarioDir, "coverage", "lighthouse")
		if repo.Exists(lighthouse) {
			lines = append(lines, fmt.Sprintf("lighthouse: %s", lighthouse))
		}
	}

	// Requirements index - always relevant (used by business phase but useful context)
	if phaseRan(phases, "business") || phaseRan(phases, "structure") {
		reqIndex := filepath.Join(paths.ScenarioDir, "requirements", "index.json")
		if repo.Exists(reqIndex) {
			lines = append(lines, fmt.Sprintf("requirements index: %s", reqIndex))
		}
	}

	return lines
}
