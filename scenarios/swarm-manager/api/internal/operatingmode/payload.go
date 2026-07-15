package operatingmode

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	payloadAgentSummary         = "agent_summary"
	payloadBacklogSync          = "backlog_sync"
	payloadBacklogSyncAppliedAt = "backlog_sync_applied_at"
	payloadBacklogSyncPlan      = "backlog_sync_plan"
	payloadCanceledAt           = "canceled_at"
	payloadEvidenceGate         = "evidence_gate"
	payloadFinishedAt           = "finished_at"
	payloadProgress             = "progress"
	payloadPlanRef              = "plan_ref"
	payloadReplanNeeded         = "replan_needed"
	payloadResolution           = "resolution"
	payloadTransitionClass      = "transition_classification"
	payloadVerdict              = "verdict"
)

type RoundPayloadView struct {
	payload map[string]any
}

func RoundPayload(payload map[string]any) RoundPayloadView {
	return RoundPayloadView{payload: payload}
}

func MutableRoundPayload(round *RoundEnvelope) RoundPayloadView {
	round.Payload = ensurePayload(round.Payload)
	return RoundPayloadView{payload: round.Payload}
}

func (p RoundPayloadView) SetAgentSummary(summary string) {
	p.setString(payloadAgentSummary, summary)
}

func (p RoundPayloadView) SetBacklogSync(result BacklogSyncResult) {
	p.set(payloadBacklogSync, result)
}

func (p RoundPayloadView) SetBacklogSyncAppliedAt(value string) {
	p.setString(payloadBacklogSyncAppliedAt, value)
}

func (p RoundPayloadView) SetBacklogSyncPlan(plan *BacklogSyncPlan) {
	if plan != nil {
		p.set(payloadBacklogSyncPlan, plan)
	}
}

func (p RoundPayloadView) BacklogSyncPlan(roundNumber int) (BacklogSyncPlan, error) {
	raw, ok := p.get(payloadBacklogSyncPlan)
	if !ok {
		return BacklogSyncPlan{}, fmt.Errorf("round %03d has no backlog_sync_plan", roundNumber)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return BacklogSyncPlan{}, fmt.Errorf("marshal backlog_sync_plan: %w", err)
	}
	var plan BacklogSyncPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return BacklogSyncPlan{}, fmt.Errorf("parse backlog_sync_plan: %w", err)
	}
	return plan, nil
}

func (p RoundPayloadView) SetCanceledAt(value string) {
	p.setString(payloadCanceledAt, value)
}

func (p RoundPayloadView) SetFinishedAt(value string) {
	p.setString(payloadFinishedAt, value)
}

func (p RoundPayloadView) SetEvidenceGateState(state string) {
	p.setString(payloadEvidenceGate, state)
}

func (p RoundPayloadView) ClearEvidenceGateState() {
	delete(p.payload, payloadEvidenceGate)
}

func (p RoundPayloadView) EvidenceGateState() string {
	value, _ := p.String(payloadEvidenceGate)
	return value
}

func (p RoundPayloadView) FinishedAt() string {
	value, _ := p.String(payloadFinishedAt)
	return value
}

func (p RoundPayloadView) SetPhaseResult(result PhaseResult, envelope map[string]any) {
	if envelope == nil {
		envelope = resultEnvelopeMap(result)
	}
	p.set(resultEnvelopeKey, cloneJSONMap(envelope))
}

// ResolvedOutput returns the round's validated resolved declared output — the
// canonical envelope the resolution ladder produced from the agent's round and
// the source of truth the guards read. ok=false when the round resolved none
// (still running, or an honest abstain). It is the typed result an operation-
// runner completion seam forwards to CommitResult.
func (p RoundPayloadView) ResolvedOutput() (map[string]any, bool) {
	return payloadEnvelopeMap(p.payload)
}

// ResultFieldLookup resolves guards from the canonical persisted envelope,
// with the payload retained only as a compatibility/derived-field projection.
func (p RoundPayloadView) ResultFieldLookup() FieldLookup {
	lookups := make([]FieldLookup, 0, 2)
	if envelope, ok := payloadEnvelopeMap(p.payload); ok {
		lookups = append(lookups, NewMapFieldLookup(envelope))
	}
	lookups = append(lookups, NewMapFieldLookup(p.payload))
	return chainedFieldLookup(lookups)
}

func cloneJSONMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	data, err := json.Marshal(src)
	if err != nil {
		return src
	}
	var clone map[string]any
	if json.Unmarshal(data, &clone) != nil {
		return src
	}
	return clone
}

// validateEnvelopeProjectionConsistency rejects contradictory duplicate
// representations instead of silently choosing one. Hoisted payload fields are
// projections only; the resolved envelope remains the source of truth.
func validateEnvelopeProjectionConsistency(payload map[string]any) error {
	envelope, ok := payloadEnvelopeMap(payload)
	if !ok {
		return nil
	}
	for _, key := range []string{payloadProgress, payloadPlanRef, payloadReplanNeeded, payloadVerdict} {
		envelopeValue, inEnvelope := envelope[key]
		projectionValue, projected := payload[key]
		if !inEnvelope || !projected {
			continue
		}
		if !jsonProjectionContains(projectionValue, envelopeValue) {
			return fmt.Errorf("resolved envelope field %q conflicts with its payload projection", key)
		}
	}
	return nil
}

