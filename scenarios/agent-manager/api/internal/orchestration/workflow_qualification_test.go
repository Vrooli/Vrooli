package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/workflowruntime"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/proto"
)

// The HTTP peer stands in only for PRT. Workflow attempts, journals, reviewer
// lineage, cancellation and restart use the real SQLite repositories and adapter.
type qualificationProgramPeer struct {
	libraryconnect.UnimplementedLibraryServiceHandler
	mu             sync.Mutex
	program        *programsv1.Program
	starts, closes int
	version        uint32
	lost           bool
}

type qualificationRoundTripper struct{ handler http.Handler }

func (rt qualificationRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	response := httptest.NewRecorder()
	rt.handler.ServeHTTP(response, req)
	return response.Result(), nil
}

func TestQualificationOwnerWiredByProductionConstructor(t *testing.T) {
	_, repos := newRelayOrchestrator(t, newFakeRunLauncher())
	o := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithWorkflowExecutionRepository(repos.WorkflowExecutions),
		WithWorkflowRepository(repos.Workflows))
	t.Cleanup(o.dispatcher.Close)
	require.NotNil(t, o.workflowEngine)
	owner, ok := o.workflowEngine.Qualifications.(workflowQualificationOwner)
	require.True(t, ok, "production workflows need the existing qualification owner")
	require.Same(t, o, owner.o)
}

func (s *qualificationProgramPeer) GetDeclaredExecution(context.Context, *connect.Request[libraryv1.GetDeclaredExecutionRequest]) (*connect.Response[libraryv1.GetDeclaredExecutionResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &libraryv1.GetDeclaredExecutionResponse{AdmissionContractVersion: s.version, Found: s.program != nil}
	if s.program != nil {
		r.Program = proto.Clone(s.program).(*programsv1.Program)
	}
	return connect.NewResponse(r), nil
}

func (s *qualificationProgramPeer) retain(req *libraryv1.RunDeclaredProgramRequest) {
	if s.program == nil {
		s.program = &programsv1.Program{Id: "original-program", ProgramName: req.Name, ProgramDigest: req.ExpectedDigest, RequestDigest: strings.Repeat("d", 64), CallerRunId: req.Caller.RunId, CallerHarness: req.Caller.Harness, Status: programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING, SessionId: "original-session"}
	}
}

func (s *qualificationProgramPeer) RunDeclaredProgram(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.starts++
	s.retain(req.Msg)
	if s.lost {
		s.lost = false
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("lost response after admission"))
	}
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Program: proto.Clone(s.program).(*programsv1.Program)}), nil
}

func (s *qualificationProgramPeer) CloseDeclaredAdmission(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closes++
	if s.program == nil {
		s.retain(req.Msg)
		s.program.Status = programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED
		s.program.SessionId, s.program.FailureShape = "", "admission_closed"
	}
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Program: proto.Clone(s.program).(*programsv1.Program)}), nil
}

