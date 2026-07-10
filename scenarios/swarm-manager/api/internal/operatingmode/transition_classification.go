package operatingmode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Classification-on-transition: guards route on fields of the completed
// round's resolved output, but the natural output of an execute round is a
// handoff, not a routing enum. Instead of a dedicated classifier *phase*
// bridging that gap, a transition declares a classification contract
// (TransitionClassification) and the engine derives the routing field at the
// edge using the same resolution-ladder mechanics it already owns:
//
//   - short-circuit (not_required): the round emitted the field directly — the
//     declaration is satisfied deterministically and no classifier runs;
//   - L1 (deterministic extraction): the source object carries the field
//     inline at `<from>.<field>` — no model call;
//   - L2 (classifier fallback): the source JSON is classified against the
//     declared enum, abstaining rather than guessing;
//   - abstain: the round routes to needs_attention — a routing decision is
//     never fabricated and never crashes the loop.
//
// The derived value is written into the round payload at the declared field
// path, so the ordinary guard evaluator (which the loader fed with expanded
// eq-guards over that field) routes on it with no special-casing.

// TransitionClassificationOutcome is the result of deriving a transition's
// routing field for one completed round.
type TransitionClassificationOutcome struct {
	Field      string
	Value      string
	Outcome    ResolutionOutcome
	Layer      ResolutionLayer
	Violations []string
	Notes      []string
}

// Abstained reports whether the ladder could not honestly derive the routing
// field — the round must park in needs_attention instead of routing.
func (o TransitionClassificationOutcome) Abstained() bool {
	return o.Outcome == ResolutionAbstained
}

// Record projects the outcome into the durable PhaseResolutionRecord shape,
// marked as a transition classification via ClassifiedField/ClassifiedValue.
func (o TransitionClassificationOutcome) Record() PhaseResolutionRecord {
	rec := PhaseResolutionRecord{
		Outcome:            o.Outcome,
		Layer:              o.Layer,
		ChosenMessageIndex: -1,
		Violations:         append([]string(nil), o.Violations...),
		Notes:              append([]string(nil), o.Notes...),
		ClassifiedField:    o.Field,
		ClassifiedValue:    o.Value,
	}
	if o.Abstained() {
		rec.Missing = []string{o.Field}
	}
	return rec
}

// AbstainReason renders the single-line, operator-facing explanation used as
// the round error when classification abstains.
func (o TransitionClassificationOutcome) AbstainReason() string {
	reason := fmt.Sprintf("transition classification abstained: routing field %q could not be derived from the round's output", o.Field)
	if len(o.Violations) > 0 {
		reason += " (violations: " + strings.Join(o.Violations, ", ") + ")"
	}
	return reason
}

// allowsValue reports whether v is a member of the contract's declared enum.
func (c TransitionClassification) allowsValue(v string) bool {
	for _, member := range c.Enum {
		if member == v {
			return true
		}
	}
	return false
}

// deriveTransitionValue runs the deterministic rungs of the edge ladder —
// short-circuit (field already emitted) and L1 (inline on the source object at
// `<from>.<field>`) — over the given lookup. It is pure (no model call, no
// mutation) so the load-time example-run walk and the runtime path share it.
//
// ok=true with layer==ResolutionLayerNone means the field was emitted directly
// (classification not required); layer==ResolutionLayerExtract means L1
// derived it. ok=false with violations means a deterministic source carried an
// explicit out-of-enum value — a contract violation that is a hard abstain,
// never handed to the classifier. ok=false with no violations means the field
// is simply absent and L2 may try.
func deriveTransitionValue(contract TransitionClassification, lookup FieldLookup) (value string, layer ResolutionLayer, violations []string, ok bool) {
	if raw, present := lookup.Lookup(contract.Field); present && raw != nil {
		v := renderGuardValue(raw)
		if contract.allowsValue(v) {
			return v, ResolutionLayerNone, nil, true
		}
		return "", ResolutionLayerNone, []string{fmt.Sprintf("%s: emitted value %q not in declared enum", contract.Field, v)}, false
	}
	if strings.TrimSpace(contract.From) != "" {
		inline := contract.From + "." + contract.Field
		if raw, present := lookup.Lookup(inline); present && raw != nil {
			v := renderGuardValue(raw)
			if contract.allowsValue(v) {
				return v, ResolutionLayerExtract, nil, true
			}
			return "", ResolutionLayerExtract, []string{fmt.Sprintf("%s: value %q not in declared enum", inline, v)}, false
		}
	}
	return "", ResolutionLayerNone, nil, false
}

