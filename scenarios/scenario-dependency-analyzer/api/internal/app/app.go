package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	orderedmap "github.com/iancoleman/orderedmap"

	pq "github.com/lib/pq"
	appconfig "scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

const (
	deploymentReportVersion = 1
	tierBlockerThreshold    = 0.75
)

// Database connection and detection helpers
var (
	db                       *sql.DB
	applyDiffsHook           func(string, *types.ServiceConfig)
	resourceCommandPattern   = regexp.MustCompile(`resource-([a-z0-9-]+)`)
	resourceHeuristicCatalog = []resourceHeuristic{
		{
			Name: "postgres",
			Type: "postgres",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`postgres(ql)?:\/\/`),
				regexp.MustCompile(`PGHOST`),
			},
		},
		{
			Name: "redis",
			Type: "redis",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`redis:\/\/`),
				regexp.MustCompile(`REDIS_URL`),
			},
		},
		{
			Name: "ollama",
			Type: "ollama",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`ollama`),
			},
		},
		{
			Name: "qdrant",
			Type: "qdrant",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`qdrant`),
			},
		},
		{
			Name: "n8n",
			Type: "n8n",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`resource-?n8n`),
				regexp.MustCompile(`N8N_[A-Z0-9_]+`),
				regexp.MustCompile(`RESOURCE_PORT_N8N`),
				regexp.MustCompile(`n8n/(?:api|webhook)`),
			},
		},
		{
			Name: "minio",
			Type: "minio",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`minio`),
			},
		},
	}

	dependencyCatalogMu     sync.RWMutex
	dependencyCatalogLoaded bool
	knownScenarioNames      map[string]struct{}
	knownResourceNames      map[string]struct{}
	analysisIgnoreSegments  = map[string]struct{}{
		"docs":          {},
		"doc":           {},
		"documentation": {},
		"readme":        {},
		"test":          {},
		"tests":         {},
		"testdata":      {},
		"__tests__":     {},
		"spec":          {},
		"specs":         {},
		"coverage":      {},
		"examples":      {},
		"playbooks":     {},
		"data":          {},
		"draft":         {},
		"drafts":        {},
		"prd-drafts":    {},
		"dist":          {},
		"build":         {},
		"out":           {},
		"outputs":       {},
	}

	skipDirectoryNames = map[string]struct{}{
		"node_modules":     {},
		"dist":             {},
		"build":            {},
		"coverage":         {},
		"logs":             {},
		"tmp":              {},
		"temp":             {},
		"vendor":           {},
		"__pycache__":      {},
		".pytest_cache":    {},
		".nyc_output":      {},
		"storybook-static": {},
		".next":            {},
		".nuxt":            {},
		".svelte-kit":      {},
		".vercel":          {},
		".parcel-cache":    {},
		".turbo":           {},
		".git":             {},
		".hg":              {},
		".svn":             {},
		".idea":            {},
		".vscode":          {},
		".cache":           {},
		".output":          {},
		".yalc":            {},
		".yarn":            {},
		".pnpm":            {},
	}
	docExtensions = map[string]struct{}{
		".md":  {},
		".mdx": {},
		".rst": {},
		".txt": {},
	}
	docFileNames = map[string]struct{}{
		"readme":           {},
		"readme.md":        {},
		"readme.mdx":       {},
		"prd.md":           {},
		"prd.mdx":          {},
		"problems.md":      {},
		"requirements.md":  {},
		"requirements.mdx": {},
	}
	scenarioPortCallPattern       = regexp.MustCompile(`resolveScenarioPortViaCLI\s*\(\s*[^,]+,\s*(?:"([a-z0-9-]+)"|([A-Za-z0-9_]+))\s*,`)
	scenarioAliasDeclPattern      = regexp.MustCompile(`(?m)(?:const|var)\s+([A-Za-z0-9_]+)\s*=\s*"([a-z0-9-]+)"`)
	scenarioAliasShortPattern     = regexp.MustCompile(`(?m)([A-Za-z0-9_]+)\s*:=\s*"([a-z0-9-]+)"`)
	scenarioAliasBlockPattern     = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\s*=\s*"([a-z0-9-]+)"`)
	resourceCLIDirectoryAllowList = map[string]struct{}{
		"":               {},
		"api":            {},
		"cli":            {},
		"cmd":            {},
		"scripts":        {},
		"script":         {},
		"ui":             {},
		"src":            {},
		"server":         {},
		"services":       {},
		"service":        {},
		"lib":            {},
		"pkg":            {},
		"internal":       {},
		"tools":          {},
		"initialization": {},
		"automation":     {},
		"test":           {},
		"tests":          {},
		"integration":    {},
		"config":         {},
	}
)

type resourceHeuristic struct {
	Name     string
	Type     string
	Patterns []*regexp.Regexp
}

func ensureDependencyCatalogs() {
	dependencyCatalogMu.RLock()
	if dependencyCatalogLoaded {
		dependencyCatalogMu.RUnlock()
		return
	}
	dependencyCatalogMu.RUnlock()

	dependencyCatalogMu.Lock()
	defer dependencyCatalogMu.Unlock()
	if dependencyCatalogLoaded {
		return
	}

	scenariosDir := determineScenariosDir()
	knownScenarioNames = discoverAvailableScenarios(scenariosDir)
	resourcesDir := filepath.Join(filepath.Dir(scenariosDir), "resources")
	knownResourceNames = discoverAvailableResources(resourcesDir)
	dependencyCatalogLoaded = true
}

func determineScenariosDir() string {
	dir := os.Getenv("VROOLI_SCENARIOS_DIR")
	if dir == "" {
		dir = "../.."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return absDir
}

func discoverAvailableScenarios(dir string) map[string]struct{} {
	results := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("⚠️  Unable to read scenarios directory %s: %v", dir, err)
		return results
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		servicePath := filepath.Join(dir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(servicePath); err == nil {
			results[normalizeName(entry.Name())] = struct{}{}
		}
	}
	return results
}

func discoverAvailableResources(dir string) map[string]struct{} {
	results := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("⚠️  Unable to read resources directory %s: %v", dir, err)
		return results
	}
	for _, entry := range entries {
		if entry.IsDir() {
			results[normalizeName(entry.Name())] = struct{}{}
		}
	}
	return results
}

func shouldIgnoreDetectionFile(relPath string) bool {
	if relPath == "" {
		return false
	}
	lowerPath := strings.ToLower(relPath)
	base := strings.ToLower(filepath.Base(lowerPath))
	if _, ok := docFileNames[base]; ok {
		return true
	}
	if ext := strings.ToLower(filepath.Ext(base)); ext != "" {
		if _, ok := docExtensions[ext]; ok {
			return true
		}
	}
	if strings.HasPrefix(base, "readme") {
		return true
	}
	segments := strings.Split(lowerPath, string(filepath.Separator))
	for _, segment := range segments {
		if _, ok := analysisIgnoreSegments[segment]; ok {
			return true
		}
	}
	return false
}

func isAllowedResourceCLIPath(relPath string) bool {
	if relPath == "" {
		return true
	}
	segments := strings.Split(relPath, string(filepath.Separator))
	if len(segments) == 0 {
		return true
	}
	root := strings.ToLower(strings.TrimSpace(segments[0]))
	if root == "." {
		root = ""
	}
	_, ok := resourceCLIDirectoryAllowList[root]
	return ok
}

func isKnownScenario(name string) bool {
	ensureDependencyCatalogs()
	dependencyCatalogMu.RLock()
	defer dependencyCatalogMu.RUnlock()
	_, ok := knownScenarioNames[normalizeName(name)]
	return ok
}

func isKnownResource(name string) bool {
	ensureDependencyCatalogs()
	dependencyCatalogMu.RLock()
	defer dependencyCatalogMu.RUnlock()
	_, ok := knownResourceNames[normalizeName(name)]
	return ok
}

func refreshDependencyCatalogs() {
	dependencyCatalogMu.Lock()
	defer dependencyCatalogMu.Unlock()
	dependencyCatalogLoaded = false
	knownScenarioNames = nil
	knownResourceNames = nil
}

func normalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func shouldSkipDirectoryEntry(d fs.DirEntry) bool {
	if !d.IsDir() {
		return false
	}
	name := d.Name()
	if _, ok := skipDirectoryNames[strings.ToLower(name)]; ok {
		return true
	}
	if strings.HasPrefix(strings.ToLower(name), "node_modules") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(name), ".ignored") {
		return true
	}
	if strings.HasPrefix(name, ".") && name != ".vrooli" {
		return true
	}
	return false
}

// Core analysis functions
func analyzeScenario(scenarioName string) (*types.DependencyAnalysisResponse, error) {
	cfg := appconfig.Load()
	scenarioPath := filepath.Join(cfg.ScenariosDir, scenarioName)
	serviceConfig, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return nil, err
	}

	response := &types.DependencyAnalysisResponse{
		Scenario:              scenarioName,
		Resources:             []types.ScenarioDependency{},
		DetectedResources:     []types.ScenarioDependency{},
		Scenarios:             []types.ScenarioDependency{},
		DeclaredScenarioSpecs: map[string]types.ScenarioDependencySpec{},
		SharedWorkflows:       []types.ScenarioDependency{},
		TransitiveDepth:       0,
		ResourceDiff:          types.DependencyDiff{},
		ScenarioDiff:          types.DependencyDiff{},
	}

	declaredResources := extractDeclaredResources(scenarioName, serviceConfig)
	response.Resources = declaredResources
	response.DeclaredScenarioSpecs = normalizeScenarioSpecs(serviceConfig.Dependencies.Scenarios)

	detectedResources, err := scanForResourceUsage(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for resource usage: %v", err)
	} else {
		response.DetectedResources = detectedResources
	}

	scenarioDeps, err := scanForScenarioDependencies(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for scenario dependencies: %v", err)
	} else {
		response.Scenarios = append(response.Scenarios, scenarioDeps...)
	}

	workflowDeps, err := scanForSharedWorkflows(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for shared workflows: %v", err)
	} else {
		response.SharedWorkflows = append(response.SharedWorkflows, workflowDeps...)
	}

	declaredScenarioDeps := convertDeclaredScenariosToDependencies(scenarioName, response.DeclaredScenarioSpecs)
	response.ResourceDiff = buildResourceDiff(resolvedResourceMap(serviceConfig), response.DetectedResources)
	response.ScenarioDiff = buildScenarioDiff(response.DeclaredScenarioSpecs, response.Scenarios)

	if err := storeDependencies(response, declaredScenarioDeps); err != nil {
		log.Printf("Warning: failed to store dependencies in database: %v", err)
	}

	if err := updateScenarioMetadata(scenarioName, serviceConfig, scenarioPath); err != nil {
		log.Printf("Warning: failed to update scenario metadata: %v", err)
	}

	if deploymentReport := buildDeploymentReport(scenarioName, scenarioPath, cfg.ScenariosDir, serviceConfig); deploymentReport != nil {
		response.DeploymentReport = deploymentReport
		if err := persistDeploymentReport(scenarioPath, deploymentReport); err != nil {
			log.Printf("Warning: failed to persist deployment report: %v", err)
		}
	}

	return response, nil
}

func runScenarioOptimization(scenarioName string, req types.OptimizationRequest) (*types.OptimizationResult, error) {
	analysis, err := analyzeScenario(scenarioName)
	if err != nil {
		return nil, err
	}
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	svcCfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		svcCfg = nil
	}
	recommendations := generateOptimizationRecommendations(scenarioName, analysis, svcCfg)
	if err := persistOptimizationRecommendations(scenarioName, recommendations); err != nil {
		return nil, err
	}
	var applySummary map[string]interface{}
	applied := false
	if req.Apply && len(recommendations) > 0 {
		applySummary, err = applyOptimizationRecommendations(scenarioName, recommendations)
		if err != nil {
			return nil, err
		}
		if changed, ok := applySummary["changed"].(bool); ok && changed {
			applied = true
			analysis, err = analyzeScenario(scenarioName)
			if err != nil {
				return nil, err
			}
			recommendations = generateOptimizationRecommendations(scenarioName, analysis, svcCfg)
			if err := persistOptimizationRecommendations(scenarioName, recommendations); err != nil {
				return nil, err
			}
		}
	}
	return &types.OptimizationResult{
		Scenario:          scenarioName,
		Recommendations:   recommendations,
		Summary:           buildOptimizationSummary(recommendations),
		Applied:           applied,
		ApplySummary:      applySummary,
		AnalysisTimestamp: time.Now().UTC(),
	}, nil
}

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

func generateOptimizationRecommendations(scenarioName string, analysis *types.DependencyAnalysisResponse, cfg *types.ServiceConfig) []types.OptimizationRecommendation {
	if analysis == nil {
		return nil
	}
	timestamp := time.Now().UTC()
	recommendations := []types.OptimizationRecommendation{}
	recommendations = append(recommendations, buildUnusedResourceRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildUnusedScenarioRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildTierBlockerRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildSecretStrategyRecommendations(scenarioName, cfg, timestamp)...)
	sort.SliceStable(recommendations, func(i, j int) bool {
		if recommendations[i].Priority == recommendations[j].Priority {
			return recommendations[i].Title < recommendations[j].Title
		}
		priorityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		return priorityOrder[strings.ToLower(recommendations[i].Priority)] < priorityOrder[strings.ToLower(recommendations[j].Priority)]
	})
	return recommendations
}

func buildUnusedResourceRecommendations(scenarioName string, analysis *types.DependencyAnalysisResponse, timestamp time.Time) []types.OptimizationRecommendation {
	if analysis == nil || len(analysis.ResourceDiff.Extra) == 0 {
		return nil
	}
	recommendations := make([]types.OptimizationRecommendation, 0, len(analysis.ResourceDiff.Extra))
	for _, drift := range analysis.ResourceDiff.Extra {
		name := strings.TrimSpace(drift.Name)
		if name == "" {
			continue
		}
		recommendations = append(recommendations, types.OptimizationRecommendation{
			ID:                 uuid.New().String(),
			ScenarioName:       scenarioName,
			RecommendationType: "dependency_reduction",
			Title:              fmt.Sprintf("Remove unused resource '%s'", name),
			Description:        fmt.Sprintf("Resource '%s' is declared but not detected in the scenario codebase.", name),
			CurrentState: map[string]interface{}{
				"declared": true,
				"detected": false,
				"details":  drift.Details,
			},
			RecommendedState: map[string]interface{}{
				"action":        "remove_resource",
				"resource_name": name,
			},
			EstimatedImpact: map[string]interface{}{
				"complexity_score": -1,
				"description":      "Removes unused infrastructure dependency",
			},
			ConfidenceScore: 0.85,
			Priority:        "medium",
			Status:          "pending",
			CreatedAt:       timestamp,
		})
	}
	return recommendations
}

func buildUnusedScenarioRecommendations(scenarioName string, analysis *types.DependencyAnalysisResponse, timestamp time.Time) []types.OptimizationRecommendation {
	if analysis == nil || len(analysis.ScenarioDiff.Extra) == 0 {
		return nil
	}
	recommendations := make([]types.OptimizationRecommendation, 0, len(analysis.ScenarioDiff.Extra))
	for _, drift := range analysis.ScenarioDiff.Extra {
		name := strings.TrimSpace(drift.Name)
		if name == "" {
			continue
		}
		recommendations = append(recommendations, types.OptimizationRecommendation{
			ID:                 uuid.New().String(),
			ScenarioName:       scenarioName,
			RecommendationType: "dependency_reduction",
			Title:              fmt.Sprintf("Remove unused scenario '%s'", name),
			Description:        fmt.Sprintf("Scenario dependency '%s' is declared but not referenced in the codebase.", name),
			CurrentState: map[string]interface{}{
				"declared": true,
				"detected": false,
			},
			RecommendedState: map[string]interface{}{
				"action":          "remove_scenario_dependency",
				"scenario_name":   name,
				"dependency_type": "scenario",
			},
			EstimatedImpact: map[string]interface{}{
				"complexity_score": -1,
				"description":      "Removes unused scenario coupling",
			},
			ConfidenceScore: 0.8,
			Priority:        "medium",
			Status:          "pending",
			CreatedAt:       timestamp,
		})
	}
	return recommendations
}

func buildTierBlockerRecommendations(scenarioName string, analysis *types.DependencyAnalysisResponse, timestamp time.Time) []types.OptimizationRecommendation {
	report := analysis.DeploymentReport
	if report == nil || len(report.Dependencies) == 0 {
		return nil
	}
	nodeIndex := buildDependencyNodeIndex(report.Dependencies)
	recommendations := []types.OptimizationRecommendation{}
	blockingByTier := map[string][]string{}
	for tier, aggregate := range report.Aggregates {
		for _, blocker := range aggregate.BlockingDependencies {
			key := strings.ToLower(blocker)
			blockingByTier[key] = append(blockingByTier[key], tier)
		}
	}
	for blockerKey, tiers := range blockingByTier {
		node, ok := nodeIndex[blockerKey]
		if !ok {
			continue
		}
		for _, tier := range tiers {
			recommendations = append(recommendations, buildBlockerRecommendation(scenarioName, node, tier, timestamp))
		}
	}
	return recommendations
}

func buildSecretStrategyRecommendations(scenarioName string, cfg *types.ServiceConfig, timestamp time.Time) []types.OptimizationRecommendation {
	if cfg == nil || cfg.Deployment == nil || len(cfg.Deployment.Tiers) == 0 {
		return nil
	}
	recommendations := []types.OptimizationRecommendation{}
	for tierName, tier := range cfg.Deployment.Tiers {
		for _, secret := range tier.Secrets {
			if strings.TrimSpace(secret.StrategyRef) != "" {
				continue
			}
			recommendations = append(recommendations, types.OptimizationRecommendation{
				ID:                 uuid.New().String(),
				ScenarioName:       scenarioName,
				RecommendationType: "performance_improvement",
				Title:              fmt.Sprintf("Annotate secret '%s' on %s", secret.SecretID, tierName),
				Description:        fmt.Sprintf("Secret '%s' on tier '%s' lacks a strategy reference. Link it to a secrets-manager playbook to unblock deployment readiness.", secret.SecretID, tierName),
				CurrentState: map[string]interface{}{
					"tier":           tierName,
					"secret_id":      secret.SecretID,
					"classification": secret.Classification,
				},
				RecommendedState: map[string]interface{}{
					"action":    "annotate_secret_strategy",
					"tier":      tierName,
					"secret_id": secret.SecretID,
					"guidance":  "Use secrets-manager playbooks to define strategy_ref",
				},
				EstimatedImpact: map[string]interface{}{
					"risk":        "secrets_gap",
					"description": "Missing secret strategies block deployment",
				},
				ConfidenceScore: 0.75,
				Priority:        "high",
				Status:          "pending",
				CreatedAt:       timestamp,
			})
		}
	}
	return recommendations
}

func buildBlockerRecommendation(scenarioName string, node types.DeploymentDependencyNode, tier string, timestamp time.Time) types.OptimizationRecommendation {
	alt := ""
	if len(node.Alternatives) > 0 {
		alt = node.Alternatives[0]
	}
	tierSupport := map[string]interface{}{}
	if node.TierSupport != nil {
		if support, ok := node.TierSupport[tier]; ok {
			tierSupport = map[string]interface{}{
				"supported":     support.Supported,
				"fitness_score": support.FitnessScore,
				"notes":         support.Notes,
				"alternatives":  support.Alternatives,
			}
		}
	}
	priority := "high"
	if node.Type != "resource" {
		priority = "medium"
	}
	estImpact := map[string]interface{}{
		"tier":       tier,
		"risk":       "deployment_blocker",
		"dependency": node.Name,
	}
	return types.OptimizationRecommendation{
		ID:                 uuid.New().String(),
		ScenarioName:       scenarioName,
		RecommendationType: "resource_swap",
		Title:              fmt.Sprintf("Resolve %s blocker on %s", node.Name, tier),
		Description:        fmt.Sprintf("Dependency '%s' blocks tier '%s'. Consider annotated alternatives or lighter swaps to restore fitness.", node.Name, tier),
		CurrentState: map[string]interface{}{
			"dependency":   node.Name,
			"tier":         tier,
			"tier_support": tierSupport,
		},
		RecommendedState: map[string]interface{}{
			"action":                "swap_dependency",
			"dependency":            node.Name,
			"tier":                  tier,
			"suggested_alternative": alt,
		},
		EstimatedImpact: estImpact,
		ConfidenceScore: 0.7,
		Priority:        priority,
		Status:          "pending",
		CreatedAt:       timestamp,
	}
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

func scanForScenarioDependencies(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	var dependencies []types.ScenarioDependency
	ensureDependencyCatalogs()
	scenarioNameNormalized := normalizeName(scenarioName)
	aliasCatalog := buildScenarioAliasCatalog(scenarioPath)

	// Pattern to match vrooli scenario commands
	scenarioPattern := regexp.MustCompile(`vrooli\s+scenario\s+(?:run|test|status)\s+([a-z0-9-]+)`)

	// Pattern to match direct CLI calls to other scenarios
	cliPattern := regexp.MustCompile(`([a-z0-9-]+)-cli\.sh|\b([a-z0-9-]+)\s+(?:analyze|process|generate|run)`)

	// Walk through all files in the scenario
	err := filepath.WalkDir(scenarioPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		if d.IsDir() && path != scenarioPath && shouldSkipDirectoryEntry(d) {
			return filepath.SkipDir
		}

		// Only scan certain file types
		ext := strings.ToLower(filepath.Ext(path))
		if !contains([]string{".go", ".js", ".sh", ".py", ".md"}, ext) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		contentStr := string(content)
		relPath, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			relPath = path
		}
		if shouldIgnoreDetectionFile(relPath) {
			return nil
		}

		// Find scenario references
		matches := scenarioPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range matches {
			if len(match) > 1 {
				depName := normalizeName(match[1])
				if depName == scenarioNameNormalized || !isKnownScenario(depName) {
					continue
				}
				dep := types.ScenarioDependency{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					DependencyType: "scenario",
					DependencyName: depName,
					Required:       false, // Inter-scenario deps are typically optional
					Purpose:        fmt.Sprintf("Referenced in %s", filepath.Base(path)),
					AccessMethod:   "vrooli scenario",
					Configuration: map[string]interface{}{
						"found_in_file": relPath,
						"pattern_type":  "vrooli_cli",
					},
					DiscoveredAt: time.Now(),
					LastVerified: time.Now(),
				}
				dependencies = append(dependencies, dep)
			}
		}

		// Find CLI references
		cliMatches := cliPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range cliMatches {
			var scenarioRef string
			if match[1] != "" {
				scenarioRef = match[1]
			} else if match[2] != "" {
				scenarioRef = match[2]
			}

			scenarioRef = normalizeName(scenarioRef)
			if scenarioRef == "" || scenarioRef == scenarioNameNormalized || !isKnownScenario(scenarioRef) {
				continue
			}
			dep := types.ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "scenario",
				DependencyName: scenarioRef,
				Required:       false,
				Purpose:        fmt.Sprintf("CLI reference in %s", filepath.Base(path)),
				AccessMethod:   "direct_cli",
				Configuration: map[string]interface{}{
					"found_in_file": relPath,
					"pattern_type":  "cli_reference",
				},
				DiscoveredAt: time.Now(),
				LastVerified: time.Now(),
			}
			dependencies = append(dependencies, dep)
		}

		// Find scenario references via resolveScenarioPortViaCLI helpers
		portMatches := scenarioPortCallPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range portMatches {
			var depName string
			if len(match) > 1 && match[1] != "" {
				depName = normalizeName(match[1])
			} else if len(match) > 2 && match[2] != "" {
				if aliasDep, ok := aliasCatalog[match[2]]; ok {
					depName = aliasDep
				} else {
					depName = normalizeName(match[2])
				}
			}

			if depName == "" || depName == scenarioNameNormalized || !isKnownScenario(depName) {
				continue
			}

			dep := types.ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "scenario",
				DependencyName: depName,
				Required:       true,
				Purpose:        fmt.Sprintf("References %s port via CLI", depName),
				AccessMethod:   "scenario_port_cli",
				Configuration: map[string]interface{}{
					"found_in_file": relPath,
					"pattern_type":  "resolve_cli_port",
				},
				DiscoveredAt: time.Now(),
				LastVerified: time.Now(),
			}
			dependencies = append(dependencies, dep)
		}

		return nil
	})

	return dependencies, err
}

func buildScenarioAliasCatalog(scenarioPath string) map[string]string {
	aliases := map[string]string{}
	addAlias := func(identifier, scenario string) {
		if identifier == "" {
			return
		}
		normalized := normalizeName(scenario)
		if !isKnownScenario(normalized) {
			return
		}
		aliases[identifier] = normalized
	}

	filepath.WalkDir(scenarioPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != scenarioPath && shouldSkipDirectoryEntry(d) {
			return filepath.SkipDir
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !contains([]string{".go", ".js", ".sh", ".py", ".md"}, ext) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		contentStr := string(content)
		relPath, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			relPath = path
		}
		if shouldIgnoreDetectionFile(relPath) {
			return nil
		}
		for _, match := range scenarioAliasDeclPattern.FindAllStringSubmatch(contentStr, -1) {
			if len(match) < 3 {
				continue
			}
			addAlias(match[1], match[2])
		}
		for _, match := range scenarioAliasShortPattern.FindAllStringSubmatch(contentStr, -1) {
			if len(match) < 3 {
				continue
			}
			addAlias(match[1], match[2])
		}
		for _, match := range scenarioAliasBlockPattern.FindAllStringSubmatch(contentStr, -1) {
			if len(match) < 3 {
				continue
			}
			addAlias(match[1], match[2])
		}
		return nil
	})

	return aliases
}

func scanForSharedWorkflows(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	var dependencies []types.ScenarioDependency

	// Look for references to shared workflows in initialization directory
	initPath := filepath.Join(scenarioPath, "initialization")
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		return dependencies, nil
	}

	// Pattern to match shared workflow references
	sharedPattern := regexp.MustCompile(`initialization/(?:automation/)?(?:n8n|huginn|windmill)/([^/]+\.json)`)

	err := filepath.WalkDir(initPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Check if this is a workflow file that might reference shared workflows
		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			matches := sharedPattern.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) > 1 {
					dep := types.ScenarioDependency{
						ID:             uuid.New().String(),
						ScenarioName:   scenarioName,
						DependencyType: "shared_workflow",
						DependencyName: match[1],
						Required:       true, // Shared workflows are typically required
						Purpose:        "Shared workflow dependency",
						AccessMethod:   "workflow_trigger",
						Configuration: map[string]interface{}{
							"found_in_file": strings.TrimPrefix(path, scenarioPath),
							"workflow_type": strings.Split(match[0], "/")[1],
						},
						DiscoveredAt: time.Now(),
						LastVerified: time.Now(),
					}
					dependencies = append(dependencies, dep)
				}
			}
		}

		return nil
	})

	return dependencies, err
}

func loadServiceConfigFromFile(scenarioPath string) (*types.ServiceConfig, error) {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("scenario %s not found or missing service.json", filepath.Base(scenarioPath))
	}

	configData, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service.json: %w", err)
	}

	var serviceConfig types.ServiceConfig
	if err := json.Unmarshal(configData, &serviceConfig); err != nil {
		return nil, fmt.Errorf("failed to parse service.json: %w", err)
	}

	return &serviceConfig, nil
}

func collectAnalysisMetrics() (gin.H, error) {
	metrics := gin.H{
		"scenarios_found":     0,
		"resources_available": 0,
		"database_status":     "unknown",
		"last_analysis":       nil,
	}

	if db == nil {
		return metrics, fmt.Errorf("database connection not initialized")
	}

	if err := db.Ping(); err != nil {
		metrics["database_status"] = "unreachable"
		return metrics, err
	}

	metrics["database_status"] = "connected"

	var scenarioCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM scenario_metadata").Scan(&scenarioCount); err != nil {
		metrics["database_status"] = "error"
		return metrics, err
	}
	metrics["scenarios_found"] = scenarioCount

	var resourceCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM scenario_dependencies WHERE dependency_type = 'resource'").Scan(&resourceCount); err != nil {
		metrics["database_status"] = "error"
		return metrics, err
	}
	metrics["resources_available"] = resourceCount

	var lastAnalysis sql.NullTime
	if err := db.QueryRow("SELECT MAX(last_scanned) FROM scenario_metadata").Scan(&lastAnalysis); err == nil && lastAnalysis.Valid {
		metrics["last_analysis"] = lastAnalysis.Time.UTC().Format(time.RFC3339)
	}

	return metrics, nil
}

func resolvedResourceMap(cfg *types.ServiceConfig) map[string]types.Resource {
	if cfg.Dependencies.Resources != nil && len(cfg.Dependencies.Resources) > 0 {
		return cfg.Dependencies.Resources
	}
	if cfg.Resources == nil {
		return map[string]types.Resource{}
	}
	return cfg.Resources
}

func extractDeclaredResources(scenarioName string, cfg *types.ServiceConfig) []types.ScenarioDependency {
	resources := resolvedResourceMap(cfg)
	declared := make([]types.ScenarioDependency, 0, len(resources))
	for resourceName, resource := range resources {
		dep := types.ScenarioDependency{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			DependencyType: "resource",
			DependencyName: resourceName,
			Required:       resource.Required,
			Purpose:        resource.Purpose,
			AccessMethod:   fmt.Sprintf("resource-%s", resourceName),
			Configuration: map[string]interface{}{
				"type":           resource.Type,
				"enabled":        resource.Enabled,
				"initialization": resource.Initialization,
				"models":         resource.Models,
				"source":         "declared",
			},
			DiscoveredAt: time.Now(),
			LastVerified: time.Now(),
		}
		declared = append(declared, dep)
	}
	sort.Slice(declared, func(i, j int) bool {
		return declared[i].DependencyName < declared[j].DependencyName
	})
	return declared
}

func normalizeScenarioSpecs(specs map[string]types.ScenarioDependencySpec) map[string]types.ScenarioDependencySpec {
	if specs == nil {
		return map[string]types.ScenarioDependencySpec{}
	}
	return specs
}

func convertDeclaredScenariosToDependencies(scenarioName string, specs map[string]types.ScenarioDependencySpec) []types.ScenarioDependency {
	declared := make([]types.ScenarioDependency, 0, len(specs))
	for depName, spec := range specs {
		dep := types.ScenarioDependency{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			DependencyType: "scenario",
			DependencyName: depName,
			Required:       spec.Required,
			Purpose:        spec.Description,
			AccessMethod:   "declared",
			Configuration: map[string]interface{}{
				"source":        "declared",
				"version":       spec.Version,
				"version_range": spec.VersionRange,
			},
			DiscoveredAt: time.Now(),
			LastVerified: time.Now(),
		}
		declared = append(declared, dep)
	}
	return declared
}

func buildResourceDiff(declared map[string]types.Resource, detected []types.ScenarioDependency) types.DependencyDiff {
	declaredSet := map[string]types.Resource{}
	for name, cfg := range declared {
		declaredSet[name] = cfg
	}
	detectedSet := map[string]types.ScenarioDependency{}
	for _, dep := range detected {
		detectedSet[dep.DependencyName] = dep
	}

	missing := []types.DependencyDrift{}
	for name, dep := range detectedSet {
		if _, ok := declaredSet[name]; !ok {
			missing = append(missing, types.DependencyDrift{
				Name:    name,
				Details: dep.Configuration,
			})
		}
	}

	extra := []types.DependencyDrift{}
	for name, cfg := range declaredSet {
		if _, ok := detectedSet[name]; !ok {
			extra = append(extra, types.DependencyDrift{
				Name: name,
				Details: map[string]interface{}{
					"type":     cfg.Type,
					"required": cfg.Required,
				},
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Name < missing[j].Name })
	sort.Slice(extra, func(i, j int) bool { return extra[i].Name < extra[j].Name })

	return types.DependencyDiff{Missing: missing, Extra: extra}
}

func buildScenarioDiff(declared map[string]types.ScenarioDependencySpec, detected []types.ScenarioDependency) types.DependencyDiff {
	declaredSet := map[string]types.ScenarioDependencySpec{}
	for name, spec := range declared {
		declaredSet[name] = spec
	}
	detectedSet := map[string]types.ScenarioDependency{}
	for _, dep := range detected {
		detectedSet[dep.DependencyName] = dep
	}

	missing := []types.DependencyDrift{}
	for name, dep := range detectedSet {
		if _, ok := declaredSet[name]; !ok {
			missing = append(missing, types.DependencyDrift{
				Name:    name,
				Details: dep.Configuration,
			})
		}
	}

	extra := []types.DependencyDrift{}
	for name, spec := range declaredSet {
		if _, ok := detectedSet[name]; !ok {
			extra = append(extra, types.DependencyDrift{
				Name: name,
				Details: map[string]interface{}{
					"required": spec.Required,
					"version":  spec.Version,
				},
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Name < missing[j].Name })
	sort.Slice(extra, func(i, j int) bool { return extra[i].Name < extra[j].Name })

	return types.DependencyDiff{Missing: missing, Extra: extra}
}

func updateScenarioMetadata(name string, cfg *types.ServiceConfig, scenarioPath string) error {
	if db == nil {
		return nil
	}

	serviceConfigJSON, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO scenario_metadata (scenario_name, display_name, description, tags, service_config, file_path, last_scanned)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (scenario_name) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			tags = EXCLUDED.tags,
			service_config = EXCLUDED.service_config,
			file_path = EXCLUDED.file_path,
			last_scanned = EXCLUDED.last_scanned,
			updated_at = NOW();
	`

	_, err = db.Exec(query,
		name,
		cfg.Service.DisplayName,
		cfg.Service.Description,
		pqArray(cfg.Service.Tags),
		serviceConfigJSON,
		scenarioPath,
		time.Now(),
	)
	return err
}

