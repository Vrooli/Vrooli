package dependencies

import (
	"context"
	"path/filepath"
	"strings"

	"security-health/internal/validation"
)

// Annotator adds known-vulnerability status to discovered dependency records
// using osv-scanner (run once per scenario). It is a thin adapter over the
// validation package's OSV runner so the SBOM index and the validation gate
// agree on vuln data.
type Annotator struct {
	repoRoot string
	cmd      validation.Commander
}

// NewAnnotator returns an annotator rooted at repoRoot. A nil Commander uses
// the real exec-backed one.
func NewAnnotator(repoRoot string, cmd validation.Commander) *Annotator {
	if cmd == nil {
		cmd = validation.NewExecCommander()
	}
	return &Annotator{repoRoot: repoRoot, cmd: cmd}
}

// vulnIndex maps a dependency key (ecosystem|name|version) to its known vulns.
type vulnEntry struct {
	ids         []string
	maxSeverity string
}

// Annotate enriches records in place with vuln_ids + max_severity. Records are
// grouped by scenario; osv-scanner runs once per scenario directory. A scenario
// whose scan fails is left unannotated (no vulns recorded) rather than failing
// the whole reconcile.
func (a *Annotator) Annotate(ctx context.Context, records []DependencyRecord) {
	byScenario := map[string][]int{}
	for i, r := range records {
		byScenario[r.Scenario] = append(byScenario[r.Scenario], i)
	}
	for scenario, idxs := range byScenario {
		scenarioDir := filepath.Join(a.repoRoot, "scenarios", scenario)
		report, err := validation.RunOSVScanner(ctx, a.cmd, scenarioDir)
		if err != nil {
			continue
		}
		index := buildVulnIndex(report)
		for _, i := range idxs {
			if entry, ok := index[records[i].matchKey()]; ok {
				records[i].VulnIDs = entry.ids
				records[i].MaxSeverity = entry.maxSeverity
			}
		}
	}
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
			worst := ""
			for _, v := range pkg.Vulnerabilities {
				ids = append(ids, v.ID)
				worst = worseSeverity(worst, validation.OSVSeverityWord(v.DatabaseSpecific.Severity))
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
			index[depMatchKey(eco, pkg.Package.Name, pkg.Package.Version)] = vulnEntry{ids: ids, maxSeverity: worst}
		}
	}
	return index
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
