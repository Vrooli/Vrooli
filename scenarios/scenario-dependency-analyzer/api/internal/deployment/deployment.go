// Package deployment provides deployment analysis, fitness scoring, and bundle
// manifest generation for scenarios targeting different deployment tiers.
//
// This package is the domain owner for:
//   - Dependency DAG construction (resource + scenario trees)
//   - Tier fitness scoring (desktop, server, mobile, saas, enterprise)
//   - Bundle manifest generation (files, services, swaps, secrets)
//   - Metadata gap analysis (missing deployment blocks, tier definitions)
//
// The main entry point is BuildReport, which orchestrates the full analysis workflow.
package deployment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

const (
	// ReportVersion tracks the schema version of deployment reports.
	ReportVersion = 1

	// TierBlockerThreshold is the fitness score below which a dependency blocks a tier.
	TierBlockerThreshold = 0.75
)

// BuildReport orchestrates the full deployment analysis workflow.
// It builds the dependency DAG, computes tier aggregates, generates bundle manifests,
// and identifies metadata gaps.
func BuildReport(scenarioName, scenarioPath, scenariosDir string, cfg *types.ServiceConfig) *types.DeploymentAnalysisReport {
	if cfg == nil {
		return nil
	}

	generatedAt := time.Now().UTC()
	visited := map[string]struct{}{}
	visited[config.NormalizeName(scenarioName)] = struct{}{}
	nodes := BuildDependencyNodeList(scenariosDir, scenarioName, cfg, visited)
	aggregates := ComputeTierAggregates(nodes)
	manifest := BuildBundleManifest(scenarioName, scenarioPath, generatedAt, nodes, cfg)

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

	gaps := AnalyzeGaps(scenarioName, scenariosDir, nodes, knownTiers)

	return &types.DeploymentAnalysisReport{
		Scenario:       scenarioName,
		ReportVersion:  ReportVersion,
		GeneratedAt:    generatedAt,
		Dependencies:   nodes,
		Aggregates:     aggregates,
		BundleManifest: manifest,
		MetadataGaps:   gaps,
	}
}

// PersistReport saves the deployment report to .vrooli/deployment/deployment-report.json
func PersistReport(scenarioPath string, report *types.DeploymentAnalysisReport) error {
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

// LoadReport loads a previously saved deployment report.
func LoadReport(scenarioPath string) (*types.DeploymentAnalysisReport, error) {
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
