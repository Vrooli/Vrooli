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
	for _, name := range []string{"go.mod", "tsconfig.json"} {
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
			units = append(units, goParseUnit(filepath.Dir(path), path))
		case "tsconfig.json":
			units = append(units, tsParseUnit(filepath.Dir(path), path))
		case "package.json":
			dir := filepath.Dir(path)
			if !fileExists(filepath.Join(dir, "tsconfig.json")) && hasScriptSource(dir) {
				units = append(units, nodePackageUnit(dir))
			}
		}
		return nil
	})
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
