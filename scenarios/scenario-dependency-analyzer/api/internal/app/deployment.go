package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	types "scenario-dependency-analyzer/internal/types"
)

const (
	deploymentReportVersion = 1
	tierBlockerThreshold    = 0.75
)

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

	return &types.DeploymentAnalysisReport{
		Scenario:       scenarioName,
		ReportVersion:  deploymentReportVersion,
		GeneratedAt:    generatedAt,
		Dependencies:   nodes,
		Aggregates:     aggregates,
		BundleManifest: manifest,
	}
}

func buildDependencyNodeList(scenariosDir, scenarioName string, cfg *types.ServiceConfig, visited map[string]struct{}) []types.DeploymentDependencyNode {
	nodes := []types.DeploymentDependencyNode{}
	if cfg == nil {
		return nodes
	}

	var dependencyCatalog types.DeploymentDependencyCatalog
	if cfg.Deployment != nil {
		dependencyCatalog = cfg.Deployment.Dependencies
	}

	resources := resolvedResourceMap(cfg)
	resourceNames := make([]string, 0, len(resources))
	for name := range resources {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)
	for _, name := range resourceNames {
		var meta *types.DeploymentDependency
		if dependencyCatalog.Resources != nil {
			if resourceMeta, ok := dependencyCatalog.Resources[name]; ok {
				copyMeta := resourceMeta
				meta = &copyMeta
			}
		}
		node := buildResourceDependencyNode(name, meta)
		node.Source = "declared"
		nodes = append(nodes, node)
	}

	if cfg.Dependencies.Scenarios != nil {
		scenarioNames := make([]string, 0, len(cfg.Dependencies.Scenarios))
		for name := range cfg.Dependencies.Scenarios {
			scenarioNames = append(scenarioNames, name)
		}
		sort.Strings(scenarioNames)
		for _, depName := range scenarioNames {
			var meta *types.DeploymentDependency
			if dependencyCatalog.Scenarios != nil {
				if scenarioMeta, ok := dependencyCatalog.Scenarios[depName]; ok {
					copyMeta := scenarioMeta
					meta = &copyMeta
				}
			}
			node := buildScenarioDependencyNode(scenariosDir, depName, meta, visited)
			node.Source = "declared"
			nodes = append(nodes, node)
		}
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Type == nodes[j].Type {
			return nodes[i].Name < nodes[j].Name
		}
		return nodes[i].Type < nodes[j].Type
	})

	return nodes
}

func buildResourceDependencyNode(name string, meta *types.DeploymentDependency) types.DeploymentDependencyNode {
	node := types.DeploymentDependencyNode{
		Name: name,
		Type: "resource",
	}
	if meta == nil {
		return node
	}
	node.ResourceType = meta.ResourceType
	node.Requirements = meta.Footprint
	node.TierSupport = convertTierSupportMap(meta.PlatformSupport)
	node.Alternatives = collectDependencyAlternatives(meta)
	return node
}

func buildScenarioDependencyNode(scenariosDir, scenarioName string, parentMeta *types.DeploymentDependency, visited map[string]struct{}) types.DeploymentDependencyNode {
	node := types.DeploymentDependencyNode{
		Name: scenarioName,
		Type: "scenario",
	}

	normalized := normalizeName(scenarioName)
	if _, exists := visited[normalized]; exists {
		node.Notes = "cycle detected"
		if parentMeta != nil {
			node.TierSupport = convertTierSupportMap(parentMeta.PlatformSupport)
			node.Requirements = parentMeta.Footprint
			node.Alternatives = collectDependencyAlternatives(parentMeta)
		}
		return node
	}

	if visited == nil {
		visited = map[string]struct{}{}
	}
	visited[normalized] = struct{}{}
	defer delete(visited, normalized)

	scenarioPath := filepath.Join(scenariosDir, scenarioName)
	node.Path = scenarioPath
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		node.Notes = fmt.Sprintf("unable to load scenario: %v", err)
		if parentMeta != nil {
			node.TierSupport = convertTierSupportMap(parentMeta.PlatformSupport)
			node.Requirements = parentMeta.Footprint
			node.Alternatives = collectDependencyAlternatives(parentMeta)
		}
		return node
	}

	var scenarioTierSupport map[string]types.TierSupportSummary
	if cfg.Deployment != nil {
		node.Requirements = cfg.Deployment.AggregateRequirements
		scenarioTierSupport = convertTierTierMap(cfg.Deployment.Tiers)
		node.Alternatives = append(node.Alternatives, collectAdaptationAlternatives(cfg.Deployment.Tiers)...)
	}

	if node.Requirements == nil && parentMeta != nil {
		node.Requirements = parentMeta.Footprint
	}
	fallbackSupport := convertTierSupportMap(nil)
	if parentMeta != nil {
		fallbackSupport = convertTierSupportMap(parentMeta.PlatformSupport)
		node.Alternatives = append(node.Alternatives, collectDependencyAlternatives(parentMeta)...)
	}
	node.TierSupport = mergeTierSupportMaps(scenarioTierSupport, fallbackSupport)
	node.Alternatives = dedupeStrings(node.Alternatives)
	node.Children = buildDependencyNodeList(scenariosDir, scenarioName, cfg, visited)
	return node
}