func pqArray(values []string) interface{} {
	if len(values) == 0 {
		return pq.StringArray([]string{})
	}
	return pq.StringArray(values)
}

func scanForResourceUsage(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	results := map[string]types.ScenarioDependency{}
	ensureDependencyCatalogs()

	relevantExts := []string{".go", ".js", ".ts", ".tsx", ".sh", ".py", ".md", ".json", ".yml", ".yaml"}

	recordDetection := func(name, method, pattern, file string, resourceType string) {
		canonical := normalizeName(name)
		if canonical == "" || !isKnownResource(canonical) {
			return
		}
		existing, ok := results[canonical]
		if !ok {
			existing = types.ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "resource",
				DependencyName: canonical,
				Required:       true,
				Purpose:        "Detected via static analysis",
				AccessMethod:   method,
				Configuration:  map[string]interface{}{"source": "detected"},
				DiscoveredAt:   time.Now(),
				LastVerified:   time.Now(),
			}
		}

		if existing.Configuration == nil {
			existing.Configuration = map[string]interface{}{}
		}
		existing.Configuration["resource_type"] = resourceType
		match := map[string]interface{}{
			"pattern": pattern,
			"method":  method,
			"file":    file,
		}
		matches, _ := existing.Configuration["matches"].([]map[string]interface{})
		matches = append(matches, match)
		existing.Configuration["matches"] = matches
		results[canonical] = existing
	}

	err := filepath.WalkDir(scenarioPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != scenarioPath && shouldSkipDirectoryEntry(d) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !contains(relevantExts, ext) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		contentStr := string(content)
		relPath, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			relPath = path
		}
		if shouldIgnoreDetectionFile(relPath) {
			return nil
		}

		cmdMatches := resourceCommandPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range cmdMatches {
			if len(match) > 1 {
				if !isAllowedResourceCLIPath(relPath) {
					continue
				}
				resourceName := normalizeName(match[1])
				recordDetection(resourceName, "resource_cli", "resource-cli", relPath, resourceName)
			}
		}

		for _, heuristic := range resourceHeuristicCatalog {
			for _, pattern := range heuristic.Patterns {
				if pattern.MatchString(contentStr) {
					recordDetection(heuristic.Name, "heuristic", pattern.String(), relPath, heuristic.Type)
					break
				}
			}
		}

		return nil
	})

	augmentResourceDetectionsWithInitialization(results, scenarioPath, scenarioName)

	deps := make([]types.ScenarioDependency, 0, len(results))
	for _, dep := range results {
		deps = append(deps, dep)
	}
	return deps, err
}

