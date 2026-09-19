package trials

import (
	"context"
	"testing"
	"time"

	internaltrials "meta-optimization-manager/internal/trials"

	"connectrpc.com/connect"

	trialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials"
)

type fakeService struct {
	tasks   []internaltrials.TrialTask
	runs    []internaltrials.TrialRun
	history internaltrials.History
	run     internaltrials.TrialRun
	runErr  error
	gate    internaltrials.GateCoverage
	lastReq struct{ suite, taskID string }
}

func (f *fakeService) ListTrialTasks(_ context.Context, suite string) ([]internaltrials.TrialTask, error) {
	f.lastReq.suite = suite
	return f.tasks, nil
}

func (f *fakeService) RunTrials(_ context.Context, suite, taskID string) ([]internaltrials.TrialRun, error) {
	f.lastReq.suite, f.lastReq.taskID = suite, taskID
	return f.runs, nil
}

func (f *fakeService) GetTrialHistory(_ context.Context, _, _ string) (internaltrials.History, error) {
	return f.history, nil
}

func (f *fakeService) GetTrialRun(_ context.Context, _ string) (internaltrials.TrialRun, error) {
	return f.run, f.runErr
}

func (f *fakeService) GetGateCoverage(_ context.Context) (internaltrials.GateCoverage, error) {
	return f.gate, nil
}

func TestHandlerListTasks(t *testing.T) {
	svc := &fakeService{tasks: []internaltrials.TrialTask{{ID: "trial/g1", Suite: internaltrials.SuiteAddFeature, GuideTaskID: "g1", Description: "x"}}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.ListTrialTasks(context.Background(), connect.NewRequest(&trialsv1.ListTrialTasksRequest{Suite: "add-feature"}))
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastReq.suite != "add-feature" {
		t.Fatalf("suite not threaded: %q", svc.lastReq.suite)
	}
	if len(resp.Msg.GetTasks()) != 1 || resp.Msg.GetTasks()[0].GetId() != "trial/g1" {
		t.Fatalf("tasks = %+v", resp.Msg.GetTasks())
	}
}

func TestHandlerRunTrialsThreadsArgs(t *testing.T) {
	svc := &fakeService{runs: []internaltrials.TrialRun{{ID: "r1", Verdict: internaltrials.VerdictPass, At: time.Now()}}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.RunTrials(context.Background(), connect.NewRequest(&trialsv1.RunTrialsRequest{Suite: "bugfix", TaskId: "trial/x"}))
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastReq.taskID != "trial/x" || svc.lastReq.suite != "bugfix" {
		t.Fatalf("args not threaded: %+v", svc.lastReq)
	}
	if len(resp.Msg.GetRuns()) != 1 || resp.Msg.GetRuns()[0].GetVerdict() != trialsv1.TrialVerdict_TRIAL_VERDICT_PASS {
		t.Fatalf("runs = %+v", resp.Msg.GetRuns())
	}
}

func TestHandlerGetRunNotFound(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{runErr: context.DeadlineExceeded}})
	_, err := h.GetTrialRun(context.Background(), connect.NewRequest(&trialsv1.GetTrialRunRequest{Id: "nope"}))
	if err == nil || connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestHandlerGateCoverage(t *testing.T) {
	svc := &fakeService{gate: internaltrials.GateCoverage{GuideTasksTotal: 4, GuideTasksWithGate: 1, Ratio: 0.25}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.GetGateCoverage(context.Background(), connect.NewRequest(&trialsv1.GetGateCoverageRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetGuideTasksTotal() != 4 || resp.Msg.GetGateCoverageRatio() != 0.25 {
		t.Fatalf("gate coverage = %+v", resp.Msg)
	}
}
