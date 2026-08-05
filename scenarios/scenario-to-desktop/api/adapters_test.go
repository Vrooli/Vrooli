package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/records"
	"scenario-to-desktop-api/smoketest"
)

func TestBuildStoreAdaptersPreserveOperatorVisibleState(t *testing.T) {
	store := build.NewStore()
	createdAt := time.Now().Add(-time.Minute).UTC()
	store.Save(&build.Status{BuildID: "build-1", Status: "building", OutputPath: "/tmp/desktop", CreatedAt: createdAt, Metadata: map[string]interface{}{"source": "test"}, BuildLog: []string{"started"}, Artifacts: map[string]string{"linux": "appimage"}})

	systemAdapter := &systemBuildStoreAdapter{store: store}
	if snapshot := systemAdapter.Snapshot(); snapshot["build-1"].Status != "building" {
		t.Fatalf("system snapshot = %#v", snapshot)
	}

	recordsAdapter := &recordsBuildStoreAdapter{store: store}
	view, ok := recordsAdapter.Get("build-1")
	if !ok || view.OutputPath != "/tmp/desktop" {
		t.Fatalf("record view = %#v, %v", view, ok)
	}
	if err := recordsAdapter.Update("build-1", func(status *records.BuildStatusView) {
		status.Status = "ready"
		status.OutputPath = "/tmp/released"
	}); err != nil {
		t.Fatalf("update build record: %v", err)
	}
	if updated, ok := recordsAdapter.Get("build-1"); !ok || updated.Status != "ready" || updated.OutputPath != "/tmp/released" {
		t.Fatalf("updated record view = %#v, %v", updated, ok)
	}
	if err := recordsAdapter.Update("missing", func(*records.BuildStatusView) {}); err == nil {
		t.Fatal("missing build update succeeded")
	}
}

func TestRecordAdaptersExposeGeneratedArtifactAndSmokeEvidence(t *testing.T) {
	recordStore, err := records.NewFileStore(t.TempDir() + "/records.json")
	if err != nil {
		t.Fatal(err)
	}
	record := &generation.DesktopAppRecord{ID: "record-1", BuildID: "build-3", ScenarioName: "calculator", AppDisplayName: "Calculator", TemplateType: "basic", Framework: "electron", OutputPath: "/tmp/calculator", LocationMode: "proper"}
	if err := (&generationRecordStoreAdapter{store: recordStore}).Upsert(record); err != nil {
		t.Fatalf("persist generated record: %v", err)
	}
	list := (&scenarioRecordStoreAdapter{store: recordStore}).List()
	if len(list) != 1 || list[0].ScenarioName != "calculator" || list[0].OutputPath != "/tmp/calculator" {
		t.Fatalf("scenario records = %#v", list)
	}
	if (&scenarioRecordStoreAdapter{}).List() != nil {
		t.Fatal("nil record store should expose no records")
	}
	if err := (&generationRecordStoreAdapter{}).Upsert(record); err != nil {
		t.Fatalf("nil generation record store should remain non-fatal: %v", err)
	}

	smokeStore := smoketest.NewInMemoryStore()
	smokeStore.Save(&smoketest.Status{SmokeTestID: "smoke-1", ScenarioName: "calculator", ScreenRecording: &smoketest.RecordingStatus{Recorded: true, CaptureID: "capture-1"}})
	smokeID, recording, ok := (&smokeTestRecordAdapter{store: smokeStore}).GetByScenario("calculator")
	if !ok || smokeID != "smoke-1" || recording == nil || !recording.Recorded || recording.CaptureID != "capture-1" {
		t.Fatalf("smoke evidence = %q %#v %v", smokeID, recording, ok)
	}
	if _, _, ok := (&smokeTestRecordAdapter{store: smokeStore}).GetByScenario("missing"); ok {
		t.Fatal("missing smoke evidence was found")
	}
}

type adapterOrchestrator struct{ status *pipeline.Status }

func (f adapterOrchestrator) RunPipeline(context.Context, *pipeline.Config) (*pipeline.Status, error) {
	return nil, nil
}

