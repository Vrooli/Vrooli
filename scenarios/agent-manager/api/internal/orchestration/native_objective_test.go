package orchestration

import (
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"github.com/vrooli/vrooli/packages/proto/contracts"
)

func nativeObjectiveRun(runnerType domain.RunnerType, until string) *domain.Run {
	return &domain.Run{
		RunMode: domain.RunModeSandboxed,
		ResolvedConfig: &domain.RunConfig{
			RunnerType:    runnerType,
			Until:         until,
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking},
		},
	}
}

func trackingNativeCaps() runner.Capabilities {
	return runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{
		ExecutionMode: "interactive", SandboxModes: []string{"tracking"}, NativeObjective: true,
	}}}
}

func TestNativeObjectiveForUsesDeclaredInteractiveCapability(t *testing.T) {
	run := nativeObjectiveRun(domain.RunnerTypeCodex, "all phases complete")
	caps := trackingNativeCaps()
	if got := nativeObjectiveFor(run, caps); got != "all phases complete" {
		t.Fatalf("nativeObjectiveFor() = %q", got)
	}
	caps.SpawnCapabilities[0].NativeObjective = false
	if got := nativeObjectiveFor(run, caps); got != "" {
		t.Fatalf("nativeObjectiveFor() without declaration = %q", got)
	}
	caps.SpawnCapabilities[0].SandboxModes = []string{"protected"}
	if got := nativeObjectiveFor(run, caps); got != "" {
		t.Fatalf("nativeObjectiveFor() in protected mode = %q", got)
	}
}

// A Codex objective longer than compaction keeps is delegated, so the native
// goal survives compaction and the task prompt carries the whole contract.
func TestNativeObjectiveForDelegatesWhatCompactionWouldOmit(t *testing.T) {
	until := strings.Repeat("engine-owned contract ", 40) // 880 characters
	codex := nativeObjectiveRun(domain.RunnerTypeCodex, until)
	if got := nativeObjectiveFor(codex, trackingNativeCaps()); got != runner.NativeGoalDelegation {
		t.Fatalf("codex nativeObjectiveFor() = %q, want the delegation", got)
	}
	if prompt := initialPromptWithUntil(codex, "do the work"); !strings.HasSuffix(prompt, strings.TrimSpace(until)) {
		t.Fatal("the task prompt must carry the full contract the native goal delegates to")
	}
	claude := nativeObjectiveRun(domain.RunnerTypeClaudeCode, until)
	if got := nativeObjectiveFor(claude, trackingNativeCaps()); got != strings.TrimSpace(until) {
		t.Fatalf("claude keeps an 880-character condition; nativeObjectiveFor() = %.40q", got)
	}
}

func TestMaximumCompletionContractStaysInTaskPromptAndDelegatesNativeObjective(t *testing.T) {
	until := strings.Repeat("x", contracts.AgentManagerMaxGoalCompletionBytes)
	for _, runnerType := range []domain.RunnerType{domain.RunnerTypeCodex, domain.RunnerTypeClaudeCode} {
		run := nativeObjectiveRun(runnerType, until)
		if got := nativeObjectiveFor(run, trackingNativeCaps()); got != runner.NativeGoalDelegation {
			t.Fatalf("%s nativeObjectiveFor() = %.40q, want the short delegation", runnerType, got)
		}
		if prompt := initialPromptWithUntil(run, "do the work"); !strings.HasSuffix(prompt, until) {
			t.Fatalf("%s task prompt truncated the maximum-size completion contract", runnerType)
		}
	}
}

func TestInitialNativeGoalOriginUsesOnlyAdmittedExplicitIntent(t *testing.T) {
	for _, kind := range []string{"human", "run", "agent-baseline", "operation-grant"} {
		run := nativeObjectiveRun(domain.RunnerTypeCodex, "finish exact task")
		run.OwnerSubject = "operator"
		run.ResolvedConfig.Admission = &domain.RunAdmission{RequestedGoalMode: "until", EffectiveUntil: run.ResolvedConfig.Until, CreateCaller: &domain.CreateRunCaller{Kind: kind, Subject: "operator"}}
		if got := initialNativeGoalOrigin(run); got != webconsole.NativeGoalUserDirected {
			t.Fatalf("%s: origin=%v", kind, got)
		}
	}
	for _, tc := range []struct {
		name   string
		change func(*domain.Run)
	}{
		{"historical", func(r *domain.Run) { r.ResolvedConfig.Admission = nil }},
		{"inherited default", func(r *domain.Run) { r.ResolvedConfig.Admission.RequestedGoalMode = "" }},
		{"changed objective", func(r *domain.Run) { r.ResolvedConfig.Until = "new objective" }},
		{"missing caller", func(r *domain.Run) { r.ResolvedConfig.Admission.CreateCaller = nil }},
		{"standing service", func(r *domain.Run) { r.ResolvedConfig.Admission.CreateCaller.Kind = "service-standing" }},
		{"unknown caller", func(r *domain.Run) { r.ResolvedConfig.Admission.CreateCaller.Kind = "guessed" }},
		{"owner mismatch", func(r *domain.Run) { r.ResolvedConfig.Admission.CreateCaller.Subject = "other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := nativeObjectiveRun(domain.RunnerTypeCodex, "finish exact task")
			run.OwnerSubject = "operator"
			run.ResolvedConfig.Admission = &domain.RunAdmission{RequestedGoalMode: "until", EffectiveUntil: run.ResolvedConfig.Until, CreateCaller: &domain.CreateRunCaller{Kind: "human", Subject: "operator"}}
			tc.change(run)
			if got := initialNativeGoalOrigin(run); got != webconsole.NativeGoalAutomatic {
				t.Fatalf("unknown intent upgraded to user: %v", got)
			}
		})
	}
}

func TestInitialNativeGoalOriginSurvivesNativeObjectiveDelegation(t *testing.T) {
	run := nativeObjectiveRun(domain.RunnerTypeCodex, strings.Repeat("complete the exact admitted contract; ", 100))
	run.OwnerSubject = "operator"
	run.ResolvedConfig.Admission = buildRunAdmission(CreateRunRequest{Until: run.ResolvedConfig.Until, caller: &domain.CreateRunCaller{Kind: "human", Subject: "operator"}}, run.ResolvedConfig)
	if nativeObjectiveFor(run, trackingNativeCaps()) != runner.NativeGoalDelegation {
		t.Fatal("fixture did not require native objective delegation")
	}
	if initialNativeGoalOrigin(run) != webconsole.NativeGoalUserDirected {
		t.Fatal("provider objective transformation lost the frozen admitted source")
	}
}
