package supervision

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const supervisionProgramName = "agent-manager.supervision-evaluate"

type declaredProgramRunner interface {
	RunDeclaredProgram(context.Context, *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error)
}

type ProgramRuntimeEvaluator struct {
	resolve  func(context.Context) (string, error)
	client   *http.Client
	runner   declaredProgramRunner
	policies *PolicyStore
}

func NewProgramRuntimeEvaluator(policies ...*PolicyStore) *ProgramRuntimeEvaluator {
	evaluator := &ProgramRuntimeEvaluator{
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "program-runtime")
		},
		client: &http.Client{Timeout: 65 * time.Second},
	}
	if len(policies) > 0 {
		evaluator.policies = policies[0]
	}
	return evaluator
}

type supervisionEnvelope struct {
	Status  string `json:"status"`
	Signals struct {
		Disposition       string   `json:"disposition"`
		Classification    string   `json:"classification"`
		Confidence        *float64 `json:"confidence"`
		Abstained         bool     `json:"abstained"`
		RecommendedAction string   `json:"recommended_action"`
		NextCursor        *string  `json:"next_cursor"`
		CursorReset       bool     `json:"cursor_reset"`
		WakeCondition     struct {
			Kind         string `json:"kind"`
			AfterSeconds int64  `json:"after_seconds"`
		} `json:"wake_condition"`
		PolicyVersion     string         `json:"policy_version"`
		InferenceCalls    int            `json:"inference_calls"`
		InferenceIdentity map[string]any `json:"inference_identity"`
	} `json:"signals"`
	Evidence []string `json:"evidence"`
}

func (e *ProgramRuntimeEvaluator) Evaluate(ctx context.Context, input EvaluationInput) (*domainpb.WatchDecision, error) { // [REQ:REQ-P2-010]
	if input.Watch == nil || input.Watch.GetSpec() == nil {
		return nil, fmt.Errorf("program evaluator requires a watch spec")
	}
	requestInput, err := e.programInput(ctx, input)
	if err != nil {
		return nil, err
	}
	structured, err := structpb.NewStruct(requestInput)
	if err != nil {
		return nil, fmt.Errorf("encode supervision program input: %w", err)
	}
	runner := e.runner
	if runner == nil {
		base, resolveErr := e.resolve(ctx)
		if resolveErr != nil {
			return unavailableProgramDecision(input, "program_runtime_unavailable"), nil
		}
		runner = libraryconnect.NewLibraryServiceClient(e.client, strings.TrimRight(base, "/"))
	}
	expectedDigest := ""
	if e.policies != nil {
		record, err := e.policies.Get(ctx, input.Watch.GetSpec().GetPolicyVersion())
		if err != nil {
			return nil, err
		}
		expectedDigest = record.Policy.EvaluatorDigest
		if expectedDigest == "" {
			reader, ok := runner.(interface {
				GetLibrary(context.Context, *connect.Request[libraryv1.GetLibraryRequest]) (*connect.Response[libraryv1.GetLibraryResponse], error)
			})
			if !ok {
				return nil, fmt.Errorf("program runner cannot resolve evaluator identity")
			}
			artifact, err := reader.GetLibrary(ctx, connect.NewRequest(&libraryv1.GetLibraryRequest{Name: supervisionProgramName}))
			if err != nil {
				return unavailableProgramDecision(input, "evaluator_identity_unavailable"), nil
			}
			expectedDigest = artifact.Msg.GetProgram().GetContentDigest()
		}
		if err := e.policies.BindEvaluator(ctx, record.Policy.Version, expectedDigest); err != nil {
			return nil, err
		}
	}
	response, err := runner.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{
		Name: supervisionProgramName, ExpectedDigest: expectedDigest, Inputs: structured, Provenance: programsv1.Provenance_PROVENANCE_AGENT,
	}))
	if err != nil || response.Msg.GetProgram() == nil || !response.Msg.GetTerminal() {
		return unavailableProgramDecision(input, "program_runtime_unavailable"), nil
	}
	if response.Msg.GetProgram().GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
		return unavailableProgramDecision(input, "program_execution_unavailable"), nil
	}
	var envelope supervisionEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(response.Msg.GetProgram().GetStdout())), &envelope); err != nil {
		return nil, fmt.Errorf("decode supervision program envelope: %w", err)
	}
	if envelope.Signals.PolicyVersion != input.Watch.GetSpec().GetPolicyVersion() {
		return nil, fmt.Errorf("supervision program policy version mismatch: got %q want %q", envelope.Signals.PolicyVersion, input.Watch.GetSpec().GetPolicyVersion())
	}
	if envelope.Signals.InferenceCalls < 0 || envelope.Signals.InferenceCalls > 1 {
		return nil, fmt.Errorf("supervision program exceeded inference-call contract")
	}
	if envelope.Status != "ok" && envelope.Status != "unavailable" && envelope.Status != "refused" {
		return nil, fmt.Errorf("supervision program failed with status %q", envelope.Status)
	}
	if envelope.Signals.InferenceCalls > 0 && envelope.Status == "ok" && envelope.Signals.Disposition != "unavailable" && e.policies != nil {
		if err := e.policies.BindInference(ctx, input.Watch.GetSpec().GetPolicyVersion(), envelope.Signals.InferenceIdentity); err != nil {
			return unavailableProgramDecision(input, "inference_identity_mismatch"), nil
		}
	}
	decision := &domainpb.WatchDecision{
		Disposition:       dispositionFromProgram(envelope.Signals.Disposition),
		Classification:    envelope.Signals.Classification,
		RecommendedAction: actionFromProgram(envelope.Signals.RecommendedAction),
		EvidenceIds:       boundedStrings(envelope.Evidence, 20),
	}
	if decision.GetDisposition() == domainpb.WatchDisposition_WATCH_DISPOSITION_UNSPECIFIED {
		return nil, fmt.Errorf("supervision program returned unknown disposition %q", envelope.Signals.Disposition)
	}
	// Semantic classification cannot override the run owner's terminal state.
	if decision.GetDisposition() == domainpb.WatchDisposition_WATCH_DISPOSITION_TERMINAL {
		if !input.Watch.GetSpec().GetTriggers().GetTerminal() || len(input.Subjects) == 0 {
			return nil, fmt.Errorf("terminal decision requires an enabled complete cohort")
		}
		if len(input.Subjects) != len(input.Watch.GetSpec().GetSubjects()) {
			return nil, fmt.Errorf("terminal decision requires every watched subject")
		}
		expected := map[string]bool{}
		for _, subject := range input.Watch.GetSpec().GetSubjects() {
			expected[subject.GetRunId()] = true
		}
		for _, subject := range input.Subjects {
			if !subject.Terminal || !expected[subject.RunID] {
				return nil, fmt.Errorf("terminal decision contradicted authoritative run state: %s", subject.RunID)
			}
		}
	}
	if envelope.Signals.Confidence != nil {
		if math.IsNaN(*envelope.Signals.Confidence) || math.IsInf(*envelope.Signals.Confidence, 0) || *envelope.Signals.Confidence < 0 || *envelope.Signals.Confidence > 1 {
			return nil, fmt.Errorf("supervision program returned invalid confidence")
		}
		decision.Confidence = *envelope.Signals.Confidence
	}
	if envelope.Signals.CursorReset && envelope.Signals.NextCursor != nil {
		return nil, fmt.Errorf("supervision program returned both cursor reset and next cursor")
	}
	if err := validateProgramCursor(input, envelope); err != nil {
		return nil, err
	}
	switch envelope.Signals.WakeCondition.Kind {
	case "immediate", "terminal":
		decision.NextWakeAt = timestamppb.New(input.Now.UTC())
	case "after":
		decision.NextWakeAt = timestamppb.New(input.Now.UTC().Add(time.Duration(envelope.Signals.WakeCondition.AfterSeconds) * time.Second))
	default:
		return nil, fmt.Errorf("supervision program returned unknown wake condition %q", envelope.Signals.WakeCondition.Kind)
	}
	return decision, nil
}

