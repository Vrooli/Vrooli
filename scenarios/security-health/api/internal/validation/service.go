package validation

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// Service validates one scenario at a time. It owns the detector → scanner
// pipeline and the severity contract; the Connect handler and the CLI are thin
// translation layers over ValidateScenario.
type Service struct {
	repoRoot string
	cmd      Commander
	scanners []Scanner
	logger   *log.Logger
	policy   RolloutProfile
	facts    FactDiscovery
	evidence *EvidenceCoordinator
	clock    func() time.Time
}

// Deps wires the Service's seams. Commander and Scanners default to the real
// implementations when nil/empty, so production callers pass only repoRoot.
type Deps struct {
	RepoRoot  string
	Commander Commander
	Scanners  []Scanner
	Logger    *log.Logger
	// FactDiscovery is the preferred Code Facts-backed substrate source. When
	// nil, the bounded filesystem detector remains the explicit fallback.
	FactDiscovery FactDiscovery
	// PolicyMode defaults to advisory. Required callers opt into guarded or
	// enforcing behavior explicitly; this keeps local development usable while
	// making CI/release coverage fail closed.
	PolicyMode RolloutProfile
	// EvidenceCoordinator is shared across requests and scanners. When nil, New
	// installs an in-memory admission coordinator; scanners never bypass it.
	EvidenceCoordinator *EvidenceCoordinator
	Clock               func() time.Time
	// OSVReportCache shares raw advisory evidence with dependency reconciliation.
	OSVReportCache OSVReportCache
}

// New constructs a Service. The scanner set defaults to DefaultScanners(cmd).
func New(d Deps) *Service {
	cmd := d.Commander
	if cmd == nil {
		cmd = NewExecCommander()
	}
	scanners := d.Scanners
	if len(scanners) == 0 {
		scanners = defaultScanners(cmd, d.OSVReportCache, d.Clock)
	}
	logger := d.Logger
	if logger == nil {
		logger = log.Default()
	}
	policy := d.PolicyMode
	if policy == "" {
		policy = RolloutAdvisory
	}
	clock := d.Clock
	if clock == nil {
		clock = time.Now
	}
	evidence := d.EvidenceCoordinator
	if evidence == nil {
		evidence = NewEvidenceCoordinator(EvidenceCoordinatorDeps{Clock: clock, Capacity: 4})
	}
	return &Service{repoRoot: d.RepoRoot, cmd: cmd, scanners: scanners, logger: logger, policy: policy, facts: d.FactDiscovery, evidence: evidence, clock: clock}
}

// DefaultScanners returns the v1 scanner set in stable order. gitleaks +
// gosec + pnpm-audit are the always-available trio; govulncheck + osv-scanner
// are install-gated and degrade to skipped when absent.
func DefaultScanners(cmd Commander) []Scanner {
	return defaultScanners(cmd, nil, nil)
}

