package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/agentcatalog/harness"
)

func TestDeriveAgentIDRecognizesTheCommandsWebConsoleEmits(t *testing.T) {
	for _, tc := range []struct {
		command string
		want    string
	}{
		{"vrooli agent launch --runner claude --arg=--dangerously-skip-permissions", "claude"},
		{"vrooli agent launch --runner=codex --arg=--yolo", "codex"},
		{"vrooli-agent-launcher --agent grok --", "grok"},
		{"vrooli-agent-launcher --agent=opencode", "opencode"},
		{"codex --yolo", "codex"},
		{"codex --yolo resume 01a029f4", "codex"},
		{"claude --resume abc --dangerously-skip-permissions", "claude"},
		{"opencode", "opencode"},
		{"grok", "grok"},
		{"agy", "agy"},
		{"exec /usr/local/bin/codex --yolo", "codex"},
		{`C:\Program Files\codex.exe --yolo`, "codex"},
	} {
		if got := deriveAgentID(tc.command); got != tc.want {
			t.Errorf("deriveAgentID(%q) = %q, want %q", tc.command, got, tc.want)
		}
	}
}

// A command that launches no catalogued agent must derive nothing. Returning a
// wrong id here would attach an operator's deploy script to an agent card and
// make it disappear from the custom-command list.
func TestDeriveAgentIDReturnsEmptyForPlainOperatorCommands(t *testing.T) {
	for _, command := range []string{
		"",
		"ls -la",
		"make deploy ENV=staging",
		"vrooli scenario start audio-tools",
		"git status",
	} {
		if got := deriveAgentID(command); got != "" {
			t.Errorf("deriveAgentID(%q) = %q, want empty", command, got)
		}
	}
}

// `--runner` is the governed form and must outrank a bare token elsewhere in
// the same command. Without this precedence a wrapper that mentions two agents
// resolves to whichever appeared first by accident.
func TestDeriveAgentIDPrefersTheExplicitRunnerFlag(t *testing.T) {
	command := "codex-wrapper --note opencode -- vrooli agent launch --runner grok"
	if got := deriveAgentID(command); got != "grok" {
		t.Fatalf("deriveAgentID = %q, want grok", got)
	}
}

// An id the catalogue does not know must not be stored verbatim; otherwise a
// typo mints an agent no probe will ever report and the entry silently stops
// matching any card.
func TestNormalizeAgentIDRejectsUnknownIdentifiers(t *testing.T) {
	for _, id := range []string{"", "  ", "cursor", "claude-code-v2"} {
		if got := normalizeAgentID(id); got != "" {
			t.Errorf("normalizeAgentID(%q) = %q, want empty", id, got)
		}
	}
	for input, want := range map[string]string{
		"CODEX":       "codex",
		" claude ":    "claude",
		"antigravity": "agy",
		"claude-code": "claude",
	} {
		if got := normalizeAgentID(input); got != want {
			t.Errorf("normalizeAgentID(%q) = %q, want %q", input, got, want)
		}
	}
}

// Identity and display resolution cover every resource-defined harness.
func TestAgentIdentityCoversEveryCataloguedHarness(t *testing.T) {
	for _, descriptor := range harness.Active().All() {
		if normalizeAgentID(descriptor.ID) != descriptor.ID {
			t.Errorf("descriptor %q does not normalize to itself", descriptor.ID)
		}
		if agentDisplayName(descriptor.ID) != descriptor.DisplayName {
			t.Errorf("agentDisplayName(%q) = %q, want %q", descriptor.ID, agentDisplayName(descriptor.ID), descriptor.DisplayName)
		}
	}
}

