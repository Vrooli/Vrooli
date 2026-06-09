package aisearch

// override.go is the QUERY-TIME override channel: the seam that lets a single
// request vary the query-time tuning factors (rerank policy, shortlist, floor
// band) without re-constructing the Service or touching the stored vectors. It
// exists so the search-hub sweep can full-factorial the cheap query-time tier in
// one pass (one request per arm, no reindex) and so an operator can A/B a knob
// live — both through the same typed, bounded, policy-gated path.
//
// The split this file enforces is the one tuning.go names: ONLY QueryTime
// factors are overridable. The IndexTime factors (engine, embed_model,
// embed_task_prefix) change the embedded representation and need a reindex, so
// they have no field here — the schema makes "vary an index-time factor per
// request" unrepresentable, not merely rejected.
//
// Security posture is layered. This package owns the INNER layer: a Service
// honors overrides only when its construction-time OverridePolicy permits them
// (the secure default — no policy — denies all), and even a permitted value is
// always clamped to its taxonomy-legal range and the rerank_blend⇒rerank_enabled
// invariant is re-enforced, so a buggy or hostile caller cannot push a factor
// out of bounds. The OUTER layer (a control-token match + a per-environment
// experiment flag) lives at the provider's request handler — see the cli-health
// search handler — because authentication is a transport concern, not a
// read-path one.

// SearchOverrides carries per-request overrides for the QUERY-TIME tuning
// factors only. Each field is a pointer so an UNSET factor (nil) is distinct
// from one explicitly set to its zero value (false / 0): an unset field falls
// through to the Service's constructed default, while a set one overrides it.
// This mirrors the wire shape proto routing.SearchOverrides (Phase 3), whose
// fields are all `optional` for exactly this presence semantics. There is
// deliberately no engine/embed field: those are index-time (see tuning.go).
// The json tags use the snake_case factor keys (matching tuning.go / search.json
// and the proto field names) so the same struct serializes as the override
// transport header (see override_transport.go). Pointers + omitempty keep an
// unset factor off the wire entirely.
type SearchOverrides struct {
	// RerankEnabled gates the rerank pass for this request.
	RerankEnabled *bool `json:"rerank_enabled,omitempty"`
	// RerankBlend fuses the rerank order with retrieval via RRF instead of
	// reordering outright. Honored only when the effective rerank_enabled is true
	// (the Service drops it otherwise — the same invariant TuningConfig.Validate
	// enforces).
	RerankBlend *bool `json:"rerank_blend,omitempty"`
	// RerankShortlist is the over-fetch depth handed to the reranker. Clamped to
	// [MinRerankShortlist, MaxRerankShortlist].
	RerankShortlist *int `json:"rerank_shortlist,omitempty"`
	// FloorMaxGap / FloorHardFloor override the relevance-floor band for this
	// request. Clamped to [0,1]; 0 keeps the regime default (FloorForMethodLeg).
	FloorMaxGap    *float64 `json:"floor_max_gap,omitempty"`
	FloorHardFloor *float64 `json:"floor_hard_floor,omitempty"`
}

// IsZero reports whether no override field is set — used by a handler to skip
// the override path entirely (and its telemetry) for an ordinary request.
func (o SearchOverrides) IsZero() bool {
	return o.RerankEnabled == nil && o.RerankBlend == nil && o.RerankShortlist == nil &&
		o.FloorMaxGap == nil && o.FloorHardFloor == nil
}

// OverridePolicy decides WHICH per-request overrides a Service honors. A provider
// supplies one to reject overrides outright, allow them all, or permit only a
// custom subset. It is the inner, construction-time half of the override gate:
// the Service consults it once per call, then ALWAYS clamps whatever it permits
// to the taxonomy-legal ranges and re-enforces rerank_blend⇒rerank_enabled — so
// a policy decides membership, never bounds. A nil policy (the default) denies
// everything: a Service built without one never honors an override, no matter
// what a request carries.
type OverridePolicy interface {
	// Permit returns the subset of o the provider allows to take effect. A
	// returned field that is nil is dropped (falls through to the Service
	// default); a non-nil field is applied (after the Service's own clamp).
	Permit(o SearchOverrides) SearchOverrides
}

// denyOverrides drops every override. It is the policy a Service falls back to
// when none is configured, so "secure by default" is the literal zero value.
type denyOverrides struct{}

func (denyOverrides) Permit(SearchOverrides) SearchOverrides { return SearchOverrides{} }

