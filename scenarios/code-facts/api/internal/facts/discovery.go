package facts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func discoverSurfaces(target *factsv1.TargetContext) []*factsv1.Surface {
	if !target.GetScenarioAware() {
		return []*factsv1.Surface{{
			Id:       "target",
			Path:     target.GetRootPath(),
			Status:   factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN,
			Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Generic target exists.", target.GetRootPath())},
		}}
	}

	root := target.GetRootPath()
	surfaces := []*factsv1.Surface{
		scenarioSurface(root, "api", factsv1.SurfaceKind_SURFACE_KIND_API),
		scenarioSurface(root, "cli", factsv1.SurfaceKind_SURFACE_KIND_CLI),
		scenarioSurface(root, "runtime", factsv1.SurfaceKind_SURFACE_KIND_RUNTIME),
		scenarioSurface(root, "ui", factsv1.SurfaceKind_SURFACE_KIND_UI),
	}
	for _, name := range []string{"sidecar", "sidecars", "workers", "jobs"} {
		path := filepath.Join(root, name)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			kind := factsv1.SurfaceKind_SURFACE_KIND_SIDECAR
			if name == "workers" {
				kind = factsv1.SurfaceKind_SURFACE_KIND_WORKER
			}
			if name == "jobs" {
				kind = factsv1.SurfaceKind_SURFACE_KIND_JOB
			}
			surfaces = append(surfaces, &factsv1.Surface{
				Id:       name,
				Kind:     kind,
				Path:     path,
				Status:   unsupportedIfNoParseUnit(path),
				Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, "Optional execution surface is present; analyzer support depends on discovered parse units.", path)},
			})
		}
	}

	return addManifestEvidence(root, surfaces)
}

func scenarioSurface(root, id string, kind factsv1.SurfaceKind) *factsv1.Surface {
	path := filepath.Join(root, id)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return &factsv1.Surface{
			Id:       id,
			Kind:     kind,
			Path:     path,
			Status:   factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN,
			Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, id+" surface directory exists.", path)},
		}
	}
	return &factsv1.Surface{
		Id:       id,
		Kind:     kind,
		Path:     path,
		Status:   factsv1.SurfaceStatus_SURFACE_STATUS_MISSING,
		Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING, id+" surface directory is absent.", path)},
	}
}

func unsupportedIfNoParseUnit(root string) factsv1.SurfaceStatus {
	for _, name := range []string{"go.mod", "tsconfig.json", "package.json", "pnpm-lock.yaml", "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "bun.lock", "bun.lockb", "requirements.txt", "Pipfile.lock", "poetry.lock", "pyproject.toml", "Cargo.lock", "Cargo.toml"} {
		if fileExists(filepath.Join(root, name)) {
			return factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN
		}
	}
	return factsv1.SurfaceStatus_SURFACE_STATUS_UNSUPPORTED
}

func addManifestEvidence(root string, surfaces []*factsv1.Surface) []*factsv1.Surface {
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	var service struct {
		Components map[string]componentSurfaceDeclaration `json:"components"`
		CLI        struct {
			Enabled bool `json:"enabled"`
			Adapter struct {
				Kind      string `json:"kind"`
				ModuleDir string `json:"module_dir"`
			} `json:"adapter"`
		} `json:"cli"`
	}
	if readJSON(servicePath, &service) == nil {
		ids := make([]string, 0, len(service.Components))
		for id := range service.Components {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			declared := service.Components[id].surface(root, id, servicePath)
			if existing := findSurface(surfaces, id); existing != nil {
				*existing = *declared
			} else {
				surfaces = append(surfaces, declared)
			}
		}
		if cli := findSurface(surfaces, "cli"); cli != nil && service.CLI.Enabled {
			cli.Evidence = append(cli.Evidence, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "service.json declares CLI adapter "+service.CLI.Adapter.Kind+".", servicePath))
		}
	}
	endpointsPath := filepath.Join(root, ".vrooli", "endpoints.json")
	if fileExists(endpointsPath) {
		if api := findSurface(surfaces, "api"); api != nil {
			api.Evidence = append(api.Evidence, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "endpoints.json declares API endpoint metadata.", endpointsPath))
		}
	}
	cliManifestPath := filepath.Join(root, "cli", "manifest.json")
	if fileExists(cliManifestPath) {
		if cli := findSurface(surfaces, "cli"); cli != nil {
			cli.Evidence = append(cli.Evidence, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "cli/manifest.json declares CLI command metadata.", cliManifestPath))
		}
	}
	return surfaces
}

