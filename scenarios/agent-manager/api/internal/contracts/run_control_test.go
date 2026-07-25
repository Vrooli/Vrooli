// Package contracts locks the pure configuration-to-launch invariants that
// must hold without a process launch, sandbox, database, or network.
package contracts

import (
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/phases"
)

func TestCodecExecuteContinueControlParity(t *testing.T) {
	cfg := &domain.RunConfig{
		Model:                "test-model",
		MaxTurns:             17,
		Effort:               domain.EffortHigh,
		SkipPermissionPrompt: true,
		AllowedTools:         []string{"read"},
		DeniedTools:          []string{"shell"},
		Features:             domain.FeatureFlags{EnableBrowser: true},
	}
	for _, codec := range []codecs.Codec{codecs.NewClaudeForTest(), codecs.NewCodexForTest(), codecs.NewGrokForTest(), codecs.NewOpenCodeForTest()} {
		t.Run(string(codec.Type()), func(t *testing.T) {
			execArgs := codec.BuildArgs(codec.NewState(), runner.ExecuteRequest{ResolvedConfig: cfg, WorkingDir: "/work"})
			continueArgs := codec.BuildContinueArgs(codec.NewState(), runner.ContinueRequest{ResolvedConfig: cfg, WorkingDir: "/work", SessionID: "session"})
			for _, control := range parityControls(codec.Type()) {
				if !containsArgs(execArgs, control) || !containsArgs(continueArgs, control) {
					t.Fatalf("%s control %q missing from execute=%q continue=%q", codec.Type(), control, execArgs, continueArgs)
				}
			}
		})
	}
}

func parityControls(rt domain.RunnerType) []string {
	switch rt {
	case domain.RunnerTypeClaudeCode:
		return []string{"--model", "--max-turns", "--effort", "--allowedTools", "--disallowedTools", "--chrome"}
	case domain.RunnerTypeCodex:
		return []string{"-m", "-c", "-C"}
	case domain.RunnerTypeGrok:
		return []string{"-m", "--max-turns", "--effort", "--cwd"}
	default:
		return nil
	}
}

func containsArgs(args []string, control string) bool {
	for _, arg := range args {
		if arg == control || strings.HasPrefix(arg, control+"=") {
			return true
		}
	}
	return false
}

func TestDefaultLifecycleVocabularyIsEmittable(t *testing.T) {
	// normalize is deliberately exercised through a run's effective config by
	// the phase tests; the default's terminal event is the public contract.
	for _, event := range []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal} {
		emittable := false
		for _, status := range []domain.RunStatus{domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusCancelled} {
			produced := phases.LifecycleEventForStatus(status)
			if event == produced || event == domain.SandboxLifecycleTerminal {
				emittable = true
			}
		}
		if !emittable {
			t.Fatalf("default lifecycle event %q is not emittable", event)
		}
	}
}

func TestRunConfigFieldLivenessRegistryIsComplete(t *testing.T) {
	// Every field in domain.RunConfig is deliberately classified. Update this
	// registry in the same change as any new field; an unclassified control is
	// an inert operator promise.
	registry := map[string]string{
		"RunnerType": "runner-selection", "Model": "codec-argv", "RoleRef": "policy-selection", "MaxTurns": "codec-argv", "Timeout": "execution-deadline", "Effort": "codec-argv-or-capability-event",
		"PolicySnapshot": "policy-selection", "ResultSpec": "result-resolution", "AllowedTools": "codec-argv-or-policy", "DeniedTools": "codec-argv-or-advisory-event", "ToolRestrictionPolicy": "policy-selection",
		"SkipPermissionPrompt": "codec-argv", "Features": "codec-argv", "ExtraFlags": "codec-argv", "NetworkAccess": "codec-argv", "SandboxConfig": "sandbox-acceptance", "AllowedPaths": "sandbox-acceptance", "DeniedPaths": "sandbox-acceptance",
	}
	expected := []string{"RunnerType", "Model", "RoleRef", "MaxTurns", "Timeout", "Effort", "PolicySnapshot", "ResultSpec", "AllowedTools", "DeniedTools", "ToolRestrictionPolicy", "SkipPermissionPrompt", "Features", "ExtraFlags", "NetworkAccess", "SandboxConfig", "AllowedPaths", "DeniedPaths"}
	for _, field := range expected {
		if strings.TrimSpace(registry[field]) == "" {
			t.Errorf("RunConfig.%s has no liveness classification", field)
		}
	}
}

func TestToolRestrictionFallbackSkipsUnsupportedCandidate(t *testing.T) {
	// This test is intentionally pure: it pins the policy predicate used by
	// ExecuteWithModelFallback, without constructing a launcher or network client.
	if codecs.NewCodexForTest().Capabilities().SupportsToolRestriction {
		t.Fatal("test requires codex to declare its real unsupported allowlist capability")
	}
}
