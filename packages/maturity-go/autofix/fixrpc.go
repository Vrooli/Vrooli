package autofix

import (
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// CandidatesToProto maps shared autofix Candidates to the shared
// scenario-validation FixCandidate wire shape. It is the single conversion used
// by every provider built on this registry so the Fix RPC stays consistent.
func CandidatesToProto(candidates []Candidate) []*scenariovalidationv1.FixCandidate {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]*scenariovalidationv1.FixCandidate, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, &scenariovalidationv1.FixCandidate{
			RuleId:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Before:      c.Before,
			After:       c.After,
			Applied:     c.Applied,
		})
	}
	return out
}

// CandidatesFromProto maps shared scenario-validation FixCandidates back to the
// shared autofix Candidate shape so consumers (e.g. test-genie's deterministic
// aggregate) can reuse a single in-memory representation across providers.
func CandidatesFromProto(candidates []*scenariovalidationv1.FixCandidate) []Candidate {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if c == nil {
			continue
		}
		out = append(out, Candidate{
			RuleID:      c.GetRuleId(),
			FilePath:    c.GetFilePath(),
			Description: c.GetDescription(),
			Before:      c.GetBefore(),
			After:       c.GetAfter(),
			Applied:     c.GetApplied(),
		})
	}
	return out
}

// NoFixesMessage is the canonical note returned when a provider had no
// deterministic remediations for the request. Centralizing it keeps the
// dry-run/apply reports uniform across providers and consumers.
const NoFixesMessage = "No auto-fixable findings are available."

// BuildFixResponse assembles a shared FixResponse from a provider's candidates,
// stamping the canonical empty-set message when nothing was fixable. Both
// PreviewFixResponse and ApplyFixResponse funnel through it.
func BuildFixResponse(scenario string, applied bool, candidates []Candidate) *scenariovalidationv1.FixResponse {
	resp := &scenariovalidationv1.FixResponse{
		Scenario:   scenario,
		Applied:    applied,
		Candidates: CandidatesToProto(candidates),
	}
	if len(candidates) == 0 {
		resp.Messages = append(resp.Messages, NoFixesMessage)
	}
	return resp
}

// PreviewFixResponse previews the registry's remediations for root (restricted
// to ruleIDs when non-empty) and returns the shared FixResponse. scenario is
// echoed back into the response for the caller's report; root is the resolved
// scenario directory the fixers operate on.
func (r *Registry) PreviewFixResponse(scenario, root string, ruleIDs []string) (*scenariovalidationv1.FixResponse, error) {
	candidates, err := r.Preview(root, ruleIDs)
	if err != nil {
		return nil, err
	}
	return BuildFixResponse(scenario, false, candidates), nil
}

// ApplyFixResponse applies the registry's remediations for root (restricted to
// ruleIDs when non-empty) and returns the shared FixResponse with applied=true.
func (r *Registry) ApplyFixResponse(scenario, root string, ruleIDs []string) (*scenariovalidationv1.FixResponse, error) {
	candidates, err := r.Apply(root, ruleIDs)
	if err != nil {
		return nil, err
	}
	return BuildFixResponse(scenario, true, candidates), nil
}
