package validators

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AnalyzeTestFileQuality analyzes test file quality using basic heuristics
func AnalyzeTestFileQuality(testFilePath, scenarioRoot string) TestFileQuality {
	fullPath := filepath.Join(scenarioRoot, testFilePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return TestFileQuality{
			Exists:       false,
			IsMeaningful: false,
			Reason:       "file_not_found",
		}
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Count non-empty lines
	var nonEmptyLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	// Remove comment-only lines
	var nonCommentLines []string
	for _, line := range nonEmptyLines {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "//") &&
			!strings.HasPrefix(t, "/*") &&
			!strings.HasPrefix(t, "*") &&
			!strings.HasPrefix(t, "#") { // Python/shell comments
			nonCommentLines = append(nonCommentLines, line)
		}
	}

	// Count test functions
	testFunctionPattern := regexp.MustCompile(`(?i)func Test|@test|test\(|it\(|describe\(|def test_`)
	testFunctionMatches := testFunctionPattern.FindAllString(content, -1)
	testFunctionCount := len(testFunctionMatches)

	// Count assertions
	assertionPattern := regexp.MustCompile(`(?i)assert|expect|require|Should|Equal|Contains|Error|True|False|toBe|toHaveBeenCalled`)
	assertionMatches := assertionPattern.FindAllString(content, -1)
	assertionCount := len(assertionMatches)

	assertionDensity := 0.0
	if len(nonCommentLines) > 0 {
		assertionDensity = float64(assertionCount) / float64(len(nonCommentLines))
	}

	// Quality heuristics
	hasMinimumCode := len(nonCommentLines) >= 20        // At least 20 LOC
	hasAssertions := assertionCount > 0
	hasMultipleTestFunctions := testFunctionCount >= 3 // Require ≥3 test functions
	hasGoodAssertionDensity := assertionDensity >= 0.1 // ≥1 assertion per 10 LOC
	hasTestFunctions := testFunctionCount > 0

	// Calculate quality score (0-5, need ≥4 to pass)
	qualityScore := 0
	if hasMinimumCode {
		qualityScore++
	}
	if hasAssertions {
		qualityScore++
	}
	if hasTestFunctions {
		qualityScore++
	}
	if hasMultipleTestFunctions {
		qualityScore++
	}
	if hasGoodAssertionDensity {
		qualityScore++
	}

	reason := "ok"
	if qualityScore < 4 {
		reason = "insufficient_quality"
	}

	return TestFileQuality{
		Exists:            true,
		LOC:               len(nonCommentLines),
		HasAssertions:     hasAssertions,
		HasTestFunctions:  hasTestFunctions,
		TestFunctionCount: testFunctionCount,
		AssertionCount:    assertionCount,
		AssertionDensity:  assertionDensity,
		IsMeaningful:      qualityScore >= 4, // Need 4+ indicators
		QualityScore:      qualityScore,
		Reason:            reason,
	}
}

// AnalyzePlaybookQuality analyzes e2e playbook quality
func AnalyzePlaybookQuality(playbookPath, scenarioRoot string) PlaybookQuality {
	fullPath := filepath.Join(scenarioRoot, playbookPath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return PlaybookQuality{
			Exists:       false,
			IsMeaningful: false,
			Reason:       "file_not_found",
		}
	}

	fileSize := len(data)

	// Try to parse as JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return PlaybookQuality{
			Exists:       true,
			IsMeaningful: false,
			Reason:       "parse_error",
		}
	}

	// Check for BAS format (nodes + edges)
	hasNodes := false
	nodeCount := 0
	hasNodeActions := false
	if nodes, ok := parsed["nodes"].([]interface{}); ok && len(nodes) > 0 {
		hasNodes = true
		nodeCount = len(nodes)
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				if _, hasType := nodeMap["type"]; hasType {
					hasNodeActions = true
					break
				}
				if _, hasAction := nodeMap["action"]; hasAction {
					hasNodeActions = true
					break
				}
			}
		}
	}

	// Check for old format (steps)
	hasSteps := false
	stepCount := 0
	hasStepActions := false
	if steps, ok := parsed["steps"].([]interface{}); ok && len(steps) > 0 {
		hasSteps = true
		stepCount = len(steps)
		for _, step := range steps {
			if stepMap, ok := step.(map[string]interface{}); ok {
				if _, hasAction := stepMap["action"]; hasAction {
					hasStepActions = true
					break
				}
				if _, hasType := stepMap["type"]; hasType {
					hasStepActions = true
					break
				}
			}
		}
	}

	// Accept either format
	isValid := (hasNodes && hasNodeActions && fileSize >= 100) ||
		(hasSteps && hasStepActions && fileSize >= 100)

	reason := "ok"
	if !isValid {
		if !hasSteps && !hasNodes {
			reason = "no_steps_or_nodes"
		} else {
			reason = "no_actions"
		}
	}

	count := stepCount
	if nodeCount > count {
		count = nodeCount
	}

	return PlaybookQuality{
		Exists:       true,
		StepCount:    count,
		HasActions:   hasStepActions || hasNodeActions,
		FileSize:     fileSize,
		IsMeaningful: isValid,
		Reason:       reason,
	}
}

// LowQualityTestInfo represents information about a low-quality test
type LowQualityTestInfo struct {
	Requirement  string `json:"requirement"`
	Ref          string `json:"ref"`
	QualityScore int    `json:"quality_score"`
	Reason       string `json:"reason"`
}

// FindLowQualityTests finds all low-quality test files referenced by requirements
func FindLowQualityTests(requirements []Requirement, scenarioRoot string) []LowQualityTestInfo {
	var lowQualityTests []LowQualityTestInfo

	for _, req := range requirements {
		for _, v := range req.Validation {
			if v.Type == "test" && v.Ref != "" {
				quality := AnalyzeTestFileQuality(v.Ref, scenarioRoot)
				if quality.Exists && !quality.IsMeaningful {
					lowQualityTests = append(lowQualityTests, LowQualityTestInfo{
						Requirement:  req.ID,
						Ref:          v.Ref,
						QualityScore: quality.QualityScore,
						Reason:       quality.Reason,
					})
				}
			}
		}
	}

	return lowQualityTests
}
