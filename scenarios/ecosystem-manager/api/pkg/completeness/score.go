// Package completeness is the closed-loop controller's read seam onto the
// scenario-completeness-scoring authority. EM no longer measures scenario health
// itself: the maturity rung, build state, operational-targets completion, and
// composite score are computed once — by completeness-scoring, over the cached
// test-genie phase-results EM's own audit just wrote — and consumed here.
//
// This replaces EM's deleted self-collected MetricsSnapshot subsystem. There is
// no fallback collector (operator mandate, plan D2): when the authority is
// unavailable the controller degrades loudly rather than substituting a
// home-grown approximation.
package completeness

import (
	"context"
	"regexp"
	"strconv"
	"time"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
)

// Provider fetches a scenario's completeness score. The controller depends on
// this interface (not the concrete client) so tests inject fakes.
type Provider interface {
	// Score returns the current cached completeness payload for a scenario. It
	// returns an error — never a zero-value substitute — when the authority is
	// unreachable, so the controller can degrade loudly (plan D2).
	Score(ctx context.Context, scenario string) (Score, error)
}

// Score is EM's local projection of completeness-scoring's GetScore payload: the
// single measurement the controller gates on. It carries the maturity rung
// (computed by the shared maturity-go ladder, a single evaluation per plan D4),
// the build verdict, operational-targets completion, and the composite score for
// observability.
type Score struct {
	// WorkingRung is the lowest unsatisfied rung label across R0–R4 (empty when
	// LadderClean). SatisfiedThrough is the highest contiguously satisfied rung
	// from R0. Both are the shared maturity-go ladder labels.
	WorkingRung      string `json:"working_rung"`
	SatisfiedThrough string `json:"satisfied_through"`
	LadderClean      bool   `json:"ladder_clean"`

	// BuildPassing is the build-critical verdict (approximated from cached phase
	// statuses; rung R0 encodes build).
	BuildPassing bool `json:"build_passing"`

	// Composite is the 0–100 completeness score (observability only — gates use
	// the rung and operational targets).
	Composite int `json:"composite"`

	// Operational-targets completion, parsed from the composite's
	// target_pass_rate metric line.
	OTTotal      int     `json:"ot_total"`
	OTPassing    int     `json:"ot_passing"`
	OTPercentage float64 `json:"ot_percentage"`
	OTHasTargets bool    `json:"ot_has_targets"`
	// OTKnown reports whether the operational-targets signal was actually
	// collected (the requirements collector contributed). False when that
	// collector degraded — the ladder's R4 gate then treats it as "unknown"
	// rather than "no targets declared".
	OTKnown bool `json:"ot_known"`

	// CalculatedAt is the server time the payload was computed. Plan D1: EM reads
	// the score AFTER its own audit writes fresh phase-results, so this reflects
	// the current iteration.
	CalculatedAt time.Time `json:"calculated_at"`
}

// otObservedRe parses completeness-scoring's stable preformatted operational-
// targets metric ("N total, M passing (P%)"). The percentage is integer-rounded
// upstream, which is fine for a threshold gate.
var otObservedRe = regexp.MustCompile(`(\d+)\s+total,\s+(\d+)\s+passing\s+\((\d+)%\)`)

// scoreFromProto maps the GetScore response onto the EM projection.
func scoreFromProto(resp *scoringv1.GetScoreResponse) Score {
	s := Score{}
	if resp == nil {
		return s
	}
	if m := resp.GetMaturity(); m != nil {
		s.WorkingRung = m.GetWorkingRung()
		s.SatisfiedThrough = m.GetSatisfiedThrough()
		s.LadderClean = m.GetLadderClean()
		s.BuildPassing = m.GetBuildPassing()
	}
	if c := resp.GetComposite(); c != nil {
		s.Composite = int(c.GetScore())
		s.parseOperationalTargets(c)
	}
	if t := resp.GetCalculatedAt(); t != nil {
		s.CalculatedAt = t.AsTime()
	}

	// OTKnown is true unless the requirements collector — the source of the
	// operational-targets signal — degraded for this payload.
	s.OTKnown = true
	for _, d := range resp.GetDegradations() {
		if d.GetCollector() == "requirements" {
			s.OTKnown = false
		}
	}
	s.OTHasTargets = s.OTTotal > 0
	return s
}

// parseOperationalTargets extracts the operational-targets counts from the
// composite's quality/target_pass_rate metric line.
func (s *Score) parseOperationalTargets(c *scoringv1.CompositeScore) {
	for _, g := range c.GetGroups() {
		if g.GetId() != "quality" {
			continue
		}
		for _, ml := range g.GetMetrics() {
			if ml.GetId() != "target_pass_rate" {
				continue
			}
			m := otObservedRe.FindStringSubmatch(ml.GetObserved())
			if len(m) != 4 {
				return
			}
			s.OTTotal, _ = strconv.Atoi(m[1])
			s.OTPassing, _ = strconv.Atoi(m[2])
			pct, _ := strconv.Atoi(m[3])
			s.OTPercentage = float64(pct)
			return
		}
	}
}