type componentSurfaceDeclaration struct {
	Role  string `json:"role"`
	Build struct {
		Dir string `json:"dir"`
	} `json:"build"`
	Run struct {
		CWD string `json:"cwd"`
	} `json:"run"`
}

func (c componentSurfaceDeclaration) surface(root, id, manifestPath string) *factsv1.Surface {
	kind := factsv1.SurfaceKind(factsv1.SurfaceKind_value["SURFACE_KIND_"+strings.ToUpper(c.Role)])
	surface := &factsv1.Surface{Id: id, Kind: kind, Status: factsv1.SurfaceStatus_SURFACE_STATUS_UNSUPPORTED}
	dir := c.Build.Dir
	if dir == "" {
		dir = c.Run.CWD
	}
	if dir == "" {
		dir = "."
	}
	if !filepath.IsLocal(dir) || kind == factsv1.SurfaceKind_SURFACE_KIND_UNSPECIFIED {
		surface.Evidence = []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "Declared component has an unsupported role or a root outside the target.", manifestPath)}
		return surface
	}
	path := filepath.Join(root, dir)
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		resolvedRoot, rootErr := filepath.EvalSymlinks(root)
		rel, relErr := filepath.Rel(resolvedRoot, resolved)
		if rootErr != nil || relErr != nil || !filepath.IsLocal(rel) {
			surface.Evidence = []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "Declared component resolves outside the target.", manifestPath)}
			return surface
		}
	}
	surface.Path = path
	surface.Evidence = []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "service.json declares component role "+c.Role+".", manifestPath)}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		surface.Status = unsupportedIfNoParseUnit(path)
	} else if os.IsNotExist(err) {
		surface.Status = factsv1.SurfaceStatus_SURFACE_STATUS_MISSING
	}
	return surface
}

func findSurface(surfaces []*factsv1.Surface, id string) *factsv1.Surface {
	for _, surface := range surfaces {
		if surface.GetId() == id {
			return surface
		}
	}
	return nil
}

func discoverParseUnits(target *factsv1.TargetContext) []*factsv1.ParseUnit {
	if target.GetResolvedKind() == factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE {
		if unit := discoverControlPlaneGoUnit(target); unit != nil {
			return []*factsv1.ParseUnit{unit}
		}
	}
	if roots := target.GetRootPaths(); len(roots) > 0 {
		var units []*factsv1.ParseUnit
		for _, root := range roots {
			child := &factsv1.TargetContext{
				RootPath:      root,
				Scenario:      scenarioFromPath(root, filepath.Dir(filepath.Dir(root))),
				ScenarioAware: hasServiceManifest(root),
			}
			units = append(units, discoverParseUnitsAtRoot(child)...)
		}
		sort.SliceStable(units, func(i, j int) bool { return units[i].GetId() < units[j].GetId() })
		return dedupeParseUnits(units)
	}
	return discoverParseUnitsAtRoot(target)
}

func discoverControlPlaneGoUnit(target *factsv1.TargetContext) *factsv1.ParseUnit {
	root := target.GetRootPath()
	gomod := filepath.Join(root, "go.mod")
	if !fileExists(gomod) {
		return nil
	}
	patterns := make([]string, 0, len(target.GetRootPaths()))
	for _, path := range target.GetRootPaths() {
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		patterns = append(patterns, "./"+filepath.ToSlash(rel)+"/...")
	}
	if len(patterns) == 0 {
		return nil
	}
	unit := goParseUnit(root, gomod, hasSourceWithExtension(root, ".bats"))
	unit.PackagePatterns = patterns
	return unit
}

