package dependencies

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"security-health/internal/validation"

	"github.com/vrooli/api-core/schedule"
)

const (
	// EnvScanConcurrency caps how many scenarios are scanned in parallel during a
	// fleet annotate. Bounds peak CPU (the fleet walk runs ~110 osv-scanner
	// subprocesses) while still shortening the unavoidable changed-scenario pass.
	// Mirrors the architecture-cartographer *_VALIDATE_CONCURRENCY precedent.
	EnvScanConcurrency = "SECURITY_HEALTH_SCAN_CONCURRENCY"
	// DefaultScanConcurrency is the conservative parallelism used when the env
	// knob is unset/invalid — small so a fleet re-scan can't re-storm the CPU.
	DefaultScanConcurrency = 4
)

// scanCache is the result-cache surface the Annotator needs (satisfied by
// *Store). Kept as an interface so annotate is testable without a real DB and so
// a nil cache cleanly disables caching (cold path = always scan).
type scanCache interface {
	GetOSVScanCache(ctx context.Context, scenario, key string) ([]byte, bool)
	PutOSVScanCache(ctx context.Context, scenario, key string, report []byte, now string) error
}

// Annotator adds known-vulnerability status to discovered dependency records
// using osv-scanner. It is a thin adapter over the validation package's OSV
// runner so the SBOM index and the validation gate agree on vuln data.
//
// Scans are content-cached: a per-scenario key over every lockfile's content
// plus the osv-scanner version and offline OSV-DB epoch gates the subprocess, so
// an unchanged scenario re-uses its parsed report instead of re-running the
// scanner. The fleet annotate runs scenarios in bounded parallel (concurrency
// env-capped) against the offline OSV DB.
type Annotator struct {
	repoRoot string
	cmd      validation.Commander
	cache    scanCache
	clock    schedule.Clock

	// scannerVersion + dayEpoch are folded into every cache key. scannerVersion
	// invalidates the whole cache on an osv-scanner upgrade (a new scanner can
	// surface new findings); dayEpoch (the UTC date) bounds cache staleness to a
	// day so a scenario with unchanged dependencies still re-scans daily and
	// picks up vulnerabilities newly published upstream. Computed once per
	// Annotate call.
	scannerVersion string
	dayEpoch       string

	// last run's scan counters (observability): how many scenarios re-ran the
	// scanner vs. were served from cache.
	lastScansRun     atomic.Int64
	lastScansSkipped atomic.Int64
}

// NewAnnotator returns an annotator rooted at repoRoot. A nil Commander uses
// the real exec-backed one. Result caching is opt-in via WithCache; without it
// every scenario scans every reconcile (the pre-cache behaviour).
func NewAnnotator(repoRoot string, cmd validation.Commander) *Annotator {
	if cmd == nil {
		cmd = validation.NewExecCommander()
	}
	return &Annotator{repoRoot: repoRoot, cmd: cmd, clock: schedule.System()}
}

// WithCache attaches the result cache (the SQLite store). Returns the annotator
// for chaining at construction time.
func (a *Annotator) WithCache(cache scanCache) *Annotator {
	a.cache = cache
	return a
}

// ScanStats reports the most recent Annotate run's cache effectiveness.
type ScanStats struct {
	ScansRun     int
	ScansSkipped int
}

// LastScanStats returns the scenario scan/skip counts from the most recent
// Annotate call (observability for the reconcile loop).
func (a *Annotator) LastScanStats() ScanStats {
	return ScanStats{
		ScansRun:     int(a.lastScansRun.Load()),
		ScansSkipped: int(a.lastScansSkipped.Load()),
	}
}

// vulnIndex maps a dependency key (ecosystem|name|version) to its known vulns.
type vulnEntry struct {
	ids             []string
	maxSeverity     string
	vulnerabilities []VulnerabilityRecord
}

