package intentalignment

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"

	intent "intent-go"
)

type Detector struct {
	matcher Matcher
}

func New() *Detector {
	return NewWithMatcher(LexicalMatcher{})
}

func NewWithMatcher(matcher Matcher) *Detector {
	return &Detector{matcher: matcher}
}

func (Detector) Name() string { return "intent_alignment" }

func (Detector) Description() string {
	return "Checks PRD and requirement intent claims against the derived domain map."
}

func (Detector) EmitsTypes() []string {
	return []string{
		intent.CodeReqUnownedDomain,
		intent.CodeReqTransportOwned,
		intent.CodeDomainUnrequired,
		intent.CodeOTNoDomain,
		intent.CodeVocabDrift,
	}
}

func (Detector) Class() conflicts.FindingClass {
	return conflicts.FindingClassDeterministic
}

func (d Detector) Detect(ctx context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	if in.ClaimProvider == nil {
		return nil, nil
	}
	claims, err := in.ClaimProvider.Claims(ctx, in.Scenario)
	if err != nil {
		return nil, err
	}
	outcomes, requirements := splitClaims(claims)
	domainRequired := map[string]struct{}{}
	outcomeHasRequirement := map[string]struct{}{}
	outcomeHasDomain := map[string]struct{}{}
	requirementsByOutcome := map[string][]intent.CapabilityClaim{}
	requirementsByDomain := map[string][]intent.CapabilityClaim{}
	var out []conflicts.Conflict

	sort.Slice(requirements, func(i, j int) bool { return requirements[i].ID < requirements[j].ID })
	for _, req := range requirements {
		prdRef := intent.RequirementPRDRef(req)
		if prdRef != "" {
			outcomeHasRequirement[prdRef] = struct{}{}
			requirementsByOutcome[prdRef] = append(requirementsByOutcome[prdRef], req)
		}
		for _, ref := range req.Refs {
			if ref.Kind != intent.RefCode || ref.Path == "" {
				continue
			}
			switch domain := domainForRef(in.DomainMap, ref.Path); {
			case domain != "":
				domainRequired[domain] = struct{}{}
				requirementsByDomain[domain] = append(requirementsByDomain[domain], req)
				if prdRef != "" {
					outcomeHasDomain[prdRef] = struct{}{}
				}
			case in.DomainMap.IsSharedSubstrate(ref.Path):
				out = append(out, transportOwned(d.Name(), in.Scenario, req, ref))
			default:
				out = append(out, unownedDomain(d.Name(), in.Scenario, req, ref))
			}
		}
	}

	matches, err := runMatcher(ctx, d.matcher, outcomes, requirementsByOutcome, requirementsByDomain, in.DomainMap)
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		if match.Type != MatchVocabDrift {
			continue
		}
		out = append(out, vocabDrift(d.Name(), in.Scenario, match))
	}

	for _, domain := range in.DomainMap.Domains {
		if _, ok := domainRequired[domain.Name]; ok {
			continue
		}
		out = append(out, domainUnrequired(d.Name(), in.Scenario, domain))
	}

	sort.Slice(outcomes, func(i, j int) bool { return outcomes[i].ID < outcomes[j].ID })
	for _, outcome := range outcomes {
		if _, hasReq := outcomeHasRequirement[strings.ToUpper(outcome.ID)]; !hasReq {
			continue
		}
		if _, hasDomain := outcomeHasDomain[strings.ToUpper(outcome.ID)]; hasDomain {
			continue
		}
		out = append(out, outcomeNoDomain(d.Name(), in.Scenario, outcome))
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return strings.Join(out[i].Locations, "\x00") < strings.Join(out[j].Locations, "\x00")
	})
	return out, nil
}

func runMatcher(
	ctx context.Context,
	matcher Matcher,
	outcomes []intent.CapabilityClaim,
	requirementsByOutcome map[string][]intent.CapabilityClaim,
	requirementsByDomain map[string][]intent.CapabilityClaim,
	domainMap domains.DerivedDomainMap,
) ([]Match, error) {
	if matcher == nil {
		return nil, nil
	}
	return matcher.Match(ctx, MatchInput{
		Outcomes:              outcomes,
		RequirementsByOutcome: requirementsByOutcome,
		RequirementsByDomain:  requirementsByDomain,
		Domains:               domainClaims(domainMap),
	})
}

func splitClaims(claims []intent.CapabilityClaim) (outcomes, requirements []intent.CapabilityClaim) {
	for _, claim := range claims {
		switch claim.Altitude {
		case intent.Outcome:
			claim.ID = strings.ToUpper(strings.TrimSpace(claim.ID))
			outcomes = append(outcomes, claim)
		case intent.Requirement:
			requirements = append(requirements, claim)
		}
	}
	return outcomes, requirements
}

