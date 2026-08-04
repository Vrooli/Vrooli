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
	"architecture-cartographer/internal/zones"
)

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
	zoneConfig := zones.LoadForScenarioName(in.Scenario)
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
		from := classifyPackage(zoneConfig, fromPkg, in.DomainMap)
		to := classifyPackage(zoneConfig, toPkg, in.DomainMap)
		if from.Zone == zones.Unknown || to.Zone == zones.Unknown {
			continue
		}
		violation := classifyViolation(zoneConfig, from, to)
		if violation.kind == "" {
			continue
		}
		out = append(out, conflicts.Conflict{
			Scenario:  in.Scenario,
			Detector:  d.Name(),
			Type:      "layering",
			Subtype:   violation.kind,
			Severity:  d.severityFor(violation.blockerEligible),
			Locations: []string{from.Path, to.Path},
			Domains:   domainList(from.Domain, to.Domain),
			Evidence: []conflicts.Evidence{
				{
					Kind:    "import_edge",
					Summary: fmt.Sprintf("%s imports %s", from.Path, to.Path),
					Locator: fmt.Sprintf("%s -> %s", from.Path, to.Path),
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

type packageZone = zones.Info

type violation struct {
	kind            string
	summary         string
	fix             string
	blockerEligible bool
}

func classifyViolation(zoneConfig zones.Config, from, to packageZone) violation {
	if from.Zone == zones.CompositionRoot || to.Zone == zones.CompositionRoot {
		return violation{}
	}
	for _, rule := range []func(zones.Config, packageZone, packageZone) violation{
		domainImportsTransport,
		substrateImportsProduct,
		domainImportsSiblingDomain,
		handlerImportsSiblingDomain,
	} {
		if v := rule(zoneConfig, from, to); v.kind != "" {
			return v
		}
	}
	return violation{}
}

func domainImportsTransport(_ zones.Config, from, to packageZone) violation {
	if from.Zone == zones.Domain && to.Zone == zones.Transport {
		return violation{
			kind:            "domain-imports-transport",
			summary:         "domain packages must not import transport handlers",
			fix:             "Invert the dependency: keep transport translation in handlers and pass domain-owned interfaces into the domain package.",
			blockerEligible: true,
		}
	}
	return violation{}
}

func substrateImportsProduct(_ zones.Config, from, to packageZone) violation {
	if from.Zone == zones.Substrate && (to.Zone == zones.Domain || to.Zone == zones.Transport) {
		return violation{
			kind:            "substrate-imports-product",
			summary:         "shared substrate must stay business-vocabulary-free and must not import product domains or handlers",
			fix:             "Move product vocabulary behind an interface owned by the consuming domain, or move the code out of shared substrate.",
			blockerEligible: true,
		}
	}
	return violation{}
}

func domainImportsSiblingDomain(zoneConfig zones.Config, from, to packageZone) violation {
	if from.Zone == zones.Domain && to.Zone == zones.Domain && from.Domain != "" && to.Domain != "" && from.Domain != to.Domain && !zoneConfig.MayCoordinate(from.Archetype) {
		if !from.Declared {
			return unownedSourcePath(zoneConfig, from, to)
		}
		return violation{
			kind:    "domain-imports-sibling-domain",
			summary: "a non-coordinating domain package must not import sibling domain internals",
			fix: fmt.Sprintf("Either DECLARE %q's archetype as a coordinating role (%s) in DOMAINS.md if it legitimately composes other domains, "+
				"or INVERT the dependency: introduce a narrow interface owned by %q and wire the concrete sibling from the composition root.",
				from.Domain, zoneConfig.CoordinatingVocabulary(), from.Domain),
			blockerEligible: false,
		}
	}
	return violation{}
}

func handlerImportsSiblingDomain(zoneConfig zones.Config, from, to packageZone) violation {
	if from.Zone == zones.Transport && to.Zone == zones.Domain && from.Domain != "" && to.Domain != "" && from.Domain != to.Domain && !zoneConfig.MayCoordinate(from.Archetype) {
		if !from.Declared {
			return unownedSourcePath(zoneConfig, from, to)
		}
		return violation{
			kind:    "handler-imports-sibling-domain",
			summary: "a handler package should translate requests for its own domain, not reach into sibling domains",
			fix: fmt.Sprintf("Either DECLARE %q's archetype as a coordinating role (%s) in DOMAINS.md if it is an aggregator/provider that composes other domains, "+
				"or INVERT the dependency: move cross-domain coordination behind an interface owned by %q (wired from the composition root).",
				from.Domain, zoneConfig.CoordinatingVocabulary(), from.Domain),
			blockerEligible: false,
		}
	}
	return violation{}
}

// unownedSourcePath is returned when an importing package sits in a domain-ish
// zone but its path is owned by no declared domain (the domain was inferred from
// the path segment). Rather than emit a misleading "imports sibling domain"
// finding against a phantom domain, point the operator at the DOMAINS.md gap.
func unownedSourcePath(zoneConfig zones.Config, from, to packageZone) violation {
	return violation{
		kind:    "unowned-source-path",
		summary: "this source path is owned by no declared domain, so its cross-domain import cannot be governed",
		fix: fmt.Sprintf("Declare %q in DOMAINS.md: add it to the Source Paths of the domain that owns it (or a new domain), "+
			"giving it an archetype — a coordinating role (%s) if it legitimately composes %q.",
			from.Path, zoneConfig.CoordinatingVocabulary(), to.Domain),
		blockerEligible: false,
	}
}

func classifyPackage(zoneConfig zones.Config, pkg graph.PackageNode, m domains.DerivedDomainMap) packageZone {
	return zoneConfig.Classify(pkg.RepoPath, m)
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
