package autosteer

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// standardsResolver builds a resolver where every given skill targets standards.
func standardsResolver(skills ...string) *skillmap.Resolver {
	decls := make([]skillmap.SkillDeclaration, len(skills))
	for i, s := range skills {
		decls[i] = skillmap.SkillDeclaration{ID: s, Dimensions: []string{"standards"}}
	}
	return skillmap.NewResolverWithWarner(&skillmap.FakeCatalog{Declarations: decls}, func(string, ...any) {})
}

func standardsState() findings.FindingsState {
	return findings.BuildState([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
}

func banditSelector(res SkillResolver, store effectiveness.Store, epsilon float64, seed uint64) *Selector {
	return NewSelectorWithConfig(SelectorConfig{
		Resolver:      res,
		Effectiveness: store,
		Prior:         0,
		ShrinkageK:    effectiveness.DefaultShrinkageK,
		Epsilon:       epsilon,
		Seed:          seed,
	})
}

// (a) With no evidence, the bandit reproduces v0 greedy ordering exactly.
func TestBandit_ColdStartEqualsGreedy(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	state := standardsState()

	greedy := NewSelector(res).SelectNextSkill(state, profile)
	bandit := banditSelector(res, effectiveness.NewMemoryStore(), 0, 1).SelectNextSkill(state, profile)

	if greedy.SkillID != bandit.SkillID {
		t.Fatalf("cold-start bandit must match greedy: greedy=%q bandit=%q", greedy.SkillID, bandit.SkillID)
	}
	if bandit.SkillID != "refactor" {
		t.Fatalf("expected first allow-set skill 'refactor', got %q", bandit.SkillID)
	}
}

// (b) A proven high-efficacy skill beats the first-alphabetical/allow-order one.
func TestBandit_HighEfficacyWins(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	store := effectiveness.NewMemoryStore()
	// lint-fix has proven, cheap, positive net efficacy; refactor is unobserved.
	store.Seed(effectiveness.Stat{SkillID: "lint-fix", Dimension: "standards", ClosedCount: 20, TotalRuns: 5, TotalTokens: 5000})

	got := banditSelector(res, store, 0, 1).SelectNextSkill(standardsState(), profile)
	if got.SkillID != "lint-fix" {
		t.Fatalf("expected proven 'lint-fix' to win, got %q (%s)", got.SkillID, got.Rationale)
	}
}

// (c) A skill with negative net efficacy is deprioritized below an unobserved one.
func TestBandit_NegativeEfficacyDeprioritized(t *testing.T) {
	res := standardsResolver("bad", "fresh")
	profile := &AutoSteerProfile{AllowedSkills: []string{"bad", "fresh"}}
	store := effectiveness.NewMemoryStore()
	// 'bad' introduces more than it closes (net-negative); 'fresh' is unobserved (prior 0).
	store.Seed(effectiveness.Stat{SkillID: "bad", Dimension: "standards", ClosedCount: 0, IntroducedCount: 10, TotalRuns: 4, TotalTokens: 4000})

	got := banditSelector(res, store, 0, 1).SelectNextSkill(standardsState(), profile)
	if got.SkillID != "fresh" {
		t.Fatalf("expected net-negative 'bad' deprioritized in favour of 'fresh', got %q (%s)", got.SkillID, got.Rationale)
	}
}

// (d) Exploration is deterministic for a fixed (task, iteration) seed.
func TestBandit_ExplorationDeterministic(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix", "polish")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix", "polish"}}
	store := effectiveness.NewMemoryStore()
	store.Seed(effectiveness.Stat{SkillID: "refactor", Dimension: "standards", ClosedCount: 30, TotalRuns: 5, TotalTokens: 5000})

	seed := explorationSeed("task-x", 1)
	first := banditSelector(res, store, 1.0, seed).SelectNextSkill(standardsState(), profile)
	for i := 0; i < 10; i++ {
		got := banditSelector(res, store, 1.0, seed).SelectNextSkill(standardsState(), profile)
		if got.SkillID != first.SkillID {
			t.Fatalf("exploration not deterministic for fixed seed: %q vs %q", got.SkillID, first.SkillID)
		}
	}

	// And the epsilon schedule decays with iteration.
	if explorationEpsilon(0) <= explorationEpsilon(5) {
		t.Fatalf("epsilon must decay: ε(0)=%v ε(5)=%v", explorationEpsilon(0), explorationEpsilon(5))
	}
}
