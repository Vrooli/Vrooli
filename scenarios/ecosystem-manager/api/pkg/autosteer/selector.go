package autosteer

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
)

// SkillResolver is the subset of skillmap.Resolver the selector needs. It is an
// interface so selection can be unit-tested without a full catalog.
//
// seam: SkillResolver maps a dimension to the skills (intersected with the
// profile allow-set) that can close it.
type SkillResolver interface {
	EligibleSkills(dim dimensions.Dimension, allow []string) []string
}

var _ SkillResolver = (*skillmap.Resolver)(nil)

// EligibilityFilter is the Layer-1 hard-gate seam (DTV, P2): it may veto a skill
// from selection before the bandit ranks it, making a dimension non-actionable
// when no eligible skill passes. P1 wires the permissive AllowAllFilter; P2
// supplies a development-toolchain-validator-backed filter without touching
// selection logic.
//
// seam: EligibilityFilter is the pre-selection skill gate; production (P1) wires
// AllowAllFilter, P2 swaps a DTV-backed implementation.
type EligibilityFilter interface {
	Allow(skillID string, dim dimensions.Dimension) bool
}

// AllowAllFilter permits every skill (the P1 default — no Layer-1 gating).
type AllowAllFilter struct{}

// Allow implements EligibilityFilter.
func (AllowAllFilter) Allow(string, dimensions.Dimension) bool { return true }

// PriorProvider supplies the cold-start expected-efficacy-per-token prior for an
// unobserved (skill, dimension). P1 wires UniformPrior (a single neutral value
// for every skill); P2 wires the DTV trust/cost prior. The bandit blends this
// prior toward observed evidence (see effectiveness.ExpectedEfficacyPerToken),
// so accumulating live runs washes it out regardless of source.
//
// seam: PriorProvider is the cold-start prior source; production (P1) wires
// UniformPrior, P2 swaps a DTV-backed implementation.
type PriorProvider interface {
	Prior(skillID string, dim dimensions.Dimension) float64
}

// UniformPrior returns the same prior for every skill — the P1 default. Value 0
// reproduces v0 greedy ordering at cold start (all skills tie).
type UniformPrior struct{ Value float64 }

// Prior implements PriorProvider.
func (u UniformPrior) Prior(string, dimensions.Dimension) float64 { return u.Value }

// TrustRanker scores a skill's DTV trust (higher = more trusted). It is the
// ordering key for the degraded-gate fallback: when the Layer-1 gate would veto
// every candidate (all-red) the selector proceeds with the highest-trust skill
// rather than first-in-order (the "least-bad" pick of the proceed-cap-flag
// policy). Nil ranker ⇒ candidate order is left untouched.
//
// seam: TrustRanker orders the all-red fallback set; production wires the DTV
// FitnessSnapshot, tests wire a stub.
type TrustRanker interface {
	Trust(skillID string) float64
}

// Selection is the controller's SELECT-stage decision.
type Selection struct {
	// SkillID is the chosen skill, or "" when no eligible skill exists for any
	// open dimension (the caller treats this as "nothing actionable").
	SkillID string
	// Dimension is the heaviest open dimension the chosen skill targets.
	Dimension dimensions.Dimension
	// WeightedScore is that dimension's profile-weighted open score.
	WeightedScore float64
	// Rationale is a human-readable explanation for the decision trace.
	Rationale string
	// PredictedReduction is the controller's forward estimate of the weighted-score
	// reduction the chosen skill will realize: the bandit's expected
	// reduction-per-token (ExpectedEfficacyPerToken, in net findings closed per
	// 1000 tokens) scaled by the estimated run tokens and the chosen dimension's
	// weight, so it is comparable to the realized weighted-score delta (EM-P4). 0
	// at greedy cold start (no ledger) or when no efficacy estimate is computable.
	PredictedReduction float64
	// ExcludedSkills lists skills the Layer-1 eligibility gate denied for the
	// chosen dimension (their ids). Empty under allow-all (P1). The reason is
	// attached by the caller, which owns the gate's data source.
	ExcludedSkills []string
	// GateOverride is true when the eligibility gate would have emptied the
	// chosen dimension and the selector fell back to the trust-ordered unfiltered
	// set rather than stall (the all-red branch of the proceed-cap-flag policy:
	// proceed with the least-bad skill). The caller flags the iteration and halves
	// the remaining budget once. Surfaced in the trace.
	GateOverride bool
}

