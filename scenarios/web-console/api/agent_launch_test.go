package main

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	"web-console/handlers/modelselection"
	sessionsH "web-console/handlers/sessions"
)

func harnessFor(t *testing.T, id string) harness.Descriptor {
	t.Helper()
	descriptor, ok := harness.Active().ByID(id)
	if !ok {
		t.Fatalf("no harness %q", id)
	}
	return descriptor
}

func TestComposableLaunchCommandSeparatesPlainStartsFromTheOperatorsOwn(t *testing.T) {
	claude, codex := harnessFor(t, "claude"), harnessFor(t, "codex")
	for _, command := range []string{"claude", "claude --dangerously-skip-permissions", "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions"} {
		if !composableLaunchCommand(claude, command) {
			t.Errorf("%q should take settings", command)
		}
	}
	if !composableLaunchCommand(codex, "codex --yolo") {
		t.Error("codex --yolo should take settings")
	}
	for _, command := range []string{"", "claude | tee log", "my-wrapper claude", "claude && rm x", "claude --model opus", "CLAUDE_CODE_AUTO_COMPACT_WINDOW=300000 claude", "vrooli agent launch --runner codex", "vrooli agent launch", "claude -mopus", "claude --model=opus", "vrooli agent launch --runner claude --arg=-mopus", "claude\rrm -rf x"} {
		if composableLaunchCommand(claude, command) {
			t.Errorf("%q is the operator's own and must stay literal", command)
		}
	}
	if composableLaunchCommand(codex, "codex -c model_auto_compact_token_limit=1") {
		t.Error("a command stating its own limit must stay literal")
	}
	for _, command := range []string{"codex -c model=gpt-6-luna", "codex --config model=gpt-6-luna", "codex --config=model=gpt-6-luna", "vrooli agent launch --runner codex --arg=-c --arg=model=gpt-6-luna"} {
		if composableLaunchCommand(codex, command) {
			t.Errorf("%q names its own model and must stay literal", command)
		}
	}
	if !composableLaunchCommand(codex, "codex -c sandbox_mode=workspace-write") {
		t.Error("a config override that is not the model still takes settings")
	}
}

func TestComposeAgentLaunchBuildsCommandsFromTheDescriptor(t *testing.T) {
	useRepoPolicy(t, "claude-code", nil)
	s := &Server{}
	ctx := t.Context()
	for _, tc := range []struct {
		name, command, want string
		intent              sessionsH.AgentLaunch
	}{
		{"no settings runs as written", "claude | tee log", "claude | tee log", sessionsH.AgentLaunch{AgentID: "claude"}},
		{"model on a bare command", "claude", "claude --model opus", sessionsH.AgentLaunch{AgentID: "claude", Model: "opus"}},
		{"model through the launcher", "vrooli agent launch --runner claude", "vrooli agent launch --runner claude --arg=--model --arg=fable", sessionsH.AgentLaunch{AgentID: "claude", Model: "fable"}},
	} {
		got, err := s.composeAgentLaunch(ctx, tc.intent, tc.command, "local")
		if err != nil || got != tc.want {
			t.Errorf("%s: got %q, %v; want %q", tc.name, got, err, tc.want)
		}
	}
}

