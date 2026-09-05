package pipeline

import (
	"context"
	"testing"
)

func newManagerTestIndex(t *testing.T) *ScenarioIndexStore {
	t.Helper()
	store, err := NewScenarioIndexStore(t.TempDir(), WithIndexStoreTimeFunc(func() int64 { return 100 }))
	if err != nil {
		t.Fatalf("NewScenarioIndexStore: %v", err)
	}
	return store
}

func TestManagerGetOrCreateActivePipelineUsesExistingThenRepairsStaleIndex(t *testing.T) {
	index := newManagerTestIndex(t)
	orchestrator := &mockOrchestrator{getResult: &Status{PipelineID: "existing", Status: StatusIdle}, getFound: true}
	if err := index.SetActivePipeline("demo", "existing"); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(WithManagerIndexStore(index), WithManagerOrchestrator(orchestrator))

	status, created, err := manager.GetOrCreateActivePipeline(context.Background(), "demo", nil)
	if err != nil || created || status.PipelineID != "existing" {
		t.Fatalf("existing active pipeline = %#v, created=%v, err=%v", status, created, err)
	}

	orchestrator.getFound = false
	orchestrator.createResult = &Status{PipelineID: "replacement", Status: StatusIdle}
	status, created, err = manager.GetOrCreateActivePipeline(context.Background(), "demo", &Config{Platforms: []string{"linux-amd64"}})
	if err != nil || !created || status.PipelineID != "replacement" {
		t.Fatalf("replacement active pipeline = %#v, created=%v, err=%v", status, created, err)
	}
	if got := index.Get("demo").ActivePipelineID; got != "replacement" {
		t.Fatalf("active index = %q, want replacement", got)
	}
}

func TestManagerHistoryAndRunningReflectOrchestratorState(t *testing.T) {
	index := newManagerTestIndex(t)
	if err := index.SetActivePipeline("demo", "running"); err != nil {
		t.Fatal(err)
	}
	if _, err := index.ArchiveActive("demo"); err != nil {
		t.Fatal(err)
	}
	if err := index.SetActivePipeline("demo", "running"); err != nil {
		t.Fatal(err)
	}
	running := &Status{PipelineID: "running", Status: StatusRunning}
	orchestrator := &mockOrchestrator{getResult: running, getFound: true}
	manager := NewManager(WithManagerIndexStore(index), WithManagerOrchestrator(orchestrator))
	if !manager.IsRunning("demo") {
		t.Fatal("running pipeline was not reported as running")
	}
	orchestrator.pipelines = []*Status{running}
	history, total, err := manager.GetPipelineHistory("demo", 10)
	if err != nil || total != 1 || len(history) != 1 || history[0].PipelineID != "running" {
		t.Fatalf("history = %#v, total=%d, err=%v", history, total, err)
	}
	orchestrator.getResult.Status = StatusIdle
	if manager.IsRunning("demo") {
		t.Fatal("idle pipeline must not be reported as running")
	}
}

func TestManagerCreateNewAndResetArchiveThePreviousPipeline(t *testing.T) {
	index := newManagerTestIndex(t)
	if err := index.SetActivePipeline("demo", "old"); err != nil {
		t.Fatal(err)
	}
	orchestrator := &mockOrchestrator{createResult: &Status{PipelineID: "new", Status: StatusIdle}}
	manager := NewManager(WithManagerIndexStore(index), WithManagerOrchestrator(orchestrator))
	status, archived, err := manager.CreateNewPipeline(context.Background(), "demo", &Config{Framework: FrameworkElectron})
	if err != nil || archived != "old" || status.PipelineID != "new" {
		t.Fatalf("CreateNewPipeline = %#v, archived=%q, err=%v", status, archived, err)
	}
	if got := index.Get("demo").ActivePipelineID; got != "new" {
		t.Fatalf("active pipeline = %q, want new", got)
	}
	archived, err = manager.ResetActivePipeline("demo")
	if err != nil || archived != "new" {
		t.Fatalf("ResetActivePipeline = %q, %v; want new, nil", archived, err)
	}
	if got := index.Get("demo").ActivePipelineID; got != "" {
		t.Fatalf("active pipeline after reset = %q, want empty", got)
	}
}

func TestManagerStartActivePipelineUpdatesIdleConfigAndHandlesRunningState(t *testing.T) {
	index := newManagerTestIndex(t)
	if err := index.SetActivePipeline("demo", "idle"); err != nil {
		t.Fatal(err)
	}
	idle := &Status{PipelineID: "idle", Status: StatusIdle}
	started := &Status{PipelineID: "idle", Status: StatusRunning}
	orchestrator := &mockOrchestrator{getResult: idle, getFound: true, startResult: started}
	manager := NewManager(WithManagerIndexStore(index), WithManagerOrchestrator(orchestrator))
	overrides := &Config{ScenarioName: "demo", Platforms: []string{"linux"}}
	status, err := manager.StartActivePipeline(context.Background(), "demo", overrides)
	if err != nil || status != started || orchestrator.updatedConfig != overrides {
		t.Fatalf("start idle = %#v, %v; updated=%#v", status, err, orchestrator.updatedConfig)
	}

	idle.Status = StatusRunning
	status, err = manager.StartActivePipeline(context.Background(), "demo", nil)
	if err != nil || status != idle {
		t.Fatalf("start running = %#v, %v", status, err)
	}
}

