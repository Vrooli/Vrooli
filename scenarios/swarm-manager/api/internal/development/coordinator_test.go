package development

import (
	"context"
	"errors"
	"sync"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
	"swarm-manager/internal/workflowcontract"
)

type fencedCoordinatorStub struct {
	mu      sync.Mutex
	starts  int
	entered chan struct{}
	release chan struct{}
}

func (s *fencedCoordinatorStub) Start(_ context.Context, invocation workflowcontract.Invocation) (workflowcontract.Start, error) {
	s.mu.Lock()
	s.starts++
	if s.starts == 1 {
		close(s.entered)
	}
	s.mu.Unlock()
	<-s.release
	return workflowcontract.Start{ExecutionID: "fenced-owner-run", ApprovalDigest: invocation.ApprovalDigest, GrantDigest: invocation.GrantDigest}, nil
}

func (*fencedCoordinatorStub) Collect(context.Context, string) (workflowcontract.Completion, error) {
	return workflowcontract.Completion{}, nil
}
func (*fencedCoordinatorStub) Cancel(context.Context, string, string, string) error { return nil }

type coordinatorWorkflowStub struct {
	started    workflowcontract.Start
	completion workflowcontract.Completion
	invocation workflowcontract.Invocation
	cancelled  bool
	starts     int
	reconciles int
	startErr   error
	reconciled workflowcontract.Start
	cancelErr  error
}

func (s *coordinatorWorkflowStub) Start(_ context.Context, invocation workflowcontract.Invocation) (workflowcontract.Start, error) {
	s.starts++
	s.invocation = invocation
	if s.startErr != nil {
		return workflowcontract.Start{}, s.startErr
	}
	started := s.started
	if started.ApprovalDigest == "" {
		started.ApprovalDigest = invocation.ApprovalDigest
	}
	if started.GrantDigest == "" {
		started.GrantDigest = invocation.GrantDigest
	}
	return started, nil
}

func (s *coordinatorWorkflowStub) ReconcileStart(context.Context, string) (workflowcontract.Start, error) {
	s.reconciles++
	return s.reconciled, nil
}

func (s *coordinatorWorkflowStub) Collect(context.Context, string) (workflowcontract.Completion, error) {
	return s.completion, nil
}

func (s *coordinatorWorkflowStub) Cancel(context.Context, string, string, string) error {
	s.cancelled = true
	return s.cancelErr
}

