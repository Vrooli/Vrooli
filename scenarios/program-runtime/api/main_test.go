package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"program-runtime/handlers/bindings"
	registrybindings "program-runtime/internal/bindings"
	"program-runtime/internal/programs"
)

func TestDeclaredPrivateBindingNamesMatchKernelNamespace(t *testing.T) {
	names := declaredBindingNames([]programs.BindingSpec{{Scenario: "client-rehearsal", Group: "bounty-cases", Command: "resolve"}})
	require.Equal(t, []string{"client_rehearsal.bounty_cases.resolve"}, names)
}

func TestInferenceCallerCohortDeclaresDemandStartedAIService(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	repoRoot := filepath.Clean(filepath.Join(workingDirectory, "../../.."))
	registry, err := registrybindings.Load(repoRoot)
	require.NoError(t, err)
	const dependency = "ai-gateway/inference/run-batch"
	binding, ok := registry.Binding(dependency)
	require.True(t, ok, "the inference target must resolve through the governed registry")

	for _, caller := range []string{"typed-inference", "batch-inference"} {
		t.Run(caller, func(t *testing.T) {
			path := filepath.Join(repoRoot, "scenarios/program-runtime/.vrooli/program-runtime", caller+".json")
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			var contract struct {
				Bindings []struct {
					ID string `json:"id"`
				} `json:"bindings"`
			}
			require.NoError(t, json.Unmarshal(raw, &contract))
			require.Contains(t, contract.Bindings, struct {
				ID string `json:"id"`
			}{ID: dependency}, "caller contract must declare its actual AI Gateway dependency")
			spec := runtimeBindingSpec(binding, registry)
			require.Equal(t, "ai-gateway", spec.Scenario)
			require.True(t, spec.DemandStart, "the Go control plane owns demand start and lease handling")
		})
	}
}

func TestDeclaredBrowserBindingsUseOnlyTheBrowserAlias(t *testing.T) {
	names := declaredBindingNames([]programs.BindingSpec{
		{ID: "browser-automation-studio/session/open", Scenario: "browser-automation-studio", Namespace: "browser", Group: "session", Command: "open"},
		{ID: "browser-automation-studio/session/observe", Scenario: "browser-automation-studio", Namespace: "browser", Command: "observe"},
		{ID: "browser-automation-studio/session/act", Scenario: "browser-automation-studio", Namespace: "browser", Command: "act"},
		{ID: "browser-automation-studio/task/run-task", Scenario: "browser-automation-studio", Namespace: "browser", Command: "run_task"},
		{ID: "browser-automation-studio/session/promote-trace", Scenario: "browser-automation-studio", Namespace: "browser", Command: "promote_trace"},
		{ID: "browser-automation-studio/session/draft-trace", Scenario: "browser-automation-studio", Namespace: "browser", Command: "draft_trace"},
		{ID: "browser-automation-studio/task/replay-flow", Scenario: "browser-automation-studio", Namespace: "browser", Command: "replay_flow"},
	})
	require.Equal(t, []string{"browser.session.open", "browser.observe", "browser.act", "browser.run_task", "browser.promote_trace", "browser.draft_trace", "browser.replay_flow"}, names)
	other := declaredBindingNames([]programs.BindingSpec{{ID: "browser-automation-studio/workflows/get", Scenario: "browser-automation-studio", Group: "workflows", Command: "get"}})
	require.Equal(t, []string{"browser_automation_studio.workflows.get"}, other)
}

func TestBASManifestPublishesOnlyGovernedSessionAliases(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	repoRoot := filepath.Clean(filepath.Join(workingDirectory, "../../.."))
	registry, err := registrybindings.LoadFiles(
		filepath.Join(repoRoot, "packages/proto/gen/descriptor/image.binpb"),
		[]string{filepath.Join(repoRoot, "scenarios/browser-automation-studio/cli/manifest.json")},
	)
	require.NoError(t, err)

	want := map[string]string{
		"browser-automation-studio/session/open":          "browser.session.open",
		"browser-automation-studio/session/observe":       "browser.observe",
		"browser-automation-studio/session/act":           "browser.act",
		"browser-automation-studio/session/attach":        "browser.session.attach",
		"browser-automation-studio/session/read-trace":    "browser.session.read_trace",
		"browser-automation-studio/session/promote-trace": "browser.promote_trace",
		"browser-automation-studio/session/draft-trace":   "browser.draft_trace",
		"browser-automation-studio/session/close":         "browser.session.close",
		"browser-automation-studio/task/run-task":         "browser.run_task",
		"browser-automation-studio/task/replay-flow":      "browser.replay_flow",
	}
	for id, expectedName := range want {
		binding, ok := registry.Binding(id)
		require.Truef(t, ok, "BAS manifest binding %s must resolve to its descriptor", id)
		require.Truef(t, bindings.IsBrowserSessionBindingID(id), "only declared BAS session IDs receive the invocation-local grant relay: %s", id)
		spec := runtimeBindingSpec(binding, registry)
		require.Equal(t, expectedName, declaredBindingNames([]programs.BindingSpec{spec})[0], "binding %s", id)
		require.NotEmpty(t, spec.ContractDigest, "binding %s must have a descriptor-backed contract digest", id)
	}
	for _, binding := range registry.List("browser-automation-studio", "") {
		if bindings.IsBrowserSessionBindingID(binding.GetId()) {
			continue
		}
		spec := runtimeBindingSpec(binding, registry)
		require.NotEqual(t, "browser", spec.Namespace, "non-session BAS binding %s must not receive browser authority", binding.GetId())
	}
}