func (e *ProgramRuntimeEvaluator) programInput(ctx context.Context, input EvaluationInput) (map[string]any, error) {
	triggers := input.Watch.GetSpec().GetTriggers()
	if triggers == nil {
		return nil, fmt.Errorf("program evaluator requires watch triggers")
	}
	quietSeconds := int64(30)
	if triggers.GetQuietTime().IsValid() && triggers.GetQuietTime().AsDuration() > 0 {
		quietSeconds = max(1, int64(triggers.GetQuietTime().AsDuration()/time.Second))
	}
	eventThreshold := max(1, int(triggers.GetEventCount()))
	frictionThreshold := triggers.GetFrictionScore()
	if frictionThreshold <= 0 {
		frictionThreshold = 1
	}
	events := make([]any, 0, len(input.Events))
	for _, event := range input.Events {
		events = append(events, map[string]any{"event_id": event.ID.String(), "run_id": event.RunID.String(), "sequence": event.Sequence, "event_type": string(event.EventType)})
	}
	runs := make([]any, 0, len(input.Subjects))
	friction := make([]any, 0, len(input.Subjects))
	selectedFriction := []FrictionSummary{}
	for _, subject := range input.Subjects {
		runs = append(runs, map[string]any{"friction_unavailable": subject.FrictionUnavailable, "run_id": subject.RunID, "status": subject.Status, "blocked": strings.EqualFold(subject.Status, "blocked"), "needs_review": strings.EqualFold(subject.Status, "needs_review")})
		selectedFriction = append(selectedFriction, subject.Friction...)
	}
	sort.SliceStable(selectedFriction, func(i, j int) bool {
		if selectedFriction[i].Score == selectedFriction[j].Score {
			return selectedFriction[i].EvidenceID < selectedFriction[j].EvidenceID
		}
		return selectedFriction[i].Score > selectedFriction[j].Score
	})
	for _, episode := range selectedFriction[:min(16, len(selectedFriction))] {
		friction = append(friction, map[string]any{"evidence_id": episode.EvidenceID, "score": episode.Score, "pattern": episode.Pattern, "fingerprint": episode.Fingerprint, "owner": episode.Owner})
	}

	history := []any{}
	if previous := input.Watch.GetLastDecision(); previous != nil {
		history = append(history, map[string]any{"decision_id": previous.GetDecisionId(), "disposition": strings.TrimPrefix(strings.ToLower(previous.GetDisposition().String()), "watch_disposition_"), "classification": previous.GetClassification()})
	}
	deadlineReached := triggers.GetDeadline().IsValid() && !input.Now.Before(triggers.GetDeadline().AsTime())
	quietReached := false
	if triggers.GetQuietTime().IsValid() && triggers.GetQuietTime().AsDuration() > 0 {
		lastActivity := input.Watch.GetUpdatedAt().AsTime()
		if len(input.Events) > 0 {
			lastActivity = input.Events[len(input.Events)-1].Timestamp
		}
		quietReached = !input.Now.Before(lastActivity.Add(triggers.GetQuietTime().AsDuration()))
	}
	allowedActions := []any{"observe", "park", "escalate", "wake_parent"}
	if e.policies != nil {
		policyRecord, err := e.policies.Get(ctx, input.Watch.GetSpec().GetPolicyVersion())
		if err != nil {
			return nil, fmt.Errorf("resolve immutable supervision policy: %w", err)
		}
		allowedActions = make([]any, 0, len(policyRecord.Policy.AllowedActions))
		for _, action := range policyRecord.Policy.AllowedActions {
			allowedActions = append(allowedActions, action)
		}
	}
	return map[string]any{
		"events": events, "friction_episodes": friction, "run_summaries": runs, "prior_decisions": history,
		"policy": map[string]any{
			"version": input.Watch.GetSpec().GetPolicyVersion(), "event_count_threshold": eventThreshold,
			"friction_threshold": frictionThreshold, "quiet_seconds": quietSeconds,
			"event_count_enabled": triggers.GetEventCount() > 0, "friction_enabled": triggers.GetFrictionScore() > 0,
			"terminal_enabled": triggers.GetTerminal(), "deadline_reached": deadlineReached, "quiet_reached": quietReached,
			"allowed_actions": allowedActions,
		},
		"current_cursor": input.Watch.GetCursor().GetToken(), "proposed_next_cursor": input.ProposedCursor,
		"cursor_reset_required": input.Reset, "reset_reason": fmt.Sprintf("retention_generation_changed:%d:%d", input.ResetFrom, input.ResetTo),
		"allow_inference": true, "input_byte_budget": 32768,
	}, nil
}

