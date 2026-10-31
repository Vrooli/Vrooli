package authoring

import (
	"context"
	"fmt"
	"net/http"

	internalauthoring "plan-manager/internal/authoring"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	baselinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

// gctSourceEvidenceAdvisor lives at the composition edge so authoring stays
// independent of GCT's generated client. It is deliberately an advisory seam:
// callers surface its result but retain deterministic readiness authority.
type gctSourceEvidenceAdvisor struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	http connect.HTTPClient
}

func newGCTSourceEvidenceAdvisor() internalauthoring.SourceEvidenceAdvisor {
	return gctSourceEvidenceAdvisor{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: http.DefaultClient}
}

func (c gctSourceEvidenceAdvisor) AdviseSourceEvidence(ctx context.Context, repoPaths []string) (internalauthoring.SourceEvidenceAdvisory, error) {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, "git-control-tower")
	if err != nil {
		return internalauthoring.SourceEvidenceAdvisory{}, fmt.Errorf("resolve git-control-tower URL: %w", err)
	}
	client := baselinesconnect.NewBaselinesServiceClient(c.http, baseURL)
	resp, err := client.EstimatePathSnapshot(ctx, connect.NewRequest(&baselinesv1.EstimatePathSnapshotRequest{Selections: repoPaths}))
	if err != nil {
		return internalauthoring.SourceEvidenceAdvisory{}, fmt.Errorf("estimate git-control-tower source evidence: %w", err)
	}
	estimate := resp.Msg.GetEstimate()
	if estimate == nil {
		return internalauthoring.SourceEvidenceAdvisory{}, fmt.Errorf("git-control-tower returned no source-evidence estimate")
	}
	out := internalauthoring.SourceEvidenceAdvisory{
		EligibleFiles:  int(estimate.GetEligibleFiles()),
		EligibleBytes:  estimate.GetEligibleBytes(),
		RepairRequired: estimate.GetRepairRequired(),
	}
	for _, issue := range estimate.GetIssues() {
		out.Issues = append(out.Issues, internalauthoring.SourceEvidenceIssue{Code: issue.GetCode(), Severity: issue.GetSeverity(), Detail: issue.GetDetail()})
	}
	for _, recommendation := range estimate.GetRecommendations() {
		out.Recommendations = append(out.Recommendations, internalauthoring.SourceEvidenceRecommendation{Selection: recommendation.GetSelection(), Reason: recommendation.GetReason()})
	}
	return out, nil
}
