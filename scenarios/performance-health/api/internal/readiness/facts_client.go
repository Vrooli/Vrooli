package readiness

import (
	"context"
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

// CodeFactsClient is the production FactsClient: it resolves the target root,
// asks Code Facts for the scenario's surfaces, and derives the UI framework from
// the filesystem (Code Facts surfaces carry no framework). When Code Facts is
// unavailable or returns nothing, it degrades to a pure filesystem scan and
// records an explicit DegradedReason — readiness never silently invents facts.
//
// React is NEVER assumed: the framework comes from package.json dependencies
// (react + vite ⇒ "react-vite"; react alone ⇒ "react"), so a non-React UI is
// correctly held at Tier 0.
type CodeFactsClient struct {
	RepoRoot string

	// Seams for tests; nil values fall back to live implementations.
	Resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	HTTPClient connect.HTTPClient
}

// NewCodeFactsClient builds a CodeFactsClient rooted at repoRoot.
func NewCodeFactsClient(repoRoot string) *CodeFactsClient {
	return &CodeFactsClient{RepoRoot: repoRoot}
}

var _ FactsClient = (*CodeFactsClient)(nil)

// Describe resolves the target and returns its surfaces + UI framework + root.
func (c *CodeFactsClient) Describe(ctx context.Context, scenario, path string) (Facts, error) {
	scenarioName, root, err := c.locate(scenario, path)
	if err != nil {
		return Facts{}, err
	}

	surfaces, degraded := c.surfacesFromCodeFacts(ctx, scenarioName, root)
	if len(surfaces) == 0 {
		surfaces, degraded = fallbackSurfaces(root), joinDegraded(degraded, "Code Facts returned no surfaces; used filesystem scan")
	}

	return Facts{
		Scenario:       scenarioName,
		Surfaces:       surfaces,
		UIFramework:    frameworkFromFilesystem(root),
		RootPath:       root,
		DegradedReason: degraded,
	}, nil
}

// locate resolves scenario/path into (name, absolute root).
func (c *CodeFactsClient) locate(scenario, path string) (string, string, error) {
	scenario = strings.TrimSpace(scenario)
	path = strings.TrimSpace(path)
	if path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", "", err
		}
		name := scenario
		if name == "" {
			name = filepath.Base(abs)
		}
		return name, abs, nil
	}
	if scenario == "" {
		return "", "", fmt.Errorf("scenario or path is required")
	}
	repoRoot := strings.TrimSpace(c.RepoRoot)
	if repoRoot == "" {
		resolved, err := repocontract.ResolveRepoRoot()
		if err != nil {
			return "", "", err
		}
		repoRoot = resolved
	}
	root, err := repocontract.ResolveScenarioPath(repoRoot, scenario)
	if err != nil {
		return "", "", err
	}
	return scenario, root, nil
}

// surfacesFromCodeFacts asks Code Facts for the scenario's surface ids. On any
// failure it returns nil surfaces and a degraded reason so the caller can fall
// back to a filesystem scan.
func (c *CodeFactsClient) surfacesFromCodeFacts(ctx context.Context, scenario, root string) ([]string, string) {
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
		return nil, fmt.Sprintf("Code Facts unavailable: %v", err)
	}
	client := factsconnect.NewCodeFactsServiceClient(httpClient, baseURL)
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
			Scenario: scenario,
			Path:     root,
		},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES},
	}))
	if err != nil {
		return nil, fmt.Sprintf("Code Facts unavailable: %v", err)
	}
	var surfaces []string
	for _, s := range resp.Msg.GetSurfaces() {
		kind := strings.TrimPrefix(strings.ToLower(s.GetKind().String()), "surface_kind_")
		if kind != "" && kind != "unspecified" {
			surfaces = append(surfaces, kind)
		}
	}
	return surfaces, ""
}

// fallbackSurfaces scans the conventional surface directories under root.
func fallbackSurfaces(root string) []string {
	var surfaces []string
	for _, dir := range []string{"api", "cli", "ui"} {
		if isDir(filepath.Join(root, dir)) {
			surfaces = append(surfaces, dir)
		}
	}
	return surfaces
}

// frameworkFromFilesystem derives the UI framework from the UI surface's
// package.json. React is only reported when react is an actual dependency.
func frameworkFromFilesystem(root string) string {
	pkgPath := filepath.Join(root, "ui", "package.json")
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		// No conventional ui/package.json: try the scenario root itself.
		raw, err = os.ReadFile(filepath.Join(root, "package.json"))
		if err != nil {
			return ""
		}
	}
	text := string(raw)
	hasReact := strings.Contains(text, `"react"`)
	hasVite := strings.Contains(text, `"vite"`)
	switch {
	case hasReact && hasVite:
		return "react-vite"
	case hasReact:
		return "react"
	case hasVite:
		return "vite"
	case strings.TrimSpace(text) != "":
		return "node"
	default:
		return ""
	}
}

func joinDegraded(existing, addition string) string {
	existing = strings.TrimSpace(existing)
	addition = strings.TrimSpace(addition)
	switch {
	case existing == "":
		return addition
	case addition == "":
		return existing
	default:
		return existing + "; " + addition
	}
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
