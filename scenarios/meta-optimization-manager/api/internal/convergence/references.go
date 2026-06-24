package convergence

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"meta-optimization-manager/internal/clock"

	repocontract "github.com/vrooli/repo-contract-go"
)

// ReferenceScanner computes gold-star reference-scenario health/eligibility from
// the REFERENCE_SCENARIOS.md registry. Production parses the registry, derives
// stability from the Generated date, breadth from the reference's domain layout,
// and clean-on-all-tools via a soft scenario-auditor read; tests fake it. The
// precise stale-from-template (git last-commit vs Generated) comparison is a
// documented refinement seam — surfaced as a conservative false until wired.
type ReferenceScanner interface {
	Scan(ctx context.Context) ([]ReferenceHealth, error)
}

type fsReferenceScanner struct {
	root  string
	clock clock.Clock
	run   CommandRunner
}

// NewReferenceScanner returns the production ReferenceScanner.
func NewReferenceScanner(clk clock.Clock) ReferenceScanner {
	return &fsReferenceScanner{clock: clk, run: execRunner}
}

// NewReferenceScannerWithDeps returns a scanner with injected root/clock/runner
// (tests point root at a fixture and fake the runner).
func NewReferenceScannerWithDeps(root string, clk clock.Clock, run CommandRunner) ReferenceScanner {
	return &fsReferenceScanner{root: root, clock: clk, run: run}
}

var _ ReferenceScanner = (*fsReferenceScanner)(nil)

// referenceRow matches a markdown table row in REFERENCE_SCENARIOS.md, capturing
// the scenario name (backticked) and the Generated date column.
var (
	backtickName = regexp.MustCompile("`([a-z0-9][a-z0-9-]+)`")
	isoDate      = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
)

func (s *fsReferenceScanner) Scan(ctx context.Context) ([]ReferenceHealth, error) {
	root := s.root
	if root == "" {
		r, err := repocontract.FindRepoRootFromEnvOrCWD()
		if err != nil {
			return nil, err
		}
		root = r
	}
	regPath := filepath.Join(root, "docs", "agent-system", "REFERENCE_SCENARIOS.md")
	refs, err := parseReferenceRegistry(regPath)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now().UTC()
	out := make([]ReferenceHealth, 0, len(refs))
	for _, r := range refs {
		h := ReferenceHealth{Scenario: r.scenario}
		if !r.generated.IsZero() {
			h.LastTemplateSync = r.generated
			days := int(now.Sub(r.generated).Hours() / 24)
			if days < 0 {
				days = 0
			}
			h.StabilityDays = days
		}
		h.Breadth = breadthOf(root, r.scenario)
		h.CleanOnAllTools = s.cleanOnAllTools(ctx, r.scenario)
		// stale-from-template: documented refinement seam (git last-commit vs
		// Generated). Conservative false until wired — never a false positive.
		h.StaleFromTemplate = false
		h.Eligibility = deriveEligibility(h)
		out = append(out, h)
	}
	return out, nil
}

type refRow struct {
	scenario  string
	generated time.Time
}

// parseReferenceRegistry extracts reference rows from the markdown table. Rows
// without a backticked scenario name, or whose name is a placeholder, are
// skipped. The first backticked token is the scenario; the first ISO date is the
// Generated column.
func parseReferenceRegistry(path string) ([]refRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []refRow
	seen := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		// Only data rows that name a reference scenario (Role column is bold prose).
		if !strings.Contains(line, "reference") && !strings.Contains(line, "Gold-star") && !strings.Contains(line, "Secondary") {
			// not a reference data row
		}
		m := backtickName.FindAllStringSubmatch(line, -1)
		if len(m) == 0 {
			continue
		}
		name := m[0][1]
		// Skip the template path token and placeholders.
		if name == "" || seen[name] || looksLikeTemplate(line, name) {
			continue
		}
		seen[name] = true
		row := refRow{scenario: name}
		if dm := isoDate.FindString(line); dm != "" {
			if t, perr := time.Parse("2006-01-02", dm); perr == nil {
				row.generated = t.UTC()
			}
		}
		out = append(out, row)
	}
	return out, sc.Err()
}

// looksLikeTemplate skips rows whose first backticked token is actually a
// template path (the registry pairs scenario + template; we want the scenario).
func looksLikeTemplate(line, name string) bool {
	// A reference scenario name is the first backticked token in the row; the
	// template token is prefixed by "path:" in this registry. If the first token
	// is the template, the scenario column was empty (placeholder rows).
	idx := strings.Index(line, "`"+name+"`")
	if idx <= 0 {
		return false
	}
	return strings.Contains(line[:idx], "path:") && strings.HasSuffix(name, "vite") && strings.Contains(line[:idx], "templates")
}

// breadthOf counts the patterns a reference demonstrates: its API domain dirs
// (handlers/* or api/internal/* leaf domains). Zero when the scenario dir is
// absent (an honest "cannot measure breadth").
func breadthOf(root, scenario string) int {
	scenarioRoot, err := repocontract.ResolveScenarioPath(root, scenario)
	if err != nil {
		return 0
	}
	count := 0
	for _, sub := range []string{filepath.Join("api", "handlers"), filepath.Join("api", "internal")} {
		entries, err := os.ReadDir(filepath.Join(scenarioRoot, sub))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				count++
			}
		}
	}
	return count
}

// cleanOnAllTools is a soft read of scenario-auditor for the reference. A nil
// runner or any error yields false (honest "not confirmed clean"), never a
// fabricated pass.
func (s *fsReferenceScanner) cleanOnAllTools(ctx context.Context, scenario string) bool {
	if s.run == nil {
		return false
	}
	out, err := s.run(ctx, "scenario-auditor", "audit", scenario, "--json")
	if err != nil {
		return false
	}
	low := strings.ToLower(string(out))
	// Conservative: clean only when the auditor explicitly reports no violations.
	return strings.Contains(low, "\"violations\": 0") ||
		strings.Contains(low, "\"violations\":0") ||
		strings.Contains(low, "\"clean\": true") ||
		strings.Contains(low, "\"clean\":true")
}
