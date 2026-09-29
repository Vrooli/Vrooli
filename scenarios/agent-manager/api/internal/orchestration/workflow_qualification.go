package orchestration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/workflowruntime"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type workflowQualificationOwner struct {
	o       *Orchestrator
	resolve func(context.Context) (string, error)
	http    *http.Client
}

func (l workflowQualificationOwner) Prepare(ctx context.Context, executionID, attemptID, reviewerID uuid.UUID) (workflowruntime.QualificationCandidate, error) {
	var candidate workflowruntime.QualificationCandidate
	run, err := l.o.GetRun(ctx, reviewerID)
	if err != nil {
		return candidate, err
	}
	if run == nil || run.CustomEnv[workflowExecutionEnv] != executionID.String() || run.CustomEnv[workflowAttemptEnv] != attemptID.String() || run.ResolvedConfig == nil || !childStateFromRun(run).Terminal {
		return candidate, workflowruntime.NewQualificationPreparationError("qualification_review_invalid", "qualification review does not belong to this workflow")
	}
	cfg := run.ResolvedConfig.SandboxConfig
	if cfg == nil || cfg.Mode != domain.SandboxModeProtected || cfg.WritePolicy == nil || len(cfg.WritePolicy.Paths) != 0 || cfg.NetworkMode != domain.NetworkAccessNone || cfg.GetAutoApply() {
		return candidate, workflowruntime.NewQualificationPreparationError("qualification_review_invalid", "qualification review lacks independent read-only authority")
	}
	sourceID, err := uuid.Parse(run.CustomEnv["VROOLI_REVIEW_SOURCE_RUN_ID"])
	if err != nil || sourceID == reviewerID {
		return candidate, workflowruntime.NewQualificationPreparationError("qualification_review_invalid", "qualification review source is invalid")
	}
	requestID, err := uuid.Parse(run.CustomEnv["VROOLI_REVIEW_REQUEST_ID"])
	if err != nil {
		return candidate, workflowruntime.NewQualificationPreparationError("qualification_review_invalid", "qualification review snapshot is missing")
	}
	digest := run.CustomEnv["VROOLI_REVIEW_SHA256"]
	spec, err := structuredresult.BindReviewCandidate(run.ResolvedConfig.ResultSpec, digest)
	if err != nil {
		return candidate, err
	}
	// Use the completed attempt's journal receipt, not the Run's mutable latest
	// turn result. A later continuation cannot replace an earlier acceptance.
	journal, err := l.o.workflowExecutions.ListJournal(ctx, executionID, 0, 0)
	if err != nil {
		return candidate, err
	}
	var receipt *domain.StructuredResult
	for _, entry := range journal {
		if entry.Kind == domain.WorkflowJournalStructured && entry.AttemptID != nil && *entry.AttemptID == attemptID {
			if receipt != nil {
				return candidate, workflowruntime.NewQualificationPreparationError("qualification_review_invalid", "qualification review receipt is ambiguous")
			}
			receipt = &domain.StructuredResult{}
			if err = json.Unmarshal(entry.Payload, receipt); err != nil {
				return candidate, err
			}
		}
	}
	if receipt == nil || receipt.Status != domain.StructuredResultSuccess || receipt.SchemaDigest != spec.SchemaDigest || structuredresult.ValidateValue(spec.Schema, receipt.Value) != nil {
		return candidate, workflowruntime.NewQualificationPreparationError("qualification_review_invalid", "qualification requires a valid candidate-bound review receipt")
	}
	var verdict struct {
		Accepted bool   `json:"accepted"`
		SHA256   string `json:"candidateSha256"`
	}
	if json.Unmarshal(receipt.Value, &verdict) != nil || !verdict.Accepted || verdict.SHA256 != digest {
		return candidate, workflowruntime.NewQualificationPreparationError("qualification_review_rejected", "independent review did not accept this candidate")
	}
	source, err := l.o.GetRun(ctx, sourceID)
	if err != nil {
		return candidate, err
	}
	if source == nil || source.SandboxID == nil || source.CustomEnv[workflowExecutionEnv] != executionID.String() || source.ExecutionMode.Normalized() != domain.ExecutionModeCodecPipe || !childStateFromRun(source).Terminal {
		return candidate, workflowruntime.NewQualificationPreparationError("qualification_review_invalid", "qualification source is not the retained workflow worker")
	}
	encoded, _ := json.Marshal(receipt)
	return workflowruntime.QualificationCandidate{ReviewAttemptID: attemptID, VerdictSHA256: fmt.Sprintf("%x", sha256.Sum256(encoded)), ReviewerRunID: reviewerID, SourceRunID: sourceID, SandboxID: *source.SandboxID, ReviewRequestID: requestID, SHA256: digest}, nil
}

