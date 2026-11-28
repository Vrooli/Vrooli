// Package calibration compares independently authored expectations with adapter
// observations. Evaluators receive source inputs, never expected outcomes.
package calibration

import (
	"context"
	"fmt"
	"sort"
	"unit-health/internal/testquality"
)

type Input struct {
	ID                string            `json:"id"`
	Profile           string            `json:"profile"`
	Kind              string            `json:"kind"` // parser or native; native execution is adapter-owned.
	Root              string            `json:"root"`
	Files             []string          `json:"files"`
	GOOS              string            `json:"goos,omitempty"`
	GOARCH            string            `json:"goarch,omitempty"`
	RegisteredHelpers []string          `json:"registeredHelpers,omitempty"`
	TestKinds         map[string]string `json:"testKinds,omitempty"`
	SelectedTests     []string          `json:"selectedTests,omitempty"`
}

type Expectation struct {
	Scope       string             `json:"scope,omitempty"`
	RuleID      string             `json:"ruleId"`
	RuleVersion string             `json:"ruleVersion"`
	File        string             `json:"file"`
	TestID      string             `json:"testId"`
	Line        int                `json:"line"`
	Column      int                `json:"column"`
	Status      testquality.Status `json:"status"`
	Reason      testquality.Reason `json:"reason,omitempty"`
}

type Case struct {
	Input      Input         `json:"input"`
	Partition  string        `json:"partition"` // development and reviewed-holdout never merge.
	Rationale  string        `json:"rationale"`
	Provenance string        `json:"provenance"`
	Expected   []Expectation `json:"expected"`
	Forbidden  []Expectation `json:"forbidden"`
}

type Difference struct {
	Kind     string             `json:"kind"`
	Identity string             `json:"identity"`
	Expected testquality.Status `json:"expected,omitempty"`
	Observed testquality.Status `json:"observed,omitempty"`
	Detail   string             `json:"detail,omitempty"`
}

type CaseResult struct {
	ID          string               `json:"id"`
	Passed      bool                 `json:"passed"`
	Unknown     bool                 `json:"unknown"`
	Error       string               `json:"error,omitempty"`
	Differences []Difference         `json:"differences"`
	Observed    []testquality.Result `json:"observed"`
}

type Report struct {
	Version          string       `json:"version"`
	Partition        string       `json:"partition"`
	TotalCases       int          `json:"totalCases"`
	PassedCases      int          `json:"passedCases"`
	FailedCases      int          `json:"failedCases"`
	UnknownCases     int          `json:"unknownCases"`
	FalsePositives   int          `json:"falsePositives"`
	MissedViolations int          `json:"missedViolations"`
	Results          []CaseResult `json:"results"`
}

type Evaluator func(context.Context, Input) ([]testquality.Result, error)

type identity struct {
	rule, version, file, test string
	scope                     string
	line, column              int
}

func (i identity) String() string {
	return fmt.Sprintf("%s/%s:%s:%d:%d:%s:%s", i.rule, i.version, i.file, i.line, i.column, i.test, i.scope)
}
func expectedKey(e Expectation) identity {
	return identity{e.RuleID, e.RuleVersion, e.File, e.TestID, e.Scope, e.Line, e.Column}
}
func observedKey(r testquality.Result) identity {
	return identity{r.RuleID, r.RuleVersion, r.Target.File, r.Target.TestID, r.Target.Scope, r.Location.Line, r.Location.Column}
}

func ValidateCases(cases []Case, partition string) error {
	if partition != "development" && partition != "reviewed-holdout" {
		return fmt.Errorf("unknown calibration partition %q", partition)
	}
	if len(cases) == 0 {
		return fmt.Errorf("empty calibration case set")
	}
	seen := map[string]bool{}
	for _, c := range cases {
		if c.Input.ID == "" || seen[c.Input.ID] {
			return fmt.Errorf("missing or duplicate case id %q", c.Input.ID)
		}
		seen[c.Input.ID] = true
		if c.Partition != partition {
			return fmt.Errorf("case %s belongs to %s, not %s", c.Input.ID, c.Partition, partition)
		}
		if c.Input.Profile == "" || c.Input.Root == "" || len(c.Input.Files) == 0 || (c.Input.Kind != "parser" && c.Input.Kind != "native") || c.Rationale == "" || c.Provenance == "" {
			return fmt.Errorf("case %s lacks input or review provenance", c.Input.ID)
		}
		if len(c.Expected) == 0 {
			return fmt.Errorf("case %s has no positive expectation (unknown must be explicit)", c.Input.ID)
		}
		keys := map[identity]Expectation{}
		for _, e := range c.Expected {
			key := expectedKey(e)
			validScope := (e.Scope == "" && e.TestID != "") || (e.Scope == "file" && e.TestID == "")
			if e.RuleID == "" || e.RuleVersion == "" || e.File == "" || !validScope || e.Line < 0 || e.Column < 0 || testquality.NormalizeStatus(e.Status) != e.Status {
				return fmt.Errorf("case %s has malformed expectation", c.Input.ID)
			}
			if _, exists := keys[key]; exists {
				return fmt.Errorf("case %s has duplicate/contradictory expectation %s", c.Input.ID, key)
			}
			if e.Status == testquality.Unknown && (e.Reason == "" || e.Reason == testquality.ReasonNone) {
				return fmt.Errorf("case %s unknown needs a reason", c.Input.ID)
			}
			keys[key] = e
		}
		for _, f := range c.Forbidden {
			if testquality.NormalizeStatus(f.Status) != f.Status {
				return fmt.Errorf("case %s has malformed forbidden status", c.Input.ID)
			}
			if e, exists := keys[expectedKey(f)]; exists && e.Status == f.Status {
				return fmt.Errorf("case %s requires and forbids the same observation", c.Input.ID)
			}
		}
	}
	return nil
}