func validateProgramCursor(input EvaluationInput, envelope supervisionEnvelope) error {
	disposition := envelope.Signals.Disposition
	if disposition == "cursor_reset" {
		if !envelope.Signals.CursorReset || envelope.Signals.NextCursor != nil {
			return fmt.Errorf("supervision program returned invalid reset cursor contract")
		}
		return nil
	}
	if envelope.Signals.CursorReset || envelope.Signals.NextCursor == nil {
		return fmt.Errorf("supervision program omitted next cursor for disposition %q", disposition)
	}
	want := input.ProposedCursor
	if disposition == "unavailable" {
		want = input.Watch.GetCursor().GetToken()
	}
	if *envelope.Signals.NextCursor != want {
		return fmt.Errorf("supervision program cursor mismatch for %q", disposition)
	}
	return nil
}

func unavailableProgramDecision(input EvaluationInput, classification string) *domainpb.WatchDecision {
	return &domainpb.WatchDecision{Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_UNAVAILABLE, Classification: classification, RecommendedAction: domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE, NextWakeAt: timestamppb.New(input.Now.UTC().Add(30 * time.Second))}
}

func dispositionFromProgram(value string) domainpb.WatchDisposition {
	return map[string]domainpb.WatchDisposition{"quiet": domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET, "signal": domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, "terminal": domainpb.WatchDisposition_WATCH_DISPOSITION_TERMINAL, "cursor_reset": domainpb.WatchDisposition_WATCH_DISPOSITION_CURSOR_RESET, "unavailable": domainpb.WatchDisposition_WATCH_DISPOSITION_UNAVAILABLE}[value]
}

func actionFromProgram(value string) domainpb.WatchActionKind {
	return map[string]domainpb.WatchActionKind{"observe": domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE, "nudge": domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, "park": domainpb.WatchActionKind_WATCH_ACTION_KIND_PARK, "continue": domainpb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE, "stop": domainpb.WatchActionKind_WATCH_ACTION_KIND_STOP, "escalate": domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE, "wake_parent": domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT}[value]
}

func boundedStrings(values []string, limit int) []string {
	if len(values) > limit {
		values = values[:limit]
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value[:min(len(value), 128)])
	}
	return out
}
