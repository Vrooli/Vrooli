package dependencyhealth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

type surfaceInventory struct {
	Surfaces       []*healthv1.DependencyHealthSurface
	DegradedReason string
}

type codeFactsSurfaceDiscoverer struct{}

func (codeFactsSurfaceDiscoverer) Discover(ctx context.Context, scenario, scenarioDir, repoRoot string, useCache bool) (surfaceInventory, error) {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	baseURL, err := resolver.ResolveScenarioURLDefault(ctx, codeFactsScenarioID)
	if err != nil {
		return fallbackSurfaceInventory(scenarioDir, fmt.Sprintf("Code Facts unavailable: %v", err)), nil
	}
	client := factsconnect.NewCodeFactsServiceClient(http.DefaultClient, baseURL)
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
			Scenario: scenario,
			Path:     scenarioDir,
			RepoRoot: repoRoot,
		},
		Include:  []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES, factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
		UseCache: useCache,
	}))
	if err != nil {
		return fallbackSurfaceInventory(scenarioDir, fmt.Sprintf("Code Facts unavailable: %v", err)), nil
	}
	return surfacesFromCodeFacts(resp.Msg, scenarioDir), nil
}

func surfacesFromCodeFacts(report *factsv1.CodeFactsReport, scenarioDir string) surfaceInventory {
	if report == nil {
		return fallbackSurfaceInventory(scenarioDir, "Code Facts returned an empty report")
	}
	root := firstNonEmpty(report.GetTarget().GetRootPath(), scenarioDir)
	parseByRoot := map[string]*factsv1.ParseUnit{}
	for _, unit := range report.GetParseUnits() {
		parseByRoot[filepath.Clean(absPath(root, unit.GetRootPath()))] = unit
	}
	var surfaces []*healthv1.DependencyHealthSurface
	for _, src := range report.GetSurfaces() {
		rootPath := absPath(root, src.GetPath())
		unit := nearestParseUnit(parseByRoot, rootPath)
		surfaces = append(surfaces, &healthv1.DependencyHealthSurface{
			Id:             firstNonEmpty(src.GetId(), filepath.Base(rootPath)),
			Kind:           enumSuffix(src.GetKind().String(), "SURFACE_KIND_"),
			Language:       surfaceLanguage(unit, rootPath),
			Framework:      frameworkFromRoot(rootPath),
			RootPath:       rootPath,
			ParseUnitRoot:  parseUnitRoot(unit, root),
			ConfigPath:     parseUnitConfig(unit, root),
			Status:         enumSuffix(src.GetStatus().String(), "SURFACE_STATUS_"),
			PackageManager: packageManagerFromRoot(rootPath),
			Confidence:     bestConfidence(src.GetEvidence()),
		})
	}
	if len(surfaces) == 0 {
		return fallbackSurfaceInventory(scenarioDir, "Code Facts returned no surfaces")
	}
	sort.Slice(surfaces, func(i, j int) bool { return surfaces[i].GetId() < surfaces[j].GetId() })
	return surfaceInventory{Surfaces: surfaces}
}

func fallbackSurfaceInventory(scenarioDir, reason string) surfaceInventory {
	var surfaces []*healthv1.DependencyHealthSurface
	for _, spec := range []struct {
		id   string
		kind string
		dir  string
	}{
		{id: "api", kind: "api", dir: "api"},
		{id: "cli", kind: "cli", dir: "cli"},
		{id: "ui", kind: "ui", dir: "ui"},
	} {
		root := filepath.Join(scenarioDir, spec.dir)
		if !dirExists(root) {
			continue
		}
		surfaces = append(surfaces, &healthv1.DependencyHealthSurface{
			Id:             spec.id,
			Kind:           spec.kind,
			Language:       languageFromRoot(root),
			Framework:      frameworkFromRoot(root),
			RootPath:       root,
			ParseUnitRoot:  root,
			ConfigPath:     configPathFromRoot(root),
			Status:         "known",
			PackageManager: packageManagerFromRoot(root),
			Confidence:     0.4,
		})
	}
	return surfaceInventory{Surfaces: surfaces, DegradedReason: reason}
}

func nearestParseUnit(units map[string]*factsv1.ParseUnit, root string) *factsv1.ParseUnit {
	root = filepath.Clean(root)
	var best *factsv1.ParseUnit
	bestLen := -1
	for unitRoot, unit := range units {
		if strings.HasPrefix(root, unitRoot) && len(unitRoot) > bestLen {
			best = unit
			bestLen = len(unitRoot)
		}
	}
	return best
}

func surfaceLanguage(unit *factsv1.ParseUnit, root string) string {
	if unit != nil && strings.TrimSpace(unit.GetLanguage()) != "" {
		return normalizedLanguage(unit.GetLanguage())
	}
	return languageFromRoot(root)
}

func normalizedLanguage(language string) string {
	value := strings.ToLower(strings.TrimSpace(language))
	switch value {
	case "js", "javascript":
		return "javascript"
	case "ts", "typescript":
		return "typescript"
	case "py", "python3":
		return "python"
	default:
		return value
	}
}

func parseUnitRoot(unit *factsv1.ParseUnit, root string) string {
	if unit == nil {
		return ""
	}
	return absPath(root, unit.GetRootPath())
}

func parseUnitConfig(unit *factsv1.ParseUnit, root string) string {
	if unit == nil || unit.GetConfigPath() == "" {
		return ""
	}
	return absPath(root, unit.GetConfigPath())
}

func absPath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(root, filepath.FromSlash(path)))
}

func enumSuffix(value, prefix string) string {
	value = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(value)), prefix)
	value = strings.ToLower(value)
	if value == "unspecified" {
		return "unknown"
	}
	return value
}

func bestConfidence(evidence []*factsv1.Evidence) float64 {
	best := 0.0
	for _, ev := range evidence {
		if ev.GetConfidence() > best {
			best = ev.GetConfidence()
		}
	}
	if best == 0 {
		return 0.8
	}
	return best
}

func languageFromRoot(root string) string {
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		return "go"
	case fileExists(filepath.Join(root, "tsconfig.json")):
		return "typescript"
	case fileExists(filepath.Join(root, "package.json")):
		return "javascript"
	case fileExists(filepath.Join(root, "pyproject.toml")), fileExists(filepath.Join(root, "requirements.txt")):
		return "python"
	default:
		return "unknown"
	}
}

func frameworkFromRoot(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return ""
	}
	text := string(data)
	switch {
	case strings.Contains(text, `"vite"`) && strings.Contains(text, `"react"`):
		return "react-vite"
	case strings.Contains(text, `"react"`):
		return "react"
	case strings.Contains(text, `"vite"`):
		return "vite"
	default:
		return "node"
	}
}

func packageManagerFromRoot(root string) string {
	switch {
	case fileExists(filepath.Join(root, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(root, "package-lock.json")):
		return "npm"
	case fileExists(filepath.Join(root, "yarn.lock")):
		return "yarn"
	default:
		return ""
	}
}

func packageManagerForSurface(surface *healthv1.DependencyHealthSurface) string {
	if manager := strings.TrimSpace(surface.GetPackageManager()); manager != "" {
		return manager
	}
	return "pnpm"
}

func configPathFromRoot(root string) string {
	for _, name := range []string{"go.mod", "tsconfig.json", "package.json", "pyproject.toml", "requirements.txt"} {
		path := filepath.Join(root, name)
		if fileExists(path) {
			return path
		}
	}
	return ""
}
