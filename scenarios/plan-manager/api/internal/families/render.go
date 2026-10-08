package families

import (
	"context"
	"fmt"
	"strings"

	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
)

// Render returns a deterministic markdown projection. The stored aggregate is
// authoritative and the projection can always be regenerated.
func (s *Service) Render(ctx context.Context, id string) (*familiesv1.PlanFamily, string, error) {
	family, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, "", err
	}
	return family, RenderMarkdown(family), nil
}

func RenderMarkdown(f *familiesv1.PlanFamily) string {
	if f == nil {
		return ""
	}
	var out strings.Builder
	fmt.Fprintf(&out, "# Plan family: %s\n\n", f.GetSlug())
	fmt.Fprintf(&out, "Family ID: `%s`  \nRevision: `%d`  \nExecution state: `%s`\n\n", f.GetFamilyId(), f.GetRevision(), f.GetExecutionState().String())
	fmt.Fprintf(&out, "## Outcome\n\n%s\n\n## Shared context\n\n%s\n\n", f.GetOutcome(), f.GetSharedContext())
	out.WriteString("## Members\n\n| Plan | Role | State | Execution | Detail |\n| --- | --- | --- | --- | --- |\n")
	for _, member := range f.GetMembers() {
		fmt.Fprintf(&out, "| `%s` | %s | %s | `%s` | %s |\n", cell(member.GetPlanId()), member.GetRole().String(), member.GetState().String(), cell(member.GetExecutionId()), cell(member.GetDetail()))
	}
	out.WriteString("\n## Resource claims\n\n| Claim | Plan | Kind | Access | Resource | Resolved |\n| --- | --- | --- | --- | --- | --- |\n")
	for _, claim := range f.GetClaims() {
		fmt.Fprintf(&out, "| `%s` | `%s` | %s | %s | `%s` | %t |\n", cell(claim.GetClaimId()), cell(claim.GetPlanId()), claim.GetKind().String(), claim.GetAccess().String(), cell(claim.GetResource()), claim.GetResolved())
	}
	out.WriteString("\n## Proposed graph\n\n")
	if f.GetGraph() == nil {
		out.WriteString("No graph proposal.\n")
	} else {
		fmt.Fprintf(&out, "Graph revision: `%d`; cyclic: `%t`\n\n", f.GetGraph().GetRevision(), f.GetGraph().GetCyclic())
		for _, edge := range f.GetGraph().GetEdges() {
			fmt.Fprintf(&out, "- `%s` → `%s` — %s, %s: %s\n", cell(edge.GetFromPlanId()), cell(edge.GetToPlanId()), edge.GetKind().String(), edge.GetProvenance().String(), cell(edge.GetReason()))
		}
	}
	out.WriteString("\n## Review\n\n")
	if f.GetReview() == nil {
		out.WriteString("Unreviewed. This family is not launchable.\n")
	} else {
		fmt.Fprintf(&out, "%s by **%s** for graph revision `%d`: %s\n", f.GetReview().GetDecision().String(), cell(f.GetReview().GetReviewer()), f.GetReview().GetGraphRevision(), cell(f.GetReview().GetRationale()))
	}
	out.WriteString("\n## Proposed frontier\n\n")
	if f.GetGraph() != nil {
		for _, batch := range f.GetGraph().GetProposedFrontier() {
			fmt.Fprintf(&out, "%d. %s\n", batch.GetOrdinal(), strings.Join(batch.GetPlanIds(), ", "))
		}
	}
	return out.String()
}

func cell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "|", "\\|"), "\n", " ")
}
