package measures

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/interventions"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

type interventionSummaries struct {
	summary interventions.Summary
	rows    []interventions.Row
	version string
}

func (s *interventionSummaries) Rows(_ context.Context, _ string, _ interventions.Gate, _, _ time.Time, _ string, limit int) ([]interventions.Row, error) {
	return s.rows[:min(limit, len(s.rows))], nil
}

func (s *interventionSummaries) Summarize(_ context.Context, version string, _ interventions.Gate, _, _ time.Time) (interventions.Summary, error) {
	s.version = version
	return s.summary, nil
}

// An unproven category reports what was classified but no rate: an empty
// cell, never zero, because the classifier cannot tell. Coverage is the
// classified share of the window's interventions.
func TestInterventionsMeasureReportsCoverageAndNoUnprovenRate(t *testing.T) {
	source := &interventionSummaries{summary: interventions.Summary{
		Categories: []interventions.CategorySummary{
			{Category: interventions.QualityMiss, Classified: 7},
			{Category: interventions.EarlyStop, Classified: 2},
			{Category: interventions.ScopeChange, Classified: 9},
		},
		Classified: 18, Unclassified: 2, AgentHours: 12.5,
	}}
	h := NewHandler(nil, func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) })
	h.SetInterventions(source)
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: UsageInterventions, Params: map[string]string{"window": "last_7d"}})
	if err != nil {
		t.Fatal(err)
	}
	if source.version != interventions.ClassifierVersion {
		t.Fatalf("summarized version %q, want the current classifier", source.version)
	}
	if len(served.Fields) != 4 {
		t.Fatalf("rows = %v, want two friction categories, the pooled bucket and the coverage row (scope_change is not friction)", served.Fields)
	}
	quality := served.Fields[0]
	if quality["category"] != "quality_miss" || quality["shown"] != "false" || quality["classified"] != "7" || quality["per_agent_hour"] != "" || quality["interventions"] != "" || quality["owner"] == "" {
		t.Fatalf("quality_miss row = %v, want classified 7, not shown, no rate", quality)
	}
	if pooled := served.Fields[2]; pooled["category"] != string(interventions.PooledFriction) || pooled["shown"] != "false" || pooled["interventions"] != "" || pooled["per_agent_hour"] != "" {
		t.Fatalf("pooled row = %v, want the unproven bucket listed with no rate", pooled)
	}
	all := served.Fields[3]
	if all["category"] != "all" || all["coverage"] != "0.9000" || all["unclassified"] != "2" || all["agent_hours"] != "12.50" {
		t.Fatalf("coverage row = %v", all)
	}
}

// Once the pooled bucket is proven the measure counts it per agent-hour beside
// the categories, as `usage friction` does.
func TestInterventionsMeasureCountsThePooledBucketOnceProven(t *testing.T) {
	original := interventions.ProvenGate
	interventions.ProvenGate = interventions.Gate{"fake:decide": {interventions.PooledFriction: 0.5}}
	defer func() { interventions.ProvenGate = original }()
	source := &interventionSummaries{summary: interventions.Summary{
		Pooled:     interventions.PooledSummary{Shown: 6, Repeats: 1},
		Classified: 20, AgentHours: 12,
	}}
	h := NewHandler(nil, func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) })
	h.SetInterventions(source)
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: UsageInterventions, Params: map[string]string{"window": "last_7d"}})
	if err != nil {
		t.Fatal(err)
	}
	var pooled map[string]string
	for _, row := range served.Fields {
		if row["category"] == string(interventions.PooledFriction) {
			pooled = row
		}
	}
	if pooled == nil || pooled["shown"] != "true" || pooled["interventions"] != "6" || pooled["repeats"] != "1" || pooled["per_agent_hour"] != "0.5000" {
		t.Fatalf("pooled row = %v, want 6 interventions at 0.5 per agent-hour", pooled)
	}
}

func TestInterventionsMeasureIsDefined(t *testing.T) {
	if definitionFor(UsageInterventions).SourceTable == "" {
		t.Fatal("usage.interventions needs a backend definition")
	}
}