func TestManagerStartActivePipelineFailsGracefullyWithoutConfigUpdateCapability(t *testing.T) {
	index := newManagerTestIndex(t)
	if err := index.SetActivePipeline("demo", "idle"); err != nil {
		t.Fatal(err)
	}
	// The normal mock supports the capability; wrap it in an Orchestrator-only
	// adapter to prove Manager returns an actionable error instead of panicking.
	base := &orchestratorOnly{mockOrchestrator: &mockOrchestrator{getResult: &Status{PipelineID: "idle", Status: StatusIdle}, getFound: true}}
	manager := NewManager(WithManagerIndexStore(index), WithManagerOrchestrator(base))
	if _, err := manager.StartActivePipeline(context.Background(), "demo", &Config{ScenarioName: "demo"}); err == nil {
		t.Fatal("expected missing config-update capability error")
	}
}

func TestManagerStartActivePipelineBlockingStartsIdleAndReplacesTerminalPipeline(t *testing.T) {
	t.Run("idle", func(t *testing.T) {
		index := newManagerTestIndex(t)
		if err := index.SetActivePipeline("demo", "idle"); err != nil {
			t.Fatal(err)
		}
		orchestrator := &mockOrchestrator{
			getResult: &Status{PipelineID: "idle", Status: StatusIdle}, getFound: true,
			startResult: &Status{PipelineID: "idle", Status: StatusCompleted},
		}
		manager := NewManager(WithManagerIndexStore(index), WithManagerOrchestrator(orchestrator))
		status, err := manager.StartActivePipelineBlocking(context.Background(), "demo", &Config{ScenarioName: "demo"}, 1)
		if err != nil || status.Status != StatusCompleted || orchestrator.updatedConfig == nil {
			t.Fatalf("blocking idle start = %#v, %v; updated=%#v", status, err, orchestrator.updatedConfig)
		}
	})

	t.Run("terminal creates replacement", func(t *testing.T) {
		index := newManagerTestIndex(t)
		if err := index.SetActivePipeline("demo", "old"); err != nil {
			t.Fatal(err)
		}
		orchestrator := &mockOrchestrator{
			getResult: &Status{PipelineID: "old", Status: StatusFailed}, getFound: true,
			createResult: &Status{PipelineID: "new", Status: StatusIdle},
			startResult:  &Status{PipelineID: "new", Status: StatusCompleted},
		}
		manager := NewManager(WithManagerIndexStore(index), WithManagerOrchestrator(orchestrator))
		status, err := manager.StartActivePipelineBlocking(context.Background(), "demo", nil, 1)
		if err != nil || status.PipelineID != "new" || index.Get("demo").ActivePipelineID != "new" {
			t.Fatalf("blocking terminal replacement = %#v, %v; index=%#v", status, err, index.Get("demo"))
		}
	})
}

type orchestratorOnly struct{ mockOrchestrator *mockOrchestrator }

func (o *orchestratorOnly) RunPipeline(ctx context.Context, config *Config) (*Status, error) {
	return o.mockOrchestrator.RunPipeline(ctx, config)
}

func (o *orchestratorOnly) CreateIdlePipeline(config *Config) (*Status, error) {
	return o.mockOrchestrator.CreateIdlePipeline(config)
}

func (o *orchestratorOnly) StartPipeline(ctx context.Context, id string) (*Status, error) {
	return o.mockOrchestrator.StartPipeline(ctx, id)
}

func (o *orchestratorOnly) RunPipelineBlocking(ctx context.Context, config *Config, timeout int) (*Status, error) {
	return o.mockOrchestrator.RunPipelineBlocking(ctx, config, timeout)
}

func (o *orchestratorOnly) StartPipelineBlocking(ctx context.Context, id string, timeout int) (*Status, error) {
	return o.mockOrchestrator.StartPipelineBlocking(ctx, id, timeout)
}

func (o *orchestratorOnly) ResumePipeline(ctx context.Context, id string, config *Config) (*Status, error) {
	return o.mockOrchestrator.ResumePipeline(ctx, id, config)
}

func (o *orchestratorOnly) GetStatus(id string) (*Status, bool) {
	return o.mockOrchestrator.GetStatus(id)
}

func (o *orchestratorOnly) CancelPipeline(id string) bool {
	return o.mockOrchestrator.CancelPipeline(id)
}
func (o *orchestratorOnly) ListPipelines() []*Status { return o.mockOrchestrator.ListPipelines() }
