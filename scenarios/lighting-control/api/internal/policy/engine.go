package policy

import (
	"errors"
	"fmt"
	"math"
	"time"

	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrInvalidManualRequest = errors.New("invalid manual preset request")

// SourceInput is a narrow, caller-populated snapshot. The policy package never
// fetches or retains source telemetry; adapters own source collection.
type SourceInput struct {
	Name         string
	State        policyv1.EvidenceState
	ObservedAt   time.Time
	SourceTime   time.Time
	Reason       string
	Measurements map[string]float64
}

// Inputs contains only the two upstream facts used by automatic lighting.
type Inputs struct {
	Now           time.Time
	SystemMonitor SourceInput
	AgentManager  SourceInput
}

// State is the minimal replayable state needed for debounce, load hysteresis,
// and a durable manual request. Callers may persist it as one serialized value.
type State struct {
	ActivePreset policyv1.PresetKind `json:"active_preset"`
	LoadActive   bool                `json:"load_active"`
	Candidate    policyv1.PresetKind `json:"candidate"`
	CandidateAt  time.Time           `json:"candidate_at"`
	ManualPreset policyv1.PresetKind `json:"manual_preset"`
	ManualExpiry time.Time           `json:"manual_expiry,omitempty"`
}

type Engine struct{ Config *policyv1.PolicyConfig }

func (e Engine) Validate() error {
	c := e.Config
	if c == nil {
		return errors.New("policy config is required")
	}
	if !validPreset(c.DefaultPreset) || c.DefaultPreset == policyv1.PresetKind_PRESET_KIND_UNSPECIFIED {
		return errors.New("default preset must be a supported preset")
	}
	if c.SourceMaxAgeSeconds < 5 || c.SourceMaxAgeSeconds > 86400 {
		return errors.New("source max age must be between 5 and 86400 seconds")
	}
	if c.LoadEnterPercent > 100 || c.LoadExitPercent > 100 || c.LoadExitPercent > c.LoadEnterPercent {
		return errors.New("load thresholds must be 0..100 with exit <= enter")
	}
	if c.BrightnessCapPercent > 100 || c.StaleFallbackBrightnessPercent > 100 {
		return errors.New("brightness percentages must be 0..100")
	}
	seen := map[policyv1.PresetKind]bool{}
	for _, p := range c.Presets {
		if p == nil || !validPreset(p.Kind) || p.Kind == policyv1.PresetKind_PRESET_KIND_UNSPECIFIED {
			return errors.New("presets must have supported kinds")
		}
		if seen[p.Kind] {
			return fmt.Errorf("duplicate preset %s", p.Kind)
		}
		seen[p.Kind] = true
		if p.BrightnessCapPercent > 100 {
			return fmt.Errorf("preset %s brightness cap exceeds 100", p.Kind)
		}
		if p.Color != nil && (p.Color.Red > 255 || p.Color.Green > 255 || p.Color.Blue > 255) {
			return fmt.Errorf("preset %s color channels must be 0..255", p.Kind)
		}
	}
	if c.StaleFallbackColor != nil && (c.StaleFallbackColor.Red > 255 || c.StaleFallbackColor.Green > 255 || c.StaleFallbackColor.Blue > 255) {
		return errors.New("fallback color channels must be 0..255")
	}
	return nil
}

func (e Engine) SetManual(state State, kind policyv1.PresetKind, expires time.Time, now time.Time) (State, error) {
	if err := e.Validate(); err != nil {
		return state, err
	}
	if !validPreset(kind) || kind == policyv1.PresetKind_PRESET_KIND_UNSPECIFIED {
		return state, fmt.Errorf("%w: preset must be supported", ErrInvalidManualRequest)
	}
	if !expires.IsZero() && !expires.After(now) {
		return state, fmt.Errorf("%w: expiry must be in the future", ErrInvalidManualRequest)
	}
	state.ManualPreset, state.ManualExpiry = kind, expires
	return state, nil
}

func (e Engine) ClearManual(state State) State {
	state.ManualPreset = policyv1.PresetKind_PRESET_KIND_UNSPECIFIED
	state.ManualExpiry = time.Time{}
	return state
}

func (e Engine) Evaluate(state State, in Inputs) (State, *policyv1.EffectivePolicy, error) {
	if err := e.Validate(); err != nil {
		return state, nil, err
	}
	if in.Now.IsZero() {
		return state, nil, errors.New("evaluation time is required")
	}
	if state.ManualPreset != policyv1.PresetKind_PRESET_KIND_UNSPECIFIED && (state.ManualExpiry.IsZero() || in.Now.Before(state.ManualExpiry)) {
		p := e.preset(state.ManualPreset)
		return state, e.effective(p, state.ManualPreset, "manual request has priority", in.Now, nil, true, state.ManualExpiry, false), nil
	}
	if state.ManualPreset != policyv1.PresetKind_PRESET_KIND_UNSPECIFIED {
		state = e.ClearManual(state)
	}

	load, agents := assess(in.SystemMonitor, in.Now, e.Config.SourceMaxAgeSeconds), assess(in.AgentManager, in.Now, e.Config.SourceMaxAgeSeconds)
	kind, reason, fallback := e.choose(&state, load, agents)
	if state.ActivePreset == policyv1.PresetKind_PRESET_KIND_UNSPECIFIED {
		state.ActivePreset = kind
	}
	if fallback {
		state.ActivePreset, state.Candidate, state.CandidateAt = kind, policyv1.PresetKind_PRESET_KIND_UNSPECIFIED, time.Time{}
	} else if kind != state.ActivePreset {
		if state.Candidate != kind {
			state.Candidate, state.CandidateAt = kind, in.Now
		}
		if in.Now.Sub(state.CandidateAt) >= time.Duration(e.Config.DebounceSeconds)*time.Second {
			state.ActivePreset, state.Candidate = kind, policyv1.PresetKind_PRESET_KIND_UNSPECIFIED
			state.CandidateAt = time.Time{}
		}
	} else {
		state.Candidate = policyv1.PresetKind_PRESET_KIND_UNSPECIFIED
		state.CandidateAt = time.Time{}
	}
	// Before debounce completes, preserve the currently effective preset.
	if state.ActivePreset != kind {
		kind = state.ActivePreset
		reason = "awaiting policy debounce"
	}
	p := e.preset(kind)
	var evidence []*policyv1.EvidenceSource
	if needsLoad(e.Config.DefaultPreset) {
		evidence = append(evidence, load.proto())
	}
	if needsAgents(e.Config.DefaultPreset) {
		evidence = append(evidence, agents.proto())
	}
	return state, e.effective(p, kind, reason, in.Now, evidence, false, time.Time{}, fallback), nil
}

type assessed struct {
	input  SourceInput
	state  policyv1.EvidenceState
	reason string
}

func assess(s SourceInput, now time.Time, maxAge uint32) assessed {
	a := assessed{input: s, state: s.State, reason: s.Reason}
	if a.state == policyv1.EvidenceState_EVIDENCE_STATE_UNSPECIFIED {
		a.state = policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE
	}
	if a.state == policyv1.EvidenceState_EVIDENCE_STATE_FRESH {
		stamp := s.SourceTime
		if stamp.IsZero() {
			stamp = s.ObservedAt
		}
		if stamp.IsZero() {
			a.state = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
			a.reason = "fresh evidence has no source timestamp"
		} else if stamp.After(now.Add(5 * time.Second)) {
			a.state = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
			a.reason = "source timestamp is in the future"
		} else if now.Sub(stamp) > time.Duration(maxAge)*time.Second {
			a.state = policyv1.EvidenceState_EVIDENCE_STATE_STALE
			a.reason = "source evidence exceeded maximum age"
		}
	}
	if a.reason == "" {
		a.reason = a.state.String()
	}
	return a
}
func (a assessed) usable() bool { return a.state == policyv1.EvidenceState_EVIDENCE_STATE_FRESH }
func (a assessed) proto() *policyv1.EvidenceSource {
	out := &policyv1.EvidenceSource{Source: a.input.Name, State: a.state, Reason: a.reason, Measurements: a.input.Measurements}
	if !a.input.ObservedAt.IsZero() {
		out.ObservedAt = timestamppb.New(a.input.ObservedAt)
	}
	if !a.input.SourceTime.IsZero() {
		out.SourceTimestamp = timestamppb.New(a.input.SourceTime)
	}
	return out
}

func (e Engine) choose(s *State, load, agents assessed) (policyv1.PresetKind, string, bool) {
	base := e.Config.DefaultPreset
	static := policyv1.PresetKind_PRESET_KIND_STATIC
	if base == policyv1.PresetKind_PRESET_KIND_OFF || base == static {
		return base, "configured static policy", false
	}
	if base == policyv1.PresetKind_PRESET_KIND_LOAD {
		if !load.usable() {
			return static, "load evidence is " + load.state.String() + ": " + load.reason, true
		}
		v, ok := utilization(load.input.Measurements)
		if !ok {
			return static, "load evidence lacks a recognized measured utilization", true
		}
		if !s.LoadActive && v >= float64(e.Config.LoadEnterPercent) {
			s.LoadActive = true
		} else if s.LoadActive && v <= float64(e.Config.LoadExitPercent) {
			s.LoadActive = false
		}
		if s.LoadActive {
			return policyv1.PresetKind_PRESET_KIND_LOAD, "measured utilization crossed load threshold", false
		}
		return static, "measured utilization below load threshold", false
	}
	if base == policyv1.PresetKind_PRESET_KIND_WORK {
		if !agents.usable() {
			return static, "activity evidence is " + agents.state.String() + ": " + agents.reason, true
		}
		v, ok := metric(agents.input.Measurements, "active_runs")
		if !ok {
			return static, "activity evidence lacks active run count", true
		}
		if v > 0 {
			return policyv1.PresetKind_PRESET_KIND_WORK, "active runs are reported", false
		}
		return static, "no active runs reported", false
	}
	if base == policyv1.PresetKind_PRESET_KIND_HEALTH {
		if !load.usable() {
			return static, "health evidence is " + load.state.String() + ": " + load.reason, true
		}
		v, ok := healthFailure(load.input.Measurements)
		if !ok {
			return static, "health evidence lacks an explicit health-failure signal", true
		}
		if v > 0 {
			return policyv1.PresetKind_PRESET_KIND_HEALTH, "source reported explicit health failure", false
		}
		return static, "source reports no health failure", false
	}
	// Combined uses explicit failure, then load hysteresis, then active work.
	if base == policyv1.PresetKind_PRESET_KIND_COMBINED {
		if !load.usable() || !agents.usable() {
			bad := load
			if bad.usable() {
				bad = agents
			}
			return static, "combined evidence is " + bad.state.String() + ": " + bad.reason, true
		}
		if h, ok := healthFailure(load.input.Measurements); ok && h > 0 {
			return policyv1.PresetKind_PRESET_KIND_HEALTH, "source reported explicit health failure", false
		}
		if v, ok := utilization(load.input.Measurements); ok {
			if !s.LoadActive && v >= float64(e.Config.LoadEnterPercent) {
				s.LoadActive = true
			} else if s.LoadActive && v <= float64(e.Config.LoadExitPercent) {
				s.LoadActive = false
			}
			if s.LoadActive {
				return policyv1.PresetKind_PRESET_KIND_LOAD, "measured utilization crossed load threshold", false
			}
		} else {
			return static, "combined load evidence lacks recognized utilization", true
		}
		if n, ok := metric(agents.input.Measurements, "active_runs"); ok && n > 0 {
			return policyv1.PresetKind_PRESET_KIND_WORK, "active runs are reported", false
		}
		return static, "no combined policy signal is active", false
	}
	return static, "unsupported automatic preset; using static fallback", true
}

func utilization(m map[string]float64) (float64, bool) {
	var max float64
	found := false
	for _, key := range []string{"cpu_percent", "cpu_usage_percent", "gpu_usage_percent", "utilization_percent", "load_percent"} {
		if value, ok := metric(m, key); ok {
			if value < 0 || value > 100 {
				continue
			}
			if !found || value > max {
				max = value
			}
			found = true
		}
	}
	return max, found
}

func healthFailure(m map[string]float64) (float64, bool) {
	if value, ok := metric(m, "health_failed", "unhealthy"); ok {
		return value, true
	}
	if level, ok := metric(m, "memory_pressure_level"); ok {
		if level >= 3 {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func metric(m map[string]float64, keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok && !math.IsNaN(v) && !math.IsInf(v, 0) {
			return v, true
		}
	}
	return 0, false
}

func needsLoad(k policyv1.PresetKind) bool {
	return k == policyv1.PresetKind_PRESET_KIND_LOAD || k == policyv1.PresetKind_PRESET_KIND_HEALTH || k == policyv1.PresetKind_PRESET_KIND_COMBINED
}

func needsAgents(k policyv1.PresetKind) bool {
	return k == policyv1.PresetKind_PRESET_KIND_WORK || k == policyv1.PresetKind_PRESET_KIND_COMBINED
}

func (e Engine) preset(kind policyv1.PresetKind) *policyv1.Preset {
	for _, p := range e.Config.Presets {
		if p != nil && p.Kind == kind && p.Enabled {
			return p
		}
	}
	if kind == policyv1.PresetKind_PRESET_KIND_STATIC {
		return &policyv1.Preset{Kind: kind, Label: "Static fallback", Color: e.Config.StaleFallbackColor, BrightnessCapPercent: e.Config.StaleFallbackBrightnessPercent, Enabled: true}
	}
	if kind == policyv1.PresetKind_PRESET_KIND_OFF {
		return &policyv1.Preset{Kind: kind, Label: "Off", BrightnessCapPercent: 0, Enabled: true}
	}
	return &policyv1.Preset{Kind: kind, Label: kind.String(), Enabled: true}
}

func (e Engine) effective(p *policyv1.Preset, kind policyv1.PresetKind, reason string, now time.Time, evidence []*policyv1.EvidenceSource, manual bool, expiry time.Time, fallback bool) *policyv1.EffectivePolicy {
	brightness := p.BrightnessCapPercent
	if e.Config.BrightnessCapPercent < brightness {
		brightness = e.Config.BrightnessCapPercent
	}
	if kind == policyv1.PresetKind_PRESET_KIND_OFF {
		brightness = 0
	}
	out := &policyv1.EffectivePolicy{Preset: kind, Color: p.Color, BrightnessPercent: brightness, EffectId: p.EffectId, Reason: reason, EvaluatedAt: timestamppb.New(now), Evidence: evidence, ManualPriority: manual, StaticFallback: fallback}
	if !expiry.IsZero() {
		out.RequestExpiresAt = timestamppb.New(expiry)
	}
	return out
}

func validPreset(k policyv1.PresetKind) bool {
	return k >= policyv1.PresetKind_PRESET_KIND_OFF && k <= policyv1.PresetKind_PRESET_KIND_STATIC
}
