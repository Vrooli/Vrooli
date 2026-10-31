package measures

import (
	"context"
	"time"
)

// Confidence levels assigned by the deterministic resolution tiers. The
// non-deterministic extractor supplies its own confidence.
const (
	// ConfidenceDeterministic is assigned when a canonical-typed param resolves
	// with no LLM (e.g. a time-window phrase matched exactly).
	ConfidenceDeterministic = 1.0
	// ConfidenceDefault is assigned when a param falls back to its manifest
	// default because the question did not name it.
	ConfidenceDefault = 0.8
)

// ExtractResult is a single non-deterministic param extraction outcome.
type ExtractResult struct {
	// Value is the extracted param value (must be within `allowed` when the
	// param is constrained). Meaningful only when Found is true.
	Value string
	// Found reports whether the extractor located the param in the question.
	Found bool
	// Confidence is the extractor's self-reported confidence in [0,1].
	Confidence float64
}

// ParamExtractor is the seam for non-deterministic param extraction — the
// tiers below "canonical-typed". Phase 3 wires a constrained-LLM implementation
// behind this interface; Phase 1 ships NoopExtractor so the package builds and
// abstains safely (a missing required param goes to needs[] rather than being
// guessed).
//
// `allowed` is the constrained value space (a static or resolved-dynamic enum's
// members); it is empty for a bare best-effort field, in which case the
// extractor may consult p.Min/Max/Format/Description for grounding.
type ParamExtractor interface {
	Extract(ctx context.Context, question string, p Param, allowed []string) (ExtractResult, error)
}

// NoopExtractor never finds a value. It is the Phase 1 default: everything
// non-canonical abstains, so ResolveParams correctly populates needs[] instead
// of guessing until Phase 3 wires the real extractor.
type NoopExtractor struct{}

// Extract always returns not-found.
func (NoopExtractor) Extract(context.Context, string, Param, []string) (ExtractResult, error) {
	return ExtractResult{}, nil
}

// ValuesProvider resolves a dynamic-enum `values_source` name to its live value
// set (e.g. "initiative_names" → the current initiative names). The owning
// scenario supplies the implementation; a nil provider means dynamic enums have
// no values and therefore abstain.
type ValuesProvider interface {
	Values(ctx context.Context, source string) ([]string, error)
}

// ResolveOptions carries the explicit inputs ResolveParams needs. Time and
// timezone are explicit (never an ambient wall-clock read) so resolution is
// reproducible; Extractor and Values are seams.
type ResolveOptions struct {
	// Now anchors relative time-window resolution. Zero means time.Now() — but
	// prefer passing it explicitly so results are deterministic.
	Now time.Time
	// Loc is the timezone for time-window resolution; nil means UTC.
	Loc *time.Location
	// Extractor handles non-canonical params; nil means NoopExtractor.
	Extractor ParamExtractor
	// Values resolves dynamic-enum value sets; nil means none available.
	Values ValuesProvider
}

// ResolveResult is the outcome of ResolveParams: the resolved param values, the
// names of required params that could not be resolved (abstention), and an
// overall confidence (the minimum confidence across resolved params).
type ResolveResult struct {
	// Params maps param name → resolved string value (a token for time_window,
	// an enum member, or an extracted scalar). Only successfully-resolved params
	// appear here.
	Params map[string]string
	// Needs lists required params that could not be resolved, in deterministic
	// order. A non-empty Needs means the measure must NOT auto-execute — it must
	// ask the user rather than guess.
	Needs []string
	// Confidence is the minimum confidence over resolved params (1.0 when no
	// params required resolution). It does NOT account for Needs — the caller
	// gates on `len(Needs)==0 && Confidence >= θ`.
	Confidence float64
}

// Complete reports whether every required param resolved (no abstention).
func (r ResolveResult) Complete() bool { return len(r.Needs) == 0 }

// ResolveParams maps a natural-language question onto a measure's parameters via
// three-tier degradation:
//
//  1. canonical-typed (time_window) → deterministic phrase resolver, no LLM;
//  2. constrained (static/dynamic enum, numeric bounds) → the ParamExtractor
//     constrained against the proto-derived value space;
//  3. bare name → the ParamExtractor best-effort.
//
// A param that cannot be resolved falls back to its manifest default; a
// required param with neither a resolution nor a default goes to Needs. Optional
// params that resolve to nothing are simply omitted (no penalty, no need).
func ResolveParams(ctx context.Context, question string, decl MeasureDeclaration, opts ResolveOptions) (ResolveResult, error) {
	extractor := opts.Extractor
	if extractor == nil {
		extractor = NoopExtractor{}
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	res := ResolveResult{
		Params:     make(map[string]string),
		Confidence: ConfidenceDeterministic,
	}
	minConf := ConfidenceDeterministic
	resolvedAny := false

	set := func(name, value string, conf float64) {
		res.Params[name] = value
		if conf < minConf {
			minConf = conf
		}
		resolvedAny = true
	}

	for _, name := range decl.ParamNames() {
		p := decl.Params[name]

		value, conf, found, err := resolveOne(ctx, question, p, opts, extractor, now)
		if err != nil {
			return ResolveResult{}, err
		}
		switch {
		case found:
			set(name, value, conf)
		case p.Default != "":
			set(name, p.Default, ConfidenceDefault)
		case p.Required:
			res.Needs = append(res.Needs, name)
		default:
			// optional, unresolved → omit silently
		}
	}

	if resolvedAny {
		res.Confidence = minConf
	}
	return res, nil
}

// resolveOne resolves a single param, returning (value, confidence, found).
func resolveOne(ctx context.Context, question string, p Param, opts ResolveOptions, extractor ParamExtractor, now time.Time) (string, float64, bool, error) {
	// Tier 1 — canonical-typed: deterministic.
	if p.Type == ParamTypeTimeWindow {
		if token, ok := MatchTimeWindowToken(question); ok {
			return string(token), ConfidenceDeterministic, true, nil
		}
		return "", 0, false, nil
	}

	// Tier 2 — constrained: resolve the allowed value space, then extract.
	allowed, err := allowedValues(ctx, p, opts.Values)
	if err != nil {
		return "", 0, false, err
	}
	if p.Type == ParamTypeEnum && len(allowed) == 0 {
		// A dynamic enum whose value source is unavailable cannot be constrained
		// or safely extracted — abstain.
		return "", 0, false, nil
	}

	// Tier 2 (enum/bounds) and Tier 3 (bare) both go through the extractor; the
	// difference is whether `allowed` is non-empty.
	er, err := extractor.Extract(ctx, question, p, allowed)
	if err != nil {
		return "", 0, false, err
	}
	if !er.Found {
		return "", 0, false, nil
	}
	// Enforce the constraint: a constrained extraction must land in `allowed`.
	if len(allowed) > 0 && !contains(allowed, er.Value) {
		return "", 0, false, nil
	}
	return er.Value, er.Confidence, true, nil
}

// allowedValues returns the constrained value space for a param: its static
// proto enum values, or a dynamic enum's runtime values resolved via the
// ValuesProvider. Returns nil for non-enum params (bare/bounded extraction).
func allowedValues(ctx context.Context, p Param, vp ValuesProvider) ([]string, error) {
	if len(p.EnumValues) > 0 {
		return p.EnumValues, nil
	}
	if p.ValuesSource != "" {
		if vp == nil {
			return nil, nil
		}
		return vp.Values(ctx, p.ValuesSource)
	}
	return nil, nil
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
