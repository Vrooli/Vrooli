package autosteer

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// TestMetricsCollector handles collection of testing metrics
type TestMetricsCollector struct {
	projectRoot string
}

// NewTestMetricsCollector creates a new test metrics collector
func NewTestMetricsCollector(projectRoot string) *TestMetricsCollector {
	return &TestMetricsCollector{
		projectRoot: projectRoot,
	}
}

// Collect gathers all test metrics for a scenario
func (c *TestMetricsCollector) Collect(scenarioName string) (*TestMetrics, error) {
	metrics := &TestMetrics{}

	// Collect each metric independently (best effort)
	metrics.UnitTestCoverage = c.getUnitTestCoverage(scenarioName)
	metrics.IntegrationTestCoverage = c.getIntegrationTestCoverage(scenarioName)
	metrics.UITestCoverage = c.getUITestCoverage(scenarioName)
	metrics.EdgeCasesCovered = c.countEdgeCases(scenarioName)
	metrics.FlakyTests = c.detectFlakyTests(scenarioName)
	metrics.TestQualityScore = c.calculateTestQuality(scenarioName)

	return metrics, nil
}

// getUnitTestCoverage gets unit test coverage for both Go and TypeScript
func (c *TestMetricsCollector) getUnitTestCoverage(scenarioName string) float64 {
	goCoverage := c.getGoTestCoverage(scenarioName)
	tsCoverage := c.getTypeScriptTestCoverage(scenarioName)

	// If both exist, average them
	if goCoverage > 0 && tsCoverage > 0 {
		return (goCoverage + tsCoverage) / 2.0
	} else if goCoverage > 0 {
		return goCoverage
	} else if tsCoverage > 0 {
		return tsCoverage
	}

	return 0
}

// getGoTestCoverage runs Go tests with coverage
func (c *TestMetricsCollector) getGoTestCoverage(scenarioName string) float64 {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		return 0
	}

	// Run go test with coverage
	cmd := exec.Command("go", "test", "./...", "-cover", "-coverprofile=coverage.out")
	cmd.Dir = apiPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Tests may fail, but we can still get coverage
	}

	// Parse coverage from output
	coverage := c.parseGoCoverageOutput(string(output))
	if coverage > 0 {
		return coverage
	}

	// Try to read from coverage.out file
	coverageFile := filepath.Join(apiPath, "coverage.out")
	if _, err := os.Stat(coverageFile); err == nil {
		return c.parseGoCoverageFile(coverageFile)
	}

	return 0
}

// parseGoCoverageOutput parses coverage percentage from go test output
func (c *TestMetricsCollector) parseGoCoverageOutput(output string) float64 {
	// Look for pattern: "coverage: XX.X% of statements"
	re := regexp.MustCompile(`coverage:\s+([\d.]+)%`)
	matches := re.FindStringSubmatch(output)

	if len(matches) < 2 {
		return 0
	}

	coverage, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	return coverage
}

// parseGoCoverageFile parses a Go coverage profile file
func (c *TestMetricsCollector) parseGoCoverageFile(coverageFile string) float64 {
	cmd := exec.Command("go", "tool", "cover", "-func="+coverageFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}

	// Look for total coverage in last line: "total: (statements) XX.X%"
	lines := strings.Split(string(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "total:") {
			re := regexp.MustCompile(`([\d.]+)%`)
			matches := re.FindStringSubmatch(lines[i])
			if len(matches) >= 2 {
				coverage, err := strconv.ParseFloat(matches[1], 64)
				if err == nil {
					return coverage
				}
			}
		}
	}

	return 0
}

// getTypeScriptTestCoverage gets TypeScript/JavaScript test coverage
func (c *TestMetricsCollector) getTypeScriptTestCoverage(scenarioName string) float64 {
	coveragePath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "coverage", "coverage-summary.json")

	data, err := os.ReadFile(coveragePath)
	if err != nil {
		// Try running tests to generate coverage
		return c.runTypeScriptTests(scenarioName)
	}

	return c.parseTSCoverageSummary(data)
}

// runTypeScriptTests runs TypeScript tests with coverage
func (c *TestMetricsCollector) runTypeScriptTests(scenarioName string) float64 {
	uiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui")

	if _, err := os.Stat(uiPath); os.IsNotExist(err) {
		return 0
	}

	// Check if package.json has test script
	packageJSON := filepath.Join(uiPath, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return 0
	}

	// Run npm test with coverage (assuming vitest or jest)
	cmd := exec.Command("npm", "run", "test:coverage")
	cmd.Dir = uiPath

	// Run with timeout
	if err := cmd.Run(); err != nil {
		// Tests may fail, but coverage might still be generated
	}

	// Try to read coverage summary
	coveragePath := filepath.Join(uiPath, "coverage", "coverage-summary.json")
	data, err := os.ReadFile(coveragePath)
	if err != nil {
		return 0
	}

	return c.parseTSCoverageSummary(data)
}