func defaultScanners(cmd Commander, osvCache OSVReportCache, clock func() time.Time) []Scanner {
	return []Scanner{
		newGitleaksScanner(cmd),
		newGosecScanner(cmd),
		newGovulncheckScanner(cmd),
		newPnpmAuditScanner(cmd),
		newOSVScannerWithCache(cmd, osvCache, clock),
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

	collector := metricsFrom(ctx)

	detect := collector.Stage("detect-substrate")
	scenarioDir, ok := resolveScenarioDir(s.repoRoot, scenario)
	if !ok {
		detect.End()
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

	var sub Substrate
	var err error
	if s.facts != nil {
		facts, factErr := s.facts(ctx, scenarioDir)
		if factErr != nil {
			s.logger.Printf("[security-health] Code Facts discovery degraded for %q: %v", scenario, factErr)
			sub, err = DetectSubstrate(scenarioDir)
		} else {
			sub = DetectSubstrateFromFacts(facts)
		}
	} else {
		sub, err = DetectSubstrate(scenarioDir)
	}
	detect.End()
	if err != nil {
		return Report{}, fmt.Errorf("detect substrate for %q: %w", scenario, err)
	}

	var findings []Finding
	var skipped []string

	headers := collector.Stage("security-headers")
	headerFindings, err := runSecurityHeaderChecks(scenarioDir)
	headers.Gauge("findings", float64(len(headerFindings)))
	headers.End()
	if err != nil {
		s.logger.Printf("[security-health] security headers check degraded for %q: %v", scenario, err)
		findings = append(findings, Finding{
			RuleID:      "security-health.security-headers-degraded",
			Severity:    SeverityInfo,
			Title:       "Security headers check could not complete",
			Description: fmt.Sprintf("The in-process security headers validator could not inspect the target API: %v", err),
			Remediation: "Inspect the target API tree and rerun Security Health after filesystem/read errors are resolved.",
			Scanner:     "security-headers",
		})
	} else {
		findings = append(findings, headerFindings...)
	}

	scan := collector.Stage("scan")
	for _, sc := range s.scanners {
		if !sc.Applies(sub) {
			continue
		}
		scannerStage := scan.Stage("scanner:" + sc.Name())
		if _, lookErr := s.cmd.LookPath(sc.Binary()); lookErr != nil {
			scannerStage.Gauge("available", 0).End()
			skipped = append(skipped, sc.Name())
			findings = append(findings, Finding{
				RuleID:        "security-health.scanner-absent",
				Severity:      coverageSeverity(s.policy),
				Title:         fmt.Sprintf("Scanner %q not installed", sc.Name()),
				Description:   fmt.Sprintf("The %q scanner applies to this scenario's substrate but its binary (%q) is not on PATH, so this class of issue was not checked.", sc.Name(), sc.Binary()),
				Remediation:   fmt.Sprintf("Install %q to enable this check (see docs/concepts/INTEGRATIONS.md). Until then this is informational, not a failure.", sc.Name()),
				FilePath:      "",
				Scanner:       sc.Name(),
				Class:         FindingScannerHealth,
				EvidenceState: EvidenceUnavailable,
				Confidence:    "degraded",
				Owner:         "security-health",
				PolicyImpact:  string(s.policy),
			})
			continue
		}
		scannerStage.Gauge("available", 1)
		run := func(runCtx context.Context) ([]Finding, error) { return sc.Scan(runCtx, scenarioDir, sub) }
		var scanFindings []Finding
		var scanErr error
		var outcome EvidenceOutcome
		fingerprintStarted := time.Now()
		if incremental, ok := sc.(IncrementalScanner); ok {
			plan, planErr := incremental.EvidencePlan(ctx, scenarioDir, sub, s.clock())
			scannerStage.Gauge("fingerprint_ms", float64(time.Since(fingerprintStarted).Milliseconds()))
			if planErr == nil {
				scannerStage.Gauge("weight", float64(plan.Weight))
				scanFindings, outcome, scanErr = s.evidence.Execute(ctx, EvidenceKey{Scenario: scenario, Scanner: sc.Name(), Fingerprint: plan.Fingerprint}, plan.Weight, plan.FreshFor, run)
			} else {
				s.logger.Printf("[security-health] scanner %q fingerprint degraded for %q; executing uncached: %v", sc.Name(), scenario, planErr)
				scannerStage.Gauge("weight", 1)
				scanFindings, outcome, scanErr = s.evidence.ExecuteUncached(ctx, sc.Name(), 1, run)
			}
		} else {
			scannerStage.Gauge("fingerprint_ms", 0)
			scannerStage.Gauge("weight", 1)
			scanFindings, outcome, scanErr = s.evidence.ExecuteUncached(ctx, sc.Name(), 1, run)
		}
		scannerStage.
			Gauge("cache_hit", boolGauge(outcome.Source == EvidenceSourceCache)).
			Gauge("cache_miss", boolGauge(outcome.Source == EvidenceSourceExecution || outcome.Source == EvidenceSourceCoalesced)).
			Gauge("coalesced", boolGauge(outcome.Source == EvidenceSourceCoalesced)).
			Gauge("executed", boolGauge(outcome.Source == EvidenceSourceExecution || outcome.Source == EvidenceSourceUncached)).
			Gauge("uncached", boolGauge(outcome.Source == EvidenceSourceUncached)).
			Gauge("failed", boolGauge(scanErr != nil)).
			Gauge("cache_error", boolGauge(outcome.CacheError)).
			Gauge("admission_wait_ms", float64(outcome.AdmissionWait.Milliseconds())).
			Gauge("execution_ms", float64(outcome.ExecutionTime.Milliseconds())).
			Gauge("findings", float64(len(scanFindings)))
		scannerStage.End()
		if scanErr != nil {
			s.logger.Printf("[security-health] scanner %q degraded for %q: %v", sc.Name(), scenario, scanErr)
			findings = append(findings, Finding{
				RuleID:        "security-health.scanner-degraded",
				Severity:      coverageSeverity(s.policy),
				Title:         fmt.Sprintf("Scanner %q could not complete", sc.Name()),
				Description:   fmt.Sprintf("%q is installed but did not produce a parseable result: %v", sc.Name(), scanErr),
				Remediation:   fmt.Sprintf("Run %q against this scenario by hand to diagnose; until it completes this check is informational, not a failure.", sc.Name()),
				FilePath:      "",
				Scanner:       sc.Name(),
				Class:         FindingScannerHealth,
				EvidenceState: EvidenceFailed,
				Confidence:    "degraded",
				Owner:         "security-health",
				PolicyImpact:  string(s.policy),
			})
			continue
		}
		findings = append(findings, scanFindings...)
	}
	scan.Gauge("findings", float64(len(findings)))
	scan.End()

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
	for _, target := range sub.Targets {
		if target.Coverage != CoverageEvidence {
			continue
		}
		findings = append(findings, Finding{
			RuleID:        "security-health.coverage-evidence-only." + string(target.Ecosystem),
			Severity:      coverageSeverity(s.policy),
			Title:         fmt.Sprintf("%s dependency evidence is not vulnerability-scanned", target.Ecosystem),
			Description:   target.Reason,
			Remediation:   "Use the ecosystem adapter's supported scanner or register a provider before selecting a required policy.",
			FilePath:      strings.Join(target.Manifests, ","),
			Scanner:       "security-health",
			Class:         FindingCoverage,
			EvidenceState: EvidenceUnsupported,
			Confidence:    "degraded",
			Owner:         "security-health",
			FixClass:      FixProhibited,
			PolicyImpact:  string(s.policy),
		})
	}

	sortFindings(findings)
	sort.Strings(skipped)
	report := finalize(scenario, findings, dedupeStrings(skipped))
	report.PolicyMode = s.policy
	return report, nil
}

func boolGauge(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func coverageSeverity(policy RolloutProfile) Severity {
	if policy == RolloutGuarded || policy == RolloutEnforcing {
		return SeverityError
	}
	return SeverityInfo
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
