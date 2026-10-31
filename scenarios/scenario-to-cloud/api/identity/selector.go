package identity

import (
	"context"
	"sort"
	"strings"

	"scenario-to-cloud/apierrors"
)

// SelectorKind names the four accepted selector forms.
type SelectorKind string

const (
	SelectByID                  SelectorKind = "id"
	SelectByScenarioEnvironment SelectorKind = "scenario_environment"
	SelectByScenarioDomain      SelectorKind = "scenario_domain"
	SelectByScenarioHost        SelectorKind = "scenario_host"
)

// Selector is a conjunction of identity facets. Exactly one canonical form is
// accepted: by ID; by (scenario, environment); by (scenario, domain); by
// (scenario, host). Scenario name alone is never enough because one scenario
// may be installed many times.
type Selector struct {
	ID          string `json:"id,omitempty"`
	ScenarioID  string `json:"scenario_id,omitempty"`
	Environment string `json:"environment,omitempty"`
	Domain      string `json:"domain,omitempty"`
	Host        string `json:"host,omitempty"`
}

// Normalized trims every facet.
func (s Selector) Normalized() Selector {
	return Selector{
		ID:          strings.TrimSpace(s.ID),
		ScenarioID:  strings.TrimSpace(s.ScenarioID),
		Environment: strings.TrimSpace(s.Environment),
		Domain:      strings.TrimSpace(s.Domain),
		Host:        strings.TrimSpace(s.Host),
	}
}

// Kind classifies the selector or returns a typed deployment_selector_invalid
// error when the facets do not form one of the four canonical shapes.
func (s Selector) Kind() (SelectorKind, error) {
	n := s.Normalized()
	if n.ID != "" {
		if n.ScenarioID != "" || n.Environment != "" || n.Domain != "" || n.Host != "" {
			return "", invalidSelector("id cannot be combined with other facets")
		}
		return SelectByID, nil
	}
	if n.ScenarioID == "" {
		return "", invalidSelector("scenario_id is required unless id is given")
	}
	facets := 0
	var kind SelectorKind
	if n.Environment != "" {
		facets++
		kind = SelectByScenarioEnvironment
	}
	if n.Domain != "" {
		facets++
		kind = SelectByScenarioDomain
	}
	if n.Host != "" {
		facets++
		kind = SelectByScenarioHost
	}
	switch facets {
	case 0:
		return "", invalidSelector("scenario_id must be paired with environment, domain or host")
	case 1:
		return kind, nil
	default:
		return "", invalidSelector("only one of environment, domain or host may accompany scenario_id")
	}
}

func invalidSelector(reason string) *apierrors.Error {
	return apierrors.New(apierrors.CodeDeploymentSelectorInvalid, "Deployment selector is invalid: "+reason).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "document", Reference: "docs/reference/identity-and-selectors.md", Label: "Selector forms"})
}

// Resolver is the narrow persistence seam Resolve needs. The repository
// implements it; tests substitute an in-memory fake.
type Resolver interface {
	ResolveDeployments(ctx context.Context, selector Selector) ([]DeploymentRef, error)
}

// Resolve maps one selector to exactly one deployment. No match is a typed
// deployment_not_found; more than one match is a typed
// deployment_selector_ambiguous that lists every candidate so the caller can
// narrow the selector instead of guessing.
func Resolve(ctx context.Context, repo Resolver, selector Selector) (DeploymentRef, error) {
	if repo == nil {
		return DeploymentRef{}, apierrors.Internal("Deployment resolver is not configured", nil)
	}
	kind, err := selector.Kind()
	if err != nil {
		return DeploymentRef{}, err
	}
	matches, err := repo.ResolveDeployments(ctx, selector.Normalized())
	if err != nil {
		return DeploymentRef{}, apierrors.Internal("Failed to resolve deployment selector", err)
	}
	switch len(matches) {
	case 0:
		return DeploymentRef{}, apierrors.New(apierrors.CodeDeploymentNotFound, "No deployment matches the selector").
			WithDetail("selector", selector.Normalized()).
			WithDetail("selector_kind", string(kind))
	case 1:
		return matches[0], nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].ID < matches[j].ID })
	candidates := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		candidates = append(candidates, map[string]any{
			"id":          m.ID,
			"scenario_id": m.ScenarioID,
			"environment": m.Environment,
			"target_key":  m.Target.Key(),
		})
	}
	return DeploymentRef{}, apierrors.New(apierrors.CodeDeploymentSelectorAmbiguous, "Selector matches more than one deployment; select by id or add environment").
		WithDetail("selector", selector.Normalized()).
		WithDetail("selector_kind", string(kind)).
		WithDetail("candidates", candidates).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "selector", Reference: "id", Label: "Select by deployment id"})
}
