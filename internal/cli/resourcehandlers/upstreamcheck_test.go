package resourcehandlers

import (
	"testing"

	"github.com/vrooli/vrooli/internal/cli/resourcecli"
)

func TestSelectUpstreamEntries_AllByDefault(t *testing.T) {
	got := selectUpstreamEntries(resourcecli.UpstreamCheckRequest{})
	if len(got) != len(codingAgentUpstreamEntries) {
		t.Fatalf("empty request should select all, got %d", len(got))
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
