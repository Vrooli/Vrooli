package autosteer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// MetricsCollector coordinates collection of all metrics using specialized collectors
type MetricsCollector struct {
	projectRoot string

	// Specialized collectors for each metrics category
	uxCollector          *UXMetricsCollector
	refactorCollector    *RefactorMetricsCollector
	testCollector        *TestMetricsCollector
	performanceCollector *PerformanceMetricsCollector
	securityCollector    *SecurityMetricsCollector
}

// NewMetricsCollector creates a new comprehensive metrics collector
func NewMetricsCollector(projectRoot string) *MetricsCollector {
	return &MetricsCollector{
		projectRoot:          projectRoot,
		uxCollector:          NewUXMetricsCollector(projectRoot),
		refactorCollector:    NewRefactorMetricsCollector(projectRoot, ""), // tidiness-manager URL can be configured
		testCollector:        NewTestMetricsCollector(projectRoot),
		performanceCollector: NewPerformanceMetricsCollector(projectRoot),
		securityCollector:    NewSecurityMetricsCollector(projectRoot),
	}
}

// CollectMetrics collects all available metrics for a scenario
func (m *MetricsCollector) CollectMetrics(scenarioName string, currentLoops int) (*MetricsSnapshot, error) {
	// Validate scenario name
	if scenarioName == "" {
		return nil, fmt.Errorf("scenario name cannot be empty")
	}

	// Verify scenario directory exists
	scenarioPath := filepath.Join(m.projectRoot, "scenarios", scenarioName)
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("scenario directory does not exist: %s", scenarioName)
	}

	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
		Loops:     currentLoops,
	}

	// Collect universal metrics (required - will fail if these fail)
	if err := m.collectUniversalMetrics(scenarioName, snapshot); err != nil {
		return nil, fmt.Errorf("failed to collect universal metrics: %w", err)
	}

	// Collect mode-specific metrics using specialized collectors (best effort - don't fail if unavailable)
	m.collectUXMetrics(scenarioName, snapshot)
	m.collectRefactorMetrics(scenarioName, snapshot)
	m.collectTestMetrics(scenarioName, snapshot)
	m.collectPerformanceMetrics(scenarioName, snapshot)
	m.collectSecurityMetrics(scenarioName, snapshot)

	return snapshot, nil
}

// collectUniversalMetrics collects the core metrics all scenarios have
func (m *MetricsCollector) collectUniversalMetrics(scenarioName string, snapshot *MetricsSnapshot) error {
	// Parse operational targets from requirements
	targets, err := m.parseOperationalTargets(scenarioName)
	if err != nil {
		return fmt.Errorf("failed to parse operational targets: %w", err)
	}

	snapshot.OperationalTargetsTotal = targets.Total
	snapshot.OperationalTargetsPassing = targets.Passing

	if targets.Total > 0 {
		snapshot.OperationalTargetsPercentage = (float64(targets.Passing) / float64(targets.Total)) * 100
	}

	// Check build status
	buildPassing, err := m.checkBuildStatus(scenarioName)
	if err != nil {
		// Non-fatal - assume build is broken if we can't check
		snapshot.BuildStatus = 0
	} else {
		if buildPassing {
			snapshot.BuildStatus = 1
		} else {
			snapshot.BuildStatus = 0
		}
	}

	return nil
}

// collectUXMetrics uses the specialized UX collector
func (m *MetricsCollector) collectUXMetrics(scenarioName string, snapshot *MetricsSnapshot) {
	uxMetrics, err := m.uxCollector.Collect(scenarioName)
	if err == nil && m.hasUXData(uxMetrics) {
		snapshot.UX = uxMetrics
	}
}

// collectRefactorMetrics uses the specialized refactor collector
func (m *MetricsCollector) collectRefactorMetrics(scenarioName string, snapshot *MetricsSnapshot) {
	refactorMetrics, err := m.refactorCollector.Collect(scenarioName)
	if err == nil && m.hasRefactorData(refactorMetrics) {
		snapshot.Refactor = refactorMetrics
	}
}

// collectTestMetrics uses the specialized test collector
func (m *MetricsCollector) collectTestMetrics(scenarioName string, snapshot *MetricsSnapshot) {
	testMetrics, err := m.testCollector.Collect(scenarioName)
	if err == nil && m.hasTestData(testMetrics) {
		snapshot.Test = testMetrics
	}
}

// collectPerformanceMetrics uses the specialized performance collector
func (m *MetricsCollector) collectPerformanceMetrics(scenarioName string, snapshot *MetricsSnapshot) {
	perfMetrics, err := m.performanceCollector.Collect(scenarioName)
	if err == nil && m.hasPerformanceData(perfMetrics) {
		snapshot.Performance = perfMetrics
	}
}

// collectSecurityMetrics uses the specialized security collector
func (m *MetricsCollector) collectSecurityMetrics(scenarioName string, snapshot *MetricsSnapshot) {
	securityMetrics, err := m.securityCollector.Collect(scenarioName)
	if err == nil && m.hasSecurityData(securityMetrics) {
		snapshot.Security = securityMetrics
	}
}