func qualificationOrchestrator(t *testing.T, peer *qualificationProgramPeer) (*Orchestrator, *database.Repositories, *domain.WorkflowExecution, workflowruntime.QualificationCandidate) {
	t.Helper()
	fake := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, fake)
	_, handler := libraryconnect.NewLibraryServiceHandler(peer)
	qualificationOwner, ownerOK := o.workflowEngine.Qualifications.(workflowQualificationOwner)
	if !ownerOK {
		// newRelayOrchestrator replaces the constructor's engine with a child
		// test engine; install the same production adapter on that harness.
		qualificationOwner = workflowQualificationOwner{o: o}
	}
	qualificationOwner.http = &http.Client{Transport: qualificationRoundTripper{handler: handler}}
	qualificationOwner.resolve = func(context.Context) (string, error) { return "http://program-runtime.test", nil }
	o.workflowEngine.Qualifications = qualificationOwner
	r := relayDefinition()
	r.Definition.EntryNode = "qualify"
	r.Definition.Nodes = []domain.WorkflowNode{{ID: "qualify", Kind: domain.WorkflowNodeQualification, Qualification: &domain.WorkflowQualificationNode{ReviewFromNode: "review", ProgramName: "owner.qualify", ProgramDigest: strings.Repeat("a", 64)}}, {ID: "end", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	r.Definition.Edges = []domain.WorkflowEdge{{From: "qualify", To: "end"}}
	require.NoError(t, repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{r}))
	x, err := o.workflowEngine.Start(t.Context(), r, json.RawMessage(`{}`), "qualification-owner")
	require.NoError(t, err)
	c := workflowruntime.QualificationCandidate{ReviewAttemptID: uuid.New(), ReviewerRunID: uuid.New(), SourceRunID: uuid.New(), SandboxID: uuid.New(), ReviewRequestID: uuid.New(), SHA256: strings.Repeat("b", 64)}
	task := &domain.Task{ID: uuid.New(), Title: "retained candidate", ScopePath: ".", Status: domain.TaskStatusQueued}
	require.NoError(t, repos.Tasks.Create(t.Context(), task))
	off := false
	spec, err := structuredresult.BindReviewCandidate(&domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(`{"type":"object","properties":{"accepted":{"type":"boolean"},"candidateSha256":{"type":"string"}},"required":["accepted","candidateSha256"]}`)}, c.SHA256)
	require.NoError(t, err)
	now := time.Now().UTC()
	source := &domain.Run{ID: c.SourceRunID, TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, ExecutionMode: domain.ExecutionModeCodecPipe, RunMode: domain.RunModeSandboxed, SandboxID: &c.SandboxID, CreatedAt: now, EndedAt: &now, CustomEnv: map[string]string{workflowExecutionEnv: x.ID.String()}}
	require.NoError(t, repos.Runs.Create(t.Context(), source))
	reviewer := &domain.Run{ID: c.ReviewerRunID, TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, ExecutionMode: domain.ExecutionModeCodecPipe, RunMode: domain.RunModeSandboxed, CreatedAt: now, EndedAt: &now, CustomEnv: map[string]string{workflowExecutionEnv: x.ID.String(), workflowAttemptEnv: c.ReviewAttemptID.String(), "VROOLI_REVIEW_SOURCE_RUN_ID": source.ID.String(), "VROOLI_REVIEW_REQUEST_ID": c.ReviewRequestID.String(), "VROOLI_REVIEW_SHA256": c.SHA256}, ResolvedConfig: &domain.RunConfig{ResultSpec: spec, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected, AutoApply: &off, WritePolicy: &domain.WorkspaceWritePolicy{}, NetworkMode: domain.NetworkAccessNone}}}
	require.NoError(t, repos.Runs.Create(t.Context(), reviewer))
	attempt := &domain.WorkflowNodeAttempt{ID: c.ReviewAttemptID, ExecutionID: x.ID, NodeID: "review", Ordinal: 1, Strategy: domain.WorkflowAttemptFreshRun, Status: domain.WorkflowAttemptCompleted, RunID: &reviewer.ID, Version: 1, CreatedAt: now, UpdatedAt: now, CompletedAt: &now}
	receipt, err := json.Marshal(&domain.StructuredResult{Status: domain.StructuredResultSuccess, SchemaDigest: spec.SchemaDigest, Value: json.RawMessage(`{"accepted":true,"candidateSha256":"` + c.SHA256 + `"}`)})
	require.NoError(t, err)
	journal, err := repos.WorkflowExecutions.ListJournal(t.Context(), x.ID, 0, 0)
	require.NoError(t, err)
	sequence := int64(len(journal) + 1)
	entry := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: x.ID, Sequence: sequence, Kind: domain.WorkflowJournalStructured, NodeID: "review", AttemptID: &attempt.ID, Payload: receipt, CreatedAt: now}
	x.Version++
	ok, err := repos.WorkflowExecutions.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: attempt, Journal: []*domain.WorkflowJournalEntry{entry}})
	require.NoError(t, err)
	require.True(t, ok)
	fake.states[reviewer.ID] = workflowruntime.ChildState{RunID: reviewer.ID, Terminal: true, TokensKnown: true, ChargeMeasured: true, Tokens: 7}
	return o, repos, x, c
}

