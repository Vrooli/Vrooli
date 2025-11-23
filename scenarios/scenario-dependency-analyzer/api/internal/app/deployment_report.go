package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	types "scenario-dependency-analyzer/internal/types"
)

const (
	deploymentReportVersion = 1
	tierBlockerThreshold    = 0.75
)

// buildDeploymentReport orchestrates the full deployment analysis workflow.
// It builds the dependency DAG, computes tier aggregates, generates bundle manifests,
// and identifies metadata gaps.
func buildDeploymentReport(scenarioName, scenarioPath, scenariosDir string, cfg *types.ServiceConfig) *types.DeploymentAnalysisReport {
	if cfg == nil {
		return nil
	}

	generatedAt := time.Now().UTC()
	visited := map[string]struct{}{}
	visited[normalizeName(scenarioName)] = struct{}{}
	nodes := buildDependencyNodeList(scenariosDir, scenarioName, cfg, visited)
	aggregates := computeTierAggregates(nodes)
	manifest := buildBundleManifest(scenarioName, scenarioPath, generatedAt, nodes)

	// Extract known tiers from aggregates
	knownTiers := make([]string, 0, len(aggregates))
	for tier := range aggregates {
		knownTiers = append(knownTiers, tier)
	}
	// Also check the config for tier definitions
	if cfg.Deployment != nil && cfg.Deployment.Tiers != nil {
		for tier := range cfg.Deployment.Tiers {
			found := false
			for _, kt := range knownTiers {
				if kt == tier {
					found = true
					break
				}
			}
			if !found {
				knownTiers = append(knownTiers, tier)
			}
		}
	}
	// Add standard tiers if none found
	if len(knownTiers) == 0 {
		knownTiers = []string{"desktop", "server", "mobile", "saas"}
	}

	gaps := analyzeDeploymentGaps(scenarioName, scenariosDir, nodes, knownTiers)

	return &types.DeploymentAnalysisReport{
		Scenario:       scenarioName,
		ReportVersion:  deploymentReportVersion,
		GeneratedAt:    generatedAt,
		Dependencies:   nodes,
		Aggregates:     aggregates,
		BundleManifest: manifest,
		MetadataGaps:   gaps,
	}
}

// persistDeploymentReport saves the deployment report to .vrooli/deployment/deployment-report.json
func persistDeploymentReport(scenarioPath string, report *types.DeploymentAnalysisReport) error {
	if report == nil {
		return nil
	}
	reportDir := filepath.Join(scenarioPath, ".vrooli", "deployment")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	reportPath := filepath.Join(reportDir, "deployment-report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := reportPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, reportPath)
}

// loadPersistedDeploymentReport loads a previously saved deployment report
func loadPersistedDeploymentReport(scenarioPath string) (*types.DeploymentAnalysisReport, error) {
	reportPath := filepath.Join(scenarioPath, ".vrooli", "deployment", "deployment-report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}
	var report types.DeploymentAnalysisReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}