func discoverParseUnitsAtRoot(target *factsv1.TargetContext) []*factsv1.ParseUnit {
	root := target.GetRootPath()
	if isFile(root) {
		if filepath.Base(root) == "tsconfig.json" {
			return []*factsv1.ParseUnit{tsParseUnit(filepath.Dir(root), root, hasSourceWithExtension(filepath.Dir(root), ".bats"))}
		}
		root = filepath.Dir(root)
	}

	var units []*factsv1.ParseUnit
	var sourceByDir map[string]sourceKinds
	scenarioRoot := ""
	if target.GetScenarioAware() {
		scenarioRoot = nearestScenarioRoot(root)
	}
	if scenarioRoot == "" || filepath.Clean(root) != scenarioRoot {
		if goMod := nearestFile(root, "go.mod"); goMod != "" && (scenarioRoot == "" || strings.HasPrefix(goMod, scenarioRoot+string(filepath.Separator))) {
			moduleRoot := filepath.Dir(goMod)
			unit := goParseUnit(moduleRoot, goMod, hasSourceWithExtension(moduleRoot, ".bats"))
			unit.PackagePatterns = scopedPackagePatterns(moduleRoot, root)
			units = append(units, unit)
		}
		if tsconfig := nearestFile(root, "tsconfig.json"); tsconfig != "" && (scenarioRoot == "" || strings.HasPrefix(tsconfig, scenarioRoot+string(filepath.Separator))) {
			units = append(units, tsParseUnit(filepath.Dir(tsconfig), tsconfig, hasSourceWithExtension(filepath.Dir(tsconfig), ".bats")))
		}
	}

	if len(units) == 0 && isDir(root) {
		units, sourceByDir = discoverNestedParseUnits(root)
	}
	if len(units) == 0 && fileExists(filepath.Join(root, "package.json")) {
		kinds := sourceByDir[filepath.Clean(root)]
		units = append(units, nodePackageUnit(root, kinds.javascript, kinds.bats))
	}
	if len(units) == 0 {
		if manifest, language := dependencyManifest(root); manifest != "" {
			units = append(units, dependencyParseUnit(root, manifest, language, sourceByDir[filepath.Clean(root)].bats))
		}
	}
	if len(units) == 0 {
		units = append(units, unknownUnit(root))
	}

	sort.SliceStable(units, func(i, j int) bool { return units[i].GetId() < units[j].GetId() })
	return dedupeParseUnits(units)
}

// scopedPackagePatterns narrows a Go parse unit to the directory the caller
// asked about when that directory sits inside a module. Without the scope, a
// PATH target like <module>/internal loads every package in the module, and a
// module tree that also contains nested modules is rejected outright as
// multiple_go_mod_files. An empty result means the target is the module root,
// where the loader's default ./... is already correct.
func scopedPackagePatterns(moduleRoot, targetRoot string) []string {
	rel, err := filepath.Rel(filepath.Clean(moduleRoot), filepath.Clean(targetRoot))
	if err != nil || rel == "." || !filepath.IsLocal(rel) {
		return nil
	}
	return []string{"./" + filepath.ToSlash(rel) + "/..."}
}

