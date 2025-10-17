package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/scenarios/react-component-library/models"
)

type TestingService struct {
	// Add dependencies like axe-core, lighthouse, etc.
}

func NewTestingService() *TestingService {
	return &TestingService{}
}

// RunTests executes the specified tests on a component
func (s *TestingService) RunTests(componentID uuid.UUID, req models.ComponentTestRequest) (*models.ComponentTestResponse, error) {
	startTime := time.Now()
	var testResults []models.TestResult
	var totalScore float64

	// Run each requested test type
	for _, testType := range req.TestTypes {
		result, err := s.runSingleTest(componentID, testType, req.TestConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to run %s test: %w", testType, err)
		}
		testResults = append(testResults, *result)
		totalScore += result.Score
	}

	// Calculate overall score
	overallScore := totalScore / float64(len(req.TestTypes))

	// Generate recommendations based on test results
	recommendations := s.generateRecommendations(testResults)

	response := &models.ComponentTestResponse{
		TestResults:     testResults,
		OverallScore:    overallScore,
		Recommendations: recommendations,
		TestDurationMs:  time.Since(startTime).Milliseconds(),
	}

	return response, nil
}

// runSingleTest runs a specific test type
func (s *TestingService) runSingleTest(componentID uuid.UUID, testType models.TestType, config map[string]string) (*models.TestResult, error) {
	result := &models.TestResult{
		ID:          uuid.New(),
		ComponentID: componentID,
		TestType:    testType,
		TestedAt:    time.Now(),
	}

	switch testType {
	case models.TestTypeAccessibility:
		return s.runAccessibilityTest(result, config)
	case models.TestTypePerformance:
		return s.runPerformanceTest(result, config)
	case models.TestTypeVisual:
		return s.runVisualTest(result, config)
	case models.TestTypeUnitTest:
		return s.runUnitTest(result, config)
	case models.TestTypeLinting:
		return s.runLintingTest(result, config)
	default:
		return nil, fmt.Errorf("unsupported test type: %s", testType)
	}
}

// runAccessibilityTest runs accessibility tests using axe-core
func (s *TestingService) runAccessibilityTest(result *models.TestResult, config map[string]string) (*models.TestResult, error) {
	startTime := time.Now()

	// Simulate accessibility testing (in real implementation, would use axe-core)
	// This would involve:
	// 1. Rendering component in headless browser
	// 2. Running axe-core accessibility checks
	// 3. Analyzing results for WCAG compliance

	// Mock results for demonstration
	mockResults := map[string]interface{}{
		"violations": []map[string]interface{}{
			{
				"id":          "color-contrast",
				"impact":      "serious",
				"description": "Elements must have sufficient color contrast",
				"nodes":       1,
			},
		},
		"passes": []map[string]interface{}{
			{
				"id":          "aria-labels", 
				"description": "Elements have appropriate aria-labels",
				"nodes":       3,
			},
		},
		"incomplete": []map[string]interface{}{},
		"inapplicable": []map[string]interface{}{
			{
				"id":          "video-captions",
				"description": "Video elements have captions", 
				"nodes":       0,
			},
		},
	}

	resultsJson, _ := json.Marshal(mockResults)
	result.Results = resultsJson

	// Calculate score based on violations
	violations := len(mockResults["violations"].([]map[string]interface{}))
	passes := len(mockResults["passes"].([]map[string]interface{}))
	
	if violations == 0 {
		result.Score = 100.0
		result.Passed = true
	} else {
		// Score based on ratio of passes to total checks
		total := violations + passes
		result.Score = float64(passes) / float64(total) * 100
		result.Passed = result.Score >= 80.0 // Pass threshold
	}

	result.TestDuration = int(time.Since(startTime).Milliseconds())
	return result, nil
}

// runPerformanceTest runs performance benchmarking
func (s *TestingService) runPerformanceTest(result *models.TestResult, config map[string]string) (*models.TestResult, error) {
	startTime := time.Now()

	// Mock performance metrics (in real implementation, would use Lighthouse or similar)
	mockResults := map[string]interface{}{
		"render_time_ms":    15.2,
		"bundle_size_kb":    4.8,
		"memory_usage_mb":   2.1,
		"paint_metrics": map[string]interface{}{
			"first_paint_ms":           8.1,
			"first_contentful_paint":   12.3,
			"largest_contentful_paint": 15.2,
		},
		"cpu_usage_percent": 0.8,
		"network_requests":  0,
	}

	resultsJson, _ := json.Marshal(mockResults)
	result.Results = resultsJson

	// Calculate performance score
	renderTime := mockResults["render_time_ms"].(float64)
	bundleSize := mockResults["bundle_size_kb"].(float64)
	
	// Score based on performance thresholds
	renderScore := 100.0
	if renderTime > 16.0 { // 16ms threshold for 60fps
		renderScore = 100.0 * (16.0 / renderTime)
	}
	
	bundleScore := 100.0
	if bundleSize > 10.0 { // 10kb threshold
		bundleScore = 100.0 * (10.0 / bundleSize)
	}
	
	result.Score = (renderScore + bundleScore) / 2
	result.Passed = result.Score >= 80.0

	result.TestDuration = int(time.Since(startTime).Milliseconds())
	return result, nil
}

