package sttengine

import "audio-tools/internal/stt/egress"

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
	"low_footprint":        {},
}

// Validate enforces the manifest invariants the schema cannot express in pure
// JSON-schema terms (cross-field rules). registry_test.go additionally checks
// the manifest against schema.json.
