package calibration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"unit-health/internal/testquality"
)

// These doubles test the comparator, not any analyzer's correctness.
func fixture() Case {
	return Case{Input: Input{ID: "harness-1", Profile: "go-syntax-v1", Kind: "parser", Root: "testdata", Files: []string{"x_test.go"}}, Partition: "development", Rationale: "Comparator control", Provenance: "harness unit test",
		Expected: []Expectation{{RuleID: "assertion-observation", RuleVersion: "1", File: "x_test.go", TestID: "TestX/first", Line: 10, Column: 2, Status: testquality.CheckedClean}}}
}
func observation() testquality.Result {
	return testquality.Result{RuleID: "assertion-observation", RuleVersion: "1", SupportProfile: "go-syntax-v1", Target: testquality.Target{File: "x_test.go", TestID: "TestX/first"}, Location: testquality.Location{Line: 10, Column: 2}, Status: testquality.CheckedClean, Reason: testquality.ReasonNone}
}
func constant(results ...testquality.Result) Evaluator {
	return func(context.Context, Input) ([]testquality.Result, error) { return results, nil }
}

func TestPassingControlAndInvertedExpectation(t *testing.T) {
	c := fixture()
	report, err := Run(context.Background(), []Case{c}, "development", constant(observation()))
	if err != nil || report.PassedCases != 1 {
		t.Fatalf("control: %+v %v", report, err)
	}
	c.Expected[0].Status = testquality.Violation
	report, err = Run(context.Background(), []Case{c}, "development", constant(observation()))
	if err != nil || report.FailedCases != 1 || report.MissedViolations != 1 {
		t.Fatalf("inverted oracle was not detected: %+v %v", report, err)
	}
}

func TestEmptyErrorAndPanicCannotPass(t *testing.T) {
	for name, evaluator := range map[string]Evaluator{
		"empty": constant(),
		"error": func(context.Context, Input) ([]testquality.Result, error) {
			return nil, errors.New("unreadable source")
		},
		"panic": func(context.Context, Input) ([]testquality.Result, error) { panic("parser crashed") },
	} {
		t.Run(name, func(t *testing.T) {
			c := fixture()
			c.Expected[0].Status = testquality.Unknown
			c.Expected[0].Reason = testquality.ParseFailure
			r, err := Run(context.Background(), []Case{c}, "development", evaluator)
			if err != nil || r.FailedCases != 1 || len(r.Results[0].Differences) == 0 {
				t.Fatalf("failure passed: %+v %v", r, err)
			}
		})
	}
}

func TestUnknownObservationRemainsInDenominator(t *testing.T) {
	c := fixture()
	c.Expected[0].Status = testquality.Unknown
	c.Expected[0].Reason = testquality.ExternalHelperUnresolved
	o := observation()
	o.Status = testquality.Unknown
	o.Reason = testquality.ExternalHelperUnresolved
	r, err := Run(context.Background(), []Case{c}, "development", constant(o))
	if err != nil || r.TotalCases != 1 || r.UnknownCases != 1 || r.PassedCases != 1 {
		t.Fatalf("unknown denominator: %+v %v", r, err)
	}
}

func TestObservationFromWrongSupportProfileCannotPass(t *testing.T) {
	o := observation()
	o.SupportProfile = "unsupported-future-profile"
	r, err := Run(context.Background(), []Case{fixture()}, "development", constant(o))
	if err != nil || r.FailedCases != 1 {
		t.Fatalf("wrong profile accepted: %+v %v", r, err)
	}
	if r.Results[0].Differences[0].Kind != "profile-mismatch" {
		t.Fatalf("wrong mismatch: %+v", r.Results[0])
	}
}

func TestSubtestIdentityLocationAndFalsePositiveAccounting(t *testing.T) {
	c := fixture()
	second := c.Expected[0]
	second.TestID = "TestX/second"
	second.Line = 20
	c.Expected = append(c.Expected, second)
	o := observation()
	o.Status = testquality.Violation
	c.Forbidden = []Expectation{c.Expected[0]}
	c.Forbidden[0].Status = testquality.Violation
	r, err := Run(context.Background(), []Case{c}, "development", constant(o))
	if err != nil || r.FalsePositives != 1 || r.MissedViolations != 0 || len(r.Results[0].Differences) != 3 {
		t.Fatalf("identity/forbidden mismatch lost: %+v %v", r, err)
	}
}

func TestMissingContradictoryAndMixedPartitionExpectationsFail(t *testing.T) {
	for _, mutate := range []func(*Case){
		func(c *Case) { c.Expected = nil },
		func(c *Case) { c.Expected = append(c.Expected, c.Expected[0]) },
		func(c *Case) { c.Forbidden = []Expectation{c.Expected[0]} },
		func(c *Case) { c.Partition = "reviewed-holdout" },
	} {
		c := fixture()
		mutate(&c)
		if err := ValidateCases([]Case{c}, "development"); err == nil {
			t.Fatal("accepted missing/contradictory or mixed-partition expectations")
		}
	}
}

func TestSemanticReportDeterministicAcrossInputOrder(t *testing.T) {
	a, b := fixture(), fixture()
	b.Input.ID = "harness-2"
	o, extra := observation(), observation()
	extra.Target.TestID = "unexpected"
	extra.Status = testquality.Violation
	r1, err := Run(context.Background(), []Case{a, b}, "development", constant(o, extra))
	if err != nil {
		t.Fatal(err)
	}
	r2, err := Run(context.Background(), []Case{b, a}, "development", constant(extra, o))
	if err != nil {
		t.Fatal(err)
	}
	one, _ := json.Marshal(r1)
	two, _ := json.Marshal(r2)
	if string(one) != string(two) {
		t.Fatalf("semantic report order changed:\n%s\n%s", one, two)
	}
}