func (f adapterOrchestrator) CreateIdlePipeline(*pipeline.Config) (*pipeline.Status, error) {
	return nil, nil
}

func (f adapterOrchestrator) StartPipeline(context.Context, string) (*pipeline.Status, error) {
	return nil, nil
}

func (f adapterOrchestrator) RunPipelineBlocking(context.Context, *pipeline.Config, int) (*pipeline.Status, error) {
	return nil, nil
}

func (f adapterOrchestrator) StartPipelineBlocking(context.Context, string, int) (*pipeline.Status, error) {
	return nil, nil
}

func (f adapterOrchestrator) ResumePipeline(context.Context, string, *pipeline.Config) (*pipeline.Status, error) {
	return nil, nil
}

func (f adapterOrchestrator) GetStatus(id string) (*pipeline.Status, bool) {
	return f.status, f.status != nil && id == f.status.PipelineID
}
func (f adapterOrchestrator) CancelPipeline(string) bool        { return false }
func (f adapterOrchestrator) ListPipelines() []*pipeline.Status { return nil }

func TestPipelineAdapterForwardsStatusLookup(t *testing.T) {
	status := &pipeline.Status{PipelineID: "pipeline-1"}
	adapter := &pipelineStoreAdapter{store: adapterOrchestrator{status: status}}
	if got, ok := adapter.Get("pipeline-1"); !ok || got != status {
		t.Fatalf("Get() = %#v, %v", got, ok)
	}
	if _, ok := adapter.Get("missing"); ok {
		t.Fatal("missing pipeline was found")
	}
}

func TestGenerationBuildStoreAdapterRoundTripsMutableBuildState(t *testing.T) {
	store := build.NewStore()
	adapter := &generationBuildStoreAdapter{store: store}
	created := adapter.Create("build-2")
	if created.BuildID != "build-2" || created.Status != "building" {
		t.Fatalf("created status = %#v", created)
	}
	completedAt := time.Now().UTC()
	adapter.Update("build-2", func(status *generation.BuildStatus) {
		status.Status = "ready"
		status.OutputPath = "/tmp/ready"
		status.CompletedAt = &completedAt
		status.BuildLog = append(status.BuildLog, "complete")
	})
	got, ok := adapter.Get("build-2")
	if !ok || got.Status != "ready" || got.OutputPath != "/tmp/ready" || got.CompletedAt == nil || len(got.BuildLog) != 1 {
		t.Fatalf("round-tripped status = %#v, %v", got, ok)
	}
	if _, ok := adapter.Get("missing"); ok {
		t.Fatal("missing build was found")
	}
}

type adapterExecutor struct {
	result *smoketest.ExecutionResult
	err    error
}

func (f adapterExecutor) Execute(context.Context, string, string, []string, []string, time.Duration) (string, error) {
	return "", f.err
}

func (f adapterExecutor) ExecuteWithResult(context.Context, string, string, []string, []string, time.Duration) (*smoketest.ExecutionResult, error) {
	return f.result, f.err
}
func (f adapterExecutor) LookPath(string) (string, error) { return "", nil }

func TestScreenRecordingAdapterPreservesDiagnosticOutputOnFailure(t *testing.T) {
	wantErr := errors.New("command failed")
	adapter := &screenrecordingExecutorAdapter{executor: adapterExecutor{result: &smoketest.ExecutionResult{Stdout: "stdout", Stderr: "stderr", ExitCode: 9}, err: wantErr}}
	result, err := adapter.ExecuteWithResult(context.Background(), "", "capture", nil, nil, time.Second)
	if !errors.Is(err, wantErr) || result == nil || result.Stderr != "stderr" || result.ExitCode != 9 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	adapter.executor = adapterExecutor{err: wantErr}
	if result, err := adapter.ExecuteWithResult(context.Background(), "", "capture", nil, nil, time.Second); result != nil || !errors.Is(err, wantErr) {
		t.Fatalf("nil execution result = %#v, %v", result, err)
	}
}
