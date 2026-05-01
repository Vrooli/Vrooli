package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

const fixesContextResponse = `{
  "scenario_name": "web-console",
  "initiatives": [],
  "orphan_items": [],
  "rollup": {"total":0,"completed":0,"in_progress":0,"failed":0,"pending":0,"archived":0},
  "fixes": {
    "active": [
      {"name":"fix-alpha","title":"Alpha bug","status":"backlog","priority":2,"updated":"2026-04-22T10:00:00Z","path":"fix/fix-alpha"},
      {"name":"fix-beta","title":"Beta crash","status":"in_progress","priority":1,"updated":"2026-04-21T10:00:00Z","path":"fix/fix-beta"}
    ],
    "archived": [
      {"name":"fix-gamma","title":"Gamma regression","status":"completed","priority":1,"archived_at":"2026-04-15T10:00:00Z","path":"fix/fix-gamma"}
    ]
  }
}`

func newFixesTestApp(t *testing.T) *App {
	t.Helper()
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scenarios/web-console/context" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(fixesContextResponse))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app
}

func TestCmdScenariosFixes_RequiresName(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdScenariosFixes([]string{}); err == nil {
		t.Fatal("expected usage error when --name missing")
	}
}

func TestCmdScenariosFixes_MutuallyExclusiveScopeFlags(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdScenariosFixes([]string{"--name", "x", "--active", "--archived"}); err == nil {
		t.Fatal("expected error when both --active and --archived are set")
	}
}

func TestCmdScenariosFixes_AllScopeJSONIncludesBothPartitions(t *testing.T) {
	app := newFixesTestApp(t)

	stdout := clitest.CaptureStdout(t, func() error {
		return app.cmdScenariosFixes([]string{"--name", "web-console", "--all", "--json"})
	})

	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout)
	}
	if out["scope"] != "all" {
		t.Errorf("scope = %v, want all", out["scope"])
	}
	fixes := out["fixes"].(map[string]any)
	if len(fixes["active"].([]any)) != 2 {
		t.Errorf("active count = %d, want 2", len(fixes["active"].([]any)))
	}
	if len(fixes["archived"].([]any)) != 1 {
		t.Errorf("archived count = %d, want 1", len(fixes["archived"].([]any)))
	}
}

func TestCmdScenariosFixes_ActiveScopeExcludesArchived(t *testing.T) {
	app := newFixesTestApp(t)

	stdout := clitest.CaptureStdout(t, func() error {
		return app.cmdScenariosFixes([]string{"--name", "web-console", "--active", "--json"})
	})

	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	fixes := out["fixes"].(map[string]any)
	if len(fixes["archived"].([]any)) != 0 {
		t.Errorf("archived must be empty for --active scope, got %v", fixes["archived"])
	}
	if len(fixes["active"].([]any)) != 2 {
		t.Errorf("active count = %d, want 2", len(fixes["active"].([]any)))
	}
}

func TestCmdScenariosFixes_SearchFiltersByTitle(t *testing.T) {
	app := newFixesTestApp(t)

	stdout := clitest.CaptureStdout(t, func() error {
		return app.cmdScenariosFixes([]string{"--name", "web-console", "--all", "--search", "regression", "--json"})
	})

	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	fixes := out["fixes"].(map[string]any)
	if len(fixes["active"].([]any)) != 0 {
		t.Errorf("active must be empty after search filter, got %v", fixes["active"])
	}
	archived := fixes["archived"].([]any)
	if len(archived) != 1 {
		t.Fatalf("archived count = %d, want 1 after search", len(archived))
	}
	if got := archived[0].(map[string]any)["name"]; got != "fix-gamma" {
		t.Errorf("archived[0].name = %v, want fix-gamma", got)
	}
}

func TestCmdScenariosFixes_HumanOutputRendersHeadings(t *testing.T) {
	app := newFixesTestApp(t)

	stdout := clitest.CaptureStdout(t, func() error {
		return app.cmdScenariosFixes([]string{"--name", "web-console"})
	})
	for _, want := range []string{"Active Fixes", "Archived Fixes", "fix-alpha", "fix-gamma"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected output to contain %q\n--- output ---\n%s", want, stdout)
		}
	}
}

func TestFilterFixes_CaseInsensitive(t *testing.T) {
	in := []ScenarioFix{
		{Name: "fix-a", Title: "Frobnicator broke"},
		{Name: "fix-b", Title: "Unrelated"},
	}
	got := filterFixes(in, "FROB")
	if len(got) != 1 || got[0].Name != "fix-a" {
		t.Errorf("got %+v, want [fix-a]", got)
	}
}
