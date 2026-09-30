// Package convergence measures the upstream generators of scenarios: per-template
// fitness across the four reference-pattern-fitness lenses, gold-star
// generated-golden health/eligibility, and the convergence trend over dated
// fitness-audit records. It delegates raw code structure to
// code-facts/architecture-cartographer and toolchain-clean results to
// test-genie/scenario-auditor — it never re-runs the toolchain. It surfaces
// numbers + candidates only; tiering/substrate/nomination stay agentic.
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {FitnessScanner, ReferenceScanner, Repository}
//	             ↑              ↑ (faked in tests)              ↑
//	          (proto edge)   live fs / upstream reads     fitness-audit index
//
// The proto wire types live one floor up and never import this package; the
// handler is the only translation point (api-steer §7). See
// docs/concepts/DOMAINS.md (convergence) and
// docs/agent-system/REFERENCE_PATTERN_FITNESS.md (the four sub-lenses).
package convergence

import "time"

// FitnessTier is a coarse advisory grade derived from the four lenses.
type FitnessTier int

const (
	TierUnspecified FitnessTier = iota
	TierStrong
	TierFair
	TierWeak
)

// ReferenceEligibility is the gold-star eligibility verdict for a reference.
type ReferenceEligibility int

const (
	EligibilityUnspecified ReferenceEligibility = iota
	EligibilityEligible
	EligibilityCandidate
	EligibilityIneligible
)

// TemplateFitness is the four-lens fitness of one template. The counts are
// honest filesystem-derived proxies for the reference-pattern-fitness lenses
// (the precise code-facts/cartographer mechanization is a documented refinement
// seam); the tier is advisory only.
type TemplateFitness struct {
	Template                 string
	PerReplicaCost           int // lens 1: LOC each replica must carry
	DriftSurfaceCount        int // lens 2: surfaces where replicas drift (local fakes / duplicate surfaces)
	CommentOnlyContractCount int // lens 3: contracts expressed in comments, not code
	CoordinatedEditCount     int // lens 4: central files an add/delete must touch together
	Tier                     FitnessTier
}

// ReferenceHealth is the gold-star health of one generated golden. The Scenario
// field name is retained for the existing proto/API contract and carries the
// durable golden slug.
type ReferenceHealth struct {
	Scenario          string
	StaleFromTemplate bool
	LastTemplateSync  time.Time
	CleanOnAllTools   bool
	StabilityDays     int // >= 60 for eligibility
	Breadth           int // number of patterns it demonstrates
	Eligibility       ReferenceEligibility
}

// FitnessTrendPoint is one dated point in a template's convergence trend.
type FitnessTrendPoint struct {
	Template             string
	At                   time.Time
	PerReplicaCost       int
	CoordinatedEditCount int
}

// Status is the convergence summary across all templates + references.
type Status struct {
	Templates  []TemplateFitness
	References []ReferenceHealth
}
