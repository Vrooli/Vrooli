package topics

import (
	"errors"
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestCommandsRegistersTopicCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Topics" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "topic" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsTree(t *testing.T) {
	if !strings.Contains(usageText(), "tree") {
		t.Fatalf("usage text missing tree command: %s", usageText())
	}
}

func TestCmdListPrintsTopicsAndRecordsAPIContract(t *testing.T) {
	ctx := clitest.NewContext(t)
	parent := "core"
	ctx.Respond("GET", "/topics", []Topic{
		{ID: "core", Name: "Core", Status: "active"},
		{ID: "testing", Name: "Testing", ParentTopicID: &parent, Skills: []string{"unit-testing", "test-architecture"}, Status: "active"},
	})

	stdout, stderr, err := clitest.Output(t, func() error {
		return cmdList(ctx, nil)
	})
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	req := ctx.LastRequest()
	if req.Method != "GET" || req.Path != "/topics" {
		t.Fatalf("unexpected request: %+v", req)
	}
	for _, want := range []string{
		"Topics (2):",
		"core - Core",
		"testing - Testing (parent: core) [2 skills]",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}
}

func TestCmdCreateRejectsMissingNameBeforeAPI(t *testing.T) {
	ctx := clitest.NewContext(t)
	err := cmdCreate(ctx, []string{"--id=missing-name"})
	if err == nil {
		t.Fatal("expected missing name to fail")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx.RequireNoRequests()
}

func TestCmdSearchPostsQueriesAndPrintsAccumulatedSkills(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/topics/match", MatchResponse{
		Method: "vector",
		Topics: []MatchedTopic{{
			ID:           "testing",
			Name:         "Testing",
			Description:  "Quality and unit test design",
			ScorePercent: 88,
		}},
		Skills: []string{"unit-testing", "test-architecture"},
	})

	stdout, stderr, err := clitest.Output(t, func() error {
		return cmdSearch(ctx, []string{"unit", "coverage", "--limit=4"})
	})
	if err != nil {
		t.Fatalf("cmdSearch: %v", err)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/topics/match" {
		t.Fatalf("unexpected request: %+v", req)
	}
	payload, ok := req.Payload.(MatchRequest)
	if !ok {
		t.Fatalf("payload type = %T, want MatchRequest", req.Payload)
	}
	if strings.Join(payload.Queries, ",") != "unit,coverage" || payload.Limit != 4 {
		t.Fatalf("unexpected search payload: %+v", payload)
	}
	for _, want := range []string{
		"Matched Topics (1, vector search):",
		"testing - Testing (88%)",
		"Quality and unit test design",
		"Accumulated Skills (2):",
		"unit-testing",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}
}

func TestCmdDeleteSurfacesAPIError(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Fail("DELETE", "/topics/bad%2Ftopic", errors.New("permission denied"))

	err := cmdDelete(ctx, []string{"bad/topic"})
	if err == nil {
		t.Fatal("expected API error")
	}
	if !strings.Contains(err.Error(), "deleting topic") || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("unexpected error: %v", err)
	}

	req := ctx.LastRequest()
	if req.Method != "DELETE" || req.Path != "/topics/bad%2Ftopic" {
		t.Fatalf("unexpected request: %+v", req)
	}
}
