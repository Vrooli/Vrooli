package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// osvScanner runs google/osv-scanner across the scenario's lockfiles
// (go.mod, pnpm-lock.yaml, …) and reports known vulnerabilities for the exact
// pinned versions. It applies whenever any scannable lockfile substrate is
// present. The same parsed output also feeds the Dependency & Vulnerability
// Intelligence index (see internal/dependencies). The binary is install-gated;
// when absent the Service records it as a skipped scanner.
type osvScanner struct {
	cmd   Commander
	cache OSVReportCache
	clock func() time.Time
}

func newOSVScanner(cmd Commander) Scanner { return &osvScanner{cmd: cmd} }

// OSVReportCache is the shared raw-report cache seam. The dependencies store
// satisfies it directly, allowing reconciliation and validation to reuse one
// authoritative scanner result without coupling the validation package to the
// dependencies domain.
type OSVReportCache interface {
	GetOSVScanCache(ctx context.Context, scenario, key string) ([]byte, bool)
	PutOSVScanCache(ctx context.Context, scenario, key string, report []byte, now string) error
}

func newOSVScannerWithCache(cmd Commander, cache OSVReportCache, clock func() time.Time) Scanner {
	if clock == nil {
		clock = time.Now
	}
	return &osvScanner{cmd: cmd, cache: cache, clock: clock}
}

// RunOSVScanner runs osv-scanner against scenarioDir and returns the parsed
// report. Exposed so the dependencies domain can annotate its SBOM index with
// the same vuln data the validation gate sees, without duplicating the
// invocation. Returns an empty report (not an error) when osv-scanner is
// absent or finds nothing.
//
// The scan runs online (osv-scanner resolves the live OSV database). Offline
// mode was evaluated and rejected: osv-scanner loads its full per-ecosystem
// database into memory on every invocation, and the npm database alone is
// ~200 MB — measured at ~100 s and ~2.6 GB RSS per scan, which turns the
// fleet's ~110-scenario cold pass into a ~45-minute, multi-GB-RAM storm. The
// dependencies result cache (keyed on lockfile content + scanner version + a
// daily epoch) is what eliminates steady-state scanning; online keeps the
// unavoidable cold/daily scans cheap and detection-fresh.
func RunOSVScanner(ctx context.Context, cmd Commander, scenarioDir string) (OSVReport, error) {
	if cmd == nil {
		cmd = NewExecCommander()
	}
	if _, err := cmd.LookPath("osv-scanner"); err != nil {
		return OSVReport{}, nil
	}
	return (&osvScanner{cmd: cmd}).run(ctx, scenarioDir)
}

// OSVSeverityWord maps an OSV severity string (CVSS word or numeric score) to
// the normalized lowercase word used by DependencyRecord.max_severity.
// Exposed for the dependencies domain.
func OSVSeverityWord(raw string) string { return osvMaxSeverityWord(osvSeverity(raw)) }

func (o *osvScanner) Name() string   { return "osv-scanner" }
func (o *osvScanner) Binary() string { return "osv-scanner" }
func (o *osvScanner) Applies(s Substrate) bool {
	if s.Go || s.PnpmUI {
		return true
	}
	for _, target := range s.Targets {
		if target.Coverage == CoverageSupported {
			return true
		}
	}
	return false
}
func (o *osvScanner) EvidencePlan(ctx context.Context, scenarioDir string, sub Substrate, now time.Time) (ScannerEvidencePlan, error) {
	return scannerEvidencePlan(ctx, o.cmd, o.Name(), o.Binary(), scenarioDir, sub, now)
}

// OSVReport is the top-level shape of `osv-scanner --format json`. Exported so
// the dependencies domain can reuse the same parser when annotating the SBOM
// index with vuln status.
type OSVReport struct {
	Results []OSVResult `json:"results"`
}

type OSVResult struct {
	Source   OSVSource    `json:"source"`
	Packages []OSVPackage `json:"packages"`
}

type OSVSource struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type OSVPackage struct {
	Package         OSVPackageInfo `json:"package"`
	Vulnerabilities []OSVVuln      `json:"vulnerabilities"`
	Groups          []OSVGroup     `json:"groups"`
}

type OSVPackageInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Ecosystem string `json:"ecosystem"`
}

