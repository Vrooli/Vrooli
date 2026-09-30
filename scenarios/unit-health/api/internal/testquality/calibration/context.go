package calibration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ContextCase captures non-source inputs for owner-adapter and behavioral
// reference checks. Its outcomes are not test-quality rule statuses: e.g. an
// invalid mutation experiment is neither a passing test nor a clean rule check.
type ContextCase struct {
	ID               string          `json:"id"`
	Group            string          `json:"group"`
	Partition        string          `json:"partition"`
	Provenance       string          `json:"provenance"`
	Rationale        string          `json:"rationale"`
	Input            json.RawMessage `json:"input"`
	ExpectedOutcome  string          `json:"expectedOutcome"`
	RequiresEvidence bool            `json:"requiresEvidence"`
}

type ContextObservation struct {
	ID            string   `json:"id"`
	Outcome       string   `json:"outcome"`
	EvidenceRefs  []string `json:"evidenceRefs"`
	UnknownReason string   `json:"unknownReason,omitempty"`
	Error         string   `json:"error,omitempty"`
}

type ContextCaseResult struct {
	ID          string              `json:"id"`
	Matched     bool                `json:"matched"`
	Observation *ContextObservation `json:"observation,omitempty"`
	Differences []string            `json:"differences"`
}
type ContextReport struct {
	TotalCases    int                 `json:"totalCases"`
	MatchedCases  int                 `json:"matchedCases"`
	UnknownCases  int                 `json:"unknownCases"`
	Results       []ContextCaseResult `json:"results"`
	UnexpectedIDs []string            `json:"unexpectedIds"`
}

func LoadContextCases(root string) ([]ContextCase, error) {
	data, err := os.ReadFile(filepath.Join(root, "context-cases.json"))
	if err != nil {
		return nil, err
	}
	var cases []ContextCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("empty context fixture set")
	}
	seen := map[string]bool{}
	for _, c := range cases {
		if c.ID == "" || seen[c.ID] || c.Group == "" || c.Provenance == "" || c.Rationale == "" || c.Partition != "development" || c.ExpectedOutcome == "" || !c.RequiresEvidence {
			return nil, fmt.Errorf("invalid context fixture %q", c.ID)
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(c.Input, &input); err != nil || len(input) == 0 {
			return nil, fmt.Errorf("case %s lacks structured input", c.ID)
		}
		seen[c.ID] = true
	}
	return cases, nil
}

// CompareContext consumes owner-produced observations, never deriving an
// outcome from the expectation. Missing evidence remains a failed comparison,
// even when the case intentionally expects an unknown owner response.
func CompareContext(cases []ContextCase, observations []ContextObservation) ContextReport {
	ordered := append([]ContextCase(nil), cases...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	report := ContextReport{TotalCases: len(cases), Results: []ContextCaseResult{}, UnexpectedIDs: []string{}}
	wanted := map[string]bool{}
	for _, c := range ordered {
		wanted[c.ID] = true
		row := ContextCaseResult{ID: c.ID, Differences: []string{}}
		matches := 0
		for _, observed := range observations {
			if observed.ID != c.ID {
				continue
			}
			matches++
			copy := observed
			copy.EvidenceRefs = append([]string(nil), observed.EvidenceRefs...)
			row.Observation = &copy
			if observed.Error != "" {
				row.Differences = append(row.Differences, "owner evaluation error: "+observed.Error)
			}
			if observed.Outcome != c.ExpectedOutcome {
				row.Differences = append(row.Differences, fmt.Sprintf("expected %s; observed %s", c.ExpectedOutcome, observed.Outcome))
			}
			hasEvidence := false
			for _, ref := range observed.EvidenceRefs {
				if ref != "" {
					hasEvidence = true
				}
			}
			if !hasEvidence {
				row.Differences = append(row.Differences, "missing owner evidence reference")
			}
			if observed.Outcome == "unknown" && observed.UnknownReason == "" {
				row.Differences = append(row.Differences, "unknown outcome lacks reason")
			}
		}
		if matches != 1 {
			row.Differences = append(row.Differences, fmt.Sprintf("expected one owner observation; got %d", matches))
		}
		if row.Observation != nil && row.Observation.Outcome == "unknown" {
			report.UnknownCases++
		}
		row.Matched = len(row.Differences) == 0
		if row.Matched {
			report.MatchedCases++
		}
		report.Results = append(report.Results, row)
	}
	for _, observed := range observations {
		if !wanted[observed.ID] {
			report.UnexpectedIDs = append(report.UnexpectedIDs, observed.ID)
		}
	}
	sort.Strings(report.UnexpectedIDs)
	return report
}