func convertTierSupportMap(support map[string]types.DependencyTierSupport) map[string]types.TierSupportSummary {
	if len(support) == 0 {
		return nil
	}
	result := make(map[string]types.TierSupportSummary, len(support))
	for tier, value := range support {
		result[tier] = types.TierSupportSummary{
			Supported:    value.Supported,
			FitnessScore: value.FitnessScore,
			Reason:       value.Reason,
			Notes:        value.Notes,
			Requirements: value.Requirements,
			Alternatives: append([]string(nil), value.Alternatives...),
		}
	}
	return result
}

func convertTierTierMap(tiers map[string]types.DeploymentTier) map[string]types.TierSupportSummary {
	if len(tiers) == 0 {
		return nil
	}
	result := make(map[string]types.TierSupportSummary, len(tiers))
	for tier, value := range tiers {
		var supported *bool
		switch strings.ToLower(value.Status) {
		case "ready", "supported":
			flag := true
			supported = &flag
		case "limited", "blocked":
			flag := false
			supported = &flag
		}
		result[tier] = types.TierSupportSummary{
			Supported:    supported,
			FitnessScore: value.FitnessScore,
			Notes:        value.Notes,
			Requirements: value.Requirements,
		}
	}
	return result
}

func mergeTierSupportMaps(preferred, fallback map[string]types.TierSupportSummary) map[string]types.TierSupportSummary {
	if len(preferred) == 0 && len(fallback) == 0 {
		return nil
	}
	merged := make(map[string]types.TierSupportSummary)
	for tier, value := range fallback {
		merged[tier] = value
	}
	for tier, value := range preferred {
		merged[tier] = value
	}
	return merged
}

func mapKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func collectDependencyAlternatives(meta *types.DeploymentDependency) []string {
	if meta == nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, swap := range meta.SwappableWith {
		if swap.ID == "" {
			continue
		}
		set[swap.ID] = struct{}{}
	}
	for _, support := range meta.PlatformSupport {
		for _, alt := range support.Alternatives {
			if alt == "" {
				continue
			}
			set[alt] = struct{}{}
		}
	}
	return mapKeys(set)
}

func collectAdaptationAlternatives(tiers map[string]types.DeploymentTier) []string {
	set := map[string]struct{}{}
	for _, tier := range tiers {
		for _, adaptation := range tier.Adaptations {
			if adaptation.Swap == "" {
				continue
			}
			set[adaptation.Swap] = struct{}{}
		}
	}
	return mapKeys(set)
}

