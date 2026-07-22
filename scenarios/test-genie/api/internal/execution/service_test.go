package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

type serviceEngine struct {
	result *orchestrator.SuiteExecutionResult
	err    error
}

func TestSuiteExecutionServicePersistsOnlyCompactPhaseHistory(t *testing.T) {
	recorder := &serviceRecorder{}
	service := NewSuiteExecutionService(serviceEngine{result: &orchestrator.SuiteExecutionResult{
		RunID: "run-compact", ScenarioName: "demo", StartedAt: time.Now().Add(-time.Second), CompletedAt: time.Now(), Success: true,
		Phases: []phases.ExecutionResult{{
			Name: "security", Status: "failed", DurationSeconds: 2, LogPath: "coverage/logs/huge.log",
			Observations: []phases.Observation{phases.NewErrorObservation("huge observation")},
			Findings:     []*architecturev1.ArchitectureFinding{{Code: "huge.finding", Message: "detailed payload"}},
		}},
	}}, recorder)
	if _, err := service.Execute(context.Background(), SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo"}}); err != nil {
		t.Fatal(err)
	}
	stored := recorder.records[0].Phases[0]
	if stored.LogPath != "" || len(stored.Observations) != 0 || len(stored.Findings) != 0 || stored.Metrics != nil {
		t.Fatalf("SQLite history retained detailed phase payload: %+v", stored)
	}
	if stored.Name != "security" || stored.Status != "failed" || stored.DurationSeconds != 2 {
		t.Fatalf("compact phase lost summary fields: %+v", stored)
	}
}

func TestSuiteExecutionServicePersistsCompactPreparationStages(t *testing.T) {
	recorder := &serviceRecorder{}
	service := NewSuiteExecutionService(serviceEngine{result: &orchestrator.SuiteExecutionResult{
		RunID: "run-stages", ScenarioName: "demo", StartedAt: time.Now().Add(-time.Second), CompletedAt: time.Now(), Success: true,
		PreparationStages: []orchestrator.PreparationStage{{Name: "provider_readiness", DurationMilliseconds: 100}, {Name: "provider_check", Parent: "provider_readiness", Subject: "unit-health", Status: "ready", DurationMilliseconds: 20}},
	}}, recorder)
	if _, err := service.Execute(context.Background(), SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo"}}); err != nil {
		t.Fatal(err)
	}
	stages := recorder.records[0].PreparationStages
	if len(stages) != 2 || stages[1].Subject != "unit-health" || stages[1].DurationMilliseconds != 20 {
		t.Fatalf("compact preparation stages = %#v", stages)
	}
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

func TestSuiteExecutionServiceCollectsRetentionOnlyAfterPersistence(t *testing.T) {
	recorder := &serviceRecorder{}
	service := NewSuiteExecutionService(serviceEngine{result: &orchestrator.SuiteExecutionResult{RunID: "run-retention", ScenarioName: "demo", StartedAt: time.Now().Add(-time.Second), CompletedAt: time.Now(), Success: true}}, recorder)
	collected := make(chan string, 1)
	service.SetRetentionCollector(func(_ context.Context, scenario string) {
		if len(recorder.records) != 1 {
			t.Errorf("retention ran before persistence: %d records", len(recorder.records))
		}
		collected <- scenario
	})
	if _, err := service.Execute(context.Background(), SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo"}}); err != nil {
		t.Fatal(err)
	}
	select {
	case scenario := <-collected:
		if scenario != "demo" {
			t.Fatalf("retention scenario = %q", scenario)
		}
	case <-time.After(time.Second):
		t.Fatal("retention collector was not called")
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
