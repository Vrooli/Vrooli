package graph

import (
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestUsageTextMentionsTopicsAndDrainStatus(t *testing.T) {
	usage := usageText()
	for _, want := range []string{"topics", "drain-status"} {
		if !strings.Contains(usage, want) {
			t.Errorf("usage text missing %q: %s", want, usage)
		}
	}
}

func TestCmdTopicsRendersHumanOutput(t *testing.T) {
	ctx := clitest.NewContext(t)

	resp := topicsGraphResponse{
		Nodes: []topicNode{
			{Kind: "member", ID: "member:marketing-crew/researcher", Label: "researcher"},
			{Kind: "external", ID: "external:vision-walk", Label: "vision-walk"},
			{Kind: "knowledge_sink", ID: "prefix:research-inbox/*", Label: "research-inbox/*"},
			{Kind: "knowledge_sink", ID: "prefix:audience-scan/*", Label: "audience-scan/*"},
			{Kind: "decision", ID: "decision:audience-update", Label: "audience-update"},
		},
		Edges: []topicEdge{
			{From: "external:vision-walk", To: "member:marketing-crew/researcher", Prefix: "", Kind: "external_producer"},
			{From: "prefix:research-inbox/*", To: "member:marketing-crew/researcher", Prefix: "research-inbox/*", Kind: "intake"},
			{From: "member:marketing-crew/researcher", To: "prefix:audience-scan/*", Prefix: "audience-scan/*", Kind: "output"},
			{From: "member:marketing-crew/researcher", To: "decision:audience-update", Prefix: "audience-update", Kind: "decision_owned"},
		},
	}
	resp.Nodes[0].Ref.Team = "marketing-crew"
	resp.Nodes[0].Ref.Member = "researcher"

	ctx.Respond("GET", "/topics/graph", resp)

	stdout, _, err := clitest.Output(t, func() error {
		return cmdTopics(ctx, []string{})
	})
	if err != nil {
		t.Fatalf("cmdTopics: %v", err)
	}

	for _, want := range []string{
		"Topic Flow Graph (all teams)",
		"marketing-crew/researcher",
		"audience-scan/* (output)",
		"audience-update (decision_owned)",
		"<- research-inbox/* (intake)",
		"Validation: clean",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q\n--- output ---\n%s", want, stdout)
		}
	}
}

func TestCmdTopicsShowsValidationErrors(t *testing.T) {
	ctx := clitest.NewContext(t)
	resp := topicsGraphResponse{
		Nodes: []topicNode{
			{Kind: "member", ID: "member:t/m", Label: "m"},
		},
		Edges: nil,
		Validation: topicValidation{
			Findings: []topicFinding{
				{Rule: "orphan_input", Severity: "error", Member: topicMemberRef{Team: "t", Member: "m"}, Prefix: "x/*", Detail: "no producer"},
			},
			Errors: 1,
		},
	}
	resp.Nodes[0].Ref.Team = "t"
	resp.Nodes[0].Ref.Member = "m"
	ctx.Respond("GET", "/topics/graph", resp)

	stdout, _, err := clitest.Output(t, func() error {
		return cmdTopics(ctx, []string{"--team", "t"})
	})
	if err == nil {
		t.Errorf("expected non-nil error when validation has error findings (cliapp uses err for exit code)")
	}
	for _, want := range []string{
		"Validation: 1 error(s)",
		"orphan_input",
		"no producer",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q\n--- output ---\n%s", want, stdout)
		}
	}
}

func TestCmdTopicsTeamFilter_PassesQueryParam(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/graph", topicsGraphResponse{})

	_, _, _ = clitest.Output(t, func() error {
		return cmdTopics(ctx, []string{"--team", "marketing-crew"})
	})
	req := ctx.LastRequest()
	if req.Query.Get("team") != "marketing-crew" {
		t.Errorf("expected team query param, got %v", req.Query)
	}
}

func TestCmdDrainStatus_NoEntries(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/drain-status", drainStatusResponse{
		Note: "drain-status backend not wired (KnowledgeQuery is nil)",
	})
	stdout, _, err := clitest.Output(t, func() error {
		return cmdDrainStatus(ctx, []string{})
	})
	if err != nil {
		t.Fatalf("cmdDrainStatus: %v", err)
	}
	if !strings.Contains(stdout, "backend not wired") {
		t.Errorf("expected backend-note in output, got %s", stdout)
	}
	if !strings.Contains(stdout, "no drain-status entries") {
		t.Errorf("expected empty placeholder, got %s", stdout)
	}
}

func TestCmdDrainStatus_RendersEntries(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/drain-status", drainStatusResponse{
		Entries: []drainStatusEntry{
			{
				Member:        topicMemberRef{Team: "marketing-crew", Member: "researcher"},
				Prefix:        "research-inbox/*",
				UnroutedCount: 7,
				OldestAt:      "2026-04-20T00:00:00Z",
				OldestAgeSecs: 5 * 24 * 3600,
			},
		},
	})
	stdout, _, err := clitest.Output(t, func() error {
		return cmdDrainStatus(ctx, []string{"--team", "marketing-crew"})
	})
	if err != nil {
		t.Fatalf("cmdDrainStatus: %v", err)
	}
	if !strings.Contains(stdout, "Drain Status (team=marketing-crew)") {
		t.Errorf("expected team header, got %s", stdout)
	}
	if !strings.Contains(stdout, "research-inbox/*") {
		t.Errorf("expected prefix in output, got %s", stdout)
	}
	if !strings.Contains(stdout, "unrouted=7") {
		t.Errorf("expected unrouted count, got %s", stdout)
	}
	if !strings.Contains(stdout, "oldest 5d") {
		t.Errorf("expected oldest-age in days, got %s", stdout)
	}
	req := ctx.LastRequest()
	if req.Query.Get("team") != "marketing-crew" {
		t.Errorf("expected team query, got %v", req.Query)
	}
}