func computeTierAggregates(nodes []types.DeploymentDependencyNode) map[string]types.DeploymentTierAggregate {
	accum := map[string]*tierAccumulator{}
	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		for tier, support := range node.TierSupport {
			acc := accum[tier]
			if acc == nil {
				acc = &tierAccumulator{Blockers: map[string]struct{}{}}
				accum[tier] = acc
			}
			acc.Count++
			req := selectRequirements(node.Requirements, support.Requirements)
			acc.RAM += req.RAMMB
			acc.Disk += req.DiskMB
			acc.CPU += req.CPUCores
			if support.FitnessScore != nil {
				acc.FitnessSum += *support.FitnessScore
			}
			if (support.Supported != nil && !*support.Supported) || (support.FitnessScore != nil && *support.FitnessScore < tierBlockerThreshold) {
				acc.Blockers[node.Name] = struct{}{}
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}

	result := make(map[string]types.DeploymentTierAggregate, len(accum))
	for tier, acc := range accum {
		if acc == nil || acc.Count == 0 {
			continue
		}
		aggregate := types.DeploymentTierAggregate{
			DependencyCount: acc.Count,
			EstimatedRequirements: types.AggregatedRequirements{
				RAMMB:    acc.RAM,
				DiskMB:   acc.Disk,
				CPUCores: acc.CPU,
			},
		}
		aggregate.FitnessScore = acc.FitnessSum / float64(acc.Count)
		aggregate.BlockingDependencies = mapKeys(acc.Blockers)
		sort.Strings(aggregate.BlockingDependencies)
		result[tier] = aggregate
	}
	return result
}

func selectRequirements(base, override *types.DeploymentRequirements) requirementNumbers {
	if override != nil {
		if numbers := requirementsToNumbers(override); numbers.hasValues() {
			return numbers
		}
	}
	return requirementsToNumbers(base)
}

func requirementsToNumbers(req *types.DeploymentRequirements) requirementNumbers {
	var numbers requirementNumbers
	if req == nil {
		return numbers
	}
	if req.RAMMB != nil {
		numbers.RAMMB = *req.RAMMB
	}
	if req.DiskMB != nil {
		numbers.DiskMB = *req.DiskMB
	}
	if req.CPUCores != nil {
		numbers.CPUCores = *req.CPUCores
	}
	return numbers
}

type requirementNumbers struct {
	RAMMB    float64
	DiskMB   float64
	CPUCores float64
}

func (r requirementNumbers) hasValues() bool {
	return r.RAMMB != 0 || r.DiskMB != 0 || r.CPUCores != 0
}

type tierAccumulator struct {
	RAM        float64
	Disk       float64
	CPU        float64
	FitnessSum float64
	Count      int
	Blockers   map[string]struct{}
}

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

func buildBundleManifest(scenarioName, scenarioPath string, generatedAt time.Time, nodes []types.DeploymentDependencyNode) types.BundleManifest {
	return types.BundleManifest{
		Scenario:     scenarioName,
		GeneratedAt:  generatedAt,
		Files:        discoverBundleFiles(scenarioName, scenarioPath),
		Dependencies: flattenBundleDependencies(nodes),
	}
}

func discoverBundleFiles(scenarioName, scenarioPath string) []types.BundleFileEntry {
	candidates := []struct {
		Path  string
		Type  string
		Notes string
	}{
		{Path: filepath.Join(".vrooli", "service.json"), Type: "service-config"},
		{Path: "api", Type: "api-source"},
		{Path: filepath.Join("api", fmt.Sprintf("%s-api", scenarioName)), Type: "api-binary"},
		{Path: filepath.Join("ui", "dist"), Type: "ui-bundle"},
		{Path: filepath.Join("ui", "dist", "index.html"), Type: "ui-entry"},
		{Path: filepath.Join("cli", scenarioName), Type: "cli-binary"},
	}
	entries := make([]types.BundleFileEntry, 0, len(candidates))
	for _, candidate := range candidates {
		absolute := filepath.Join(scenarioPath, candidate.Path)
		_, err := os.Stat(absolute)
		entries = append(entries, types.BundleFileEntry{
			Path:   filepath.ToSlash(candidate.Path),
			Type:   candidate.Type,
			Exists: err == nil,
			Notes:  candidate.Notes,
		})
	}
	return entries
}

func flattenBundleDependencies(nodes []types.DeploymentDependencyNode) []types.BundleDependencyEntry {
	seen := map[string]types.BundleDependencyEntry{}
	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		key := fmt.Sprintf("%s:%s", node.Type, node.Name)
		if _, exists := seen[key]; !exists {
			seen[key] = types.BundleDependencyEntry{
				Name:         node.Name,
				Type:         node.Type,
				ResourceType: node.ResourceType,
				TierSupport:  node.TierSupport,
				Alternatives: dedupeStrings(node.Alternatives),
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	entries := make([]types.BundleDependencyEntry, 0, len(seen))
	for _, entry := range seen {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == entries[j].Type {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Type < entries[j].Type
	})
	return entries
}

func buildDependencyNodeIndex(nodes []types.DeploymentDependencyNode) map[string]types.DeploymentDependencyNode {
	index := map[string]types.DeploymentDependencyNode{}
	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		key := strings.ToLower(node.Name)
		if key != "" {
			if _, exists := index[key]; !exists {
				index[key] = node
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	return index
}

func dedupeStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return mapKeys(set)
}
