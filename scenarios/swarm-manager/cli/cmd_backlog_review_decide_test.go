package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

func TestCmdBacklogReviewDecide_RequiresKindAndName(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	cases := [][]string{
		{"--accept"},
		{"--kind", "execute", "--accept"},
		{"--name", "foo", "--accept"},
	}
	for _, args := range cases {
		if err := app.cmdBacklogReviewDecide(args); err == nil {
			t.Errorf("expected usage error for args=%v", args)
		}
	}
}

func TestCmdBacklogReviewDecide_RejectsZeroOrMultipleDecisionFlags(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	base := []string{"--kind", "execute", "--name", "foo"}
	cases := [][]string{
		base,
		append(append([]string{}, base...), "--accept", "--fail"),
		append(append([]string{}, base...), "--accept", "--followup"),
		append(append([]string{}, base...), "--fail", "--followup"),
		append(append([]string{}, base...), "--accept", "--fail", "--followup"),
	}
	for _, args := range cases {
		err := app.cmdBacklogReviewDecide(args)
		if err == nil {
			t.Errorf("expected error for args=%v", args)
			continue
		}
		if !strings.Contains(err.Error(), "exactly one of --accept, --fail, --followup") {
			t.Errorf("args=%v: expected mutual-exclusion error, got %v", args, err)
		}
	}
}

func TestCmdBacklogReviewDecide_PostsDecisionPayload(t *testing.T) {
	cases := []struct {
		flag     string
		decision string
	}{
		{"--accept", "accept"},
		{"--fail", "fail"},
		{"--followup", "followup"},
	}
	for _, tc := range cases {
		t.Run(tc.decision, func(t *testing.T) {
			var got map[string]any
			var gotPath string
			clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method: %s", r.Method)
				}
				gotPath = r.URL.Path
				body, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("decode: %v", err)
				}
				_, _ = w.Write([]byte(`{"decision":"` + tc.decision + `","status":"completed","decided_at":"2026-04-23T00:00:00Z"}`))
			}))

			app, err := NewApp()
			if err != nil {
				t.Fatalf("NewApp: %v", err)
			}
			args := []string{"--kind", "execute", "--name", "foo", tc.flag, "--rationale", "ship it", "--json"}
			if err := app.cmdBacklogReviewDecide(args); err != nil {
				t.Fatalf("run: %v", err)
			}
			if gotPath != "/api/v1/backlog/execute/foo/review-decide" {
				t.Errorf("path: %s", gotPath)
			}
			if got["decision"] != tc.decision {
				t.Errorf("decision: %v", got["decision"])
			}
			if got["rationale"] != "ship it" {
				t.Errorf("rationale: %v", got["rationale"])
			}
		})
	}
}
