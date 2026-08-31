// Package sttengine is the single source of truth for which speech-to-text
// engines audio-tools can run and what each one is capable of. One checked-in
// JSON manifest (manifest.json, validated against schema.json by
// registry_test.go) describes every engine with positive capability
// declarations plus the speaker-isolation axis. The strategy selector derives
// strategy eligibility from each engine's strategies[]; the post-recognition
// egress gate derives stage applicability from provides.*. No code branches on
// an engine id — callers ask the registry, so adding an engine is a manifest
// edit (plus a resource folder + provider adapter for kind=local_resource).
//
// Layering: this package owns static capability facts only. The active engine
// selection + the operator tunables live in the persisted StreamConfig (proto
// admin API). Behaviour = active selection -> look up capabilities here ->
// derive strategy + egress stages.
package sttengine

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"audio-tools/internal/stt/egress"
)

//go:embed manifest.json
var manifestJSON []byte

// schemaJSON is the JSON-schema doc for manifest.json. It is embedded so the
// validation test can assert it is well-formed and ships alongside the binary
// as the authoritative shape reference.
//
//go:embed schema.json
var schemaJSON []byte

// Engine kinds — the discriminator that generalises the Local/BYOK/Vrooli
// tiers. Only local_resource engines own a resources/<name>/ folder.
const (
	KindLocalResource = "local_resource"
	KindBYOKAPI       = "byok_api"
	KindVrooliHosted  = "vrooli_hosted"
)

// Confidence-signal identifiers an engine may declare in provides.
const (
	SignalNoSpeechProb = "no_speech_prob"
	SignalAvgLogProb   = "avg_logprob"
)

// Provides is an engine's positive capability declaration. Every field is a
// capability the engine HAS — there are no negative "disable" flags.
type Provides struct {
	NativeStreaming   bool     `json:"nativeStreaming"`
	BuiltinVad        bool     `json:"builtinVad"`
	ConfidenceSignals []string `json:"confidenceSignals"`
	WordTimestamps    bool     `json:"wordTimestamps"`
}

// HasConfidenceSignals reports whether the engine emits the per-segment
// confidence signals the signal-domain egress stage consumes.
func (p Provides) HasConfidenceSignals() bool {
	for _, s := range p.ConfidenceSignals {
		if s == SignalNoSpeechProb || s == SignalAvgLogProb {
			return true
		}
	}
	return false
}

// Requires is what an engine needs from the ingress substrate.
type Requires struct {
	PCM16kMono bool `json:"pcm16kMono"`
}

// Engine is one selectable STT engine.
type Engine struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Kind        string   `json:"kind"`
	Resource    string   `json:"resource"`
	Provides    Provides `json:"provides"`
	Requires    Requires `json:"requires"`
	Strategies  []string `json:"strategies"`
}

// SpeakerMethod is one pluggable speaker-isolation method.
type SpeakerMethod struct {
	BackendResource  string  `json:"backendResource"`
	DefaultThreshold float64 `json:"defaultThreshold"`
	Status           string  `json:"status"`
}

// SpeakerIsolation is the parallel pluggable axis: which method is active and
// the catalog of methods. Swapping the method is a one-field manifest edit.
type SpeakerIsolation struct {
	Active  string                   `json:"active"`
	Methods map[string]SpeakerMethod `json:"methods"`
}

// Manifest is the decoded manifest.json.
type Manifest struct {
	Engines          []Engine         `json:"engines"`
	SelectionPolicy  SelectionPolicy  `json:"selectionPolicy"`
	SpeakerIsolation SpeakerIsolation `json:"speakerIsolation"`
}

type SelectionPolicy struct {
	HardFilters       []string `json:"hardFilters"`
	RankedPreferences []string `json:"rankedPreferences"`
	TiebreakRank      []string `json:"tiebreakRank"`
	ProbeTTLSeconds   int      `json:"probeTtlSeconds"`
}

// Registry is the loaded, validated engine manifest.
//
// seam: EngineRegistry is the engine-capability seam (SEAMS.md row
// "sttengine.Registry"). Production wires the embedded manifest via Default();
// tests build a Registry from an in-test manifest via New.
type Registry struct {
	manifest Manifest
	byID     map[string]Engine
}

// Default returns the Registry loaded from the embedded manifest.json. It
// panics on a malformed/invalid manifest because a broken capability source is
// a build-time defect, not a runtime condition.
func Default() *Registry {
	r, err := Load(manifestJSON)
	if err != nil {
		panic(fmt.Sprintf("sttengine: embedded manifest invalid: %v", err))
	}
	return r
}