// Selector implements the controller's SELECT stage. The outer loop ranks open
// dimensions by profile-weighted severity (which dimension matters most now);
// within the heaviest actionable dimension the v1 bandit ranks eligible skills
// by expected reduction-per-token from the effectiveness ledger, blended toward
// a cold-start prior so a fresh target degrades to v0 greedy ordering (see
// CONTROL-MODEL.md "Selection Policy"). With no ledger wired it is pure greedy.
type Selector struct {
	resolver SkillResolver
	eff      effectiveness.Store
	prior    PriorProvider
	k        float64
	// epsilon is the (already iteration-decayed) exploration probability; seed
	// makes the explore decision deterministic for a fixed (task, iteration).
	epsilon float64
	seed    uint64
	// filter is the Layer-1 hard gate (P1: allow-all). cooldown holds skills that
	// regressed their own target dimension recently and are deprioritized
	// (hysteresis) — soft, overridden only when every eligible skill is cooling.
	filter   EligibilityFilter
	cooldown map[string]bool
	// redRank orders the all-red fallback set by DTV trust (least-bad first) when
	// the gate degrades. Nil ⇒ candidate order is unchanged.
	redRank TrustRanker
}

// NewSelector creates a pure-greedy Selector (no effectiveness weighting, no
// exploration) over a skill resolver. Used where no ledger is available.
func NewSelector(resolver SkillResolver) *Selector {
	return &Selector{resolver: resolver, prior: UniformPrior{}, k: effectiveness.DefaultShrinkageK}
}

// SelectorConfig wires the v1 effectiveness-weighted selector.
type SelectorConfig struct {
	Resolver      SkillResolver
	Effectiveness effectiveness.Store
	// Prior supplies the cold-start efficacy for an unobserved skill. nil wires
	// UniformPrior{0} (P1 greedy cold start); P2 wires the DTV trust/cost prior.
	Prior PriorProvider
	// ShrinkageK is the shrinkage constant; <=0 uses the package default.
	ShrinkageK float64
	// Epsilon is the exploration probability for this selection (already decayed
	// by iteration); 0 disables exploration.
	Epsilon float64
	// Seed makes exploration reproducible for a fixed (task, iteration).
	Seed uint64
	// Filter is the Layer-1 hard gate; nil means allow-all.
	Filter EligibilityFilter
	// Cooldown lists skills deprioritized this iteration (hysteresis); nil = none.
	Cooldown map[string]bool
	// RedRank orders the all-red degraded-gate fallback by DTV trust (least-bad
	// first); nil leaves candidate order unchanged.
	RedRank TrustRanker
}

// NewSelectorWithConfig creates a Selector from an explicit configuration.
func NewSelectorWithConfig(cfg SelectorConfig) *Selector {
	k := cfg.ShrinkageK
	if k <= 0 {
		k = effectiveness.DefaultShrinkageK
	}
	filter := cfg.Filter
	if filter == nil {
		filter = AllowAllFilter{}
	}
	prior := cfg.Prior
	if prior == nil {
		prior = UniformPrior{}
	}
	return &Selector{
		resolver: cfg.Resolver,
		eff:      cfg.Effectiveness,
		prior:    prior,
		k:        k,
		epsilon:  cfg.Epsilon,
		seed:     cfg.Seed,
		filter:   filter,
		cooldown: cfg.Cooldown,
		redRank:  cfg.RedRank,
	}
}

// weightedDimension is a dimension with its profile-weighted open score.
type weightedDimension struct {
	dim    dimensions.Dimension
	score  float64
	weight float64
	count  int
}

// rankDimensions orders open dimensions by profile-weighted score descending,
// with a deterministic alphabetical tiebreak. A dimension absent from the
// weights map defaults to weight 1.0.
func rankDimensions(state findings.FindingsState, profile *AutoSteerProfile) []weightedDimension {
	weights := map[string]float64{}
	if profile != nil {
		weights = profile.Objective.DimensionWeights
	}
	ranked := make([]weightedDimension, 0, len(state.DimensionScore))
	for dim, raw := range state.DimensionScore {
		if raw <= 0 {
			continue
		}
		w := 1.0
		if wv, ok := weights[string(dim)]; ok {
			w = wv
		}
		ranked = append(ranked, weightedDimension{
			dim:    dim,
			score:  raw * w,
			weight: w,
			count:  state.DimensionCount[dim],
		})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].dim < ranked[j].dim
	})
	return ranked
}

