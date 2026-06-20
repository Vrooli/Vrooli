// Package fleet rolls structure-health's per-scenario engine up across the whole
// fleet into deterministic offender queries: per-scenario structure rollups,
// profile/surface distributions, per-rule conformance, and auto-fixable
// coverage. The queries are exact and structured (no semantic search), so this
// is intentionally NOT a search-hub data provider.
package fleet

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"structure-health/internal/validation"
)

// freshnessFindingCode marks a missing buildable-surface freshness check — the
// silent-rebuild offender signal the fleet view surfaces.
const freshnessFindingCode = "FRESHNESS_CHECK_MISSING"

// Lister enumerates the scenarios to scan. It is a seam so tests can inject a
// fixed list without touching the filesystem.
type Lister interface {
	Scenarios() ([]string, error)
}

// FilesystemLister lists every directory under <RepoRoot>/scenarios that carries
// a .vrooli/service.json (the canonical "this is a scenario" marker).
type FilesystemLister struct {
	RepoRoot string
}

// Scenarios returns the sorted slugs of every discovered scenario.
func (f FilesystemLister) Scenarios() ([]string, error) {
	root := filepath.Join(f.RepoRoot, "scenarios")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, statErr := os.Stat(filepath.Join(root, e.Name(), ".vrooli", "service.json")); statErr != nil {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}

// Engine is the per-scenario validation seam the scanner grades each scenario
// through; *validation.Service satisfies it.
type Engine interface {
	Validate(ctx context.Context, req validation.Request) (validation.Response, error)
}

// Scanner grades scenarios and aggregates the structure rollup.
type Scanner struct {
	Engine Engine
	Lister Lister
}

// New builds a Scanner.
func New(engine Engine, lister Lister) *Scanner {
	return &Scanner{Engine: engine, Lister: lister}
}

// ScenarioEntry is one scenario's structure rollup.
type ScenarioEntry struct {
	Scenario          string
	Passed            bool
	ProfileID         string
	ProfileRecognized bool
	ErrorCount        int
	WarningCount      int
	TotalFindings     int
	AutofixableCount  int
	MissingFreshness  bool
	Surfaces          []string
	DegradedReason    string
}

// RuleConformance rolls one finding code up across the fleet.
type RuleConformance struct {
	Code               string
	OffendingScenarios int
	TotalFindings      int
	Autofixable        int
	WorstSeverity      string
}

// ProfileCount counts scenarios per detected profile.
type ProfileCount struct {
	ProfileID     string
	ScenarioCount int
	Recognized    bool
}

// ScanError records a scenario that was enumerated but could not be graded.
type ScanError struct {
	Scenario string
	Reason   string
}

// Result is the aggregated fleet rollup.
type Result struct {
	Entries             []ScenarioEntry
	RuleConformance     []RuleConformance
	ProfileDistribution []ProfileCount
	ScenarioCount       int
	PassingCount        int
	MissingFreshness    int
	AutofixableTotal    int
	Errors              []ScanError
}

// severityRank orders severities so the worst can be tracked.
func severityRank(sev string) int {
	switch sev {
	case "error":
		return 3
	case "warning", "warn":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// Scan grades the requested scenarios (or every discovered scenario when the
// list is empty) and returns the aggregated rollup.
func (s *Scanner) Scan(ctx context.Context, scenarios []string) (Result, error) {
	if s == nil || s.Engine == nil {
		return Result{}, errors.New("fleet: scanner not wired")
	}
	if len(scenarios) == 0 {
		if s.Lister == nil {
			return Result{}, errors.New("fleet: no scenario lister wired")
		}
		all, err := s.Lister.Scenarios()
		if err != nil {
			return Result{}, err
		}
		scenarios = all
	} else {
		scenarios = append([]string(nil), scenarios...)
	}
	sort.Strings(scenarios)

	var result Result
	ruleAcc := map[string]*RuleConformance{}
	profileAcc := map[string]*ProfileCount{}

	for _, name := range scenarios {
		resp, err := s.Engine.Validate(ctx, validation.Request{Scenario: name})
		if err != nil {
			result.Errors = append(result.Errors, ScanError{Scenario: name, Reason: err.Error()})
			continue
		}
		entry := rollup(name, resp)
		result.Entries = append(result.Entries, entry)
		result.ScenarioCount++
		if entry.Passed {
			result.PassingCount++
		}
		if entry.MissingFreshness {
			result.MissingFreshness++
		}
		result.AutofixableTotal += entry.AutofixableCount

		if pc, ok := profileAcc[entry.ProfileID]; ok {
			pc.ScenarioCount++
		} else {
			profileAcc[entry.ProfileID] = &ProfileCount{ProfileID: entry.ProfileID, ScenarioCount: 1, Recognized: entry.ProfileRecognized}
		}

		seenCodes := map[string]bool{}
		for _, f := range resp.Findings {
			rc, ok := ruleAcc[f.Code]
			if !ok {
				rc = &RuleConformance{Code: f.Code}
				ruleAcc[f.Code] = rc
			}
			rc.TotalFindings++
			if f.AutofixAvailable {
				rc.Autofixable++
			}
			if severityRank(f.Severity) > severityRank(rc.WorstSeverity) {
				rc.WorstSeverity = normalizeSeverity(f.Severity)
			}
			if !seenCodes[f.Code] {
				seenCodes[f.Code] = true
				rc.OffendingScenarios++
			}
		}
	}

	for _, rc := range ruleAcc {
		result.RuleConformance = append(result.RuleConformance, *rc)
	}
	sort.Slice(result.RuleConformance, func(i, j int) bool {
		a, b := result.RuleConformance[i], result.RuleConformance[j]
		if a.OffendingScenarios != b.OffendingScenarios {
			return a.OffendingScenarios > b.OffendingScenarios
		}
		return a.Code < b.Code
	})

	for _, pc := range profileAcc {
		result.ProfileDistribution = append(result.ProfileDistribution, *pc)
	}
	sort.Slice(result.ProfileDistribution, func(i, j int) bool {
		a, b := result.ProfileDistribution[i], result.ProfileDistribution[j]
		if a.ScenarioCount != b.ScenarioCount {
			return a.ScenarioCount > b.ScenarioCount
		}
		return a.ProfileID < b.ProfileID
	})

	return result, nil
}

func normalizeSeverity(sev string) string {
	if sev == "warn" {
		return "warning"
	}
	return sev
}

func rollup(name string, resp validation.Response) ScenarioEntry {
	entry := ScenarioEntry{
		Scenario:          name,
		ProfileID:         resp.Profile.ID,
		ProfileRecognized: resp.Profile.Recognized,
		TotalFindings:     len(resp.Findings),
		DegradedReason:    resp.DegradedReason,
	}
	for _, f := range resp.Findings {
		switch f.Severity {
		case "error":
			entry.ErrorCount++
		case "warning", "warn":
			entry.WarningCount++
		}
		if f.AutofixAvailable {
			entry.AutofixableCount++
		}
		if f.Code == freshnessFindingCode {
			entry.MissingFreshness = true
		}
	}
	entry.Passed = entry.ErrorCount == 0
	for _, sr := range resp.Surfaces {
		if sr.Declared {
			entry.Surfaces = append(entry.Surfaces, sr.Surface)
		}
	}
	sort.Strings(entry.Surfaces)
	return entry
}
