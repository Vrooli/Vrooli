package investigation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	internalbrief "plan-manager/internal/investigationbrief"
	internalpolicy "plan-manager/internal/investigationpolicy"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/investigation"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/investigation/investigationconnect"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

const compositionProgram = "plan-manager.investigate"

type declaredProgramRunner interface {
	RunDeclaredProgram(context.Context, *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error)
}

type ConnectHandler struct {
	policyconnect.UnimplementedInvestigationPolicyServiceHandler
	repository *internalpolicy.Repository
	logger     *log.Logger
	runner     declaredProgramRunner
	brief      internalbrief.Provider
}

func NewConnectHandler(repository *internalpolicy.Repository, logger *log.Logger) *ConnectHandler {
	return &ConnectHandler{repository: repository, logger: logger}
}

func NewConnectHandlerWithRunner(repository *internalpolicy.Repository, logger *log.Logger, runner declaredProgramRunner) *ConnectHandler {
	return &ConnectHandler{repository: repository, logger: logger, runner: runner}
}

func NewConnectHandlerWithRunnerAndBrief(repository *internalpolicy.Repository, logger *log.Logger, runner declaredProgramRunner, brief internalbrief.Provider) *ConnectHandler {
	return &ConnectHandler{repository: repository, logger: logger, runner: runner, brief: brief}
}

func policyRecord(policy internalpolicy.Policy, active bool, sourceScope string, raw []byte) *policyv1.PolicyRecord {
	digest, _ := policy.Digest()
	scopeJSON, _ := json.Marshal(policy.Scope)
	return &policyv1.PolicyRecord{
		SchemaVersion: policy.SchemaVersion,
		Version:       policy.Version,
		Mode:          string(policy.Mode),
		PolicyJson:    string(raw),
		Active:        active,
		Digest:        digest,
		ScopeJson:     string(scopeJSON),
		SourceScope:   sourceScope,
	}
}

func observationScope(observation internalpolicy.Observation) internalpolicy.PolicyScope {
	return internalpolicy.PolicyScope{FamilyID: observation.FamilyID, ExecutionID: observation.ExecutionID, PhaseID: observation.PhaseID}
}

func (h *ConnectHandler) GetPolicy(ctx context.Context, req *connect.Request[policyv1.GetPolicyRequest]) (*connect.Response[policyv1.PolicyRecord], error) {
	scope := internalpolicy.PolicyScope{}
	if req != nil && req.Msg != nil {
		scope = internalpolicy.PolicyScope{FamilyID: req.Msg.GetFamilyId(), ExecutionID: req.Msg.GetExecutionId(), PhaseID: req.Msg.GetPhaseId()}
	}
	policy, sourceScope, err := h.repository.ResolvePolicy(ctx, scope)
	if err != nil {
		return nil, policyError(err)
	}
	raw, _ := json.Marshal(policy)
	return connect.NewResponse(policyRecord(policy, true, sourceScope, raw)), nil
}

func (h *ConnectHandler) PutPolicy(ctx context.Context, req *connect.Request[policyv1.PutPolicyRequest]) (*connect.Response[policyv1.PolicyRecord], error) {
	if req == nil || req.Msg == nil || req.Msg.GetPolicyJson() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("policy_json is required"))
	}
	var policy internalpolicy.Policy
	if err := json.Unmarshal([]byte(req.Msg.GetPolicyJson()), &policy); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.repository.PutPolicyIfVersion(ctx, policy, req.Msg.GetActive(), req.Msg.GetExpectedVersion()); err != nil {
		return nil, policyError(err)
	}
	if h.logger != nil {
		h.logger.Printf("investigation policy stored version=%s mode=%s active=%t", policy.Version, policy.Mode, req.Msg.GetActive())
	}
	raw, _ := json.Marshal(policy)
	return connect.NewResponse(policyRecord(policy, req.Msg.GetActive(), policy.Scope.Key(), raw)), nil
}

func (h *ConnectHandler) PreviewTrigger(ctx context.Context, req *connect.Request[policyv1.PreviewTriggerRequest]) (*connect.Response[policyv1.TriggerDecision], error) {
	if req == nil || req.Msg == nil || req.Msg.GetObservationJson() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation_json is required"))
	}
	var observation internalpolicy.Observation
	if err := json.Unmarshal([]byte(req.Msg.GetObservationJson()), &observation); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	policy, _, err := h.repository.ResolvePolicy(ctx, observationScope(observation))
	if err != nil {
		return nil, policyError(err)
	}
	decision, err := policy.Evaluate(observation)
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(triggerDecisionProto(decision)), nil
}