// Helper methods to check if metrics have meaningful data

func (m *MetricsCollector) hasUXData(metrics *UXMetrics) bool {
	return metrics != nil && (
		metrics.AccessibilityScore > 0 ||
		metrics.UITestCoverage > 0 ||
		metrics.ResponsiveBreakpoints > 0 ||
		metrics.UserFlowsImplemented > 0 ||
		metrics.LoadingStatesCount > 0 ||
		metrics.ErrorHandlingCoverage > 0)
}

func (m *MetricsCollector) hasRefactorData(metrics *RefactorMetrics) bool {
	return metrics != nil && (
		metrics.CyclomaticComplexityAvg > 0 ||
		metrics.DuplicationPercentage > 0 ||
		metrics.TidinessScore > 0 ||
		metrics.StandardsViolations > 0 ||
		metrics.TechDebtItems > 0)
}

func (m *MetricsCollector) hasTestData(metrics *TestMetrics) bool {
	return metrics != nil && (
		metrics.UnitTestCoverage > 0 ||
		metrics.IntegrationTestCoverage > 0 ||
		metrics.UITestCoverage > 0 ||
		metrics.EdgeCasesCovered > 0 ||
		metrics.TestQualityScore > 0)
}

func (m *MetricsCollector) hasPerformanceData(metrics *PerformanceMetrics) bool {
	return metrics != nil && (
		metrics.BundleSizeKB > 0 ||
		metrics.InitialLoadTimeMS > 0 ||
		metrics.LCPMS > 0)
}

func (m *MetricsCollector) hasSecurityData(metrics *SecurityMetrics) bool {
	return metrics != nil && (
		metrics.InputValidationCoverage > 0 ||
		metrics.AuthImplementationScore > 0 ||
		metrics.SecurityScanScore > 0)
}

// OperationalTargetCounts represents counts of operational targets
type OperationalTargetCounts struct {
	Total   int
	Passing int
}

// parseOperationalTargets parses operational target completion from requirements
func (m *MetricsCollector) parseOperationalTargets(scenarioName string) (*OperationalTargetCounts, error) {
	requirementsPath := filepath.Join(m.projectRoot, "scenarios", scenarioName, "requirements", "index.json")

	data, err := os.ReadFile(requirementsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read requirements file: %w", err)
	}

	var requirements struct {
		Modules []struct {
			OperationalTargets []struct {
				ID     string `json:"id"`
				Status string `json:"status,omitempty"`
			} `json:"operationalTargets"`
		} `json:"modules"`
	}

	if err := json.Unmarshal(data, &requirements); err != nil {
		return nil, fmt.Errorf("failed to parse requirements JSON: %w", err)
	}

	counts := &OperationalTargetCounts{}

	for _, module := range requirements.Modules {
		for _, target := range module.OperationalTargets {
			counts.Total++
			if target.Status == "passing" {
				counts.Passing++
			}
		}
	}

	return counts, nil
}

// checkBuildStatus checks if the scenario builds successfully
func (m *MetricsCollector) checkBuildStatus(scenarioName string) (bool, error) {
	scenarioPath := filepath.Join(m.projectRoot, "scenarios", scenarioName)

	// Check if Makefile exists
	makefilePath := filepath.Join(scenarioPath, "Makefile")
	if _, err := os.Stat(makefilePath); os.IsNotExist(err) {
		// No Makefile - can't check build status
		return false, fmt.Errorf("no Makefile found")
	}

	// Run make build with a timeout
	cmd := exec.Command("make", "build")
	cmd.Dir = scenarioPath

	// Suppress output
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return false, nil
	}

	return true, nil
}

// SetTidinessManagerURL configures the tidiness-manager service URL
func (m *MetricsCollector) SetTidinessManagerURL(url string) {
	m.refactorCollector.tidinessManagerURL = url
}

// CollectWithURL is a convenience method for collecting metrics with dynamic URL for running apps
func (m *MetricsCollector) CollectWithURL(scenarioName string, currentLoops int, appURL string) (*MetricsSnapshot, error) {
	// Collect standard metrics
	snapshot, err := m.CollectMetrics(scenarioName, currentLoops)
	if err != nil {
		return nil, err
	}

	// If app URL is provided, collect runtime metrics
	if appURL != "" {
		// Try to get Lighthouse metrics
		if perfMetrics, err := m.performanceCollector.RunLighthouse(appURL); err == nil {
			snapshot.Performance = perfMetrics
		}

		// Try to get accessibility score via axe
		if accessScore, err := m.uxCollector.RunAxeAccessibility(appURL); err == nil {
			if snapshot.UX == nil {
				snapshot.UX = &UXMetrics{}
			}
			snapshot.UX.AccessibilityScore = accessScore
		}
	}

	return snapshot, nil
}
