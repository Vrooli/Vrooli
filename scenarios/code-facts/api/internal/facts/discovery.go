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

	addManifestEvidence(root, surfaces)
	return surfaces
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

func addManifestEvidence(root string, surfaces []*factsv1.Surface) {
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	var service struct {
		CLI struct {
			Enabled bool `json:"enabled"`
			Adapter struct {
				Kind      string `json:"kind"`
				ModuleDir string `json:"module_dir"`
			} `json:"adapter"`
		} `json:"cli"`
	}
	if readJSON(servicePath, &service) == nil && service.CLI.Enabled {
		if cli := findSurface(surfaces, "cli"); cli != nil {
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
	root := target.GetRootPath()
	if isFile(root) {
		if filepath.Base(root) == "tsconfig.json" {
			return []*factsv1.ParseUnit{tsParseUnit(filepath.Dir(root), root)}
		}
		root = filepath.Dir(root)
	}

	var units []*factsv1.ParseUnit
	scenarioRoot := ""
	if target.GetScenarioAware() {
		scenarioRoot = nearestScenarioRoot(root)
	}
	if scenarioRoot == "" || filepath.Clean(root) != scenarioRoot {
		if goMod := nearestFile(root, "go.mod"); goMod != "" && (scenarioRoot == "" || strings.HasPrefix(goMod, scenarioRoot+string(filepath.Separator))) {
			units = append(units, goParseUnit(filepath.Dir(goMod), goMod))
		}
		if tsconfig := nearestFile(root, "tsconfig.json"); tsconfig != "" && (scenarioRoot == "" || strings.HasPrefix(tsconfig, scenarioRoot+string(filepath.Separator))) {
			units = append(units, tsParseUnit(filepath.Dir(tsconfig), tsconfig))
		}
	}

	if len(units) == 0 && isDir(root) {
		units = append(units, discoverNestedParseUnits(root)...)
	}
	if len(units) == 0 && fileExists(filepath.Join(root, "package.json")) {
		units = append(units, nodePackageUnit(root))
	}
	if len(units) == 0 {
		if manifest, language := dependencyManifest(root); manifest != "" {
			units = append(units, dependencyParseUnit(root, manifest, language))
		}
	}
	if len(units) == 0 {
		units = append(units, unknownUnit(root))
	}

	sort.SliceStable(units, func(i, j int) bool { return units[i].GetId() < units[j].GetId() })
	return dedupeParseUnits(units)
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

func discoverNestedParseUnits(root string) []*factsv1.ParseUnit {
	var units []*factsv1.ParseUnit
	// Bash has no manifest file, so we infer a shell parse unit from the
	// presence of .sh/.bats sources in a directory not already owned by a
	// language with a manifest (go.mod/tsconfig/package.json).
	manifestDirs := map[string]bool{}
	shellDirs := map[string]bool{}
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
		switch d.Name() {
		case "go.mod":
			dir := filepath.Dir(path)
			manifestDirs[dir] = true
			units = append(units, goParseUnit(dir, path))
		case "tsconfig.json":
			dir := filepath.Dir(path)
			manifestDirs[dir] = true
			units = append(units, tsParseUnit(dir, path))
		case "package.json":
			dir := filepath.Dir(path)
			manifestDirs[dir] = true
			if !fileExists(filepath.Join(dir, "tsconfig.json")) && hasScriptSource(dir) {
				units = append(units, nodePackageUnit(dir))
			}
		case "pnpm-lock.yaml", "package-lock.json", "npm-shrinkwrap.json":
			dir := filepath.Dir(path)
			manifestDirs[dir] = true
			units = append(units, dependencyParseUnit(dir, path, "node"))
		case "yarn.lock", "bun.lock", "bun.lockb":
			dir := filepath.Dir(path)
			manifestDirs[dir] = true
			units = append(units, dependencyParseUnit(dir, path, "node"))
		case "requirements.txt", "Pipfile.lock", "poetry.lock", "pyproject.toml":
			dir := filepath.Dir(path)
			manifestDirs[dir] = true
			units = append(units, dependencyParseUnit(dir, path, "python"))
		case "Cargo.lock", "Cargo.toml":
			dir := filepath.Dir(path)
			manifestDirs[dir] = true
			units = append(units, dependencyParseUnit(dir, path, "rust"))
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".sh", ".bats":
			shellDirs[filepath.Dir(path)] = true
		}
		return nil
	})
	units = append(units, bashParseUnits(shellDirs, manifestDirs)...)
	return units
}

// bashParseUnits emits one bash parse unit per shell-source directory that is
// not owned by a manifest language, collapsing nested shell directories to
// their shallowest ancestor so a tree of scripts reports as a single unit.
func bashParseUnits(shellDirs, manifestDirs map[string]bool) []*factsv1.ParseUnit {
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
		units = append(units, bashParseUnit(dir))
	}
	return units
}