func (h *ConnectHandler) RecordTrigger(ctx context.Context, req *connect.Request[policyv1.RecordTriggerRequest]) (*connect.Response[policyv1.TriggerRecord], error) {
	if req == nil || req.Msg == nil || req.Msg.GetObservationJson() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation_json is required"))
	}
	var observation internalpolicy.Observation
	if err := json.Unmarshal([]byte(req.Msg.GetObservationJson()), &observation); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	policy, _, err := h.repository.ResolvePolicy(ctx, observationScope(observation))
	if err != nil {
		return nil, policyError(err)
	}
	decision, err := policy.Evaluate(observation)
	if err != nil {
		return nil, policyError(err)
	}
	if policy.Mode == internalpolicy.ModeAutomatic && decision.Eligible {
		if err := validateDispatchObservation(observation); err != nil {
			return nil, policyError(err)
		}
	}
	incident, reused, err := h.repository.RecordIncident(ctx, observation, decision)
	if err != nil {
		return nil, policyError(err)
	}
	if incident != nil && policy.Mode == internalpolicy.ModeAutomatic && decision.Eligible && !decision.Queued && incident.ProgramID == "" {
		if recovered, recoverErr := h.repository.RecoverStaleDispatch(ctx, incident.IncidentFingerprint); recoverErr != nil {
			return nil, policyError(recoverErr)
		} else if recovered {
			if refreshed, found, refreshErr := h.repository.GetIncident(ctx, incident.IncidentFingerprint); refreshErr == nil && found {
				incident = refreshed
			}
		}
		claimKey := dispatchClaimKey(observation, incident.IncidentFingerprint)
		claimed, claimErr := h.repository.ClaimDispatch(ctx, incident.IncidentFingerprint, claimKey)
		if claimErr != nil {
			return nil, policyError(claimErr)
		}
		if claimed {
			if err := h.dispatch(ctx, observation, incident); err != nil {
				_ = h.repository.MarkDispatchFailed(ctx, incident.IncidentFingerprint, err.Error())
				return nil, policyError(err)
			}
		}
		if refreshed, found, refreshErr := h.repository.GetIncident(ctx, incident.IncidentFingerprint); refreshErr == nil && found {
			incident = refreshed
		}
	}
	if incident != nil && policy.Mode == internalpolicy.ModeAutomatic && decision.Eligible && incident.State == "dispatched" && incident.ProgramID != "" {
		if err := h.reconcileDispatchedProgram(ctx, incident); err != nil && h.logger != nil {
			h.logger.Printf("automatic investigation readback deferred incident=%s error=%v", incident.IncidentFingerprint, err)
		}
		if refreshed, found, refreshErr := h.repository.GetIncident(ctx, incident.IncidentFingerprint); refreshErr == nil && found {
			incident = refreshed
		}
	}
	response := &policyv1.TriggerRecord{Decision: triggerDecisionProto(decision), Reused: reused}
	if occurrence, found, occurrenceErr := h.repository.FindOccurrence(ctx, observation, decision); occurrenceErr == nil && found {
		response.OccurrenceId = occurrence.ID
	}
	if incident != nil {
		raw, _ := json.Marshal(incident)
		response.IncidentJson = string(raw)
	}
	return connect.NewResponse(response), nil
}

func validateDispatchObservation(observation internalpolicy.Observation) error {
	if strings.TrimSpace(observation.Question) == "" {
		return fmt.Errorf("automatic investigation requires question")
	}
	if len(observation.RunIDs) < 1 || len(observation.RunIDs) > 8 {
		return fmt.Errorf("automatic investigation requires one to eight explicit run_ids")
	}
	for _, runID := range observation.RunIDs {
		if strings.TrimSpace(runID) == "" {
			return fmt.Errorf("automatic investigation run_ids cannot contain empty values")
		}
	}
	if observation.WaitSeconds < 0 || observation.WaitSeconds > 300 {
		return fmt.Errorf("automatic investigation wait_seconds must be in [0, 300]")
	}
	return nil
}

