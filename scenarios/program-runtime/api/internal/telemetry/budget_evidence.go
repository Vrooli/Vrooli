package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"program-runtime/internal/budgets"

	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

// BindingCallEvidence contains only the measured call fields needed to
// correlate one budget observation with the invocation ledger.
type BindingCallEvidence struct {
	InvocationID string
	BindingID    string
	Outcome      string
	Latency      time.Duration
	LatencyKnown bool
}

// ProgramLatencyEvidence is an allowlisted projection of persisted lifecycle
// and binding events. Model phase and thermal state are deliberately explicit
// unknowns until trusted telemetry supplies them.
type ProgramLatencyEvidence struct {
	ProgramID         string
	Status            string
	OccurredAt        time.Time
	Total             time.Duration
	TotalKnown        bool
	BindingCalls      []BindingCallEvidence
	BindingPhase      time.Duration
	BindingPhaseKnown bool
	ModelPhase        time.Duration
	ModelPhaseKnown   bool
	ThermalState      string
	ThermalStateKnown bool
}

// LatencyObservation adapts measured fields to the existing quality gate.
// Missing model and thermal measurements stay zero/unknown, so qualification
// rejects the observation until those dimensions are actually measured.
func (e ProgramLatencyEvidence) LatencyObservation() budgets.LatencyObservation {
	thermal := e.ThermalState
	if !e.ThermalStateKnown || thermal == "" {
		thermal = "unknown"
	}
	modelPhase := e.ModelPhase
	if !e.ModelPhaseKnown {
		modelPhase = 0
	}
	return budgets.LatencyObservation{
		Status: e.Status, ThermalState: thermal, Total: e.Total,
		BindingPhase: e.BindingPhase, ModelPhase: modelPhase,
	}
}

type latencyEventReader interface {
	listBudgetLatencyEvents(context.Context, time.Time, time.Time) ([]budgetLatencyEvent, error)
}

type budgetLatencyEvent struct {
	ProgramID    string
	Kind         telemetryv1.EventKind
	OccurredAt   time.Time
	BindingID    string
	InvocationID string
	Outcome      string
	TotalMillis  optionalProtoInt64
	CallMillis   optionalProtoInt64
}

// optionalProtoInt64 reads protojson int64 values, which are encoded as JSON
// strings, while also accepting a numeric representation from older fixtures.
type optionalProtoInt64 struct {
	Value   int64
	Present bool
}

func (v *optionalProtoInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var value int64
	if len(data) > 0 && data[0] == '"' {
		var encoded string
		if err := json.Unmarshal(data, &encoded); err != nil {
			return err
		}
		parsed, err := strconv.ParseInt(encoded, 10, 64)
		if err != nil {
			return err
		}
		value = parsed
	} else if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	v.Value, v.Present = value, true
	return nil
}

// persistedLatencyFields intentionally excludes caller identity, error detail,
// source, inputs, and every unrelated event field. The outbox query also limits
// rows to terminal lifecycle and measured binding kinds before decoding.
type persistedLatencyFields struct {
	ExecutionWallTimeMS  optionalProtoInt64 `json:"execution_wall_time_ms"`
	BindingID            string             `json:"binding_id"`
	BindingInvocationID  string             `json:"binding_invocation_id"`
	BindingCallLatencyMS optionalProtoInt64 `json:"binding_call_latency_ms"`
	BindingCallOutcome   string             `json:"binding_call_outcome"`
}

