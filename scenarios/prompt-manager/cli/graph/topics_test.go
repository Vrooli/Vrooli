package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestUsageTextMentionsTopicsAndDrainStatus(t *testing.T) {
	usage := usageText()
	for _, want := range []string{"topics", "operating-model", "drain-status"} {
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

func TestCmdTopics_FindingsOutWritesFile(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/graph", topicsGraphResponse{
		Nodes: []topicNode{{Kind: "member", ID: "member:t/m", Label: "m"}},
		Validation: topicValidation{
			Findings: []topicFinding{
				{Rule: "unread_required", Severity: "warning", Member: topicMemberRef{Team: "t", Member: "m"}, Prefix: "x/*", Detail: "no producer"},
			},
			Warnings: 1,
		},
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "findings.json")
	stdout, _, err := clitest.Output(t, func() error {
		return cmdTopics(ctx, []string{"--findings-out", path})
	})
	if err != nil {
		t.Fatalf("cmdTopics: %v", err)
	}
	if !strings.Contains(stdout, "Findings artifact written to "+path) {
		t.Errorf("stdout missing artifact-written notice\n--- output ---\n%s", stdout)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("artifact not written: %v", err)
	}
	var art findingsArtifact
	if err := json.Unmarshal(raw, &art); err != nil {
		t.Fatalf("artifact not valid JSON: %v", err)
	}
	if art.SchemaVersion != findingsArtifactSchemaVersion {
		t.Errorf("SchemaVersion = %d", art.SchemaVersion)
	}
	if art.Warnings != 1 || art.Errors != 0 {
		t.Errorf("counts mismatch: errors=%d warnings=%d", art.Errors, art.Warnings)
	}
	if len(art.Findings) != 1 || art.Findings[0].Rule != "unread_required" {
		t.Errorf("findings mismatch: %+v", art.Findings)
	}
}

func TestCmdTopics_FindingsOutEmptySuppressesWrite(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/graph", topicsGraphResponse{})

	dir := t.TempDir()
	t.Chdir(dir)

	stdout, _, err := clitest.Output(t, func() error {
		return cmdTopics(ctx, []string{})
	})
	if err != nil {
		t.Fatalf("cmdTopics: %v", err)
	}
	if strings.Contains(stdout, "Findings artifact written") {
		t.Errorf("expected no artifact-written notice with default empty flag\n--- output ---\n%s", stdout)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read tempdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			t.Errorf("default run wrote unexpected JSON file %s", e.Name())
		}
	}
}

func TestCmdTopics_FindingsOutWithJSONSuppressesNotice(t *testing.T) {
	// --json + --findings-out: file is written, but stdout must remain
	// pure JSON (no human-readable trailing line that would corrupt
	// parsing).
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/graph", topicsGraphResponse{
		Validation: topicValidation{Warnings: 0},
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "findings.json")

	stdout, _, err := clitest.Output(t, func() error {
		return cmdTopics(ctx, []string{"--json", "--findings-out", path})
	})
	if err != nil {
		t.Fatalf("cmdTopics: %v", err)
	}
	if strings.Contains(stdout, "Findings artifact written") {
		t.Errorf("--json output must not contain human notice; got:\n%s", stdout)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("artifact file missing: %v", err)
	}
	// stdout must be parseable JSON (graph response).
	var graph topicsGraphResponse
	if err := json.Unmarshal([]byte(stdout), &graph); err != nil {
		t.Errorf("--json stdout not valid JSON: %v\n%s", err, stdout)
	}
}

func TestCmdTopics_FindingsOutOnValidationError(t *testing.T) {
	// Errors must still produce a non-zero exit (returned err) AND the
	// findings artifact — CI captures both.
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/graph", topicsGraphResponse{
		Validation: topicValidation{
			Findings: []topicFinding{
				{Rule: "orphan_input", Severity: "error", Member: topicMemberRef{Team: "t", Member: "m"}, Prefix: "x/*", Detail: "no producer"},
			},
			Errors: 1,
		},
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "findings.json")

	_, _, err := clitest.Output(t, func() error {
		return cmdTopics(ctx, []string{"--findings-out", path})
	})
	if err == nil {
		t.Error("expected non-nil error from validation failure")
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("artifact must be written even on validation failure: %v", statErr)
	}
	raw, _ := os.ReadFile(path)
	var art findingsArtifact
	if err := json.Unmarshal(raw, &art); err != nil {
		t.Fatalf("artifact unparseable: %v", err)
	}
	if art.Errors != 1 {
		t.Errorf("artifact missing error count: %+v", art)
	}
}

func TestCmdTopics_FindingsOutWriteFailureDoesNotMaskValidation(t *testing.T) {
	// CONTRACT: telemetry writes failing must not change the exit code.
	// Simulate failure by pointing at a path under a missing parent dir.
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/graph", topicsGraphResponse{})

	missing := filepath.Join(t.TempDir(), "missing-parent", "findings.json")
	stdout, stderr, err := clitest.Output(t, func() error {
		return cmdTopics(ctx, []string{"--findings-out", missing})
	})
	if err != nil {
		t.Fatalf("expected nil error (clean validation), got: %v", err)
	}
	if !strings.Contains(stderr, "warning:") {
		t.Errorf("expected stderr warning on artifact write failure; got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "Validation: clean") {
		t.Errorf("validation output missing despite artifact failure:\n%s", stdout)
	}
}

func TestCmdTopics_UsageMentionsFindingsOut(t *testing.T) {
	usage := usageText()
	if !strings.Contains(usage, "--findings-out") {
		t.Errorf("usage text missing --findings-out: %s", usage)
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