// parseTSCoverageSummary parses TypeScript coverage summary JSON
func (c *TestMetricsCollector) parseTSCoverageSummary(data []byte) float64 {
	var summary struct {
		Total struct {
			Lines struct {
				Pct float64 `json:"pct"`
			} `json:"lines"`
			Statements struct {
				Pct float64 `json:"pct"`
			} `json:"statements"`
		} `json:"total"`
	}

	if err := json.Unmarshal(data, &summary); err != nil {
		return 0
	}

	// Use line coverage if available, otherwise statements
	if summary.Total.Lines.Pct > 0 {
		return summary.Total.Lines.Pct
	}

	return summary.Total.Statements.Pct
}

// getIntegrationTestCoverage gets integration test coverage
func (c *TestMetricsCollector) getIntegrationTestCoverage(scenarioName string) float64 {
	// Integration tests are typically in test/ directory
	testPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "test")

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		return 0
	}

	// Count integration tests that are implemented
	totalTests := c.countIntegrationTests(testPath)
	passingTests := c.runIntegrationTests(scenarioName)

	if totalTests == 0 {
		return 0
	}

	// Coverage based on passing tests
	return (float64(passingTests) / float64(totalTests)) * 100
}

// countIntegrationTests counts total integration test files
func (c *TestMetricsCollector) countIntegrationTests(testPath string) int {
	count := 0

	filepath.Walk(testPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Count BATS test files and shell scripts
		if filepath.Ext(path) == ".bats" || filepath.Ext(path) == ".sh" {
			count++
		}

		return nil
	})

	return count
}

// runIntegrationTests runs integration tests and counts passing tests
func (c *TestMetricsCollector) runIntegrationTests(scenarioName string) int {
	testPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "test")

	// Run BATS tests if available
	cmd := exec.Command("bats", ".")
	cmd.Dir = testPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Some tests may fail
	}

	// Parse BATS output for passing tests
	outputStr := string(output)
	re := regexp.MustCompile(`(\d+)\s+tests?,\s+(\d+)\s+failures?`)
	matches := re.FindStringSubmatch(outputStr)

	if len(matches) >= 3 {
		total, _ := strconv.Atoi(matches[1])
		failures, _ := strconv.Atoi(matches[2])
		return total - failures
	}

	return 0
}

// getUITestCoverage gets UI test coverage from playbooks/E2E tests
func (c *TestMetricsCollector) getUITestCoverage(scenarioName string) float64 {
	playbooksPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "test", "playbooks")

	if _, err := os.Stat(playbooksPath); os.IsNotExist(err) {
		return 0
	}

	// Count total operational targets vs playbook coverage
	totalTargets := c.countOperationalTargets(scenarioName)
	playbookCount := c.countPlaybooks(playbooksPath)

	if totalTargets == 0 {
		return 0
	}

	coverage := (float64(playbookCount) / float64(totalTargets)) * 100

	if coverage > 100 {
		coverage = 100
	}

	return coverage
}

// countOperationalTargets counts total operational targets
func (c *TestMetricsCollector) countOperationalTargets(scenarioName string) int {
	requirementsPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "requirements", "index.json")

	data, err := os.ReadFile(requirementsPath)
	if err != nil {
		return 0
	}

	var requirements struct {
		Modules []struct {
			OperationalTargets []interface{} `json:"operationalTargets"`
		} `json:"modules"`
	}

	if err := json.Unmarshal(data, &requirements); err != nil {
		return 0
	}

	count := 0
	for _, module := range requirements.Modules {
		count += len(module.OperationalTargets)
	}

	return count
}

