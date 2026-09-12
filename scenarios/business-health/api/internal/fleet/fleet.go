// Package fleet is the compute-on-read business-contract debt sweep:
// every discovered scenario graded through the same check engine that
// backs per-scenario validation, rolled up worst-first with as-of stamps
// (test-genie fleet-status honesty rules — nothing cached is presented
// as current).
package fleet

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"business-health/internal/checks"

	intent "intent-go"
)

// Entry is one scenario's business-contract rollup.
type Entry struct {
	Scenario         string
	Passed           bool
	ErrorCount       int
	WarningCount     int
	TotalFindings    int
	AutofixableCount int
	StarterRegistry  bool
	TemplateVersion  string
	TemplateLaggard  bool
	OrphanedTargets  int
	UnprovenClaims   int
	DebtScore        int
	DegradedReason   string
}

// ScanError records a scenario that was enumerated but could not be graded.
type ScanError struct {
	Scenario string
	Reason   string
}

// Result is the fleet sweep output.
type Result struct {
	Entries              []Entry
	Errors               []ScanError
	AsOf                 time.Time
	ScenarioCount        int
	PassingCount         int
	StarterRegistryCount int
	TemplateLaggardCount int
}

// Engine is the per-scenario validation seam (implemented by
// checks.Engine).
type Engine interface {
	ValidateScenario(ctx context.Context, scenario, path string) (checks.Report, error)
}

// Sweeper computes fleet rollups.
type Sweeper struct {
	repoRoot string
	engine   Engine
	// autofixable is the base-code set with registered fixers.
	autofixable map[string]struct{}
	now         func() time.Time
}

// NewSweeper builds a Sweeper. autofixRules is the fixer rule-id list
// (base codes); nil now means time.Now.
func NewSweeper(repoRoot string, engine Engine, autofixRules []string, now func() time.Time) *Sweeper {
	rules := make(map[string]struct{}, len(autofixRules))
	for _, r := range autofixRules {
		rules[r] = struct{}{}
	}
	if now == nil {
		now = time.Now
	}
	return &Sweeper{repoRoot: repoRoot, engine: engine, autofixable: rules, now: now}
}

// Scan grades every discovered scenario (or the requested subset),
// worst-first by debt score.
func (s *Sweeper) Scan(ctx context.Context, scenarios []string) (Result, error) {
	res := Result{AsOf: s.now().UTC()}
	slugs := scenarios
	if len(slugs) == 0 {
		discovered, err := s.discover()
		if err != nil {
			return res, err
		}
		slugs = discovered
	}
	currentVersions := s.templateVersions()

	for _, slug := range slugs {
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}
		entry, err := s.grade(ctx, slug, currentVersions)
		if err != nil {
			res.Errors = append(res.Errors, ScanError{Scenario: slug, Reason: err.Error()})
			continue
		}
		res.Entries = append(res.Entries, entry)
	}

	sort.SliceStable(res.Entries, func(i, j int) bool {
		if res.Entries[i].DebtScore != res.Entries[j].DebtScore {
			return res.Entries[i].DebtScore > res.Entries[j].DebtScore
		}
		return res.Entries[i].Scenario < res.Entries[j].Scenario
	})
	res.ScenarioCount = len(res.Entries)
	for _, e := range res.Entries {
		if e.Passed {
			res.PassingCount++
		}
		if e.StarterRegistry {
			res.StarterRegistryCount++
		}
		if e.TemplateLaggard {
			res.TemplateLaggardCount++
		}
	}
	return res, nil
}

// discover lists scenario slugs: directories under scenarios/ carrying
// .vrooli/service.json.
func (s *Sweeper) discover() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.repoRoot, "scenarios"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(s.repoRoot, "scenarios", entry.Name(), ".vrooli", "service.json")); err == nil {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *Sweeper) grade(ctx context.Context, slug string, currentVersions map[string]string) (Entry, error) {
	report, err := s.engine.ValidateScenario(ctx, slug, "")
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{Scenario: slug, DegradedReason: report.DegradedReason}
	for _, f := range report.Findings {
		entry.TotalFindings++
		base := baseCode(f.Code)
		switch f.Severity {
		case "error", "SEVERITY_ERROR":
			entry.ErrorCount++
		case "warning", "SEVERITY_WARNING":
			entry.WarningCount++
		}
		if _, ok := s.autofixable[base]; ok {
			entry.AutofixableCount++
		}
		switch base {
		case "business_starter_template":
			entry.StarterRegistry = true
		case intent.CodeOTOrphan:
			entry.OrphanedTargets++
		case "business_unproven_claim", "business_status_unearned":
			entry.UnprovenClaims++
		}
	}
	entry.Passed = entry.ErrorCount == 0

	templateID, version := generationStamp(filepath.Join(s.repoRoot, "scenarios", slug, ".vrooli", "service.json"))
	entry.TemplateVersion = version
	if current, ok := currentVersions[templateID]; ok && version != "" && version != current {
		entry.TemplateLaggard = true
	}

	entry.DebtScore = entry.ErrorCount*10 + entry.WarningCount*2 +
		entry.OrphanedTargets*3 + entry.UnprovenClaims*5 +
		boolScore(entry.StarterRegistry, 20) + boolScore(entry.TemplateLaggard, 10)
	return entry, nil
}

// templateVersions reads the current contract version per template id.
func (s *Sweeper) templateVersions() map[string]string {
	out := map[string]string{}
	entries, err := os.ReadDir(filepath.Join(s.repoRoot, "templates", "scenarios"))
	if err != nil {
		return out
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.repoRoot, "templates", "scenarios", entry.Name(), "template.json"))
		if err != nil {
			continue
		}
		var t struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &t) == nil && t.Version != "" {
			out[entry.Name()] = t.Version
		}
	}
	return out
}

func generationStamp(servicePath string) (templateID, version string) {
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return "", ""
	}
	var svc struct {
		Generation struct {
			Template struct {
				ID      string `json:"id"`
				Version string `json:"version"`
			} `json:"template"`
		} `json:"generation"`
	}
	if json.Unmarshal(data, &svc) != nil {
		return "", ""
	}
	return svc.Generation.Template.ID, svc.Generation.Template.Version
}

func baseCode(code string) string {
	if i := strings.IndexByte(code, ':'); i > 0 {
		return code[:i]
	}
	return code
}

func boolScore(b bool, weight int) int {
	if b {
		return weight
	}
	return 0
}