// With a proven category the friction section counts it per owner and per
// agent-hour, carries the agent work before it and redacted examples with
// their run and the earlier report they repeat.
func TestInterventionsSectionShowsProvenCategories(t *testing.T) {
	p := 0.9
	source := &interventionSummaries{
		summary: interventions.Summary{
			Categories: []interventions.CategorySummary{
				{Category: interventions.QualityMiss, Classified: 7, Shown: 4, Repeats: 1, Work: interventions.Work{Requests: 20, ToolCalls: 12, Tokens: 9000}},
				{Category: interventions.EarlyStop, Classified: 2},
			},
			Classified: 18, Unclassified: 2, AgentHours: 8,
		},
		rows: []interventions.Row{{EventID: "e2", RunID: "run-1", OccurredAt: "2026-10-29T10:00:00Z", Category: interventions.QualityMiss, Probability: &p, Work: interventions.Work{Requests: 5, ToolCalls: 3, Tokens: 2500}, RepeatOf: "e1", Text: "Still broken"}},
	}
	gate := interventions.Gate{"fake:decide": {interventions.QualityMiss: 0.5}}
	section, err := interventionsSection(context.Background(), source, gate, time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(section.GetCategories()) != 1 || section.GetUnshownReason() != "" || section.GetOperatorMessages() != 20 || section.GetCoverage() != 0.9 {
		t.Fatalf("section = %v", section)
	}
	c := section.GetCategories()[0]
	if c.GetCategory() != "quality_miss" || c.GetOwner() == "" || c.GetInterventions() != 4 || c.GetPerAgentHour() != 0.5 || c.GetRepeats() != 1 || c.GetWorkRequests() != 20 || c.GetWorkToolCalls() != 12 || c.GetWorkTokens() != 9000 {
		t.Fatalf("category = %v", c)
	}
	e := section.GetExamples()
	if len(e) != 1 || e[0].GetRunId() != "run-1" || e[0].GetRepeatOfEventId() != "e1" || e[0].GetText() != "Still broken" || e[0].GetOwner() != c.GetOwner() || e[0].GetProbability() != 0.9 || e[0].GetWorkTokens() != 2500 {
		t.Fatalf("examples = %v", e)
	}
}

// When nothing is shown the section states coverage and why, never "no friction".
func TestInterventionsSectionSaysWhyNothingIsShown(t *testing.T) {
	source := &interventionSummaries{summary: interventions.Summary{
		Categories: []interventions.CategorySummary{{Category: interventions.QualityMiss, Classified: 7}},
		Classified: 18, Unclassified: 2, AgentHours: 8,
	}}
	for name, tc := range map[string]struct {
		gate interventions.Gate
		want string
	}{
		"no category proven": {interventions.Gate{}, "no category has passed the accuracy gate yet"},
		"proven elsewhere":   {interventions.Gate{"fake:decide": {interventions.TooDense: 0.5}}, "falls in a category that has passed"},
	} {
		section, err := interventionsSection(context.Background(), source, tc.gate, time.Time{}, time.Time{}, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(section.GetCategories()) != 0 || len(section.GetExamples()) != 0 || !strings.Contains(section.GetUnshownReason(), tc.want) || section.GetOperatorMessages() != 20 || section.GetClassified() != 18 || section.GetCoverage() != 0.9 {
			t.Errorf("%s: section = %v", name, section)
		}
	}
	section, err := interventionsSection(context.Background(), &interventionSummaries{}, interventions.Gate{}, time.Time{}, time.Time{}, 10)
	if err != nil || !strings.Contains(section.GetUnshownReason(), "no operator messages") {
		t.Fatalf("empty window section = %v, %v", section, err)
	}
}

// The friction RPC carries the section; with the empty production gate it
// states why nothing is shown.
func TestUsageFrictionCarriesTheInterventionsSection(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	h.SetInterventions(&interventionSummaries{summary: interventions.Summary{Classified: 9, Unclassified: 1}})
	response, err := h.UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	section := response.Msg.GetInterventions()
	if section.GetOperatorMessages() != 10 || len(section.GetCategories()) != 0 || !strings.Contains(section.GetUnshownReason(), "accuracy gate") {
		t.Fatalf("interventions = %v", section)
	}
}

// A pooled gate shows "operator friction (category unproven)": real counts,
// with per-category hints and examples carrying the same unproven flag.
func TestInterventionsSectionShowsThePooledBucketWithUnprovenHints(t *testing.T) {
	source := &interventionSummaries{
		summary: interventions.Summary{
			Classified: 30, AgentHours: 10,
			Pooled: interventions.PooledSummary{Shown: 5, Repeats: 1, Work: interventions.Work{Requests: 10, ToolCalls: 4, Tokens: 700}, Hints: []interventions.CategoryHint{{Category: interventions.EarlyStop, Count: 2}, {Category: interventions.QualityMiss, Count: 3}}},
		},
		rows: []interventions.Row{{EventID: "e1", RunID: "run-1", Category: interventions.QualityMiss, Pooled: true, Text: "Still broken"}},
	}
	section, err := interventionsSection(context.Background(), source, interventions.Gate{"fake:decide": {interventions.PooledFriction: 0.5}}, time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(section.GetCategories()) != 1 || section.GetUnshownReason() != "" {
		t.Fatalf("section = %v", section)
	}
	c := section.GetCategories()[0]
	if !c.GetPooled() || c.GetCategory() != "operator_friction" || c.GetInterventions() != 5 || c.GetPerAgentHour() != 0.5 || c.GetRepeats() != 1 || c.GetWorkTokens() != 700 || len(c.GetHints()) != 2 || c.GetHints()[1].GetCategory() != "quality_miss" || c.GetHints()[1].GetCount() != 3 || c.GetHints()[1].GetOwner() == "" {
		t.Fatalf("pooled category = %v", c)
	}
	if e := section.GetExamples(); len(e) != 1 || !e[0].GetPooled() {
		t.Fatalf("examples = %v, want one flagged pooled", e)
	}
}
