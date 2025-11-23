package autosteer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// UXMetricsCollector handles collection of UX-specific metrics
type UXMetricsCollector struct {
	projectRoot string
}

// NewUXMetricsCollector creates a new UX metrics collector
func NewUXMetricsCollector(projectRoot string) *UXMetricsCollector {
	return &UXMetricsCollector{
		projectRoot: projectRoot,
	}
}

// Collect gathers all UX metrics for a scenario
func (c *UXMetricsCollector) Collect(scenarioName string) (*UXMetrics, error) {
	metrics := &UXMetrics{}

	// Collect each metric independently (best effort)
	metrics.AccessibilityScore = c.getAccessibilityScore(scenarioName)
	metrics.UITestCoverage = c.getUITestCoverage(scenarioName)
	metrics.ResponsiveBreakpoints = c.countResponsiveBreakpoints(scenarioName)
	metrics.UserFlowsImplemented = c.countUserFlows(scenarioName)
	metrics.LoadingStatesCount = c.countLoadingStates(scenarioName)
	metrics.ErrorHandlingCoverage = c.assessErrorHandling(scenarioName)

	return metrics, nil
}

// getAccessibilityScore runs axe-core accessibility audit
func (c *UXMetricsCollector) getAccessibilityScore(scenarioName string) float64 {
	// Check if UI exists
	uiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui")
	if _, err := os.Stat(uiPath); os.IsNotExist(err) {
		return 0
	}

	// For now, we'd need the UI to be running to test accessibility
	// This would require integration with browser-automation-studio
	// Placeholder: scan for basic accessibility attributes in code

	srcPath := filepath.Join(uiPath, "src")
	score := c.estimateAccessibilityFromCode(srcPath)

	return score
}

// estimateAccessibilityFromCode provides a rough accessibility score based on code analysis
func (c *UXMetricsCollector) estimateAccessibilityFromCode(srcPath string) float64 {
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return 0
	}

	// Count components with accessibility attributes
	totalComponents := 0
	accessibleComponents := 0

	// Walk through all TSX/JSX files
	filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Check TSX and JSX files
		ext := filepath.Ext(path)
		if ext != ".tsx" && ext != ".jsx" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Count component definitions
		componentRegex := regexp.MustCompile(`(?:function|const)\s+\w+.*(?:React\.FC|=>)`)
		components := componentRegex.FindAllString(fileContent, -1)
		totalComponents += len(components)

		// Check for accessibility attributes
		hasARIA := strings.Contains(fileContent, "aria-")
		hasRole := strings.Contains(fileContent, "role=")
		hasAlt := strings.Contains(fileContent, "alt=")
		hasLabel := strings.Contains(fileContent, "aria-label")

		if hasARIA || hasRole || hasAlt || hasLabel {
			accessibleComponents++
		}

		return nil
	})

	if totalComponents == 0 {
		return 50.0 // Default score for scenarios without clear component structure
	}

	// Calculate score (0-100)
	score := (float64(accessibleComponents) / float64(totalComponents)) * 100

	// Cap at reasonable levels since this is just an estimate
	if score > 85 {
		score = 85 // Actual testing would be needed for higher scores
	}

	return score
}

// getUITestCoverage retrieves UI test coverage from coverage reports
func (c *UXMetricsCollector) getUITestCoverage(scenarioName string) float64 {
	coveragePath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "coverage", "coverage-summary.json")

	data, err := os.ReadFile(coveragePath)
	if err != nil {
		return 0
	}

	var summary struct {
		Total struct {
			Lines struct {
				Pct float64 `json:"pct"`
			} `json:"lines"`
		} `json:"total"`
	}

	if err := json.Unmarshal(data, &summary); err != nil {
		return 0
	}

	return summary.Total.Lines.Pct
}

// countResponsiveBreakpoints counts unique responsive breakpoints in the UI
func (c *UXMetricsCollector) countResponsiveBreakpoints(scenarioName string) int {
	srcPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "src")

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return 0
	}

	breakpointSet := make(map[string]bool)

	// Walk through CSS, SCSS, and styled-component files
	filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".css" && ext != ".scss" && ext != ".tsx" && ext != ".jsx" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Find @media queries
		mediaRegex := regexp.MustCompile(`@media\s*\([^)]*\)`)
		matches := mediaRegex.FindAllString(string(content), -1)

		for _, match := range matches {
			// Extract breakpoint value
			breakpointSet[match] = true
		}

		return nil
	})

	return len(breakpointSet)
}