func (r *outboxRepository) listBudgetLatencyEvents(ctx context.Context, since, until time.Time) ([]budgetLatencyEvent, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("telemetry outbox repository is unavailable")
	}
	if since.IsZero() || until.IsZero() || !since.Before(until) {
		return nil, errors.New("latency evidence window must be a non-empty [since, until) range")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT program_id, kind, occurred_at, payload FROM event_outbox WHERE program_id <> '' AND occurred_at >= ? AND occurred_at < ? AND kind IN (?, ?) ORDER BY program_id, occurred_at, event_id`,
		formatTime(since), formatTime(until), telemetryv1.EventKind_PROGRAM_SUCCEEDED, telemetryv1.EventKind_PROGRAM_FAILED)
	if err != nil {
		return nil, fmt.Errorf("read measured latency events: %w", err)
	}
	defer rows.Close()
	var out []budgetLatencyEvent
	programIDs := make(map[string]struct{})
	for rows.Next() {
		var event budgetLatencyEvent
		var kind int32
		var occurredAt, payload string
		if err := rows.Scan(&event.ProgramID, &kind, &occurredAt, &payload); err != nil {
			return nil, fmt.Errorf("scan measured latency event: %w", err)
		}
		event.Kind = telemetryv1.EventKind(kind)
		event.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt)
		if err != nil {
			return nil, fmt.Errorf("parse measured latency event timestamp: %w", err)
		}
		fields, err := decodeLatencyFields(payload)
		if err != nil {
			return nil, err
		}
		event.TotalMillis = fields.ExecutionWallTimeMS
		programIDs[event.ProgramID] = struct{}{}
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate measured latency events: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close terminal latency events: %w", err)
	}
	if len(programIDs) == 0 {
		return out, nil
	}
	ids := make([]string, 0, len(programIDs))
	for id := range programIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	// Fetch all retained measured calls for the completed runs, regardless of
	// call timestamp. A call can precede the terminal event by hours and still
	// belong to a run whose completion falls inside the requested window.
	const batchSize = 500
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		args := make([]any, 0, len(batch)+1)
		args = append(args, telemetryv1.EventKind_BINDING_CALL_MEASURED)
		for _, id := range batch {
			args = append(args, id)
		}
		bindingRows, err := r.db.QueryContext(ctx, `SELECT program_id, occurred_at, payload FROM event_outbox WHERE kind = ? AND program_id IN (`+placeholders+`) ORDER BY program_id, occurred_at, event_id`, args...)
		if err != nil {
			return nil, fmt.Errorf("read binding latency events: %w", err)
		}
		for bindingRows.Next() {
			var event budgetLatencyEvent
			var occurredAt, payload string
			if err := bindingRows.Scan(&event.ProgramID, &occurredAt, &payload); err != nil {
				bindingRows.Close()
				return nil, fmt.Errorf("scan binding latency event: %w", err)
			}
			event.Kind = telemetryv1.EventKind_BINDING_CALL_MEASURED
			event.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt)
			if err != nil {
				bindingRows.Close()
				return nil, fmt.Errorf("parse binding latency event timestamp: %w", err)
			}
			fields, err := decodeLatencyFields(payload)
			if err != nil {
				bindingRows.Close()
				return nil, err
			}
			event.BindingID = fields.BindingID
			event.InvocationID = fields.BindingInvocationID
			event.Outcome = fields.BindingCallOutcome
			event.CallMillis = fields.BindingCallLatencyMS
			out = append(out, event)
		}
		if err := bindingRows.Err(); err != nil {
			bindingRows.Close()
			return nil, fmt.Errorf("iterate binding latency events: %w", err)
		}
		if err := bindingRows.Close(); err != nil {
			return nil, fmt.Errorf("close binding latency events: %w", err)
		}
	}
	return out, nil
}

func decodeLatencyFields(payload string) (persistedLatencyFields, error) {
	var fields persistedLatencyFields
	if err := json.Unmarshal([]byte(payload), &fields); err != nil {
		return persistedLatencyFields{}, fmt.Errorf("decode allowlisted latency fields: %w", err)
	}
	return fields, nil
}

// ReadBudgetLatencyEvidence reads only completed lifecycle and measured
// binding events from the persistent outbox. It does not expose their original
// payloads and does not infer missing model or thermal measurements.
func (s *Store) ReadBudgetLatencyEvidence(ctx context.Context, since, until time.Time) ([]ProgramLatencyEvidence, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("telemetry store is unavailable")
	}
	reader, ok := s.repo.(latencyEventReader)
	if !ok {
		return nil, errors.New("budget latency evidence requires the persistent telemetry outbox")
	}
	events, err := reader.listBudgetLatencyEvents(ctx, since, until)
	if err != nil {
		return nil, err
	}
	executions := make(map[string]*ProgramLatencyEvidence)
	bindingCalls := make(map[string][]BindingCallEvidence)
	bindingInvocationIDs := make(map[string]map[string]struct{})
	for _, event := range events {
		switch event.Kind {
		case telemetryv1.EventKind_PROGRAM_SUCCEEDED, telemetryv1.EventKind_PROGRAM_FAILED:
			if _, duplicate := executions[event.ProgramID]; duplicate {
				return nil, fmt.Errorf("program %q has duplicate terminal lifecycle events in latency window", event.ProgramID)
			}
			status := "success"
			if event.Kind == telemetryv1.EventKind_PROGRAM_FAILED {
				status = "failure"
			}
			total, known, err := durationFromOptionalMillis(event.TotalMillis)
			if err != nil {
				return nil, fmt.Errorf("program %q execution duration: %w", event.ProgramID, err)
			}
			executions[event.ProgramID] = &ProgramLatencyEvidence{
				ProgramID: event.ProgramID, Status: status, OccurredAt: event.OccurredAt,
				Total: total, TotalKnown: known, ThermalState: "unknown", ThermalStateKnown: false,
			}
		case telemetryv1.EventKind_BINDING_CALL_MEASURED:
			if event.BindingID == "" || event.InvocationID == "" {
				return nil, fmt.Errorf("program %q has a measured binding event without binding and invocation IDs", event.ProgramID)
			}
			latency, known, err := durationFromOptionalMillis(event.CallMillis)
			if err != nil {
				return nil, fmt.Errorf("program %q binding %q duration: %w", event.ProgramID, event.BindingID, err)
			}
			seen := bindingInvocationIDs[event.ProgramID]
			if seen == nil {
				seen = make(map[string]struct{})
				bindingInvocationIDs[event.ProgramID] = seen
			}
			if _, duplicate := seen[event.InvocationID]; duplicate {
				return nil, fmt.Errorf("program %q has duplicate binding invocation ID %q", event.ProgramID, event.InvocationID)
			}
			seen[event.InvocationID] = struct{}{}
			bindingCalls[event.ProgramID] = append(bindingCalls[event.ProgramID], BindingCallEvidence{
				InvocationID: event.InvocationID, BindingID: event.BindingID, Outcome: event.Outcome,
				Latency: latency, LatencyKnown: known,
			})
		}
	}

	out := make([]ProgramLatencyEvidence, 0, len(executions))
	for _, execution := range executions {
		execution.BindingCalls = bindingCalls[execution.ProgramID]
		if len(execution.BindingCalls) > 0 {
			phase := time.Duration(0)
			known := true
			for _, call := range execution.BindingCalls {
				if !call.LatencyKnown || call.Latency > time.Duration(math.MaxInt64)-phase {
					known = false
					break
				}
				phase += call.Latency
			}
			if known {
				execution.BindingPhase, execution.BindingPhaseKnown = phase, true
			}
		}
		out = append(out, *execution)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProgramID < out[j].ProgramID })
	return out, nil
}

// QualifyPersistedBudgetLatencyEvidence sends repository-backed measurements
// through the shared budgets gate. It deliberately propagates qualification
// errors: unknown model phase or thermal state must not become a candidate
// deadline summary merely because the values came from persistent storage.
func (s *Store) QualifyPersistedBudgetLatencyEvidence(ctx context.Context, since, until time.Time, policy budgets.LatencyEvidencePolicy) (budgets.StratifiedLatencySummary, error) {
	evidence, err := s.ReadBudgetLatencyEvidence(ctx, since, until)
	if err != nil {
		return budgets.StratifiedLatencySummary{}, err
	}
	observations := make([]budgets.LatencyObservation, 0, len(evidence))
	for _, execution := range evidence {
		observations = append(observations, execution.LatencyObservation())
	}
	return budgets.QualifyLatencyEvidence(observations, policy)
}

func durationFromOptionalMillis(value optionalProtoInt64) (time.Duration, bool, error) {
	if !value.Present {
		return 0, false, nil
	}
	if value.Value < 0 || value.Value > math.MaxInt64/int64(time.Millisecond) {
		return 0, false, fmt.Errorf("invalid millisecond duration %d", value.Value)
	}
	return time.Duration(value.Value) * time.Millisecond, true, nil
}
