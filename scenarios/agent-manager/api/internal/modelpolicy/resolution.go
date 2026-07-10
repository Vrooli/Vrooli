package modelpolicy

import (
	"fmt"
	"strings"

	"agent-manager/internal/domain"
)

const (
	ResolutionSourceNamedPolicy   = "named_policy"
	ResolutionSourceDirectModel   = "direct_model"
	ResolutionSourceRunnerDefault = "runner_default"
)

// ResolvePolicy resolves a named policy from the currently active immutable
// revision into a run-owned snapshot.
func (s *State) ResolvePolicy(policyRef string) (*domain.ExecutionPolicySnapshot, error) {
	policyRef = strings.TrimSpace(policyRef)
	return s.resolvePolicy(policyRef, domain.PolicyResolutionExplanation{
		Source:             ResolutionSourceNamedPolicy,
		Summary:            fmt.Sprintf("named policy %q selected from the active catalog", policyRef),
		RequestedPolicyRef: policyRef,
	})
}

// ResolveDirectModel validates an explicit model against the active catalog's
// static inventory or declared dynamic namespace and returns a one-candidate
// snapshot. Runtime availability is checked by orchestration's runner preflight.
func (s *State) ResolveDirectModel(runnerType domain.RunnerType, model string) (*domain.ExecutionPolicySnapshot, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, domain.NewValidationError("model", "field is required")
	}
	revision, catalog, err := s.activeCatalog()
	if err != nil {
		return nil, err
	}
	inventory, ok := catalog.Runners[runnerType]
	if !ok {
		return nil, domain.NewValidationError("runnerType", fmt.Sprintf("runner %q is not declared in the active model policy catalog", runnerType))
	}
	if !inventoryHasModel(inventory, model) && !matchesDynamicPrefix(inventory.DynamicModelPrefixes, model) {
		return nil, domain.NewValidationError("model", fmt.Sprintf("model %q is not declared for runner %q and does not match a dynamic model namespace", model, runnerType))
	}
	candidate := domain.ExecutionCandidate{
		RunnerType:    runnerType,
		SelectionType: domain.ModelSelectionTypeModel,
		Model:         model,
	}
	return newSnapshot(revision.Digest(), "", []domain.ExecutionCandidate{candidate}, domain.PolicyResolutionExplanation{
		Source:          ResolutionSourceDirectModel,
		Summary:         fmt.Sprintf("explicit model %q selected for runner %s", model, runnerType),
		RequestedRunner: runnerType,
		RequestedModel:  model,
	}), nil
}

// ResolveRunnerDefault validates explicit runner-default support and returns a
// one-candidate snapshot without manufacturing an empty model identifier.
func (s *State) ResolveRunnerDefault(runnerType domain.RunnerType) (*domain.ExecutionPolicySnapshot, error) {
	revision, catalog, err := s.activeCatalog()
	if err != nil {
		return nil, err
	}
	inventory, ok := catalog.Runners[runnerType]
	if !ok {
		return nil, domain.NewValidationError("runnerType", fmt.Sprintf("runner %q is not declared in the active model policy catalog", runnerType))
	}
	if !inventory.SupportsRunnerDefault {
		return nil, domain.NewValidationError("runnerType", fmt.Sprintf("runner %q does not support runner_default", runnerType))
	}
	candidate := domain.ExecutionCandidate{
		RunnerType:    runnerType,
		SelectionType: domain.ModelSelectionTypeRunnerDefault,
	}
	return newSnapshot(revision.Digest(), "", []domain.ExecutionCandidate{candidate}, domain.PolicyResolutionExplanation{
		Source:          ResolutionSourceRunnerDefault,
		Summary:         fmt.Sprintf("runner %s default selected explicitly", runnerType),
		RequestedRunner: runnerType,
	}), nil
}

func (s *State) resolvePolicy(policyRef string, explanation domain.PolicyResolutionExplanation) (*domain.ExecutionPolicySnapshot, error) {
	policyRef = strings.TrimSpace(policyRef)
	if policyRef == "" {
		return nil, domain.NewValidationError("policyRef", "field is required")
	}
	revision, catalog, err := s.activeCatalog()
	if err != nil {
		return nil, err
	}
	policy, ok := catalog.Policies[policyRef]
	if !ok {
		return nil, domain.NewValidationError("policyRef", fmt.Sprintf("policy %q is not declared in active catalog %s", policyRef, revision.Digest()))
	}
	candidates := make([]domain.ExecutionCandidate, 0, len(policy.Candidates))
	for _, candidate := range policy.Candidates {
		resolved := domain.ExecutionCandidate{RunnerType: candidate.Runner}
		switch candidate.Selection.Type {
		case SelectionTypeModel:
			resolved.SelectionType = domain.ModelSelectionTypeModel
			resolved.Model = candidate.Selection.Model
		case SelectionTypeRunnerDefault:
			resolved.SelectionType = domain.ModelSelectionTypeRunnerDefault
		default:
			return nil, domain.NewValidationError("policyRef", fmt.Sprintf("policy %q contains unsupported selection %q", policyRef, candidate.Selection.Type))
		}
		candidates = append(candidates, resolved)
	}
	return newSnapshot(revision.Digest(), policyRef, candidates, explanation), nil
}

func (s *State) activeCatalog() (*Revision, *Catalog, error) {
	if s == nil {
		return nil, nil, domain.NewValidationError("modelPolicyCatalog", "state is not configured")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.active == nil || s.active.catalog == nil {
		message := "no validated catalog revision is active"
		if s.lastAttempt != nil && s.lastAttempt.Diagnostic != nil {
			message = s.lastAttempt.Diagnostic.Message
		}
		return nil, nil, domain.NewValidationError("modelPolicyCatalog", message)
	}
	return s.active, s.active.catalog.Clone(), nil
}

func newSnapshot(digest, policyRef string, candidates []domain.ExecutionCandidate, explanation domain.PolicyResolutionExplanation) *domain.ExecutionPolicySnapshot {
	snapshot := &domain.ExecutionPolicySnapshot{
		CatalogDigest: digest,
		PolicyRef:     policyRef,
		Candidates:    append([]domain.ExecutionCandidate(nil), candidates...),
		SelectedIndex: 0,
		Explanation:   explanation,
	}
	if len(candidates) > 0 {
		snapshot.SelectedCandidate = candidates[0]
	}
	return snapshot
}

func matchesDynamicPrefix(prefixes []string, model string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}