func nearestScenarioRoot(start string) string {
	dir := start
	if isFile(dir) {
		dir = filepath.Dir(dir)
	}
	for {
		if hasServiceManifest(dir) {
			return filepath.Clean(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

type sourceKinds struct {
	script     bool
	javascript bool
	bats       bool
}

type parseManifest struct {
	path string
	name string
}

func discoverNestedParseUnits(root string) ([]*factsv1.ParseUnit, map[string]sourceKinds) {
	// Bash has no manifest file, so infer shell units from source paths not
	// owned by a manifest language. The same walk also records source kinds for
	// each ancestor directory; package routing and toolchain observations reuse
	// that inventory instead of recursively rescanning each package subtree.
	manifestDirs := map[string]bool{}
	shellDirs := map[string]bool{}
	sourceByDir := map[string]sourceKinds{}
	manifests := []parseManifest{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && shouldPruneDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		recordSourceKinds(root, path, sourceByDir)
		switch d.Name() {
		case "go.mod", "tsconfig.json", "package.json", "pnpm-lock.yaml", "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "bun.lock", "bun.lockb", "requirements.txt", "Pipfile.lock", "poetry.lock", "pyproject.toml", "Cargo.lock", "Cargo.toml":
			dir := filepath.Dir(path)
			manifestDirs[dir] = true
			manifests = append(manifests, parseManifest{path: path, name: d.Name()})
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".sh", ".bats":
			shellDirs[filepath.Dir(path)] = true
		}
		return nil
	})

	var units []*factsv1.ParseUnit
	for _, manifest := range manifests {
		dir := filepath.Dir(manifest.path)
		kinds := sourceByDir[dir]
		switch manifest.name {
		case "go.mod":
			units = append(units, goParseUnit(dir, manifest.path, kinds.bats))
		case "tsconfig.json":
			units = append(units, tsParseUnit(dir, manifest.path, kinds.bats))
		case "package.json":
			if !fileExists(filepath.Join(dir, "tsconfig.json")) && kinds.script {
				units = append(units, nodePackageUnit(dir, kinds.javascript, kinds.bats))
			}
		case "pnpm-lock.yaml", "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "bun.lock", "bun.lockb":
			units = append(units, dependencyParseUnit(dir, manifest.path, "node", kinds.bats))
		case "requirements.txt", "Pipfile.lock", "poetry.lock", "pyproject.toml":
			units = append(units, dependencyParseUnit(dir, manifest.path, "python", kinds.bats))
		case "Cargo.lock", "Cargo.toml":
			units = append(units, dependencyParseUnit(dir, manifest.path, "rust", kinds.bats))
		}
	}
	units = append(units, bashParseUnits(shellDirs, manifestDirs, sourceByDir)...)
	return units, sourceByDir
}

func recordSourceKinds(root, path string, sourceByDir map[string]sourceKinds) {
	extension := strings.ToLower(filepath.Ext(path))
	kinds := sourceKinds{
		script:     extension == ".ts" || extension == ".tsx" || extension == ".js" || extension == ".jsx",
		javascript: extension == ".js" || extension == ".jsx" || extension == ".mjs" || extension == ".cjs",
		bats:       extension == ".bats",
	}
	if !kinds.script && !kinds.javascript && !kinds.bats {
		return
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		current := sourceByDir[dir]
		current.script = current.script || kinds.script
		current.javascript = current.javascript || kinds.javascript
		current.bats = current.bats || kinds.bats
		sourceByDir[dir] = current
		if dir == root {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
	}
}

// bashParseUnits emits one bash parse unit per shell-source directory that is
// not owned by a manifest language, collapsing nested shell directories to
// their shallowest ancestor so a tree of scripts reports as a single unit.
func bashParseUnits(shellDirs, manifestDirs map[string]bool, sourceByDir map[string]sourceKinds) []*factsv1.ParseUnit {
	candidates := make([]string, 0, len(shellDirs))
	for dir := range shellDirs {
		if manifestDirs[dir] {
			continue
		}
		candidates = append(candidates, dir)
	}
	sort.Strings(candidates)
	var roots []string
	for _, dir := range candidates {
		nested := false
		for _, kept := range roots {
			if dir == kept || strings.HasPrefix(dir, kept+string(filepath.Separator)) {
				nested = true
				break
			}
		}
		if !nested {
			roots = append(roots, dir)
		}
	}
	units := make([]*factsv1.ParseUnit, 0, len(roots))
	for _, dir := range roots {
		units = append(units, bashParseUnit(dir, sourceByDir[dir].bats))
	}
	return units
}

func goParseUnit(root, gomod string, hasBats bool) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:         "go:" + filepath.ToSlash(root),
		Language:   "go",
		RootPath:   root,
		ConfigPath: gomod,
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Go module discovered from go.mod.", gomod)},
		Toolchain:  toolchainObservation(root, "go", gomod, hasBats),
	}
}

func tsParseUnit(root, tsconfig string, hasBats bool) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:         "typescript:" + filepath.ToSlash(root),
		Language:   "typescript",
		RootPath:   root,
		ConfigPath: tsconfig,
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "TypeScript project discovered from tsconfig.json.", tsconfig)},
		Toolchain:  toolchainObservation(root, "node", tsconfig, hasBats),
	}
}