// classifyTransitionRouting derives the completed round's transition-owned
// routing field against the process registry's mode definition, with the full
// ladder (L2 allowed). Returns nil when the phase declares no classification
// contract. It never returns an error: an underivable field is an honest
// abstain the caller routes to needs_attention.
func (s *Service) classifyTransitionRouting(ctx context.Context, round *RoundEnvelope, envelope map[string]any) *TransitionClassificationOutcome {
	bundle, def, err := s.definitionBundleForRound(*round)
	if err != nil {
		return nil
	}
	return s.classifyTransitionRoutingWithResolver(ctx, def, bundle.Definition, round, envelope, true)
}

// classifyTransitionRoutingForDef is the definition-explicit variant used by
// simulation (which walks draft/registered definitions deterministically, so
// allowClassifier=false keeps simulation free of live model calls). On a
// non-abstain the derived value is hoisted into the round payload at the
// declared field path — feeding the expanded route guards — and the durable
// classification record is written to the payload in every case.
func (s *Service) classifyTransitionRoutingForDef(ctx context.Context, def Definition, round *RoundEnvelope, envelope map[string]any, allowClassifier bool) *TransitionClassificationOutcome {
	return s.classifyTransitionRoutingWithResolver(ctx, def, DefinitionFor, round, envelope, allowClassifier)
}

func (s *Service) classifyTransitionRoutingWithResolver(ctx context.Context, def Definition, resolve func(Mode) (Definition, error), round *RoundEnvelope, envelope map[string]any, allowClassifier bool) *TransitionClassificationOutcome {
	// A delegated round classifies against the sub-mode's phase contract: the
	// sub-phase owns the classified edge (e.g. the drain deriving `progress`
	// from the handoff), and the derived value is what both the sub-route and
	// the parent's guards read.
	_, phaseDef, err := effectiveRoundExecutionWithResolver(def, *round, resolve)
	if err != nil || phaseDef.TransitionClassification == nil {
		return nil
	}
	contract := *phaseDef.TransitionClassification
	lookup := roundClassificationLookup(round.Payload, envelope)

	outcome := TransitionClassificationOutcome{Field: contract.Field}
	value, layer, violations, ok := deriveTransitionValue(contract, lookup)
	switch {
	case ok && layer == ResolutionLayerNone:
		outcome.Value = value
		outcome.Outcome = ResolutionNotRequired
		outcome.Notes = []string{fmt.Sprintf("routing field %q already emitted by the round; classification not required", contract.Field)}
	case ok:
		outcome.Value = value
		outcome.Outcome = ResolutionResolved
		outcome.Layer = layer
		outcome.Notes = []string{fmt.Sprintf("derived %s = %s deterministically from %s", contract.Field, value, contract.From)}
	case len(violations) > 0:
		outcome.Outcome = ResolutionAbstained
		outcome.Layer = layer
		outcome.Violations = violations
		outcome.Notes = []string{"an explicit out-of-enum routing value is a contract violation; the classifier never overrides it"}
	default:
		s.classifyTransitionFallback(ctx, contract, round.Payload, envelope, allowClassifier, &outcome)
	}

	if !outcome.Abstained() {
		setPayloadField(round.Payload, contract.Field, outcome.Value)
		if stored, ok := payloadEnvelopeMap(round.Payload); ok {
			setPayloadField(stored, contract.Field, outcome.Value)
			round.Payload[resultEnvelopeKey] = stored
		}
	}
	MutableRoundPayload(round).SetTransitionClassification(outcome.Record())
	return &outcome
}

