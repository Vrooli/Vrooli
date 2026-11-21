package transitioncatalog

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/stats"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitions"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

type fakeRunner struct {
	started                  transitionrun.Correlation
	applied                  transitionrun.Correlation
	correlation              transitionrun.Correlation
	correlationErr           error
	correlationID            string
	startKey, startSubject   string
	applyKey, applyExecution string
	startCalls, applyCalls   int
}

type fakeDeterministic struct{ key, subject string }

func (f *fakeDeterministic) Dispatch(_ context.Context, key, subject string) (string, error) {
	f.key, f.subject = key, subject
	return "achieved", nil
}

func (f *fakeRunner) Start(_ context.Context, key, subjectRef string) (transitionrun.Correlation, error) {
	f.startCalls++
	f.startKey, f.startSubject = key, subjectRef
	return f.started, nil
}

func (f *fakeRunner) GetCorrelation(executionID string) (transitionrun.Correlation, error) {
	f.correlationID = executionID
	return f.correlation, f.correlationErr
}

// [REQ:SWM-P0-014] deterministic transitions dispatch through the catalog.
func TestTransitionServiceDispatchesDeterministicTransition(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &fakeDeterministic{}
	response, err := NewService(registry, nil, dispatcher).StartTransition(humanSwarmContext(), connect.NewRequest(&api.StartTransitionRequest{TransitionKey: "goal.close_out", SubjectRef: &api.SubjectReference{Subject: "goal", Value: "release"}}))
	if err != nil {
		t.Fatal(err)
	}
	if dispatcher.key != "goal.close_out" || dispatcher.subject != "release" || response.Msg.GetEntityVersion() != "achieved" {
		t.Fatalf("deterministic dispatch = %#v response=%#v", dispatcher, response.Msg)
	}
}

func TestTransitionServiceRejectsUndeclaredSubjectReference(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewService(registry, nil).StartTransition(context.Background(), connect.NewRequest(&api.StartTransitionRequest{
		TransitionKey: "capture.classify",
		SubjectRef:    &api.SubjectReference{Subject: "goal", Value: "cap-1"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("subject mismatch error = %v, want invalid argument", err)
	}
}

func (f *fakeRunner) Apply(_ context.Context, key, executionID string) (transitionrun.Correlation, error) {
	f.applyCalls++
	f.applyKey, f.applyExecution = key, executionID
	return f.applied, nil
}

func humanSwarmContext() context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"swarm-manager:write"}})
}

func captureBookkeepingContext(target string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "run-1", Verified: true, Scopes: []string{"swarm-manager:write"},
		Grant: &identity.OperationGrant{ID: "existing-bookkeeping", Approver: "operator", Operations: []string{"swarm.bookkeeping"}, Targets: []string{target}, ExpiresAt: time.Now().Add(time.Hour)},
	})
}

func TestListTransitionsReturnsEveryDeclaredDefinition(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("load transition registry: %v", err)
	}
	response, err := NewService(registry, nil).ListTransitions(context.Background(), connect.NewRequest(&api.ListTransitionsRequest{}))
	if err != nil {
		t.Fatalf("ListTransitions: %v", err)
	}
	definitions := registry.Definitions()
	if len(response.Msg.Transitions) != len(definitions) {
		t.Fatalf("transitions = %d, want %d", len(response.Msg.Transitions), len(definitions))
	}
	for index, definition := range definitions {
		actual := response.Msg.Transitions[index]
		if actual.GetKey() != definition.Key || actual.GetSubject() != definition.Subject || actual.GetApplyAction() != definition.ApplyAction || actual.GetInputContract() != definition.InputContract {
			t.Fatalf("transition %d = %#v, want registry definition %#v", index, actual, definition)
		}
		if len(actual.GetHumanGates()) != len(definition.HumanGates) {
			t.Fatalf("transition %q gates = %d, want %d", definition.Key, len(actual.GetHumanGates()), len(definition.HumanGates))
		}
	}
}

