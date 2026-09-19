package validation

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Service validates one source target at a time. It owns the detector → scanner
// pipeline and the severity contract; the Connect handler and the CLI are thin
// translation layers over ValidateTarget.
type Service struct {
	repoRoot                string
	cmd                     Commander
	scanners                []Scanner
	logger                  *log.Logger
	policy                  RolloutProfile
	facts                   FactDiscovery
	evidence                *EvidenceCoordinator
	clock                   func() time.Time
	controlPlaneErrorBudget ErrorBudget
}

// ErrorBudget is the aggregate ratchet for pre-existing blocking findings on
// a target. Declared distinguishes an explicit zero budget from no budget.
// Baseline records the measured starting point so a ratcheted budget cannot be
// loosened while an implementation change is in flight.
type ErrorBudget struct {
	Limit    int
	Baseline int
	Ratchet  bool
	Declared bool
}

func (b ErrorBudget) allows(observed int) bool {
	if !b.Declared {
		return observed == 0
	}
	if b.Ratchet && b.Limit > b.Baseline {
		return false
	}
	if observed > b.Limit {
		return false
	}
	return !b.Ratchet || observed <= b.Baseline
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
	OSVReportCache          OSVReportCache
	ControlPlaneErrorBudget ErrorBudget
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
	logger.Printf("[security-health] scanner subprocess GOMAXPROCS=%d", scannerMaxProcs())
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
	return &Service{repoRoot: d.RepoRoot, cmd: cmd, scanners: scanners, logger: logger, policy: policy, facts: d.FactDiscovery, evidence: evidence, clock: clock, controlPlaneErrorBudget: d.ControlPlaneErrorBudget}
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
// ValidationTargetKind identifies the repository-owned source tree being
// scanned. The scanner engine is intentionally kind-agnostic; the kind only
// supplies a stable report/evidence identity at the resolver boundary.
type ValidationTargetKind string

const (
	ValidationTargetScenario     ValidationTargetKind = "scenario"
	ValidationTargetControlPlane ValidationTargetKind = "control-plane"
)

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
	return s.ValidateTarget(ctx, ValidationTargetScenario, scenarioDir)
}