func TestCoordinatorReservesBindsAndCarriesOwnerGrant(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	stub := &coordinatorWorkflowStub{started: workflowcontract.Start{ExecutionID: "agent-execution-1", WorkflowDigest: "sha256:workflow-v1"}}
	input, _ := structpb.NewValue(map[string]any{"objective": "repair"})
	coordinator := NewCoordinator(service, stub)
	result, err := coordinator.Start(context.Background(), StartRequest{
		WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "repair-one", Mode: "workflow-fallback", Upper: Usage{Tokens: 5000, WallSeconds: 300},
		Invocation: workflowcontract.Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/development", WorkflowDigest: "sha256:workflow-v1", Input: input},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Execution.ExecutionID != "agent-execution-1" || result.Engagement.Attempts[0].ExecutionID != "agent-execution-1" {
		t.Fatalf("start result=%+v", result)
	}
	if stub.invocation.Grant == nil || stub.invocation.Grant.MaxTokens != 5000 || stub.invocation.Grant.MaxWallTimeSeconds != 300 || len(stub.invocation.Grant.AllowedEffects) != len(proposal.AllowedEffects) {
		t.Fatalf("owner grant=%+v", stub.invocation.Grant)
	}
	if stub.invocation.GrantDigest != workflowcontract.GrantDigest(stub.invocation.Grant) {
		t.Fatalf("grant digest=%q does not bind grant=%+v", stub.invocation.GrantDigest, stub.invocation.Grant)
	}
	if stub.invocation.IdempotencyKey != "repair-one" {
		t.Fatalf("owner idempotency key=%q", stub.invocation.IdempotencyKey)
	}
	if stub.invocation.ApprovalDigest != state.Digest || result.Execution.WorkflowDigest != "sha256:workflow-v1" || result.Engagement.Attempts[0].WorkflowDigest != "sha256:workflow-v1" {
		t.Fatalf("identity binding result=%+v invocation=%+v", result, stub.invocation)
	}
	retry, err := coordinator.Start(context.Background(), StartRequest{WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "repair-one", Mode: "workflow-fallback", Upper: Usage{Tokens: 5000, WallSeconds: 300}, Invocation: workflowcontract.Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/development"}})
	if err != nil || retry.Execution.ExecutionID != "agent-execution-1" || stub.starts != 1 {
		t.Fatalf("duplicate dispatch retry=%+v err=%v starts=%d", retry, err, stub.starts)
	}
}

func TestCoordinatorRejectsMissingExpectedWorkflowDigest(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	coordinator := NewCoordinator(service, &coordinatorWorkflowStub{started: workflowcontract.Start{ExecutionID: "owner-without-revision"}})
	_, err := coordinator.Start(context.Background(), StartRequest{
		WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "missing-workflow", Mode: "workflow-fallback",
		Upper: Usage{Tokens: 5000, WallSeconds: 300}, Invocation: workflowcontract.Invocation{WorkflowDigest: "sha256:required"},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("missing expected workflow digest err=%v", err)
	}
}

func TestCoordinatorReconcilesLostStartWithoutSecondOwnerGrant(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	transportErr := errors.New("owner response lost")
	stub := &coordinatorWorkflowStub{
		startErr:   transportErr,
		reconciled: workflowcontract.Start{ExecutionID: "reconciled-owner-run", WorkflowDigest: "sha256:workflow-v1", ApprovalDigest: state.Digest, GrantDigest: workflowcontract.GrantDigest(&workflowcontract.Grant{MaxTokens: 5000, MaxWallTimeSeconds: 300, AllowedEffects: proposal.AllowedEffects})},
	}
	coordinator := NewCoordinator(service, stub)
	req := StartRequest{WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "lost-start", Mode: "workflow-fallback", Upper: Usage{Tokens: 5000, WallSeconds: 300}, Invocation: workflowcontract.Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/development", WorkflowDigest: "sha256:workflow-v1"}}
	if _, err := coordinator.Start(context.Background(), req); !errors.Is(err, transportErr) {
		t.Fatalf("first start err=%v", err)
	}
	stub.startErr = nil
	result, err := coordinator.Start(context.Background(), req)
	if err != nil || result.Execution.ExecutionID != "reconciled-owner-run" || stub.starts != 1 || stub.reconciles != 1 {
		t.Fatalf("reconciled result=%+v err=%v starts=%d reconciles=%d", result, err, stub.starts, stub.reconciles)
	}
	state, _ = service.Get(context.Background(), proposal.WorkItem)
	if len(state.Attempts) != 1 || state.Attempts[0].ExecutionID != "reconciled-owner-run" || state.Attempts[0].WorkflowDigest != "sha256:workflow-v1" || state.Reserved.Tokens != 5000 {
		t.Fatalf("lost start state=%+v", state)
	}
}

func TestCoordinatorFencesConcurrentDispatchersBeforeOwnerStart(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	stub := &fencedCoordinatorStub{entered: make(chan struct{}), release: make(chan struct{})}
	coordinator := NewCoordinator(service, stub)
	req := StartRequest{WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "concurrent-start", Mode: "workflow-fallback", Upper: Usage{Tokens: 5000, WallSeconds: 300}}
	first := make(chan error, 1)
	go func() {
		_, err := coordinator.Start(context.Background(), req)
		first <- err
	}()
	<-stub.entered
	_, secondErr := coordinator.Start(context.Background(), req)
	if !errors.Is(secondErr, ErrConflict) {
		t.Fatalf("concurrent dispatcher err=%v", secondErr)
	}
	close(stub.release)
	if err := <-first; err != nil {
		t.Fatalf("owner dispatcher err=%v", err)
	}
	stub.mu.Lock()
	starts := stub.starts
	stub.mu.Unlock()
	if starts != 1 {
		t.Fatalf("owner starts=%d, expected one fenced dispatch", starts)
	}
}

func TestCoordinatorFailsClosedForUnqualifiedNativeGoal(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	coordinator := NewCoordinator(service, &coordinatorWorkflowStub{})
	if _, err := coordinator.Start(context.Background(), StartRequest{WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "native-one", Mode: "native-goal", Upper: Usage{Tokens: 10, WallSeconds: 10}}); err == nil {
		t.Fatal("native dispatch was accepted without a native owner")
	}
}

func TestCoordinatorRejectsEffectOutsideApprovedSetBeforeOwnerStart(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	stub := &coordinatorWorkflowStub{started: workflowcontract.Start{ExecutionID: "should-not-start"}}
	coordinator := NewCoordinator(service, stub)
	_, err := coordinator.Start(context.Background(), StartRequest{
		WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "effect-fence", Mode: "workflow-fallback", Upper: Usage{Tokens: 5000, WallSeconds: 300},
		Effects: []Effect{{Identity: "provider.paid", Parameters: map[string]string{"vendor": "external"}}},
	})
	if !errors.Is(err, ErrEffectNotAllowed) || !errors.Is(err, ErrDenied) {
		t.Fatalf("outside effect err=%v", err)
	}
	if stub.starts != 0 {
		t.Fatalf("owner was called for denied effect: %d", stub.starts)
	}
	state, _ = service.Get(context.Background(), proposal.WorkItem)
	if len(state.Attempts) != 0 || state.Reserved != (Usage{}) {
		t.Fatalf("denied effect changed engagement: %+v", state)
	}
}

func TestCoordinatorRejectsOwnerRevisionMismatch(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	stub := &coordinatorWorkflowStub{started: workflowcontract.Start{ExecutionID: "agent-execution-wrong", WorkflowDigest: "sha256:other"}}
	coordinator := NewCoordinator(service, stub)
	if _, err := coordinator.Start(context.Background(), StartRequest{WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "digest-fence", Mode: "workflow-fallback", Upper: Usage{Tokens: 10, WallSeconds: 10}, Invocation: workflowcontract.Invocation{WorkflowDigest: "sha256:expected"}}); err == nil {
		t.Fatal("owner revision mismatch was accepted")
	}
	state, _ = service.Get(context.Background(), proposal.WorkItem)
	if state.Reserved == (Usage{}) {
		t.Fatal("mismatched start lost reservation instead of preserving reconciliation state")
	}
}

func TestCoordinatorSettlesOnlyMeasuredTerminalUsage(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	stub := &coordinatorWorkflowStub{started: workflowcontract.Start{ExecutionID: "agent-execution-2"}, completion: workflowcontract.Completion{ExecutionID: "agent-execution-2", Usage: &workflowcontract.Usage{Tokens: 120, TokensKnown: true}}}
	coordinator := NewCoordinator(service, stub)
	if _, err := coordinator.Start(context.Background(), StartRequest{WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "repair-two", Mode: "workflow-fallback", Upper: Usage{Tokens: 5000, WallSeconds: 300}, Invocation: workflowcontract.Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/development"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.Collect(context.Background(), proposal.WorkItem, "repair-two", "agent-execution-2"); err == nil {
		t.Fatal("missing terminal wall usage was settled")
	}
	state, _ = service.Get(context.Background(), proposal.WorkItem)
	if state.Reserved.Tokens != 5000 || state.Used.Tokens != 0 {
		t.Fatalf("reservation changed after unknown usage: %+v", state)
	}
	stub.completion.Usage.WallSeconds = 12
	settled, err := coordinator.Collect(context.Background(), proposal.WorkItem, "repair-two", "agent-execution-2")
	if err != nil || settled.Used.Tokens != 120 || settled.Used.WallSeconds != 12 {
		t.Fatalf("settled=%+v err=%v", settled, err)
	}
}

func TestCoordinatorCancelPropagatesBeforeRevocation(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	stub := &coordinatorWorkflowStub{started: workflowcontract.Start{ExecutionID: "agent-execution-cancel"}}
	coordinator := NewCoordinator(service, stub)
	started, err := coordinator.Start(context.Background(), StartRequest{WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "repair-cancel", Mode: "workflow-fallback", Upper: Usage{Tokens: 5000, WallSeconds: 300}, Invocation: workflowcontract.Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/development"}})
	if err != nil {
		t.Fatal(err)
	}
	state, err = coordinator.Cancel(operatorContext(), proposal.WorkItem, "repair-cancel", started.Execution.ExecutionID, started.Engagement.Version, "operator stopped")
	if err != nil || !stub.cancelled || state.Status != "cancelled" {
		t.Fatalf("cancel state=%+v err=%v propagated=%t", state, err, stub.cancelled)
	}
}

func TestCoordinatorPersistsRevocationBeforeOwnerCancellationFailure(t *testing.T) {
	service, proposal, _ := setupService(t)
	state, _ := service.Get(context.Background(), proposal.WorkItem)
	stub := &coordinatorWorkflowStub{started: workflowcontract.Start{ExecutionID: "agent-execution-cancel-failed"}, cancelErr: errors.New("owner unavailable")}
	coordinator := NewCoordinator(service, stub)
	started, err := coordinator.Start(context.Background(), StartRequest{WorkItem: proposal.WorkItem, Digest: state.Digest, AttemptKey: "repair-cancel-failed", Mode: "workflow-fallback", Upper: Usage{Tokens: 5000, WallSeconds: 300}, Invocation: workflowcontract.Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/development"}})
	if err != nil {
		t.Fatal(err)
	}
	state, err = coordinator.Cancel(operatorContext(), proposal.WorkItem, "repair-cancel-failed", started.Execution.ExecutionID, started.Engagement.Version, "owner outage")
	if !errors.Is(err, stub.cancelErr) || state.Status != "cancelled" || state.Cancellation == nil || state.Cancellation.State != "requested" {
		t.Fatalf("revocation=%+v err=%v", state, err)
	}
	if _, err := service.Reserve(operatorContext(), proposal.WorkItem, state.Digest, "after-revoke", "workflow-fallback", Usage{1, 1}); !errors.Is(err, ErrDenied) {
		t.Fatalf("dispatch after durable revocation err=%v", err)
	}
}
