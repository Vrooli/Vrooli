package resourcehandlers

import (
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/codingagents"
)

func TestSelectUpstreamEntries_AllByDefault(t *testing.T) {
	got := selectUpstreamEntries(resourcecli.UpstreamCheckRequest{})
	if len(got) != len(codingAgentUpstreamEntries) {
		t.Fatalf("empty request should select all, got %d", len(got))
	}
}

func TestUpstreamEntriesMatchCodingAgentCatalog(t *testing.T) {
	var gotNames []string
	var gotCommands []string
	for _, entry := range codingAgentUpstreamEntries {
		gotNames = append(gotNames, entry.Name)
		if len(entry.CheckCmd) == 0 {
			t.Fatalf("entry %q has empty check command", entry.Name)
		}
		gotCommands = append(gotCommands, entry.CheckCmd[0])
	}
	if want := codingagents.Names(); !reflect.DeepEqual(gotNames, want) {
		t.Fatalf("upstream entry names = %v, want catalog names %v", gotNames, want)
	}
	if want := codingagents.ResourceCLIs(); !reflect.DeepEqual(gotCommands, want) {
		t.Fatalf("upstream entry commands = %v, want catalog CLIs %v", gotCommands, want)
	}
}

func TestSelectUpstreamEntries_AllFlag(t *testing.T) {
	got := selectUpstreamEntries(resourcecli.UpstreamCheckRequest{Name: "codex", All: true})
	if len(got) != len(codingAgentUpstreamEntries) {
		t.Fatalf("--all should override name, got %d", len(got))
	}
}

func TestSelectUpstreamEntries_ByName(t *testing.T) {
	got := selectUpstreamEntries(resourcecli.UpstreamCheckRequest{Name: "opencode"})
	if len(got) != 1 || got[0].Name != "opencode" {
		t.Fatalf("name filter = %+v, want [opencode]", got)
	}
}

func TestSelectUpstreamEntries_UnknownName(t *testing.T) {
	if got := selectUpstreamEntries(resourcecli.UpstreamCheckRequest{Name: " collaborator"}); got != nil {
		t.Fatalf("unknown name should select nothing, got %+v", got)
	}
}
