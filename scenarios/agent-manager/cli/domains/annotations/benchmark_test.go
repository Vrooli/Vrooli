package annotations

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func boolPtr(v bool) *bool { return &v }

func writeFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGoldRowsSelectTheNamedLabelOnly(t *testing.T) {
	rows := []goldFileRow{
		{RunID: "run-1", Label: "work.blocked_by_tool", Positive: boolPtr(true)},
		{RunID: "run-2", Label: "operator.frustrated", Positive: boolPtr(true)},
		{RunID: "run-3", Labels: map[string]bool{"work.blocked_by_tool": false, "operator.frustrated": true}},
	}
	got := goldRowsFor(rows, "work.blocked_by_tool")
	if len(got) != 2 || got[0].RunID != "run-1" || !got[0].Positive || got[1].RunID != "run-3" || got[1].Positive {
		t.Fatalf("rows = %+v", got)
	}
}

func TestSingleGoldLabelRejectsAmbiguity(t *testing.T) {
	rows := []goldFileRow{
		{RunID: "run-1", Labels: map[string]bool{"work.blocked_by_tool": true}},
		{RunID: "run-2", Labels: map[string]bool{"operator.frustrated": true}},
	}
	if _, err := singleGoldLabel(rows); err == nil {
		t.Fatal("ambiguous labels were accepted")
	}
}