func (h *ConnectHandler) dispatch(ctx context.Context, observation internalpolicy.Observation, incident *internalpolicy.Incident) error {
	if h.runner == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("program-runtime is unavailable for automatic investigation dispatch"))
	}
	callerKey := dispatchClaimKey(observation, incident.IncidentFingerprint)
	waitSeconds := observation.WaitSeconds
	if waitSeconds == 0 {
		waitSeconds = 30
	}
	inputs, err := structpb.NewStruct(map[string]any{
		"caller_key":       callerKey,
		"execution_id":     observation.ExecutionID,
		"phase_id":         observation.PhaseID,
		"phase_generation": observation.PhaseGeneration,
		"question":         observation.Question,
		"brief_ref":        observation.BriefRef,
		"run_ids":          stringSliceValues(observation.RunIDs),
		"wait_seconds":     int64(waitSeconds),
	})
	if err != nil {
		return fmt.Errorf("encode automatic investigation inputs: %w", err)
	}
	response, err := h.runner.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{
		Name:       compositionProgram,
		Inputs:     inputs,
		Provenance: programsv1.Provenance_PROVENANCE_AGENT,
	}))
	if err != nil {
		return fmt.Errorf("dispatch automatic investigation: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetProgram() == nil || strings.TrimSpace(response.Msg.GetProgram().GetId()) == "" {
		return errors.New("automatic investigation dispatch returned no durable program identity")
	}
	program := response.Msg.GetProgram()
	status := program.GetStatus().String()
	if err := h.repository.LinkProgram(ctx, incident.IncidentFingerprint, program.GetId(), status); err != nil {
		return fmt.Errorf("persist automatic investigation dispatch: %w", err)
	}
	stdout := program.GetStdout()
	// A terminal Program Runtime acknowledgement may race stdout persistence.
	// Perform one authoritative readback by program identity before classifying
	// the nested diagnosis; this is a bounded read, never a polling loop.
	if strings.TrimSpace(stdout) == "" && program.GetStatus() == programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
		if reader, ok := h.runner.(interface {
			GetProgram(context.Context, *connect.Request[programsv1.GetProgramRequest]) (*connect.Response[programsv1.GetProgramResponse], error)
		}); ok {
			if refreshed, readErr := reader.GetProgram(ctx, connect.NewRequest(&programsv1.GetProgramRequest{Id: program.GetId()})); readErr == nil && refreshed != nil && refreshed.Msg != nil && refreshed.Msg.GetProgram() != nil {
				stdout = refreshed.Msg.GetProgram().GetStdout()
			}
		}
	}
	if investigationID, terminal := nestedInvestigationIdentity(stdout); investigationID != "" {
		if terminal {
			if err := h.repository.MarkInvestigationCompleted(ctx, incident.IncidentFingerprint, investigationID); err != nil {
				return fmt.Errorf("persist completed automatic investigation: %w", err)
			}
		} else if err := h.repository.LinkInvestigation(ctx, incident.IncidentFingerprint, investigationID); err != nil {
			return fmt.Errorf("persist automatic investigation identity: %w", err)
		}
	}
	if status := nestedCompositionStatus(stdout); status == "unavailable" || status == "failed" {
		if err := h.repository.MarkDispatchRetryable(ctx, incident.IncidentFingerprint, program.GetId(), "composition returned "+status+" before nested investigation admission"); err != nil {
			return fmt.Errorf("persist retryable automatic investigation dispatch: %w", err)
		}
	}
	if h.logger != nil {
		h.logger.Printf("automatic investigation dispatched incident=%s program=%s status=%s", incident.IncidentFingerprint, program.GetId(), status)
	}
	return nil
}

func dispatchClaimKey(observation internalpolicy.Observation, fingerprint string) string {
	callerKey := strings.TrimSpace(observation.CallerKey)
	if callerKey == "" {
		callerKey = "plan-trigger/" + fingerprint
	}
	return callerKey
}