func nodePackageUnit(root string, hasJavaScriptFiles, hasBats bool) *factsv1.ParseUnit {
	packagePath := filepath.Join(root, "package.json")
	if hasJavaScriptFiles {
		return &factsv1.ParseUnit{
			Id:         "javascript:" + filepath.ToSlash(root),
			Language:   "javascript",
			RootPath:   root,
			ConfigPath: packagePath,
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
			Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "JavaScript package discovered and routed to the shared TypeScript AST provider.", packagePath)},
			Toolchain:  toolchainObservation(root, "node", packagePath, hasBats),
		}
	}
	return &factsv1.ParseUnit{
		Id:         "node:" + filepath.ToSlash(root),
		Language:   "node",
		RootPath:   root,
		ConfigPath: packagePath,
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "Node package has no tsconfig.json, so it cannot route to a supported analyzer.", packagePath)},
		Toolchain:  toolchainObservation(root, "node", packagePath, hasBats),
	}
}

func dependencyManifest(root string) (string, string) {
	manifests := []struct {
		name     string
		language string
	}{
		{"pnpm-lock.yaml", "node"},
		{"package-lock.json", "node"},
		{"npm-shrinkwrap.json", "node"},
		{"yarn.lock", "node"},
		{"bun.lock", "node"},
		{"bun.lockb", "node"},
		{"requirements.txt", "python"},
		{"Pipfile.lock", "python"},
		{"poetry.lock", "python"},
		{"pyproject.toml", "python"},
		{"Cargo.lock", "rust"},
		{"Cargo.toml", "rust"},
	}
	for _, manifest := range manifests {
		path := filepath.Join(root, manifest.name)
		if fileExists(path) {
			return path, manifest.language
		}
	}
	return "", ""
}

func dependencyParseUnit(root, manifest, language string, hasBats bool) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:         language + ":dependency:" + filepath.ToSlash(root),
		Language:   language,
		RootPath:   root,
		ConfigPath: manifest,
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Dependency manifest discovered for downstream typed ecosystem adapters; Code Facts does not parse package vulnerability state.", manifest)},
		Toolchain:  toolchainObservation(root, language, manifest, hasBats),
	}
}

// bashParseUnit reports a discovered shell-script root. Code Facts has no bash
// analyzer, so the unit is UNSUPPORTED (like a node package without tsconfig):
// the language and root are proven facts, but no graph is produced. Consumers
// such as unit-health use the language + root to attribute bats test surfaces.
func bashParseUnit(root string, hasBats bool) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:        "bash:" + filepath.ToSlash(root),
		Language:  "bash",
		RootPath:  root,
		Status:    factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Evidence:  []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "Shell scripts (.sh/.bats) discovered; Code Facts reports the language but has no bash graph analyzer.", root)},
		Toolchain: toolchainObservation(root, "bash", "", hasBats),
	}
}

