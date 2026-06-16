package dependencies

import (
	"context"
	"path/filepath"
	"sort"
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
	ids             []string
	maxSeverity     string
	vulnerabilities []VulnerabilityRecord
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
				records[i].Vulnerabilities = entry.vulnerabilities
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
