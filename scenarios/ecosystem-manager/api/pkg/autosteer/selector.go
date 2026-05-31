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
	prior    float64
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
}

// NewSelector creates a pure-greedy Selector (no effectiveness weighting, no
// exploration) over a skill resolver. Used where no ledger is available.
func NewSelector(resolver SkillResolver) *Selector {
	return &Selector{resolver: resolver, k: effectiveness.DefaultShrinkageK}
}

// SelectorConfig wires the v1 effectiveness-weighted selector.
type SelectorConfig struct {
	Resolver      SkillResolver
	Effectiveness effectiveness.Store
	// Prior is the cold-start efficacy assumed for an unobserved skill (uniform
	// across skills in P1; the injection point for P2's DTV trust/cost priors).
	Prior float64
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
	return &Selector{
		resolver: cfg.Resolver,
		eff:      cfg.Effectiveness,
		prior:    cfg.Prior,
		k:        k,
		epsilon:  cfg.Epsilon,
		seed:     cfg.Seed,
		filter:   filter,
		cooldown: cfg.Cooldown,
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
		eligible := s.applyFilter(wd.dim, s.resolver.EligibleSkills(wd.dim, allow))
		if len(eligible) == 0 {
			skipped = append(skipped, string(wd.dim))
			continue
		}
		chosen, pick := s.chooseSkill(wd.dim, eligible)
		rationale := fmt.Sprintf(
			"dimension %q is the heaviest open cluster with an eligible skill (weighted score %.2f, %d findings, weight %.2f); %s",
			wd.dim, wd.score, wd.count, wd.weight, pick.describe(chosen),
		)
		if len(skipped) > 0 {
			rationale += fmt.Sprintf(" (skipped %v — no eligible skill in allow-set)", skipped)
		}
		return Selection{
			SkillID:       chosen,
			Dimension:     wd.dim,
			WeightedScore: wd.score,
			Rationale:     rationale,
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
	samples   int64
	avgTokens int64
}

// skillPick records how a skill was chosen, for a transparent rationale.
type skillPick struct {
	method     string // "greedy" | "effectiveness" | "exploration"
	candidates []candidate
}

// applyFilter drops skills the Layer-1 gate vetoes (P1: none). A nil filter
// permits everything.
func (s *Selector) applyFilter(dim dimensions.Dimension, eligible []string) []string {
	if s.filter == nil {
		return eligible
	}
	out := make([]string, 0, len(eligible))
	for _, id := range eligible {
		if s.filter.Allow(id, dim) {
			out = append(out, id)
		}
	}
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
		cands[i] = candidate{
			id:        id,
			efficacy:  st.ExpectedEfficacyPerToken(s.prior, s.k),
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
			parts = append(parts, fmt.Sprintf("%s=%.2f/khtok(n=0,prior)", c.id, c.efficacy))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%.2f/khtok(n=%d,~%dtok/run)", c.id, c.efficacy, c.samples, c.avgTokens))
	}
	return "[" + strings.Join(parts, " ") + "]"
}
