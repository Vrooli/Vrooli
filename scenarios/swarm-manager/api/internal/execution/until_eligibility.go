package execution

import (
	"context"
	"strings"

	"swarm-manager/internal/apierr"
)

// untilDrainEligibility enforces the operator-approval precondition at queue
// time. A warm engagement cannot pause for an operator between irreversible
// phases, so plans that explicitly declare destructive or irreversible work
// stay on the reviewed phased strategy.
func (s *Service) untilDrainEligibility(ctx context.Context, item backlogItem) (string, error) {
	if s.planRenderer == nil {
		return "", apierr.Unavailable("until-drain eligibility cannot be evaluated: plan-manager renderer is unavailable")
	}
	rendered, err := resolveRenderedPlanContent(ctx, item, s.planRenderer)
	if err != nil {
		return "", apierr.BadGateway("until-drain eligibility: %s", err)
	}
	content := strings.ToLower(rendered.Markdown)
	for _, marker := range []string{"irreversible", "irreversibly", "destructive phase", "destructive operation"} {
		if strings.Contains(content, marker) {
			return "plan declares an irreversible or destructive operation", nil
		}
	}
	return "", nil
}
