package usageattribution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/pricing"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/toolfriction"

	"github.com/google/uuid"
)

// The recall fixture (SPEC §12.4 P4-A2) is real command calls from this week's
// web-console Codex rollouts and Claude Code sessions, sanitised:
// ../toolfriction/testdata/recall.json. Each seeded friction must come out of
// the production path (projection, observation, profiles, findings, the
// service) as an owner-level finding with a fileable report.

type recallFixture struct {
	Runs []struct {
		ID     string `json:"id"`
		Events []struct {
			Seed       string `json:"seed"`
			At         string `json:"at"`
			Tool       string `json:"tool"`
			Command    string `json:"command"`
			ExitCode   *int   `json:"exitCode"`
			DurationMS int64  `json:"durationMs"`
			Output     string `json:"output"`
			Usage      *struct {
				Input, CachedInput, Output int
			} `json:"usage"`
		} `json:"events"`
	} `json:"runs"`
}

// elided stands for output bytes the fixture leaves out.
var elided = regexp.MustCompile(`«BYTES:(\d+)»`)

// expand restores an elided output's size with filler words, so token
// estimates and context tax read the real weight.
func expand(text string) string {
	return elided.ReplaceAllStringFunc(text, func(marker string) string {
		n, _ := strconv.Atoi(elided.FindStringSubmatch(marker)[1])
		return strings.Repeat("elided ", n/7+1)[:n]
	})
}

// recallCalls projects the fixture's runs through the read model, as the
// production projection does, and reads each stored fact back as the
// repository does.
func recallCalls(t *testing.T) []toolfriction.Call {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "toolfriction", "testdata", "recall.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture recallFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	var calls []toolfriction.Call
	for _, fr := range fixture.Runs {
		run := &domain.Run{ID: uuid.MustParse(fr.ID), Status: domain.RunStatusComplete, Tag: "recall", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}}
		var events []*domain.RunEvent
		add := func(at time.Time, kind domain.RunEventType, data domain.EventPayload) *domain.RunEvent {
			event := &domain.RunEvent{ID: uuid.New(), RunID: run.ID, Sequence: int64(len(events) + 1), EventType: kind, Timestamp: at, Data: data}
			events = append(events, event)
			return event
		}
		for i, e := range fr.Events {
			at, err := time.Parse(time.RFC3339Nano, e.At)
			if err != nil {
				t.Fatal(err)
			}
			at = at.Add(time.Duration(i) * time.Millisecond)
			if e.Usage != nil {
				add(at, domain.EventTypeMetric, &domain.UsageEventData{InputTokens: e.Usage.Input, CacheReadTokens: e.Usage.CachedInput, OutputTokens: e.Usage.Output})
				continue
			}
			id := fmt.Sprintf("call-%d", i)
			input := map[string]any{"cmd": e.Command}
			if e.Tool == "Bash" {
				input = map[string]any{"command": e.Command}
			}
			add(at, domain.EventTypeToolCall, &domain.ToolCallEventData{ToolName: e.Tool, ToolCallID: id, Input: input})
			duration := e.DurationMS
			add(at.Add(time.Duration(duration)*time.Millisecond), domain.EventTypeToolResult, &domain.ToolResultEventData{ToolName: e.Tool, ToolCallID: id, Output: expand(e.Output), Success: e.ExitCode != nil && *e.ExitCode == 0, ExitCode: e.ExitCode, DurationMS: &duration})
		}
		facts, _ := invocationreadmodel.Project(run, events, time.Now(), nil)
		for _, fact := range facts {
			if fact.Tool != nil {
				calls = append(calls, toolfriction.Observe(toolfriction.Fact{InvocationFact: fact.InvocationFact, RunID: fact.RunID, At: fact.OccurredAt, Runner: fact.RunnerType}, *fact.Tool))
			}
		}
	}
	return calls
}

type recallTools []toolfriction.Call

func (r recallTools) ToolCalls(context.Context, time.Time, time.Time) (toolfriction.Window, error) {
	return toolfriction.Window{Calls: r}, nil
}

type noReadings struct{}

func (noReadings) QuotaReadingSpans(context.Context, string, time.Time) ([]pricing.QuotaReadingSpan, error) {
	return nil, nil
}