// countUserFlows counts implemented user flows from test playbooks
func (c *UXMetricsCollector) countUserFlows(scenarioName string) int {
	playbooksPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "test", "playbooks")

	if _, err := os.Stat(playbooksPath); os.IsNotExist(err) {
		return 0
	}

	// Check registry.json for playbook count
	registryPath := filepath.Join(playbooksPath, "registry.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		// Fallback: count .json files in playbooks directory
		return c.countPlaybookFiles(playbooksPath)
	}

	var registry struct {
		Playbooks []interface{} `json:"playbooks"`
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		return c.countPlaybookFiles(playbooksPath)
	}

	return len(registry.Playbooks)
}

// countPlaybookFiles counts playbook JSON files
func (c *UXMetricsCollector) countPlaybookFiles(playbooksPath string) int {
	count := 0

	filepath.Walk(playbooksPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".json" && filepath.Base(path) != "registry.json" {
			count++
		}

		return nil
	})

	return count
}

// countLoadingStates counts loading state implementations in the UI
func (c *UXMetricsCollector) countLoadingStates(scenarioName string) int {
	srcPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "src")

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return 0
	}

	loadingStateCount := 0

	filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".tsx" && ext != ".jsx" && ext != ".ts" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Look for common loading state patterns
		patterns := []string{
			"isLoading",
			"loading",
			"pending",
			"<Spinner",
			"<Loading",
			"<CircularProgress",
			"data-loading",
		}

		for _, pattern := range patterns {
			if strings.Contains(fileContent, pattern) {
				loadingStateCount++
				break // Count once per file
			}
		}

		return nil
	})

	return loadingStateCount
}

// assessErrorHandling estimates error handling coverage
func (c *UXMetricsCollector) assessErrorHandling(scenarioName string) float64 {
	srcPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "src")

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return 0
	}

	totalAsyncOps := 0
	handledErrors := 0

	filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".tsx" && ext != ".jsx" && ext != ".ts" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Count async operations (fetch, axios, etc.)
		asyncPatterns := []string{
			"fetch(",
			"axios.",
			"async ",
			".then(",
		}

		for _, pattern := range asyncPatterns {
			totalAsyncOps += strings.Count(fileContent, pattern)
		}

		// Count error handling patterns
		errorPatterns := []string{
			".catch(",
			"try {",
			"} catch",
			"onError",
			"error:",
		}

		for _, pattern := range errorPatterns {
			handledErrors += strings.Count(fileContent, pattern)
		}

		return nil
	})

	if totalAsyncOps == 0 {
		return 100.0 // No async ops means no error handling needed
	}

	// Calculate coverage percentage
	coverage := (float64(handledErrors) / float64(totalAsyncOps)) * 100

	// Cap at 100%
	if coverage > 100 {
		coverage = 100
	}

	return coverage
}

// RunAxeAccessibility runs axe-core against a running application (requires URL)
func (c *UXMetricsCollector) RunAxeAccessibility(url string) (float64, error) {
	// Check if axe-cli is installed
	if _, err := exec.LookPath("axe"); err != nil {
		return 0, fmt.Errorf("axe-cli not installed: %w", err)
	}

	// Run axe-core accessibility audit
	cmd := exec.Command("axe", url, "--stdout")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// axe returns non-zero if violations found, but output is still valid
		// Continue processing
	}

	var result struct {
		Violations []struct {
			ID     string `json:"id"`
			Impact string `json:"impact"`
		} `json:"violations"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("failed to parse axe output: %w", err)
	}

	// Weight violations by impact
	weightedViolations := 0.0
	for _, violation := range result.Violations {
		switch violation.Impact {
		case "critical":
			weightedViolations += 3.0
		case "serious":
			weightedViolations += 2.0
		case "moderate":
			weightedViolations += 1.0
		case "minor":
			weightedViolations += 0.5
		}
	}

	// Score calculation: start at 100, deduct points for violations
	score := 100.0 - (weightedViolations * 2.0)

	if score < 0 {
		score = 0
	}

	return score, nil
}