func TestContextLimitFollowsTheDescriptorsMechanism(t *testing.T) {
	claude, codex := harnessFor(t, "claude"), harnessFor(t, "codex")
	for _, tc := range []struct {
		descriptor harness.Descriptor
		command    string
		want       string
	}{
		{claude, "claude --dangerously-skip-permissions", "CLAUDE_CODE_AUTO_COMPACT_WINDOW=150000 claude --dangerously-skip-permissions"},
		{codex, "codex --yolo --no-alt-screen", "codex -c model_auto_compact_token_limit=150000 --yolo --no-alt-screen"},
		{codex, "vrooli agent launch --runner codex", "vrooli agent launch --runner codex --arg=-c --arg=model_auto_compact_token_limit=150000"},
	} {
		if got := applyContextLimit(tc.descriptor, tc.command, 150000); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}

func TestComposeAgentLaunchRefusesWhatTheCatalogDoesNotOffer(t *testing.T) {
	useRepoPolicy(t, "claude-code", nil)
	s := &Server{}
	ctx := t.Context()
	for name, tc := range map[string]struct {
		intent  sessionsH.AgentLaunch
		command string
	}{
		"unknown agent":            {sessionsH.AgentLaunch{AgentID: "nobody", Model: "x"}, "claude"},
		"unoffered model":          {sessionsH.AgentLaunch{AgentID: "claude", Model: "gpt-6-astra"}, "claude"},
		"settings on a wrapper":    {sessionsH.AgentLaunch{AgentID: "claude", Model: "opus"}, "my-wrapper claude"},
		"limit nobody offers":      {sessionsH.AgentLaunch{AgentID: "claude", ContextLimitTokens: 42}, "claude"},
		"limit on a plain harness": {sessionsH.AgentLaunch{AgentID: "grok", ContextLimitTokens: 150000}, "grok"},
	} {
		if _, err := s.composeAgentLaunch(ctx, tc.intent, tc.command, "local"); !errors.Is(err, sessionsH.ErrInvalidArgument) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestCheckContextLimitHoldsAnExplicitModelToItsOwnOptions(t *testing.T) {
	claude := harnessFor(t, "claude")
	big := modelselection.Choice{Model: "big", ContextLimitTokens: []int64{300000, 150000}}
	small := modelselection.Choice{Model: "small", ContextLimitTokens: []int64{150000}}
	policy := modelselection.Result{Choices: []modelselection.Choice{big, small}}
	if err := checkContextLimit(claude, policy, &small, 300000); !errors.Is(err, sessionsH.ErrInvalidArgument) {
		t.Fatalf("a model must not take a limit it does not offer: %v", err)
	}
	if err := checkContextLimit(claude, policy, nil, 300000); err != nil {
		t.Fatalf("with no explicit model, any offered limit stands: %v", err)
	}
}

func TestComposeAgentLaunchRefusesAMeteredChoiceFromAStaleList(t *testing.T) {
	useRepoPolicy(t, "opencode", func(doc map[string]any) {
		doc["provenance"].(map[string]any)["observed_at"] = "2020-01-01"
	})
	s := &Server{}
	policy := s.GetRoleModelChoices(t.Context(), "opencode", "local", "opencode")
	if !policy.CatalogStale {
		t.Fatalf("the list should be stale: %+v", policy.PolicyStatus)
	}
	var metered, unmetered modelselection.Choice
	for _, choice := range policy.Choices {
		if choice.Metered && metered.Model == "" {
			metered = choice
		}
		if !choice.Metered && unmetered.Model == "" {
			unmetered = choice
		}
	}
	if _, err := s.composeAgentLaunch(t.Context(), sessionsH.AgentLaunch{AgentID: "opencode", Model: metered.Model, Provider: metered.Source}, "opencode", "local"); !errors.Is(err, sessionsH.ErrFailedPrecondition) {
		t.Fatalf("metered and stale must be refused: %v", err)
	}
	got, err := s.composeAgentLaunch(t.Context(), sessionsH.AgentLaunch{AgentID: "opencode", Model: unmetered.Model, Provider: unmetered.Source}, "opencode", "local")
	if err != nil || got != "opencode -m "+unmetered.LaunchModel {
		t.Fatalf("a stale list still offers unmetered choices: %q %v", got, err)
	}
}

func TestCreateComposesBeforeAnythingStarts(t *testing.T) {
	refused := errors.New("refused")
	adapter := &sessionsH.Adapter{ComposeAgentLaunch: func(_ context.Context, intent sessionsH.AgentLaunch, command, targetID string) (string, error) {
		if intent.AgentID != "claude" || command != "claude" || targetID != "local" {
			t.Errorf("intent %+v, command %q, target %q", intent, command, targetID)
		}
		return "", refused
	}}
	if _, err := adapter.Create(t.Context(), sessionsH.CreateInput{LaunchCommand: "claude", TargetID: "local", AgentLaunch: &sessionsH.AgentLaunch{AgentID: "claude", Model: "opus"}}); !errors.Is(err, refused) {
		t.Fatalf("a refused launch must not start: %v", err)
	}
}

func TestResolveLaunchChoiceNeverPicksAMeteredSourceSilently(t *testing.T) {
	metered := modelselection.Choice{Model: "opus", Source: "openrouter", Metered: true}
	subscription := modelselection.Choice{Model: "opus", Source: "subscription"}
	for name, choices := range map[string][]modelselection.Choice{"metered listed first": {metered, subscription}, "metered listed last": {subscription, metered}} {
		if got := resolveLaunchChoice(choices, "opus", ""); got == nil || got.Source != "subscription" {
			t.Errorf("%s: got %+v", name, got)
		}
	}
	if got := resolveLaunchChoice([]modelselection.Choice{metered, subscription}, "opus", "openrouter"); got == nil || got.Source != "openrouter" {
		t.Errorf("a named provider stands: %+v", got)
	}
	if got := resolveLaunchChoice([]modelselection.Choice{metered}, "opus", ""); got == nil || got.Source != "openrouter" {
		t.Errorf("the only offer stands: %+v", got)
	}
	if got := resolveLaunchChoice([]modelselection.Choice{subscription}, "opus", "openrouter"); got != nil {
		t.Errorf("a provider that does not offer it finds nothing: %+v", got)
	}
}
