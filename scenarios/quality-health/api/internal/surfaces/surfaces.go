package surfaces

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	repocontract "github.com/vrooli/repo-contract-go"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

const codeFactsScenarioID = "code-facts"

type Surface struct {
	ID             string
	Kind           string
	Language       string
	Framework      string
	RootPath       string
	PackageManager string
	Status         string
	Confidence     float64
}

type Inventory struct {
	Scenario string
	// TargetKind is Code Facts' answer to "where is the code": "scenario" or
	// "path". It is not the governance kind — a path under scenarios/ is
	// "path" here and still a scenario target.
	TargetKind string
	// ValidationTargetKind is the repo-contract governance kind supplied by
	// the caller: scenario, package, control-plane, team, docs, and so on.
	// Empty when the caller did not resolve one.
	ValidationTargetKind string
	RootPath             string
	Surfaces             []Surface
	DegradedReason       string
}

type Locator interface {
	Locate(ctx context.Context, scenario, path string) (scenarioName, targetKind, rootPath string, err error)
}

type Discoverer interface {
	Discover(ctx context.Context, scenario, path string, useCache bool) (Inventory, error)
}

type DefaultLocator struct {
	RepoRoot string
}

func (l DefaultLocator) Locate(_ context.Context, scenario, path string) (string, string, string, error) {
	scenario = strings.TrimSpace(scenario)
	path = strings.TrimSpace(path)
	if path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", "", "", err
		}
		name := scenario
		if name == "" {
			name = filepath.Base(abs)
		}
		return name, "path", abs, nil
	}
	if scenario == "" {
		return "", "", "", errors.New("scenario or path is required")
	}
	repoRoot := strings.TrimSpace(l.RepoRoot)
	if repoRoot == "" {
		root, err := repocontract.ResolveRepoRoot()
		if err != nil {
			return "", "", "", err
		}
		repoRoot = root
	}
	root, err := repocontract.ResolveScenarioPath(repoRoot, scenario)
	if err != nil {
		return "", "", "", err
	}
	return scenario, "scenario", root, nil
}

type CodeFactsClient struct {
	Resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	HTTPClient connect.HTTPClient
	Locator    Locator
}

func (c CodeFactsClient) Discover(ctx context.Context, scenario, path string, useCache bool) (Inventory, error) {
	locator := c.Locator
	if locator == nil {
		locator = DefaultLocator{}
	}
	scenarioName, targetKind, rootPath, err := locator.Locate(ctx, scenario, path)
	if err != nil {
		return Inventory{}, err
	}

	resolver := c.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, err := resolver.ResolveScenarioURLDefault(ctx, codeFactsScenarioID)
	if err != nil {
		inv := fallbackInventory(scenarioName, targetKind, rootPath)
		inv.DegradedReason = fmt.Sprintf("Code Facts unavailable: %v", err)
		return inv, nil
	}
	client := factsconnect.NewCodeFactsServiceClient(httpClient, baseURL)
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     targetKindToProto(targetKind),
			Scenario: scenarioName,
			Path:     rootPath,
		},
		Include:  []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES, factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
		UseCache: useCache,
	}))
	if err != nil {
		inv := fallbackInventory(scenarioName, targetKind, rootPath)
		inv.DegradedReason = fmt.Sprintf("Code Facts unavailable: %v", err)
		return inv, nil
	}
	return fromCodeFacts(resp.Msg, scenarioName, targetKind, rootPath), nil
}

func targetKindToProto(kind string) factsv1.TargetKind {
	if kind == "scenario" {
		return factsv1.TargetKind_TARGET_KIND_SCENARIO
	}
	return factsv1.TargetKind_TARGET_KIND_PATH
}

func fromCodeFacts(report *factsv1.CodeFactsReport, scenarioName, targetKind, rootPath string) Inventory {
	inv := Inventory{Scenario: scenarioName, TargetKind: targetKind, RootPath: rootPath}
	if report == nil {
		inv.DegradedReason = "Code Facts returned an empty report"
		return inv
	}
	if report.GetTarget().GetScenario() != "" {
		inv.Scenario = report.GetTarget().GetScenario()
	}
	if report.GetTarget().GetRootPath() != "" {
		inv.RootPath = report.GetTarget().GetRootPath()
	}
	parseByRoot := map[string]*factsv1.ParseUnit{}
	for _, unit := range report.GetParseUnits() {
		parseByRoot[filepath.Clean(unit.GetRootPath())] = unit
	}
	for _, s := range report.GetSurfaces() {
		root := s.GetPath()
		if !filepath.IsAbs(root) && inv.RootPath != "" {
			root = filepath.Join(inv.RootPath, root)
		}
		unit := nearestParseUnit(parseByRoot, root)
		inv.Surfaces = append(inv.Surfaces, Surface{
			ID:             s.GetId(),
			Kind:           strings.TrimPrefix(strings.ToLower(s.GetKind().String()), "surface_kind_"),
			Language:       languageFromUnit(unit, root),
			Framework:      frameworkFromRoot(root),
			RootPath:       root,
			PackageManager: packageManagerFromRoot(root),
			Status:         strings.TrimPrefix(strings.ToLower(s.GetStatus().String()), "surface_status_"),
			Confidence:     confidence(s.GetEvidence()),
		})
	}
	if len(inv.Surfaces) == 0 {
		fallback := fallbackInventory(inv.Scenario, inv.TargetKind, inv.RootPath)
		fallback.DegradedReason = "Code Facts returned no surfaces"
		return fallback
	}
	return inv
}

func fallbackInventory(scenarioName, targetKind, rootPath string) Inventory {
	inv := Inventory{Scenario: scenarioName, TargetKind: targetKind, RootPath: rootPath}
	for _, spec := range []struct {
		id   string
		kind string
		dir  string
	}{
		{id: "api", kind: "api", dir: "api"},
		{id: "cli", kind: "cli", dir: "cli"},
		{id: "ui", kind: "ui", dir: "ui"},
	} {
		dir := filepath.Join(rootPath, spec.dir)
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			inv.Surfaces = append(inv.Surfaces, Surface{
				ID:             spec.id,
				Kind:           spec.kind,
				Language:       languageFromRoot(dir),
				Framework:      frameworkFromRoot(dir),
				RootPath:       dir,
				PackageManager: packageManagerFromRoot(dir),
				Status:         "known",
				Confidence:     0.4,
			})
		}
	}
	return inv
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

func confidence(evidence []*factsv1.Evidence) float64 {
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

func languageFromUnit(unit *factsv1.ParseUnit, root string) string {
	if unit != nil && unit.GetLanguage() != "" {
		return strings.ToLower(unit.GetLanguage())
	}
	return languageFromRoot(root)
}

func languageFromRoot(root string) string {
	switch {
	case exists(filepath.Join(root, "go.mod")):
		return "go"
	case exists(filepath.Join(root, "tsconfig.json")):
		return "typescript"
	case exists(filepath.Join(root, "package.json")):
		return "javascript"
	default:
		return "unknown"
	}
}

func frameworkFromRoot(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return ""
	}
	text := string(raw)
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
	case exists(filepath.Join(root, "pnpm-lock.yaml")):
		return "pnpm"
	case exists(filepath.Join(root, "package-lock.json")):
		return "npm"
	case exists(filepath.Join(root, "yarn.lock")):
		return "yarn"
	default:
		return ""
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