func evaluate(ctx context.Context, input Input, evaluator Evaluator) (results []testquality.Result, err error) {
	defer func() {
		if p := recover(); p != nil {
			results = nil
			err = fmt.Errorf("evaluator panic: %v", p)
		}
	}()
	return evaluator(ctx, input)
}

// Run retains every mismatch. Empty observations and evaluator failures cannot
// satisfy an expected unknown or an expected clean result.
func Run(ctx context.Context, cases []Case, partition string, evaluator Evaluator) (Report, error) {
	if err := ValidateCases(cases, partition); err != nil {
		return Report{}, err
	}
	if evaluator == nil {
		return Report{}, fmt.Errorf("evaluator required")
	}
	ordered := append([]Case(nil), cases...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Input.ID < ordered[j].Input.ID })
	report := Report{Version: "quality-calibration/v1", Partition: partition, TotalCases: len(cases), Results: []CaseResult{}}
	for _, c := range ordered {
		row := CaseResult{ID: c.Input.ID, Differences: []Difference{}, Observed: []testquality.Result{}}
		observations, err := evaluate(ctx, c.Input, evaluator)
		if err != nil {
			row.Error = err.Error()
			row.Differences = append(row.Differences, Difference{Kind: "evaluation-error", Detail: err.Error()})
		}
		actual := map[identity]testquality.Result{}
		for _, raw := range observations {
			r := raw.Normalized()
			key := observedKey(r)
			if r.SupportProfile != c.Input.Profile {
				row.Differences = append(row.Differences, Difference{Kind: "profile-mismatch", Identity: key.String(), Detail: fmt.Sprintf("expected %s; observed %s", c.Input.Profile, r.SupportProfile)})
			}
			if _, exists := actual[key]; exists {
				row.Differences = append(row.Differences, Difference{Kind: "duplicate-observation", Identity: key.String()})
			}
			actual[key] = r
			row.Observed = append(row.Observed, r)
			if r.Status == testquality.Unknown {
				row.Unknown = true
			}
		}
		wanted := map[identity]bool{}
		for _, e := range c.Expected {
			key := expectedKey(e)
			wanted[key] = true
			got, exists := actual[key]
			if !exists {
				row.Differences = append(row.Differences, Difference{Kind: "missing-observation", Identity: key.String(), Expected: e.Status})
				if e.Status == testquality.Violation {
					report.MissedViolations++
				}
				continue
			}
			if got.Status != e.Status {
				row.Differences = append(row.Differences, Difference{Kind: "status-mismatch", Identity: key.String(), Expected: e.Status, Observed: got.Status})
				if got.Status == testquality.Violation {
					report.FalsePositives++
				}
				if e.Status == testquality.Violation {
					report.MissedViolations++
				}
			}
			if e.Reason != "" && got.Reason != e.Reason {
				row.Differences = append(row.Differences, Difference{Kind: "reason-mismatch", Identity: key.String(), Detail: fmt.Sprintf("expected %s; observed %s", e.Reason, got.Reason)})
			}
		}
		for key, got := range actual {
			if !wanted[key] {
				row.Differences = append(row.Differences, Difference{Kind: "unexpected-observation", Identity: key.String(), Observed: got.Status})
				if got.Status == testquality.Violation {
					report.FalsePositives++
				}
			}
		}
		for _, f := range c.Forbidden {
			if got, exists := actual[expectedKey(f)]; exists && got.Status == f.Status {
				row.Differences = append(row.Differences, Difference{Kind: "forbidden-observation", Identity: expectedKey(f).String(), Observed: got.Status})
			}
		}
		sort.Slice(row.Observed, func(i, j int) bool {
			return observedKey(row.Observed[i]).String() < observedKey(row.Observed[j]).String()
		})
		sort.Slice(row.Differences, func(i, j int) bool {
			a, b := row.Differences[i], row.Differences[j]
			if a.Identity != b.Identity {
				return a.Identity < b.Identity
			}
			return a.Kind < b.Kind
		})
		row.Passed = len(row.Differences) == 0
		if row.Passed {
			report.PassedCases++
		} else {
			report.FailedCases++
		}
		if row.Unknown {
			report.UnknownCases++
		}
		report.Results = append(report.Results, row)
	}
	return report, nil
}