func augmentResourceDetectionsWithInitialization(results map[string]types.ScenarioDependency, scenarioPath, scenarioName string) {
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return
	}

	resources := resolvedResourceMap(cfg)
	for resourceName, resource := range resources {
		if len(resource.Initialization) == 0 {
			continue
		}
		canonical := normalizeName(resourceName)
		if canonical == "" || !isKnownResource(canonical) {
			continue
		}

		files := extractInitializationFiles(resource.Initialization)
		entry, exists := results[canonical]
		if !exists {
			entry = types.ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "resource",
				DependencyName: canonical,
				Required:       true,
				Purpose:        "Initialization data references this resource",
				AccessMethod:   "initialization",
				Configuration:  map[string]interface{}{},
				DiscoveredAt:   time.Now(),
				LastVerified:   time.Now(),
			}
		}

		if entry.Configuration == nil {
			entry.Configuration = map[string]interface{}{}
		}
		entry.Configuration["initialization_detected"] = true
		if len(files) > 0 {
			entry.Configuration["initialization_files"] = mergeInitializationFiles(entry.Configuration["initialization_files"], files)
		}
		results[canonical] = entry
	}
}

func extractInitializationFiles(entries []map[string]interface{}) []string {
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if file, ok := entry["file"].(string); ok && file != "" {
			files = append(files, file)
		}
	}
	return files
}

