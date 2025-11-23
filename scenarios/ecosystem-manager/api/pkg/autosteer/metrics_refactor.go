package autosteer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RefactorMetricsCollector handles collection of code quality/refactoring metrics
type RefactorMetricsCollector struct {
	projectRoot        string
	tidinessManagerURL string
}

// NewRefactorMetricsCollector creates a new refactor metrics collector
func NewRefactorMetricsCollector(projectRoot string, tidinessManagerURL string) *RefactorMetricsCollector {
	return &RefactorMetricsCollector{
		projectRoot:        projectRoot,
		tidinessManagerURL: tidinessManagerURL,
	}
}

// Collect gathers all refactor/quality metrics for a scenario
func (c *RefactorMetricsCollector) Collect(scenarioName string) (*RefactorMetrics, error) {
	metrics := &RefactorMetrics{}

	// Collect each metric independently (best effort)
	metrics.CyclomaticComplexityAvg = c.calculateComplexity(scenarioName)
	metrics.DuplicationPercentage = c.calculateDuplication(scenarioName)
	metrics.TidinessScore = c.getTidinessScore(scenarioName)
	metrics.StandardsViolations = c.getStandardsViolations(scenarioName)
	metrics.TechDebtItems = c.countTechDebt(scenarioName)

	return metrics, nil
}

// getTidinessScore retrieves tidiness score from tidiness-manager service
func (c *RefactorMetricsCollector) getTidinessScore(scenarioName string) float64 {
	if c.tidinessManagerURL == "" {
		// If tidiness-manager is not available, fallback to local analysis
		return c.estimateTidinessLocally(scenarioName)
	}

	// Call tidiness-manager API
	url := fmt.Sprintf("%s/api/scan/%s", c.tidinessManagerURL, scenarioName)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		// Fallback to local analysis if service is unavailable
		return c.estimateTidinessLocally(scenarioName)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.estimateTidinessLocally(scenarioName)
	}

	var result struct {
		Score      float64 `json:"score"`
		Violations int     `json:"violations"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.estimateTidinessLocally(scenarioName)
	}

	return result.Score
}

// estimateTidinessLocally provides a rough tidiness estimate without tidiness-manager
func (c *RefactorMetricsCollector) estimateTidinessLocally(scenarioName string) float64 {
	scenarioPath := filepath.Join(c.projectRoot, "scenarios", scenarioName)

	score := 100.0

	// Check for common tidiness issues
	// 1. Unused files
	unusedCount := c.countUnusedFiles(scenarioPath)
	score -= float64(unusedCount) * 2.0

	// 2. TODO/FIXME comments
	todoCount := c.countTODOs(scenarioPath)
	score -= float64(todoCount) * 0.5

	// 3. Large files (> 500 lines)
	largeFileCount := c.countLargeFiles(scenarioPath)
	score -= float64(largeFileCount) * 3.0

	// 4. Deep nesting (> 4 levels)
	deepNestingCount := c.countDeepNesting(scenarioPath)
	score -= float64(deepNestingCount) * 1.5

	if score < 0 {
		score = 0
	}

	return score
}

// getStandardsViolations counts standards violations
func (c *RefactorMetricsCollector) getStandardsViolations(scenarioName string) int {
	violations := 0

	// Run linters for both Go and TypeScript/JavaScript
	violations += c.runGoLinter(scenarioName)
	violations += c.runESLint(scenarioName)

	return violations
}

// runGoLinter runs golangci-lint for Go code
func (c *RefactorMetricsCollector) runGoLinter(scenarioName string) int {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		return 0
	}

	// Check if golangci-lint is available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		return 0
	}

	// Run golangci-lint
	cmd := exec.Command("golangci-lint", "run", "--out-format", "json")
	cmd.Dir = apiPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// golangci-lint returns non-zero if issues found
		// Continue parsing output
	}

	var result struct {
		Issues []struct {
			Text string `json:"text"`
		} `json:"Issues"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return 0
	}

	return len(result.Issues)
}