// SelectNextSkill picks the skill that best closes the heaviest open dimension.
// It falls through to the next dimension when the heaviest has no eligible
// skill in the allow-set, and returns an empty SkillID when nothing is
// actionable.
func (s *Selector) SelectNextSkill(state findings.FindingsState, profile *AutoSteerProfile) Selection {
	ranked := rankDimensions(state, profile)
	if len(ranked) == 0 {
		return Selection{Rationale: "no open findings — nothing to select"}
	}

	var allow []string
	if profile != nil {
		allow = profile.AllowedSkills
	}

	skipped := make([]string, 0)
	for _, wd := range ranked {
		raw := s.resolver.EligibleSkills(wd.dim, allow)
		if len(raw) == 0 {
			skipped = append(skipped, string(wd.dim))
			continue
		}
		eligible, excluded := s.applyFilter(wd.dim, raw)
		gateOverride := false
		if len(eligible) == 0 {
			// The Layer-1 gate vetoed every skill for this dimension. Per the
			// proceed-cap-flag policy we do NOT stall: fall back to the unfiltered
			// set, but proceed with the least-bad skill — order the red set by DTV
			// trust descending (stable) so chooseSkill's tiebreak lands on the
			// highest-trust candidate. The iteration is flagged (GateOverride) and
			// the caller halves the remaining budget once.
			eligible = s.orderByTrust(raw)
			gateOverride = true
		}
		chosen, pick := s.chooseSkill(wd.dim, eligible)
		predicted := pick.predictedReduction(chosen, wd.weight)
		rationale := fmt.Sprintf(
			"dimension %q is the heaviest open cluster with an eligible skill (weighted score %.2f, %d findings, weight %.2f); %s",
			wd.dim, wd.score, wd.count, wd.weight, pick.describe(chosen),
		)
		switch {
		case gateOverride:
			rationale += fmt.Sprintf(" (Layer-1 gate vetoed all candidate(s) %v — proceeding anyway to avoid stalling)", excluded)
		case len(excluded) > 0:
			rationale += fmt.Sprintf(" (Layer-1 gate excluded %v)", excluded)
		}
		if len(skipped) > 0 {
			rationale += fmt.Sprintf(" (skipped %v — no eligible skill in allow-set)", skipped)
		}
		return Selection{
			SkillID:            chosen,
			Dimension:          wd.dim,
			WeightedScore:      wd.score,
			Rationale:          rationale,
			ExcludedSkills:     excluded,
			GateOverride:       gateOverride,
			PredictedReduction: predicted,
		}
	}

	return Selection{
		Rationale: fmt.Sprintf("no eligible skill in allow-set for any open dimension %v", skipped),
	}
}

// candidate is one eligible skill scored for selection.
type candidate struct {
	id        string
	efficacy  float64
	prior     float64
	samples   int64
	avgTokens int64
}

// skillPick records how a skill was chosen, for a transparent rationale.
type skillPick struct {
	method     string // "greedy" | "effectiveness" | "exploration"
	candidates []candidate
}

// defaultEstimatedRunTokens is the cold-start token estimate (in tokens) used to
// scale a forward reduction prediction when the chosen skill has no observed
// per-run token cost yet. Order-of-magnitude of a typical steer run; only the
// calibration trend matters, and live runs replace it with the observed mean.
const defaultEstimatedRunTokens = 3000

// predictedReduction is the controller's forward estimate of the weighted-score
// reduction the chosen candidate will realize this iteration (EM-P4):
//
//	efficacy (net findings closed / 1000 tok) · estTokens/1000 · dimensionWeight
//
// The dimension-weight factor lifts the net-findings-per-token efficacy into the
// same weighted-score units as the realized delta (an approximation: it assumes
// an average per-finding severity weight of 1). Greedy picks (no candidate
// stats) predict 0. estTokens is the chosen skill's observed mean, or the
// cold-start default when it has no runs yet.
func (p skillPick) predictedReduction(chosen string, dimWeight float64) float64 {
	for _, c := range p.candidates {
		if c.id != chosen {
			continue
		}
		estTokens := c.avgTokens
		if estTokens <= 0 {
			estTokens = defaultEstimatedRunTokens
		}
		if dimWeight <= 0 {
			dimWeight = 1.0
		}
		return c.efficacy * (float64(estTokens) / 1000.0) * dimWeight
	}
	return 0
}

// applyFilter splits eligible skills into those the Layer-1 gate permits and
// those it vetoes (P1: none vetoed). A nil filter permits everything. The
// excluded list (empty under allow-all) feeds the decision trace.
func (s *Selector) applyFilter(dim dimensions.Dimension, eligible []string) (allowed, excluded []string) {
	if s.filter == nil {
		return eligible, nil
	}
	allowed = make([]string, 0, len(eligible))
	for _, id := range eligible {
		if s.filter.Allow(id, dim) {
			allowed = append(allowed, id)
		} else {
			excluded = append(excluded, id)
		}
	}
	return allowed, excluded
}

