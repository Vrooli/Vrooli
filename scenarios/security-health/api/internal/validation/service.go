package validation

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
)

// Service validates one scenario at a time. It owns the detector → scanner
// pipeline and the severity contract; the Connect handler and the CLI are thin
// translation layers over ValidateScenario.
type Service struct {
	repoRoot string
	cmd      Commander
	scanners []Scanner
	logger   *log.Logger
}

// Deps wires the Service's seams. Commander and Scanners default to the real
// implementations when nil/empty, so production callers pass only repoRoot.
type Deps struct {
	RepoRoot  string
	Commander Commander
	Scanners  []Scanner
	Logger    *log.Logger
}

// New constructs a Service. The scanner set defaults to DefaultScanners(cmd).
func New(d Deps) *Service {
	cmd := d.Commander
	if cmd == nil {
		cmd = NewExecCommander()
	}
	scanners := d.Scanners
	if len(scanners) == 0 {
		scanners = DefaultScanners(cmd)
	}
	logger := d.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Service{repoRoot: d.RepoRoot, cmd: cmd, scanners: scanners, logger: logger}
}

// DefaultScanners returns the v1 scanner set in stable order. gitleaks +
// gosec + pnpm-audit are the always-available trio; govulncheck + osv-scanner
// are install-gated and degrade to skipped when absent.
func DefaultScanners(cmd Commander) []Scanner {
	return []Scanner{
		newGitleaksScanner(cmd),
		newGosecScanner(cmd),
		newGovulncheckScanner(cmd),
		newPnpmAuditScanner(cmd),
		newOSVScanner(cmd),
	}
}

// ValidateScenario detects the scenario's substrates, runs every applicable
// scanner whose binary is present, and aggregates normalized findings into a
// Report. A scanner that applies but is absent is recorded in
// SkippedScanners and surfaced as an INFO finding (degraded, not failed). A
// scanner that errors mid-run is likewise downgraded to an INFO observation so
// one flaky tool can't spuriously gate a scenario.
func (s *Service) ValidateScenario(ctx context.Context, scenario string) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Report{}, fmt.Errorf("scenario name is required")
	}
	scenarioDir, ok := resolveScenarioDir(s.repoRoot, scenario)
	if !ok {
		// Mirror the cli-health sibling: a missing target is a graceful skip
		// (WARNING, passed=true), not a hard error. This keeps the test-genie
		// `security` phase non-failing when pointed at a scenario that doesn't
		// exist on disk (e.g. a synthetic orchestrator fixture) rather than
		// crashing the whole suite.
		return finalize(scenario, []Finding{{
			RuleID:      "security-health.scenario-not-found",
			Severity:    SeverityWarning,
			Title:       fmt.Sprintf("Scenario %q not found under scenarios/", scenario),
			Description: fmt.Sprintf("No directory scenarios/%s exists; there is nothing to scan.", scenario),
			Remediation: "Check the scenario id, or generate it via `vrooli scenario generate`.",
			Scanner:     "security-health",
		}}, nil), nil
	}

	sub, err := DetectSubstrate(scenarioDir)
	if err != nil {
		return Report{}, fmt.Errorf("detect substrate for %q: %w", scenario, err)
	}

	var findings []Finding
	var skipped []string

	for _, sc := range s.scanners {
		if !sc.Applies(sub) {
			continue
		}
		if _, lookErr := s.cmd.LookPath(sc.Binary()); lookErr != nil {
			skipped = append(skipped, sc.Name())
			findings = append(findings, Finding{
				RuleID:      "security-health.scanner-absent",
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf("Scanner %q not installed", sc.Name()),
				Description: fmt.Sprintf("The %q scanner applies to this scenario's substrate but its binary (%q) is not on PATH, so this class of issue was not checked.", sc.Name(), sc.Binary()),
				Remediation: fmt.Sprintf("Install %q to enable this check (see docs/concepts/INTEGRATIONS.md). Until then this is informational, not a failure.", sc.Name()),
				FilePath:    "",
				Scanner:     sc.Name(),
			})
			continue
		}
		scanFindings, scanErr := sc.Scan(ctx, scenarioDir, sub)
		if scanErr != nil {
			s.logger.Printf("[security-health] scanner %q degraded for %q: %v", sc.Name(), scenario, scanErr)
			findings = append(findings, Finding{
				RuleID:      "security-health.scanner-degraded",
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf("Scanner %q could not complete", sc.Name()),
				Description: fmt.Sprintf("%q is installed but did not produce a parseable result: %v", sc.Name(), scanErr),
				Remediation: fmt.Sprintf("Run %q against this scenario by hand to diagnose; until it completes this check is informational, not a failure.", sc.Name()),
				FilePath:    "",
				Scanner:     sc.Name(),
			})
			continue
		}
		findings = append(findings, scanFindings...)
	}

	// Record unsupported substrates as INFO context (never a failure).
	for _, u := range sub.Unsupported {
		findings = append(findings, Finding{
			RuleID:      "security-health.substrate-unsupported",
			Severity:    SeverityInfo,
			Title:       fmt.Sprintf("Unsupported substrate: %s", u),
			Description: fmt.Sprintf("Detected a %s substrate, which security-health does not scan in v1.", u),
			Remediation: fmt.Sprintf("Track %s scanner support as a future enhancement; no action required today.", u),
			Scanner:     "security-health",
		})
	}

	sortFindings(findings)
	sort.Strings(skipped)
	return finalize(scenario, findings, dedupeStrings(skipped)), nil
}

// sortFindings orders findings deterministically: severity (ERROR→WARNING→INFO),
// then scanner, then rule id, then file. Stable output keeps CLI --json and
// fixtures diff-clean.
func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].Severity != f[j].Severity {
			return f[i].Severity < f[j].Severity // ERROR(1) < WARNING(2) < INFO(3)
		}
		if f[i].Scanner != f[j].Scanner {
			return f[i].Scanner < f[j].Scanner
		}
		if f[i].RuleID != f[j].RuleID {
			return f[i].RuleID < f[j].RuleID
		}
		return f[i].FilePath < f[j].FilePath
	})
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
