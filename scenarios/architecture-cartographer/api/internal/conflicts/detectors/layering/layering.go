// Package layering detects dependency-direction violations between the
// canonical Vrooli surface zones.
package layering

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

const (
	zoneUnknown         = ""
	zoneTransport       = "transport"
	zoneDomain          = "domain"
	zoneSubstrate       = "substrate"
	zoneCompositionRoot = "composition-root"
	zoneCLI             = "cli"
	zoneUI              = "ui"
)

var builtInSubstrateSegments = map[string]struct{}{
	"clock":        {},
	"config":       {},
	"database":     {},
	"git":          {},
	"httpc":        {},
	"httpx":        {},
	"middleware":   {},
	"server":       {},
	"suppressions": {},
	"testutil":     {},
}

var builtInCompositionRootSegments = map[string]struct{}{
	"app":     {},
	"module":  {},
	"modules": {},
}

// coordinatingArchetypes are the domain archetypes permitted to import sibling
// domains: a domain declared with one of these roles is a legitimate coordinator
// (it composes other domains by design) rather than a leaky boundary. `provider`
// and `aggregation` join the orchestration-family roles because a provider
// (serves a composed read/validation surface over several domains) and an
// aggregator (rolls many domains' data into one view) coordinate by definition.
// Keep this in sync with the archetype vocabulary in
// internal/domains/extractor_doc.go.
var coordinatingArchetypes = map[string]struct{}{
	"aggregation":      {},
	"composition-root": {},
	"infrastructure":   {},
	"mutation":         {},
	"orchestration":    {},
	"provider":         {},
	"service":          {},
}

