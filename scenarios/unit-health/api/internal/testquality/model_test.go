package testquality

import (
	"encoding/json"
	"fmt"
	"testing"
)

func result(id string, status Status) Result {
	return Result{RuleID: "assertion-observation", RuleVersion: "1", SupportProfile: "go-syntax-v1",
		Target: Target{Workspace: "api", File: "service_test.go", TestID: id}, Status: status,
		Reason: ReasonNone, EvidenceKind: Static, Severity: Warning, Enforcement: Advisory,
		EvidenceRefs: []string{"source:service_test.go"}}
}

func TestUnknownReasonsHaveActionableGuidance(t *testing.T) {
	for _, reason := range append(Reasons(), Reason("future")) {
		if reason == ReasonNone {
			continue
		}
		if ReasonGuidance(reason) == "" {
			t.Fatalf("no action for %q", reason)
		}
	}
	if ReasonGuidance(ReasonNone) != "" {
		t.Fatal("clean reason invented a recovery action")
	}
}

// [REQ:UH-ANALYZE-008]
func TestIncompleteCollectionRetainsIndependentFindingsAndDenominators(t *testing.T) {
	row := result("observed", Violation)
	row.Enforcement = Blocking
	report, err := BuildReport("1", []Result{row}, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	report.MarkIncomplete(OwnerUnavailable)
	report.MarkIncomplete(OwnerUnavailable)
	got := report.Normalized()
	if len(got.IncompleteReasons) != 1 || got.UnavailableReason != "" || len(got.Coverage) != 1 || got.Coverage[0].Assessed != 1 || got.Results[0].Status != Violation || got.Results[0].Enforcement != Blocking {
		t.Fatalf("independent evidence weakened: %+v", got)
	}
	got.SchemaVersion = "future"
	invalid := got.Normalized()
	if len(invalid.Coverage) != 0 || invalid.Results[0].Status != Unknown {
		t.Fatal("partial collection bypassed schema integrity")
	}
}

func TestStatusRoundtrip(t *testing.T) {
	for _, status := range []Status{Violation, CheckedClean, Unknown, NotApplicable} {
		t.Run(string(status), func(t *testing.T) {
			in := result("test", status)
			if status == Unknown {
				in.Reason = MissingInput
			}
			data, err := json.Marshal(in)
			if err != nil {
				t.Fatal(err)
			}
			var got Result
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if got.Status != status || got.Enforcement != Advisory || got.EvidenceKind != Static {
				t.Fatalf("lost result contract: %+v", got)
			}
		})
	}
}

func TestHistoricalAndFutureValuesRemainUnknown(t *testing.T) {
	for _, tc := range []struct {
		input  string
		reason Reason
	}{
		{`{}`, MissingAnalysis},
		{`{"status":"future-clean","reasonCode":"none"}`, UnrecognizedValue},
		{`{"status":"unknown","reasonCode":"future-reason"}`, UnrecognizedValue},
		{`{"status":"unknown","reasonCode":"none"}`, MissingInput},
	} {
		var got Result
		if err := json.Unmarshal([]byte(tc.input), &got); err != nil {
			t.Fatal(err)
		}
		if got.Status != Unknown || got.Reason != tc.reason {
			t.Fatalf("%s: %+v", tc.input, got)
		}
	}
}

func TestEveryReasonRoundtrips(t *testing.T) {
	for _, reason := range Reasons() {
		data := fmt.Sprintf(`{"status":"not_applicable","reasonCode":%q}`, reason)
		var got Result
		if err := json.Unmarshal([]byte(data), &got); err != nil {
			t.Fatal(err)
		}
		if got.Reason != reason {
			t.Fatalf("reason %q became %q", reason, got.Reason)
		}
	}
}

func TestCoverageUsesCompleteDenominatorBeforeTruncation(t *testing.T) {
	inputs := []Result{result("violates", Violation), result("clean", CheckedClean), result("unresolved", Unknown), result("benchmark", NotApplicable)}
	report, err := BuildReport("1", inputs, 2, "run:abc/quality-details")
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalResults != 4 || !report.Truncated || len(report.Results) != 2 || report.DetailsRef != "run:abc/quality-details" {
		t.Fatalf("bad bounds: %+v", report)
	}
	if len(report.Coverage) != 1 {
		t.Fatalf("coverage rows: %+v", report.Coverage)
	}
	c := report.Coverage[0]
	if c.Discovered != 4 || c.Assessed != 2 || c.Unknown != 1 || c.NotApplicable != 1 {
		t.Fatalf("bad denominator: %+v", c)
	}
	inputs[0].EvidenceRefs[0] = "mutated"
	if report.Results[0].EvidenceRefs[0] != "source:service_test.go" {
		t.Fatal("report aliases caller evidence")
	}
}

func TestReportRejectsInvisibleTruncationAndDuplicateIdentity(t *testing.T) {
	one := result("one", CheckedClean)
	if _, err := BuildReport("1", []Result{one, result("two", Unknown)}, 1, ""); err == nil {
		t.Fatal("accepted truncated results without retrieval reference")
	}
	if _, err := BuildReport("1", []Result{one, one}, 3, ""); err == nil {
		t.Fatal("duplicate test inflated denominator")
	}
	if _, err := BuildReport("1", nil, 0, ""); err == nil {
		t.Fatal("accepted invalid limit")
	}
	one.Target.TestID = ""
	if _, err := BuildReport("1", []Result{one}, 2, ""); err == nil {
		t.Fatal("accepted absent test identity")
	}
}

func TestAbsentHistoricalReportIsNotAnEmptyCleanReport(t *testing.T) {
	var old struct {
		Quality *Report `json:"quality"`
	}
	if err := json.Unmarshal([]byte(`{}`), &old); err != nil {
		t.Fatal(err)
	}
	if old.Quality != nil {
		t.Fatal("manufactured analysis for historical response")
	}
}

func TestReportSchemaAndDenominatorCompatibility(t *testing.T) {
	current, err := BuildReport("1", []Result{result("case", CheckedClean)}, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   Reason
	}{
		{"current", func(*Report) {}, ""},
		{"absent schema", func(r *Report) { r.SchemaVersion = "" }, MissingAnalysis},
		{"future schema", func(r *Report) { r.SchemaVersion = "test-quality/v99" }, UnsupportedVersion},
		{"missing denominator", func(r *Report) { r.Coverage = nil }, MissingInput},
		{"negative total", func(r *Report) { r.TotalResults = -1 }, MissingInput},
		{"hidden truncation", func(r *Report) { r.TotalResults = 2 }, MissingInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := current
			tc.mutate(&candidate)
			data, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Report
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.UnavailableReason != tc.want {
				t.Fatalf("got reason %q, want %q", decoded.UnavailableReason, tc.want)
			}
			if tc.want == "" {
				if decoded.Results[0].Status != CheckedClean || decoded.Coverage[0].Assessed != 1 {
					t.Fatal("valid report lost scoped assessment")
				}
			} else if decoded.Results[0].Status != Unknown || len(decoded.Coverage) != 0 {
				t.Fatalf("incompatible report claims analysis: %+v", decoded)
			}
			if current.Results[0].Status != CheckedClean {
				t.Fatal("normalization rewrote original evidence")
			}
		})
	}
}

func TestFileScopeNeverInflatesTestCaseDenominators(t *testing.T) {
	testRow := result("test-case", CheckedClean)
	fileRow := testRow
	fileRow.Target.Scope = "file"
	fileRow.Target.TestID = ""
	report, err := BuildReport("1", []Result{testRow, fileRow}, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(report.Coverage) != 2 {
		t.Fatalf("mixed scopes merged: %+v", report.Coverage)
	}
	for _, coverage := range report.Coverage {
		if coverage.Discovered != 1 {
			t.Fatalf("inflated count: %+v", coverage)
		}
	}
	for _, scope := range []string{"", "future"} {
		fileRow.Target.Scope = scope
		if _, err := BuildReport("1", []Result{fileRow}, 10, ""); err == nil {
			t.Fatalf("missing/future scope accepted as file: %s", scope)
		}
	}
}

func TestMixedRuleVersionsCannotShareDenominator(t *testing.T) {
	one, two := result("one", CheckedClean), result("two", CheckedClean)
	two.RuleVersion = "2"
	if _, err := BuildReport("1", []Result{one, two}, 2, ""); err == nil {
		t.Fatal("mixed rule versions collapsed into one denominator")
	}
}
