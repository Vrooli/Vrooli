package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	capsH "web-console/handlers/capabilities"
	"web-console/internal/capabilities"
)

// Every table Web Console keeps about harnesses must read from the descriptors,
// or fail here.
func TestProbeDefinitionsComeFromDescriptors(t *testing.T) {
	definitions := aiToolDefinitions(harness.Builtin())
	if !reflect.DeepEqual(definitions, capabilityprobe.AITools) {
		t.Fatalf("got %+v, want %+v", definitions, capabilityprobe.AITools)
	}
	for _, definition := range definitions {
		if resource, ok := agentResource(definition.ID); !ok || resource == "" {
			t.Errorf("%s has no resource", definition.ID)
		}
	}
}

func capabilityDefs() map[string]capabilities.Def {
	byID := map[string]capabilities.Def{}
	for _, def := range capabilities.KnownForPlatform("linux") {
		byID[def.ID] = def
	}
	return byID
}

func TestCapabilityOverlaysComeFromDescriptors(t *testing.T) {
	byID := capabilityDefs()
	for id, want := range map[string]struct{ name, command string }{
		"claude": {"Claude Code", "vrooli resource install claude-code --json"},
		"codex":  {"Codex", "vrooli resource install codex --json"},
	} {
		def, ok := byID[id]
		if !ok || def.Name != want.name || def.OperatorCommand != want.command || def.ActionLabel != "Install "+want.name {
			t.Errorf("%s: %+v (found %t)", id, def, ok)
		}
	}
	if _, ok := byID["claude-code"]; ok {
		t.Error("the public capability ID must stay claude")
	}
}

func TestListHarnessesReturnsDescriptorsAndRefusesAnUnknownTarget(t *testing.T) {
	s := &Server{}
	got, err := s.listHarnesses(t.Context(), "local")
	if err != nil {
		t.Fatal(err)
	}
	byAgent := map[string]capsH.HarnessDescriptor{}
	for _, descriptor := range got {
		byAgent[descriptor.AgentID] = descriptor
	}
	claude, codex, opencode, grok := byAgent["claude"], byAgent["codex"], byAgent["opencode"], byAgent["grok"]
	if claude.Resource != "claude-code" || !claude.SupportsContextLimit || !claude.SupportsUsageMode || !claude.SupportsModelSelection || claude.SupportsProviderSelection {
		t.Errorf("claude: %+v", claude)
	}
	if !reflect.DeepEqual(claude.UsageWindows, []capsH.HarnessUsageWindow{{ID: "five_hour", WindowMinutes: 300}, {ID: "seven_day", WindowMinutes: 10080}}) {
		t.Errorf("claude windows: %+v", claude.UsageWindows)
	}
	if !reflect.DeepEqual(codex.UsageWindows, []capsH.HarnessUsageWindow{{ID: "weekly", WindowMinutes: 10080}}) {
		t.Errorf("codex windows: %+v", codex.UsageWindows)
	}
	if !opencode.SupportsProviderSelection || opencode.SupportsContextLimit || opencode.SupportsUsageMode {
		t.Errorf("opencode: %+v", opencode)
	}
	if grok.SupportsModelSelection || grok.SupportsContextLimit || len(grok.UsageWindows) != 0 {
		t.Errorf("grok: %+v", grok)
	}
	if !slices.IsSortedFunc(got, func(a, b capsH.HarnessDescriptor) int { return cmpStrings(a.AgentID, b.AgentID) }) {
		t.Errorf("not sorted: %+v", got)
	}
	if _, err := s.listHarnesses(t.Context(), " "); err == nil {
		t.Error("a blank target must be refused")
	}
	if _, err := s.listHarnesses(t.Context(), "bridge-node:absent"); !errors.Is(err, capsH.ErrTargetNotFound) {
		t.Errorf("an unknown target: %v", err)
	}
}

func cmpStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func TestSixthHarnessReachesTheLauncherFromItsResourceDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "resources", "sixth")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"sixth","harness":{"id":"six","binary":"six","display_name":"Sixth","icon":"six","home_isolation":{"kind":"none"},
		"usage_windows":[{"id":"daily","minutes":1440}],
		"features":{"model_selection":{"flag":"--model","sources":["sixth-plan"]},"context_limit":{"kind":"env","key":"SIX_LIMIT","minimum":300000},"provider_selection":false,"resume":false,"goal_mode":false,"mcp":false,"hooks":false}}}`
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	policy := `{"schema_version":"v1","runner":"sixth","provenance":{"source":"test","observed_at":"2099-01-01"},
		"billing":{"mode":"subscription","sources":[{"id":"sixth-plan","label":"Sixth Plan","billing_mode":"subscription"}]},
		"roles":{"code.default":{"model":"six-1","models":[{"model":"six-1","source":"sixth-plan","effort":null}],"description":"d","capabilities":["code"]},
		"code.fast":{"model":"six-1","description":"d","capabilities":["code"]},"code.smart":{"model":"six-1","description":"d","capabilities":["code"]},"code.cheap":{"model":"six-1","description":"d","capabilities":["code"]}}}`
	if err := os.WriteFile(filepath.Join(dir, "model-policy.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCE_ROOT", root)

	listed, err := (&Server{}).listHarnesses(t.Context(), "local")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(listed, func(d capsH.HarnessDescriptor) bool { return d.AgentID == "six" && d.SupportsContextLimit }) {
		t.Fatalf("not listed: %+v", listed)
	}
	if !slices.ContainsFunc(aiToolDefinitions(harness.Active()), func(d capabilityprobe.Definition) bool { return d.ID == "six" && d.Command == "six" }) {
		t.Fatal("not probed")
	}
	choices := (&Server{}).GetRoleModelChoices(t.Context(), "six", "local", "")
	if choices.PolicyStatus != "ready" || choices.ModelFlag != "--model" || len(choices.Choices) != 1 {
		t.Fatalf("no model choices: %+v", choices)
	}
	if got := choices.Choices[0]; got.ContextWindowTokens != 0 || !slices.Equal(got.ContextLimitTokens, []int64{1000000, 300000}) {
		t.Fatalf("limits: %+v", got)
	}
	if _, ok := capabilityDefs()["claude"]; !ok {
		t.Fatal("an overlay for an undeclared dependency must not break the catalogue")
	}
	if usageModeProvider("six") != "" || usageModeModelFlag("six") != "" {
		t.Fatal("a harness that declares no usage mode must not support one")
	}
}
