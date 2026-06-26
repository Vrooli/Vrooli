package validation

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	planmodel "plan-manager/internal/planmodel"
	internalvalidation "plan-manager/internal/validation"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

const (
	codeFactsScenarioID     = "code-facts"
	defaultCodeFactsTimeout = 3 * time.Second
)

// scenarioURLResolver is the api-core/discovery surface used by the production
// code-facts adapter. Tests can replace it without touching global discovery.
type scenarioURLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// codeFactsReferenceResolver enriches validation reference resolution with the
// real code-facts Connect service while preserving FileResolver as the explicit
// honest floor. code-facts does not currently expose a generic "resolve ref"
// RPC, so this adapter maps CODE/DOC references onto DescribeCodeFacts(PATH)
// and leaves unsupported shapes to the floor instead of fabricating certainty.
type codeFactsReferenceResolver struct {
	fallback   internalvalidation.FileResolver
	repoRoot   string
	resolver   scenarioURLResolver
	httpClient connect.HTTPClient
	timeout    time.Duration
}

var _ internalvalidation.ReferenceResolver = codeFactsReferenceResolver{}

func newCodeFactsReferenceResolver(root string) codeFactsReferenceResolver {
	floor := internalvalidation.NewFileResolver(root)
	return codeFactsReferenceResolver{
		fallback: floor,
		repoRoot: root,
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
	}
}

func (r codeFactsReferenceResolver) Resolve(ctx context.Context, ref planmodel.Reference) (planmodel.Reference, error) {
	floor, floorErr := r.fallback.Resolve(ctx, ref)
	if ref.Future {
		return floor, floorErr
	}
	if ref.Kind != planmodel.ReferenceCode && ref.Kind != planmodel.ReferenceDoc {
		if floor.Note == "" {
			floor.Note = "code-facts has no generic resolver for this reference kind"
		}
		return floor, floorErr
	}
	enriched, err := r.resolveWithCodeFacts(ctx, floor)
	if err != nil {
		if floor.Note == "" {
			floor.Note = "code-facts unavailable; filesystem floor used: " + err.Error()
		}
		return floor, nil
	}
	return enriched, nil
}

func (r codeFactsReferenceResolver) resolveWithCodeFacts(ctx context.Context, ref planmodel.Reference) (planmodel.Reference, error) {
	resolver := r.resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := r.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	timeout := r.timeout
	if timeout <= 0 {
		timeout = defaultCodeFactsTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	baseURL, err := resolver.ResolveScenarioURLDefault(callCtx, codeFactsScenarioID)
	if err != nil {
		return ref, fmt.Errorf("resolve code-facts URL: %w", err)
	}
	targetPath := r.absoluteTarget(ref.Target)
	resp, err := factsconnect.NewCodeFactsServiceClient(httpClient, baseURL).DescribeCodeFacts(callCtx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_PATH,
			Path:     targetPath,
			RepoRoot: r.repoRoot,
		},
		Include: []factsv1.FactFamily{
			factsv1.FactFamily_FACT_FAMILY_SURFACES,
			factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS,
		},
		UseCache: true,
	}))
	if err != nil {
		return ref, fmt.Errorf("describe code facts: %w", err)
	}
	if hasMissingEvidence(resp.Msg) {
		ref.Resolution = planmodel.ResolutionMissing
		ref.Note = "code-facts reported missing target evidence"
		return ref, nil
	}
	if hasProvenEvidence(resp.Msg) || resp.Msg.GetTarget().GetRootPath() != "" {
		ref.Resolution = planmodel.ResolutionResolved
		ref.Note = "resolved by code-facts"
		return ref, nil
	}
	ref.Resolution = planmodel.ResolutionUnresolved
	ref.Note = "code-facts returned no resolving evidence"
	return ref, nil
}

func (r codeFactsReferenceResolver) absoluteTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" || filepath.IsAbs(target) || r.repoRoot == "" {
		return target
	}
	return filepath.Join(r.repoRoot, target)
}

func hasProvenEvidence(report *factsv1.CodeFactsReport) bool {
	if report == nil {
		return false
	}
	for _, surface := range report.GetSurfaces() {
		if surface.GetStatus() == factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN {
			return true
		}
		for _, ev := range surface.GetEvidence() {
			if ev.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN {
				return true
			}
		}
	}
	for _, unit := range report.GetParseUnits() {
		if unit.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN {
			return true
		}
		for _, ev := range unit.GetEvidence() {
			if ev.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN {
				return true
			}
		}
	}
	for _, ev := range report.GetEvidence() {
		if ev.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN {
			return true
		}
	}
	return false
}

func hasMissingEvidence(report *factsv1.CodeFactsReport) bool {
	if report == nil {
		return false
	}
	for _, surface := range report.GetSurfaces() {
		if surface.GetStatus() == factsv1.SurfaceStatus_SURFACE_STATUS_MISSING {
			return true
		}
	}
	for _, ev := range report.GetEvidence() {
		if ev.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING {
			return true
		}
	}
	return false
}