// jsonProjectionContains compares JSON-shaped values while allowing a typed
// projection to add normalized/default metadata (for example progress.updated_at).
// Every value actually emitted in the envelope must still match exactly.
func jsonProjectionContains(projection, emitted any) bool {
	normalize := func(value any) any {
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		var out any
		if json.Unmarshal(data, &out) != nil {
			return nil
		}
		return out
	}
	var contains func(any, any) bool
	contains = func(projected, source any) bool {
		switch value := source.(type) {
		case map[string]any:
			projectedMap, ok := projected.(map[string]any)
			if !ok {
				return false
			}
			for key, sourceValue := range value {
				projectedValue, ok := projectedMap[key]
				if !ok || !contains(projectedValue, sourceValue) {
					return false
				}
			}
			return true
		case []any:
			projectedSlice, ok := projected.([]any)
			if !ok || len(projectedSlice) != len(value) {
				return false
			}
			for i := range value {
				if !contains(projectedSlice[i], value[i]) {
					return false
				}
			}
			return true
		default:
			return fmt.Sprint(projected) == fmt.Sprint(source)
		}
	}
	return contains(normalize(projection), normalize(emitted))
}

func (p RoundPayloadView) SetProgress(progress ProgressState) {
	p.set(payloadProgress, progress)
}

func (p RoundPayloadView) SetPlanRef(ref PlanRef) {
	p.set(payloadPlanRef, ref)
}

func (p RoundPayloadView) Progress() (ProgressState, bool) {
	raw, ok := p.get(payloadProgress)
	if !ok {
		return ProgressState{}, false
	}
	switch progress := raw.(type) {
	case ProgressState:
		return progress, true
	case map[string]any:
		decision, _ := progress["decision"].(string)
		state := ProgressState{Decision: ProgressDecision(strings.TrimSpace(decision))}
		if state.Validate() == nil {
			return state, true
		}
	}
	return ProgressState{}, false
}

// SetResolution records the resolution-ladder outcome for the round so the CLI
// and UI can surface which rung resolved it (or why it abstained).
func (p RoundPayloadView) SetResolution(record PhaseResolutionRecord) {
	p.set(payloadResolution, record)
}

// Resolution returns the durable resolution-ladder record, if one was written.
func (p RoundPayloadView) Resolution() (PhaseResolutionRecord, bool) {
	return p.resolutionRecord(payloadResolution)
}

// SetTransitionClassification records the classification-on-transition outcome
// — how the round's routing field was derived at the edge (or why derivation
// abstained) — distinct from the phase-output resolution record.
func (p RoundPayloadView) SetTransitionClassification(record PhaseResolutionRecord) {
	p.set(payloadTransitionClass, record)
}

// TransitionClassification returns the durable classification-on-transition
// record, if the completed phase's transition declared a classification.
func (p RoundPayloadView) TransitionClassification() (PhaseResolutionRecord, bool) {
	return p.resolutionRecord(payloadTransitionClass)
}

func (p RoundPayloadView) resolutionRecord(key string) (PhaseResolutionRecord, bool) {
	raw, ok := p.get(key)
	if !ok {
		return PhaseResolutionRecord{}, false
	}
	switch rec := raw.(type) {
	case PhaseResolutionRecord:
		return rec, true
	case map[string]any:
		data, err := json.Marshal(rec)
		if err != nil {
			return PhaseResolutionRecord{}, false
		}
		var out PhaseResolutionRecord
		if err := json.Unmarshal(data, &out); err != nil {
			return PhaseResolutionRecord{}, false
		}
		return out, true
	default:
		return PhaseResolutionRecord{}, false
	}
}

func (p RoundPayloadView) SetReplanNeeded(value bool) {
	if value {
		p.set(payloadReplanNeeded, true)
	}
}

func (p RoundPayloadView) ReplanNeeded() bool {
	replan, _ := p.get(payloadReplanNeeded)
	value, _ := replan.(bool)
	return value
}

func (p RoundPayloadView) SetVerdict(verdict string) {
	p.setString(payloadVerdict, verdict)
}

func (p RoundPayloadView) Verdict() string {
	value, _ := p.String(payloadVerdict)
	return value
}

func (p RoundPayloadView) String(key string) (string, bool) {
	raw, ok := p.get(key)
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	return strings.TrimSpace(value), ok
}

func (p RoundPayloadView) get(key string) (any, bool) {
	if p.payload == nil {
		return nil, false
	}
	raw, ok := p.payload[key]
	return raw, ok
}

func (p RoundPayloadView) set(key string, value any) {
	if p.payload == nil {
		return
	}
	p.payload[key] = value
}

// clear removes the key from the payload entirely. Used by callers that
// need a "missing" semantic distinct from a present-but-nil value (e.g.,
// the auto-start retry marker, where presence is the signal).
func (p RoundPayloadView) clear(key string) {
	if p.payload == nil {
		return
	}
	delete(p.payload, key)
}

func (p RoundPayloadView) setString(key, value string) {
	if strings.TrimSpace(value) != "" {
		p.set(key, strings.TrimSpace(value))
	}
}
