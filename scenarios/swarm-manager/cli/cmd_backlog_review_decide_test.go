package main

import (
	"io"
	"net/http"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/proto"
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
	base := []string{"--kind", "execute", "--name", "foo", "--round", "1", "--decided-by", "operator:test"}
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
			var got apipb.DecideAttemptRequest
			var gotPath string
			clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method: %s", r.Method)
				}
				gotPath = r.URL.Path
				body, _ := io.ReadAll(r.Body)
				if err := proto.Unmarshal(body, &got); err != nil {
					t.Fatalf("decode: %v", err)
				}
				response, err := proto.Marshal(&apipb.DecideAttemptResponse{Decision: tc.decision, Status: "completed", DecidedAt: "2026-04-23T00:00:00Z"})
				if err != nil {
					t.Fatalf("encode response: %v", err)
				}
				w.Header().Set("Content-Type", "application/proto")
				_, _ = w.Write(response)
			}))

			app, err := NewApp()
			if err != nil {
				t.Fatalf("NewApp: %v", err)
			}
			args := []string{"--kind", "execute", "--name", "foo", "--round", "2", "--decided-by", "operator:test", tc.flag, "--rationale", "ship it", "--json"}
			if err := app.cmdBacklogReviewDecide(args); err != nil {
				t.Fatalf("run: %v", err)
			}
			if gotPath != "/vrooli.swarm_manager.v1.api.BacklogService/DecideAttempt" {
				t.Errorf("path: %s", gotPath)
			}
			if got.GetDecision() != tc.decision {
				t.Errorf("decision: %v", got.GetDecision())
			}
			if got.GetRationale() != "ship it" {
				t.Errorf("rationale: %v", got.GetRationale())
			}
			if got.GetSubjectKind() != "backlog-item" || got.GetSubjectRef() != "execute/foo" || got.GetRoundNum() != 2 {
				t.Errorf("attempt target = %s %s round %d", got.GetSubjectKind(), got.GetSubjectRef(), got.GetRoundNum())
			}
		})
	}
}