// DenyOverrides returns the reject-all policy (the secure default). A Service
// with this policy ignores every per-request override.
func DenyOverrides() OverridePolicy { return denyOverrides{} }

// allowOverrides passes every override through (still clamped by the Service).
type allowOverrides struct{}

func (allowOverrides) Permit(o SearchOverrides) SearchOverrides { return o }

// AllowOverrides returns the permit-all policy: every query-time factor is
// tunable per request. The Service still clamps each permitted value to its
// taxonomy-legal range and enforces the blend invariant, and the provider's
// request handler still gates the whole channel on a control token + experiment
// flag, so "allow all" here means "all query-time factors are in play", not
// "unauthenticated and unbounded".
func AllowOverrides() OverridePolicy { return allowOverrides{} }

// OverrideBool / OverrideInt / OverrideFloat build the pointer fields of a
// SearchOverrides ergonomically (Go has no literal for &true). A consumer
// translating the proto presence fields uses them directly:
//
//	ov := SearchOverrides{}
//	if m := req.Overrides; m != nil {
//	    if m.RerankEnabled != nil { ov.RerankEnabled = OverrideBool(*m.RerankEnabled) }
//	}
func OverrideBool(b bool) *bool        { return &b }
func OverrideInt(i int) *int           { return &i }
func OverrideFloat(f float64) *float64 { return &f }

// searchSettings is the resolved per-call option set (currently just the
// overrides). It is the target the SearchOption closures write into.
type searchSettings struct {
	overrides *SearchOverrides
}

// SearchOption customizes a single Service.Search call. The only option today is
// WithOverrides; the variadic form keeps the common no-option call
// (`s.Search(ctx, q)`) unchanged and leaves room for future per-call knobs
// without another signature break.
type SearchOption func(*searchSettings)

// WithOverrides supplies the per-request query-time overrides for one Search
// call. They are subject to the Service's OverridePolicy and clamping; with the
// default (deny) policy they are ignored.
func WithOverrides(o SearchOverrides) SearchOption {
	return func(s *searchSettings) { s.overrides = &o }
}

// effectiveParams is the resolved query-time configuration for one Search call:
// the Service's constructed defaults, overlaid by any policy-permitted,
// clamped overrides. The read path (vectorSearch / autoSearch) reads these
// instead of the Service fields directly, so a per-call override changes only
// this request's behavior and never mutates the shared Service.
type effectiveParams struct {
	rerankEnabled bool
	rerankBlend   bool
	shortlist     int
	applyFloor    bool
	floor         FloorConfig
}

// resolveEffective computes the per-call query-time parameters from the Service
// defaults and the options. Overrides are applied only when an OverridePolicy is
// configured and permits them, and every permitted value is clamped to its
// taxonomy-legal range; the rerank_blend⇒rerank_enabled invariant is re-enforced
// after the overlay so blend can never be on with rerank off.
func (s *Service) resolveEffective(opts ...SearchOption) effectiveParams {
	eff := effectiveParams{
		rerankEnabled: s.rerankEnabled,
		rerankBlend:   s.rerankBlend,
		shortlist:     s.shortlist,
		applyFloor:    s.applyFloor,
		floor:         s.floor,
	}

	var set searchSettings
	for _, opt := range opts {
		if opt != nil {
			opt(&set)
		}
	}
	if set.overrides == nil || set.overrides.IsZero() || s.overridePolicy == nil {
		return eff
	}

	permitted := s.overridePolicy.Permit(*set.overrides)
	if permitted.RerankEnabled != nil {
		eff.rerankEnabled = *permitted.RerankEnabled
	}
	if permitted.RerankBlend != nil {
		eff.rerankBlend = *permitted.RerankBlend
	}
	if permitted.RerankShortlist != nil {
		eff.shortlist = clampInt(*permitted.RerankShortlist, MinRerankShortlist, MaxRerankShortlist)
	}
	if permitted.FloorMaxGap != nil {
		eff.floor.MaxGap = clampFloat(*permitted.FloorMaxGap, 0, 1)
	}
	if permitted.FloorHardFloor != nil {
		eff.floor.HardFloor = clampFloat(*permitted.FloorHardFloor, 0, 1)
	}
	// Re-enforce the package invariant: a blend with rerank off is a no-op that
	// would only confuse the floor-regime classification, so drop it.
	if !eff.rerankEnabled {
		eff.rerankBlend = false
	}
	return eff
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
