// Package profile is structure-health's code-facts intake seam. It asks Code
// Facts for the target scenario's surfaces + parse units and derives the
// language/framework profile that keys structure-health's conformance rules.
//
// Code Facts is the primary source. When it is unavailable or returns nothing,
// the package degrades to a filesystem scan and records an explicit
// DegradedReason so the validator never silently invents (or omits) surfaces.
package profile

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

// Surface is a detected scenario surface tagged with the language/framework
// facts structure-health needs.
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

// Facts is the normalized code-facts result for a target.
type Facts struct {
	Scenario       string
	TargetKind     string
	RootPath       string
	Surfaces       []Surface
	DegradedReason string
}

// Locator resolves a scenario id or path to a concrete root directory.
type Locator interface {
	Locate(ctx context.Context, scenario, path string) (scenarioName, targetKind, rootPath string, err error)
}

// Describer returns the code-facts surface inventory for a target. The seam lets
// tests inject facts without a running Code Facts service.
type Describer interface {
	Describe(ctx context.Context, scenario, path string) (Facts, error)
}

// DefaultLocator resolves scenario ids through the repo-contract package.
type DefaultLocator struct {
	RepoRoot string
}

// Locate resolves scenario/path into (name, kind, absolute root).
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

// CodeFactsClient describes surfaces by calling the Code Facts service, falling
// back to a filesystem scan when it is unavailable.
type CodeFactsClient struct {
	Resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	HTTPClient connect.HTTPClient
	Locator    Locator
}

var _ Describer = CodeFactsClient{}

// Describe resolves the target and asks Code Facts for surfaces + parse units.
func (c CodeFactsClient) Describe(ctx context.Context, scenario, path string) (Facts, error) {
	locator := c.Locator
	if locator == nil {
		locator = DefaultLocator{}
	}
	scenarioName, targetKind, rootPath, err := locator.Locate(ctx, scenario, path)
	if err != nil {
		return Facts{}, err
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
		f := fallbackFacts(scenarioName, targetKind, rootPath)
		f.DegradedReason = fmt.Sprintf("Code Facts unavailable: %v", err)
		return f, nil
	}
	client := factsconnect.NewCodeFactsServiceClient(httpClient, baseURL)
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     targetKindToProto(targetKind),
			Scenario: scenarioName,
			Path:     rootPath,
		},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES, factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
	}))
	if err != nil {
		f := fallbackFacts(scenarioName, targetKind, rootPath)
		f.DegradedReason = fmt.Sprintf("Code Facts unavailable: %v", err)
		return f, nil
	}
	return fromCodeFacts(resp.Msg, scenarioName, targetKind, rootPath), nil
}

func targetKindToProto(kind string) factsv1.TargetKind {
	if kind == "scenario" {
		return factsv1.TargetKind_TARGET_KIND_SCENARIO
	}
	return factsv1.TargetKind_TARGET_KIND_PATH
}

func fromCodeFacts(report *factsv1.CodeFactsReport, scenarioName, targetKind, rootPath string) Facts {
	f := Facts{Scenario: scenarioName, TargetKind: targetKind, RootPath: rootPath}
	if report == nil {
		f.DegradedReason = "Code Facts returned an empty report"
		return f
	}
	if report.GetTarget().GetScenario() != "" {
		f.Scenario = report.GetTarget().GetScenario()
	}
	if report.GetTarget().GetRootPath() != "" {
		f.RootPath = report.GetTarget().GetRootPath()
	}
	parseByRoot := map[string]*factsv1.ParseUnit{}
	for _, unit := range report.GetParseUnits() {
		parseByRoot[filepath.Clean(unit.GetRootPath())] = unit
	}
	for _, s := range report.GetSurfaces() {
		root := s.GetPath()
		if !filepath.IsAbs(root) && f.RootPath != "" {
			root = filepath.Join(f.RootPath, root)
		}
		unit := nearestParseUnit(parseByRoot, root)
		f.Surfaces = append(f.Surfaces, Surface{
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
	if len(f.Surfaces) == 0 {
		fallback := fallbackFacts(f.Scenario, f.TargetKind, f.RootPath)
		fallback.DegradedReason = "Code Facts returned no surfaces"
		return fallback
	}
	return f
}

func fallbackFacts(scenarioName, targetKind, rootPath string) Facts {
	f := Facts{Scenario: scenarioName, TargetKind: targetKind, RootPath: rootPath}
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
			f.Surfaces = append(f.Surfaces, Surface{
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
	return f
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
	case hasPythonIndicators(root):
		return "python"
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
	case strings.Contains(text, `"vue"`):
		return "vue"
	case strings.Contains(text, `"svelte"`):
		return "svelte"
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

func hasPythonIndicators(root string) bool {
	for _, name := range []string{"pyproject.toml", "setup.py", "setup.cfg", "requirements.txt"} {
		if exists(filepath.Join(root, name)) {
			return true
		}
	}
	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
