// Package crossscenario detects direct imports into another scenario's
// internal implementation packages.
package crossscenario

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/graph"
)

// Detector flags cross-scenario internal imports. Scenarios may consume each
// other through published APIs, CLIs, or shared packages; importing
// scenarios/<other>/api/internal/... couples to another scenario's private
// implementation boundary.
type Detector struct{}

// New returns the production detector.
func New() *Detector { return &Detector{} }

func (Detector) Name() string { return "cross_scenario" }

func (Detector) Description() string {
	return "Flags imports into another scenario's private api/internal packages."
}

func (Detector) EmitsTypes() []string { return []string{"cross_scenario"} }

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
		targetScenario, targetPath, ok := internalScenarioImport(toPkg.ImportPath)
		if !ok {
			targetScenario, targetPath, ok = internalScenarioImport(toPkg.RepoPath)
		}
		if !ok || targetScenario == "" || targetScenario == in.Scenario {
			continue
		}
		out = append(out, conflicts.Conflict{
			Scenario:  in.Scenario,
			Detector:  d.Name(),
			Type:      "cross_scenario",
			Subtype:   "internal_import",
			Severity:  conflicts.SeverityError,
			Locations: []string{fromPkg.RepoPath, firstNonEmpty(toPkg.ImportPath, toPkg.RepoPath)},
			Domains:   domainList(in.DomainMap.DomainFor(fromPkg.RepoPath)),
			Evidence: []conflicts.Evidence{{
				Kind:    "import_edge",
				Summary: fmt.Sprintf("%s imports private implementation from scenario %q", fromPkg.RepoPath, targetScenario),
				Locator: fmt.Sprintf("%s -> %s", fromPkg.RepoPath, targetPath),
			}},
			SuggestedFixes: []conflicts.Fix{{
				Kind:       conflicts.FixKindAddDependency,
				Summary:    fmt.Sprintf("Replace the private import with %s's public API, CLI, or a shared package contract.", targetScenario),
				Confidence: 0.75,
			}},
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.Join(out[i].Locations, "\x00") < strings.Join(out[j].Locations, "\x00")
	})
	return out, nil
}

func internalScenarioImport(path string) (scenario string, targetPath string, ok bool) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return "", "", false
	}
	parts := strings.Split(path, "/")
	for i := 0; i+3 < len(parts); i++ {
		if parts[i] != "scenarios" || parts[i+2] != "api" || parts[i+3] != "internal" {
			continue
		}
		return parts[i+1], strings.Join(parts[i:], "/"), true
	}
	return "", "", false
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

func domainList(domain string) []string {
	if domain == "" {
		return nil
	}
	return []string{domain}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

var _ conflicts.Detector = (*Detector)(nil)