// ValidateTarget detects substrates beneath an already-resolved source root,
// runs every applicable scanner, and returns the same report contract used by
// ValidateScenario. Control-plane targets use the repository root, which puts
// the root module and nested package modules under the provider-owned SAST and
// vulnerability pipeline without teaching scanners about repository layout.
func (s *Service) ValidateTarget(ctx context.Context, kind ValidationTargetKind, path string) (Report, error) {
	ctx = withEvidenceWalkCache(ctx)
	kind = ValidationTargetKind(strings.ToLower(strings.TrimSpace(string(kind))))
	path = strings.TrimSpace(path)
	if path == "" {
		return Report{}, fmt.Errorf("validation target path is required")
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return Report{}, fmt.Errorf("validation target path %q is not a readable directory", path)
	}

	var targetID string
	switch kind {
	case ValidationTargetScenario:
		targetID = filepath.Base(filepath.Clean(path))
	case ValidationTargetControlPlane:
		targetID = "control-plane"
	default:
		return Report{}, fmt.Errorf("unsupported security validation target kind %q", kind)
	}

	collector := metricsFrom(ctx)
	detect := collector.Stage("detect-substrate")

	var sub Substrate
	if s.facts != nil {
		facts, factErr := s.facts(ctx, path)
		if factErr != nil {
			s.logger.Printf("[security-health] Code Facts discovery degraded for %q: %v", targetID, factErr)
			sub, err = DetectSubstrate(path)
		} else {
			sub = DetectSubstrateFromFacts(facts)
		}
	} else {
		sub, err = DetectSubstrate(path)
	}
	if kind == ValidationTargetControlPlane {
		sub = controlPlaneSubstrate(sub)
	}
	detect.End()
	if err != nil {
		return Report{}, fmt.Errorf("detect substrate for %q: %w", targetID, err)
	}

	var findings []Finding
	var skipped []string

	headers := collector.Stage("security-headers")
	headerFindings, err := runSecurityHeaderChecks(path)
	headers.Gauge("findings", float64(len(headerFindings)))
	headers.End()
	if err != nil {
		s.logger.Printf("[security-health] security headers check degraded for %q: %v", targetID, err)
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
		if kind == ValidationTargetControlPlane && !controlPlaneScanner(sc.Name()) {
			continue
		}
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
		run := func(runCtx context.Context) ([]Finding, error) { return sc.Scan(runCtx, path, sub) }
		var scanFindings []Finding
		var scanErr error
		var outcome EvidenceOutcome
		fingerprintStarted := time.Now()
		if incremental, ok := sc.(IncrementalScanner); ok {
			plan, planErr := incremental.EvidencePlan(ctx, path, sub, s.clock())
			scannerStage.Gauge("fingerprint_ms", float64(time.Since(fingerprintStarted).Milliseconds()))
			if planErr == nil {
				scannerStage.Gauge("weight", float64(plan.Weight))
				scanFindings, outcome, scanErr = s.evidence.Execute(ctx, EvidenceKey{Scenario: targetID, Scanner: sc.Name(), Fingerprint: plan.Fingerprint}, plan.Weight, plan.FreshFor, run)
			} else {
				s.logger.Printf("[security-health] scanner %q fingerprint degraded for %q; executing uncached: %v", sc.Name(), targetID, planErr)
				scannerStage.Gauge("weight", 1)
				scanFindings, outcome, scanErr = s.evidence.ExecuteUncached(ctx, sc.Name(), 1, run)
			}
		} else {
			scannerStage.Gauge("fingerprint_ms", 0)
			scannerStage.Gauge("weight", 1)
			scanFindings, outcome, scanErr = s.evidence.ExecuteUncached(ctx, sc.Name(), 1, run)
		}
		scannerStage.
			Gauge("child_peak_rss_bytes", float64(collector.ChildPeakRSSBytes())).
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
			s.logger.Printf("[security-health] scanner %q degraded for %q: %v", sc.Name(), targetID, scanErr)
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

	if kind == ValidationTargetControlPlane {
		findings = dedupeControlPlaneFindings(findings)
	}
	sortFindings(findings)
	sort.Strings(skipped)
	report := finalize(targetID, findings, dedupeStrings(skipped))
	report.PolicyMode = s.policy
	if kind == ValidationTargetControlPlane {
		report.Passed = s.controlPlaneErrorBudget.allows(report.Summary.Errors)
	}
	return report, nil
}

func dedupeControlPlaneFindings(findings []Finding) []Finding {
	seen := make(map[string]struct{}, len(findings))
	out := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		key := strings.Join([]string{finding.Scanner, finding.RuleID, finding.FilePath, finding.Description}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, finding)
	}
	return out
}

// controlPlaneSubstrate keeps the repository-root module and shared package
// modules in scope while excluding nested scenarios and templates, which are
// validated under their own target identities. The root module is bounded to
// cmd/ and internal/, including the credential and privilege-broker packages.
func controlPlaneSubstrate(sub Substrate) Substrate {
	goDirs := make([]string, 0, len(sub.GoModDirs))
	patterns := make(map[string][]string)
	for _, dir := range sub.GoModDirs {
		clean := filepath.ToSlash(filepath.Clean(dir))
		if clean == "." || clean == "" {
			goDirs = append(goDirs, ".")
			patterns[dir] = []string{"./internal/...", "./cmd/..."}
			continue
		}
		if strings.HasPrefix(clean, "packages/") {
			goDirs = append(goDirs, dir)
		}
	}
	sub.GoModDirs = goDirs
	sub.GoPackagePatterns = patterns
	sub.Go = len(goDirs) > 0
	// Control-plane JavaScript lives in separately owned package/scenario
	// targets. This phase is the control-plane Go SAST and reachable-vuln gate.
	sub.PnpmUI = false
	sub.PnpmLockDirs = nil
	return sub
}

func controlPlaneScanner(name string) bool {
	switch name {
	case "gosec", "govulncheck":
		return true
	default:
		return false
	}
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
