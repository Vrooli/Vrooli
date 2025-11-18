package detection

import (
    "io/fs"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
    "sync"
    "time"

    "github.com/google/uuid"

    appconfig "scenario-dependency-analyzer/internal/config"
    types "scenario-dependency-analyzer/internal/types"
)

// Detector encapsulates resource/scenario scanning and catalog discovery logic.
type Detector struct {
    cfg              appconfig.Config
    mu               sync.RWMutex
    catalogsLoaded   bool
    knownScenarios   map[string]struct{}
    knownResources   map[string]struct{}
}

// New creates a detector bound to the provided configuration.
func New(cfg appconfig.Config) *Detector {
    return &Detector{cfg: cfg}
}

// RefreshCatalogs invalidates cached scenario/resource catalogs.
func (d *Detector) RefreshCatalogs() {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.catalogsLoaded = false
    d.knownScenarios = nil
    d.knownResources = nil
}

// KnownScenario reports whether the provided name exists in the catalog.
func (d *Detector) KnownScenario(name string) bool {
    d.ensureCatalogs()
    d.mu.RLock()
    defer d.mu.RUnlock()
    if len(d.knownScenarios) == 0 {
        return true
    }
    _, ok := d.knownScenarios[normalizeName(name)]
    return ok
}

// KnownResource reports whether the provided resource is part of the catalog.
func (d *Detector) KnownResource(name string) bool {
    d.ensureCatalogs()
    d.mu.RLock()
    defer d.mu.RUnlock()
    if len(d.knownResources) == 0 {
        return true
    }
    _, ok := d.knownResources[normalizeName(name)]
    return ok
}

// ScenarioCatalog returns a copy of the cached scenario name set.
func (d *Detector) ScenarioCatalog() map[string]struct{} {
    d.ensureCatalogs()
    d.mu.RLock()
    defer d.mu.RUnlock()
    snapshot := make(map[string]struct{}, len(d.knownScenarios))
    for k := range d.knownScenarios {
        snapshot[k] = struct{}{}
    }
    return snapshot
}

// ScanResources performs resource detection for the scenario.
func (d *Detector) ScanResources(scenarioPath, scenarioName string, cfg *types.ServiceConfig) ([]types.ScenarioDependency, error) {
    results := map[string]types.ScenarioDependency{}
    d.ensureCatalogs()

    err := filepath.WalkDir(scenarioPath, func(path string, entry fs.DirEntry, walkErr error) error {
        if walkErr != nil {
            return nil
        }
        if entry.IsDir() {
            if path != scenarioPath && shouldSkipDirectoryEntry(entry) {
                return filepath.SkipDir
            }
            return nil
        }
        ext := strings.ToLower(filepath.Ext(path))
        if !contains(resourceDetectionExtensions, ext) {
            return nil
        }

        content, err := os.ReadFile(path)
        if err != nil {
            return nil
        }
        rel, relErr := filepath.Rel(scenarioPath, path)
        if relErr != nil {
            rel = path
        }
        if shouldIgnoreDetectionFile(rel) {
            return nil
        }

        contentStr := string(content)
        matches := resourceCommandPattern.FindAllStringSubmatch(contentStr, -1)
        for _, match := range matches {
            if len(match) > 1 {
                if !isAllowedResourceCLIPath(rel) {
                    continue
                }
                resourceName := normalizeName(match[1])
                d.recordDetection(results, scenarioName, resourceName, "resource_cli", "resource-cli", rel, resourceName)
            }
        }

        for _, heuristic := range resourceHeuristicCatalog {
            for _, pattern := range heuristic.Patterns {
                if pattern.MatchString(contentStr) {
                    d.recordDetection(results, scenarioName, heuristic.Name, "heuristic", pattern.String(), rel, heuristic.Type)
                    break
                }
            }
        }

        return nil
    })

    if cfg != nil {
        augmentResourceDetectionsWithInitialization(results, scenarioName, cfg)
    }

    deps := make([]types.ScenarioDependency, 0, len(results))
    for _, dep := range results {
        deps = append(deps, dep)
    }
    sort.Slice(deps, func(i, j int) bool { return deps[i].DependencyName < deps[j].DependencyName })
    return deps, err
}

