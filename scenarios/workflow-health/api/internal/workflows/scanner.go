package workflows

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const registryFileName = "registry.json"

// ScanScenario scans a scenario-owned bas/ tree into a deterministic catalog.
func ScanScenario(scenarioDir string) (*ScenarioWorkflowCatalog, error) {
	scanner := scanner{scenarioDir: filepath.Clean(scenarioDir), scenario: filepath.Base(filepath.Clean(scenarioDir))}
	return scanner.scan()
}

type scanner struct {
	scenarioDir string
	scenario    string
}

func (s scanner) scan() (*ScenarioWorkflowCatalog, error) {
	registry, err := s.scanRegistry()
	if err != nil {
		return nil, err
	}

	requirements, err := s.collectRequirementValidations()
	if err != nil {
		return nil, err
	}

	catalog := &ScenarioWorkflowCatalog{
		Scenario:    s.scenario,
		ScenarioDir: s.scenarioDir,
		Registry:    registry,
	}

	registryByPath := make(map[string]RegistryEntry, len(registry.Entries))
	for _, entry := range registry.Entries {
		registryByPath[entry.File] = entry
	}

	for _, family := range []struct {
		dir  string
		typ  AssetType
		role AssetRole
	}{
		{dir: "cases", typ: AssetTypeCase, role: AssetRoleValidationCase},
		{dir: "flows", typ: AssetTypeFlow, role: AssetRoleAgentFlow},
		{dir: "actions", typ: AssetTypeAction, role: AssetRoleFragment},
	} {
		files, err := collectJSONFiles(filepath.Join(s.scenarioDir, "bas", family.dir))
		if err != nil {
			return nil, err
		}
		for _, absPath := range files {
			asset := s.scanWorkflowAsset(absPath, family.typ, family.role, requirements, registryByPath)
			catalog.Assets = append(catalog.Assets, asset)
			switch family.typ {
			case AssetTypeCase:
				catalog.Cases = append(catalog.Cases, WorkflowCase{WorkflowAsset: asset})
			case AssetTypeFlow:
				catalog.Flows = append(catalog.Flows, WorkflowFlow{WorkflowAsset: asset})
			case AssetTypeAction:
				catalog.Actions = append(catalog.Actions, WorkflowAction{WorkflowAsset: asset})
			}
			catalog.DependencyEdges = append(catalog.DependencyEdges, asset.Dependencies...)
			delete(registryByPath, asset.Path)
		}
	}

	seeds, err := s.scanSeeds()
	if err != nil {
		return nil, err
	}
	catalog.Seeds = seeds
	for _, seed := range seeds {
		catalog.Assets = append(catalog.Assets, WorkflowAsset{
			ID:       seed.ID,
			Scenario: seed.Scenario,
			Path:     seed.Path,
			Type:     AssetTypeSeed,
			Role:     AssetRoleSeed,
			Name:     seed.Name,
		})
	}

	for path := range registryByPath {
		catalog.RegistryOnlyPaths = append(catalog.RegistryOnlyPaths, path)
		catalog.Assets = append(catalog.Assets, WorkflowAsset{
			ID:       assetID(s.scenario, path),
			Scenario: s.scenario,
			Path:     path,
			Type:     AssetTypeRegistryOnly,
			Role:     AssetRoleRegistryOnly,
		})
	}

	sort.Strings(catalog.RegistryOnlyPaths)
	sortAssets(catalog.Assets)
	sortWorkflowCases(catalog.Cases)
	sortWorkflowFlows(catalog.Flows)
	sortWorkflowActions(catalog.Actions)
	sortSeeds(catalog.Seeds)
	sortDependencyEdges(catalog.DependencyEdges)

	return catalog, nil
}