func (noReadings) ScopedQuotaReadingSpans(context.Context, string, string, time.Time) ([]pricing.QuotaReadingSpan, error) {
	return nil, nil
}

type memoryStatuses map[string]ProblemStatus

func (m memoryStatuses) FrictionStatuses(context.Context) (map[string]ProblemStatus, error) {
	return m, nil
}

func (m memoryStatuses) SetFrictionStatus(_ context.Context, status ProblemStatus) error {
	m[status.Fingerprint] = status
	return nil
}

type capturedBugs []FrictionBug

func (c *capturedBugs) FileFrictionBug(_ context.Context, bug FrictionBug) (string, error) {
	*c = append(*c, bug)
	return fmt.Sprintf("knw-%d", len(*c)), nil
}

func TestRecallFixtureFindsEverySeededFrictionWithItsOwner(t *testing.T) {
	bugs := &capturedBugs{}
	service := Service{Readings: noReadings{}, Statuses: memoryStatuses{}, Bugs: bugs, Tools: recallTools(recallCalls(t))}
	now := time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC)
	report, err := service.Friction(context.Background(), time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.Tools.Coverage.Commands == 0 || !strings.Contains(report.Tools.Coverage.Sentence(), "outcome known for") {
		t.Fatalf("coverage = %+v", report.Tools.Coverage)
	}
	for _, seed := range []struct {
		name, kind, owner string
		matches           func(ToolFinding) bool
	}{
		{"identity_refresh_unavailable across CLIs", toolfriction.KindFailure, "vrooli", func(f ToolFinding) bool {
			return f.ErrorLine == "identity_refresh_unavailable" && f.CommandPath == "" && f.Calls >= 6
		}},
		{"park refusal inside test-genie runs wait", toolfriction.KindDegraded, "agent-manager", func(f ToolFinding) bool {
			return f.CommandPath == "test-genie runs wait" && strings.Contains(f.ErrorLine, "park unavailable") && strings.Contains(f.ErrorLine, "waiting inline")
		}},
		{"unknown agent-manager command", toolfriction.KindShapeGuess, "agent-manager", func(f ToolFinding) bool { return f.CommandPath == "agent-manager" }},
		{"unknown program-runtime command", toolfriction.KindShapeGuess, "program-runtime", func(f ToolFinding) bool { return f.CommandPath == "program-runtime" }},
		{"unknown device-control command", toolfriction.KindShapeGuess, "device-control", func(f ToolFinding) bool { return f.CommandPath == "device-control" }},
		{"test-genie admission saturated", toolfriction.KindFailure, "test-genie", func(f ToolFinding) bool { return strings.Contains(f.ErrorLine, "admission is saturated") }},
		{"CLI build admission denied across CLIs", toolfriction.KindFailure, "vrooli", func(f ToolFinding) bool { return strings.Contains(f.ErrorLine, "CLI build admission deny") }},
		{"prompt-manager skill output weight", toolfriction.KindOutputWeight, "prompt-manager", func(f ToolFinding) bool { return f.CommandPath == "prompt-manager skill read" && f.Cost > 0 }},
		{"vrooli locks output weight", toolfriction.KindOutputWeight, "vrooli", func(f ToolFinding) bool { return f.CommandPath == "vrooli locks" && f.Cost > 0 }},
		{"polling device-control session list", toolfriction.KindPolling, "device-control", func(f ToolFinding) bool { return f.CommandPath == "device-control session list" }},
	} {
		var found *ToolFinding
		for i, f := range report.ToolFindings {
			if f.Kind == seed.kind && f.Owner == seed.owner && seed.matches(f) {
				found = &report.ToolFindings[i]
				break
			}
		}
		if found == nil {
			t.Errorf("%s: no %s finding owned by %s", seed.name, seed.kind, seed.owner)
			continue
		}
		if found.Description == "" || len(found.RunIDs) == 0 || found.Cost <= 0 || found.CostShare <= 0 {
			t.Errorf("%s: finding lacks description, evidence or cost: %+v", seed.name, found.Finding)
		}
		filed, err := service.ReportFriction(context.Background(), found.Fingerprint, time.Time{}, now)
		if err != nil || filed.Status != StatusReported || filed.BugID == "" {
			t.Errorf("%s: report = %+v, %v", seed.name, filed, err)
			continue
		}
		bug := (*bugs)[len(*bugs)-1]
		if bug.Fingerprint != found.Fingerprint || bug.Context["owner"] != seed.owner || !strings.Contains(bug.Description, found.Kind) {
			t.Errorf("%s: fileable report = %+v", seed.name, bug)
		}
		// Reporting twice files nothing new: the inbox deduplicates on the fingerprint.
		before := len(*bugs)
		if again, err := service.ReportFriction(context.Background(), found.Fingerprint, time.Time{}, now); err != nil || again.BugID != filed.BugID || len(*bugs) != before {
			t.Errorf("%s: second report = %+v, %v", seed.name, again, err)
		}
	}
	// The fixture's sqlite3 reads of agent-manager's database are one run's:
	// counted as bypass calls, but one agent's habit is not a pattern
	// (toolfriction.MinPatternRuns), here or in the opportunities.
	if report.Tools.Totals.BypassCalls < 4 {
		t.Errorf("bypass calls = %d, want the seeded sqlite3 reads counted", report.Tools.Totals.BypassCalls)
	}
	for _, f := range report.ToolFindings {
		if f.Kind == toolfriction.KindBypass && f.Runs < toolfriction.MinPatternRuns {
			t.Errorf("a bypass of %d runs is a finding: %+v", f.Runs, f.Finding)
		}
	}
	if t.Failed() {
		for _, f := range report.ToolFindings {
			t.Logf("%-14s %-18s %-28s %q", f.Kind, f.Owner, f.CommandPath, f.ErrorLine)
		}
	}
}