func mergeInitializationFiles(existing interface{}, additions []string) []string {
	if len(additions) == 0 {
		return toStringSlice(existing)
	}
	set := map[string]struct{}{}
	merged := make([]string, 0)
	for _, item := range toStringSlice(existing) {
		if _, ok := set[item]; ok {
			continue
		}
		set[item] = struct{}{}
		merged = append(merged, item)
	}
	for _, add := range additions {
		if add == "" {
			continue
		}
		if _, ok := set[add]; ok {
			continue
		}
		set[add] = struct{}{}
		merged = append(merged, add)
	}
	return merged
}

func toStringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

func orderedMapFromStruct(value interface{}) *orderedmap.OrderedMap {
	ordered := orderedmap.New()
	if value == nil {
		return ordered
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return ordered
	}
	if err := json.Unmarshal(payload, ordered); err != nil {
		return orderedmap.New()
	}
	return ordered
}

func storeDependencies(analysis *types.DependencyAnalysisResponse, extras []types.ScenarioDependency) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing dependencies for this scenario
	_, err = tx.Exec("DELETE FROM scenario_dependencies WHERE scenario_name = $1", analysis.Scenario)
	if err != nil {
		return err
	}

	// Insert all dependencies (declared + detected)
	allDeps := make([]types.ScenarioDependency, 0, len(analysis.Resources)+len(analysis.DetectedResources)+len(analysis.Scenarios)+len(analysis.SharedWorkflows)+len(extras))
	allDeps = append(allDeps, analysis.Resources...)
	allDeps = append(allDeps, analysis.DetectedResources...)
	allDeps = append(allDeps, analysis.Scenarios...)
	allDeps = append(allDeps, analysis.SharedWorkflows...)
	allDeps = append(allDeps, extras...)

	for _, dep := range allDeps {
		configJSON, _ := json.Marshal(dep.Configuration)

		_, err = tx.Exec(`
			INSERT INTO scenario_dependencies 
			(scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			dep.ScenarioName, dep.DependencyType, dep.DependencyName, dep.Required,
			dep.Purpose, dep.AccessMethod, configJSON, dep.DiscoveredAt, dep.LastVerified)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	cleanupInvalidScenarioDependencies()
	return nil
}

func cleanupInvalidScenarioDependencies() {
	if db == nil {
		return
	}
	ensureDependencyCatalogs()
	dependencyCatalogMu.RLock()
	defer dependencyCatalogMu.RUnlock()
	if len(knownScenarioNames) == 0 {
		return
	}
	rows, err := db.Query(`
		SELECT DISTINCT dependency_name
		FROM scenario_dependencies
		WHERE dependency_type = 'scenario'`)
	if err != nil {
		log.Printf("Warning: unable to query scenario dependencies for cleanup: %v", err)
		return
	}
	defer rows.Close()

	toDelete := make([]string, 0)
	for rows.Next() {
		var depName string
		if err := rows.Scan(&depName); err != nil {
			continue
		}
		if _, ok := knownScenarioNames[normalizeName(depName)]; !ok {
			toDelete = append(toDelete, depName)
		}
	}

	for _, depName := range toDelete {
		if _, err := db.Exec(`
			DELETE FROM scenario_dependencies
			WHERE dependency_type = 'scenario' AND dependency_name = $1`, depName); err != nil {
			log.Printf("Warning: failed to delete orphaned dependency %s: %v", depName, err)
		}
	}
}

func analyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error) {
	results := make(map[string]*types.DependencyAnalysisResponse)

	envCfg := appconfig.Load()
	scenariosDir := envCfg.ScenariosDir
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()

		// Check if it has a service.json (valid scenario)
		serviceConfigPath := filepath.Join(scenariosDir, scenarioName, ".vrooli", "service.json")
		if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
			continue
		}

		analysis, err := analyzeScenario(scenarioName)
		if err != nil {
			log.Printf("Warning: failed to analyze scenario %s: %v", scenarioName, err)
			continue
		}

		results[scenarioName] = analysis
	}

	return results, nil
}

func listScenarioNames() ([]string, error) {
	scenariosDir := appconfig.Load().ScenariosDir
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		serviceConfigPath := filepath.Join(scenariosDir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func generateDependencyGraph(graphType string) (*types.DependencyGraph, error) {
	nodes := []types.GraphNode{}
	edges := []types.GraphEdge{}

	// Query all dependencies from database
	rows, err := db.Query(`
		SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
		FROM scenario_dependencies
		ORDER BY scenario_name, dependency_type, dependency_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodeSet := make(map[string]bool)

	for rows.Next() {
		var scenarioName, depType, depName, purpose, accessMethod string
		var required bool
		var configJSON []byte
		var discoveredAt, lastVerified time.Time

		err := rows.Scan(&scenarioName, &depType, &depName, &required, &purpose, &accessMethod, &configJSON, &discoveredAt, &lastVerified)
		if err != nil {
			continue
		}

		var configuration map[string]interface{}
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &configuration); err != nil {
				configuration = map[string]interface{}{}
			}
		}

		// Filter by graph type
		if graphType == "resource" && depType != "resource" {
			continue
		}
		if graphType == "scenario" && depType == "resource" {
			continue
		}
		if depType == "scenario" && !isKnownScenario(depName) {
			continue
		}

		// Add nodes if not already present
		if !nodeSet[scenarioName] {
			nodes = append(nodes, types.GraphNode{
				ID:    scenarioName,
				Label: scenarioName,
				Type:  "scenario",
				Group: "scenarios",
				Metadata: map[string]interface{}{
					"node_type": "scenario",
				},
			})
			nodeSet[scenarioName] = true
		}

		if !nodeSet[depName] {
			nodeGroup := "resources"
			nodeType := "resource"
			if depType == "scenario" {
				nodeGroup = "scenarios"
				nodeType = "scenario"
			} else if depType == "shared_workflow" {
				nodeGroup = "workflows"
				nodeType = "workflow"
			}

			nodes = append(nodes, types.GraphNode{
				ID:    depName,
				Label: depName,
				Type:  nodeType,
				Group: nodeGroup,
				Metadata: map[string]interface{}{
					"node_type": depType,
				},
			})
			nodeSet[depName] = true
		}

		// Add edge
		weight := 1.0
		if required {
			weight = 2.0
		}

		edges = append(edges, types.GraphEdge{
			Source:   scenarioName,
			Target:   depName,
			Label:    depType,
			Type:     depType,
			Required: required,
			Weight:   weight,
			Metadata: map[string]interface{}{
				"purpose":       purpose,
				"access_method": accessMethod,
				"configuration": configuration,
				"discovered_at": discoveredAt,
				"last_verified": lastVerified,
			},
		})
	}

	graph := &types.DependencyGraph{
		ID:    uuid.New().String(),
		Type:  graphType,
		Nodes: nodes,
		Edges: edges,
		Metadata: map[string]interface{}{
			"total_nodes":      len(nodes),
			"total_edges":      len(edges),
			"generated_at":     time.Now(),
			"complexity_score": calculateComplexityScore(nodes, edges),
		},
	}

	return graph, nil
}