// runESLint runs ESLint for TypeScript/JavaScript code
func (c *RefactorMetricsCollector) runESLint(scenarioName string) int {
	uiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui")

	if _, err := os.Stat(uiPath); os.IsNotExist(err) {
		return 0
	}

	// Check if eslint is available in node_modules
	eslintPath := filepath.Join(uiPath, "node_modules", ".bin", "eslint")
	if _, err := os.Stat(eslintPath); os.IsNotExist(err) {
		return 0
	}

	// Run ESLint
	cmd := exec.Command(eslintPath, "src", "--format", "json")
	cmd.Dir = uiPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// ESLint returns non-zero if issues found
		// Continue parsing output
	}

	var results []struct {
		Messages []struct {
			Message string `json:"message"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(output, &results); err != nil {
		return 0
	}

	count := 0
	for _, result := range results {
		count += len(result.Messages)
	}

	return count
}

// calculateComplexity calculates average cyclomatic complexity
func (c *RefactorMetricsCollector) calculateComplexity(scenarioName string) float64 {
	// Calculate for both Go and TypeScript
	goComplexity := c.calculateGoComplexity(scenarioName)
	tsComplexity := c.calculateTSComplexity(scenarioName)

	// Average the two (if both exist)
	if goComplexity > 0 && tsComplexity > 0 {
		return (goComplexity + tsComplexity) / 2.0
	} else if goComplexity > 0 {
		return goComplexity
	} else if tsComplexity > 0 {
		return tsComplexity
	}

	return 0
}

// calculateGoComplexity calculates Go code complexity using gocyclo
func (c *RefactorMetricsCollector) calculateGoComplexity(scenarioName string) float64 {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		return 0
	}

	// Check if gocyclo is available
	if _, err := exec.LookPath("gocyclo"); err != nil {
		// Fallback to simple estimation
		return c.estimateGoComplexity(apiPath)
	}

	// Run gocyclo with average flag
	cmd := exec.Command("gocyclo", "-avg", ".")
	cmd.Dir = apiPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.estimateGoComplexity(apiPath)
	}

	// Parse output: "Average: X.XX"
	outputStr := string(output)
	re := regexp.MustCompile(`Average:\s+([\d.]+)`)
	matches := re.FindStringSubmatch(outputStr)

	if len(matches) < 2 {
		return c.estimateGoComplexity(apiPath)
	}

	complexity, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return c.estimateGoComplexity(apiPath)
	}

	return complexity
}

// estimateGoComplexity provides a rough complexity estimate
func (c *RefactorMetricsCollector) estimateGoComplexity(apiPath string) float64 {
	totalComplexity := 0
	functionCount := 0

	filepath.Walk(apiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Count functions
		funcRegex := regexp.MustCompile(`func\s+(?:\([^)]*\)\s+)?\w+`)
		functions := funcRegex.FindAllString(fileContent, -1)
		functionCount += len(functions)

		// Simple complexity estimation: count control flow statements
		controlFlow := []string{"if ", "for ", "switch ", "case ", "||", "&&"}
		for _, cf := range controlFlow {
			totalComplexity += strings.Count(fileContent, cf)
		}

		return nil
	})

	if functionCount == 0 {
		return 0
	}

	return float64(totalComplexity) / float64(functionCount)
}

// calculateTSComplexity calculates TypeScript complexity
func (c *RefactorMetricsCollector) calculateTSComplexity(scenarioName string) float64 {
	srcPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "src")

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return 0
	}

	// Simple complexity estimation for TypeScript
	totalComplexity := 0
	functionCount := 0

	filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".tsx" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Count functions/components
		funcRegex := regexp.MustCompile(`(?:function|const)\s+\w+\s*(?:\([^)]*\)|=)`)
		functions := funcRegex.FindAllString(fileContent, -1)
		functionCount += len(functions)

		// Count control flow
		controlFlow := []string{"if (", "for (", "while (", "switch (", "case ", "||", "&&", "? ", ": "}
		for _, cf := range controlFlow {
			totalComplexity += strings.Count(fileContent, cf)
		}

		return nil
	})

	if functionCount == 0 {
		return 0
	}

	return float64(totalComplexity) / float64(functionCount)
}

// calculateDuplication calculates code duplication percentage
func (c *RefactorMetricsCollector) calculateDuplication(scenarioName string) float64 {
	scenarioPath := filepath.Join(c.projectRoot, "scenarios", scenarioName)

	// Check if jscpd is available for JavaScript/TypeScript
	jscpdPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "node_modules", ".bin", "jscpd")
	if _, err := os.Stat(jscpdPath); err == nil {
		return c.runJSCPD(scenarioName, jscpdPath)
	}

	// Fallback to simple duplication detection
	return c.estimateDuplication(scenarioPath)
}

// runJSCPD runs jscpd for duplication detection
func (c *RefactorMetricsCollector) runJSCPD(scenarioName string, jscpdPath string) float64 {
	uiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui")

	cmd := exec.Command(jscpdPath, "src", "--format", "json", "--output", ".jscpd-report")
	cmd.Dir = uiPath

	if err := cmd.Run(); err != nil {
		return c.estimateDuplication(uiPath)
	}

	// Read jscpd report
	reportPath := filepath.Join(uiPath, ".jscpd-report", "jscpd-report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return 0
	}

	var report struct {
		Statistics struct {
			Total struct {
				Percentage float64 `json:"percentage"`
			} `json:"total"`
		} `json:"statistics"`
	}

	if err := json.Unmarshal(data, &report); err != nil {
		return 0
	}

	return report.Statistics.Total.Percentage
}

// estimateDuplication provides a rough duplication estimate
func (c *RefactorMetricsCollector) estimateDuplication(scenarioPath string) float64 {
	// Simple hash-based duplication detection
	// This is a placeholder - real implementation would use AST-based comparison
	return 0
}

// countTechDebt counts technical debt items (TODOs, FIXMEs, HACK comments)
func (c *RefactorMetricsCollector) countTechDebt(scenarioName string) int {
	scenarioPath := filepath.Join(c.projectRoot, "scenarios", scenarioName)
	return c.countTODOs(scenarioPath)
}

// countTODOs counts TODO/FIXME/HACK comments
func (c *RefactorMetricsCollector) countTODOs(scenarioPath string) int {
	count := 0

	filepath.Walk(scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		validExts := map[string]bool{
			".go": true, ".ts": true, ".tsx": true, ".js": true,
			".jsx": true, ".py": true, ".sh": true,
		}

		if !validExts[ext] {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Count tech debt markers
		markers := []string{"TODO", "FIXME", "HACK", "XXX", "REFACTOR"}
		for _, marker := range markers {
			count += strings.Count(fileContent, marker)
		}

		return nil
	})

	return count
}

// countUnusedFiles counts potentially unused files
func (c *RefactorMetricsCollector) countUnusedFiles(scenarioPath string) int {
	// This is a simplified implementation
	// Real implementation would need dependency analysis
	count := 0

	filepath.Walk(scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Check for common patterns of unused files
		basename := filepath.Base(path)
		if strings.HasSuffix(basename, ".bak") ||
			strings.HasSuffix(basename, ".old") ||
			strings.HasSuffix(basename, "~") ||
			strings.Contains(basename, "unused") ||
			strings.Contains(basename, "deprecated") {
			count++
		}

		return nil
	})

	return count
}

// countLargeFiles counts files over 500 lines
func (c *RefactorMetricsCollector) countLargeFiles(scenarioPath string) int {
	count := 0

	filepath.Walk(scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		validExts := map[string]bool{
			".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		}

		if !validExts[ext] {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Count(string(content), "\n")
		if lines > 500 {
			count++
		}

		return nil
	})

	return count
}

// countDeepNesting counts files with deep nesting (> 4 levels)
func (c *RefactorMetricsCollector) countDeepNesting(scenarioPath string) int {
	count := 0

	filepath.Walk(scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".ts" && ext != ".tsx" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		maxNesting := 0
		currentNesting := 0

		for _, line := range lines {
			openBraces := strings.Count(line, "{")
			closeBraces := strings.Count(line, "}")
			currentNesting += openBraces - closeBraces

			if currentNesting > maxNesting {
				maxNesting = currentNesting
			}
		}

		if maxNesting > 4 {
			count++
		}

		return nil
	})

	return count
}