func (l workflowQualificationOwner) clients(ctx context.Context) (libraryconnect.LibraryServiceClient, programsconnect.ProgramServiceClient, error) {
	resolve := l.resolve
	if resolve == nil {
		resolve = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "program-runtime")
		}
	}
	url, err := resolve(ctx)
	if err != nil {
		return nil, nil, err
	}
	client := l.http
	if client == nil {
		client = &http.Client{Timeout: 310 * time.Second}
	}
	return libraryconnect.NewLibraryServiceClient(client, url), programsconnect.NewProgramServiceClient(client, url), nil
}

func (l workflowQualificationOwner) authorized(ctx context.Context, req workflowruntime.QualificationRequest, closing bool) error {
	x, err := l.o.workflowExecutions.Get(ctx, req.ExecutionID)
	if err != nil {
		return err
	}
	if x == nil || !strings.HasPrefix(req.ProgramName, x.Owner+".") || closing != (x.Status.Terminal() || x.Status == domain.WorkflowExecutionCancelling) {
		return fmt.Errorf("qualification execution is not authorized for this operation")
	}
	attempts, err := l.o.workflowExecutions.ListAttempts(ctx, x.ID)
	if err != nil {
		return err
	}
	exact, _ := json.Marshal(req)
	admitted := false
	for _, a := range attempts {
		if a.Strategy == domain.WorkflowAttemptQualification && (closing || (a.NodeID == x.CurrentNodeID && a.Status == domain.WorkflowAttemptDispatchPending)) && a.IdempotencyKey == req.IdempotencyKey && bytes.Equal(a.InputSnapshot, exact) {
			admitted = true
			break
		}
	}
	if !admitted {
		return fmt.Errorf("qualification intent is not retained by the workflow attempt")
	}
	return nil
}

func (l workflowQualificationOwner) Start(ctx context.Context, req workflowruntime.QualificationRequest) (workflowruntime.QualificationState, error) {
	empty := workflowruntime.QualificationState{}
	if err := l.authorized(ctx, req, false); err != nil {
		return empty, err
	}
	library, _, err := l.clients(ctx)
	if err != nil {
		return empty, err
	}
	// A stale runtime must refuse before any effects. Merely adding unknown
	// request fields is not proof that the serving API enforces their semantics.
	observed, err := library.GetDeclaredExecution(ctx, connect.NewRequest(&libraryv1.GetDeclaredExecutionRequest{Name: req.ProgramName, IdempotencyKey: req.IdempotencyKey}))
	if err != nil {
		return empty, err
	}
	if observed.Msg.GetAdmissionContractVersion() != 2 {
		return empty, fmt.Errorf("program runtime lacks bounded keyed admission")
	}
	request, err := qualificationProgramRequest(req)
	if err != nil {
		return empty, err
	}
	r, err := library.RunDeclaredProgram(ctx, connect.NewRequest(request))
	if err != nil {
		return empty, err
	}
	return qualificationProgramState(req, r.Msg.GetProgram())
}

func qualificationProgramRequest(req workflowruntime.QualificationRequest) (*libraryv1.RunDeclaredProgramRequest, error) {
	inputs := map[string]any{}
	if err := json.Unmarshal(req.Inputs, &inputs); err != nil || inputs == nil {
		return nil, fmt.Errorf("invalid retained qualification inputs")
	}
	if _, exists := inputs["candidate"]; exists {
		return nil, fmt.Errorf("qualification candidate cannot be caller-overridden")
	}
	data, _ := json.Marshal(req.Candidate)
	var candidate map[string]any
	if err := json.Unmarshal(data, &candidate); err != nil {
		return nil, err
	}
	inputs["candidate"] = candidate
	structured, err := structpb.NewStruct(inputs)
	if err != nil {
		return nil, err
	}
	return &libraryv1.RunDeclaredProgramRequest{Name: req.ProgramName, ExpectedDigest: req.ProgramDigest, IdempotencyKey: req.IdempotencyKey, AdmissionDeadline: req.AdmissionDeadline.UTC().Format(time.RFC3339Nano), Inputs: structured, Async: true, Grants: req.Grants, Provenance: programsv1.Provenance_PROVENANCE_AGENT, Caller: &programsv1.Caller{RunId: req.ExecutionID.String(), Harness: "workflow-qualification"}}, nil
}

