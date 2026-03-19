package main

import (
	"context"
	"testing"
	"time"

	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/toolexecution"
)

func TestToPipelineConfig_MapsDeployFlags(t *testing.T) {
	cfg := toPipelineConfig(&toolexecution.PipelineConfig{
		ScenarioName:   "demo-scenario",
		DeployTarget:   "production",
		DeployTo:       "landing-page-business-suite",
		RemoteProfile:  "prod",
		AppKey:         "demo-app",
		StopAfterStage: pipeline.StageDeploy,
	})

	if cfg.ScenarioName != "demo-scenario" {
		t.Fatalf("expected scenario_name demo-scenario, got %q", cfg.ScenarioName)
	}
	if cfg.StopAfterStage != pipeline.StageDeploy {
		t.Fatalf("expected stop_after_stage %q, got %q", pipeline.StageDeploy, cfg.StopAfterStage)
	}
	if cfg.DeployConfig == nil {
		t.Fatalf("expected deploy config to be mapped")
	}
	if cfg.DeployConfig.TargetName != "production" {
		t.Fatalf("expected deploy target production, got %q", cfg.DeployConfig.TargetName)
	}
	if cfg.DeployConfig.ScenarioName != "landing-page-business-suite" {
		t.Fatalf("expected deploy scenario landing-page-business-suite, got %q", cfg.DeployConfig.ScenarioName)
	}
	if cfg.DeployConfig.RemoteProfile != "prod" {
		t.Fatalf("expected remote profile prod, got %q", cfg.DeployConfig.RemoteProfile)
	}
	if cfg.DeployConfig.AppKey != "demo-app" {
		t.Fatalf("expected app key demo-app, got %q", cfg.DeployConfig.AppKey)
	}
}

func TestToToolPipelineStatus_MapsStages(t *testing.T) {
	status := &pipeline.Status{
		PipelineID:   "pipeline-123",
		ScenarioName: "demo-scenario",
		Status:       pipeline.StatusRunning,
		CurrentStage: pipeline.StageDeploy,
		StartedAt:    1700000000,
		Stages: map[string]*pipeline.StageResult{
			pipeline.StageSmokeTest: {
				Stage:       pipeline.StageSmokeTest,
				Status:      pipeline.StatusCompleted,
				StartedAt:   1700000001,
				CompletedAt: 1700000002,
			},
			pipeline.StageDeploy: {
				Stage:     pipeline.StageDeploy,
				Status:    pipeline.StatusRunning,
				StartedAt: 1700000003,
			},
		},
	}

	mapped := toToolPipelineStatus(status)
	if mapped == nil {
		t.Fatalf("expected mapped status")
	}
	if mapped.CurrentStage != pipeline.StageDeploy {
		t.Fatalf("expected current stage deploy, got %q", mapped.CurrentStage)
	}
	if len(mapped.Stages) != 2 {
		t.Fatalf("expected 2 stage statuses, got %d", len(mapped.Stages))
	}
}

// fakeProcessExecutor is a test double for smoketest.ProcessExecutor.
type fakeProcessExecutor struct {
	result *smoketest.ExecutionResult
	err    error
}

func (f *fakeProcessExecutor) Execute(_ context.Context, _, _ string, _, _ []string, _ time.Duration) (string, error) {
	return f.result.Combined, f.err
}

func (f *fakeProcessExecutor) ExecuteWithResult(_ context.Context, _, _ string, _, _ []string, _ time.Duration) (*smoketest.ExecutionResult, error) {
	return f.result, f.err
}

func (f *fakeProcessExecutor) LookPath(name string) (string, error) {
	return name, nil
}

func TestScreenrecordingExecutorAdapter(t *testing.T) {
	fake := &fakeProcessExecutor{
		result: &smoketest.ExecutionResult{
			Stdout:   "capture started",
			Stderr:   "warning: something",
			ExitCode: 0,
		},
	}

	adapter := &screenrecordingExecutorAdapter{executor: fake}

	result, err := adapter.ExecuteWithResult(context.Background(), "/tmp", "ffmpeg", nil, nil, 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Stdout != "capture started" {
		t.Errorf("expected stdout 'capture started', got %q", result.Stdout)
	}
	if result.Stderr != "warning: something" {
		t.Errorf("expected stderr 'warning: something', got %q", result.Stderr)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Verify it returns the right type
	var _ *screenrecording.ExecutionResult = result
}

func TestScreenrecordingExecutorAdapter_Error(t *testing.T) {
	fake := &fakeProcessExecutor{
		err: context.DeadlineExceeded,
	}

	adapter := &screenrecordingExecutorAdapter{executor: fake}

	_, err := adapter.ExecuteWithResult(context.Background(), "/tmp", "ffmpeg", nil, nil, 10*time.Second)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}
