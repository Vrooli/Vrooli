package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

func TestCmdInitiativesList_PassesScenarioFlag(t *testing.T) {
	var seenQuery string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/initiatives" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		seenQuery = r.URL.RawQuery
		resp := ListInitiativesResponse{Items: []InitiativeResponse{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdInitiativesList([]string{"--scenario", "web-console"}); err != nil {
		t.Fatalf("cmdInitiativesList: %v", err)
	}
	if seenQuery != "scenario=web-console" {
		t.Errorf("expected query 'scenario=web-console', got %q", seenQuery)
	}
}

func TestCmdInitiativesList_PassesScenarioFlagCSV(t *testing.T) {
	var seenQuery string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		resp := ListInitiativesResponse{Items: []InitiativeResponse{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdInitiativesList([]string{"--scenario", "web-console,command-center"}); err != nil {
		t.Fatalf("cmdInitiativesList: %v", err)
	}
	// url.Values percent-encodes comma as %2C. Accept either raw or encoded
	// depending on how net/url renders the query string.
	want1 := "scenario=web-console%2Ccommand-center"
	want2 := "scenario=web-console,command-center"
	if seenQuery != want1 && seenQuery != want2 {
		t.Errorf("expected scenario CSV query, got %q", seenQuery)
	}
}

func TestCmdInitiativesList_OmitsScenarioWhenAbsent(t *testing.T) {
	var seenQuery string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		resp := ListInitiativesResponse{Items: []InitiativeResponse{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdInitiativesList([]string{}); err != nil {
		t.Fatalf("cmdInitiativesList: %v", err)
	}
	if seenQuery != "" {
		t.Errorf("expected empty query string, got %q", seenQuery)
	}
}

func TestCmdInitiativesList_RendersTargetScenarios(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ListInitiativesResponse{
			Items: []InitiativeResponse{
				{
					Initiative: Initiative{
						Name: "audio-platform", Title: "Audio Platform", Status: "active",
						Items: []string{"execute/web-console-audio"},
					},
					Rollup:          InitiativeRollup{Total: 1, Pending: 1},
					TargetScenarios: []string{"web-console"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdInitiativesList([]string{}); err != nil {
		t.Fatalf("cmdInitiativesList: %v", err)
	}
}

func TestCmdInitiativesList_RendersDependsOn(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ListInitiativesResponse{
			Items: []InitiativeResponse{
				{
					Initiative: Initiative{
						Name: "web-console-readiness", Title: "Web Console Readiness", Status: "active",
						DependsOn: []string{"continuous-audio-platform", "protected-agent-sandboxing"},
					},
					Rollup: InitiativeRollup{Total: 1, Pending: 1},
				},
				{
					Initiative: Initiative{
						Name: "no-deps-init", Title: "No Deps", Status: "active",
					},
					Rollup: InitiativeRollup{Total: 1, Pending: 1},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	out := clitest.CaptureStdout(t, func() error { return app.cmdInitiativesList([]string{}) })

	if !strings.Contains(out, "Depends on: continuous-audio-platform, protected-agent-sandboxing") {
		t.Errorf("expected depends-on line with both deps in order, got:\n%s", out)
	}

	// Find the "no-deps-init" section and assert no "Depends on" line appears before the next initiative.
	noDepsIdx := strings.Index(out, "no-deps-init")
	if noDepsIdx < 0 {
		t.Fatalf("output missing no-deps-init row:\n%s", out)
	}
	tail := out[noDepsIdx:]
	// End of this row is a blank line or end-of-string. Check the section up to the blank line.
	if end := strings.Index(tail, "\n\n"); end >= 0 {
		tail = tail[:end]
	}
	if strings.Contains(tail, "Depends on:") {
		t.Errorf("expected no Depends on line for no-deps-init, got:\n%s", tail)
	}
}

func TestCmdInitiativesGet_RendersDependsOn(t *testing.T) {
	cases := []struct {
		name      string
		deps      []string
		wantLine  string
		wantEmpty bool
	}{
		{name: "empty", deps: nil, wantEmpty: true},
		{name: "single", deps: []string{"foo"}, wantLine: "Depends on: foo"},
		{name: "multi-preserves-order", deps: []string{"foo", "bar", "baz"}, wantLine: "Depends on: foo, bar, baz"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := InitiativeResponse{
					Initiative: Initiative{
						Name: "target", Title: "Target", Status: "active",
						DependsOn: tc.deps,
					},
					Rollup: InitiativeRollup{Total: 0},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))

			app, err := NewApp()
			if err != nil {
				t.Fatalf("NewApp: %v", err)
			}

			out := clitest.CaptureStdout(t, func() error { return app.cmdInitiativesGet([]string{"--name", "target"}) })

			if tc.wantEmpty {
				if strings.Contains(out, "Depends on") {
					t.Errorf("expected no Depends on line, got:\n%s", out)
				}
				return
			}
			if !strings.Contains(out, tc.wantLine) {
				t.Errorf("expected output to contain %q, got:\n%s", tc.wantLine, out)
			}
		})
	}
}
