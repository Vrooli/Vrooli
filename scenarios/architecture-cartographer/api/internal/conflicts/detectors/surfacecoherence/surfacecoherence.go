// Package surfacecoherence detects domain surfaces that were declared or
// strongly implied but have no implementation evidence.
package surfacecoherence

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
)

const (
	surfaceAPI = "api"
	surfaceCLI = "cli"
	surfaceUI  = "ui"
)

// Detector flags expected API/CLI/UI domain surfaces that are missing from
// the actual graph or lower-rung domain declarations. It complements
// convergence_drift by using per-domain source paths and archetype intent
// instead of only comparing source-level name sets.
type Detector struct{}

// New returns the production detector.
func New() *Detector { return &Detector{} }

func (Detector) Name() string { return "surface_coherence" }

func (Detector) Description() string {
	return "Flags domains whose declared or implied surfaces are missing implementation evidence."
}

func (Detector) EmitsTypes() []string { return []string{"surface_coherence"} }

func (Detector) Class() conflicts.FindingClass {
	return conflicts.FindingClassDeterministic
}

func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	actual := actualSurfaces(in)
	active := activeSurfaces(in)
	var out []conflicts.Conflict
	for _, domain := range in.DomainMap.Domains {
		expected := expectedSurfaces(domain, active)
		for surface, reason := range expected {
			if _, ok := active[surface]; !ok {
				continue
			}
			if actual[domain.Name][surface] {
				continue
			}
			out = append(out, conflictFor(in.Scenario, domain.Name, surface, reason))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Domains[0] != out[j].Domains[0] {
			return out[i].Domains[0] < out[j].Domains[0]
		}
		return out[i].Subtype < out[j].Subtype
	})
	return out, nil
}

func actualSurfaces(in conflicts.DetectInput) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	add := func(domain, surface string) {
		if domain == "" || surface == "" {
			return
		}
		if out[domain] == nil {
			out[domain] = map[string]bool{}
		}
		out[domain][surface] = true
	}
	for _, decl := range in.DomainMap.Declarations {
		surface := surfaceForSource(decl.Source)
		for _, name := range decl.DomainNames {
			add(name, surface)
		}
	}
	for _, f := range in.Snapshot.Files {
		add(in.DomainMap.DomainFor(f.Path), surfaceForPath(f.Path))
	}
	for _, p := range in.Snapshot.Packages {
		add(in.DomainMap.DomainFor(p.RepoPath), surfaceForPath(p.RepoPath))
	}
	return out
}

func activeSurfaces(in conflicts.DetectInput) map[string]struct{} {
	out := map[string]struct{}{}
	for _, decl := range in.DomainMap.Declarations {
		if surface := surfaceForSource(decl.Source); surface != "" && len(decl.DomainNames) > 0 {
			out[surface] = struct{}{}
		}
	}
	for _, f := range in.Snapshot.Files {
		if surface := surfaceForPath(f.Path); surface != "" {
			out[surface] = struct{}{}
		}
	}
	for _, p := range in.Snapshot.Packages {
		if surface := surfaceForPath(p.RepoPath); surface != "" {
			out[surface] = struct{}{}
		}
	}
	return out
}

func expectedSurfaces(domain domains.DerivedDomain, active map[string]struct{}) map[string]string {
	out := map[string]string{}
	for _, path := range domain.Paths {
		if surface := surfaceForPath(path); surface != "" {
			out[surface] = "declared source path"
		}
	}
	for _, surface := range impliedSurfaces(domain.Archetype) {
		if _, ok := active[surface]; ok {
			if _, declared := out[surface]; !declared {
				out[surface] = "archetype " + strings.TrimSpace(domain.Archetype)
			}
		}
	}
	return out
}

func impliedSurfaces(archetype string) []string {
	archetype = strings.ToLower(strings.TrimSpace(archetype))
	if i := strings.Index(archetype, "/"); i >= 0 {
		archetype = strings.TrimSpace(archetype[:i])
	}
	switch archetype {
	case "service", "orchestration", "validation", "mutation", "reporting":
		return []string{surfaceAPI}
	default:
		return nil
	}
}

func conflictFor(scenario, domain, surface, reason string) conflicts.Conflict {
	location := fmt.Sprintf("%s:%s", domain, surface)
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  "surface_coherence",
		Type:      "surface_coherence",
		Subtype:   "missing_" + surface,
		Severity:  conflicts.SeverityWarn,
		Locations: []string{location},
		Domains:   []string{domain},
		Evidence: []conflicts.Evidence{{
			Kind:    "surface_expectation",
			Summary: fmt.Sprintf("domain %q expects a %s surface from %s but no implementation evidence was found", domain, surface, reason),
			Locator: location,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    fmt.Sprintf("Add %s implementation for domain %q, or remove that surface expectation from DOMAINS.md.", surface, domain),
			Confidence: 0.6,
		}},
	}
}

func surfaceForSource(src domains.Source) string {
	switch src {
	case domains.SourceAPIFolders:
		return surfaceAPI
	case domains.SourceCLIGroups:
		return surfaceCLI
	case domains.SourceUIFeatures:
		return surfaceUI
	default:
		return ""
	}
}

func surfaceForPath(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	switch {
	case strings.HasPrefix(path, "api/"):
		return surfaceAPI
	case strings.HasPrefix(path, "cli/"):
		return surfaceCLI
	case strings.HasPrefix(path, "ui/"):
		return surfaceUI
	default:
		return ""
	}
}

var _ conflicts.Detector = (*Detector)(nil)
