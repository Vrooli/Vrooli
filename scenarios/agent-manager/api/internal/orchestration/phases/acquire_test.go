package phases

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestAcquireRunnerUsesResolvedRunSelectionAndReportsActionableFallback(t *testing.T) {
	ctx := context.Background()
	registry := runner.NewRegistry()
	codex := runner.NewMockRunner(domain.RunnerTypeCodex)
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(runner.NewStubRunner(domain.RunnerTypeClaudeCode, "claude missing")); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}}
	got, err := AcquireRunner(ctx, AcquireRunnerInput{Run: run, Runners: registry})
	if err != nil || got != codex {
		t.Fatalf("acquired runner=%T err=%v", got, err)
	}

	run.ResolvedConfig.RunnerType = domain.RunnerTypeClaudeCode
	_, err = AcquireRunner(ctx, AcquireRunnerInput{Run: run, Runners: registry})
	runnerErr, ok := err.(*domain.RunnerError)
	if !ok || runnerErr.Operation != "availability_check" || !runnerErr.IsTransient || runnerErr.Alternative != string(domain.RunnerTypeCodex) {
		t.Fatalf("unavailable runner error = %#v", err)
	}
}

func TestAcquireRunnerRejectsMissingRegistryAndMissingConfiguredRunner(t *testing.T) {
	ctx := context.Background()
	run := &domain.Run{ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode}}
	if _, err := AcquireRunner(ctx, AcquireRunnerInput{Run: run}); err == nil {
		t.Fatal("missing registry accepted")
	}
	registry := runner.NewRegistry()
	if _, err := AcquireRunner(ctx, AcquireRunnerInput{Run: run, Runners: registry}); err == nil {
		t.Fatal("missing configured runner accepted")
	}
	if got := GetRunnerType(nil, nil); got != domain.RunnerTypeClaudeCode {
		t.Fatalf("snapshot-less runner type=%s, want default", got)
	}
	if got := lastResortAlternative(nil, domain.RunnerTypeCodex); got != "" {
		t.Fatalf("nil registry alternative=%q", got)
	}
}