func calculateComplexityScore(nodes []types.GraphNode, edges []types.GraphEdge) float64 {
	if len(nodes) == 0 {
		return 0.0
	}

	// Simple complexity score based on edge-to-node ratio
	ratio := float64(len(edges)) / float64(len(nodes))

	// Normalize to 0-1 scale (assuming max ratio of 5 = complex system)
	score := ratio / 5.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func analyzeProposedScenario(req types.ProposedScenarioRequest) (map[string]interface{}, error) {
	predictedResources := []map[string]interface{}{}
	similarPatterns := []map[string]interface{}{}
	recommendations := []map[string]interface{}{}

	// Simple heuristic predictions based on requirements
	for _, reqResource := range req.Requirements {
		predictedResources = append(predictedResources, map[string]interface{}{
			"resource_name": reqResource,
			"confidence":    0.9,
			"reasoning":     "Explicitly mentioned in requirements",
		})
	}

	// Use Claude Code for intelligent analysis of the scenario description
	claudeAnalysis, err := analyzeWithClaudeCode(req.Name, req.Description)
	if err != nil {
		log.Printf("Claude Code analysis failed: %v", err)
	} else {
		// Merge Claude Code predictions with our results
		for _, prediction := range claudeAnalysis.PredictedResources {
			predictedResources = append(predictedResources, prediction)
		}
		for _, recommendation := range claudeAnalysis.Recommendations {
			recommendations = append(recommendations, recommendation)
		}
	}

	// Use Qdrant for semantic similarity matching
	qdrantMatches, err := findSimilarScenariosQdrant(req.Description, req.SimilarScenarios)
	if err != nil {
		log.Printf("Qdrant similarity matching failed: %v", err)
	} else {
		similarPatterns = qdrantMatches
	}

	// Add common patterns based on description keywords (fallback heuristics)
	description := strings.ToLower(req.Description)
	heuristicResources := getHeuristicPredictions(description)
	predictedResources = append(predictedResources, heuristicResources...)

	// Calculate confidence scores
	resourceConfidence := calculateResourceConfidence(predictedResources)
	scenarioConfidence := calculateScenarioConfidence(similarPatterns)

	return map[string]interface{}{
		"predicted_resources": deduplicateResources(predictedResources),
		"similar_patterns":    similarPatterns,
		"recommendations":     recommendations,
		"confidence_scores": map[string]float64{
			"resource": resourceConfidence,
			"scenario": scenarioConfidence,
		},
	}, nil
}

// HTTP Handlers
// HTTP handler functions moved to handlers.go for clarity.

func loadScenarioMetadataMap() (map[string]types.ScenarioSummary, error) {
	results := map[string]types.ScenarioSummary{}
	if db == nil {
		return results, nil
	}
	rows, err := db.Query("SELECT scenario_name, display_name, description, tags, last_scanned FROM scenario_metadata")
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var summary types.ScenarioSummary
		var tags pq.StringArray
		var lastScanned sql.NullTime
		if err := rows.Scan(&summary.Name, &summary.DisplayName, &summary.Description, &tags, &lastScanned); err != nil {
			continue
		}
		summary.Tags = []string(tags)
		if lastScanned.Valid {
			summary.LastScanned = &lastScanned.Time
		}
		results[summary.Name] = summary
	}

	return results, nil
}

