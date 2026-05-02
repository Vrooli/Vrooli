package graph

import (
	"errors"
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestCommandsRegistersGraphCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Graph" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "graph" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsHealthCommand(t *testing.T) {
	if !strings.Contains(usageText(), "health") {
		t.Fatalf("usage text missing health command: %s", usageText())
	}
}

func TestCmdShowFetchesGraphAndPrintsContractSummary(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/graph", graphIndex{
		GeneratedAt: "2026-05-01T12:00:00Z",
		Graph: graph{
			Nodes: []node{
				{ID: "team-1", Type: "team", Label: "Core Team"},
				{ID: "agent-1", Type: "agent", Label: "Runtime Agent"},
				{ID: "skill-1", Type: "skill", Label: "Testing Skill"},
				{ID: "cli:prompt-manager", Type: "cli", Label: "prompt-manager"},
			},
			Edges: []edge{
				{From: "agent-1", To: "skill-1", Kind: "skill-use"},
				{From: "skill-1", To: "cli:prompt-manager", Kind: "code-usage"},
			},
			HealthScores: []healthScore{
				{NodeID: "agent-1", Score: 0.90},
				{NodeID: "skill-1", Score: 0.70},
			},
		},
	})

	stdout, stderr, err := clitest.Output(t, func() error {
		return cmdShow(ctx, nil)
	})
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	req := ctx.LastRequest()
	if req.Method != "GET" || req.Path != "/graph" {
		t.Fatalf("unexpected request: %+v", req)
	}
	for _, want := range []string{
		"Graph Summary (generated 2026-05-01T12:00:00Z)",
		"Teams:  1",
		"Agents: 1",
		"Skills: 1",
		"CLIs:   1",
		"skill-use: 1",
		"code-usage: 1",
		"Health: 0.80 avg across 2 scored nodes",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}
}

func TestCmdNodeRejectsMissingIDBeforeAPI(t *testing.T) {
	ctx := clitest.NewContext(t)
	err := cmdNode(ctx, nil)
	if err == nil {
		t.Fatal("expected missing node id to fail")
	}
	if !strings.Contains(err.Error(), "usage: graph node") {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx.RequireNoRequests()
}

func TestCmdPopularSurfacesAPIError(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Fail("GET", "/graph/popular", errors.New("graph index unavailable"))

	err := cmdPopular(ctx, []string{"--limit=3"})
	if err == nil {
		t.Fatal("expected API error")
	}
	if !strings.Contains(err.Error(), "failed to fetch popular nodes") || !strings.Contains(err.Error(), "graph index unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}

	req := ctx.LastRequest()
	if req.Method != "GET" || req.Path != "/graph/popular" || req.Query.Get("limit") != "3" {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestCmdHealthTypeFilterFetchesGraphAndFiltersScores(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/graph/health", []healthScore{
		{
			NodeID: "agent-1",
			Score:  0.63,
			Messages: []healthMessage{{
				Severity: "warning",
				Summary:  "missing linked skills",
			}},
		},
		{
			NodeID: "skill-1",
			Score:  0.91,
			Messages: []healthMessage{{
				Severity: "info",
				Summary:  "well connected",
			}},
		},
	})
	ctx.Respond("GET", "/graph", graphIndex{
		Graph: graph{
			Nodes: []node{
				{ID: "agent-1", Type: "agent", Label: "Runtime Agent"},
				{ID: "skill-1", Type: "skill", Label: "Testing Skill"},
			},
		},
	})

	stdout, stderr, err := clitest.Output(t, func() error {
		return cmdHealth(ctx, []string{"--type=skill"})
	})
	if err != nil {
		t.Fatalf("cmdHealth: %v", err)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stdout, "Health Scores (1 nodes):") || !strings.Contains(stdout, "skill-1") || !strings.Contains(stdout, "well connected") {
		t.Fatalf("stdout missing filtered health result: %s", stdout)
	}
	if strings.Contains(stdout, "agent-1") || strings.Contains(stdout, "missing linked skills") {
		t.Fatalf("stdout included filtered-out agent score: %s", stdout)
	}

	requests := ctx.Requests()
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2: %+v", len(requests), requests)
	}
	if requests[0].Method != "GET" || requests[0].Path != "/graph/health" {
		t.Fatalf("unexpected health request: %+v", requests[0])
	}
	if requests[1].Method != "GET" || requests[1].Path != "/graph" {
		t.Fatalf("unexpected graph request: %+v", requests[1])
	}
}
