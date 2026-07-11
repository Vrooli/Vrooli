package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"test-genie/internal/orchestrator"
)

type serviceEngine struct {
	result *orchestrator.SuiteExecutionResult
	err    error
}

func (e serviceEngine) ExecuteWithEvents(_ context.Context, _ orchestrator.SuiteExecutionRequest, _ orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	return e.result, e.err
}

type serviceRecorder struct{ records []*SuiteExecutionRecord }

func (r *serviceRecorder) Create(_ context.Context, record *SuiteExecutionRecord) error {
	r.records = append(r.records, record)
	return nil
}

func TestSuiteExecutionServicePersistsRunEvidence(t *testing.T) {
	recorder := &serviceRecorder{}
	service := NewSuiteExecutionService(serviceEngine{result: &orchestrator.SuiteExecutionResult{RunID: "run-1", ScenarioName: "demo", StartedAt: time.Now().Add(-time.Second), CompletedAt: time.Now(), Success: true}}, recorder)
	result, err := service.Execute(context.Background(), SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionID.String() == "" || len(recorder.records) != 1 || recorder.records[0].RunID != "run-1" {
		t.Fatalf("result=%+v records=%+v", result, recorder.records)
	}
}

func TestSuiteExecutionServiceRecordsTerminalEngineFailure(t *testing.T) {
	recorder := &serviceRecorder{}
	service := NewSuiteExecutionService(serviceEngine{err: errors.New("boom")}, recorder)
	if _, err := service.Execute(context.Background(), SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo", RunID: "run-2"}}); err == nil {
		t.Fatal("expected engine failure")
	}
	if len(recorder.records) != 1 || recorder.records[0].RunID != "run-2" || recorder.records[0].TerminalOutcome == "" {
		t.Fatalf("records=%+v", recorder.records)
	}
}
