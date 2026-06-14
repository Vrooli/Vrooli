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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"scenario-dependency-analyzer/internal/config"

	"github.com/vrooli/api-core/storage"

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
	aggregateNodes := append([]types.DeploymentDependencyNode{}, nodes...)
	if root := buildRootScenarioNode(scenarioName, scenarioPath, cfg); root != nil {
		aggregateNodes = append(aggregateNodes, *root)
	}
	aggregates := ComputeTierAggregates(aggregateNodes)
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
	sort.Strings(knownTiers)

	gaps := AnalyzeGaps(scenarioName, scenarioPath, scenariosDir, nodes, knownTiers)

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

func buildRootScenarioNode(scenarioName, scenarioPath string, cfg *types.ServiceConfig) *types.DeploymentDependencyNode {
	if cfg == nil || cfg.Deployment == nil {
		return nil
	}

	required := true
	enabled := true
	node := &types.DeploymentDependencyNode{
		Name:     scenarioName,
		Type:     "scenario",
		Path:     scenarioPath,
		Required: &required,
		Enabled:  &enabled,
		Source:   "root",
	}

	if cfg.Deployment.AggregateRequirements != nil {
		node.Requirements = cfg.Deployment.AggregateRequirements
	}

	if len(cfg.Deployment.Tiers) > 0 {
		node.TierSupport = convertTierTierMap(cfg.Deployment.Tiers)
	}

	if node.Requirements == nil && len(node.TierSupport) == 0 {
		return nil
	}

	return node
}

// PersistReport saves the deployment report to canonical scenario runtime storage.
func PersistReport(scenarioPath string, report *types.DeploymentAnalysisReport) error {
	if report == nil {
		return nil
	}
	reportPath, err := reportPathForScenario(scenarioPath)
	if err != nil {
		return err
	}
	reportDir := filepath.Dir(reportPath)
	if err := os.MkdirAll(reportDir, 0o750); err != nil {
		return err
	}
	if existing, err := LoadReport(scenarioPath); err == nil && existing != nil {
		if reportsEqualIgnoringGeneratedAt(existing, report) {
			report.GeneratedAt = existing.GeneratedAt
			report.BundleManifest.GeneratedAt = existing.BundleManifest.GeneratedAt
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if existingData, err := os.ReadFile(reportPath); err == nil && string(existingData) == string(data) { // #nosec G304 -- reportPath is derived from the scenario runtime path.
		return nil
	}
	tmpPath := reportPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, reportPath)
}

func reportsEqualIgnoringGeneratedAt(a, b *types.DeploymentAnalysisReport) bool {
	if a == nil || b == nil {
		return a == b
	}

	cloneA, err := cloneReport(a)
	if err != nil {
		return false
	}
	cloneB, err := cloneReport(b)
	if err != nil {
		return false
	}

	cloneA.GeneratedAt = time.Time{}
	cloneA.BundleManifest.GeneratedAt = time.Time{}
	cloneB.GeneratedAt = time.Time{}
	cloneB.BundleManifest.GeneratedAt = time.Time{}

	return reflect.DeepEqual(cloneA, cloneB)
}

func cloneReport(report *types.DeploymentAnalysisReport) (*types.DeploymentAnalysisReport, error) {
	data, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	var cloned types.DeploymentAnalysisReport
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
}

// LoadReport loads a previously saved deployment report.
func LoadReport(scenarioPath string) (*types.DeploymentAnalysisReport, error) {
	reportPath, err := reportPathForScenario(scenarioPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(reportPath) // #nosec G304 -- reportPath is derived from the scenario runtime path.
	if err != nil {
		return nil, err
	}
	var report types.DeploymentAnalysisReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func reportPathForScenario(scenarioPath string) (string, error) {
	scenarioName := filepath.Base(filepath.Clean(scenarioPath))
	if scenarioName == "." || scenarioName == string(filepath.Separator) || scenarioName == "" {
		return "", fmt.Errorf("resolve scenario report path: invalid scenario path %q", scenarioPath)
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	return resolver.Path(
		storage.Options{ScenarioID: scenarioName},
		storage.ClassData,
		filepath.Join("deployment", "deployment-report.json"),
	)
}