// orderByTrust returns the candidates ordered by DTV trust descending (stable,
// alphabetical id tiebreak) so the all-red fallback picks the least-bad skill.
// With no ranker wired the original order is preserved.
func (s *Selector) orderByTrust(ids []string) []string {
	if s.redRank == nil || len(ids) < 2 {
		return ids
	}
	out := make([]string, len(ids))
	copy(out, ids)
	sort.SliceStable(out, func(i, j int) bool {
		ti, tj := s.redRank.Trust(out[i]), s.redRank.Trust(out[j])
		if ti != tj {
			return ti > tj
		}
		return out[i] < out[j]
	})
	return out
}

// preferred applies the hysteresis cooldown: it drops skills cooling down after
// regressing their own target dimension, unless every eligible skill is cooling
// (then it returns all, to avoid stalling for lack of an alternative).
func (s *Selector) preferred(eligible []string) []string {
	if len(s.cooldown) == 0 {
		return eligible
	}
	out := make([]string, 0, len(eligible))
	for _, id := range eligible {
		if !s.cooldown[id] {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return eligible
	}
	return out
}

// chooseSkill ranks eligible skills within a dimension. Without a ledger it is
// greedy (first eligible). With a ledger it picks the highest expected
// reduction-per-token, occasionally exploring per the epsilon schedule. Skills
// on cooldown are deprioritized first (hysteresis).
func (s *Selector) chooseSkill(dim dimensions.Dimension, eligible []string) (string, skillPick) {
	eligible = s.preferred(eligible)
	if s.eff == nil {
		return eligible[0], skillPick{method: "greedy"}
	}

	stats, _ := s.eff.Bulk(dim) // a read error degrades to all-prior (greedy order)
	cands := make([]candidate, len(eligible))
	for i, id := range eligible {
		st := stats[id] // zero value = cold start (n=0 → prior)
		prior := s.prior.Prior(id, dim)
		cands[i] = candidate{
			id:        id,
			efficacy:  st.ExpectedEfficacyPerToken(prior, s.k),
			prior:     prior,
			samples:   st.TotalRuns,
			avgTokens: avgTokens(st),
		}
	}

	// Exploitation: highest efficacy, ties broken by eligible (allow-set) order.
	best := 0
	for i := 1; i < len(cands); i++ {
		if cands[i].efficacy > cands[best].efficacy {
			best = i
		}
	}

	if explore, idx := s.exploreDecision(len(cands)); explore && idx != best {
		return cands[idx].id, skillPick{method: "exploration", candidates: cands}
	}
	return cands[best].id, skillPick{method: "effectiveness", candidates: cands}
}

// exploreDecision returns whether to explore and which candidate index to take,
// deterministically for the selector's seed. No exploration with ε<=0 or a
// single candidate.
func (s *Selector) exploreDecision(n int) (bool, int) {
	if s.epsilon <= 0 || n <= 1 {
		return false, 0
	}
	rng := rand.New(rand.NewSource(int64(s.seed))) //nolint:gosec // deterministic, non-cryptographic exploration
	if rng.Float64() < s.epsilon {
		return true, rng.Intn(n)
	}
	return false, 0
}

// avgTokens returns the mean tokens per run for a stat (0 when no runs).
func avgTokens(st effectiveness.Stat) int64 {
	if st.TotalRuns <= 0 {
		return 0
	}
	return st.TotalTokens / st.TotalRuns
}

// describe renders the human-readable selection clause for the decision trace.
func (p skillPick) describe(chosen string) string {
	switch p.method {
	case "greedy":
		return fmt.Sprintf("selected %q (greedy: no effectiveness data yet)", chosen)
	case "exploration":
		return fmt.Sprintf("selected %q by exploration %s", chosen, p.candidateSummary())
	default:
		return fmt.Sprintf("selected %q by effectiveness %s", chosen, p.candidateSummary())
	}
}

// candidateSummary lists each candidate's efficacy estimate and sample size.
func (p skillPick) candidateSummary() string {
	if len(p.candidates) == 0 {
		return ""
	}
	parts := make([]string, 0, len(p.candidates))
	for _, c := range p.candidates {
		if c.samples == 0 {
			parts = append(parts, fmt.Sprintf("%s=%.2f/khtok(n=0,prior=%.2f)", c.id, c.efficacy, c.prior))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%.2f/khtok(n=%d,~%dtok/run,prior=%.2f)", c.id, c.efficacy, c.samples, c.avgTokens, c.prior))
	}
	return "[" + strings.Join(parts, " ") + "]"
}
