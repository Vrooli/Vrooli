package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var defaultEpochLimits = epochThresholds{FlatlineK: 10, OverrunFactor: 2, RefactorGrowth: 300, FeatureFactor: 1.5, WallClock: 24 * time.Hour}

// fixtureNow is two hours after the fixtures' Started time.
var fixtureNow = time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)

func checkFixture(t *testing.T, name string, limits epochThresholds, now time.Time, weighted *float64) *epochReport {
	t.Helper()
	path := filepath.Join("testdata", "epochs", name)
	handle, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	file, err := parseEpochFile(handle)
	if err != nil {
		t.Fatal(err)
	}
	return evaluateEpoch(path, file, limits, now, weighted)
}

func firedNames(report *epochReport) []string {
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
		limits   epochThresholds
		weighted *float64
		want     string
	}{
		{"flatline.md", defaultEpochLimits, nil, "flatline"},
		{"overrun.md", defaultEpochLimits, nil, "overrun"},
		{"growth-refactor.md", defaultEpochLimits, nil, "growth"},
		{"growth-feature.md", defaultEpochLimits, nil, "growth"},
		{"healthy.md", epochThresholds{FlatlineK: 10, OverrunFactor: 2, RefactorGrowth: 300, FeatureFactor: 1.5, WallClock: 24 * time.Hour, SpendThreshold: 1_000_000}, &spent, "spend"},
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
	report := checkFixture(t, "healthy.md", defaultEpochLimits, fixtureNow, nil)
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
	report := checkFixture(t, "healthy.md", defaultEpochLimits, fixtureNow.Add(30*time.Hour), nil)
	if len(report.StepBack) != 0 || len(report.Review) != 1 || !strings.HasPrefix(report.Review[0], "wall-clock:") {
		t.Fatalf("wall clock must prompt a review only: stepBack=%v review=%v", report.StepBack, report.Review)
	}
}

func TestEpochCheckTracksOpenDirectivesMalformedLinesAndAcceptance(t *testing.T) {
	report := checkFixture(t, "open-directive.md", defaultEpochLimits, fixtureNow, nil)
	if strings.Join(report.OpenDirectives, ",") != "D2" {
		t.Fatalf("open directives = %v, want D2", report.OpenDirectives)
	}
	if report.WorkUnits != 2 || len(report.MalformedLogLines) != 1 {
		t.Fatalf("malformed line must be reported and not counted: units=%d malformed=%v", report.WorkUnits, report.MalformedLogLines)
	}
	if !strings.HasPrefix(report.Accepted, "ACCEPTED 2026-09-29T10:00:00Z") {
		t.Fatalf("acceptance line lost: %q", report.Accepted)
	}
}

func TestParseTierWeightsWeighsSolAboveLuna(t *testing.T) {
	weights, err := parseTierWeights("")
	if err != nil || tierWeight("gpt-6-sol", weights) != 10 || tierWeight("gpt-6-luna", weights) != 1 || tierWeight("unknown-model", weights) != 1 {
		t.Fatalf("default tier weights wrong: %v %v", weights, err)
	}
	if weights, err = parseTierWeights("sol=12, luna=0.5"); err != nil || weights["sol"] != 12 || weights["luna"] != 0.5 {
		t.Fatalf("override weights = %v %v", weights, err)
	}
	if _, err := parseTierWeights("sol"); err == nil {
		t.Fatal("malformed weight entry must be refused")
	}
}

func TestParseSliceLineAcceptsUnknownFieldsAndFreeFormAcks(t *testing.T) {
	for _, tc := range []struct {
		name, line, what, metric string
		runtime, test            int
		acks                     []string
		malformed                bool
	}{
		{name: "canonical", line: "2026-09-29T08:00:00Z | unit | exit metric=3 | net runtime lines=+50 | net test lines=-5 | ack=D1", what: "unit", metric: "3", runtime: 50, test: -5, acks: []string{"D1"}},
		{name: "unknown fields ignored", line: "2026-09-29T08:00:00Z | unit | exit metric=3 | tokens=12000 | phase=J02 | ack=-", what: "unit", metric: "3"},
		{name: "free-form ack", line: "2026-09-29T08:00:00Z | unit | exit metric=3 | ack=D2 done; D3 deferred until J02 passes", what: "unit", metric: "3", acks: []string{"D2", "D3"}},
		{name: "extra free text joins the change", line: "2026-09-29T08:00:00Z | unit | shadow restarted | exit metric=3", what: "unit | shadow restarted", metric: "3"},
		{name: "missing metric", line: "2026-09-29T08:00:00Z | unit | ack=D1", malformed: true},
		{name: "bad time", line: "yesterday | unit | exit metric=3", malformed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSliceLine(tc.line)
			if tc.malformed {
				if err == nil {
					t.Fatalf("accepted %q", tc.line)
				}
				return
			}
			if err != nil || got.What != tc.what || got.Metric != tc.metric || got.RuntimeLines != tc.runtime || got.TestLines != tc.test || strings.Join(got.Acks, ",") != strings.Join(tc.acks, ",") {
				t.Fatalf("parsed %+v, %v", got, err)
			}
		})
	}
}

func TestEpochCheckReturnsStepBackExitCodeWithoutExiting(t *testing.T) {
	app := &App{}
	output := captureStdout(t, func() error {
		err := app.effortEpochCheck([]string{filepath.Join("testdata", "epochs", "flatline.md")})
		var exit exitCodeError
		if !errors.As(err, &exit) || exit.code != exitStepBack {
			t.Fatalf("want exit code %d, got %v", exitStepBack, err)
		}
		return nil
	})
	if !strings.Contains(output, "STEP_BACK flatline") {
		t.Fatalf("report not printed before the exit status: %s", output)
	}
}

func TestRunWakeByKeyAsksTheServerToMatchProducerAndKey(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	if err := app.runWake([]string{"--key", "orchestrator-run", "--result", "STEP_BACK flatline"}); err != nil {
		t.Fatal(err)
	}
	requests := recorder.Requests()
	if len(requests) != 1 || requests[0].Method != "POST" || requests[0].Path != "/api/v1/runs/wake-by-key" {
		t.Fatalf("wake by key must be one server-side lookup, got %+v", requests)
	}
}