func domainForRef(m domains.DerivedDomainMap, path string) string {
	return m.DomainFor(strings.TrimSpace(path))
}

func domainClaims(m domains.DerivedDomainMap) []intent.CapabilityClaim {
	claims := make([]intent.CapabilityClaim, 0, len(m.Domains))
	for _, domain := range m.Domains {
		claims = append(claims, intent.CapabilityClaim{
			ID:         domain.Name,
			Altitude:   intent.Domain,
			Text:       strings.Join(append([]string{domain.Name}, domain.Glossary...), " "),
			Anchor:     strings.Join(domain.Paths, ", "),
			Provenance: "domains",
		})
	}
	return claims
}

func unownedDomain(detector, scenario string, req intent.CapabilityClaim, ref intent.Ref) conflicts.Conflict {
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  detector,
		Type:      intent.CodeReqUnownedDomain,
		Subtype:   req.ID,
		Severity:  conflicts.SeverityError,
		Locations: compactLocations(req.Anchor, ref.Path),
		Evidence: []conflicts.Evidence{{
			Kind:    "intent.requirement_ref_domain_join",
			Summary: "Requirement " + req.ID + " validates " + ref.Path + ", but no product domain owns that path.",
			Locator: ref.Path,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Add the path to the owning domain's Source Paths, or move validation to domain-owned code.",
			Confidence: 0.85,
		}},
	}
}

func transportOwned(detector, scenario string, req intent.CapabilityClaim, ref intent.Ref) conflicts.Conflict {
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  detector,
		Type:      intent.CodeReqTransportOwned,
		Subtype:   req.ID,
		Severity:  conflicts.SeverityInfo,
		Locations: compactLocations(req.Anchor, ref.Path),
		Evidence: []conflicts.Evidence{{
			Kind:    "intent.requirement_ref_non_domain",
			Summary: "Requirement " + req.ID + " validates " + ref.Path + ", which is declared as shared substrate or a non-domain transport zone.",
			Locator: ref.Path,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Keep transport validations as support evidence, but add domain-owned validation when this requirement should prove product behavior.",
			Confidence: 0.65,
		}},
	}
}

func domainUnrequired(detector, scenario string, domain domains.DerivedDomain) conflicts.Conflict {
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  detector,
		Type:      intent.CodeDomainUnrequired,
		Subtype:   domain.Name,
		Severity:  conflicts.SeverityWarn,
		Domains:   []string{domain.Name},
		Locations: append([]string(nil), domain.Paths...),
		Evidence: []conflicts.Evidence{{
			Kind:    "intent.domain_requirement_coverage",
			Summary: "Domain " + domain.Name + " owns source paths, but no requirement validation points into them.",
			Locator: domain.Name,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Connect at least one requirement validation to this domain, or remove the stale domain declaration.",
			Confidence: 0.75,
		}},
	}
}

func outcomeNoDomain(detector, scenario string, outcome intent.CapabilityClaim) conflicts.Conflict {
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  detector,
		Type:      intent.CodeOTNoDomain,
		Subtype:   outcome.ID,
		Severity:  conflicts.SeverityWarn,
		Locations: compactLocations(outcome.Anchor),
		Evidence: []conflicts.Evidence{{
			Kind:    "intent.outcome_domain_reachability",
			Summary: "Operational target " + outcome.ID + " has requirements, but none reaches a product domain through validation refs.",
			Locator: outcome.Anchor,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Add domain-owned validation for a requirement that points at " + outcome.ID + ".",
			Confidence: 0.7,
		}},
	}
}

func vocabDrift(detector, scenario string, match Match) conflicts.Conflict {
	domain := match.Domain.ID
	token := strings.Join(match.Tokens, ", ")
	if token == "" {
		token = "glossary"
	}
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  detector,
		Type:      intent.CodeVocabDrift,
		Subtype:   domain,
		Severity:  conflicts.SeverityWarn,
		Domains:   compactLocations(domain),
		Locations: compactLocations(match.Domain.Anchor),
		Evidence: []conflicts.Evidence{{
			Kind:    "intent.lexical_vocabulary_alignment",
			Summary: fmt.Sprintf("Domain %s declares vocabulary %q that is absent from the outcome and requirement text pointing at that domain.", domain, token),
			Locator: match.Domain.Anchor,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Align the domain glossary with PRD and requirement wording, or update the requirements that prove this domain's responsibility.",
			Confidence: 0.6,
		}},
	}
}

func compactLocations(values ...string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