func TestListTransitionsProjectsLiveGateEvidence(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewServiceWithGateProjection(registry, nil, func() (map[string]string, map[string]stats.KindRate) {
		return map[string]string{"capture-to-suggested": "auto"}, map[string]stats.KindRate{"capture-to-suggested": {Rate: .9, SampleSize: 20}}
	})
	response, err := service.ListTransitions(context.Background(), connect.NewRequest(&api.ListTransitionsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, transition := range response.Msg.GetTransitions() {
		for _, gate := range transition.GetHumanGates() {
			if gate.GetId() != "capture-to-suggested" {
				continue
			}
			if gate.GetMode() != "auto" || gate.GetSampleSize() != 20 || gate.GetReadiness() != "ready" {
				t.Fatalf("live gate projection = %#v", gate)
			}
			return
		}
	}
	t.Fatal("capture-to-suggested gate not found")
}

// [REQ:SWM-P0-014] workflow transitions start and apply through the catalog.
func TestTransitionServiceDispatchesGenericStartAndApply(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{
		started: transitionrun.Correlation{ExecutionID: "exec-1", DefinitionDigest: "sha256:def", EntityVersion: "v1"},
		applied: transitionrun.Correlation{ExecutionID: "exec-1", SubjectKind: "capture", TransitionKey: "capture.classify", SubjectRef: "cap-1", Outcome: "classified", AppliedTime: "2026-07-28T00:00:00Z"},
	}
	service := NewService(registry, runner)
	started, err := service.StartTransition(humanSwarmContext(), connect.NewRequest(&api.StartTransitionRequest{TransitionKey: "capture.classify", SubjectRef: &api.SubjectReference{Subject: "capture", Value: "cap-1"}}))
	if err != nil {
		t.Fatal(err)
	}
	if runner.startKey != "capture.classify" || runner.startSubject != "cap-1" || started.Msg.GetExecutionId() != "exec-1" {
		t.Fatalf("start dispatch = %#v, response = %#v", runner, started.Msg)
	}
	serviceRunnerCorrelation(runner, transitionrun.Correlation{ExecutionID: "exec-1", SubjectKind: "capture", TransitionKey: "capture.classify", SubjectRef: "cap-1"})
	applied, err := service.ApplyTransition(humanSwarmContext(), connect.NewRequest(&api.ApplyTransitionRequest{TransitionKey: "capture.classify", ExecutionId: "exec-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if runner.applyKey != "capture.classify" || runner.applyExecution != "exec-1" || applied.Msg.GetOutcome() != "classified" {
		t.Fatalf("apply dispatch = %#v, response = %#v", runner, applied.Msg)
	}
}

func serviceRunnerCorrelation(runner *fakeRunner, correlation transitionrun.Correlation) {
	runner.correlation = correlation
}

func TestCaptureClassificationStartRequiresExactBookkeepingOperationBeforeRunner(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, target string
		wantError    connect.Code
	}{
		{name: "exact existing action", target: "swarm-manager:capture/cap-1"},
		{name: "wrong object", target: "swarm-manager:capture/other", wantError: connect.CodePermissionDenied},
		{name: "wrong operation", target: "swarm-manager:capture/cap-1", wantError: connect.CodePermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &fakeRunner{started: transitionrun.Correlation{ExecutionID: "exec-1"}}
			ctx := captureBookkeepingContext(tc.target)
			if tc.name == "wrong operation" {
				ctx = identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Verified: true, Scopes: []string{"swarm-manager:write"}, Grant: &identity.OperationGrant{ID: "existing", Approver: "operator", Operations: []string{"agent.manage"}, Targets: []string{tc.target}, ExpiresAt: time.Now().Add(time.Hour)}})
			}
			_, err := NewService(registry, runner).StartTransition(ctx, connect.NewRequest(&api.StartTransitionRequest{TransitionKey: "capture.classify", SubjectRef: &api.SubjectReference{Subject: "capture", Value: "cap-1"}}))
			if tc.wantError == 0 && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantError != 0 && connect.CodeOf(err) != tc.wantError {
				t.Fatalf("code = %v, want %v (err=%v)", connect.CodeOf(err), tc.wantError, err)
			}
			wantCalls := 0
			if tc.wantError == 0 {
				wantCalls = 1
			}
			if runner.startCalls != wantCalls {
				t.Fatalf("runner start calls=%d, want %d", runner.startCalls, wantCalls)
			}
		})
	}
}

