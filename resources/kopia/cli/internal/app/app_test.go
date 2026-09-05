package app

import (
	"testing"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/maintenance"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repo"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/snapshot"
)

func TestNewBuildsResourceApp(t *testing.T) {
	a, err := New("fp", "ts", "")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	if a == nil || a.CLI == nil {
		t.Fatal("expected non-nil app + CLI")
	}
	if a.StaleChecker == nil {
		t.Fatal("expected non-nil stale checker")
	}
}

func TestSubcommandSurface(t *testing.T) {
	a, err := New("fp", "ts", "")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	groups := subcommandGroups(a, repo.Service{}, snapshot.Service{}, policy.Service{}, maintenance.Service{})

	want := map[string][]string{
		"repo":        {"create", "connect", "status", "stats", "list", "disconnect", "delete", "validate"},
		"snapshot":    {"create", "list", "browse", "restore", "verify", "delete"},
		"policy":      {"set", "show", "list"},
		"maintenance": {"run", "set"},
	}
	got := map[string][]string{}
	for _, g := range groups {
		var subs []string
		for _, c := range g.Subcommands {
			if c.Run == nil {
				t.Errorf("subcommand %s %s has no Run handler", g.Name, c.Name)
			}
			subs = append(subs, c.Name)
		}
		got[g.Name] = subs
	}

	for group, subs := range want {
		gotSubs, ok := got[group]
		if !ok {
			t.Errorf("missing command group %q", group)
			continue
		}
		set := map[string]bool{}
		for _, s := range gotSubs {
			set[s] = true
		}
		for _, s := range subs {
			if !set[s] {
				t.Errorf("group %q missing subcommand %q (have %v)", group, s, gotSubs)
			}
		}
	}
}