// Annotate enriches records in place with vuln_ids + max_severity. Records are
// grouped by scenario; osv-scanner runs once per scenario directory. A scenario
// whose scan fails is left unannotated (no vulns recorded) rather than failing
// the whole reconcile.
//
// Scenarios scan in bounded parallel (SECURITY_HEALTH_SCAN_CONCURRENCY). When a
// cache is wired, an unchanged scenario (lockfile content + scanner version +
// OSV-DB epoch all match) re-uses its stored report and skips the subprocess —
// the headline steady-state CPU win. The cache key folds in everything that can
// change the result, so any real change forces a re-scan (no false skips).
func (a *Annotator) Annotate(ctx context.Context, records []DependencyRecord) {
	a.lastScansRun.Store(0)
	a.lastScansSkipped.Store(0)

	byScenario := map[string][]int{}
	for i, r := range records {
		byScenario[r.Scenario] = append(byScenario[r.Scenario], i)
	}

	// Capture the scanner version + day epoch once for the whole pass: both are
	// loop-invariant, and a per-scenario probe would add a subprocess per scan.
	a.scannerVersion = validation.OSVScannerVersion(ctx, a.cmd)
	a.dayEpoch = a.now()[:10] // UTC date YYYY-MM-DD → at most one re-scan/day

	// Stable scenario order so the bounded pass is deterministic and tests can
	// reason about it.
	scenarios := make([]string, 0, len(byScenario))
	for scenario := range byScenario {
		scenarios = append(scenarios, scenario)
	}
	sort.Strings(scenarios)

	// annotated entries are applied back into the shared records slice under a
	// mutex; each goroutine only touches its own scenario's indices, but the
	// write-back is serialized to keep the race detector and reviewers happy.
	var writeMu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(a.concurrency())
	for _, scenario := range scenarios {
		scenario, idxs := scenario, byScenario[scenario]
		g.Go(func() error {
			if gctx.Err() != nil {
				return nil
			}
			report, served, ok := a.scanScenario(gctx, scenario)
			if !ok {
				return nil // scan failed: leave this scenario unannotated
			}
			if served {
				a.lastScansSkipped.Add(1)
			} else {
				a.lastScansRun.Add(1)
			}
			index := buildVulnIndex(report)
			writeMu.Lock()
			for _, i := range idxs {
				if entry, ok := index[records[i].matchKey()]; ok {
					records[i].VulnIDs = entry.ids
					records[i].MaxSeverity = entry.maxSeverity
					records[i].Vulnerabilities = entry.vulnerabilities
				}
			}
			writeMu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
}

// concurrency resolves the parallel-scan cap from the environment, clamping to a
// sane minimum and falling back to the default for an unset/invalid value.
func (a *Annotator) concurrency() int {
	raw := strings.TrimSpace(os.Getenv(EnvScanConcurrency))
	if raw == "" {
		return DefaultScanConcurrency
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		log.Printf("[security-health] invalid %s=%q, using default %d", EnvScanConcurrency, raw, DefaultScanConcurrency)
		return DefaultScanConcurrency
	}
	return n
}

// scanScenario returns the OSV report for one scenario, served from the cache
// when the content key matches (served=true) or freshly scanned otherwise. A
// fresh successful scan is written back to the cache. ok=false means the scan
// failed and the scenario should be left unannotated.
func (a *Annotator) scanScenario(ctx context.Context, scenario string) (report validation.OSVReport, served, ok bool) {
	scenarioDir := filepath.Join(a.repoRoot, "scenarios", scenario)
	key := a.scenarioCacheKey(scenarioDir)

	if a.cache != nil && key != "" {
		if payload, hit := a.cache.GetOSVScanCache(ctx, scenario, key); hit {
			var cached validation.OSVReport
			if err := json.Unmarshal(payload, &cached); err == nil {
				return cached, true, true
			}
			// Corrupt payload: fall through and re-scan (then overwrite).
		}
	}

	report, ok2 := a.runScan(ctx, scenarioDir)
	if !ok2 {
		return validation.OSVReport{}, false, false
	}
	a.storeCache(ctx, scenario, key, report)
	return report, false, true
}

func (a *Annotator) runScan(ctx context.Context, scenarioDir string) (validation.OSVReport, bool) {
	report, err := validation.RunOSVScanner(ctx, a.cmd, scenarioDir)
	if err != nil {
		return validation.OSVReport{}, false
	}
	return report, true
}

func (a *Annotator) storeCache(ctx context.Context, scenario, key string, report validation.OSVReport) {
	if a.cache == nil || key == "" {
		return
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return
	}
	if err := a.cache.PutOSVScanCache(ctx, scenario, key, payload, a.now()); err != nil {
		log.Printf("[security-health] osv scan cache write failed for %s: %v", scenario, err)
	}
}

func (a *Annotator) now() string {
	c := a.clock
	if c == nil {
		c = schedule.System()
	}
	return c.Now().UTC().Format(time.RFC3339)
}

// osvLockfiles is the set of resolved-version manifests osv-scanner reads for
// the ecosystems this annotator keeps (Go + npm). Every name here is folded into
// the cache key, so a change to any of them re-scans the scenario. npm versions
// can be pinned by pnpm-lock.yaml OR package-lock.json OR yarn.lock OR
// npm-shrinkwrap.json (the repo has 55+ package-lock.json files), so all four
// npm lockfiles must be covered — omitting any is a false-skip bug (a changed
// package-lock.json would be masked by a stale cache hit). A superset here only
// ever causes an extra (correct) re-scan, never a missed one.
var osvLockfiles = map[string]struct{}{
	"go.mod":              {},
	"go.sum":              {},
	"pnpm-lock.yaml":      {},
	"package-lock.json":   {},
	"yarn.lock":           {},
	"npm-shrinkwrap.json": {},
}

// scenarioCacheKey hashes everything that can change a scenario's osv-scanner
// result: every resolved-version lockfile's content (osvLockfiles) under the
// scenario tree, plus the osv-scanner version and the UTC day epoch. The
// lockfile walk mirrors DiscoverScenario (same skipDirs) so the key tracks
// exactly the inputs the scan reads. Returns "" when the dir can't be walked, in
// which case the caller bypasses the cache and always scans (fail-safe: never a
// false skip).
func (a *Annotator) scenarioCacheKey(scenarioDir string) string {
	h := sha256.New()
	fmt.Fprintf(h, "v1\x00scanner=%s\x00day=%s\n", a.scannerVersion, a.dayEpoch)

	type lock struct {
		rel  string
		data []byte
	}
	var locks []lock
	walkErr := filepath.WalkDir(scenarioDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			if path != scenarioDir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := osvLockfiles[d.Name()]; ok {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			rel, _ := filepath.Rel(scenarioDir, path)
			locks = append(locks, lock{rel: filepath.ToSlash(rel), data: data})
		}
		return nil
	})
	if walkErr != nil {
		return ""
	}
	// Deterministic order regardless of walk traversal.
	sort.Slice(locks, func(i, j int) bool { return locks[i].rel < locks[j].rel })
	for _, l := range locks {
		fmt.Fprintf(h, "%s\x00%d\n", l.rel, len(l.data))
		h.Write(l.data)
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// buildVulnIndex turns an OSV report into a per-dependency lookup keyed the
// same way DependencyRecord.Key() is.
func buildVulnIndex(report validation.OSVReport) map[string]vulnEntry {
	index := map[string]vulnEntry{}
	for _, res := range report.Results {
		for _, pkg := range res.Packages {
			eco := normalizeOSVEcosystem(pkg.Package.Ecosystem)
			if eco == EcosystemUnspecified {
				continue
			}
			ids := make([]string, 0, len(pkg.Vulnerabilities))
			vulns := make([]VulnerabilityRecord, 0, len(pkg.Vulnerabilities))
			worst := ""
			for _, v := range pkg.Vulnerabilities {
				ids = append(ids, v.ID)
				normalized := validation.OSVSeverityWord(v.DatabaseSpecific.Severity)
				worst = worseSeverity(worst, normalized)
				vulns = append(vulns, osvVulnerabilityRecord(pkg, v, normalized))
			}
			// Fold in group max_severity (CVSS score) as a fallback when the
			// per-vuln database_specific severity is absent.
			for _, g := range pkg.Groups {
				worst = worseSeverity(worst, validation.OSVSeverityWord(g.MaxSeverity))
			}
			if len(ids) == 0 {
				continue
			}
			// Join on (eco,name,version) so the same dependency in two
			// scenarios shares vuln data regardless of scenario.
			index[depMatchKey(eco, pkg.Package.Name, pkg.Package.Version)] = vulnEntry{ids: ids, maxSeverity: worst, vulnerabilities: vulns}
		}
	}
	return index
}

func osvVulnerabilityRecord(pkg validation.OSVPackage, vuln validation.OSVVuln, normalizedSeverity string) VulnerabilityRecord {
	affected := make([]AffectedVersionRange, 0)
	fixed := make([]FixedVersionRange, 0)
	for _, affectedSet := range vuln.Affected {
		for _, r := range affectedSet.Ranges {
			for _, event := range r.Events {
				ar := AffectedVersionRange{
					Introduced:   strings.TrimSpace(event.Introduced),
					Fixed:        strings.TrimSpace(event.Fixed),
					LastAffected: strings.TrimSpace(event.LastAffected),
				}
				switch {
				case ar.Introduced != "" && ar.Fixed != "":
					ar.Range = ">=" + ar.Introduced + " <" + ar.Fixed
				case ar.Fixed != "":
					ar.Range = "<" + ar.Fixed
				case ar.Introduced != "":
					ar.Range = ">=" + ar.Introduced
				case ar.LastAffected != "":
					ar.Range = "<= " + ar.LastAffected
				}
				if ar.Range != "" || ar.Introduced != "" || ar.Fixed != "" || ar.LastAffected != "" {
					affected = append(affected, ar)
				}
				if ar.Fixed != "" {
					fixed = append(fixed, FixedVersionRange{Range: ">= " + ar.Fixed, Version: ar.Fixed})
				}
			}
		}
	}
	return VulnerabilityRecord{
		VulnerabilityID:    strings.TrimSpace(vuln.ID),
		Aliases:            trimAndSort(vuln.Aliases),
		Ecosystem:          normalizeOSVEcosystem(pkg.Package.Ecosystem),
		Name:               strings.TrimSpace(pkg.Package.Name),
		Version:            strings.TrimSpace(pkg.Package.Version),
		AffectedRanges:     affected,
		FixedRanges:        fixed,
		Severity:           strings.TrimSpace(vuln.DatabaseSpecific.Severity),
		NormalizedSeverity: normalizedSeverity,
		AdvisoryURL:        "https://osv.dev/vulnerability/" + strings.TrimSpace(vuln.ID),
		Summary:            strings.TrimSpace(vuln.Summary),
		Details:            strings.TrimSpace(vuln.Detail),
		Source:             VulnerabilitySourceOSV,
		Reachability:       ReachabilityLockfileAffected,
		Confidence:         EvidenceConfidenceAdvisory,
		Production:         true,
		Remediation:        "Upgrade " + strings.TrimSpace(pkg.Package.Name) + " to a fixed version (" + firstFixedRange(fixed) + ").",
	}
}

func firstFixedRange(ranges []FixedVersionRange) string {
	for _, r := range ranges {
		if strings.TrimSpace(r.Range) != "" {
			return strings.TrimSpace(r.Range)
		}
	}
	return "latest patched release"
}

func trimAndSort(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// depMatchKey is the scenario-independent identity used to join osv results to
// records (the same dependency in two scenarios shares vuln data).
func depMatchKey(eco Ecosystem, name, version string) string {
	return string(eco) + "|" + name + "|" + version
}

// Key override for matching: records join on (eco,name,version), not scenario.
// We re-key the index lookup in Annotate via a small shim.
func (r DependencyRecord) matchKey() string {
	return depMatchKey(r.Ecosystem, r.Name, r.Version)
}

func normalizeOSVEcosystem(raw string) Ecosystem {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "go":
		return EcosystemGo
	case "npm":
		return EcosystemNPM
	default:
		return EcosystemUnspecified
	}
}

// severityRank orders the normalized severity words for max-folding.
var severityRank = map[string]int{"": 0, "low": 1, "moderate": 2, "high": 3, "critical": 4}

func worseSeverity(a, b string) string {
	if severityRank[b] > severityRank[a] {
		return b
	}
	return a
}