// coordinatingVocabulary is the sorted, human-readable list of coordinating
// archetypes, used in remediation messages so the fix is discoverable.
func coordinatingVocabulary() string {
	roles := make([]string, 0, len(coordinatingArchetypes))
	for r := range coordinatingArchetypes {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	return strings.Join(roles, ", ")
}

// Detector flags wrong-direction imports. Strict mode promotes blocker-class
// direction violations to blocker severity; non-strict mode keeps them error
// severity so operators can roll the detector out without gating.
type Detector struct {
	strict bool
}

// New returns the production detector.
func New() *Detector { return NewWithStrict(true) }

// NewWithStrict returns a detector with the strictness control surface applied.
func NewWithStrict(strict bool) *Detector { return &Detector{strict: strict} }

func (Detector) Name() string { return "layering" }

func (Detector) Description() string {
	return "Flags wrong-direction imports between transport, domain, and substrate zones."
}

func (Detector) EmitsTypes() []string { return []string{"layering"} }

func (Detector) Class() conflicts.FindingClass {
	return conflicts.FindingClassDeterministic
}

func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	packages := packageIndex(in.Snapshot)
	files := fileIndex(in.Snapshot)
	var out []conflicts.Conflict
	for _, edge := range in.Snapshot.Imports {
		if edge.TestOnly {
			continue
		}
		fromPkg := resolvePackage(edge.From, packages, files)
		toPkg := packages[edge.ToPackageID]
		if fromPkg.ID == "" || toPkg.ID == "" {
			continue
		}
		from := classifyPackage(fromPkg, in.DomainMap)
		to := classifyPackage(toPkg, in.DomainMap)
		if from.zone == zoneUnknown || to.zone == zoneUnknown {
			continue
		}
		violation := classifyViolation(from, to)
		if violation.kind == "" {
			continue
		}
		out = append(out, conflicts.Conflict{
			Scenario:  in.Scenario,
			Detector:  d.Name(),
			Type:      "layering",
			Subtype:   violation.kind,
			Severity:  d.severityFor(violation.blockerEligible),
			Locations: []string{from.path, to.path},
			Domains:   domainList(from.domain, to.domain),
			Evidence: []conflicts.Evidence{
				{
					Kind:    "import_edge",
					Summary: fmt.Sprintf("%s imports %s", from.path, to.path),
					Locator: fmt.Sprintf("%s -> %s", from.path, to.path),
				},
				{
					Kind:    "layer_rule",
					Summary: violation.summary,
					Locator: violation.kind,
				},
			},
			SuggestedFixes: []conflicts.Fix{{
				Kind:       conflicts.FixKindAddDependency,
				Summary:    violation.fix,
				Confidence: 0.75,
			}},
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Subtype != out[j].Subtype {
			return out[i].Subtype < out[j].Subtype
		}
		return strings.Join(out[i].Locations, "\x00") < strings.Join(out[j].Locations, "\x00")
	})
	return out, nil
}

func (d Detector) severityFor(blockerEligible bool) conflicts.Severity {
	if blockerEligible && d.strict {
		return conflicts.SeverityBlocker
	}
	return conflicts.SeverityError
}

type packageZone struct {
	path      string
	zone      string
	domain    string
	archetype string
	// declared is true when the package's domain came from a DOMAINS.md Source
	// Paths entry, false when it was inferred from the path segment (i.e. the
	// path is owned by no declared domain). An undeclared owner turns a sibling
	// import into the clearer "unowned source path" remediation.
	declared bool
}

type violation struct {
	kind            string
	summary         string
	fix             string
	blockerEligible bool
}

func classifyViolation(from, to packageZone) violation {
	if from.zone == zoneCompositionRoot || to.zone == zoneCompositionRoot {
		return violation{}
	}
	if from.zone == zoneDomain && to.zone == zoneTransport {
		return violation{
			kind:            "domain-imports-transport",
			summary:         "domain packages must not import transport handlers",
			fix:             "Invert the dependency: keep transport translation in handlers and pass domain-owned interfaces into the domain package.",
			blockerEligible: true,
		}
	}
	if from.zone == zoneSubstrate && (to.zone == zoneDomain || to.zone == zoneTransport) {
		return violation{
			kind:            "substrate-imports-product",
			summary:         "shared substrate must stay business-vocabulary-free and must not import product domains or handlers",
			fix:             "Move product vocabulary behind an interface owned by the consuming domain, or move the code out of shared substrate.",
			blockerEligible: true,
		}
	}
	if from.zone == zoneDomain && to.zone == zoneDomain && from.domain != "" && to.domain != "" && from.domain != to.domain && !mayCoordinate(from.archetype) {
		if !from.declared {
			return unownedSourcePath(from, to)
		}
		return violation{
			kind:    "domain-imports-sibling-domain",
			summary: "a non-coordinating domain package must not import sibling domain internals",
			fix: fmt.Sprintf("Either DECLARE %q's archetype as a coordinating role (%s) in DOMAINS.md if it legitimately composes other domains, "+
				"or INVERT the dependency: introduce a narrow interface owned by %q and wire the concrete sibling from the composition root.",
				from.domain, coordinatingVocabulary(), from.domain),
			blockerEligible: false,
		}
	}
	if from.zone == zoneTransport && to.zone == zoneDomain && from.domain != "" && to.domain != "" && from.domain != to.domain && !mayCoordinate(from.archetype) {
		if !from.declared {
			return unownedSourcePath(from, to)
		}
		return violation{
			kind:    "handler-imports-sibling-domain",
			summary: "a handler package should translate requests for its own domain, not reach into sibling domains",
			fix: fmt.Sprintf("Either DECLARE %q's archetype as a coordinating role (%s) in DOMAINS.md if it is an aggregator/provider that composes other domains, "+
				"or INVERT the dependency: move cross-domain coordination behind an interface owned by %q (wired from the composition root).",
				from.domain, coordinatingVocabulary(), from.domain),
			blockerEligible: false,
		}
	}
	return violation{}
}

// unownedSourcePath is returned when an importing package sits in a domain-ish
// zone but its path is owned by no declared domain (the domain was inferred from
// the path segment). Rather than emit a misleading "imports sibling domain"
// finding against a phantom domain, point the operator at the DOMAINS.md gap.
func unownedSourcePath(from, to packageZone) violation {
	return violation{
		kind:    "unowned-source-path",
		summary: "this source path is owned by no declared domain, so its cross-domain import cannot be governed",
		fix: fmt.Sprintf("Declare %q in DOMAINS.md: add it to the Source Paths of the domain that owns it (or a new domain), "+
			"giving it an archetype — a coordinating role (%s) if it legitimately composes %q.",
			from.path, coordinatingVocabulary(), to.domain),
		blockerEligible: false,
	}
}

func mayCoordinate(archetype string) bool {
	_, ok := coordinatingArchetypes[strings.ToLower(strings.TrimSpace(archetype))]
	return ok
}

func classifyPackage(pkg graph.PackageNode, m domains.DerivedDomainMap) packageZone {
	path := strings.Trim(strings.TrimSpace(pkg.RepoPath), "/")
	if path == "" {
		return packageZone{}
	}
	domain := m.DomainFor(path)
	archetype := archetypeFor(domain, m)
	declared := domain != ""
	switch {
	case strings.HasPrefix(path, "api/handlers/"):
		return packageZone{path: path, zone: zoneTransport, domain: firstNonEmpty(domain, segmentAfter(path, "api/handlers/")), archetype: archetype, declared: declared}
	case strings.HasPrefix(path, "api/internal/"):
		if isBuiltInCompositionRoot(path) {
			return packageZone{path: path, zone: zoneCompositionRoot, archetype: "composition-root", declared: true}
		}
		if m.IsSharedSubstrate(path) || isBuiltInSubstrate(path) {
			return packageZone{path: path, zone: zoneSubstrate, declared: true}
		}
		if domain != "" {
			return packageZone{path: path, zone: zoneDomain, domain: domain, archetype: archetype, declared: true}
		}
	case strings.HasPrefix(path, "cli/domains/"):
		return packageZone{path: path, zone: zoneCLI, domain: firstNonEmpty(domain, segmentAfter(path, "cli/domains/")), archetype: archetype, declared: declared}
	case strings.HasPrefix(path, "ui/src/features/"):
		return packageZone{path: path, zone: zoneUI, domain: firstNonEmpty(domain, segmentAfter(path, "ui/src/features/")), archetype: archetype, declared: declared}
	}
	return packageZone{path: path, zone: zoneUnknown, domain: domain, archetype: archetype, declared: declared}
}

func isBuiltInSubstrate(path string) bool {
	seg := segmentAfter(path, "api/internal/")
	_, ok := builtInSubstrateSegments[seg]
	return ok
}

func isBuiltInCompositionRoot(path string) bool {
	seg := segmentAfter(path, "api/internal/")
	_, ok := builtInCompositionRootSegments[seg]
	return ok
}

func archetypeFor(domain string, m domains.DerivedDomainMap) string {
	for _, d := range m.Domains {
		if d.Name == domain {
			return d.Archetype
		}
	}
	return ""
}

func packageIndex(snap graph.GraphSnapshot) map[string]graph.PackageNode {
	out := make(map[string]graph.PackageNode, len(snap.Packages))
	for _, p := range snap.Packages {
		out[p.ID] = p
	}
	return out
}

func fileIndex(snap graph.GraphSnapshot) map[string]graph.FileNode {
	out := make(map[string]graph.FileNode, len(snap.Files))
	for _, f := range snap.Files {
		out[f.ID] = f
	}
	return out
}

func resolvePackage(id string, packages map[string]graph.PackageNode, files map[string]graph.FileNode) graph.PackageNode {
	if p, ok := packages[id]; ok {
		return p
	}
	if f, ok := files[id]; ok {
		return packages[f.PackageID]
	}
	return graph.PackageNode{}
}

func segmentAfter(path, prefix string) string {
	rest := strings.TrimPrefix(path, prefix)
	if rest == path {
		return ""
	}
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return ""
	}
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		return rest[:i]
	}
	return rest
}

func firstNonEmpty(parts ...string) string {
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			return p
		}
	}
	return ""
}

func domainList(parts ...string) []string {
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