// ScanScenarioDependencies returns detected scenario-to-scenario edges.
func (d *Detector) ScanScenarioDependencies(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
    var dependencies []types.ScenarioDependency
    d.ensureCatalogs()
    normalizedScenario := normalizeName(scenarioName)
    aliasCatalog := d.buildScenarioAliasCatalog(scenarioPath)

    err := filepath.WalkDir(scenarioPath, func(path string, entry fs.DirEntry, walkErr error) error {
        if walkErr != nil {
            return nil
        }
        if entry.IsDir() && path != scenarioPath && shouldSkipDirectoryEntry(entry) {
            return filepath.SkipDir
        }

        ext := strings.ToLower(filepath.Ext(path))
        if !contains(scenarioDetectionExtensions, ext) {
            return nil
        }

        content, err := os.ReadFile(path)
        if err != nil {
            return nil
        }
        rel, relErr := filepath.Rel(scenarioPath, path)
        if relErr != nil {
            rel = path
        }
        if shouldIgnoreDetectionFile(rel) {
            return nil
        }

        for _, match := range vrooliScenarioPattern.FindAllStringSubmatch(string(content), -1) {
            if len(match) > 1 {
                dep := normalizeName(match[1])
                if dep == normalizedScenario || !d.KnownScenario(dep) {
                    continue
                }
                dependencies = append(dependencies, newScenarioDependency(scenarioName, dep, "vrooli scenario", "vrooli_cli", rel))
            }
        }

        for _, match := range cliScenarioPattern.FindAllStringSubmatch(string(content), -1) {
            ref := ""
            if match[1] != "" {
                ref = match[1]
            } else if match[2] != "" {
                ref = match[2]
            }
            ref = normalizeName(ref)
            if ref == "" || ref == normalizedScenario || !d.KnownScenario(ref) {
                continue
            }
            dependencies = append(dependencies, newScenarioDependency(scenarioName, ref, "CLI reference in "+filepath.Base(path), "direct_cli", rel))
        }

        for _, match := range scenarioPortCallPattern.FindAllStringSubmatch(string(content), -1) {
            depName := ""
            if len(match) > 1 && match[1] != "" {
                depName = normalizeName(match[1])
            } else if len(match) > 2 && match[2] != "" {
                if alias, ok := aliasCatalog[match[2]]; ok {
                    depName = alias
                } else {
                    depName = normalizeName(match[2])
                }
            }
            if depName == "" || depName == normalizedScenario || !d.KnownScenario(depName) {
                continue
            }
            dependencies = append(dependencies, newScenarioDependency(scenarioName, depName, "References "+depName+" port via CLI", "scenario_port_cli", rel))
        }

        return nil
    })

    return dependencies, err
}

// ScanSharedWorkflows detects shared workflow references.
func (d *Detector) ScanSharedWorkflows(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
    var dependencies []types.ScenarioDependency
    initPath := filepath.Join(scenarioPath, "initialization")
    if _, err := os.Stat(initPath); os.IsNotExist(err) {
        return dependencies, nil
    }

    err := filepath.WalkDir(initPath, func(path string, entry fs.DirEntry, walkErr error) error {
        if walkErr != nil {
            return nil
        }
        if strings.HasSuffix(path, ".json") {
            content, err := os.ReadFile(path)
            if err != nil {
                return nil
            }
            matches := sharedWorkflowPattern.FindAllStringSubmatch(string(content), -1)
            for _, match := range matches {
                if len(match) > 1 {
                    dependencies = append(dependencies, types.ScenarioDependency{
                        ID:             uuid.New().String(),
                        ScenarioName:   scenarioName,
                        DependencyType: "shared_workflow",
                        DependencyName: match[1],
                        Required:       true,
                        Purpose:        "Shared workflow dependency",
                        AccessMethod:   "workflow_trigger",
                        Configuration: map[string]interface{}{
                            "found_in_file": strings.TrimPrefix(path, scenarioPath),
                        },
                        DiscoveredAt: time.Now(),
                        LastVerified: time.Now(),
                    })
                }
            }
        }
        return nil
    })

    return dependencies, err
}

