package domains

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestSubcommandGroupsRegistersExpectedDomains(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	groups, err := SubcommandGroups(nil, manifest)
	if err != nil {
		t.Fatalf("SubcommandGroups: %v", err)
	}
	if len(groups) != 8 {
		t.Fatalf("SubcommandGroups() returned %d groups, want 8", len(groups))
	}

	got := make(map[string]int, len(groups))
	for _, group := range groups {
		got[group.Name] = len(group.Subcommands)
		if !group.NeedsAPI {
			t.Fatalf("group %q should require API access", group.Name)
		}
	}

	want := map[string]int{
		"repo":     13,
		"branch":   4,
		"worktree": 8,
		"review":   3,
		"audit":    1,
		"baseline": 15, // 6 record/lifecycle verbs + collection/path evidence + 7 engagement verbs
		"message":  9,  // collection/check, staged/worktree drafting, retry, and paged immutable evidence
	}
	for name, count := range want {
		if got[name] != count {
			t.Fatalf("group %q command count = %d, want %d; all groups: %#v", name, got[name], count, got)
		}
	}
}

func TestCommandGroupsRegisterCanonicalUXCapture(t *testing.T) {
	groups := CommandGroups(nil)
	if len(groups) != 1 || groups[0].Title != "UX capture" || len(groups[0].Commands) != 1 || groups[0].Commands[0].Name != "ux-capture" {
		t.Fatalf("CommandGroups() = %#v, want one ux-capture command", groups)
	}
}

func TestLocalManifestCommandsMatchRegisteredFlatCommands(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	registered := map[string]bool{}
	for _, group := range CommandGroups(nil) {
		for _, command := range group.Commands {
			registered[command.Name] = true
		}
	}
	subcommands, err := SubcommandGroups(nil, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range subcommands {
		for _, command := range group.Subcommands {
			registered[group.Name+" "+command.Name] = true
		}
	}
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			if command.Binding.Kind != "local" {
				continue
			}
			path := group.Name
			if !group.Flat {
				path += " " + command.Name
			}
			if !registered[path] {
				t.Errorf("manifest local command %q has no registered dispatch command", path)
			}
		}
	}
}