func goParseUnit(root, gomod string) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:         "go:" + filepath.ToSlash(root),
		Language:   "go",
		RootPath:   root,
		ConfigPath: gomod,
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Go module discovered from go.mod.", gomod)},
	}
}

func tsParseUnit(root, tsconfig string) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:         "typescript:" + filepath.ToSlash(root),
		Language:   "typescript",
		RootPath:   root,
		ConfigPath: tsconfig,
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "TypeScript project discovered from tsconfig.json.", tsconfig)},
	}
}

func nodePackageUnit(root string) *factsv1.ParseUnit {
	packagePath := filepath.Join(root, "package.json")
	return &factsv1.ParseUnit{
		Id:         "node:" + filepath.ToSlash(root),
		Language:   "node",
		RootPath:   root,
		ConfigPath: packagePath,
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "Node package has no tsconfig.json, so it cannot route to a supported analyzer.", packagePath)},
	}
}

func dependencyManifest(root string) (string, string) {
	manifests := []struct {
		name     string
		language string
	}{
		{"pnpm-lock.yaml", "node"}, {"package-lock.json", "node"}, {"npm-shrinkwrap.json", "node"},
		{"yarn.lock", "node"}, {"bun.lock", "node"}, {"bun.lockb", "node"},
		{"requirements.txt", "python"}, {"Pipfile.lock", "python"}, {"poetry.lock", "python"}, {"pyproject.toml", "python"},
		{"Cargo.lock", "rust"}, {"Cargo.toml", "rust"},
	}
	for _, manifest := range manifests {
		path := filepath.Join(root, manifest.name)
		if fileExists(path) {
			return path, manifest.language
		}
	}
	return "", ""
}

func dependencyParseUnit(root, manifest, language string) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:         language + ":dependency:" + filepath.ToSlash(root),
		Language:   language,
		RootPath:   root,
		ConfigPath: manifest,
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Evidence:   []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Dependency manifest discovered for downstream typed ecosystem adapters; Code Facts does not parse package vulnerability state.", manifest)},
	}
}

// bashParseUnit reports a discovered shell-script root. Code Facts has no bash
// analyzer, so the unit is UNSUPPORTED (like a node package without tsconfig):
// the language and root are proven facts, but no graph is produced. Consumers
// such as unit-health use the language + root to attribute bats test surfaces.
func bashParseUnit(root string) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:       "bash:" + filepath.ToSlash(root),
		Language: "bash",
		RootPath: root,
		Status:   factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "Shell scripts (.sh/.bats) discovered; Code Facts reports the language but has no bash graph analyzer.", root)},
	}
}

func unknownUnit(root string) *factsv1.ParseUnit {
	return &factsv1.ParseUnit{
		Id:       "unknown:" + filepath.ToSlash(root),
		Language: "unknown",
		RootPath: root,
		Status:   factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "No bounded Go module, TypeScript project, or supported package manifest was discovered.", root)},
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

func hasScriptSource(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			if path != root && shouldPruneDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".ts", ".tsx", ".js", ".jsx":
			found = true
		}
		return nil
	})
	return found
}

func shouldPruneDir(name string) bool {
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