func TestWriteGoldDraftEmitsUnfilledRowsAndTheBenchmarkRejectsThem(t *testing.T) {
	draft := goldDraft{
		Label: "work.blocked_by_tool", DefinitionHash: "hash-current", Window: "30d",
		Runs: []goldDraftRun{{RunID: "run-1", Digest: `{"run_id":"run-1"}`}},
	}
	var out bytes.Buffer
	if err := writeGoldDraft(&out, draft); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"positive":null`) || !strings.Contains(out.String(), `"run_id":"run-1"`) {
		t.Fatalf("draft row = %s", out.String())
	}
	path := writeFile(t, "gold.jsonl", out.String())
	if _, err := readGoldRows(path); err == nil || !strings.Contains(err.Error(), "unfilled") {
		t.Fatalf("unfilled draft was accepted: %v", err)
	}
}

func TestBenchmarkRequestCarriesTheSplitTheFitFileAndTheCandidate(t *testing.T) {
	tuning := writeFile(t, "tuning.jsonl", `{"run_id":"t1","label":"work.blocked_by_tool","positive":true}
{"run_id":"t2","label":"work.blocked_by_tool","positive":false}
`)
	holdout := writeFile(t, "holdout.jsonl", `{"run_id":"h1","label":"work.blocked_by_tool","positive":true}
# comment
{"run_id":"h2","label":"work.blocked_by_tool","positive":false}
`)
	request, err := benchmarkRequestFrom("", holdout, "", "holdout", tuning, "0.55", "winnow:e4b", "decide.local.accurate", "pinned")
	if err != nil {
		t.Fatal(err)
	}
	if request.Label != "work.blocked_by_tool" || request.Split != "holdout" || len(request.Rows) != 2 || len(request.FitRows) != 2 {
		t.Fatalf("request = %+v", request)
	}
	if request.Rows[0].RunID != "h1" || request.FitRows[0].RunID != "t1" || request.GoldFile != holdout {
		t.Fatalf("rows were not routed to the right side: %+v", request)
	}
	if request.Model != "winnow:e4b" || request.ResourceRole != "decide.local.accurate" || request.Source != "pinned" {
		t.Fatalf("candidate = %+v", request)
	}
	if request.Threshold == nil || *request.Threshold != 0.55 {
		t.Fatalf("threshold = %v", request.Threshold)
	}
}

func TestBenchmarkRequestRefusesInconsistentFlags(t *testing.T) {
	gold := writeFile(t, "gold.jsonl", `{"run_id":"r1","label":"work.blocked_by_tool","positive":true}
`)
	other := writeFile(t, "other.jsonl", `{"run_id":"r2","label":"operator.frustrated","positive":true}
`)
	if _, err := benchmarkRequestFrom("", gold, "", "tuning", gold, "", "", "", ""); err == nil || !strings.Contains(err.Error(), "--fit-from") {
		t.Fatalf("err = %v, want --fit-from refused for a tuning split", err)
	}
	if _, err := benchmarkRequestFrom("", gold, "", "holdout", other, "", "", "", ""); err == nil || !strings.Contains(err.Error(), "no row for label") {
		t.Fatalf("err = %v, want a fit file for another label refused", err)
	}
	if _, err := benchmarkRequestFrom("", gold, "operator.frustrated", "tuning", "", "", "", "", ""); err == nil || !strings.Contains(err.Error(), "no gold row names label") {
		t.Fatalf("err = %v, want an unnamed label refused", err)
	}
	for _, bad := range []string{"0", "1.5", "abc"} {
		if _, err := benchmarkRequestFrom("", gold, "", "tuning", "", bad, "", "", ""); err == nil {
			t.Fatalf("threshold %q accepted", bad)
		}
	}
}

func TestRenderBenchmarkPrintsTheMeasurementToRecord(t *testing.T) {
	var out bytes.Buffer
	renderBenchmark(&out, benchmarkResult{
		Label: "work.blocked_by_tool", Split: "holdout", Source: "production", Model: "winnow:e4b", ResourceRole: "decide.local.accurate",
		Labelled: 50, Gold: 48, Unavailable: 2, FitRows: 40, Precision: 0.91, Recall: 0.83, Threshold: 0.4375, Temperature: 2.1, Fitted: true,
		RawECE: 0.21, ECE: 0.04, RawBrier: 0.2, Brier: 0.11,
		Measurement: &measurement{DefinitionHash: "abc", Model: "winnow:e4b", ResourceRole: "decide.local.accurate", Gold: 48, Threshold: 0.4375, Temperature: 2.1, Split: "holdout"},
	})
	text := out.String()
	for _, want := range []string{"winnow:e4b @ decide.local.accurate", "holdout", "fitted on 40 tuning", "ECE 0.2100 -> 0.0400", `"split":"holdout"`, "gate.measurements", "not refitted"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestRenderBenchmarkNamesEveryGroupItRefused(t *testing.T) {
	var out bytes.Buffer
	renderBenchmark(&out, benchmarkResult{
		Label: "work.blocked_by_tool", Refusal: "span more than one",
		Groups: []benchmarkGroup{{Model: "laya:en", ResourceRole: "decide.local", Rows: 30}, {Model: "winnow:e4b", ResourceRole: "decide.local.accurate", Rows: 17}},
	})
	text := out.String()
	if !strings.Contains(text, "laya:en @ decide.local: 30") || !strings.Contains(text, "winnow:e4b @ decide.local.accurate: 17") || strings.Contains(text, "measurement:") {
		t.Fatalf("refusal output = %s", text)
	}
}

func TestTuningOutputSaysItNeverProves(t *testing.T) {
	var out bytes.Buffer
	renderBenchmark(&out, benchmarkResult{Label: "a", Split: "tuning", Gold: 4, Labelled: 4, Measurement: &measurement{Split: "tuning"}})
	if !strings.Contains(out.String(), "never proves") {
		t.Fatalf("tuning output = %s", out.String())
	}
}

func TestCalibrationRequestUnionsGoldFilesAndCountsARunOnce(t *testing.T) {
	first := writeFile(t, "a.jsonl", `{"run_id":"r1","label":"work.blocked_by_tool","positive":true}
{"run_id":"r2","label":"work.blocked_by_tool","positive":false}
`)
	second := writeFile(t, "b.jsonl", `{"run_id":"r2","label":"work.blocked_by_tool","positive":true}
{"run_id":"r3","label":"work.blocked_by_tool","positive":true}
`)
	request, err := calibrationRequestFrom("", []string{first, second}, "work.blocked_by_tool", "laya:en", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Rows) != 3 || request.Rows[1].RunID != "r2" || request.Rows[1].Positive {
		t.Fatalf("rows = %+v, want r2 once with its first label", request.Rows)
	}
	if request.Split != "tuning" || request.Model != "laya:en" {
		t.Fatalf("request = %+v", request)
	}
	if _, err := calibrationRequestFrom("", []string{first}, "operator.frustrated", "", "", ""); err == nil {
		t.Fatal("a file with no row for the label was accepted")
	}
}

func TestRenderCalibrationPrintsBothTablesAndTheStoredTemperature(t *testing.T) {
	bins := make([]reliabilityBin, 10)
	for index := range bins {
		bins[index] = reliabilityBin{Lo: float64(index) / 10, Hi: float64(index+1) / 10}
	}
	raw := append([]reliabilityBin(nil), bins...)
	raw[9] = reliabilityBin{Lo: 0.9, Hi: 1, Count: 20, MeanPredicted: 0.97, ObservedRate: 0.55}
	calibrated := append([]reliabilityBin(nil), bins...)
	calibrated[5] = reliabilityBin{Lo: 0.5, Hi: 0.6, Count: 20, MeanPredicted: 0.56, ObservedRate: 0.55}
	var out bytes.Buffer
	renderCalibration(&out, benchmarkResult{
		Label: "work.blocked_by_tool", Model: "laya:en", ResourceRole: "decide.local", Source: "production",
		Labelled: 20, Gold: 20, Positives: 11, Temperature: 3.2, Fitted: true, StoredTemperature: 1,
		RawReliability: raw, Reliability: calibrated, RawECE: 0.42, ECE: 0.02, RawBrier: 0.3, Brier: 0.2,
	})
	text := out.String()
	for _, want := range []string{"laya:en @ decide.local", "stored vectors (T=1.0000) · ECE 0.4200", "calibrated (T=3.2000, fitted) · ECE 0.0200", "0.9 - 1.0", "0.970", "0.550", "now: 1.0000"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
	if strings.Count(text, "mean predicted") != 2 {
		t.Fatalf("want a table for the stored and the calibrated vectors:\n%s", text)
	}
}

func TestRenderBenchmarkNamesUnobservedRowsInsteadOfDroppingThem(t *testing.T) {
	var out bytes.Buffer
	renderBenchmark(&out, benchmarkResult{
		Label: "operator.frustrated", Split: "holdout", Source: "pinned", Labelled: 4, Gold: 0,
		Unobserved: 2, UnobservedSubjects: []string{"r1-m2", "r1-m5"},
	})
	text := out.String()
	for _, want := range []string{"2 labelled row(s) have no stored answer", "r1-m2, r1-m5"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestGoldDraftRowsForMessagesCarryTheScopeID(t *testing.T) {
	var out bytes.Buffer
	draft := goldDraft{Label: "operator.frustrated", Runs: []goldDraftRun{{RunID: "run-1", ScopeID: "run-1-m3", Digest: `{"message":{}}`}}}
	if err := writeGoldDraft(&out, draft); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"scope_id":"run-1-m3"`) {
		t.Fatalf("draft row = %s", out.String())
	}
}