func (s scanner) scanWorkflowAsset(absPath string, typ AssetType, role AssetRole, requirementsByPath map[string][]string, registryByPath map[string]RegistryEntry) WorkflowAsset {
	relPath := s.rel(absPath)
	asset := WorkflowAsset{
		ID:       assetID(s.scenario, relPath),
		Scenario: s.scenario,
		Path:     relPath,
		Type:     typ,
		Role:     role,
		Labels:   map[string]string{},
	}

	doc, err := readJSONObject(absPath)
	if err != nil {
		asset.ParseError = err.Error()
	} else {
		asset.Name = firstNonEmpty(getString(doc, "metadata", "name"), strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)))
		asset.Description = getString(doc, "metadata", "description")
		asset.Version = getString(doc, "metadata", "version")
		asset.ExecutionMode = normalizeMode(getString(doc, "metadata", "execution_mode"))
		asset.Labels = labelsMap(doc)
		asset.Reset = normalizeReset(firstNonEmpty(asset.Labels["reset"], getString(doc, "metadata", "reset")))
		asset.NodeCount = nodeCount(doc)
		asset.Requirements = s.requirementLinks(doc, relPath, requirementsByPath)
		asset.Selectors = extractSelectorRefs(doc)
		asset.Routes = extractRouteRefs(doc)
		asset.Dependencies = s.extractDependencyEdges(asset.ID, relPath, doc)
	}

	if entry, ok := registryByPath[relPath]; ok {
		asset.Order = entry.Order
		if asset.Description == "" {
			asset.Description = entry.Description
		}
		if len(asset.Requirements) == 0 {
			asset.Requirements = requirementLinksFromIDs(entry.Requirements, "registry")
		}
		if asset.Reset == "" {
			asset.Reset = normalizeReset(entry.Reset)
		}
		if len(asset.Dependencies) == 0 {
			asset.Dependencies = s.fixtureEdges(asset.ID, relPath, entry.Fixtures, "registry.fixtures")
		}
	}

	asset.Safety = safetyProfile(asset.ExecutionMode, asset.Reset, asset.Labels)
	if asset.Labels != nil && len(asset.Labels) == 0 {
		asset.Labels = nil
	}
	sortRequirementLinks(asset.Requirements)
	sortSelectorRefs(asset.Selectors)
	sortRouteRefs(asset.Routes)
	sortDependencyEdges(asset.Dependencies)

	return asset
}