// Identity resolution reads the active resource descriptors directly. This
// mirrors a desktop restart after adding a harness resource and proves that
// aliases do not depend on a stale startup projection in capabilityprobe.
func TestAgentIdentityUsesTheProductionResourceCatalogue(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "nimbus")
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"nimbus","harness":{"id":"nimbus","aliases":["nimbus-code"],"binary":"nimbus","display_name":"Nimbus Agent","icon":"terminal","home_isolation":{"kind":"none"},"features":{"model_selection":{"flag":"--model","sources":["nimbus-plan"]}}}}`
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCE_ROOT", root)
	t.Setenv("VROOLI_ROOT", "")

	if got := normalizeAgentID("nimbus"); got != "nimbus" {
		t.Fatalf("normalizeAgentID(nimbus) = %q", got)
	}
	if got := normalizeAgentID("nimbus-code"); got != "nimbus" {
		t.Fatalf("normalizeAgentID(nimbus-code) = %q", got)
	}
	if got := deriveAgentID("exec nimbus --model nimbus-1"); got != "nimbus" {
		t.Fatalf("deriveAgentID(nimbus) = %q", got)
	}
	if got := deriveAgentID("vrooli agent launch --runner nimbus-code"); got != "nimbus" {
		t.Fatalf("deriveAgentID(nimbus-code) = %q", got)
	}
	if got := agentDisplayName("nimbus-code"); got != "Nimbus Agent" {
		t.Fatalf("agentDisplayName(nimbus) = %q", got)
	}
}

func TestNormalizeShortcutEntriesFillsMissingAgentIDs(t *testing.T) {
	entries := normalizeShortcutEntries([]ShortcutEntry{
		{Label: "Codex", Command: "codex --yolo"},
		{Label: "Deploy", Command: "make deploy"},
		{Label: "Wrapper", Command: "my-wrapper.sh", AgentID: "claude"},
		{Label: "Bogus", Command: "codex --yolo", AgentID: "not-an-agent"},
	})
	want := []string{"codex", "", "claude", "codex"}
	for i, expected := range want {
		if entries[i].AgentID != expected {
			t.Errorf("entry %d (%s) agent = %q, want %q", i, entries[i].Label, entries[i].AgentID, expected)
		}
	}
}

// A stored id the operator set explicitly outranks derivation: a wrapper
// script named nothing like its agent is exactly why the field exists.
func TestNormalizeShortcutEntriesKeepsExplicitAgentOverDerivation(t *testing.T) {
	entries := normalizeShortcutEntries([]ShortcutEntry{
		{Label: "Codex via wrapper", Command: "run-claude-sandbox.sh", AgentID: "codex"},
	})
	if entries[0].AgentID != "codex" {
		t.Fatalf("agent = %q, want codex (explicit id must win over the command text)", entries[0].AgentID)
	}
}

// Normalization must not write through to the shared defaults slice. It did
// not, but the built-in list is process-global and a regression here would
// corrupt every profile created afterwards.
func TestNormalizeShortcutEntriesDoesNotMutateItsInput(t *testing.T) {
	input := []ShortcutEntry{{Label: "Codex", Command: "codex --yolo"}}
	_ = normalizeShortcutEntries(input)
	if input[0].AgentID != "" {
		t.Fatalf("input was mutated: agent = %q", input[0].AgentID)
	}
}

// Every built-in default names its agent explicitly, so the shipped list never
// depends on derivation being right.
func TestDefaultShortcutsDeclareTheirAgents(t *testing.T) {
	for _, entry := range defaultShortcuts {
		if entry.AgentID == "" {
			t.Errorf("default shortcut %q has no agent id", entry.Label)
			continue
		}
		if normalizeAgentID(entry.AgentID) != entry.AgentID {
			t.Errorf("default shortcut %q names unknown agent %q", entry.Label, entry.AgentID)
		}
		if derived := deriveAgentID(entry.Command); derived != entry.AgentID {
			t.Errorf("default shortcut %q declares %q but its command derives %q", entry.Label, entry.AgentID, derived)
		}
	}
}

// A profile written before agent_id existed must come back self-describing,
// so the launcher never has to fall back to matching command text.
func TestEffectiveBackfillsAgentIDsForProfilesStoredWithoutThem(t *testing.T) {
	store := NewShortcutProfileStore()
	store.profiles["legacy"] = &ShortcutProfile{
		ID:    "legacy",
		Scope: "workspace",
		Name:  "Legacy",
		Shortcuts: []ShortcutEntry{
			{Label: "Codex", Command: "codex --yolo"},
			{Label: "Deploy", Command: "make deploy"},
		},
	}
	eff := store.Effective(context.Background())
	if eff.ProfileID != "legacy" {
		t.Fatalf("profile id = %q, want legacy", eff.ProfileID)
	}
	if eff.Shortcuts[0].AgentID != "codex" {
		t.Errorf("legacy codex entry agent = %q, want codex", eff.Shortcuts[0].AgentID)
	}
	if eff.Shortcuts[1].AgentID != "" {
		t.Errorf("legacy custom entry agent = %q, want empty", eff.Shortcuts[1].AgentID)
	}
}