func loadStoredDependencies(scenarioName string) (map[string][]types.ScenarioDependency, error) {
	if db == nil {
		return map[string][]types.ScenarioDependency{
			"resources":        []types.ScenarioDependency{},
			"scenarios":        []types.ScenarioDependency{},
			"shared_workflows": []types.ScenarioDependency{},
		}, nil
	}
	rows, err := db.Query(`
		SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
		FROM scenario_dependencies
		WHERE scenario_name = $1
		ORDER BY dependency_type, dependency_name`, scenarioName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]types.ScenarioDependency{
		"resources":        []types.ScenarioDependency{},
		"scenarios":        []types.ScenarioDependency{},
		"shared_workflows": []types.ScenarioDependency{},
	}

	for rows.Next() {
		var dep types.ScenarioDependency
		var configJSON []byte
		if err := rows.Scan(&dep.ScenarioName, &dep.DependencyType, &dep.DependencyName, &dep.Required, &dep.Purpose, &dep.AccessMethod, &configJSON, &dep.DiscoveredAt, &dep.LastVerified); err != nil {
			continue
		}
		if len(configJSON) > 0 {
			_ = json.Unmarshal(configJSON, &dep.Configuration)
		}
		result[dep.DependencyType] = append(result[dep.DependencyType], dep)
	}

	return result, nil
}

func loadOptimizationRecommendations(scenarioName string) ([]types.OptimizationRecommendation, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`
		SELECT id, scenario_name, recommendation_type, title, description, current_state, recommended_state, estimated_impact, confidence_score, priority, status, created_at
		FROM optimization_recommendations
		WHERE scenario_name = $1
		ORDER BY created_at DESC`, scenarioName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recommendations := []types.OptimizationRecommendation{}
	for rows.Next() {
		var rec types.OptimizationRecommendation
		var currentJSON, recommendedJSON, estimatedJSON []byte
		if err := rows.Scan(&rec.ID, &rec.ScenarioName, &rec.RecommendationType, &rec.Title, &rec.Description, &currentJSON, &recommendedJSON, &estimatedJSON, &rec.ConfidenceScore, &rec.Priority, &rec.Status, &rec.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(currentJSON, &rec.CurrentState)
		_ = json.Unmarshal(recommendedJSON, &rec.RecommendedState)
		_ = json.Unmarshal(estimatedJSON, &rec.EstimatedImpact)
		recommendations = append(recommendations, rec)
	}
	return recommendations, nil
}

func filterDetectedDependencies(deps []types.ScenarioDependency) []types.ScenarioDependency {
	filtered := []types.ScenarioDependency{}
	for _, dep := range deps {
		source := ""
		if dep.Configuration != nil {
			if val, ok := dep.Configuration["source"].(string); ok {
				source = val
			}
		}
		if source == "declared" {
			continue
		}
		filtered = append(filtered, dep)
	}
	return filtered
}

func applyDetectedDiffs(scenarioName string, analysis *types.DependencyAnalysisResponse, applyResources, applyScenarios bool) (map[string]interface{}, error) {
	updates := map[string]interface{}{}
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return nil, err
	}
	if os.Getenv("SCENARIO_ANALYZER_TRACE") == "1" {
		log.Printf("[dependency-apply] %s declared resources=%d scenarios=%d", scenarioName, len(cfg.Dependencies.Resources), len(cfg.Dependencies.Scenarios))
	}
	if applyDiffsHook != nil {
		applyDiffsHook(scenarioName, cfg)
	}

	rawConfig, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return nil, err
	}
	rawDependencies := ensureOrderedMap(rawConfig, "dependencies")
	rawResources := ensureOrderedMap(rawDependencies, "resources")
	rawScenarios := ensureOrderedMap(rawDependencies, "scenarios")
	rawDependencies.Set("resources", rawResources)
	rawDependencies.Set("scenarios", rawScenarios)
	if os.Getenv("SCENARIO_ANALYZER_TRACE") == "1" {
		log.Printf("[dependency-apply] raw resources keys=%d", len(rawResources.Keys()))
	}
	if len(rawResources.Keys()) == 0 && len(cfg.Dependencies.Resources) > 0 {
		rawResources = orderedMapFromStruct(cfg.Dependencies.Resources)
		rawDependencies.Set("resources", rawResources)
	}
	if len(rawScenarios.Keys()) == 0 && len(cfg.Dependencies.Scenarios) > 0 {
		rawScenarios = orderedMapFromStruct(cfg.Dependencies.Scenarios)
		rawDependencies.Set("scenarios", rawScenarios)
	}

	resourcesAdded := []string{}
	if applyResources {
		missing := map[string]struct{}{}
		for _, drift := range analysis.ResourceDiff.Missing {
			missing[drift.Name] = struct{}{}
		}
		if len(missing) > 0 {
			if cfg.Dependencies.Resources == nil {
				cfg.Dependencies.Resources = map[string]types.Resource{}
			}
			for _, dep := range analysis.DetectedResources {
				if _, ok := missing[dep.DependencyName]; !ok {
					continue
				}
				if _, exists := cfg.Dependencies.Resources[dep.DependencyName]; exists {
					continue
				}
				typeHint := "custom"
				if dep.Configuration != nil {
					if val, ok := dep.Configuration["resource_type"].(string); ok && val != "" {
						typeHint = val
					}
				}
				description := fmt.Sprintf("Auto-detected via analyzer (%s)", dep.AccessMethod)
				cfg.Dependencies.Resources[dep.DependencyName] = types.Resource{
					Type:     typeHint,
					Enabled:  true,
					Required: true,
					Purpose:  description,
				}
				resourceEntry := orderedmap.New()
				resourceEntry.Set("type", typeHint)
				resourceEntry.Set("enabled", true)
				resourceEntry.Set("required", true)
				resourceEntry.Set("description", description)
				rawResources.Set(dep.DependencyName, resourceEntry)
				resourcesAdded = append(resourcesAdded, dep.DependencyName)
			}
		}
	}

	scenariosAdded := []string{}
	if applyScenarios {
		missing := map[string]struct{}{}
		for _, drift := range analysis.ScenarioDiff.Missing {
			missing[drift.Name] = struct{}{}
		}
		if len(missing) > 0 {
			if cfg.Dependencies.Scenarios == nil {
				cfg.Dependencies.Scenarios = map[string]types.ScenarioDependencySpec{}
			}
			for _, dep := range analysis.Scenarios {
				if _, ok := missing[dep.DependencyName]; !ok {
					continue
				}
				if _, exists := cfg.Dependencies.Scenarios[dep.DependencyName]; exists {
					continue
				}
				description := fmt.Sprintf("Auto-detected dependency via %s", dep.AccessMethod)
				version, versionRange := resolveScenarioVersionSpec(dep.DependencyName)
				cfg.Dependencies.Scenarios[dep.DependencyName] = types.ScenarioDependencySpec{
					Required:     true,
					Version:      version,
					VersionRange: versionRange,
					Description:  description,
				}
				scenarioEntry := orderedmap.New()
				scenarioEntry.Set("required", true)
				scenarioEntry.Set("version", version)
				scenarioEntry.Set("versionRange", versionRange)
				scenarioEntry.Set("description", description)
				rawScenarios.Set(dep.DependencyName, scenarioEntry)
				scenariosAdded = append(scenariosAdded, dep.DependencyName)
			}
		}
	}

	changed := len(resourcesAdded) > 0 || len(scenariosAdded) > 0
	if changed {
		reordered := reorderTopLevelKeys(rawConfig)
		if err := writeRawServiceConfigMap(scenarioPath, reordered); err != nil {
			return nil, err
		}
		if err := updateScenarioMetadata(scenarioName, cfg, scenarioPath); err != nil {
			return nil, err
		}
		refreshDependencyCatalogs()
	}

	updates["changed"] = changed
	updates["resources_added"] = resourcesAdded
	updates["scenarios_added"] = scenariosAdded
	return updates, nil
}

