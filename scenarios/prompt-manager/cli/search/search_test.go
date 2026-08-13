package search

import (
	"errors"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestCommandsRegistersSearchCommands(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Search" {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if len(group.Commands) < 4 {
		t.Fatalf("expected search command family, got %d command(s)", len(group.Commands))
	}
	for _, command := range group.Commands {
		if !command.NeedsAPI {
			t.Fatalf("search command should require API: %+v", command)
		}
	}
}

func TestCmdSearchAISuccessPostsRequestAndPrintsResults(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/search/ai", AISearchResponse{
		Total:  1,
		Query:  "unit test",
		Method: "vector",
		Results: []AISearchResult{{
			ID:           "skill-test",
			Name:         "Testing Skill",
			Folder:       "core",
			ScorePercent: 91,
			Tags:         []string{"testing"},
			Description:  "Validates behavior at the CLI/API boundary.",
		}},
	})

	stdout, stderr, err := clitest.Output(t, func() error {
		return cmdSearch(ctx, []string{"unit test", "--limit=3"})
	})
	if err != nil {
		t.Fatalf("cmdSearch: %v", err)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	for _, want := range []string{"Search Results (1 found, AI search):", "Testing Skill", "(91%)", "[skill-test]"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/search/ai" {
		t.Fatalf("unexpected request: %+v", req)
	}
	payload, ok := req.Payload.(AISearchRequest)
	if !ok {
		t.Fatalf("payload type = %T, want AISearchRequest", req.Payload)
	}
	if payload.Query != "unit test" || payload.Limit != 3 || payload.Output != "results" || payload.Format != "xml" {
		t.Fatalf("unexpected AI search payload: %+v", payload)
	}
}

func TestCmdSearchCombinedOutputRequiresQueryBeforeAPI(t *testing.T) {
	ctx := clitest.NewContext(t)
	err := cmdSearch(ctx, []string{"--tag=testing", "--output=combined"})
	if err == nil {
		t.Fatal("expected combined output without query to fail")
	}
	if !strings.Contains(err.Error(), "combined output requires a query") {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx.RequireNoRequests()
}

func TestCmdSearchFallsBackToTextSearchOnAIError(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Fail("POST", "/search/ai", errors.New("vector service down"))
	ctx.Respond("GET", "/search/skills", SearchResponse{
		Total: 1,
		Query: "fallback",
		Results: []SearchResult{{
			ID:        "skill-fallback",
			Name:      "Fallback Skill",
			Folder:    "local",
			Highlight: "fallback text match",
		}},
	})

	stdout, stderr, err := clitest.Output(t, func() error {
		return cmdSearch(ctx, []string{"fallback"})
	})
	if err != nil {
		t.Fatalf("cmdSearch fallback: %v", err)
	}
	if !strings.Contains(stderr, "AI search unavailable") {
		t.Fatalf("stderr missing fallback notice: %s", stderr)
	}
	for _, want := range []string{"Search Results (1 found, text search):", "Fallback Skill", "fallback text match"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}

	requests := ctx.Requests()
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2: %+v", len(requests), requests)
	}
	if requests[1].Method != "GET" || requests[1].Path != "/search/skills" || requests[1].Query.Get("q") != "fallback" {
		t.Fatalf("unexpected fallback request: %+v", requests[1])
	}
}