func (d *Detector) recordDetection(results map[string]types.ScenarioDependency, scenarioName, name, method, pattern, file, resourceType string) {
    canonical := normalizeName(name)
    if canonical == "" || !d.KnownResource(canonical) {
        return
    }
    entry, ok := results[canonical]
    if !ok {
        entry = types.ScenarioDependency{
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
    if entry.Configuration == nil {
        entry.Configuration = map[string]interface{}{}
    }
    entry.Configuration["resource_type"] = resourceType
    match := map[string]interface{}{"pattern": pattern, "method": method, "file": file}
    if existing, ok := entry.Configuration["matches"].([]map[string]interface{}); ok {
        entry.Configuration["matches"] = append(existing, match)
    } else {
        entry.Configuration["matches"] = []map[string]interface{}{match}
    }
    results[canonical] = entry
}

func (d *Detector) buildScenarioAliasCatalog(scenarioPath string) map[string]string {
    aliases := map[string]string{}
    addAlias := func(identifier, scenario string) {
        if identifier == "" {
            return
        }
        normalized := normalizeName(scenario)
        if !d.KnownScenario(normalized) {
            return
        }
        aliases[identifier] = normalized
    }

    filepath.WalkDir(scenarioPath, func(path string, entry fs.DirEntry, walkErr error) error {
        if walkErr != nil {
            return nil
        }
        if entry.IsDir() && path != scenarioPath && shouldSkipDirectoryEntry(entry) {
            return filepath.SkipDir
        }
        ext := strings.ToLower(filepath.Ext(path))
        if !contains(scenarioDetectionExtensions, ext) {
            return nil
        }
        content, err := os.ReadFile(path)
        if err != nil {
            return nil
        }
        rel, relErr := filepath.Rel(scenarioPath, path)
        if relErr != nil {
            rel = path
        }
        if shouldIgnoreDetectionFile(rel) {
            return nil
        }
        for _, match := range scenarioAliasDeclPattern.FindAllStringSubmatch(string(content), -1) {
            if len(match) >= 3 {
                addAlias(match[1], match[2])
            }
        }
        for _, match := range scenarioAliasShortPattern.FindAllStringSubmatch(string(content), -1) {
            if len(match) >= 3 {
                addAlias(match[1], match[2])
            }
        }
        for _, match := range scenarioAliasBlockPattern.FindAllStringSubmatch(string(content), -1) {
            if len(match) >= 3 {
                addAlias(match[1], match[2])
            }
        }
        return nil
    })

    return aliases
}

func (d *Detector) ensureCatalogs() {
    d.mu.RLock()
    if d.catalogsLoaded {
        d.mu.RUnlock()
        return
    }
    d.mu.RUnlock()

    d.mu.Lock()
    defer d.mu.Unlock()
    if d.catalogsLoaded {
        return
    }

    scenariosDir := determineScenariosDir(d.cfg.ScenariosDir)
    d.knownScenarios = discoverAvailableScenarios(scenariosDir)
    resourcesDir := filepath.Join(filepath.Dir(scenariosDir), "resources")
    d.knownResources = discoverAvailableResources(resourcesDir)
    d.catalogsLoaded = true
}

// Helpers and shared data ----------------------------------------------------

var (
    resourceCommandPattern   = regexp.MustCompile(`resource-([a-z0-9-]+)`)
    resourceHeuristicCatalog = []resourceHeuristic{
        {Name: "postgres", Type: "postgres", Patterns: []*regexp.Regexp{
            regexp.MustCompile(`postgres(ql)?:\/\/`),
            regexp.MustCompile(`PGHOST`),
        }},
        {Name: "redis", Type: "redis", Patterns: []*regexp.Regexp{
            regexp.MustCompile(`redis:\/\/`),
            regexp.MustCompile(`REDIS_URL`),
        }},
        {Name: "ollama", Type: "ollama", Patterns: []*regexp.Regexp{
            regexp.MustCompile(`ollama`),
        }},
        {Name: "qdrant", Type: "qdrant", Patterns: []*regexp.Regexp{
            regexp.MustCompile(`qdrant`),
        }},
        {Name: "n8n", Type: "n8n", Patterns: []*regexp.Regexp{
            regexp.MustCompile(`resource-?n8n`),
            regexp.MustCompile(`N8N_[A-Z0-9_]+`),
            regexp.MustCompile(`RESOURCE_PORT_N8N`),
            regexp.MustCompile(`n8n/(?:api|webhook)`),
        }},
        {Name: "minio", Type: "minio", Patterns: []*regexp.Regexp{
            regexp.MustCompile(`minio`),
        }},
    }

    scenarioAliasDeclPattern  = regexp.MustCompile(`(?m)(?:const|var)\s+([A-Za-z0-9_]+)\s*=\s*"([a-z0-9-]+)"`)
    scenarioAliasShortPattern = regexp.MustCompile(`(?m)([A-Za-z0-9_]+)\s*:=\s*"([a-z0-9-]+)"`)
    scenarioAliasBlockPattern = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\s*=\s*"([a-z0-9-]+)"`)
    scenarioPortCallPattern   = regexp.MustCompile(`resolveScenarioPortViaCLI\s*\(\s*[^,]+,\s*(?:"([a-z0-9-]+)"|([A-Za-z0-9_]+))\s*,`)

    vrooliScenarioPattern = regexp.MustCompile(`vrooli\s+scenario\s+(?:run|test|status)\s+([a-z0-9-]+)`)
    cliScenarioPattern    = regexp.MustCompile(`([a-z0-9-]+)-cli\.sh|\b([a-z0-9-]+)\s+(?:analyze|process|generate|run)`)

    sharedWorkflowPattern = regexp.MustCompile(`initialization/(?:automation/)?(?:n8n|huginn|windmill)/([^/]+\.json)`)

    resourceDetectionExtensions = []string{".go", ".js", ".ts", ".tsx", ".sh", ".py", ".md", ".json", ".yml", ".yaml"}
    scenarioDetectionExtensions = []string{".go", ".js", ".sh", ".py", ".md"}
)

type resourceHeuristic struct {
    Name     string
    Type     string
    Patterns []*regexp.Regexp
}

func determineScenariosDir(dir string) string {
    if dir == "" {
        dir = "../.."
    }
    abs, err := filepath.Abs(dir)
    if err != nil {
        return dir
    }
    return abs
}

func discoverAvailableScenarios(dir string) map[string]struct{} {
    results := map[string]struct{}{}
    entries, err := os.ReadDir(dir)
    if err != nil {
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
    lower := strings.ToLower(relPath)
    base := strings.ToLower(filepath.Base(lower))
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
    segments := strings.Split(lower, string(filepath.Separator))
    for _, segment := range segments {
        if _, ok := analysisIgnoreSegments[segment]; ok {
            return true
        }
    }
    return false
}

func shouldSkipDirectoryEntry(entry fs.DirEntry) bool {
    if !entry.IsDir() {
        return false
    }
    name := entry.Name()
    lower := strings.ToLower(name)
    if _, ok := skipDirectoryNames[lower]; ok {
        return true
    }
    if strings.HasPrefix(lower, "node_modules") {
        return true
    }
    if strings.HasPrefix(lower, ".ignored") {
        return true
    }
    if strings.HasPrefix(name, ".") && name != ".vrooli" {
        return true
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

func normalizeName(name string) string {
    return strings.TrimSpace(strings.ToLower(name))
}

func contains(items []string, target string) bool {
    for _, item := range items {
        if item == target {
            return true
        }
    }
    return false
}

var (
    analysisIgnoreSegments = map[string]struct{}{
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

func augmentResourceDetectionsWithInitialization(results map[string]types.ScenarioDependency, scenarioName string, cfg *types.ServiceConfig) {
    resources := resolvedResourceMap(cfg)
    for resourceName, resource := range resources {
        if len(resource.Initialization) == 0 {
            continue
        }
        canonical := normalizeName(resourceName)
        if canonical == "" {
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

func resolvedResourceMap(cfg *types.ServiceConfig) map[string]types.Resource {
    if cfg == nil {
        return map[string]types.Resource{}
    }
    if cfg.Dependencies.Resources != nil && len(cfg.Dependencies.Resources) > 0 {
        return cfg.Dependencies.Resources
    }
    if cfg.Resources == nil {
        return map[string]types.Resource{}
    }
    return cfg.Resources
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

func newScenarioDependency(source, target, purpose, method, file string) types.ScenarioDependency {
    return types.ScenarioDependency{
        ID:             uuid.New().String(),
        ScenarioName:   source,
        DependencyType: "scenario",
        DependencyName: target,
        Required:       method == "scenario_port_cli",
        Purpose:        purpose,
        AccessMethod:   method,
        Configuration: map[string]interface{}{
            "found_in_file": file,
        },
        DiscoveredAt: time.Now(),
        LastVerified: time.Now(),
    }
}