func loadRawServiceConfigMap(scenarioPath string) (*orderedmap.OrderedMap, error) {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, err
	}
	raw := orderedmap.New()
	if err := json.Unmarshal(data, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func writeRawServiceConfigMap(scenarioPath string, cfg *orderedmap.OrderedMap) error {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return err
	}
	payload := bytes.TrimRight(buffer.Bytes(), "\n")
	payload = bytes.ReplaceAll(payload, []byte(`\u003c`), []byte("<"))
	payload = bytes.ReplaceAll(payload, []byte(`\u003e`), []byte(">"))
	payload = bytes.ReplaceAll(payload, []byte(`\u0026`), []byte("&"))
	return os.WriteFile(serviceConfigPath, payload, 0644)
}

func persistOptimizationRecommendations(scenarioName string, recs []types.OptimizationRecommendation) error {
	if db == nil {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM optimization_recommendations WHERE scenario_name = $1", scenarioName); err != nil {
		return err
	}
	insertStmt := `
		INSERT INTO optimization_recommendations
		(id, scenario_name, recommendation_type, title, description, current_state, recommended_state, estimated_impact, confidence_score, priority, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`
	for _, rec := range recs {
		currentJSON, _ := json.Marshal(rec.CurrentState)
		recommendedJSON, _ := json.Marshal(rec.RecommendedState)
		estimatedJSON, _ := json.Marshal(rec.EstimatedImpact)
		if _, err := tx.Exec(insertStmt,
			rec.ID,
			rec.ScenarioName,
			rec.RecommendationType,
			rec.Title,
			rec.Description,
			currentJSON,
			recommendedJSON,
			estimatedJSON,
			rec.ConfidenceScore,
			rec.Priority,
			rec.Status,
			rec.CreatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func buildOptimizationSummary(recs []types.OptimizationRecommendation) types.OptimizationSummary {
	summary := types.OptimizationSummary{
		RecommendationCount: len(recs),
		ByType:              map[string]int{},
		PotentialImpact:     map[string]int{},
	}
	for _, rec := range recs {
		summary.ByType[rec.RecommendationType]++
		if strings.EqualFold(rec.Priority, "high") || strings.EqualFold(rec.Priority, "critical") {
			summary.HighPriority++
		}
		if action, ok := rec.RecommendedState["action"].(string); ok {
			summary.PotentialImpact[action]++
		}
	}
	return summary
}

func applyOptimizationRecommendations(scenarioName string, recs []types.OptimizationRecommendation) (map[string]interface{}, error) {
	if len(recs) == 0 {
		return map[string]interface{}{"changed": false}, nil
	}
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	removedResources := []string{}
	scenariosRemoved := []string{}
	for _, rec := range recs {
		action, _ := rec.RecommendedState["action"].(string)
		switch action {
		case "remove_resource":
			resourceName, _ := rec.RecommendedState["resource_name"].(string)
			if resourceName == "" {
				continue
			}
			removed, err := removeResourceFromServiceConfig(scenarioPath, resourceName)
			if err != nil {
				return nil, err
			}
			if removed {
				removedResources = append(removedResources, resourceName)
			}
		case "remove_scenario_dependency":
			depName, _ := rec.RecommendedState["scenario_name"].(string)
			if depName == "" {
				continue
			}
			removed, err := removeScenarioDependencyFromServiceConfig(scenarioPath, depName)
			if err != nil {
				return nil, err
			}
			if removed {
				scenariosRemoved = append(scenariosRemoved, depName)
			}
		}
	}
	changed := len(removedResources) > 0 || len(scenariosRemoved) > 0
	if changed {
		cfg, err := loadServiceConfigFromFile(scenarioPath)
		if err == nil {
			_ = updateScenarioMetadata(scenarioName, cfg, scenarioPath)
		}
		refreshDependencyCatalogs()
	}
	return map[string]interface{}{
		"changed":           changed,
		"resources_removed": removedResources,
		"scenarios_removed": scenariosRemoved,
	}, nil
}

func ensureOrderedMap(parent *orderedmap.OrderedMap, key string) *orderedmap.OrderedMap {
	if parent == nil {
		return orderedmap.New()
	}
	if val, ok := parent.Get(key); ok {
		switch typed := val.(type) {
		case *orderedmap.OrderedMap:
			return typed
		case orderedmap.OrderedMap:
			converted := orderedmap.New()
			for _, childKey := range typed.Keys() {
				if childVal, ok := typed.Get(childKey); ok {
					converted.Set(childKey, childVal)
				}
			}
			parent.Set(key, converted)
			return converted
		case map[string]interface{}:
			converted := orderedmap.New()
			for k, v := range typed {
				converted.Set(k, v)
			}
			parent.Set(key, converted)
			return converted
		}
	}
	child := orderedmap.New()
	parent.Set(key, child)
	return child
}

func reorderTopLevelKeys(cfg *orderedmap.OrderedMap) *orderedmap.OrderedMap {
	if cfg == nil {
		return orderedmap.New()
	}
	preferred := []string{"$schema", "version", "service", "ports", "lifecycle", "dependencies"}
	reordered := orderedmap.New()
	seen := map[string]struct{}{}
	for _, key := range preferred {
		if val, ok := cfg.Get(key); ok {
			reordered.Set(key, val)
			seen[key] = struct{}{}
		}
	}
	for _, key := range cfg.Keys() {
		if _, ok := seen[key]; ok {
			continue
		}
		if val, ok := cfg.Get(key); ok {
			reordered.Set(key, val)
		}
	}
	return reordered
}

func cloneOrderedMap(src *orderedmap.OrderedMap) *orderedmap.OrderedMap {
	if src == nil {
		return orderedmap.New()
	}
	clone := orderedmap.New()
	for _, key := range src.Keys() {
		if val, ok := src.Get(key); ok {
			clone.Set(key, val)
		}
	}
	return clone
}

func resolveScenarioVersionSpec(dependencyName string) (string, string) {
	scenarioPath := filepath.Join(appconfig.Load().ScenariosDir, dependencyName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return "", ">=0.0.0"
	}
	version := strings.TrimSpace(cfg.Service.Version)
	if version == "" {
		return "", ">=0.0.0"
	}
	return version, fmt.Sprintf(">=%s", version)
}

func removeResourceFromServiceConfig(scenarioPath, resourceName string) (bool, error) {
	raw, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return false, err
	}
	deps := ensureOrderedMap(raw, "dependencies")
	resources := ensureOrderedMap(deps, "resources")
	if _, ok := resources.Get(resourceName); !ok {
		return false, nil
	}
	resources.Delete(resourceName)
	reordered := reorderTopLevelKeys(raw)
	if err := writeRawServiceConfigMap(scenarioPath, reordered); err != nil {
		return false, err
	}
	return true, nil
}

func removeScenarioDependencyFromServiceConfig(scenarioPath, scenarioName string) (bool, error) {
	raw, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return false, err
	}
	deps := ensureOrderedMap(raw, "dependencies")
	scenarios := ensureOrderedMap(deps, "scenarios")
	if _, ok := scenarios.Get(scenarioName); !ok {
		return false, nil
	}
	scenarios.Delete(scenarioName)
	reordered := reorderTopLevelKeys(raw)
	if err := writeRawServiceConfigMap(scenarioPath, reordered); err != nil {
		return false, err
	}
	return true, nil
}

// Integrate with Claude Code resource for intelligent analysis
func analyzeWithClaudeCode(scenarioName, description string) (*ClaudeCodeAnalysis, error) {
	// Create a temporary analysis file
	analysisPrompt := fmt.Sprintf(`
Analyze the following proposed Vrooli scenario and predict its likely dependencies:

Scenario Name: %s
Description: %s

Please identify:
1. Likely resource dependencies (postgres, redis, ollama, n8n, etc.)
2. Similar existing scenarios it might depend on
3. Recommended architecture patterns
4. Potential optimization opportunities

Format your response as structured analysis focusing on technical implementation needs.
`, scenarioName, description)

	// Write prompt to temporary file
	tempFile := "/tmp/claude-analysis-" + uuid.New().String() + ".txt"
	if err := os.WriteFile(tempFile, []byte(analysisPrompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to create analysis prompt file: %w", err)
	}
	defer os.Remove(tempFile)

	// Execute Claude Code analysis
	cmd := exec.Command("resource-claude-code", "analyze", tempFile, "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude-code command failed: %w", err)
	}

	// Parse Claude Code response and extract dependency predictions
	analysis := parseClaudeCodeResponse(string(output), description)
	return analysis, nil
}

// Integrate with Qdrant for semantic similarity matching
func findSimilarScenariosQdrant(description string, existingScenarios []string) ([]map[string]interface{}, error) {
	var matches []map[string]interface{}

	// Create embedding for the proposed scenario description
	embeddingCmd := exec.Command("resource-qdrant", "embed", description)
	embeddingOutput, err := embeddingCmd.Output()
	if err != nil {
		return matches, fmt.Errorf("failed to create embedding: %w", err)
	}

	// Search for similar scenarios in the vector database
	searchCmd := exec.Command("resource-qdrant", "search",
		"--collection", "scenario_embeddings",
		"--vector", string(embeddingOutput),
		"--limit", "5",
		"--output", "json")

	searchOutput, err := searchCmd.Output()
	if err != nil {
		// Qdrant search failed - return empty matches rather than error
		// This allows the analysis to continue with other methods
		return matches, nil
	}

	// Parse Qdrant search results
	var searchResults QdrantSearchResults
	if err := json.Unmarshal(searchOutput, &searchResults); err != nil {
		return matches, fmt.Errorf("failed to parse qdrant results: %w", err)
	}

	// Convert to our format
	for _, result := range searchResults.Matches {
		if result.Score > 0.7 { // Only include high-confidence matches
			matches = append(matches, map[string]interface{}{
				"scenario_name": result.ScenarioName,
				"similarity":    result.Score,
				"resources":     result.Resources,
				"description":   result.Description,
			})
		}
	}

	return matches, nil
}

// Helper functions for analysis
func getHeuristicPredictions(description string) []map[string]interface{} {
	var predictions []map[string]interface{}

	heuristics := map[string][]string{
		"postgres": {"data", "database", "store", "persist", "sql", "table"},
		"redis":    {"cache", "session", "temporary", "fast", "memory"},
		"ollama":   {"ai", "llm", "language model", "chat", "generate", "intelligent"},
		"n8n":      {"workflow", "automation", "process", "trigger", "orchestrate"},
		"qdrant":   {"vector", "semantic", "search", "similarity", "embedding"},
		"minio":    {"file", "upload", "storage", "document", "asset", "image"},
	}

	for resource, keywords := range heuristics {
		confidence := 0.0
		matches := 0

		for _, keyword := range keywords {
			if strings.Contains(description, keyword) {
				matches++
				confidence += 0.1
			}
		}

		if confidence > 0 {
			// Normalize confidence based on number of matches
			confidence = math.Min(confidence, 0.8)

			predictions = append(predictions, map[string]interface{}{
				"resource_name": resource,
				"confidence":    confidence,
				"reasoning":     fmt.Sprintf("Heuristic match: %d keywords detected", matches),
			})
		}
	}

	return predictions
}

func deduplicateResources(resources []map[string]interface{}) []map[string]interface{} {
	seen := make(map[string]float64)
	var deduplicated []map[string]interface{}

	for _, resource := range resources {
		name := resource["resource_name"].(string)
		confidence := resource["confidence"].(float64)

		if existingConfidence, exists := seen[name]; !exists || confidence > existingConfidence {
			seen[name] = confidence
		}
	}

	// Rebuild array with highest confidence for each resource
	for _, resource := range resources {
		name := resource["resource_name"].(string)
		confidence := resource["confidence"].(float64)

		if seen[name] == confidence {
			deduplicated = append(deduplicated, resource)
			delete(seen, name) // Prevent duplicates
		}
	}

	return deduplicated
}

func calculateResourceConfidence(predictions []map[string]interface{}) float64 {
	if len(predictions) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, pred := range predictions {
		totalConfidence += pred["confidence"].(float64)
	}

	return math.Min(totalConfidence/float64(len(predictions)), 1.0)
}

func calculateScenarioConfidence(patterns []map[string]interface{}) float64 {
	if len(patterns) == 0 {
		return 0.0
	}

	totalSimilarity := 0.0
	for _, pattern := range patterns {
		if sim, ok := pattern["similarity"].(float64); ok {
			totalSimilarity += sim
		}
	}

	return math.Min(totalSimilarity/float64(len(patterns)), 1.0)
}

// Data structures for external integrations
type ClaudeCodeAnalysis struct {
	PredictedResources []map[string]interface{} `json:"predicted_resources"`
	Recommendations    []map[string]interface{} `json:"recommendations"`
	ArchitectureNotes  string                   `json:"architecture_notes"`
}

type QdrantSearchResults struct {
	Matches []QdrantMatch `json:"matches"`
}

type QdrantMatch struct {
	ScenarioName string                 `json:"scenario_name"`
	Score        float64                `json:"score"`
	Resources    []string               `json:"resources"`
	Description  string                 `json:"description"`
	Metadata     map[string]interface{} `json:"metadata"`
}

func parseClaudeCodeResponse(response, originalDescription string) *ClaudeCodeAnalysis {
	// Parse Claude Code response and extract structured dependency predictions
	// This is a simplified implementation - in practice, you'd parse the AI response more sophisticatedly

	analysis := &ClaudeCodeAnalysis{
		PredictedResources: []map[string]interface{}{},
		Recommendations:    []map[string]interface{}{},
		ArchitectureNotes:  response,
	}

	// Extract resource mentions from Claude's response
	responseText := strings.ToLower(response)

	resourcePatterns := map[string]float64{
		"postgres":   0.8,
		"postgresql": 0.8,
		"database":   0.7,
		"redis":      0.8,
		"cache":      0.6,
		"ollama":     0.9,
		"llm":        0.7,
		"n8n":        0.8,
		"workflow":   0.6,
		"qdrant":     0.9,
		"vector":     0.7,
		"minio":      0.8,
		"storage":    0.5,
	}

	for pattern, confidence := range resourcePatterns {
		if strings.Contains(responseText, pattern) {
			// Map patterns to actual resource names
			resourceName := mapPatternToResource(pattern)
			if resourceName != "" {
				analysis.PredictedResources = append(analysis.PredictedResources, map[string]interface{}{
					"resource_name": resourceName,
					"confidence":    confidence,
					"reasoning":     fmt.Sprintf("Claude Code analysis mentioned '%s'", pattern),
				})
			}
		}
	}

	return analysis
}

func mapPatternToResource(pattern string) string {
	resourceMap := map[string]string{
		"postgres":   "postgres",
		"postgresql": "postgres",
		"database":   "postgres",
		"redis":      "redis",
		"cache":      "redis",
		"ollama":     "ollama",
		"llm":        "ollama",
		"n8n":        "n8n",
		"workflow":   "n8n",
		"qdrant":     "qdrant",
		"vector":     "qdrant",
		"minio":      "minio",
		"storage":    "minio",
	}

	return resourceMap[pattern]
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Run boots the HTTP API using the provided configuration and database connection.
// NOTE: Run is defined in server.go to keep this file focused on analysis logic.
