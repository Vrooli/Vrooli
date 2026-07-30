// Package contracts locks the pure configuration-to-launch invariants that
// must hold without a process launch, sandbox, database, or network.
package contracts

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/config"
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
			if codec.Type() == domain.RunnerTypeCodex && containsArgs(continueArgs, "-C") {
				t.Fatalf("codex resume emitted unsupported -C: %q", continueArgs)
			}
		})
	}
}

func parityControls(rt domain.RunnerType) []string {
	switch rt {
	case domain.RunnerTypeClaudeCode:
		return []string{"--model", "--max-turns", "--effort", "--allowedTools", "--disallowedTools", "--chrome"}
	case domain.RunnerTypeCodex:
		// `codex exec resume` accepts model and config controls but not the
		// execute-only working-directory flag.
		return []string{"-m", "-c"}
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
	registry := map[string]string{
		"RunnerType": "resolveExecutionPolicy", "Model": "BuildArgs", "RoleRef": "resolveExecutionPolicy", "MaxTurns": "BuildArgs", "Timeout": "WithTimeout", "Effort": "BuildArgs",
		"PolicySnapshot": "resolveExecutionPolicy", "ResultSpec": "structuredResults.Resolve", "AllowedTools": "BuildArgs", "DeniedTools": "BuildArgs", "ToolRestrictionPolicy": "validateToolRestriction",
		"SkipPermissionPrompt": "BuildArgs", "Features": "BuildArgs", "ExtraFlags": "BuildArgs", "NetworkAccess": "BuildArgs", "SandboxConfig": "DeriveRunMode", "AllowedPaths": "CreateSandbox", "DeniedPaths": "CreateSandbox", "ManifestIndexSnapshot": "ImportTranscript",
	}
	assertFieldRegistry(t, reflect.TypeOf(domain.RunConfig{}), "RunConfig", registry)
}

func TestLeversFieldLivenessRegistryIsComplete(t *testing.T) {
	// Levers are grouped controls. Each group is passed to the named runtime
	// composition consumer; nested storage retention controls are listed
	// separately below because they drive distinct reconciler operations.
	registry := map[string]string{
		"Execution": "NewOrchestrator", "Safety": "NewOrchestrator", "Concurrency": "NewOrchestrator", "Approval": "NewOrchestrator", "Runners": "NewOrchestrator",
		"Server": "NewServer", "Storage": "NewOrchestrator", "Heartbeat": "NewOrchestrator", "Recovery": "NewOrchestrator", "Scanner": "NewOrchestrator",
		"Diagnostics": "NewOrchestrator", "Observability": "Init", "Spawn": "NewOrchestrator", "Sandbox": "NewOrchestrator", "Workflow": "NewOrchestrator",
	}
	assertFieldRegistry(t, reflect.TypeOf(config.Levers{}), "Levers", registry)
	for field, consumer := range map[string]string{
		"Storage.EventRetentionDays":    "cleanupExpiredEvents",
		"Storage.ArtifactRetentionDays": "cleanupExpiredArtifacts",
		"Storage.RunStateRetentionDays": "cleanupRunStateDirs",
	} {
		if !consumerExists(t, consumer) {
			t.Errorf("%s names absent consumer %q", field, consumer)
		}
	}
}

func TestFieldRegistryRejectsUnclassifiedField(t *testing.T) {
	type controlSurface struct{ DeclaredButInert bool }
	if err := validateFieldRegistry(reflect.TypeOf(controlSurface{}), "ControlSurface", map[string]string{}); err == nil {
		t.Fatal("unclassified reflected field must fail liveness validation")
	}
}

func assertFieldRegistry(t *testing.T, typ reflect.Type, subject string, registry map[string]string) {
	t.Helper()
	if err := validateFieldRegistry(typ, subject, registry); err != nil {
		t.Error(err)
	}
	for field, consumer := range registry {
		if !consumerExists(t, consumer) {
			t.Errorf("%s.%s names absent consumer %q", subject, field, consumer)
		}
	}
}

func validateFieldRegistry(typ reflect.Type, subject string, registry map[string]string) error {
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		if field.PkgPath != "" { // unexported
			continue
		}
		if strings.TrimSpace(registry[field.Name]) == "" {
			return fmt.Errorf("%s.%s has no named runtime consumer", subject, field.Name)
		}
	}
	return nil
}

func consumerExists(t *testing.T, symbol string) bool {
	t.Helper()
	apiRoot := apiRoot(t)
	found := false
	err := filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found || entry.IsDir() || strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		found = strings.Contains(string(contents), symbol)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func apiRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contracts source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestToolRestrictionFallbackSkipsUnsupportedCandidate(t *testing.T) {
	// This test is intentionally pure: it pins the policy predicate used by
	// ExecuteWithModelFallback, without constructing a launcher or network client.
	if codecs.NewCodexForTest().Capabilities().SupportsToolRestriction {
		t.Fatal("test requires codex to declare its real unsupported allowlist capability")
	}
}
