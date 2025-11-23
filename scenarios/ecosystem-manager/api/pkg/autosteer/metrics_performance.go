package autosteer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PerformanceMetricsCollector handles collection of performance metrics
type PerformanceMetricsCollector struct {
	projectRoot string
}

// NewPerformanceMetricsCollector creates a new performance metrics collector
func NewPerformanceMetricsCollector(projectRoot string) *PerformanceMetricsCollector {
	return &PerformanceMetricsCollector{
		projectRoot: projectRoot,
	}
}

// Collect gathers all performance metrics for a scenario
func (c *PerformanceMetricsCollector) Collect(scenarioName string) (*PerformanceMetrics, error) {
	metrics := &PerformanceMetrics{}

	// Collect bundle size (static analysis, doesn't require running app)
	metrics.BundleSizeKB = c.getBundleSize(scenarioName)

	// For dynamic metrics (Lighthouse), we'd need the app running
	// These will be 0 unless RunLighthouse is called separately
	metrics.InitialLoadTimeMS = 0
	metrics.LCPMS = 0
	metrics.FIDMS = 0
	metrics.CLSScore = 0

	return metrics, nil
}

// getBundleSize calculates the bundle size of the UI build
func (c *PerformanceMetricsCollector) getBundleSize(scenarioName string) float64 {
	distPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "dist")

	// Check if dist directory exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		// Try to build if dist doesn't exist
		c.buildUI(scenarioName)
	}

	// Recalculate directory size
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		return 0
	}

	var size int64
	err := filepath.Walk(distPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0
	}

	// Convert to KB
	return float64(size) / 1024.0
}

// buildUI attempts to build the UI to generate dist folder
func (c *PerformanceMetricsCollector) buildUI(scenarioName string) error {
	uiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui")

	if _, err := os.Stat(uiPath); os.IsNotExist(err) {
		return fmt.Errorf("UI directory does not exist")
	}

	// Run npm build
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = uiPath

	// Suppress output
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

// RunLighthouse runs Lighthouse CI against a running application
func (c *PerformanceMetricsCollector) RunLighthouse(url string) (*PerformanceMetrics, error) {
	// Check if lhci is installed
	if _, err := exec.LookPath("lhci"); err != nil {
		return nil, fmt.Errorf("lighthouse CLI (lhci) not installed: %w", err)
	}

	// Create temp directory for lighthouse results
	tmpDir, err := os.MkdirTemp("", "lighthouse-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Run lighthouse
	cmd := exec.Command("lhci", "autorun",
		"--collect.url="+url,
		"--collect.numberOfRuns=1",
		"--upload.target=temporary-public-storage")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("lighthouse failed: %w, output: %s", err, string(output))
	}

	// Parse lighthouse results
	return c.parseLighthouseResults(tmpDir)
}

// parseLighthouseResults parses Lighthouse JSON results
func (c *PerformanceMetricsCollector) parseLighthouseResults(resultsDir string) (*PerformanceMetrics, error) {
	// Find the lighthouse report JSON file
	var reportPath string
	err := filepath.Walk(resultsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".json" {
			reportPath = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if reportPath == "" {
		return nil, fmt.Errorf("no lighthouse report found")
	}

	// Read and parse the report
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lighthouse report: %w", err)
	}

	var report struct {
		Audits struct {
			FirstContentfulPaint struct {
				NumericValue float64 `json:"numericValue"`
			} `json:"first-contentful-paint"`
			LargestContentfulPaint struct {
				NumericValue float64 `json:"numericValue"`
			} `json:"largest-contentful-paint"`
			TotalBlockingTime struct {
				NumericValue float64 `json:"numericValue"`
			} `json:"total-blocking-time"`
			CumulativeLayoutShift struct {
				NumericValue float64 `json:"numericValue"`
			} `json:"cumulative-layout-shift"`
			SpeedIndex struct {
				NumericValue float64 `json:"numericValue"`
			} `json:"speed-index"`
		} `json:"audits"`
	}

	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse lighthouse report: %w", err)
	}

	metrics := &PerformanceMetrics{
		BundleSizeKB:      0, // Not included in Lighthouse
		InitialLoadTimeMS: int(report.Audits.FirstContentfulPaint.NumericValue),
		LCPMS:            int(report.Audits.LargestContentfulPaint.NumericValue),
		FIDMS:            int(report.Audits.TotalBlockingTime.NumericValue),
		CLSScore:         report.Audits.CumulativeLayoutShift.NumericValue,
	}

	return metrics, nil
}

// RunSimplifiedPerformanceTest runs a simplified performance test without full Lighthouse
func (c *PerformanceMetricsCollector) RunSimplifiedPerformanceTest(url string) (*PerformanceMetrics, error) {
	// Use curl with timing to get basic performance metrics
	cmd := exec.Command("curl", "-w",
		`{"time_namelookup":%{time_namelookup},"time_connect":%{time_connect},"time_starttransfer":%{time_starttransfer},"time_total":%{time_total}}`,
		"-o", "/dev/null",
		"-s",
		url)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("curl failed: %w", err)
	}

	var timing struct {
		TimeNamelookup    float64 `json:"time_namelookup"`
		TimeConnect       float64 `json:"time_connect"`
		TimeStartTransfer float64 `json:"time_starttransfer"`
		TimeTotal         float64 `json:"time_total"`
	}

	if err := json.Unmarshal(output, &timing); err != nil {
		return nil, fmt.Errorf("failed to parse curl output: %w", err)
	}

	// Convert seconds to milliseconds
	metrics := &PerformanceMetrics{
		BundleSizeKB:      0,
		InitialLoadTimeMS: int(timing.TimeTotal * 1000),
		LCPMS:            int(timing.TimeStartTransfer * 1000),
		FIDMS:            0,
		CLSScore:         0,
	}

	return metrics, nil
}

// EstimatePerformanceScore provides a rough performance score
func (c *PerformanceMetricsCollector) EstimatePerformanceScore(metrics *PerformanceMetrics) float64 {
	score := 100.0

	// Penalize large bundle sizes
	if metrics.BundleSizeKB > 500 {
		// Deduct 10 points per 100KB over 500KB
		score -= ((metrics.BundleSizeKB - 500) / 100) * 10
	}

	// Penalize slow initial load
	if metrics.InitialLoadTimeMS > 3000 {
		// Deduct 5 points per second over 3 seconds
		score -= (float64(metrics.InitialLoadTimeMS-3000) / 1000) * 5
	}

	// Penalize poor LCP
	if metrics.LCPMS > 2500 {
		// Deduct 10 points for poor LCP
		score -= 10
	}

	// Penalize poor CLS
	if metrics.CLSScore > 0.1 {
		// Deduct 15 points for poor layout stability
		score -= 15
	}

	if score < 0 {
		score = 0
	}

	return score
}

// GetResourceTiming gets resource timing metrics from browser
// This would require browser automation integration
func (c *PerformanceMetricsCollector) GetResourceTiming(url string) (map[string]interface{}, error) {
	// This is a placeholder for future browser automation integration
	// Would use browser-automation-studio or Puppeteer to collect
	// performance.getEntriesByType("resource") data
	return nil, fmt.Errorf("not implemented - requires browser automation")
}