// countPlaybooks counts playbook test files
func (c *TestMetricsCollector) countPlaybooks(playbooksPath string) int {
	registryPath := filepath.Join(playbooksPath, "registry.json")

	data, err := os.ReadFile(registryPath)
	if err != nil {
		// Fallback: count JSON files
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
func (c *TestMetricsCollector) countPlaybookFiles(playbooksPath string) int {
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

// countEdgeCases counts documented edge cases in tests
func (c *TestMetricsCollector) countEdgeCases(scenarioName string) int {
	count := 0
	scenarioPath := filepath.Join(c.projectRoot, "scenarios", scenarioName)

	// Search for edge case patterns in test files
	filepath.Walk(scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Check test files
		basename := filepath.Base(path)
		isTestFile := strings.Contains(basename, "_test.") ||
			strings.Contains(basename, ".test.") ||
			strings.Contains(basename, ".spec.")

		if !isTestFile {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Count edge case patterns
		patterns := []string{
			"edge case",
			"boundary",
			"edge condition",
			"corner case",
			"limit",
			"empty",
			"null",
			"undefined",
			"zero",
			"negative",
			"overflow",
		}

		for _, pattern := range patterns {
			count += strings.Count(strings.ToLower(fileContent), pattern)
		}

		return nil
	})

	return count
}

// detectFlakyTests attempts to detect flaky tests
func (c *TestMetricsCollector) detectFlakyTests(scenarioName string) int {
	// This is a simplified implementation
	// Real flaky test detection requires running tests multiple times
	// For now, look for common flaky test patterns in code

	count := 0
	scenarioPath := filepath.Join(c.projectRoot, "scenarios", scenarioName)

	filepath.Walk(scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		basename := filepath.Base(path)
		isTestFile := strings.Contains(basename, "_test.") ||
			strings.Contains(basename, ".test.") ||
			strings.Contains(basename, ".spec.")

		if !isTestFile {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Look for common flaky test indicators
		flakyPatterns := []string{
			"setTimeout",
			"sleep(",
			"time.Sleep",
			"Date.now()",
			"Math.random()",
			"eventually",
			"retry",
			".skip",
			"xit(",
			"xdescribe(",
		}

		for _, pattern := range flakyPatterns {
			if strings.Contains(fileContent, pattern) {
				count++
				break // Count once per file
			}
		}

		return nil
	})

	return count
}

// calculateTestQuality calculates an overall test quality score
func (c *TestMetricsCollector) calculateTestQuality(scenarioName string) float64 {
	score := 0.0

	// Factor 1: Test coverage (40%)
	unitCoverage := c.getUnitTestCoverage(scenarioName)
	score += (unitCoverage / 100) * 40

	// Factor 2: Integration test coverage (30%)
	integrationCoverage := c.getIntegrationTestCoverage(scenarioName)
	score += (integrationCoverage / 100) * 30

	// Factor 3: Edge cases (15%)
	edgeCases := c.countEdgeCases(scenarioName)
	edgeCaseScore := float64(edgeCases) / 10.0 // 10+ edge cases = full score
	if edgeCaseScore > 1.0 {
		edgeCaseScore = 1.0
	}
	score += edgeCaseScore * 15

	// Factor 4: Flakiness penalty (15%)
	flakyTests := c.detectFlakyTests(scenarioName)
	flakinessScore := 1.0 - (float64(flakyTests) / 10.0) // Each flaky test reduces score
	if flakinessScore < 0 {
		flakinessScore = 0
	}
	score += flakinessScore * 15

	return score
}

// RunTestsMultipleTimes runs tests multiple times to detect flakiness
func (c *TestMetricsCollector) RunTestsMultipleTimes(scenarioName string, iterations int) map[string]int {
	results := make(map[string]int)

	for i := 0; i < iterations; i++ {
		// Run Go tests
		apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")
		if _, err := os.Stat(apiPath); err == nil {
			cmd := exec.Command("go", "test", "./...")
			cmd.Dir = apiPath

			if err := cmd.Run(); err != nil {
				results["go_tests_failed"]++
			} else {
				results["go_tests_passed"]++
			}
		}

		// Run TypeScript tests
		uiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui")
		if _, err := os.Stat(uiPath); err == nil {
			cmd := exec.Command("npm", "test")
			cmd.Dir = uiPath

			if err := cmd.Run(); err != nil {
				results["ts_tests_failed"]++
			} else {
				results["ts_tests_passed"]++
			}
		}
	}

	return results
}

// AnalyzeTestFlakiness analyzes test results for flakiness
func (c *TestMetricsCollector) AnalyzeTestFlakiness(results map[string]int) int {
	flakyCount := 0

	// If a test sometimes passes and sometimes fails, it's flaky
	if results["go_tests_passed"] > 0 && results["go_tests_failed"] > 0 {
		flakyCount++
	}

	if results["ts_tests_passed"] > 0 && results["ts_tests_failed"] > 0 {
		flakyCount++
	}

	return flakyCount
}
