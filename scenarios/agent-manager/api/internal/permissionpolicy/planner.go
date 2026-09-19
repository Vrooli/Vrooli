package permissionpolicy

import (
	"context"
	"errors"
	"sort"

	"agent-manager/internal/domain"
)

var defaultRunners = []domain.RunnerType{
	domain.RunnerTypeClaudeCode,
	domain.RunnerTypeCodex,
	domain.RunnerTypeGrok,
	domain.RunnerTypeOpenCode,
}

// AggregatePlanner produces deterministic, non-mutating evidence for every
// supported resource and declared target scope. It deliberately continues
// after individual resource failures so operators can see partial drift.
type AggregatePlanner struct {
	projector Projector
	runners   []domain.RunnerType
}

func NewAggregatePlanner(projector Projector) *AggregatePlanner {
	return newAggregatePlanner(projector, defaultRunners)
}

func newAggregatePlanner(projector Projector, runners []domain.RunnerType) *AggregatePlanner {
	return &AggregatePlanner{projector: projector, runners: append([]domain.RunnerType(nil), runners...)}
}

type AggregatePlan struct {
	CatalogDigest                 string         `json:"catalogDigest"`
	Resources                     []ResourcePlan `json:"resources"`
	HardEnforcementSatisfied      bool           `json:"hardEnforcementSatisfied"`
	MissingHardEnforcementRuleIDs []string       `json:"missingHardEnforcementRuleIds,omitempty"`
}

// ResourcePlan is one resource/scope observation. Installed is truthful only
// for this attempted CLI operation: unavailable means no usable resource CLI
// answered, while failed means one answered but its evidence was not usable.
type ResourcePlan struct {
	Runner              domain.RunnerType  `json:"runner"`
	Scope               string             `json:"scope"`
	Installed           bool               `json:"installed"`
	Status              string             `json:"status"`
	Error               string             `json:"error,omitempty"`
	DesiredDigest       string             `json:"desiredDigest,omitempty"`
	DesiredFingerprint  string             `json:"desiredFingerprint,omitempty"`
	LiveFingerprint     string             `json:"liveFingerprint,omitempty"`
	Drift               bool               `json:"drift"`
	Changes             []string           `json:"changes"`
	NativePaths         []string           `json:"nativePaths"`
	Enforcement         EnforcementPosture `json:"enforcement,omitempty"`
	UnsupportedMatchers []Matcher          `json:"unsupportedMatchers"`
}

func (p *AggregatePlanner) Plan(ctx context.Context, revision *Revision) (AggregatePlan, error) {
	if revision == nil || revision.Catalog() == nil {
		return AggregatePlan{}, ErrInvalidResourceResponse
	}
	if p == nil || p.projector == nil {
		return AggregatePlan{}, ErrResourceUnavailable
	}
	catalog := revision.Catalog()
	if err := catalog.Validate(); err != nil {
		return AggregatePlan{}, err
	}

	runners := append([]domain.RunnerType(nil), p.runners...)
	sort.Slice(runners, func(i, j int) bool { return runners[i] < runners[j] })
	result := AggregatePlan{CatalogDigest: revision.Digest(), HardEnforcementSatisfied: true}
	enforcingScopes := make(map[string]bool, len(catalog.Scopes()))
	for _, scope := range catalog.Scopes() {
		document, err := catalog.ResourceDocument(scope)
		if err != nil {
			return AggregatePlan{}, err
		}
		for _, runner := range runners {
			entry := ResourcePlan{Runner: runner, Scope: scope, Changes: []string{}, NativePaths: []string{}, UnsupportedMatchers: []Matcher{}}
			projection, err := p.projector.Plan(ctx, ProjectionRequest{Runner: runner, Document: document})
			if err != nil {
				entry.Error = err.Error()
				if errors.Is(err, ErrResourceUnavailable) {
					entry.Status = "unavailable"
				} else {
					entry.Status = "failed"
					entry.Installed = true
				}
				result.Resources = append(result.Resources, entry)
				continue
			}
			entry.Installed = true
			entry.Status = "planned"
			entry.DesiredDigest = projection.DesiredDigest
			entry.DesiredFingerprint = projection.DesiredFingerprint
			entry.LiveFingerprint = projection.LiveFingerprint
			entry.Drift = projection.Drift
			entry.Changes = append([]string(nil), projection.Changes...)
			entry.NativePaths = append([]string(nil), projection.NativePaths...)
			entry.Enforcement = projection.Enforcement
			if projection.Enforcement.Permissions == "native" || projection.Enforcement.Permissions == "hook_backed" {
				enforcingScopes[scope] = true
			}
			result.Resources = append(result.Resources, entry)
		}
	}
	for _, rule := range catalog.Rules {
		if rule.RequiresHardEnforcement && !enforcingScopes[rule.TargetScope] {
			result.HardEnforcementSatisfied = false
			result.MissingHardEnforcementRuleIDs = append(result.MissingHardEnforcementRuleIDs, rule.ID)
		}
	}
	sort.Strings(result.MissingHardEnforcementRuleIDs)
	return result, nil
}
