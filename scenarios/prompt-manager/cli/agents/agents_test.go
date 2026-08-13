package agents

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestCommandsRegistersAgentCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Agents" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	command := group.Commands[0]
	if command.Name != "agent" || !command.NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", command)
	}
}

func TestUsageTextDocumentsSearch(t *testing.T) {
	if !strings.Contains(usageText(), "search, find") {
		t.Fatalf("usage text missing search command: %s", usageText())
	}
}

func TestListPrintsAgentsFromAPI(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/agents", []Agent{
		{ID: "agent-1", DisplayName: "Planner", Status: "active"},
		{ID: "agent-2", DisplayName: "Reviewer", Status: "inactive"},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{"list"})
	})
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}

	if !strings.Contains(stdout, "Planner [agent-1] (active)") || !strings.Contains(stdout, "Reviewer [agent-2] (inactive)") {
		t.Fatalf("list output missing agents:\n%s", stdout)
	}
	req := ctx.LastRequest()
	if req.Method != "GET" || req.Path != "/agents" {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestCreateRequiresNameBeforeAPI(t *testing.T) {
	ctx := clitest.NewContext(t)

	err := route(ctx, []string{"create"})
	if err == nil || !strings.Contains(err.Error(), "usage: agent create <name>") {
		t.Fatalf("expected create usage error, got %v", err)
	}
	ctx.RequireNoRequests()
}

func TestCreateSendsPayloadAndPrintsCreatedAgent(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/agents", Agent{ID: "agent-7", DisplayName: "Builder", Status: "active"})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{
			"create", "Builder",
			"--description", "Builds plans",
			"--tags", "planning, review",
			"--body-color", "#111111",
		})
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if !strings.Contains(stdout, "Created agent: Builder [agent-7]") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/agents" {
		t.Fatalf("unexpected request: %+v", req)
	}
	payload, ok := req.Payload.(CreateAgentRequest)
	if !ok {
		t.Fatalf("unexpected payload type %T", req.Payload)
	}
	if payload.DisplayName != "Builder" || payload.Description != "Builds plans" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Appearance == nil || payload.Appearance.Body != "#111111" || payload.Appearance.Head == "" {
		t.Fatalf("unexpected appearance payload: %+v", payload.Appearance)
	}
	if got := strings.Join(payload.Tags, ","); got != "planning,review" {
		t.Fatalf("unexpected tags %q", got)
	}
}

func TestSearchAIFallbackUsesTextSearch(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Fail("POST", "/search/agents/ai", errors.New("ollama unavailable"))
	ctx.Respond("GET", "/search/agents", AgentSearchResponse{
		Total: 1,
		Query: "planner",
		Results: []AgentSearchResult{{
			ID:          "agent-1",
			DisplayName: "Planner",
			Status:      "active",
			Score:       0.9,
			Highlight:   "planner match",
		}},
	})

	stdout, stderr, err := clitest.Output(t, func() error {
		return route(ctx, []string{"search", "planner"})
	})
	if err != nil {
		t.Fatalf("search agents: %v", err)
	}
	if !strings.Contains(stderr, "AI search unavailable") {
		t.Fatalf("expected fallback warning, got %q", stderr)
	}
	if !strings.Contains(stdout, "Agent Search Results (1 found, text search)") || !strings.Contains(stdout, "Planner") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}

	requests := ctx.Requests()
	if len(requests) != 2 {
		t.Fatalf("expected AI then text requests, got %+v", requests)
	}
	if requests[0].Method != "POST" || requests[0].Path != "/search/agents/ai" {
		t.Fatalf("unexpected AI request: %+v", requests[0])
	}
	if requests[1].Method != "GET" || requests[1].Path != "/search/agents" {
		t.Fatalf("unexpected text request: %+v", requests[1])
	}
	want := url.Values{"q": []string{"planner"}}
	if requests[1].Query.Encode() != want.Encode() {
		t.Fatalf("unexpected search query: %v", requests[1].Query)
	}
}
