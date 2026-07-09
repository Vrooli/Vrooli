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
	payloadFinishedAt           = "finished_at"
	payloadProgress             = "progress"
	payloadPlanRef              = "plan_ref"
	payloadReplanNeeded         = "replan_needed"
	payloadResolution           = "resolution"
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

func (p RoundPayloadView) FinishedAt() string {
	value, _ := p.String(payloadFinishedAt)
	return value
}

func (p RoundPayloadView) SetPhaseResult(result PhaseResult) {
	p.set(resultEnvelopeKey, result)
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
	raw, ok := p.get(payloadResolution)
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
