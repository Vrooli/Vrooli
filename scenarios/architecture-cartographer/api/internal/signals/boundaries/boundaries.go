// Package boundaries computes domain-level coupling/boundary-health
// heuristics over a graph snapshot + derived domain map. It is pure and
// stateless: every input is passed in, every output is a value.
//
// The metrics follow the classic package-coupling model lifted to the
// domain level:
//
//   - Efferent coupling (Ce): how many other domains a domain depends on.
//   - Afferent coupling (Ca): how many other domains depend on it.
//   - Instability I = Ce / (Ce + Ca), in [0,1] (0 = maximally stable, only
//     depended upon; 1 = maximally unstable, only depends outward).
//   - Fan-out fraction: Ce / (domains - 1).
//   - Stable kernel: high Ca, low Ce, low instability — the derived shared
//     substrate, which is SUPPOSED to be widely depended upon and is exempt
//     from the god-domain smell.
//
// Output is a graded boundary-health score plus advisory smells — never a
// hard pass/fail. Composition-root archetypes are exempt from the
// god-domain smell.
package boundaries

import (
	"math"
	"sort"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// Smell kinds.
const (
	// SmellGodDomain: a domain depends on (imports) a large fraction of all
	// other domains — a junction box that resists change.
	SmellGodDomain = "god_domain"
	// SmellUnstableDependency: a domain is depended upon by several others
	// yet is itself highly unstable (high efferent coupling) — changes to it
	// ripple widely.
	SmellUnstableDependency = "unstable_dependency"
)

// Severity grades a smell. Advisory only.
type Severity string

const (
	SeverityInfo Severity = "info"
	SeverityWarn Severity = "warn"
)

// Smell is one advisory boundary-health finding for a domain.
type Smell struct {
	Kind     string
	Severity Severity
	Message  string
}

// DomainCoupling is the coupling profile of one domain.
type DomainCoupling struct {
	Domain       string
	Archetype    string
	Efferent     int      // Ce
	Afferent     int      // Ca
	Instability  float64  // I = Ce/(Ce+Ca)
	FanOut       float64  // Ce/(domains-1)
	DependsOn    []string // sorted distinct
	DependedBy   []string // sorted distinct
	StableKernel bool
	HealthScore  float64 // graded [0,1]; 1 = healthy
	Smells       []Smell
}

// Report is the per-scenario boundary-health analysis.
type Report struct {
	Scenario     string
	TotalDomains int
	Domains      []DomainCoupling // sorted by name
}

// Config holds the tunable thresholds. DefaultConfig supplies sane defaults;
// the control surface (Phase 6) overrides them globally.
type Config struct {
	// GodDomainFanOut is the efferent fan-out fraction at/above which a
	// non-exempt domain earns a god_domain smell. Range (0,1].
	GodDomainFanOut float64
	// InstabilityWarnBand is the instability at/above which a
	// depended-upon domain earns an unstable_dependency smell. Range [0,1].
	InstabilityWarnBand float64
	// StableKernelMaxEfferent is the highest Ce a domain may have and still
	// qualify as a stable kernel.
	StableKernelMaxEfferent int
	// StableKernelMinAfferentFraction is the fraction of other domains that
	// must depend on a domain for it to qualify as a stable kernel.
	StableKernelMinAfferentFraction float64
	// ExemptArchetypes are archetypes exempt from the god_domain smell
	// (composition roots legitimately wire many domains together).
	ExemptArchetypes map[string]bool
}

// DefaultConfig returns the production defaults.
func DefaultConfig() Config {
	return Config{
		GodDomainFanOut:                 0.6,
		InstabilityWarnBand:             0.7,
		StableKernelMaxEfferent:         1,
		StableKernelMinAfferentFraction: 0.5,
		ExemptArchetypes:                map[string]bool{"composition-root": true, "infrastructure": true},
	}
}

// Analyze computes the boundary-health report. It maps each internal
// package to its owning domain (via the derived map), lifts production
// import edges to domain→domain dependencies, and scores each domain.
func Analyze(scenario string, snap graph.GraphSnapshot, m domains.DerivedDomainMap, cfg Config) Report {
	// Every package the snapshot extracted is by definition first-party
	// for this scenario; stdlib/third-party never appears here. The
	// snapshot's own package set IS the "internal" set — there is no
	// separate Internal flag to gate on.
	pkgDir := make(map[string]string, len(snap.Packages))
	inScenario := make(map[string]bool, len(snap.Packages))
	for _, p := range snap.Packages {
		pkgDir[p.ID] = p.RepoPath
		inScenario[p.ID] = true
	}
	fileToPkg := make(map[string]string, len(snap.Files))
	for _, f := range snap.Files {
		fileToPkg[f.ID] = f.PackageID
	}

	archetype := make(map[string]string, len(m.Domains))
	for _, d := range m.Domains {
		archetype[d.Name] = d.Archetype
	}

	domainOf := func(pkgID string) string {
		dir := pkgDir[pkgID]
		if dir == "" {
			return ""
		}
		return m.DomainFor(dir)
	}

	// Build distinct domain→domain dependency sets from production edges.
	dependsOn := map[string]map[string]struct{}{}
	dependedBy := map[string]map[string]struct{}{}
	ensure := func(mp map[string]map[string]struct{}, k string) {
		if mp[k] == nil {
			mp[k] = map[string]struct{}{}
		}
	}
	for _, d := range m.Domains {
		ensure(dependsOn, d.Name)
		ensure(dependedBy, d.Name)
	}

	for _, e := range snap.Imports {
		if e.TestOnly {
			continue // production coupling only
		}
		fromPkg := e.From
		if p, ok := fileToPkg[e.From]; ok {
			fromPkg = p
		}
		if !inScenario[fromPkg] || !inScenario[e.ToPackageID] {
			continue
		}
		fromDom := domainOf(fromPkg)
		toDom := domainOf(e.ToPackageID)
		if fromDom == "" || toDom == "" || fromDom == toDom {
			continue
		}
		ensure(dependsOn, fromDom)
		ensure(dependedBy, toDom)
		dependsOn[fromDom][toDom] = struct{}{}
		dependedBy[toDom][fromDom] = struct{}{}
	}

	total := len(m.Domains)
	rep := Report{Scenario: scenario, TotalDomains: total}
	for _, d := range m.Domains {
		dc := computeDomain(d.Name, archetype[d.Name], dependsOn[d.Name], dependedBy[d.Name], total, cfg)
		rep.Domains = append(rep.Domains, dc)
	}
	sort.Slice(rep.Domains, func(i, j int) bool { return rep.Domains[i].Domain < rep.Domains[j].Domain })
	return rep
}

func computeDomain(name, archetype string, dependsSet, dependedSet map[string]struct{}, total int, cfg Config) DomainCoupling {
	dc := DomainCoupling{
		Domain:     name,
		Archetype:  archetype,
		Efferent:   len(dependsSet),
		Afferent:   len(dependedSet),
		DependsOn:  sortedKeys(dependsSet),
		DependedBy: sortedKeys(dependedSet),
	}
	if dc.Efferent+dc.Afferent > 0 {
		dc.Instability = float64(dc.Efferent) / float64(dc.Efferent+dc.Afferent)
	}
	others := total - 1
	if others > 0 {
		dc.FanOut = float64(dc.Efferent) / float64(others)
	}

	// Stable kernel: widely depended upon, depends on almost nothing.
	if others > 0 {
		afferentFraction := float64(dc.Afferent) / float64(others)
		dc.StableKernel = dc.Efferent <= cfg.StableKernelMaxEfferent &&
			afferentFraction >= cfg.StableKernelMinAfferentFraction
	}

	exempt := cfg.ExemptArchetypes[archetype]

	if !exempt && !dc.StableKernel && godDomainCandidate(dc, others, cfg) {
		dc.Smells = append(dc.Smells, Smell{
			Kind:     SmellGodDomain,
			Severity: SeverityWarn,
			Message:  "depends on a large fraction of all domains (high efferent fan-out) — a change-resistant junction box",
		})
	}
	if dc.Afferent >= 2 && dc.Instability >= cfg.InstabilityWarnBand {
		dc.Smells = append(dc.Smells, Smell{
			Kind:     SmellUnstableDependency,
			Severity: SeverityInfo,
			Message:  "is depended upon by multiple domains yet is itself unstable (high efferent coupling) — changes ripple widely",
		})
	}

	dc.HealthScore = scoreFrom(dc.Smells)
	return dc
}

func godDomainCandidate(dc DomainCoupling, others int, cfg Config) bool {
	return others >= 3 && dc.Efferent >= 2 && dc.FanOut >= cfg.GodDomainFanOut
}

// scoreFrom grades health from the smells: 1.0 baseline, warn costs 0.4,
// info costs 0.15, clamped to [0,1].
func scoreFrom(smells []Smell) float64 {
	score := 1.0
	for _, s := range smells {
		switch s.Severity {
		case SeverityWarn:
			score -= 0.4
		case SeverityInfo:
			score -= 0.15
		}
	}
	return math.Max(0, math.Min(1, score))
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