// nestedInvestigationIdentity reads only the bounded composition envelope.
// It intentionally does not inspect diagnosis prose or infer completion from
// the outer Program Runtime status.
func nestedInvestigationIdentity(stdout string) (string, bool) {
	var envelope struct {
		Status  string `json:"status"`
		Signals struct {
			ExecutionID string `json:"execution_id"`
			Pending     bool   `json:"pending"`
			Nested      struct {
				ExecutionID string `json:"execution_id"`
				Pending     bool   `json:"pending"`
			} `json:"nested_signals"`
		} `json:"signals"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		return "", false
	}
	executionID := strings.TrimSpace(envelope.Signals.ExecutionID)
	pending := envelope.Signals.Pending
	if executionID == "" {
		executionID = strings.TrimSpace(envelope.Signals.Nested.ExecutionID)
		pending = pending || envelope.Signals.Nested.Pending
	}
	if executionID == "" {
		return "", false
	}
	return executionID, envelope.Status == "ok" && !pending
}

func nestedCompositionStatus(stdout string) string {
	var envelope struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(envelope.Status))
}

func (h *ConnectHandler) reconcileDispatchedProgram(ctx context.Context, incident *internalpolicy.Incident) error {
	reader, ok := h.runner.(interface {
		GetProgram(context.Context, *connect.Request[programsv1.GetProgramRequest]) (*connect.Response[programsv1.GetProgramResponse], error)
	})
	if !ok {
		return errors.New("program runtime readback is unavailable")
	}
	response, err := reader.GetProgram(ctx, connect.NewRequest(&programsv1.GetProgramRequest{Id: incident.ProgramID}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil || response.Msg.GetProgram() == nil {
		return errors.New("program runtime readback returned no program")
	}
	program := response.Msg.GetProgram()
	if investigationID, terminal := nestedInvestigationIdentity(program.GetStdout()); investigationID != "" && terminal {
		return h.repository.MarkInvestigationCompleted(ctx, incident.IncidentFingerprint, investigationID)
	}
	return nil
}

func stringSliceValues(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func (h *ConnectHandler) LinkTrigger(ctx context.Context, req *connect.Request[policyv1.LinkTriggerRequest]) (*connect.Response[policyv1.TriggerRecord], error) {
	if req == nil || req.Msg == nil || req.Msg.GetIncidentFingerprint() == "" || req.Msg.GetInvestigationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("incident_fingerprint and investigation_id are required"))
	}
	incident, found, err := h.repository.GetIncident(ctx, req.Msg.GetIncidentFingerprint())
	if err != nil {
		return nil, policyError(err)
	}
	if !found {
		return nil, policyError(internalpolicy.ErrIncidentNotFound)
	}
	if err := h.repository.LinkInvestigation(ctx, req.Msg.GetIncidentFingerprint(), req.Msg.GetInvestigationId()); err != nil {
		return nil, policyError(err)
	}
	if refreshed, found, refreshErr := h.repository.GetIncident(ctx, req.Msg.GetIncidentFingerprint()); refreshErr == nil && found {
		incident = refreshed
	}
	raw, _ := json.Marshal(incident)
	return connect.NewResponse(&policyv1.TriggerRecord{Decision: triggerDecisionProto(incident.Decision), IncidentJson: string(raw)}), nil
}

func (h *ConnectHandler) ListIncidents(ctx context.Context, req *connect.Request[policyv1.ListIncidentsRequest]) (*connect.Response[policyv1.ListIncidentsResponse], error) {
	var executionID, familyID, state string
	limit := 50
	if req != nil && req.Msg != nil {
		executionID = req.Msg.GetExecutionId()
		familyID = req.Msg.GetFamilyId()
		state = req.Msg.GetState()
		if req.Msg.GetLimit() > 0 {
			limit = int(req.Msg.GetLimit())
		}
	}
	incidents, err := h.repository.ListIncidents(ctx, executionID, familyID, state, limit)
	if err != nil {
		return nil, policyError(err)
	}
	response := &policyv1.ListIncidentsResponse{Incidents: make([]*policyv1.IncidentRecord, 0, len(incidents))}
	for _, incident := range incidents {
		response.Incidents = append(response.Incidents, incidentRecordProto(incident))
	}
	return connect.NewResponse(response), nil
}

func (h *ConnectHandler) GetIncident(ctx context.Context, req *connect.Request[policyv1.GetIncidentRequest]) (*connect.Response[policyv1.IncidentRecord], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetIncidentFingerprint()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("incident_fingerprint is required"))
	}
	incident, found, err := h.repository.GetIncident(ctx, req.Msg.GetIncidentFingerprint())
	if err != nil {
		return nil, policyError(err)
	}
	if !found {
		return nil, policyError(internalpolicy.ErrIncidentNotFound)
	}
	return connect.NewResponse(incidentRecordProto(incident)), nil
}

func (h *ConnectHandler) ListOccurrences(ctx context.Context, req *connect.Request[policyv1.ListOccurrencesRequest]) (*connect.Response[policyv1.ListOccurrencesResponse], error) {
	var executionID string
	eligibleOnly := false
	limit := 50
	if req != nil && req.Msg != nil {
		executionID = req.Msg.GetExecutionId()
		eligibleOnly = req.Msg.GetEligibleOnly()
		if req.Msg.GetLimit() > 0 {
			limit = int(req.Msg.GetLimit())
		}
	}
	occurrences, err := h.repository.ListOccurrences(ctx, executionID, eligibleOnly, limit)
	if err != nil {
		return nil, policyError(err)
	}
	response := &policyv1.ListOccurrencesResponse{Occurrences: make([]*policyv1.IncidentOccurrence, 0, len(occurrences))}
	for _, occurrence := range occurrences {
		response.Occurrences = append(response.Occurrences, occurrenceProto(occurrence))
	}
	return connect.NewResponse(response), nil
}

func (h *ConnectHandler) GetBrief(ctx context.Context, req *connect.Request[policyv1.GetBriefRequest]) (*connect.Response[policyv1.PlanInvestigationBrief], error) {
	if h.brief == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("investigation brief provider is unavailable"))
	}
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetExecutionId()) == "" || strings.TrimSpace(req.Msg.GetPhaseId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("execution_id and phase_id are required"))
	}
	brief, err := h.brief.Get(ctx, internalbrief.Request{
		ExecutionID: req.Msg.GetExecutionId(), PhaseID: req.Msg.GetPhaseId(), RunIDs: req.Msg.GetRunIds(), ValidationOperationID: req.Msg.GetValidationOperationId(),
	})
	if err != nil {
		return nil, briefError(err)
	}
	return connect.NewResponse(briefProto(brief)), nil
}

func briefProto(brief internalbrief.Brief) *policyv1.PlanInvestigationBrief {
	out := &policyv1.PlanInvestigationBrief{
		SchemaVersion: brief.SchemaVersion, Status: brief.Status, ExecutionId: brief.ExecutionID, PlanId: brief.PlanID,
		PlanRevision: brief.PlanRevision, PhaseId: brief.PhaseID, PhaseGeneration: int32(brief.PhaseGeneration), RunIds: brief.RunIDs,
		ExpectedOutcome: brief.ExpectedOutcome, AcceptanceCriteria: brief.AcceptanceCriteria, WallTimeSeconds: brief.WallTimeSeconds,
		ActiveWorkSeconds: brief.ActiveWorkSeconds, KnownWaitSeconds: brief.KnownWaitSeconds, OmittedReasons: brief.OmittedReasons,
		Budget: &policyv1.BriefBudget{Known: brief.Budget.Known, Basis: brief.Budget.Basis, QueueSeconds: int32(brief.Budget.QueueSeconds), ExecutionSeconds: int32(brief.Budget.ExecutionSeconds), TransportWaitSeconds: int32(brief.Budget.TransportWaitSeconds)},
	}
	for _, marker := range brief.MaterialProgress {
		out.MaterialProgress = append(out.MaterialProgress, &policyv1.BriefProgressMarker{Kind: marker.Kind, EvidenceRef: marker.EvidenceRef, Detail: marker.Detail, ObservedAt: marker.ObservedAt})
	}
	if brief.ProducerWait != nil {
		out.ProducerWait = &policyv1.BriefProducerWait{OperationId: brief.ProducerWait.OperationID, Status: brief.ProducerWait.Status, QueueReason: brief.ProducerWait.QueueReason, WaitArgv: brief.ProducerWait.WaitArgv, SyncArgv: brief.ProducerWait.SyncArgv, Explicit: brief.ProducerWait.Explicit}
	}
	return out
}

func incidentRecordProto(incident *internalpolicy.Incident) *policyv1.IncidentRecord {
	if incident == nil {
		return nil
	}
	occurrences := make([]*policyv1.IncidentOccurrence, 0, len(incident.Occurrences))
	for _, occurrence := range incident.Occurrences {
		occurrences = append(occurrences, occurrenceProto(occurrence))
	}
	return &policyv1.IncidentRecord{
		IncidentId:          incident.ID,
		ExecutionId:         incident.ExecutionID,
		PhaseId:             incident.PhaseID,
		PhaseGeneration:     incident.PhaseGeneration,
		PolicyVersion:       incident.PolicyVersion,
		IncidentFingerprint: incident.IncidentFingerprint,
		Mode:                string(incident.Mode),
		State:               incident.State,
		Decision:            triggerDecisionProto(incident.Decision),
		InvestigationId:     incident.InvestigationID,
		ProgramId:           incident.ProgramID,
		ProgramStatus:       incident.ProgramStatus,
		DispatchError:       incident.DispatchError,
		CreatedAt:           incident.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:           incident.UpdatedAt.UTC().Format(time.RFC3339Nano),
		OccurrenceCount:     int32(incident.OccurrenceCount),
		Occurrences:         occurrences,
		FamilyId:            incident.FamilyID,
		SubjectExecutionIds: incident.SubjectExecutionIDs,
	}
}

func occurrenceProto(occurrence internalpolicy.Occurrence) *policyv1.IncidentOccurrence {
	return &policyv1.IncidentOccurrence{
		OccurrenceId:        occurrence.ID,
		IncidentFingerprint: occurrence.IncidentFingerprint,
		Decision:            triggerDecisionProto(occurrence.Decision),
		ObservedAt:          occurrence.ObservedAt.UTC().Format(time.RFC3339Nano),
		CreatedAt:           occurrence.CreatedAt.UTC().Format(time.RFC3339Nano),
		ExecutionId:         occurrence.ExecutionID,
		PhaseId:             occurrence.PhaseID,
		PhaseGeneration:     occurrence.PhaseGeneration,
		PolicyVersion:       occurrence.PolicyVersion,
		FamilyId:            occurrence.FamilyID,
		SharedFailureRef:    occurrence.SharedFailureRef,
	}
}

func triggerDecisionProto(decision internalpolicy.Decision) *policyv1.TriggerDecision {
	response := &policyv1.TriggerDecision{Eligible: decision.Eligible, Queued: decision.Queued, Mode: string(decision.Mode), IncidentFingerprint: decision.IncidentFingerprint, Reasons: decision.Reasons}
	for _, match := range decision.Matches {
		response.Matches = append(response.Matches, &policyv1.TriggerMatch{Kind: match.Kind, Detail: match.Detail})
	}
	return response
}

func policyError(err error) error {
	code := connect.CodeInternal
	if errors.Is(err, internalpolicy.ErrInvalidPolicy) || errors.Is(err, internalpolicy.ErrInvalidObservation) {
		code = connect.CodeInvalidArgument
	}
	if errors.Is(err, internalpolicy.ErrPolicyNotFound) {
		code = connect.CodeNotFound
	}
	if errors.Is(err, internalpolicy.ErrIncidentNotFound) {
		code = connect.CodeNotFound
	}
	if errors.Is(err, internalpolicy.ErrPolicyConflict) {
		code = connect.CodeAborted
	}
	if strings.Contains(err.Error(), "program-runtime is unavailable") || strings.Contains(err.Error(), "dispatch automatic investigation") {
		code = connect.CodeUnavailable
	}
	if strings.Contains(err.Error(), "automatic investigation requires") {
		code = connect.CodeInvalidArgument
	}
	return connect.NewError(code, err)
}

func briefError(err error) error {
	code := connect.CodeInternal
	if errors.Is(err, internalbrief.ErrInvalidRequest) || errors.Is(err, internalbrief.ErrRunSetMismatch) {
		code = connect.CodeInvalidArgument
	}
	if errors.Is(err, internalbrief.ErrExecutionMissing) {
		code = connect.CodeNotFound
	}
	if errors.Is(err, internalbrief.ErrPhaseStale) || errors.Is(err, internalbrief.ErrOperationScope) {
		code = connect.CodeFailedPrecondition
	}
	return connect.NewError(code, err)
}

// discoveredProgramRunner resolves Program Runtime at call time so Plan
// Manager remains restart-safe and does not pin a transient port at startup.
type discoveredProgramRunner struct {
	client *http.Client
}

func NewDiscoveredProgramRunner() declaredProgramRunner {
	return &discoveredProgramRunner{client: &http.Client{Timeout: 65 * time.Second}}
}

func (r *discoveredProgramRunner) RunDeclaredProgram(ctx context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "program-runtime")
	if err != nil {
		return nil, err
	}
	client := libraryconnect.NewLibraryServiceClient(r.client, strings.TrimRight(base, "/"))
	return client.RunDeclaredProgram(ctx, req)
}

func (r *discoveredProgramRunner) GetProgram(ctx context.Context, req *connect.Request[programsv1.GetProgramRequest]) (*connect.Response[programsv1.GetProgramResponse], error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "program-runtime")
	if err != nil {
		return nil, err
	}
	client := programsconnect.NewProgramServiceClient(r.client, strings.TrimRight(base, "/"))
	return client.GetProgram(ctx, req)
}
