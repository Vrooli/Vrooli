package deliveryramp

import (
	"fmt"
	"sort"
	"strings"
)

// StackFacts describes a scenario's implementation stack without assigning
// semantics to a particular delivery ramp.
type StackFacts struct {
	Components []StackComponent `json:"components"`
	Resources  []string         `json:"resources,omitempty"`
}

type StackComponent struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	BuildKind string `json:"build_kind,omitempty"`
	Directory string `json:"directory,omitempty"`
	Language  string `json:"language,omitempty"`
	Framework string `json:"framework,omitempty"`
	Bundler   string `json:"bundler,omitempty"`
	// EmbeddableServer is a ramp-neutral fact that an API publishes the
	// repository's in-process HTTP handler and lifecycle contract.
	EmbeddableServer bool     `json:"embeddable_server,omitempty"`
	Evidence         []string `json:"evidence,omitempty"`
}

type EligibilityPredicate struct {
	Kind   string `json:"kind"`
	Role   string `json:"role,omitempty"`
	Value  string `json:"value"`
	Negate bool   `json:"negate,omitempty"`
}

type PackagingStrategy struct {
	ID           string                 `json:"id"`
	Version      string                 `json:"version"`
	Ramp         string                 `json:"ramp"`
	Shell        string                 `json:"shell"`
	Topology     string                 `json:"topology"`
	UITransport  string                 `json:"ui_transport"`
	Profile      string                 `json:"profile,omitempty"`
	Platforms    []string               `json:"platforms,omitempty"`
	Predicates   []EligibilityPredicate `json:"predicates,omitempty"`
	Capabilities StrategyCapabilities   `json:"capabilities,omitempty"`
	Default      bool                   `json:"default,omitempty"`
}

// StrategyCapabilities describes prerequisites that apply only after this
// strategy is selected. In particular, host tools must not become scenario
// wide lifecycle requirements merely because a strategy is selectable.
type StrategyCapabilities struct {
	HostTools []string `json:"host_tools,omitempty"`
}

type PredicateResult struct {
	Predicate EligibilityPredicate `json:"predicate"`
	Satisfied bool                 `json:"satisfied"`
	Reason    string               `json:"reason"`
}

type StrategyEligibility struct {
	Strategy PackagingStrategy `json:"strategy"`
	Eligible bool              `json:"eligible"`
	Reasons  []PredicateResult `json:"reasons"`
}

type PackagingStrategyRegistry struct{ strategies map[string]PackagingStrategy }

func NewPackagingStrategyRegistry(strategies []PackagingStrategy) (*PackagingStrategyRegistry, error) {
	r := &PackagingStrategyRegistry{strategies: make(map[string]PackagingStrategy, len(strategies))}
	defaults := 0
	for _, strategy := range strategies {
		strategy.ID = strings.TrimSpace(strategy.ID)
		if strategy.ID == "" || strings.TrimSpace(strategy.Version) == "" || strings.TrimSpace(strategy.Ramp) == "" {
			return nil, fmt.Errorf("strategy id, version, and ramp are required")
		}
		if _, exists := r.strategies[strategy.ID]; exists {
			return nil, fmt.Errorf("strategy %q is duplicated", strategy.ID)
		}
		if strategy.Default {
			defaults++
		}
		strategy.Platforms = append([]string(nil), strategy.Platforms...)
		strategy.Predicates = append([]EligibilityPredicate(nil), strategy.Predicates...)
		strategy.Capabilities.HostTools = append([]string(nil), strategy.Capabilities.HostTools...)
		r.strategies[strategy.ID] = strategy
	}
	if defaults > 1 {
		return nil, fmt.Errorf("only one default strategy is allowed")
	}
	return r, nil
}

func (r *PackagingStrategyRegistry) Resolve(id string) (PackagingStrategy, bool) {
	if r == nil {
		return PackagingStrategy{}, false
	}
	s, ok := r.strategies[strings.TrimSpace(id)]
	s.Platforms = append([]string(nil), s.Platforms...)
	s.Predicates = append([]EligibilityPredicate(nil), s.Predicates...)
	s.Capabilities.HostTools = append([]string(nil), s.Capabilities.HostTools...)
	return s, ok
}

