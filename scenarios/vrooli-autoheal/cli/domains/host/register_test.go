package host

import (
	"testing"

	"vrooli-autoheal/cli/internal/support"
)

func TestRegisterProvidesHostCommands(t *testing.T) {
	group := Register(nil)
	want := []string{"inventory", "collect", "changes"}
	if group.Name != "host" {
		t.Fatalf("Register().Name = %q, want host", group.Name)
	}
	if len(group.Subcommands) != len(want) {
		t.Fatalf("Register() subcommand count = %d, want %d", len(group.Subcommands), len(want))
	}
	for i, name := range want {
		if group.Subcommands[i].Name != name {
			t.Fatalf("Register() subcommand[%d] = %q, want %q", i, group.Subcommands[i].Name, name)
		}
	}
}

func TestChangeLinesReportsEmptyAndFormatsChanges(t *testing.T) {
	empty := changeLines(nil)
	if len(empty) != 1 || empty[0] != "No host inventory changes recorded." {
		t.Fatalf("changeLines(nil) = %#v, want empty-state message", empty)
	}

	lines := changeLines([]support.HostInventoryChange{
		{
			CreatedAt:  "2026-05-08T12:00:00Z",
			Severity:   "critical",
			ChangeType: "kernel-module-drift",
			Summary:    "NVIDIA module missing",
		},
	})
	if len(lines) != 1 {
		t.Fatalf("changeLines() returned %d lines, want 1", len(lines))
	}
	want := "2026-05-08T12:00:00Z critical kernel-module-drift: NVIDIA module missing"
	if lines[0] != want {
		t.Fatalf("changeLines()[0] = %q, want %q", lines[0], want)
	}
}

func TestProbeStatusLinesFormatsProbeMap(t *testing.T) {
	lines := probeStatusLines(map[string]string{"kernel": "ok"})
	if len(lines) != 1 || lines[0] != "kernel: ok" {
		t.Fatalf("probeStatusLines() = %#v, want kernel status line", lines)
	}
}
