package measures

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/opportunities"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fixedTools struct {
	calls    []toolfriction.Call
	from, to time.Time
}

func (f *fixedTools) ToolCalls(_ context.Context, from, to time.Time) (toolfriction.Window, error) {
	f.from, f.to = from, to
	return toolfriction.Window{Calls: f.calls}, nil
}

type fixedLibrary opportunities.Library

func (f fixedLibrary) Library(context.Context) opportunities.Library { return opportunities.Library(f) }

func ritualCalls() []toolfriction.Call {
	var calls []toolfriction.Call
	at := time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)
	for run := 0; run < 3; run++ {
		for i, step := range [][2]string{{"search-hub", "search-hub query"}, {"prompt-manager", "prompt-manager skill read"}, {"vrooli-memory", "vrooli-memory journal note"}} {
			calls = append(calls, toolfriction.Call{RunID: fmt.Sprintf("run-%d", run), CallEventID: fmt.Sprintf("c%d-%d", run, i), At: at.Add(time.Duration(i) * time.Minute), Owner: step[0], CommandPath: step[1], ResultTokens: 50, Features: toolfriction.Features{Command: step[1] + " x"}})
		}
	}
	return calls
}

// The measure is registry-only: a coverage row, then opportunities with owner,
// impact, evidence and the verifying check as flat strings. `since` bounds the
// window's start.
func TestOpportunitiesMeasureServesRankedRowsBeyondCoverage(t *testing.T) {
	h := NewHandler(nil, func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) })
	tools := &fixedTools{calls: ritualCalls()}
	h.tools, h.programs = tools, fixedLibrary{Programs: []opportunities.Program{{Name: "x.other", Scenario: "x"}}}
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: UsageOpportunities, Params: map[string]string{"window": "last_7d", "since": "2026-10-28T00:00:00Z"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC); !tools.from.Equal(want) {
		t.Fatalf("window start %v, want since %v", tools.from, want)
	}
	if len(served.Fields) != 2 {
		t.Fatalf("rows = %v, want coverage and one sequence", served.Fields)
	}
	coverage, row := served.Fields[0], served.Fields[1]
	if coverage["kind"] != "coverage" || coverage["runs"] != "3" || coverage["calls"] != "9" || !strings.Contains(coverage["sentence"], "9 command calls (9 Vrooli CLI) in 3 runs at the current fact version") {
		t.Fatalf("coverage = %v", coverage)
	}
	if row["saved_label"] != "calls saved" || !strings.Contains(row["saved_basis"], "less the one program call") {
		t.Errorf("row states no basis for its number: %v", row)
	}
	if row["kind"] != "sequence" || row["verdict"] != "candidate" || row["owner"] != "prompt-manager" || row["sessions"] != "3" || row["calls_saved"] != "6" ||
		row["subject"] != "search-hub query → prompt-manager skill read → vrooli-memory journal note" || row["verify"] == "" || strings.Count(row["run_ids"], "\n") != 2 {
		t.Fatalf("row = %v", row)
	}
}

func TestOpportunitiesMeasureRefusesWithoutSourcesOrWithBadSince(t *testing.T) {
	h := NewHandler(nil, nil)
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: UsageOpportunities, Params: map[string]string{"window": "last_7d"}}); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("unwired measure error = %v, want unavailable", err)
	}
	h.tools, h.programs = &fixedTools{}, fixedLibrary{}
	if _, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: UsageOpportunities, Params: map[string]string{"window": "last_7d", "since": "yesterday"}}); err == nil || !strings.Contains(err.Error(), "RFC 3339") {
		t.Fatalf("bad since error = %v", err)
	}
}

// The CLI and `measures get` send only the parameters the operator typed, so
// every windowed measure must run with none: the declared default applies.
func TestWindowedMeasuresRunWithEmptyParams(t *testing.T) {
	h := NewHandler(nil, func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) })
	tools := &fixedTools{calls: ritualCalls()}
	h.tools, h.programs = tools, fixedLibrary{}
	h.SetInterventions(&interventionSummaries{})
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{UsageOpportunities, UsageInterventions} {
		if _, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: name, Params: map[string]string{}}); err != nil {
			t.Fatalf("%s with {} params: %v", name, err)
		}
	}
	if want := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC); !tools.from.Equal(want) {
		t.Fatalf("opportunities default window starts %v, want last_7d %v", tools.from, want)
	}
}

// Every action's verify line names `agent-manager usage opportunities --since
// <date of the change>`: with the placeholder given a date and the parameters
// the CLI sends (only since), the measure must run.
func TestVerifyCommandOfEveryOpportunityRuns(t *testing.T) {
	h := NewHandler(nil, func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) })
	tools := &fixedTools{calls: ritualCalls()}
	h.tools, h.programs = tools, fixedLibrary{}
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: UsageOpportunities, Params: map[string]string{}})
	if err != nil || len(first.Fields) < 2 {
		t.Fatalf("opportunities with {} params: rows %v err %v", first.Fields, err)
	}
	verify := regexp.MustCompile("`(agent-manager usage opportunities --since <[^>]+>)`").FindStringSubmatch(first.Fields[1]["verify"])
	if verify == nil {
		t.Fatalf("verify line %q names no runnable command", first.Fields[1]["verify"])
	}
	for _, date := range []string{"2026-10-28", "2026-10-28T00:00:00Z"} {
		since := strings.Fields(regexp.MustCompile("<[^>]+>").ReplaceAllString(verify[1], date))[4]
		if _, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: UsageOpportunities, Params: map[string]string{"since": since}}); err != nil {
			t.Fatalf("verify command with --since %s: %v", since, err)
		}
	}
	if want := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC); !tools.from.Equal(want) {
		t.Fatalf("verify window starts %v, want %v", tools.from, want)
	}
}

// The tool friction RPC reads the window every other tool read gets: a custom
// range is clamped to MaxLookback before its end, and never past now, so one
// request cannot materialise months of calls.
func TestToolFrictionWindowIsClampedLikeEveryToolRead(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	h := NewHandler(nil, func() time.Time { return now })
	tools := &fixedTools{calls: ritualCalls()}
	h.tools = tools
	custom := &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Custom{Custom: &sharedmeasurepb.CustomRange{From: timestamppb.New(now.Add(-400 * 24 * time.Hour)), To: timestamppb.New(now.Add(48 * time.Hour))}}}
	response, err := h.ToolFriction(context.Background(), connect.NewRequest(&measurepb.ToolFrictionRequest{Window: custom}))
	if err != nil {
		t.Fatal(err)
	}
	if !tools.to.Equal(now) || !tools.from.Equal(now.Add(-usageattribution.MaxLookback)) {
		t.Fatalf("read [%v, %v), want [now − %v, now)", tools.from, tools.to, usageattribution.MaxLookback)
	}
	if response.Msg.GetWindowStart() != now.Add(-usageattribution.MaxLookback).Format(time.RFC3339) || response.Msg.GetWindowEnd() != now.Format(time.RFC3339) {
		t.Fatalf("window = %s – %s", response.Msg.GetWindowStart(), response.Msg.GetWindowEnd())
	}
}
