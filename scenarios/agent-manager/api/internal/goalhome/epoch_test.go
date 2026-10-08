package goalhome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var defaultLimits = Thresholds{FlatlineK: 10, OverrunFactor: 2, RefactorGrowth: 300, FeatureFactor: 1.5, WallClock: 24 * time.Hour}

// fixtureNow is two hours after the synthetic fixtures' Started time.
var fixtureNow = time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)

func readFixture(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{"testdata"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func parseText(t *testing.T, text string) *Epoch {
	t.Helper()
	epoch, err := ParseEpoch(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	return epoch
}

func checkFixture(t *testing.T, name string, limits Thresholds, now time.Time, weighted *float64) *EpochReport {
	t.Helper()
	return EvaluateEpoch(name, parseText(t, readFixture(t, "epochs", name)), limits, now, weighted)
}

func firedNames(report *EpochReport) []string {
	names := []string{}
	for _, trigger := range report.Triggers {
		if trigger.Fired {
			names = append(names, trigger.Name)
		}
	}
	return names
}

func TestEpochCheckFiresEachStepBackTriggerOnItsFixture(t *testing.T) {
	spent := 5_000_000.0
	for _, tc := range []struct {
		fixture  string
		limits   Thresholds
		weighted *float64
		want     string
	}{
		{"flatline.md", defaultLimits, nil, "flatline"},
		{"overrun.md", defaultLimits, nil, "overrun"},
		{"growth-refactor.md", defaultLimits, nil, "growth"},
		{"growth-feature.md", defaultLimits, nil, "growth"},
		{"healthy.md", Thresholds{FlatlineK: 10, OverrunFactor: 2, RefactorGrowth: 300, FeatureFactor: 1.5, WallClock: 24 * time.Hour, SpendThreshold: 1_000_000}, &spent, "spend"},
	} {
		t.Run(tc.fixture+"/"+tc.want, func(t *testing.T) {
			report := checkFixture(t, tc.fixture, tc.limits, fixtureNow, tc.weighted)
			fired := firedNames(report)
			if len(fired) != 1 || fired[0] != tc.want {
				t.Fatalf("fired %v, want only %s: %+v", fired, tc.want, report.Triggers)
			}
			if len(report.StepBack) != 1 || !strings.HasPrefix(report.StepBack[0], tc.want+":") {
				t.Fatalf("STEP_BACK reasons = %v", report.StepBack)
			}
		})
	}
}

func TestEpochCheckReportsAHealthyEpochWithoutTriggers(t *testing.T) {
	report := checkFixture(t, "healthy.md", defaultLimits, fixtureNow, nil)
	if fired := firedNames(report); len(fired) != 0 || len(report.StepBack) != 0 {
		t.Fatalf("healthy epoch fired %v", fired)
	}
	if report.WorkUnits != 5 || report.NetRuntimeLines != -200 || report.NetTestLines != 50 || strings.Join(report.MetricHistory, ",") != "1,2,2,3,4" {
		t.Fatalf("unexpected totals: %+v", report)
	}
	if len(report.OpenDirectives) != 0 {
		t.Fatalf("acknowledged directive reported open: %v", report.OpenDirectives)
	}
	for _, trigger := range report.Triggers {
		if trigger.Name == "spend" && !strings.HasPrefix(trigger.Detail, "not evaluated") {
			t.Fatalf("spend without a threshold must say it was not evaluated: %s", trigger.Detail)
		}
	}
}

func TestEpochCheckWallClockPromptsReviewButNeverStepsBack(t *testing.T) {
	report := checkFixture(t, "healthy.md", defaultLimits, fixtureNow.Add(30*time.Hour), nil)
	if len(report.StepBack) != 0 || len(report.Review) != 1 || !strings.HasPrefix(report.Review[0], "wall-clock:") {
		t.Fatalf("wall clock must prompt a review only: stepBack=%v review=%v", report.StepBack, report.Review)
	}
}

func TestEpochCheckTracksOpenDirectivesMalformedLinesAndAcceptance(t *testing.T) {
	report := checkFixture(t, "open-directive.md", defaultLimits, fixtureNow, nil)
	if strings.Join(report.OpenDirectives, ",") != "D2" {
		t.Fatalf("open directives = %v, want D2", report.OpenDirectives)
	}
	if report.WorkUnits != 2 || len(report.MalformedLogLines) != 1 {
		t.Fatalf("malformed line must be reported and not counted: units=%d malformed=%v", report.WorkUnits, report.MalformedLogLines)
	}
	if !strings.HasPrefix(report.Accepted, "ACCEPTED 2026-09-29T10:00:00Z") || report.AcceptedAt == nil {
		t.Fatalf("acceptance line lost: %q", report.Accepted)
	}
}

func TestAcceptedEpochReportsFiredTriggersWithoutStepBackOrReview(t *testing.T) {
	text := readFixture(t, "epochs", "flatline.md") + "ACCEPTED 2026-09-29T09:40:00Z J02 passes\n"
	report := EvaluateEpoch("flatline.md", parseText(t, text), defaultLimits, fixtureNow.Add(48*time.Hour), nil)
	if len(report.StepBack) != 0 || len(report.Review) != 0 {
		t.Fatalf("an accepted epoch never steps back or prompts a review: stepBack=%v review=%v", report.StepBack, report.Review)
	}
	if len(report.AfterAcceptance) != 2 || !strings.HasPrefix(report.AfterAcceptance[0], "flatline:") || !strings.HasPrefix(report.AfterAcceptance[1], "wall-clock:") {
		t.Fatalf("fired triggers must still be reported: %v", report.AfterAcceptance)
	}
}

func TestOnlyADirectiveDatedAfterAcceptanceReopensTheEpoch(t *testing.T) {
	base := readFixture(t, "epochs", "flatline.md")
	for _, tc := range []struct {
		name      string
		directive string
		reopened  string
	}{
		{name: "unacknowledged earlier directive stays informational", directive: "- D1 2026-09-29T09:35:00Z Log J02 only.", reopened: ""},
		{name: "directive after ACCEPTED reopens", directive: "- D1 2026-09-29T11:00:00Z Audit reopens: acknowledge D1.", reopened: "D1"},
		{name: "date-only directive on a later day reopens", directive: "- D1 (2026-09-30): Audit reopens.", reopened: "D1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(base, "## Directives\n", "## Directives\n"+tc.directive+"\n", 1) + "ACCEPTED 2026-09-29T09:40:00Z J02 passes\n"
			report := EvaluateEpoch("flatline.md", parseText(t, text), defaultLimits, fixtureNow, nil)
			if strings.Join(report.ReopenedBy, ",") != tc.reopened {
				t.Fatalf("reopened by %v, want %q", report.ReopenedBy, tc.reopened)
			}
			if stepsBack := len(report.StepBack) > 0; stepsBack != (tc.reopened != "") {
				t.Fatalf("step-back = %v, want only for a reopened epoch", report.StepBack)
			}
		})
	}
}

func TestBASE23YieldIsMeasuredNeverTheSummedSliceLog(t *testing.T) {
	text := readFixture(t, "bas", "epochs", "E23.md")
	report := EvaluateEpoch("E23.md", parseText(t, text), defaultLimits, fixtureNow, nil)
	if report.NetRuntimeLines != -656 {
		t.Fatalf("summed slice-log deltas = %d, want the recorded -656", report.NetRuntimeLines)
	}
	if report.Yield != nil {
		t.Fatalf("a legacy ACCEPTED line has an unknown yield, got %v from %s", report.Yield, report.YieldSource)
	}
	measured := strings.Replace(text, "ACCEPTED 2026-10-07T00:14:20Z —", "ACCEPTED 2026-10-07T00:14:20Z | yield=−328 runtime lines | gates=G1 |", 1)
	report = EvaluateEpoch("E23.md", parseText(t, measured), defaultLimits, fixtureNow, nil)
	if report.Yield == nil || report.Yield.Value != -328 || report.Yield.Unit != "runtime lines" || report.YieldSource != "accepted" {
		t.Fatalf("yield = %+v from %q, want -328 runtime lines from the ACCEPTED line", report.Yield, report.YieldSource)
	}
}

func TestParseSliceLineAcceptsUnknownFieldsFreeFormAcksAndGateCells(t *testing.T) {
	for _, tc := range []struct {
		name, line, what, metric string
		runtime, test            int
		acks                     []string
		gates                    string
		malformed                bool
	}{
		{name: "canonical", line: "2026-09-29T08:00:00Z | unit | exit metric=3 | net runtime lines=+50 | net test lines=-5 | ack=D1", what: "unit", metric: "3", runtime: 50, test: -5, acks: []string{"D1"}},
		{name: "unknown fields ignored", line: "2026-09-29T08:00:00Z | unit | exit metric=3 | tokens=12000 | phase=J02 | ack=-", what: "unit", metric: "3"},
		{name: "free-form ack", line: "2026-09-29T08:00:00Z | unit | exit metric=3 | ack=D2 done; D3 deferred until J02 passes", what: "unit", metric: "3", acks: []string{"D2", "D3"}},
		{name: "extra free text joins the change", line: "2026-09-29T08:00:00Z | unit | shadow restarted | exit metric=3", what: "unit | shadow restarted", metric: "3"},
		{name: "gate cells", line: "2026-09-29T08:00:00Z | unit | exit metric=3 | gate G3=pass 12/12 | G4=FAIL 253900", what: "unit", metric: "3", gates: "G3 pass 12/12;G4 fail 253900"},
		{name: "missing metric", line: "2026-09-29T08:00:00Z | unit | ack=D1", malformed: true},
		{name: "bad time", line: "yesterday | unit | exit metric=3", malformed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSliceLine(tc.line)
			if tc.malformed {
				if err == nil {
					t.Fatalf("accepted %q", tc.line)
				}
				return
			}
			var gates []string
			for _, cell := range got.Gates {
				gates = append(gates, strings.TrimSpace(cell.Gate+" "+cell.Result+" "+cell.Value))
			}
			if err != nil || got.What != tc.what || got.Metric != tc.metric || got.RuntimeLines != tc.runtime || got.TestLines != tc.test || strings.Join(got.Acks, ",") != strings.Join(tc.acks, ",") || strings.Join(gates, ";") != tc.gates {
				t.Fatalf("parsed %+v, %v", got, err)
			}
		})
	}
}

func TestParseQuantityReadsSignsSeparatorsAndUnits(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		value float64
		unit  string
		ok    bool
	}{
		{"−600 runtime lines", -600, "runtime lines", true},
		{"-1,000 runtime lines (measured owners)", -1000, "runtime lines", true},
		{"+2 journeys", 2, "journeys", true},
		{"about −600", 0, "", false},
	} {
		got, ok := ParseQuantity(tc.raw)
		if ok != tc.ok || got.Value != tc.value || got.Unit != tc.unit {
			t.Fatalf("ParseQuantity(%q) = %+v, %v", tc.raw, got, ok)
		}
	}
}
