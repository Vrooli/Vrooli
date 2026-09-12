package transitioncatalog

import (
	"context"
	"path/filepath"
	"testing"

	"swarm-manager/internal/stats"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitions"

	"connectrpc.com/connect"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

type fakeRunner struct {
	started                  transitionrun.Correlation
	applied                  transitionrun.Correlation
	startKey, startSubject   string
	applyKey, applyExecution string
}

type fakeDeterministic struct{ key, subject string }

func (f *fakeDeterministic) Dispatch(_ context.Context, key, subject string) (string, error) {
	f.key, f.subject = key, subject
	return "achieved", nil
}

func (f *fakeRunner) Start(_ context.Context, key, subjectRef string) (transitionrun.Correlation, error) {
	f.startKey, f.startSubject = key, subjectRef
	return f.started, nil
}

// [REQ:SWM-P0-014] deterministic transitions dispatch through the catalog.
func TestTransitionServiceDispatchesDeterministicTransition(t *testing.T) {
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &fakeDeterministic{}
	response, err := NewService(registry, nil, dispatcher).StartTransition(context.Background(), connect.NewRequest(&api.StartTransitionRequest{TransitionKey: "goal.close_out", SubjectRef: &api.SubjectReference{Subject: "goal", Value: "release"}}))
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
	f.applyKey, f.applyExecution = key, executionID
	return f.applied, nil
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
		applied: transitionrun.Correlation{ExecutionID: "exec-1", TransitionKey: "capture.classify", SubjectRef: "cap-1", Outcome: "classified", AppliedTime: "2026-07-28T00:00:00Z"},
	}
	service := NewService(registry, runner)
	started, err := service.StartTransition(context.Background(), connect.NewRequest(&api.StartTransitionRequest{TransitionKey: "capture.classify", SubjectRef: &api.SubjectReference{Subject: "capture", Value: "cap-1"}}))
	if err != nil {
		t.Fatal(err)
	}
	if runner.startKey != "capture.classify" || runner.startSubject != "cap-1" || started.Msg.GetExecutionId() != "exec-1" {
		t.Fatalf("start dispatch = %#v, response = %#v", runner, started.Msg)
	}
	applied, err := service.ApplyTransition(context.Background(), connect.NewRequest(&api.ApplyTransitionRequest{TransitionKey: "capture.classify", ExecutionId: "exec-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if runner.applyKey != "capture.classify" || runner.applyExecution != "exec-1" || applied.Msg.GetOutcome() != "classified" {
		t.Fatalf("apply dispatch = %#v, response = %#v", runner, applied.Msg)
	}
}