type OSVVuln struct {
	ID               string   `json:"id"`
	Aliases          []string `json:"aliases"`
	Summary          string   `json:"summary"`
	Detail           string   `json:"detail"`
	DatabaseSpecific struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
	Affected []struct {
		Ranges []struct {
			Events []struct {
				Introduced   string `json:"introduced"`
				Fixed        string `json:"fixed"`
				LastAffected string `json:"last_affected"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
}

type OSVGroup struct {
	IDs         []string `json:"ids"`
	MaxSeverity string   `json:"max_severity"`
}

func (o *osvScanner) Scan(ctx context.Context, scenarioDir string, _ Substrate) ([]Finding, error) {
	report, err := o.run(ctx, scenarioDir)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, res := range report.Results {
		source := relPath(scenarioDir, res.Source.Path)
		for _, pkg := range res.Packages {
			maxSev := osvGroupMaxSeverity(pkg.Groups)
			for _, v := range pkg.Vulnerabilities {
				sev := v.DatabaseSpecific.Severity
				if sev == "" {
					sev = maxSev
				}
				// osv-scanner reports every CVE affecting a pinned version with
				// NO reachability analysis. The reachability-aware scanners are
				// the gating authorities: govulncheck gates Go (reachable
				// third-party vulns), pnpm-audit gates npm. osv-scanner is the
				// fleet-breadth awareness signal — it powers the Dependency &
				// Vulnerability Intelligence index (which keeps true severity
				// via the OSVReport) but stays ADVISORY (INFO) in the gate so a
				// non-reachable transitive CVE can't spuriously hold R1 or
				// double-count what govulncheck/pnpm already flagged. The
				// computed severity is surfaced in the description for context.
				findings = append(findings, Finding{
					RuleID:       "osv." + v.ID,
					Severity:     SeverityInfo,
					Title:        fmt.Sprintf("%s@%s: %s", pkg.Package.Name, pkg.Package.Version, nonEmpty(v.Summary, v.ID)),
					Description:  fmt.Sprintf("[advisory; severity=%s] %s", nonEmpty(osvMaxSeverityWord(osvSeverity(sev)), "unknown"), nonEmpty(v.Summary, v.Detail)),
					Remediation:  fmt.Sprintf("Bump %s to a fixed version (%s). See https://osv.dev/vulnerability/%s", pkg.Package.Name, osvFixHint(v), v.ID),
					FilePath:     source,
					Scanner:      o.Name(),
					Class:        FindingVulnerability,
					Owner:        "scenario-dependency-analyzer",
					FixClass:     FixAssisted,
					PolicyImpact: "dependency-upgrade-governed",
				})
			}
		}
	}
	// Stable order for deterministic CLI output / fixtures.
	sort.SliceStable(findings, func(i, j int) bool { return findings[i].RuleID < findings[j].RuleID })
	return findings, nil
}

// run executes osv-scanner and parses its JSON report. Exposed within the
// package so the dependencies domain can call it for vuln annotation.
func (o *osvScanner) run(ctx context.Context, scenarioDir string) (OSVReport, error) {
	if o.cache != nil {
		now := time.Now()
		if o.clock != nil {
			now = o.clock()
		}
		key, keyErr := OSVEvidenceFingerprint(ctx, o.cmd, scenarioDir, now)
		if keyErr == nil {
			scenario := filepath.Base(filepath.Clean(scenarioDir))
			if payload, ok := o.cache.GetOSVScanCache(ctx, scenario, key); ok {
				var cached OSVReport
				if json.Unmarshal(payload, &cached) == nil {
					return cached, nil
				}
			}
			report, err := o.runUncached(ctx, scenarioDir)
			if err != nil {
				return OSVReport{}, err
			}
			if payload, marshalErr := json.Marshal(report); marshalErr == nil {
				_ = o.cache.PutOSVScanCache(ctx, scenario, key, payload, now.UTC().Format(time.RFC3339Nano))
			}
			return report, nil
		}
	}
	return o.runUncached(ctx, scenarioDir)
}

func (o *osvScanner) runUncached(ctx context.Context, scenarioDir string) (OSVReport, error) {
	lockfiles, err := discoverOSVLockfiles(scenarioDir)
	if err != nil {
		return OSVReport{}, err
	}
	if len(lockfiles) == 0 {
		return OSVReport{}, nil
	}

	// Scan only first-party dependency manifests. Recursive scanning would also
	// read package-manager fixtures bundled below node_modules, producing CVEs
	// that cannot affect the scenario's resolved dependency graph.
	args := []string{"scan", "--format", "json"}
	for _, lockfile := range lockfiles {
		args = append(args, "--lockfile", lockfile)
	}
	stdout, stderr, _, err := o.cmd.Run(ctx, scenarioDir, "osv-scanner", args...)
	if err != nil {
		return OSVReport{}, fmt.Errorf("osv-scanner failed to run: %w", err)
	}
	if len(stdout) == 0 {
		// No vulns and no report can both yield empty stdout depending on
		// version; treat empty+quiet as clean.
		if len(stderr) > 0 && !strings.Contains(string(stderr), "No vulnerabilities found") {
			return OSVReport{}, nil
		}
		return OSVReport{}, nil
	}
	var report OSVReport
	if err := json.Unmarshal(stdout, &report); err != nil {
		return OSVReport{}, fmt.Errorf("parse osv-scanner json: %w", err)
	}
	return report, nil
}

var osvLockfileNames = map[string]struct{}{
	"go.mod":              {},
	"pnpm-lock.yaml":      {},
	"package-lock.json":   {},
	"yarn.lock":           {},
	"npm-shrinkwrap.json": {},
	"bun.lock":            {},
	"bun.lockb":           {},
	"Cargo.lock":          {},
	"Cargo.toml":          {},
	"requirements.txt":    {},
	"pyproject.toml":      {},
	"Pipfile.lock":        {},
	"poetry.lock":         {},
}

// discoverOSVLockfiles returns the scenario-owned manifests that describe its
// dependency graph. It deliberately uses the same excluded directories as
// substrate discovery, so installed dependencies and build artifacts never
// become scanner inputs.
func discoverOSVLockfiles(scenarioDir string) ([]string, error) {
	var lockfiles []string
	err := filepath.WalkDir(scenarioDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if _, skip := skipDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
			if path != scenarioDir && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := osvLockfileNames[entry.Name()]; ok {
			lockfiles = append(lockfiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(lockfiles)
	return lockfiles, nil
}

// osvGroupMaxSeverity returns the highest max_severity (CVSS score string)
// across a package's groups.
func osvGroupMaxSeverity(groups []OSVGroup) string {
	best := ""
	bestVal := -1.0
	for _, g := range groups {
		if v, err := strconv.ParseFloat(g.MaxSeverity, 64); err == nil && v > bestVal {
			bestVal = v
			best = g.MaxSeverity
		}
	}
	return best
}

// osvSeverity maps an OSV severity — either a CVSS word (CRITICAL/HIGH/…) or a
// numeric CVSS score string — onto the normalized scale.
func osvSeverity(raw string) Severity {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return SeverityInfo
	}
	if score, err := strconv.ParseFloat(raw, 64); err == nil {
		switch {
		case score >= 7.0:
			return SeverityError // high/critical CVSS
		case score >= 4.0:
			return SeverityWarning
		default:
			return SeverityInfo
		}
	}
	return NormalizeSeverity(raw)
}

func osvFixHint(v OSVVuln) string {
	for _, a := range v.Affected {
		for _, r := range a.Ranges {
			for _, e := range r.Events {
				if e.Fixed != "" {
					return ">= " + e.Fixed
				}
			}
		}
	}
	return "the latest patched release"
}

// OSVScannerVersion returns the installed osv-scanner version string (the
// `osv-scanner --version` first-line value), or "" when the binary is absent or
// the probe fails. Folded into the dependencies result-cache key so a scanner
// upgrade invalidates every cached scan (a new scanner can surface new
// findings). Cheap and called once per loop, not per scan.
func OSVScannerVersion(ctx context.Context, cmd Commander) string {
	if cmd == nil {
		cmd = NewExecCommander()
	}
	if _, err := cmd.LookPath("osv-scanner"); err != nil {
		return ""
	}
	stdout, stderr, _, err := cmd.Run(ctx, "", "osv-scanner", "--version")
	if err != nil {
		return ""
	}
	out := strings.TrimSpace(string(stdout))
	if out == "" {
		out = strings.TrimSpace(string(stderr))
	}
	// First line is "osv-scanner version: X.Y.Z"; collapse to that line so a
	// changing build-date footer doesn't churn the key needlessly.
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		out = out[:i]
	}
	return strings.TrimSpace(out)
}

// osvMaxSeverityWord maps the normalized severity back to the lowercase word
// the DependencyRecord.max_severity field uses ("", "low", "moderate",
// "high", "critical"). Exposed for the dependencies domain.
func osvMaxSeverityWord(s Severity) string {
	switch s {
	case SeverityError:
		return "high"
	case SeverityWarning:
		return "moderate"
	case SeverityInfo:
		return "low"
	default:
		return ""
	}
}
