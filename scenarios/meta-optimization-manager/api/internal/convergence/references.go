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

// ReferenceScanner computes gold-star generated-golden health/eligibility from
// the REFERENCE_SCENARIOS.md registry. Production parses the registry, derives
// stability from the first-registration date, breadth from the backing template
// layout, and clean-on-all-tools via a soft scenario-auditor read; tests fake
// it. The precise stale-from-template comparison is a documented refinement
// seam — surfaced as a conservative false until wired.
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
	backtickAny  = regexp.MustCompile("`([^`]+)`")
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
		h.Breadth = breadthOf(root, r.templatePath)
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
	scenario     string
	templatePath string
	generated    time.Time
}

// parseReferenceRegistry extracts generated-golden rows from the markdown
// table. Rows without a backticked golden slug, or whose name is a placeholder,
// are skipped. The first non-path backticked token is the golden slug, the first
// path: token is the source template, and the first ISO date is the first
// registration date.
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
		// Only data rows that name a generated golden.
		if !strings.Contains(line, "reference") && !strings.Contains(line, "Gold-star") && !strings.Contains(line, "Secondary") {
			continue
		}
		name := ""
		if m := backtickName.FindStringSubmatch(line); len(m) > 1 {
			name = m[1]
		}
		if name == "" {
			continue
		}
		templatePath := rowTemplatePath(line)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		row := refRow{scenario: name, templatePath: templatePath}
		if dm := isoDate.FindString(line); dm != "" {
			if t, perr := time.Parse("2006-01-02", dm); perr == nil {
				row.generated = t.UTC()
			}
		}
		out = append(out, row)
	}
	return out, sc.Err()
}

func rowTemplatePath(line string) string {
	for _, m := range backtickAny.FindAllStringSubmatch(line, -1) {
		token := strings.TrimSpace(m[1])
		if strings.HasPrefix(token, "path:") {
			return strings.TrimPrefix(token, "path:")
		}
	}
	return ""
}

// breadthOf counts the patterns a generated golden's template demonstrates:
// API domain dirs under handlers/* or api/internal/*. Zero when the registry
// lacks a template path or the path is absent.
func breadthOf(root, templatePath string) int {
	if strings.TrimSpace(templatePath) == "" {
		return 0
	}
	templateRoot := filepath.Join(root, filepath.FromSlash(strings.Trim(strings.TrimPrefix(templatePath, "path:"), "/")))
	count := 0
	for _, sub := range []string{filepath.Join("api", "handlers"), filepath.Join("api", "internal")} {
		entries, err := os.ReadDir(filepath.Join(templateRoot, sub))
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