// toolchainObservation reports observed manifests and neutral indicators. It
// deliberately does not select or validate a framework or adapter.
func toolchainObservation(root, ecosystem, primaryManifest string, hasBats bool) *factsv1.ToolchainObservation {
	observation := &factsv1.ToolchainObservation{
		Ecosystem: ecosystem,
		Status:    factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
	}
	if primaryManifest != "" {
		observation.ManifestPaths = append(observation.ManifestPaths, primaryManifest)
		observation.Evidence = append(observation.Evidence, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Toolchain manifest observed.", primaryManifest))
	}
	for _, name := range []string{"go.mod", "tsconfig.json", "package.json", "pyproject.toml", "requirements.txt", "Cargo.toml"} {
		path := filepath.Join(root, name)
		if fileExists(path) && !containsPath(observation.ManifestPaths, path) {
			observation.ManifestPaths = append(observation.ManifestPaths, path)
		}
	}
	for _, name := range []string{"pnpm-lock.yaml", "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "bun.lock", "bun.lockb", "go.sum", "Cargo.lock", "poetry.lock", "Pipfile.lock"} {
		path := filepath.Join(root, name)
		if fileExists(path) {
			observation.LockfilePaths = append(observation.LockfilePaths, path)
		}
	}
	if fileExists(filepath.Join(root, "go.mod")) {
		observation.BuildSystems = append(observation.BuildSystems, "go-module")
	}
	if fileExists(filepath.Join(root, "Cargo.toml")) {
		observation.BuildSystems = append(observation.BuildSystems, "cargo")
	}
	if content, err := os.ReadFile(filepath.Join(root, "go.mod")); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "go" {
				observation.ToolchainIdentity = "go@" + fields[1]
				break
			}
		}
	}
	packagePath := filepath.Join(root, "package.json")
	if fileExists(packagePath) {
		var manifest struct {
			Scripts         map[string]string `json:"scripts"`
			PackageManager  string            `json:"packageManager"`
			Dependencies    map[string]any    `json:"dependencies"`
			DevDependencies map[string]any    `json:"devDependencies"`
		}
		if readJSON(packagePath, &manifest) == nil {
			observation.PackageManager = manifest.PackageManager
			for key := range manifest.Scripts {
				observation.RunnerIndicators = append(observation.RunnerIndicators, "script:"+key)
			}
			for key := range manifest.Dependencies {
				observation.RunnerIndicators = append(observation.RunnerIndicators, "dependency:"+key)
			}
			for key := range manifest.DevDependencies {
				observation.RunnerIndicators = append(observation.RunnerIndicators, "devDependency:"+key)
			}
		}
	}
	if hasBats {
		observation.RunnerIndicators = append(observation.RunnerIndicators, "source:.bats")
	}
	if ecosystem == "python" {
		for _, name := range []string{"pyproject.toml", "setup.cfg", "setup.py"} {
			path := filepath.Join(root, name)
			if content, err := os.ReadFile(path); err == nil && strings.Contains(strings.ToLower(string(content)), "pytest") {
				observation.RunnerIndicators = append(observation.RunnerIndicators, "config:pytest")
			}
		}
	}
	sort.Strings(observation.ManifestPaths)
	sort.Strings(observation.LockfilePaths)
	sort.Strings(observation.BuildSystems)
	sort.Strings(observation.RunnerIndicators)
	return observation
}

func containsPath(paths []string, candidate string) bool {
	for _, path := range paths {
		if filepath.Clean(path) == filepath.Clean(candidate) {
			return true
		}
	}
	return false
}

func unknownUnit(root string) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:        "unknown:" + filepath.ToSlash(root),
		Language:  "unknown",
		RootPath:  root,
		Status:    factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Evidence:  []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "No bounded Go module, TypeScript project, or supported package manifest was discovered.", root)},
		Toolchain: &factsv1.ToolchainObservation{Ecosystem: "unknown", Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, "No toolchain manifest or source indicator was discovered.", root)}},
	}
}

func dedupeParseUnits(units []*factsv1.ParseUnit) []*factsv1.ParseUnit {
	seen := map[string]bool{}
	out := make([]*factsv1.ParseUnit, 0, len(units))
	for _, unit := range units {
		if seen[unit.GetId()] {
			continue
		}
		seen[unit.GetId()] = true
		out = append(out, unit)
	}
	return out
}

func nearestFile(start, name string) string {
	dir := start
	for {
		candidate := filepath.Join(dir, name)
		if fileExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func hasSourceWithExtension(root string, extensions ...string) bool {
	wanted := make(map[string]struct{}, len(extensions))
	for _, extension := range extensions {
		wanted[strings.ToLower(extension)] = struct{}{}
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if entry.IsDir() {
			if path != root && shouldPruneDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := wanted[strings.ToLower(filepath.Ext(path))]; ok {
			found = true
		}
		return nil
	})
	return found
}

func shouldPruneDir(name string) bool {
	if strings.HasPrefix(name, ".proto-gen-stage-") {
		return true
	}
	// React Component Library writes throwaway tsconfigs here to ask the TS
	// parser for import evidence. They are runtime scratch projects, not source
	// owners; discovering them recursively creates duplicate graph units and
	// feeds potentially incomplete temporary inputs into repo-wide analysis.
	if strings.HasPrefix(name, ".graph-project-") {
		return true
	}
	switch name {
	case ".git", "node_modules", "dist", "coverage", "data", ".cache", "build":
		return true
	default:
		return false
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func readJSON(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
