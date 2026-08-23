// Package fleet rolls structure-health's per-target engine up across the whole
// fleet into deterministic offender queries: per-target structure rollups,
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

// Lister enumerates the scenarios to scan. It is a seam so tests can inject a
// fixed list without touching the filesystem.
type Lister interface {
	Scenarios() ([]string, error)
}

// TargetLister is implemented by listers that can enumerate every governed
// target kind. Lister remains the compatibility seam for callers that only
// know about scenarios.
type TargetLister interface {
	Targets() ([]Target, error)
}

// Target identifies one structure-health validation subject.
type Target struct {
	Kind string
	ID   string
	Root string
}

// FilesystemLister lists every directory under <RepoRoot>/scenarios that carries
// a .vrooli/service.json (the canonical "this is a scenario" marker).
type FilesystemLister struct {
	RepoRoot string
}

// Scenarios returns the sorted slugs of every discovered scenario.
func (f FilesystemLister) Scenarios() ([]string, error) {
	targets, err := f.Targets()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, target := range targets {
		if target.Kind == "scenario" {
			out = append(out, target.ID)
		}
	}
	sort.Strings(out)
	return out, nil
}

// Targets enumerates all repository target kinds owned by structure-health.
func (f FilesystemLister) Targets() ([]Target, error) {
	var out []Target
	appendChildren := func(kind, parent, marker string) error {
		entries, err := os.ReadDir(filepath.Join(f.RepoRoot, parent))
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			root := filepath.Join(f.RepoRoot, parent, e.Name())
			if marker != "" {
				if _, err := os.Stat(filepath.Join(root, marker)); err != nil {
					continue
				}
			}
			out = append(out, Target{Kind: kind, ID: e.Name(), Root: root})
		}
		return nil
	}
	if err := appendChildren("scenario", "scenarios", filepath.Join(".vrooli", "service.json")); err != nil {
		return nil, err
	}
	if err := appendChildren("resource", "resources", "resource.json"); err != nil {
		return nil, err
	}
	if err := appendChildren("tool", filepath.Join("internal", "tools"), "tool.json"); err != nil {
		return nil, err
	}
	if err := appendChildren("safeguard", filepath.Join("internal", "safeguards"), "safeguard.json"); err != nil {
		return nil, err
	}
	if err := appendChildren("package", "packages", filepath.Join(".vrooli", "package.json")); err != nil {
		return nil, err
	}
	for _, root := range []string{"cmd", "internal"} {
		out = append(out, Target{Kind: "control-plane", ID: root, Root: filepath.Join(f.RepoRoot, root)})
	}
	out = append(out, Target{Kind: "docs", ID: "docs", Root: filepath.Join(f.RepoRoot, "docs")})
	if err := appendChildren("team", "docs", "manifest.json"); err != nil {
		return nil, err
	}
	out = append(out, Target{Kind: "project", ID: "repo", Root: f.RepoRoot})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
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
	TargetKind        string
	TargetID          string
	TargetRoot        string
	Passed            bool
	ProfileID         string
	ProfileRecognized bool
	ErrorCount        int
	WarningCount      int
	TotalFindings     int
	AutofixableCount  int
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
	AutofixableTotal    int
	Errors              []ScanError
	TargetCount         int
	PassingTargetCount  int
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
		if lister, ok := s.Lister.(TargetLister); ok {
			targets, err := lister.Targets()
			if err != nil {
				return Result{}, err
			}
			return s.scanTargets(ctx, targets)
		}
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
	targets := make([]Target, 0, len(scenarios))
	for _, scenario := range scenarios {
		targets = append(targets, Target{Kind: "scenario", ID: scenario})
	}
	return s.scanTargets(ctx, targets)
}

// ScanTargets grades an explicit set of typed targets.
func (s *Scanner) ScanTargets(ctx context.Context, targets []Target) (Result, error) {
	if s == nil || s.Engine == nil {
		return Result{}, errors.New("fleet: scanner not wired")
	}
	return s.scanTargets(ctx, append([]Target(nil), targets...))
}

func (s *Scanner) scanTargets(ctx context.Context, targets []Target) (Result, error) {
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		return targets[i].ID < targets[j].ID
	})

	var result Result
	ruleAcc := map[string]*RuleConformance{}
	profileAcc := map[string]*ProfileCount{}

	for _, target := range targets {
		name := target.ID
		resp, err := s.Engine.Validate(ctx, validation.Request{Scenario: name, TargetKind: target.Kind, TargetID: target.ID, TargetRoot: target.Root, Path: target.Root})
		if err != nil {
			result.Errors = append(result.Errors, ScanError{Scenario: targetLabel(target), Reason: err.Error()})
			continue
		}
		entry := rollup(target, resp)
		result.Entries = append(result.Entries, entry)
		result.ScenarioCount++
		result.TargetCount++
		if entry.Passed {
			result.PassingCount++
			result.PassingTargetCount++
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

func targetLabel(target Target) string {
	if target.Kind == "scenario" || target.Kind == "" {
		return target.ID
	}
	return target.Kind + ":" + target.ID
}

func normalizeSeverity(sev string) string {
	if sev == "warn" {
		return "warning"
	}
	return sev
}

func rollup(target Target, resp validation.Response) ScenarioEntry {
	entry := ScenarioEntry{
		Scenario:          target.ID,
		TargetKind:        target.Kind,
		TargetID:          target.ID,
		TargetRoot:        target.Root,
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
