package orchestration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/vrooli/packages/proto/contracts"
)

type wc3ReplayBrief struct {
	Destination   string `json:"destination"`
	Stance        string `json:"stance"`
	CharterPath   string `json:"charterPath"`
	StanceMessage string `json:"stanceMessage"`
	Kernel        string `json:"kernel"`
	Message       string `json:"message"`
}

// This is a local cold replay from a generated WC3 draft. It reads the actual
// Agent Manager capability declarations and objective-delivery code; it does
// not launch a harness or claim its native transcript was compacted.
func TestWC3ColdReplayAcrossHarnessObjectiveContracts(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
	fixturePath := filepath.Join(filepath.Dir(filename), "testdata", "wc3-cold-replay-brief.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var brief wc3ReplayBrief
	if err := json.Unmarshal(raw, &brief); err != nil {
		t.Fatal(err)
	}
	if brief.Kernel == "" || len([]rune(brief.Kernel)) > 600 || brief.Stance == "" || brief.CharterPath == "" {
		t.Fatalf("generated draft does not meet the WC3 kernel contract: %+v", brief)
	}
	if len([]rune(brief.StanceMessage)) > 3500 || !strings.Contains(brief.StanceMessage, brief.Stance) || !strings.Contains(brief.Kernel, brief.CharterPath) || !strings.Contains(brief.Kernel, "Re-read it at every unit start") {
		t.Fatal("generated draft omitted its full stance, charter, or re-read instruction")
	}
	charterPath := filepath.Join(repoRoot, filepath.FromSlash(brief.CharterPath))
	charter, err := os.ReadFile(charterPath)
	if err != nil {
		t.Fatalf("cold receiver could not read charter %q: %v", charterPath, err)
	}
	if !strings.Contains(string(charter), brief.Stance) || !strings.Contains(string(charter), brief.Destination) {
		t.Fatal("cold replay charter does not contain the exact stance and destination")
	}

	cases := []struct {
		name         string
		runner       domain.RunnerType
		capabilities runner.Capabilities
		nativeGoal   bool
	}{
		{name: "claude-code", runner: domain.RunnerTypeClaudeCode, capabilities: codecs.NewClaudeForTest().Capabilities(), nativeGoal: true},
		{name: "codex", runner: domain.RunnerTypeCodex, capabilities: codecs.NewCodexForTest().Capabilities(), nativeGoal: true},
		{name: "grok", runner: domain.RunnerTypeGrok, capabilities: codecs.NewGrokForTest().Capabilities()},
		{name: "opencode", runner: domain.RunnerTypeOpenCode, capabilities: codecs.NewOpenCodeForTest().Capabilities()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := nativeObjectiveRun(tc.runner, brief.Kernel)
			run.CharterPath = charterPath
			objective := nativeObjectiveFor(run, tc.capabilities)
			lineageHeader := domain.RunLineage{RunID: uuid.New(), TaskID: uuid.New(), CharterPath: run.CharterPath}.Header()
			instructions := domain.WithLineageHeader(lineageHeader, brief.Message)
			prompt := initialPromptWithUntil(run, instructions)
			if !strings.Contains(prompt, brief.Stance) || !strings.Contains(prompt, brief.Kernel) {
				t.Fatal("initial task prompt lost the generated full stance or kernel")
			}
			if !strings.Contains(prompt, "re-read it at every unit start and after any context compaction") || !strings.Contains(prompt, run.CharterPath) {
				t.Fatal("persisted charter path did not reach the cold-replay instructions")
			}

			if tc.nativeGoal {
				limit := runner.NativeGoalVerbatimLimit(tc.runner)
				if len([]rune(brief.Kernel)) > limit || objective != brief.Kernel {
					t.Fatalf("native goal = %q at limit %d; want the complete short kernel", objective, limit)
				}
				// Simulate a cold post-compaction view: transient user messages are
				// gone; only the harness-retained native condition and durable charter
				// are available. The charter restores the full stance verbatim.
				coldView := objective + "\n" + string(charter)
				if !strings.Contains(coldView, brief.Stance) || !strings.Contains(coldView, brief.Destination) {
					t.Fatal("cold native-goal replay did not recover the charter's full stance and destination")
				}
				t.Logf("local replay: native goal retained under %d-character contract; persisted charter re-read restored full stance", limit)
				return
			}

			if objective != "" {
				t.Fatalf("runner without a native objective installed %q", objective)
			}
			if !strings.Contains(prompt, "Completion contract (engine-owned):") {
				t.Fatal("prompt-carried fallback omitted the completion contract")
			}
			// Grok and OpenCode do not declare native-objective support. Their
			// prompt is proven at launch, but no measured compaction window exists
			// here, so the replay records that post-compaction survival as unknown.
			t.Log("local replay: prompt-carried launch contract present; post-compaction survival window unknown")
		})
	}
}

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
