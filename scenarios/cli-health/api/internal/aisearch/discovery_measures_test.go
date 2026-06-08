package aisearch

import (
	"strings"
	"testing"

	measures "github.com/vrooli/measures-go"
)

type stubMeasureSrc struct{ params []measures.ParamSchema }

func (s stubMeasureSrc) RequestParams(_, _ string) ([]measures.ParamSchema, error) {
	return s.params, nil
}

const measureManifestRaw = `{
  "name": "fixture",
  "groups": [{"name":"backlog","commands":[
    {
      "name":"completed",
      "description":"Count completed backlog items",
      "flags":[{"name":"window"}],
      "binding":{"kind":"connect-rpc","service":"StatsService","method":"BacklogCompletionCount"},
      "governance":{"effect":"read","run_eligible":true},
      "measure":{
        "intent":"How many backlog items were completed in a time window.",
        "questions":["how many backlog items did we complete this week","backlog items closed last month"],
        "params":{"window":{"type":"time_window","default":"this_week"}},
        "result":{"kind":"scalar","value_field":"count","unit":"items"}
      }
    }
  ]}]
}`

func TestAttachMeasures_PopulatesRecord(t *testing.T) {
	records, err := parseManifestRecords("swarm-manager", []byte(measureManifestRaw))
	if err != nil {
		t.Fatalf("parseManifestRecords: %v", err)
	}
	src := stubMeasureSrc{params: []measures.ParamSchema{
		{Name: "window", Type: measures.ParamTypeTimeWindow, MessageType: measures.TimeWindowMessageName},
	}}
	attachMeasures(records, []byte(measureManifestRaw), src)

	var rec *CommandRecord
	for i := range records {
		if records[i].Name == "completed" {
			rec = &records[i]
		}
	}
	if rec == nil {
		t.Fatalf("completed command not found in %+v", records)
	}
	if rec.Measure == nil {
		t.Fatalf("measure record not attached")
	}
	if rec.Measure.Name != "backlog.completed" || rec.Measure.Domain != "backlog" {
		t.Fatalf("measure identity wrong: %+v", rec.Measure)
	}
	if rec.Measure.Tier != "full" {
		t.Fatalf("expected full tier, got %q", rec.Measure.Tier)
	}
	if len(rec.Measure.Params) != 1 || rec.Measure.Params[0].Name != "window" || rec.Measure.Params[0].Type != measures.ParamTypeTimeWindow {
		t.Fatalf("expected proto-derived window param, got %+v", rec.Measure.Params)
	}
	if len(rec.Measure.Questions) != 2 {
		t.Fatalf("questions not carried: %+v", rec.Measure.Questions)
	}
}

func TestAttachMeasures_DegradesWithoutSchema(t *testing.T) {
	records, err := parseManifestRecords("swarm-manager", []byte(measureManifestRaw))
	if err != nil {
		t.Fatal(err)
	}
	// nil source -> Assemble validates on manifest-only data (no params), which
	// is well-formed, so a (paramless) declaration assembles and grades full.
	// The manifest-authored param name is still surfaced for discoverability.
	attachMeasures(records, []byte(measureManifestRaw), nil)
	var rec *CommandRecord
	for i := range records {
		if records[i].Name == "completed" {
			rec = &records[i]
		}
	}
	if rec == nil || rec.Measure == nil {
		t.Fatalf("measure record should still attach without a schema source")
	}
	if rec.Measure.Intent == "" || len(rec.Measure.Questions) == 0 {
		t.Fatalf("curated prose should survive degraded assembly: %+v", rec.Measure)
	}
}

func TestComposeEmbedding_IncludesMeasureQuestions(t *testing.T) {
	r := CommandRecord{
		FullPath: "swarm-manager backlog completed",
		Name:     "completed",
		Source:   SourceManifest,
		Measure: &MeasureRecord{
			Name:      "backlog.completed",
			Intent:    "How many backlog items were completed in a time window.",
			Questions: []string{"how many backlog items did we complete this week"},
		},
	}
	text := composeCommandEmbeddingText(r)
	if !strings.Contains(text, "how many backlog items did we complete this week") {
		t.Fatalf("embedding text must include the measure question; got:\n%s", text)
	}
	if !strings.Contains(text, "time window") {
		t.Fatalf("embedding text must include the measure intent; got:\n%s", text)
	}
}

func TestComposeEmbedding_NoMeasureUnchanged(t *testing.T) {
	// A record without a measure block must not gain measure lines (no regression
	// for the ~all existing commands).
	r := CommandRecord{FullPath: "x y z", Name: "z", Source: SourceManifest}
	if strings.Contains(composeCommandEmbeddingText(r), "Answers:") {
		t.Fatalf("non-measure record should have no Answers line")
	}
}