func TestCaptureClassificationStartRejectsPathLikeSubjectBeforeRunner(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	_, err = NewService(registry, runner).StartTransition(captureBookkeepingContext("swarm-manager:*"), connect.NewRequest(&api.StartTransitionRequest{TransitionKey: "capture.classify", SubjectRef: &api.SubjectReference{Subject: "capture", Value: "../secret"}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("path-like subject code=%v, err=%v", connect.CodeOf(err), err)
	}
	if runner.startCalls != 0 {
		t.Fatalf("runner received path-like subject: starts=%d", runner.startCalls)
	}
}

func TestCaptureClassificationApplyAuthorizesPersistedCorrelationBeforeMutation(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		correlation    transitionrun.Correlation
		correlationErr error
		target         string
		wantError      connect.Code
	}{
		{name: "authorized persisted subject", correlation: transitionrun.Correlation{ExecutionID: "exec-1", SubjectKind: "capture", TransitionKey: "capture.classify", SubjectRef: "cap-1"}, target: "swarm-manager:capture/cap-1"},
		{name: "wrong immutable subject", correlation: transitionrun.Correlation{ExecutionID: "exec-1", SubjectKind: "capture", TransitionKey: "capture.classify", SubjectRef: "cap-other"}, target: "swarm-manager:capture/cap-1", wantError: connect.CodePermissionDenied},
		{name: "wrong transition", correlation: transitionrun.Correlation{ExecutionID: "exec-1", SubjectKind: "goal", TransitionKey: "goal.plan", SubjectRef: "goal-1"}, target: "swarm-manager:capture/cap-1", wantError: connect.CodePermissionDenied},
		{name: "wrong execution", correlation: transitionrun.Correlation{ExecutionID: "exec-other", SubjectKind: "capture", TransitionKey: "capture.classify", SubjectRef: "cap-1"}, target: "swarm-manager:capture/cap-1", wantError: connect.CodePermissionDenied},
		{name: "wrong subject kind", correlation: transitionrun.Correlation{ExecutionID: "exec-1", SubjectKind: "goal", TransitionKey: "capture.classify", SubjectRef: "cap-1"}, target: "swarm-manager:capture/cap-1", wantError: connect.CodePermissionDenied},
		{name: "correlation read error", correlationErr: errors.New("missing"), target: "swarm-manager:capture/cap-1", wantError: connect.CodePermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &fakeRunner{correlation: tc.correlation, applied: transitionrun.Correlation{Outcome: "suggested"}}
			executionID := "exec-1"
			if tc.name == "authorized persisted subject" {
				executionID = " exec-1 "
			}
			_, err := NewService(registry, runner).ApplyTransition(captureBookkeepingContext(tc.target), connect.NewRequest(&api.ApplyTransitionRequest{TransitionKey: "capture.classify", ExecutionId: executionID}))
			if runner.correlationID != "exec-1" {
				t.Fatalf("correlation lookup ID=%q, want normalized exec-1", runner.correlationID)
			}
			if tc.wantError == 0 && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantError != 0 && connect.CodeOf(err) != tc.wantError {
				t.Fatalf("code = %v, want %v (err=%v)", connect.CodeOf(err), tc.wantError, err)
			}
			wantCalls := 0
			if tc.wantError == 0 {
				wantCalls = 1
			}
			if runner.applyCalls != wantCalls {
				t.Fatalf("runner apply calls=%d, want %d", runner.applyCalls, wantCalls)
			}
			if wantCalls == 1 && runner.applyExecution != "exec-1" {
				t.Fatalf("runner apply ID=%q, want normalized exec-1", runner.applyExecution)
			}
		})
	}
}

func TestOtherTransitionStartRemainsHumanOnly(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	_, err = NewService(registry, runner).StartTransition(captureBookkeepingContext("swarm-manager:*"), connect.NewRequest(&api.StartTransitionRequest{TransitionKey: "goal.plan", SubjectRef: &api.SubjectReference{Subject: "goal", Value: "g-1"}}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("agent generic transition code=%v, err=%v", connect.CodeOf(err), err)
	}
	if runner.startCalls != 0 {
		t.Fatalf("generic transition reached runner: %d starts", runner.startCalls)
	}
}

func TestOtherTransitionApplyRemainsHumanOnly(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{correlation: transitionrun.Correlation{TransitionKey: "goal.plan", SubjectRef: "g-1"}}
	_, err = NewService(registry, runner).ApplyTransition(captureBookkeepingContext("swarm-manager:*"), connect.NewRequest(&api.ApplyTransitionRequest{TransitionKey: "goal.plan", ExecutionId: "exec-1"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("agent generic transition code=%v, err=%v", connect.CodeOf(err), err)
	}
	if runner.applyCalls != 0 {
		t.Fatalf("generic transition reached runner: %d applies", runner.applyCalls)
	}
}

func TestTransitionServiceRefusesSubjectOwnedMutation(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	_, err = NewService(registry, runner).StartTransition(humanSwarmContext(), connect.NewRequest(&api.StartTransitionRequest{
		TransitionKey: "plan.execute",
		SubjectRef:    &api.SubjectReference{Subject: "backlog-item", Value: "plan-1"},
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("subject-owned transition code=%v, err=%v", connect.CodeOf(err), err)
	}
	if runner.startCalls != 0 {
		t.Fatalf("subject-owned transition reached generic runner: %d starts", runner.startCalls)
	}
}