// runVisualTest runs visual regression testing
func (s *TestingService) runVisualTest(result *models.TestResult, config map[string]string) (*models.TestResult, error) {
	startTime := time.Now()

	// Mock visual test results (in real implementation, would use Puppeteer + image comparison)
	mockResults := map[string]interface{}{
		"screenshot_url":      "/screenshots/component-123.png",
		"baseline_url":        "/screenshots/baseline-123.png",
		"diff_url":           "/screenshots/diff-123.png",
		"diff_percentage":     0.2,
		"pixel_differences":   12,
		"threshold_passed":    true,
	}

	resultsJson, _ := json.Marshal(mockResults)
	result.Results = resultsJson

	// Score based on visual differences
	diffPercentage := mockResults["diff_percentage"].(float64)
	result.Score = 100.0 - (diffPercentage * 10) // 10% diff = 99 score
	result.Passed = diffPercentage < 5.0          // Pass if less than 5% difference

	result.TestDuration = int(time.Since(startTime).Milliseconds())
	return result, nil
}

// runUnitTest runs unit tests on the component
func (s *TestingService) runUnitTest(result *models.TestResult, config map[string]string) (*models.TestResult, error) {
	startTime := time.Now()

	// Mock unit test results (in real implementation, would run Jest/React Testing Library)
	mockResults := map[string]interface{}{
		"tests_run":    5,
		"tests_passed": 4,
		"tests_failed": 1,
		"coverage": map[string]interface{}{
			"lines":      90.5,
			"functions":  85.7,
			"branches":   78.2,
			"statements": 88.9,
		},
		"failed_tests": []map[string]interface{}{
			{
				"name":    "should handle invalid props",
				"message": "Expected component to throw error with invalid props",
			},
		},
	}

	resultsJson, _ := json.Marshal(mockResults)
	result.Results = resultsJson

	// Score based on test pass rate
	testsRun := mockResults["tests_run"].(int)
	testsPassed := mockResults["tests_passed"].(int)
	
	result.Score = float64(testsPassed) / float64(testsRun) * 100
	result.Passed = result.Score >= 80.0

	result.TestDuration = int(time.Since(startTime).Milliseconds())
	return result, nil
}

// runLintingTest runs code quality/linting checks
func (s *TestingService) runLintingTest(result *models.TestResult, config map[string]string) (*models.TestResult, error) {
	startTime := time.Now()

	// Mock linting results (in real implementation, would run ESLint)
	mockResults := map[string]interface{}{
		"errors":   1,
		"warnings": 3,
		"issues": []map[string]interface{}{
			{
				"line":     15,
				"column":   8,
				"severity": "error",
				"message":  "Missing propTypes definition",
				"rule":     "react/prop-types",
			},
			{
				"line":     22,
				"column":   12,
				"severity": "warning", 
				"message":  "Unused variable 'temp'",
				"rule":     "no-unused-vars",
			},
		},
	}

	resultsJson, _ := json.Marshal(mockResults)
	result.Results = resultsJson

	// Score based on linting issues
	errors := mockResults["errors"].(int)
	warnings := mockResults["warnings"].(int)
	
	result.Score = 100.0 - float64(errors*10) - float64(warnings*2) // Errors -10, warnings -2
	if result.Score < 0 {
		result.Score = 0
	}
	result.Passed = errors == 0

	result.TestDuration = int(time.Since(startTime).Milliseconds())
	return result, nil
}

// BenchmarkComponent runs comprehensive performance benchmarking
func (s *TestingService) BenchmarkComponent(componentID uuid.UUID) (interface{}, error) {
	// Run performance test with detailed benchmarking
	result, err := s.runSingleTest(componentID, models.TestTypePerformance, nil)
	if err != nil {
		return nil, err
	}

	// Add additional benchmarking data
	benchmark := map[string]interface{}{
		"component_id": componentID,
		"timestamp":   time.Now(),
		"metrics":     result.Results,
		"score":       result.Score,
		"grade":       s.getPerformanceGrade(result.Score),
		"recommendations": s.generatePerformanceRecommendations(result),
	}

	return benchmark, nil
}

// generateRecommendations creates improvement recommendations based on test results
func (s *TestingService) generateRecommendations(results []models.TestResult) []string {
	var recommendations []string

	for _, result := range results {
		if !result.Passed {
			switch result.TestType {
			case models.TestTypeAccessibility:
				recommendations = append(recommendations, "Improve accessibility by addressing color contrast and ARIA label issues")
			case models.TestTypePerformance:
				recommendations = append(recommendations, "Optimize component performance by reducing render time and bundle size")
			case models.TestTypeVisual:
				recommendations = append(recommendations, "Review visual changes - component appearance has changed significantly")
			case models.TestTypeUnitTest:
				recommendations = append(recommendations, "Fix failing unit tests to ensure component reliability")
			case models.TestTypeLinting:
				recommendations = append(recommendations, "Address code quality issues identified by linting")
			}
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "All tests passed! Consider adding more comprehensive test coverage.")
	}

	return recommendations
}

// generatePerformanceRecommendations generates specific performance recommendations
func (s *TestingService) generatePerformanceRecommendations(result *models.TestResult) []string {
	recommendations := []string{
		"Consider memoizing expensive calculations with useMemo",
		"Lazy load components that are not immediately visible",
		"Optimize bundle size by removing unused imports",
		"Use React.memo for components that re-render frequently",
	}
	return recommendations
}

// getPerformanceGrade converts numeric score to letter grade
func (s *TestingService) getPerformanceGrade(score float64) string {
	if score >= 90 {
		return "A"
	} else if score >= 80 {
		return "B"
	} else if score >= 70 {
		return "C"
	} else if score >= 60 {
		return "D"
	}
	return "F"
}