func TestQualificationOwnerCancellationRecoversLostAdmissionWithoutDispatch(t *testing.T) {
	for _, submitted := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-dispatch", true: "lost-response"}[submitted], func(t *testing.T) {
			peer := &qualificationProgramPeer{version: 2, lost: submitted}
			o, repos, x, candidate := qualificationOrchestrator(t, peer)
			_, err := o.workflowEngine.Advance(t.Context(), x.ID)
			require.NoError(t, err)
			if submitted {
				_, err = o.workflowEngine.Advance(t.Context(), x.ID)
				require.ErrorContains(t, err, "lost response")
			}
			_, err = o.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "cancel", Reason: "operator"})
			if submitted {
				require.ErrorContains(t, err, "terminal accounting")
				peer.mu.Lock()
				peer.program.Status = programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED
				peer.program.Stdout = `{"status":"ok","signals":{"candidate_sha256":"` + candidate.SHA256 + `"},"usage":{"tokens":999,"charge_micro_usd":999,"accounting_complete":true,"charge_measured":true}}`
				peer.program.UsageAccountingComplete = true
				peer.program.UsageChargeMeasured = true
				peer.mu.Unlock()
				old := o.workflowEngine.Qualifications.(workflowQualificationOwner)
				restarted, _ := reopenRelayOrchestrator(t, repos, o.workflowEngine.Children.(*fakeRunLauncher))
				old.o = restarted
				restarted.workflowEngine.Qualifications = old
				require.NoError(t, restarted.RecoverWorkflowExecutions(t.Context()))
			} else {
				require.NoError(t, err)
			}
			settled, err := repos.WorkflowExecutions.Get(t.Context(), x.ID)
			require.NoError(t, err)
			require.Equal(t, domain.WorkflowExecutionCancelled, settled.Status)
			require.True(t, settled.BudgetUsage.AccountingComplete)
			require.Equal(t, 7, settled.BudgetUsage.Tokens)
			require.Equal(t, 2, settled.BudgetUsage.Children)
			peer.mu.Lock()
			defer peer.mu.Unlock()
			if submitted {
				require.Equal(t, 1, peer.starts)
			} else {
				require.Zero(t, peer.starts)
			}
			require.Positive(t, peer.closes)
		})
	}
}

func TestQualificationConsumesRuntimeReceiptAndRejectsStdoutUsage(t *testing.T) {
	req := workflowruntime.QualificationRequest{ProgramName: "owner.qualify", ProgramDigest: strings.Repeat("a", 64), Candidate: workflowruntime.QualificationCandidate{SHA256: strings.Repeat("b", 64)}}
	p := &programsv1.Program{Id: "p", ProgramName: req.ProgramName, ProgramDigest: req.ProgramDigest, CallerRunId: uuid.NewString(), CallerHarness: "workflow-qualification", RequestDigest: strings.Repeat("d", 64), Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: `{"status":"ok","signals":{"candidate_sha256":"` + req.Candidate.SHA256 + `"},"usage":{"tokens":999,"charge_micro_usd":999,"accounting_complete":true,"charge_measured":true}}`}
	req.ExecutionID, _ = uuid.Parse(p.CallerRunId)
	state, err := qualificationProgramState(req, p)
	require.NoError(t, err)
	require.False(t, state.Passed)
	require.False(t, state.BudgetUsage.AccountingComplete)
	p.UsageTokens, p.UsageChargeMicros = 11, 13
	p.UsageAccountingComplete, p.UsageChargeMeasured = true, true
	state, err = qualificationProgramState(req, p)
	require.NoError(t, err)
	require.True(t, state.Passed)
	require.Equal(t, 11, state.BudgetUsage.Tokens)
	require.Equal(t, int64(13), state.BudgetUsage.ChargeMicroUSD)
}

func TestQualificationOwnerRefusesStaleRuntimeBeforeEffects(t *testing.T) {
	peer := &qualificationProgramPeer{version: 1}
	o, _, x, _ := qualificationOrchestrator(t, peer)
	_, err := o.workflowEngine.Advance(t.Context(), x.ID)
	require.NoError(t, err)
	_, err = o.workflowEngine.Advance(t.Context(), x.ID)
	require.ErrorContains(t, err, "lacks bounded keyed admission")
	require.Zero(t, peer.starts)
}

func TestQualificationOwnerRejectsWrongReviewAttempt(t *testing.T) {
	o, _, x, candidate := qualificationOrchestrator(t, &qualificationProgramPeer{version: 2})
	_, err := o.workflowEngine.Qualifications.Prepare(t.Context(), x.ID, uuid.New(), candidate.ReviewerRunID)
	require.ErrorContains(t, err, "does not belong")
}