func (l workflowQualificationOwner) Observe(ctx context.Context, req workflowruntime.QualificationRequest) (workflowruntime.QualificationState, error) {
	library, _, err := l.clients(ctx)
	if err != nil {
		return workflowruntime.QualificationState{}, err
	}
	r, err := library.GetDeclaredExecution(ctx, connect.NewRequest(&libraryv1.GetDeclaredExecutionRequest{Name: req.ProgramName, IdempotencyKey: req.IdempotencyKey}))
	if err != nil {
		return workflowruntime.QualificationState{}, err
	}
	if !r.Msg.GetFound() {
		return workflowruntime.QualificationState{}, fmt.Errorf("original qualification admission is unresolved")
	}
	return qualificationProgramState(req, r.Msg.GetProgram())
}

func (l workflowQualificationOwner) CloseAdmission(ctx context.Context, req workflowruntime.QualificationRequest) (workflowruntime.QualificationState, error) {
	if err := l.authorized(ctx, req, true); err != nil {
		return workflowruntime.QualificationState{}, err
	}
	library, _, err := l.clients(ctx)
	if err != nil {
		return workflowruntime.QualificationState{}, err
	}
	request, err := qualificationProgramRequest(req)
	if err != nil {
		return workflowruntime.QualificationState{}, err
	}
	r, err := library.CloseDeclaredAdmission(ctx, connect.NewRequest(request))
	if err != nil {
		return workflowruntime.QualificationState{}, err
	}
	return qualificationProgramState(req, r.Msg.GetProgram())
}

func (l workflowQualificationOwner) Wait(ctx context.Context, req workflowruntime.QualificationRequest, id string) (workflowruntime.QualificationState, error) {
	_, programs, err := l.clients(ctx)
	if err != nil {
		return workflowruntime.QualificationState{}, err
	}
	r, err := programs.WaitForProgram(ctx, connect.NewRequest(&programsv1.WaitForProgramRequest{Id: id, TimeoutMillis: 300000}))
	if err != nil {
		return workflowruntime.QualificationState{}, err
	}
	if r.Msg.GetProgram().GetId() != id {
		return workflowruntime.QualificationState{}, fmt.Errorf("qualification program identity changed")
	}
	return qualificationProgramState(req, r.Msg.GetProgram())
}

func qualificationProgramState(req workflowruntime.QualificationRequest, p *programsv1.Program) (workflowruntime.QualificationState, error) {
	if p == nil || p.Id == "" || p.RequestDigest == "" || p.ProgramName != req.ProgramName || p.ProgramDigest != req.ProgramDigest || p.CallerRunId != req.ExecutionID.String() || p.CallerHarness != "workflow-qualification" {
		return workflowruntime.QualificationState{}, fmt.Errorf("qualification program receipt does not match the admitted intent")
	}
	state := workflowruntime.QualificationState{ProgramID: p.Id}
	state.Terminal = p.Status == programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED || p.Status == programsv1.ProgramStatus_PROGRAM_STATUS_FAILED || p.Status == programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED
	if !state.Terminal {
		return state, nil
	}
	if p.Status == programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED && p.SessionId == "" && p.FailureShape == "admission_closed" {
		state.BudgetUsage = domain.WorkflowBudgetUsage{AccountingComplete: true, ChargeMeasured: true}
		return state, nil
	}
	var envelope struct {
		Status  string `json:"status"`
		Signals struct {
			CandidateSHA256 string `json:"candidate_sha256"`
		} `json:"signals"`
	}
	if len(p.Stdout) <= 65536 && json.Unmarshal([]byte(p.Stdout), &envelope) == nil {
		state.Output = json.RawMessage(p.Stdout)
		if envelope.Signals.CandidateSHA256 == req.Candidate.SHA256 {
			state.BudgetUsage = domain.WorkflowBudgetUsage{Tokens: int(p.GetUsageTokens()), ChargeMicroUSD: p.GetUsageChargeMicros(), AccountingComplete: p.GetUsageAccountingComplete(), ChargeMeasured: p.GetUsageChargeMeasured()}
			state.Passed = p.Status == programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED && envelope.Status == "ok" && state.BudgetUsage.AccountingComplete && state.BudgetUsage.ChargeMeasured
		}
	}
	return state, nil
}