// Load parses and validates a manifest from raw JSON. Tests use it to build a
// Registry from fixture manifests.
func Load(raw []byte) (*Registry, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	r := New(m)
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// New builds a Registry from an already-decoded manifest (no validation).
func New(m Manifest) *Registry {
	byID := make(map[string]Engine, len(m.Engines))
	for _, e := range m.Engines {
		byID[e.ID] = e
	}
	return &Registry{manifest: m, byID: byID}
}

// Engines returns every declared engine in manifest order.
func (r *Registry) Engines() []Engine { return r.manifest.Engines }

// Engine looks up an engine by id.
func (r *Registry) Engine(id string) (Engine, bool) {
	e, ok := r.byID[id]
	return e, ok
}

var ErrNoServiceableEngine = errors.New("no_serviceable_engine")

type Candidate struct {
	Engine   Engine `json:"engine"`
	Verdict  string `json:"verdict"`
	Reason   string `json:"reason"`
	Selected bool   `json:"selected"`
}

type Resolution struct {
	Selected   string      `json:"selected"`
	Candidates []Candidate `json:"candidates"`
}

// EngineFacts are the bounded host observations used by ResolveFacts. They
// are intentionally supplied by the caller so the registry remains portable
// and testable; control-plane probing belongs to the audio-tools integration
// seam, not to this static manifest package.
type EngineFacts struct {
	PlatformSupported bool   `json:"platformSupported"`
	Installed         bool   `json:"installed"`
	Running           bool   `json:"running"`
	Healthy           bool   `json:"healthy"`
	WorkloadCapable   bool   `json:"workloadCapable"`
	Accelerated       bool   `json:"accelerated"`
	SignalUnavailable bool   `json:"signalUnavailable"`
	Reason            string `json:"reason,omitempty"`
}

// Resolve chooses from observed serviceability and declared capabilities. The
// registry owns policy ordering; callers provide the live liveness seam. A
// native-streaming engine is preferred because it avoids buffered fallback,
// while manifest order remains the deterministic tie-break.
func (r *Registry) Resolve(available func(string) bool) (Resolution, error) {
	facts := make(map[string]EngineFacts, len(r.manifest.Engines))
	for _, e := range r.manifest.Engines {
		ok := available == nil || available(e.ID)
		facts[e.ID] = EngineFacts{
			PlatformSupported: true,
			Installed:         ok,
			Running:           ok,
			Healthy:           ok,
			WorkloadCapable:   true,
			Reason:            "serviceability signal: backing engine is available",
		}
	}
	return r.ResolveFacts(facts)
}

// ResolveFacts chooses an engine from explicit host facts. Hard filters reject
// candidates before ranked preferences are applied. The final tie-break is
// manifest order, which is deterministic but never the primary selection
// signal.
func (r *Registry) ResolveFacts(facts map[string]EngineFacts) (Resolution, error) {
	resolution := Resolution{Candidates: make([]Candidate, 0, len(r.manifest.Engines))}
	hardFilters := r.manifest.SelectionPolicy.HardFilters
	if len(hardFilters) == 0 {
		hardFilters = []string{"platform_supported", "resource_serviceable"}
	}
	for _, e := range r.manifest.Engines {
		candidate := Candidate{Engine: e, Verdict: "rejected"}
		fact, ok := facts[e.ID]
		if !ok {
			candidate.Reason = "signal_unavailable: no host facts for candidate"
			resolution.Candidates = append(resolution.Candidates, candidate)
			continue
		}
		for _, filter := range hardFilters {
			var rejected string
			switch filter {
			case "platform_supported":
				if !fact.PlatformSupported {
					rejected = "platform_unsupported: observed platform is unsupported"
				}
			case "resource_serviceable":
				if !fact.Installed {
					rejected = "not_installed: backing resource is not installed"
				} else if !fact.Running || !fact.Healthy {
					rejected = "not_serviceable: observed running=false or healthy=false"
				}
			case "workload_capable":
				if !fact.WorkloadCapable {
					rejected = "capability_mismatch: workload_capable=false"
				}
			}
			if rejected != "" {
				if fact.Reason != "" {
					rejected += " (" + fact.Reason + ")"
				}
				candidate.Reason = rejected
				resolution.Candidates = append(resolution.Candidates, candidate)
				break
			}
		}
		if candidate.Reason != "" {
			continue
		}
		candidate.Verdict = "candidate"
		candidate.Reason = fact.Reason
		if candidate.Reason == "" {
			candidate.Reason = "serviceability signal: backing engine is available"
		}
		if fact.SignalUnavailable {
			candidate.Reason = "signal_unavailable: " + candidate.Reason
		}
		resolution.Candidates = append(resolution.Candidates, candidate)
	}
	selected := -1
	for i := range resolution.Candidates {
		if resolution.Candidates[i].Verdict != "candidate" {
			continue
		}
		if selected < 0 || betterByPolicy(r.manifest.SelectionPolicy, resolution.Candidates[i].Engine, resolution.Candidates[selected].Engine, facts) {
			selected = i
		}
	}
	if selected < 0 {
		return resolution, fmt.Errorf("%w: start a configured STT resource", ErrNoServiceableEngine)
	}
	resolution.Candidates[selected].Selected = true
	resolution.Candidates[selected].Verdict = "selected"
	resolution.Candidates[selected].Reason = selectionReason(r.manifest.SelectionPolicy, resolution.Candidates, selected, facts)
	resolution.Selected = resolution.Candidates[selected].Engine.ID
	return resolution, nil
}

func selectionReason(policy SelectionPolicy, candidates []Candidate, selected int, facts map[string]EngineFacts) string {
	for _, preference := range policy.RankedPreferences {
		if preference == "accelerated" {
			if facts[candidates[selected].Engine.ID].Accelerated {
				return "accelerated=true ranked above non-accelerated candidates"
			}
			return "accelerated=false for all candidates; tiebreak=manifest_order"
		}
		if preference == "native_streaming" && candidates[selected].Engine.Provides.NativeStreaming {
			return "native_streaming=true ranked above buffered candidates"
		}
	}
	return "tiebreak: declared policy then manifest order"
}

func betterByPolicy(policy SelectionPolicy, left, right Engine, facts map[string]EngineFacts) bool {
	for _, preference := range policy.RankedPreferences {
		switch preference {
		case "accelerated":
			if facts[left.ID].Accelerated != facts[right.ID].Accelerated {
				return facts[left.ID].Accelerated
			}
		case "native_streaming":
			if left.Provides.NativeStreaming != right.Provides.NativeStreaming {
				return left.Provides.NativeStreaming
			}
		}
	}
	return false
}

// ResolveEngineID is the compact seam used by pipeline components that need
// only the selected id. It replaces positional defaults.
func (r *Registry) ResolveEngineID(available func(string) bool) string {
	resolution, err := r.Resolve(available)
	if err != nil {
		return ""
	}
	return resolution.Selected
}

// EligibleStrategies returns the strategy whitelist for an engine as raw
// strings (the selector converts to sttchain.StrategyKind — kept string-typed
// here so this package does not import sttchain and risk an import cycle).
// Unknown engine ids return nil so the caller falls back to provider traits.
func (r *Registry) EligibleStrategies(engineID string) []string {
	e, ok := r.byID[engineID]
	if !ok {
		return nil
	}
	out := make([]string, len(e.Strategies))
	copy(out, e.Strategies)
	return out
}

// RequiresPCM reports whether the engine needs the ingress substrate to hand
// it canonical PCM (16 kHz mono s16le), per its manifest requires.pcm16kMono.
// The Segmenter uses this for the Passthrough path: a native-streaming engine
// like Kyutai that requires PCM gets the inbound chunks normalized before they
// reach it, whereas a Passthrough BYOK vendor that decodes for itself (no
// manifest entry) does not. Unknown engine ids return false.
func (r *Registry) RequiresPCM(engineID string) bool {
	e, ok := r.byID[engineID]
	if !ok {
		return false
	}
	return e.Requires.PCM16kMono
}

// EgressParams carries the operator tunables the manifest-derived egress
// stages read. The Segmenter builds it from the resolved StreamConfig +
// pipeline.IsWhisperHallucination.
type EgressParams struct {
	HallucinationFilterEnabled bool
	NoSpeechThreshold          float64
	LogProbThreshold           float64
	IsHallucination            func(string) bool
	// SpeakerIsolation is the active audio-domain identity check, built by the
	// Segmenter from the live SpeakerConfig + resource client. nil omits the
	// audio-domain stage (speaker isolation disabled / off / no resource).
	SpeakerIsolation egress.SpeakerIsolation
}

// EgressStages derives the ordered post-recognition stage list for an engine.
// The stage SET is manifest-derived (capability-gated); the stage TUNABLES come
// from EgressParams. The signal-domain stage is added only when the engine
// declares confidence signals, so a native-streaming engine that reports none
// skips it gracefully. The text-domain stage is engine-agnostic but operator-
// toggleable.
//
// Phase 4 adds the audio-domain speaker stage here, gated on
// ActiveSpeakerMethod + the engine being non-streaming-identity-blind.
func (r *Registry) EgressStages(engineID string, p EgressParams) []egress.Stage {
	var stages []egress.Stage
	if p.HallucinationFilterEnabled {
		stages = append(stages, egress.HallucinationStage{IsHallucination: p.IsHallucination})
	}
	if e, ok := r.byID[engineID]; ok && e.Provides.HasConfidenceSignals() {
		stages = append(stages, egress.ConfidenceStage{
			NoSpeechThreshold: p.NoSpeechThreshold,
			LogProbThreshold:  p.LogProbThreshold,
		})
	}
	// Audio-domain speaker stage runs last (after text + signal domains). It is
	// added only when the manifest declares a non-reserved active isolation
	// method AND the Segmenter supplied a live isolation — engine-independent
	// (operates on audio, not transcript), so every engine that carries segment
	// PCM gets it. The method itself is manifest-selected; swapping it is a
	// one-field manifest edit.
	if p.SpeakerIsolation != nil {
		if _, m, ok := r.ActiveSpeakerMethod(); ok && m.Status != "reserved" {
			stages = append(stages, egress.SpeakerStage{Isolation: p.SpeakerIsolation})
		}
	}
	return stages
}

// ActiveSpeakerMethod returns the manifest's active speaker-isolation method
// name and its config. ok is false when the active method is absent from the
// methods catalog (an invalid manifest the validator would already reject).
func (r *Registry) ActiveSpeakerMethod() (string, SpeakerMethod, bool) {
	name := r.manifest.SpeakerIsolation.Active
	m, ok := r.manifest.SpeakerIsolation.Methods[name]
	return name, m, ok
}

// knownStrategies is the set of strategy ids the manifest may reference. It is
// duplicated from sttchain.Strategy* string values on purpose so this package
// avoids importing sttchain (which would cycle via egress); registry_test.go
// asserts the two sets stay in lockstep.
var knownStrategies = map[string]struct{}{
	"vad_segment":       {},
	"overlap_agree":     {},
	"passthrough":       {},
	"buffered_fallback": {},
}

var knownSelectionSignals = map[string]struct{}{
	"platform_supported":   {},
	"resource_serviceable": {},
	"workload_capable":     {},
	"accelerated":          {},
	"native_streaming":     {},
}

// Validate enforces the manifest invariants the schema cannot express in pure
// JSON-schema terms (cross-field rules). registry_test.go additionally checks
// the manifest against schema.json.
func (r *Registry) Validate() error {
	if len(r.manifest.Engines) == 0 {
		return fmt.Errorf("manifest must declare at least one engine")
	}
	for _, signal := range append(append([]string{}, r.manifest.SelectionPolicy.HardFilters...), r.manifest.SelectionPolicy.RankedPreferences...) {
		if _, ok := knownSelectionSignals[signal]; !ok {
			return fmt.Errorf("selectionPolicy: unknown signal %q", signal)
		}
	}
	for _, tie := range r.manifest.SelectionPolicy.TiebreakRank {
		if tie != "manifest_order" {
			return fmt.Errorf("selectionPolicy: unknown tiebreak %q", tie)
		}
	}
	if r.manifest.SelectionPolicy.ProbeTTLSeconds < 0 {
		return fmt.Errorf("selectionPolicy.probeTtlSeconds must not be negative")
	}
	seen := make(map[string]struct{}, len(r.manifest.Engines))
	for _, e := range r.manifest.Engines {
		if e.ID == "" {
			return fmt.Errorf("engine has empty id")
		}
		if _, dup := seen[e.ID]; dup {
			return fmt.Errorf("duplicate engine id %q", e.ID)
		}
		seen[e.ID] = struct{}{}
		switch e.Kind {
		case KindLocalResource:
			if e.Resource == "" {
				return fmt.Errorf("engine %q: kind=local_resource requires a resource", e.ID)
			}
		case KindBYOKAPI, KindVrooliHosted:
		default:
			return fmt.Errorf("engine %q: unknown kind %q", e.ID, e.Kind)
		}
		if len(e.Strategies) == 0 {
			return fmt.Errorf("engine %q: must declare at least one strategy", e.ID)
		}
		for _, s := range e.Strategies {
			if _, ok := knownStrategies[s]; !ok {
				return fmt.Errorf("engine %q: unknown strategy %q", e.ID, s)
			}
		}
		// A native-streaming engine is only eligible for passthrough — its
		// stream is decoded server-side by the provider, not chunked by us.
		if e.Provides.NativeStreaming {
			for _, s := range e.Strategies {
				if s != "passthrough" {
					return fmt.Errorf("engine %q declares nativeStreaming but lists non-passthrough strategy %q", e.ID, s)
				}
			}
		}
	}
	si := r.manifest.SpeakerIsolation
	if si.Active == "" {
		return fmt.Errorf("speakerIsolation.active must be set")
	}
	if _, ok := si.Methods[si.Active]; !ok {
		return fmt.Errorf("speakerIsolation.active %q is not in methods", si.Active)
	}
	return nil
}