// classifyTransitionFallback is the L2 rung of the edge ladder: reconstruct
// the routing field from the source JSON via the schema-steered classifier,
// abstaining honestly when the classifier is unavailable, errors, abstains, or
// answers outside the declared enum.
func (s *Service) classifyTransitionFallback(ctx context.Context, contract TransitionClassification, payload, envelope map[string]any, allowClassifier bool, outcome *TransitionClassificationOutcome) {
	outcome.Outcome = ResolutionAbstained
	if !allowClassifier || s.classifier == nil {
		outcome.Notes = append(outcome.Notes, "no classifier available for L2 fallback")
		return
	}
	raw, rawOK := classificationSource(payload, envelope, contract)
	if !rawOK {
		outcome.Notes = append(outcome.Notes, fmt.Sprintf("classification source %q is absent from the round output", defaultString(contract.From, resultEnvelopeKey)))
		return
	}
	outcome.Layer = ResolutionLayerClassifier
	res, err := s.classifier.ClassifyField(ctx, ClassifyFieldRequest{
		RawOutput:   raw,
		FieldPath:   contract.Field,
		FieldType:   "string",
		Description: contract.Description,
		Enum:        append([]string(nil), contract.Enum...),
	})
	switch {
	case err != nil:
		outcome.Notes = append(outcome.Notes, "classify "+contract.Field+": "+err.Error())
	case !res.Found || strings.TrimSpace(res.Value) == "":
		outcome.Notes = append(outcome.Notes, "classifier abstained on "+contract.Field)
	case !contract.allowsValue(strings.TrimSpace(res.Value)):
		outcome.Violations = append(outcome.Violations, fmt.Sprintf("%s: classifier value %q not in declared enum", contract.Field, strings.TrimSpace(res.Value)))
	default:
		outcome.Value = strings.TrimSpace(res.Value)
		outcome.Outcome = ResolutionRecovered
		outcome.Notes = append(outcome.Notes, "classifier resolved "+contract.Field+" = "+outcome.Value)
	}
}

// classificationSource renders the L2 classification input: the JSON of the
// contract's From field value, or the whole structured-result envelope when
// From is empty. ok=false when the source is absent (nothing to classify).
func classificationSource(payload, envelope map[string]any, contract TransitionClassification) (string, bool) {
	lookup := roundClassificationLookup(payload, envelope)
	var source any
	if strings.TrimSpace(contract.From) != "" {
		raw, present := lookup.Lookup(contract.From)
		if !present || raw == nil {
			return "", false
		}
		source = raw
	} else {
		env := envelope
		if env == nil {
			stored, ok := payloadEnvelopeMap(payload)
			if !ok {
				return "", false
			}
			env = stored
		}
		source = env
	}
	data, err := json.Marshal(source)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// roundClassificationLookup resolves a dotted field path against the round's
// payload with envelope fallback: the hoisted top-level payload keys are
// checked first (progress/verdict/…, plus any previously derived routing
// field), then the raw resolution envelope — which, unlike the typed
// PhaseResult round-trip persisted on the payload, preserves emitted fields
// the runtime does not model (the whole point of deriving at the edge). When
// no raw envelope is supplied, the one persisted on the payload is used.
func roundClassificationLookup(payload, envelope map[string]any) FieldLookup {
	lookups := []FieldLookup{NewMapFieldLookup(payload)}
	if envelope == nil {
		if stored, ok := payloadEnvelopeMap(payload); ok {
			envelope = stored
		}
	}
	if envelope != nil {
		lookups = append(lookups, NewMapFieldLookup(envelope))
	}
	return chainedFieldLookup(lookups)
}

// payloadEnvelopeMap returns the structured-result envelope stored on the
// round payload, coerced to a generic map.
func payloadEnvelopeMap(payload map[string]any) (map[string]any, bool) {
	raw, ok := payload[resultEnvelopeKey]
	if !ok || raw == nil {
		return nil, false
	}
	return coerceToMap(raw)
}

// chainedFieldLookup tries each lookup in order and returns the first present
// value.
type chainedFieldLookup []FieldLookup

func (c chainedFieldLookup) Lookup(path string) (any, bool) {
	for _, lookup := range c {
		if value, present := lookup.Lookup(path); present {
			return value, true
		}
	}
	return nil, false
}

// setPayloadField writes a derived scalar into the payload map at a dotted
// path, coercing any struct-valued intermediate (e.g. a ProgressState stored
// under `progress`) into a generic map so the write merges instead of
// clobbering. This is where the derived routing value lands so the ordinary
// guard evaluator — which reads the round payload — routes on it.
func setPayloadField(payload map[string]any, path, value string) {
	segments := strings.Split(strings.TrimSpace(path), ".")
	if len(segments) == 0 || payload == nil {
		return
	}
	dst := payload
	for _, segment := range segments[:len(segments)-1] {
		if strings.TrimSpace(segment) == "" {
			return
		}
		next, ok := coerceToMap(dst[segment])
		if !ok {
			next = map[string]any{}
		}
		dst[segment] = next
		dst = next
	}
	leaf := strings.TrimSpace(segments[len(segments)-1])
	if leaf == "" {
		return
	}
	dst[leaf] = value
}
