package render

import (
	"fmt"
	"strings"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
)

// The capability router.
//
// Which lane draws a picture used to be decided by two constants in this file:
// every model-backed style went to the same policy pair, and every style that
// needed more than the installed local model set simply could not be made. The
// choice now belongs to the style, as a declared quality tier, and this file is
// the one place that turns a tier into a lane and records what happened.
//
// The tier is a floor AND a ceiling, and both halves matter:
//
//   - Floor, on capability. A procedural generator does not draw a photographic
//     interior. Serving a `frontier_model` style from a local SD checkpoint
//     would produce a picture wearing a label it did not earn, which is the
//     failure the plan calls "a screen over a source with no picture in it"
//     wearing a different hat.
//
//   - Ceiling, on spend. A style authorises the money its tier costs and no
//     more. Escalating past the declared tier when a lane is unavailable is
//     precisely the defect that once billed every render to a cloud provider
//     while an installed local GPU sat idle — so a lane above the declared tier
//     is never tried, and a tier no lane can meet is refused by name.
//
// The ladder is still walked cheapest-first, and every lane it passes is
// recorded with the reason it did not serve. An operator reading a job sees the
// escalation, not just its result.
const (
	// LaneProcedural draws in-process. Free, offline, deterministic.
	LaneProcedural = "procedural"
	// LaneLocalModel runs a diffusion model through image-tools' local rungs.
	// Free at the margin, offline, non-deterministic.
	LaneLocalModel = "local_model"
	// LaneFrontierModel runs a current frontier image model through
	// image-tools' BYOK rung and its ai-gateway client. Costs money.
	LaneFrontierModel = "frontier_model"
)

// ladder is the lane order, cheapest first. It is the iteration order of every
// routing decision, and the order attempts are recorded in.
var ladder = []string{LaneProcedural, LaneLocalModel, LaneFrontierModel}

// LaneAttempt is one rung of the walk: the lane and why it did not serve.
type LaneAttempt struct {
	Lane, Detail string
}

// Routing is what actually served one candidate. Every field answers a question
// with no other source after the fact: the lane decides whether the picture can
// be reproduced offline, the model decides whether it can be disclosed
// honestly, and the cost decides whether the catalog is affordable to render.
type Routing struct {
	DeclaredTier  string
	ServedLane    string
	ModelID       string
	ExecutionTier string
	CostUSD       float64
	// CostReported separates "cost nothing" from "nobody reported a cost". A
	// frontier render whose charge went unrecorded must not read as free.
	CostReported bool
	Attempts     []LaneAttempt
}

// LaneRefusedError reports that no lane could meet a style's declared tier.
//
// It names the tier and every lane tried, because the operator's next action
// differs completely by cause: an absent local model is an install, absent
// gateway credentials are a configuration, and a style whose strategy cannot
// reach its tier at all is a catalog edit.
type LaneRefusedError struct {
	StyleID  string
	Tier     string
	Attempts []LaneAttempt
}

func (e *LaneRefusedError) Error() string {
	tried := make([]string, 0, len(e.Attempts))
	for _, attempt := range e.Attempts {
		tried = append(tried, fmt.Sprintf("%s (%s)", attempt.Lane, attempt.Detail))
	}
	return fmt.Sprintf("render: style %q declares quality tier %q and no lane meets it; tried %s",
		e.StyleID, e.Tier, strings.Join(tried, ", "))
}

// strategyNeedsModel reports whether a strategy can be drawn at all without a
// model. It is a property of the strategy rather than a preference: a
// `procedural` style ships what its generator drew and never calls a model, and
// a `synthesized` style has nothing but a prompt and cannot be drawn without
// one. Both model lanes serve the same strategies; they differ only in which
// models they may reach.
func strategyNeedsModel(strategy string) bool {
	return strategy == "guided" || strategy == "synthesized"
}

// laneServesStrategy pairs a lane with the strategies it can actually draw.
func laneServesStrategy(lane, strategy string) bool {
	return (lane == LaneProcedural) != strategyNeedsModel(strategy)
}

// resolveLane turns a declared tier into the lane that will serve it, recording
// each cheaper lane it passes and why.
//
// The returned Routing carries the walk even on success, so a job records the
// reasoning and not only the verdict.
func resolveLane(style catalog.Style) (Routing, error) {
	tier := style.EffectiveQualityTier()
	routing := Routing{DeclaredTier: tier}

	for _, lane := range ladder {
		switch {
		case laneRank(lane) < laneRank(tier):
			routing.Attempts = append(routing.Attempts, LaneAttempt{
				Lane:   lane,
				Detail: fmt.Sprintf("below the declared tier %q", tier),
			})
		case laneRank(lane) > laneRank(tier):
			// Never tried. Recorded anyway so the ceiling is visible in the
			// job rather than being an unstated property of the code.
			routing.Attempts = append(routing.Attempts, LaneAttempt{
				Lane:   lane,
				Detail: fmt.Sprintf("above the declared tier %q; not authorised", tier),
			})
		case !laneServesStrategy(lane, style.Strategy):
			// The tier and the strategy disagree. Store validation refuses this
			// pairing at write time, so reaching it means a style was written
			// around the store — refuse rather than pick one of the two and
			// silently ship a picture nobody asked for.
			routing.Attempts = append(routing.Attempts, LaneAttempt{
				Lane:   lane,
				Detail: fmt.Sprintf("strategy %q cannot be drawn on it", style.Strategy),
			})
		default:
			routing.ServedLane = lane
			return routing, nil
		}
	}
	return routing, &LaneRefusedError{StyleID: style.ID, Tier: tier, Attempts: routing.Attempts}
}

func laneRank(lane string) int {
	switch lane {
	case LaneProcedural:
		return 1
	case LaneLocalModel:
		return 2
	case LaneFrontierModel:
		return 3
	default:
		return 0
	}
}

// generationPolicy is the routing policy image-tools is asked to honour for one
// lane. It is derived from the lane rather than hardcoded, which is the whole
// point: the pair used to be two constants, so a style that needed a frontier
// model had no way to say so and a style that did not still risked reaching one.
type generationPolicy struct {
	QualityPolicy, FallbackPolicy string
	AllowBYOK                     bool
}

func policyForLane(lane string) (generationPolicy, error) {
	switch lane {
	case LaneLocalModel:
		// local_only is the ceiling made mechanical: image-tools refuses rather
		// than reaching for a paid provider, so a local-tier style cannot spend
		// money even if a cloud route is configured and cheaper to reach.
		return generationPolicy{QualityPolicy: "balanced", FallbackPolicy: "local_only", AllowBYOK: false}, nil
	case LaneFrontierModel:
		return generationPolicy{QualityPolicy: "quality", FallbackPolicy: "any", AllowBYOK: true}, nil
	default:
		return generationPolicy{}, fmt.Errorf("render: lane %q runs no model and has no routing policy", lane)
	}
}

// recordGeneration folds what image-tools reported into the routing record. The
// model and execution tier are image-tools' answers, never this scenario's
// request: recording a requested model as the used one would be a fabricated
// disclosure.
func (r *Routing) recordGeneration(result imageengine.GenerationResult) {
	r.ModelID = result.ModelID
	r.ExecutionTier = result.Tier
	r.CostUSD = result.CostUSD
	r.CostReported = result.CostReported
}
