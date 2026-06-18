package domains

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	repocontract "github.com/vrooli/repo-contract-go"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

const codeFactsScenario = "code-facts"

type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// CodeFactsSurfaceProvider consumes code-facts' surface and parse-unit
// substrate. If code-facts is unreachable, it returns a local inventory plus
// a warning so downstream audit output can mark the run partial/lower
// confidence instead of silently deriving from heuristics.
type CodeFactsSurfaceProvider struct {
	resolver   URLResolver
	httpClient connect.HTTPClient
	fallback   SurfaceProvider
}

func NewCodeFactsSurfaceProvider(resolver URLResolver, httpClient connect.HTTPClient, fallback SurfaceProvider) *CodeFactsSurfaceProvider {
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if fallback == nil {
		fallback = NewLocalSurfaceProvider()
	}
	return &CodeFactsSurfaceProvider{resolver: resolver, httpClient: httpClient, fallback: fallback}
}

var _ SurfaceProvider = (*CodeFactsSurfaceProvider)(nil)

func (p *CodeFactsSurfaceProvider) Inspect(ctx context.Context, scenarioDir string) (SurfaceInventory, error) {
	baseURL, err := p.resolver.ResolveScenarioURLDefault(ctx, codeFactsScenario)
	if err != nil {
		return p.fallbackWithWarning(ctx, scenarioDir, fmt.Errorf("resolve code-facts: %w", err))
	}
	client := factsconnect.NewCodeFactsServiceClient(p.httpClient, baseURL)
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: codeFactsTarget(scenarioDir),
		Include: []factsv1.FactFamily{
			factsv1.FactFamily_FACT_FAMILY_SURFACES,
			factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS,
		},
		UseCache: true,
	}))
	if err != nil {
		return p.fallbackWithWarning(ctx, scenarioDir, fmt.Errorf("describe code facts: %w", err))
	}
	return inventoryFromCodeFacts(resp.Msg), nil
}

func codeFactsTarget(scenarioDir string) *factsv1.CodeTarget {
	repoRoot, err := repocontract.FindRepoRootFromPath(scenarioDir)
	if err != nil {
		return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: scenarioDir}
	}
	if rel, err := filepath.Rel(filepath.Join(repoRoot, "scenarios"), scenarioDir); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > 0 && parts[0] != "" {
			return &factsv1.CodeTarget{
				Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
				Scenario: parts[0],
				RepoRoot: repoRoot,
			}
		}
	}
	return &factsv1.CodeTarget{
		Kind:     factsv1.TargetKind_TARGET_KIND_PATH,
		Path:     scenarioDir,
		RepoRoot: repoRoot,
	}
}

func (p *CodeFactsSurfaceProvider) fallbackWithWarning(ctx context.Context, scenarioDir string, cause error) (SurfaceInventory, error) {
	inv, err := p.fallback.Inspect(ctx, scenarioDir)
	if err != nil {
		return SurfaceInventory{}, err
	}
	inv.Warnings = append(inv.Warnings, ExtractionWarning{
		Kind:    "code_facts.unavailable",
		Path:    scenarioDir,
		Summary: fmt.Sprintf("code-facts surface inventory unavailable; using local filesystem fallback: %v", cause),
	})
	return inv, nil
}

func inventoryFromCodeFacts(report *factsv1.CodeFactsReport) SurfaceInventory {
	if report == nil {
		return SurfaceInventory{}
	}
	out := SurfaceInventory{
		Surfaces:   make([]Surface, 0, len(report.GetSurfaces())),
		ParseUnits: make([]ParseUnit, 0, len(report.GetParseUnits())),
	}
	for _, surface := range report.GetSurfaces() {
		out.Surfaces = append(out.Surfaces, Surface{
			ID:     surface.GetId(),
			Kind:   surfaceKind(surface.GetKind()),
			Path:   surface.GetPath(),
			Status: surfaceStatusName(surface.GetStatus()),
		})
	}
	for _, unit := range report.GetParseUnits() {
		out.ParseUnits = append(out.ParseUnits, ParseUnit{
			ID:         unit.GetId(),
			Language:   unit.GetLanguage(),
			RootPath:   unit.GetRootPath(),
			ConfigPath: unit.GetConfigPath(),
			Status:     evidenceStatusName(unit.GetStatus()),
		})
	}
	return out
}

func surfaceKind(kind factsv1.SurfaceKind) string {
	switch kind {
	case factsv1.SurfaceKind_SURFACE_KIND_API:
		return "api"
	case factsv1.SurfaceKind_SURFACE_KIND_CLI:
		return "cli"
	case factsv1.SurfaceKind_SURFACE_KIND_UI:
		return "ui"
	case factsv1.SurfaceKind_SURFACE_KIND_SIDECAR:
		return "sidecar"
	case factsv1.SurfaceKind_SURFACE_KIND_WORKER:
		return "worker"
	case factsv1.SurfaceKind_SURFACE_KIND_JOB:
		return "job"
	default:
		return ""
	}
}

func surfaceStatusName(status factsv1.SurfaceStatus) string {
	switch status {
	case factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN:
		return "known"
	case factsv1.SurfaceStatus_SURFACE_STATUS_MISSING:
		return "missing"
	case factsv1.SurfaceStatus_SURFACE_STATUS_UNSUPPORTED:
		return "unsupported"
	default:
		return ""
	}
}

func evidenceStatusName(status factsv1.EvidenceStatus) string {
	switch status {
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN:
		return "proven"
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING:
		return "missing"
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED:
		return "unsupported"
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN:
		return "unknown"
	default:
		return ""
	}
}