func (s scanner) scanRegistry() (RegistrySnapshot, error) {
	path := filepath.Join(s.scenarioDir, "bas", registryFileName)
	snapshot := RegistrySnapshot{Path: filepath.ToSlash(filepath.Join("bas", registryFileName))}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot, nil
		}
		return snapshot, fmt.Errorf("read bas registry: %w", err)
	}
	snapshot.Exists = true

	var raw struct {
		Scenario    string          `json:"scenario"`
		GeneratedAt json.RawMessage `json:"generated_at"`
		Playbooks   []RegistryEntry `json:"playbooks"`
		Metadata    struct {
			ExecutionMode string `json:"execution_mode"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return snapshot, fmt.Errorf("parse bas registry: %w", err)
	}

	snapshot.Scenario = raw.Scenario
	snapshot.GeneratedAt = rawString(raw.GeneratedAt)
	snapshot.ExecutionMode = normalizeMode(raw.Metadata.ExecutionMode)
	snapshot.Entries = append(snapshot.Entries, raw.Playbooks...)
	sort.Slice(snapshot.Entries, func(i, j int) bool {
		return snapshot.Entries[i].File < snapshot.Entries[j].File
	})
	return snapshot, nil
}

func (s scanner) scanSeeds() ([]SeedContract, error) {
	root := filepath.Join(s.scenarioDir, "bas", "seeds")
	var seeds []SeedContract
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), "__") {
				return filepath.SkipDir
			}
			return nil
		}
		relPath := s.rel(path)
		seeds = append(seeds, SeedContract{
			ID:       assetID(s.scenario, relPath),
			Scenario: s.scenario,
			Path:     relPath,
			Name:     strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("scan bas/seeds: %w", err)
	}
	sortSeeds(seeds)
	return seeds, nil
}

func (s scanner) collectRequirementValidations() (map[string][]string, error) {
	result := make(map[string][]string)
	indexPath := filepath.Join(s.scenarioDir, "requirements", "index.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("read requirements index: %w", err)
	}

	var index struct {
		Imports []string `json:"imports"`
	}
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, fmt.Errorf("parse requirements index: %w", err)
	}

	for _, relModule := range index.Imports {
		modulePath := filepath.Join(s.scenarioDir, "requirements", relModule)
		moduleData, err := os.ReadFile(modulePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read requirements module %s: %w", relModule, err)
		}
		var module struct {
			Requirements []struct {
				ID         string `json:"id"`
				Validation []struct {
					Ref string `json:"ref"`
				} `json:"validation"`
			} `json:"requirements"`
		}
		if err := json.Unmarshal(moduleData, &module); err != nil {
			return nil, fmt.Errorf("parse requirements module %s: %w", relModule, err)
		}
		for _, req := range module.Requirements {
			for _, validation := range req.Validation {
				ref := strings.TrimSpace(validation.Ref)
				if strings.HasPrefix(ref, "bas/") && strings.TrimSpace(req.ID) != "" {
					result[ref] = append(result[ref], strings.TrimSpace(req.ID))
				}
			}
		}
	}

	for ref := range result {
		result[ref] = normalizeStrings(result[ref])
	}
	return result, nil
}

func (s scanner) requirementLinks(doc map[string]any, relPath string, requirementsByPath map[string][]string) []RequirementLink {
	if ids := requirementsByPath[relPath]; len(ids) > 0 {
		return requirementLinksFromIDs(ids, "requirements.validation.ref")
	}

	var ids []string
	if requirement := strings.TrimSpace(getString(doc, "metadata", "requirement")); requirement != "" {
		ids = append(ids, requirement)
	}
	if raw := strings.TrimSpace(getString(doc, "metadata", "labels", "requirements_json")); raw != "" {
		var values []string
		if err := json.Unmarshal([]byte(raw), &values); err == nil {
			ids = append(ids, values...)
		}
	}
	return requirementLinksFromIDs(ids, "workflow.metadata")
}

func (s scanner) extractDependencyEdges(assetIDValue, relPath string, doc map[string]any) []DependencyEdge {
	var edges []DependencyEdge
	nodes, _ := doc["nodes"].([]any)
	for _, nodeAny := range nodes {
		node, ok := nodeAny.(map[string]any)
		if !ok {
			continue
		}
		nodeID := getStringFromMap(node, "id")
		for _, ref := range []struct {
			path   string
			source string
		}{
			{path: getString(node, "action", "subflow", "workflow_path"), source: nodeID + ".action.subflow.workflow_path"},
			{path: getString(node, "action", "subflow", "workflowPath"), source: nodeID + ".action.subflow.workflowPath"},
			{path: getString(node, "data", "workflowPath"), source: nodeID + ".data.workflowPath"},
		} {
			if strings.TrimSpace(ref.path) == "" {
				continue
			}
			edges = append(edges, s.edgeForPath(assetIDValue, relPath, ref.path, "subflow", ref.source))
		}
		if fixtureID := strings.TrimSpace(getString(node, "data", "workflowId")); strings.HasPrefix(fixtureID, "@fixture/") {
			slug := strings.TrimPrefix(fixtureID, "@fixture/")
			if idx := strings.Index(slug, "("); idx >= 0 {
				slug = slug[:idx]
			}
			slug = strings.TrimSpace(slug)
			if slug != "" {
				edges = append(edges, s.edgeForPath(assetIDValue, relPath, "actions/"+slug+".json", "fixture", nodeID+".data.workflowId"))
			}
		}
	}
	return dedupeEdges(edges)
}

func (s scanner) fixtureEdges(assetIDValue, relPath string, fixtures []string, source string) []DependencyEdge {
	edges := make([]DependencyEdge, 0, len(fixtures))
	for _, fixture := range fixtures {
		fixture = strings.TrimSpace(fixture)
		if fixture == "" {
			continue
		}
		edges = append(edges, s.edgeForPath(assetIDValue, relPath, "actions/"+fixture+".json", "fixture", source))
	}
	return dedupeEdges(edges)
}

func (s scanner) edgeForPath(assetIDValue, fromPath, targetPath, kind, source string) DependencyEdge {
	targetPath = normalizeWorkflowTargetPath(targetPath)
	edge := DependencyEdge{
		FromAssetID: assetIDValue,
		FromPath:    fromPath,
		ToPath:      targetPath,
		Kind:        kind,
		Source:      strings.Trim(source, "."),
	}
	if targetPath != "" {
		edge.ToAssetID = assetID(s.scenario, "bas/"+targetPath)
	}
	return edge
}

func (s scanner) rel(absPath string) string {
	rel, err := filepath.Rel(s.scenarioDir, absPath)
	if err != nil {
		return filepath.ToSlash(absPath)
	}
	return filepath.ToSlash(rel)
}

func collectJSONFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), "__") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) == ".json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func assetID(scenario, path string) string {
	return scenario + ":" + filepath.ToSlash(path)
}
