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

// analyzeDeploymentGaps crawls the dependency tree and identifies missing deployment metadata.
func analyzeDeploymentGaps(scenarioName, scenariosDir string, nodes []types.DeploymentDependencyNode, knownTiers []string) *types.DeploymentMetadataGaps {
	gapsByScenario := make(map[string]types.ScenarioGapInfo)
	tierSet := make(map[string]struct{})
	for _, tier := range knownTiers {
		tierSet[tier] = struct{}{}
	}

	// Walk the tree recursively
	var walk func(node types.DeploymentDependencyNode, depth int)
	walk = func(node types.DeploymentDependencyNode, depth int) {
		if node.Type != "scenario" {
			return
		}

		// Load the scenario's service.json to check for deployment metadata
		scenarioPath := node.Path
		if scenarioPath == "" {
			scenarioPath = filepath.Join(scenariosDir, node.Name)
		}

		cfg, err := loadServiceConfigFromFile(scenarioPath)
		if err != nil {
			// Can't analyze if we can't load the config
			return
		}

		gap := types.ScenarioGapInfo{
			ScenarioName:            node.Name,
			ScenarioPath:            scenarioPath,
			HasDeploymentBlock:      cfg.Deployment != nil,
			MissingTierDefinitions:  []string{},
			MissingResourceMetadata: []string{},
			MissingScenarioMetadata: []string{},
			SuggestedActions:        []string{},
		}

		if cfg.Deployment == nil {
			gap.SuggestedActions = append(gap.SuggestedActions, "Add deployment block to .vrooli/service.json")
		} else {
			// Check if dependency catalog exists
			hasResourceCatalog := cfg.Deployment.Dependencies.Resources != nil && len(cfg.Deployment.Dependencies.Resources) > 0
			hasScenarioCatalog := cfg.Deployment.Dependencies.Scenarios != nil && len(cfg.Deployment.Dependencies.Scenarios) > 0
			gap.MissingDependencyCatalog = !hasResourceCatalog && !hasScenarioCatalog

			if gap.MissingDependencyCatalog {
				gap.SuggestedActions = append(gap.SuggestedActions, "Add deployment.dependencies catalog for resources/scenarios")
			}

			// Check tier definitions
			if cfg.Deployment.Tiers == nil || len(cfg.Deployment.Tiers) == 0 {
				for tier := range tierSet {
					gap.MissingTierDefinitions = append(gap.MissingTierDefinitions, tier)
				}
				gap.SuggestedActions = append(gap.SuggestedActions, "Define deployment.tiers with fitness scores")
			} else {
				for tier := range tierSet {
					if _, exists := cfg.Deployment.Tiers[tier]; !exists {
						gap.MissingTierDefinitions = append(gap.MissingTierDefinitions, tier)
					}
				}
			}

			// Check which declared resources are missing metadata
			resources := resolvedResourceMap(cfg)
			for resName := range resources {
				if cfg.Deployment.Dependencies.Resources == nil {
					gap.MissingResourceMetadata = append(gap.MissingResourceMetadata, resName)
				} else if _, exists := cfg.Deployment.Dependencies.Resources[resName]; !exists {
					gap.MissingResourceMetadata = append(gap.MissingResourceMetadata, resName)
				}
			}

			// Check which declared scenarios are missing metadata
			if cfg.Dependencies.Scenarios != nil {
				for scenName := range cfg.Dependencies.Scenarios {
					if cfg.Deployment.Dependencies.Scenarios == nil {
						gap.MissingScenarioMetadata = append(gap.MissingScenarioMetadata, scenName)
					} else if _, exists := cfg.Deployment.Dependencies.Scenarios[scenName]; !exists {
						gap.MissingScenarioMetadata = append(gap.MissingScenarioMetadata, scenName)
					}
				}
			}
		}

		// Only add if there are actual gaps
		if !gap.HasDeploymentBlock || gap.MissingDependencyCatalog ||
		   len(gap.MissingTierDefinitions) > 0 ||
		   len(gap.MissingResourceMetadata) > 0 ||
		   len(gap.MissingScenarioMetadata) > 0 {
			gapsByScenario[node.Name] = gap
		}

		// Recurse into children
		for _, child := range node.Children {
			walk(child, depth+1)
		}
	}

	// Walk all top-level nodes
	for _, node := range nodes {
		walk(node, 0)
	}

	// Compute summary statistics
	totalGaps := 0
	scenariosMissingAll := 0
	missingTiersSet := make(map[string]struct{})
	recommendations := []string{}

	for _, gap := range gapsByScenario {
		if !gap.HasDeploymentBlock {
			totalGaps += 10 // Weight heavily
			scenariosMissingAll++
		} else {
			if gap.MissingDependencyCatalog {
				totalGaps++
			}
			totalGaps += len(gap.MissingTierDefinitions)
			totalGaps += len(gap.MissingResourceMetadata)
			totalGaps += len(gap.MissingScenarioMetadata)
		}

		for _, tier := range gap.MissingTierDefinitions {
			missingTiersSet[tier] = struct{}{}
		}
	}

	// Generate recommendations
	if scenariosMissingAll > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d scenario(s) missing deployment blocks entirely - run scan --apply to initialize", scenariosMissingAll))
	}
	if len(missingTiersSet) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Add tier definitions for: %v", mapKeys(missingTiersSet)))
	}
	if totalGaps > 0 {
		recommendations = append(recommendations,
			"Review gaps_by_scenario for detailed per-scenario action items")
	}

	return &types.DeploymentMetadataGaps{
		TotalGaps:           totalGaps,
		ScenariosMissingAll: scenariosMissingAll,
		GapsByScenario:      gapsByScenario,
		MissingTiers:        mapKeys(missingTiersSet),
		Recommendations:     recommendations,
	}
}