func (r *PackagingStrategyRegistry) List() []PackagingStrategy {
	if r == nil {
		return nil
	}
	out := make([]PackagingStrategy, 0, len(r.strategies))
	for _, s := range r.strategies {
		s.Platforms = append([]string(nil), s.Platforms...)
		s.Predicates = append([]EligibilityPredicate(nil), s.Predicates...)
		s.Capabilities.HostTools = append([]string(nil), s.Capabilities.HostTools...)
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *PackagingStrategyRegistry) Evaluate(facts StackFacts, platform string) []StrategyEligibility {
	strategies := r.List()
	out := make([]StrategyEligibility, 0, len(strategies))
	for _, strategy := range strategies {
		result := StrategyEligibility{Strategy: strategy, Eligible: true}
		if len(strategy.Platforms) > 0 {
			found := false
			for _, supported := range strategy.Platforms {
				if supported == platform {
					found = true
					break
				}
			}
			result.Reasons = append(result.Reasons, PredicateResult{Predicate: EligibilityPredicate{Kind: "platform", Value: platform}, Satisfied: found, Reason: platformReason(platform, found)})
			result.Eligible = result.Eligible && found
		}
		for _, predicate := range strategy.Predicates {
			ok, reason := evaluatePredicate(facts, predicate)
			result.Reasons = append(result.Reasons, PredicateResult{Predicate: predicate, Satisfied: ok, Reason: reason})
			result.Eligible = result.Eligible && ok
		}
		out = append(out, result)
	}
	return out
}

func platformReason(platform string, ok bool) string {
	if ok {
		return "platform is supported"
	}
	return "platform is not supported"
}

func evaluatePredicate(facts StackFacts, p EligibilityPredicate) (bool, string) {
	match := false
	for _, c := range facts.Components {
		if p.Role != "" && !strings.EqualFold(c.Role, p.Role) {
			continue
		}
		var value string
		switch p.Kind {
		case "build.kind":
			value = c.BuildKind
		case "language":
			value = c.Language
		case "framework":
			value = c.Framework
		case "bundler":
			value = c.Bundler
		case "component.role":
			value = c.Role
		case "embeddable.server":
			if c.EmbeddableServer {
				match = true
			}
			continue
		default:
			return false, "unknown predicate kind"
		}
		if strings.EqualFold(value, p.Value) && value != "" {
			match = true
			break
		}
	}
	if p.Negate {
		match = !match
	}
	if match {
		return true, "stack facts satisfy predicate"
	}
	return false, "required stack fact is absent or does not match"
}

type SelectionPolicy struct {
	Version                  string             `json:"version"`
	DefaultStrategy          string             `json:"default_strategy"`
	Weights                  map[string]float64 `json:"weights,omitempty"`
	MinimumLaunchReliability float64            `json:"minimum_launch_reliability,omitempty"`
}

type StrategyEvidence struct {
	StrategyID        string             `json:"strategy_id"`
	Scope             string             `json:"scope"`
	JourneyParity     string             `json:"journey_parity"`
	LaunchReliability float64            `json:"launch_reliability"`
	Metrics           map[string]float64 `json:"metrics,omitempty"`
	EvidenceIDs       []string           `json:"evidence_ids,omitempty"`
}

type PackagingSelectionRequest struct {
	Policy   SelectionPolicy       `json:"policy"`
	Eligible []StrategyEligibility `json:"eligible"`
	Override string                `json:"override,omitempty"`
	Evidence []StrategyEvidence    `json:"evidence,omitempty"`
}

type StrategySelection struct {
	PolicyVersion string   `json:"policy_version"`
	StrategyID    string   `json:"strategy_id"`
	Override      bool     `json:"override"`
	Reasons       []string `json:"reasons"`
	EvidenceScope string   `json:"evidence_scope,omitempty"`
	EvidenceIDs   []string `json:"evidence_ids,omitempty"`
}

type PackagingSelectionRecord struct {
	PolicyVersion string                `json:"policy_version"`
	StackFacts    StackFacts            `json:"stack_facts"`
	Eligibility   []StrategyEligibility `json:"eligibility"`
	Selected      StrategySelection     `json:"selected"`
}

func EvaluateSelection(req PackagingSelectionRequest) (StrategySelection, error) {
	if req.Policy.Version == "" || req.Policy.DefaultStrategy == "" {
		return StrategySelection{}, fmt.Errorf("selection policy version and default strategy are required")
	}
	byID := make(map[string]StrategyEligibility, len(req.Eligible))
	for _, e := range req.Eligible {
		byID[e.Strategy.ID] = e
	}
	if req.Override != "" {
		e, ok := byID[req.Override]
		if !ok {
			return StrategySelection{}, fmt.Errorf("strategy override %q is not registered", req.Override)
		}
		if !e.Eligible {
			return StrategySelection{}, fmt.Errorf("strategy override %q is ineligible", req.Override)
		}
		return StrategySelection{PolicyVersion: req.Policy.Version, StrategyID: req.Override, Override: true, Reasons: []string{"explicit override selected"}}, nil
	}
	for _, scope := range []string{"scenario", "stack_class"} {
		var best *StrategyEvidence
		bestScore := 0.0
		for index := range req.Evidence {
			evidence := &req.Evidence[index]
			strategy, registered := byID[evidence.StrategyID]
			if !registered || !strategy.Eligible || evidence.Scope != scope || evidence.JourneyParity != "pass" || evidence.LaunchReliability < req.Policy.MinimumLaunchReliability {
				continue
			}
			score, usable := weightedEvidenceScore(req.Policy.Weights, evidence.Metrics)
			if !usable {
				continue
			}
			if best == nil || score < bestScore || (score == bestScore && evidence.StrategyID < best.StrategyID) {
				best, bestScore = evidence, score
			}
		}
		if best != nil {
			return StrategySelection{PolicyVersion: req.Policy.Version, StrategyID: best.StrategyID, Reasons: []string{"lowest policy score among evidence passing parity and reliability gates"}, EvidenceScope: scope, EvidenceIDs: append([]string(nil), best.EvidenceIDs...)}, nil
		}
	}
	if e, ok := byID[req.Policy.DefaultStrategy]; ok && e.Eligible {
		return StrategySelection{PolicyVersion: req.Policy.Version, StrategyID: e.Strategy.ID, Reasons: []string{"eligible ramp default selected; no packaged evidence policy input supplied"}}, nil
	}
	return StrategySelection{}, fmt.Errorf("default strategy %q is not eligible", req.Policy.DefaultStrategy)
}

func weightedEvidenceScore(weights, metrics map[string]float64) (float64, bool) {
	total, included := 0.0, false
	for key, weight := range weights {
		if value, ok := metrics[key]; ok {
			total += value * weight
			included = true
		}
	}
	return total, included
}
