// Package deployment provides deployment analysis, fitness scoring, and bundle
// manifest generation for scenarios targeting different deployment tiers.
//
// This package is the domain owner for:
//   - Dependency DAG construction (resource + scenario trees)
//   - Tier fitness scoring (desktop, server, mobile, saas, enterprise)
//   - Bundle manifest generation (files, services, swaps, secrets)
//   - Metadata gap analysis (missing tier-feasibility blocks and tier definitions)
//
// The main entry point is BuildReport, which orchestrates the full analysis workflow.
package deployment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	deployability "github.com/vrooli/vrooli/packages/deployability"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"

	"github.com/vrooli/api-core/storage"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

const (
	// ReportVersion tracks the schema version of deployment reports.
	ReportVersion = 1
)

// BuildReport orchestrates the full deployment analysis workflow.
// It builds the dependency DAG, computes tier aggregates, generates bundle manifests,
// and identifies metadata gaps.
func BuildReport(scenarioName, scenarioPath, scenariosDir string, cfg *types.Manifest) *types.DeploymentAnalysisReport {
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
	if cfg.TierFeasibility != nil && cfg.TierFeasibility.Tiers != nil {
		for tier := range cfg.TierFeasibility.Tiers {
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
		knownTiers = []string{
			string(deployability.TierLocal),
			string(deployability.TierDesktop),
			string(deployability.TierMobile),
			string(deployability.TierSaaS),
			string(deployability.TierEnterprise),
		}
	}
	sort.Strings(knownTiers)

	gaps := AnalyzeGaps(scenarioName, scenarioPath, scenariosDir, nodes, knownTiers)

	return &types.DeploymentAnalysisReport{
		Scenario:      scenarioName,
		ReportVersion: ReportVersion,
		GeneratedAt:   generatedAt,
		Provenance: types.DeploymentVerdictProvenance{
			Analyzer:        "scenario-dependency-analyzer",
			AnalyzerVersion: "2",
			ComputedAt:      generatedAt,
		},
		Dependencies:   nodes,
		Aggregates:     aggregates,
		BundleManifest: manifest,
		MetadataGaps:   gaps,
	}
}

func buildRootScenarioNode(scenarioName, scenarioPath string, cfg *types.Manifest) *types.DeploymentDependencyNode {
	if cfg == nil || cfg.TierFeasibility == nil {
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

	if len(cfg.TierFeasibility.Tiers) > 0 {
		node.TierSupport = convertTierTierMap(cfg.TierFeasibility.Tiers)
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
	report.Provenance.InputDigest = digestManifestInputs(scenarioPath, report)
	// Persisted reports are written only after being recomputed against the
	// current inputs. Clear a stale marker if the caller is refreshing a loaded
	// report in place.
	report.Stale = false
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

// digestManifestInputs hashes the service manifest and every declared resource
// or child scenario manifest reachable from the report. Missing inputs are
// included as explicit markers, so a missing file cannot silently look current.
func digestManifestInputs(scenarioPath string, report *types.DeploymentAnalysisReport) string {
	paths := map[string]struct{}{}
	add := func(path string) {
		if strings.TrimSpace(path) != "" {
			paths[filepath.Clean(path)] = struct{}{}
		}
	}
	add(filepath.Join(scenarioPath, ".vrooli", "service.json"))
	root := filepath.Dir(filepath.Dir(scenarioPath))
	var walk func([]types.DeploymentDependencyNode)
	walk = func(nodes []types.DeploymentDependencyNode) {
		for _, node := range nodes {
			switch node.Type {
			case "resource":
				add(filepath.Join(root, "resources", node.Name, "resource.json"))
			case "scenario":
				if node.Path != "" {
					add(filepath.Join(node.Path, ".vrooli", "service.json"))
				}
			}
			walk(node.Children)
		}
	}
	if report != nil {
		walk(report.Dependencies)
	}

	pathsList := make([]string, 0, len(paths))
	for path := range paths {
		pathsList = append(pathsList, path)
	}
	sort.Strings(pathsList)
	h := sha256.New()
	for _, path := range pathsList {
		_, _ = h.Write([]byte(path))
		data, readErr := os.ReadFile(path) // #nosec G304 -- paths are derived from the scenario workspace and manifest names.
		if readErr != nil {
			_, _ = h.Write([]byte("\x00missing\x00"))
			continue
		}
		_, _ = h.Write([]byte("\x00"))
		_, _ = h.Write(data)
		_, _ = h.Write([]byte("\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))
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
	currentDigest := digestManifestInputs(scenarioPath, &report)
	// An absent digest is also stale: it represents a legacy or manually
	// authored report whose inputs were never proven. Keep the report available
	// for diagnostics, but make the stale state explicit to every consumer.
	report.Stale = strings.TrimSpace(report.Provenance.InputDigest) == "" || report.Provenance.InputDigest != currentDigest
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