// Codex shows the model about 10k tokens of a 1 MiB command output and its
// exec result carries nested command results with output_bytes only. The shape
// goes through the same production derivation as the recall fixture (read
// model projection, observation, the service) and the output-weight finding
// and its action state the printed size against what the model read.
func TestRecallCodexExecWithNestedOneMiBOutputIsHeavyOutput(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete, Tag: "recall", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}}
	var events []*domain.RunEvent
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("call-%d", i)
		call := domain.NewToolCallEvent(run.ID, "exec", id, map[string]any{"input": `await tools.exec_command({cmd:"vrooli locks"});`})
		nested := domain.NewToolResultEvent(run.ID, runsignal.NestedCommandTool, "exec-"+id, "", nil)
		data := nested.Data.(*domain.ToolResultEventData)
		exit, known := 0, true
		data.ExitCode, data.OutcomeKnown = &exit, &known
		data.RawOutputJSON = `{"command":["/bin/bash","-lc","vrooli locks"],"output_bytes":1048576}`
		output := domain.NewToolResultEvent(run.ID, "", id, "", nil)
		output.Data.(*domain.ToolResultEventData).OutputBlocks = []domain.ToolOutputBlock{{Type: "input_text", Text: strings.Repeat("x", 40_000)}}
		usage := &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 1, TurnIndex: i + 1}}
		events = append(events, call, nested, output, usage)
	}
	var calls []toolfriction.Call
	facts, _ := invocationreadmodel.Project(run, events, time.Now(), nil)
	for _, fact := range facts {
		if fact.Tool != nil {
			calls = append(calls, toolfriction.Observe(toolfriction.Fact{InvocationFact: fact.InvocationFact, RunID: fact.RunID, At: fact.OccurredAt, Runner: fact.RunnerType}, *fact.Tool))
		}
	}
	service := Service{Readings: noReadings{}, Statuses: memoryStatuses{}, Bugs: &capturedBugs{}, Tools: recallTools(calls)}
	report, err := service.Friction(context.Background(), time.Time{}, time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var found *ToolFinding
	for i, f := range report.ToolFindings {
		if f.Kind == toolfriction.KindOutputWeight && f.CommandPath == "vrooli locks" {
			found = &report.ToolFindings[i]
		}
	}
	if found == nil || found.Cost <= 0 || found.ReadTokens != 10_000 || found.PrintedTokens != 1048576/4 {
		t.Fatalf("finding = %+v, want vrooli locks output weight: 10,000 tokens read of about 262k printed", found)
	}
	for _, a := range toolActions(report.Tools, nil, nil, 1) {
		if a.Kind == toolfriction.KindOutputWeight && strings.Contains(a.Text, "printed about") {
			return
		}
	}
	t.Error("no output-weight action states printed against read tokens")
}
