package graph

import (
	"sort"
	"strings"

	"architecture-cartographer/internal/archetype"
	"architecture-cartographer/internal/domains"
)

// Archetype-signal extraction lives with the graph package because it is pure
// graph-snapshot analysis: it turns graph evidence (packages, files, import
// edges) into an archetype.Input. Keeping it here gives the InferArchetype RPC
// and the archetype-convergence detector one shared derivation and avoids a
// standalone helper folder that would import product domains from substrate.

// importMarkers map a substring of an outbound import path to the boolean
// archetype.Input signal it sets. Substrings are intentionally specific so a
// match is meaningful evidence.
var (
	archetypeStorageMarkers  = []string{"sqlite", "/database", "/store", "/repository", "database/sql"}
	archetypeMutationMarkers = []string{"go/ast", "astutil", "/dst", "/apply", "go/format", "go/printer"}
	archetypeWorkflowMarkers = []string{"temporal", "workflow", "statemachine", "/saga"}
	archetypeExternalMarkers = []string{"/httpc", "/httpx", "outbound", "/integration"}
)

// BuildArchetypeSignals derives the archetype signals for one domain from the
// snapshot.
func BuildArchetypeSignals(snap GraphSnapshot, domainName string, paths []string) archetype.Input {
	in := archetype.Input{Name: domainName, Paths: append([]string(nil), paths...)}

	importPathByID := make(map[string]string, len(snap.Packages))
	for _, p := range snap.Packages {
		importPathByID[p.ID] = p.ImportPath
	}
	pkgByFile := make(map[string]string, len(snap.Files))
	filePathByID := make(map[string]string, len(snap.Files))
	for _, f := range snap.Files {
		pkgByFile[f.ID] = f.PackageID
		filePathByID[f.ID] = f.Path
	}

	// Which packages belong to this domain (RepoPath under a declared path).
	ownPkg := map[string]bool{}
	for _, p := range snap.Packages {
		rp := strings.Trim(strings.TrimSpace(p.RepoPath), "/")
		if rp == "" {
			continue
		}
		if archetypeSignalPathOwned(rp, paths) {
			ownPkg[p.ID] = true
			if strings.HasPrefix(rp, "api/handlers/") {
				in.HasHTTPHandler = true
				for _, file := range snap.Files {
					if file.PackageID == p.ID {
						in.HTTPHandlerPaths = append(in.HTTPHandlerPaths, file.Path)
					}
				}
			}
		}
	}

	// Outbound imports of the domain's packages → substring signals.
	for _, e := range snap.Imports {
		fromPkg := pkgByFile[e.From]
		if !ownPkg[fromPkg] {
			continue
		}
		target := strings.ToLower(importPathByID[e.ToPackageID])
		if target == "" {
			continue
		}
		if archetypeSignalAnySubstring(target, archetypeStorageMarkers) {
			in.OwnsStorage = true
			in.StoragePaths = append(in.StoragePaths, filePathByID[e.From])
		}
		if archetypeSignalAnySubstring(target, archetypeMutationMarkers) {
			in.WritesFiles = true
			in.MutationPaths = append(in.MutationPaths, filePathByID[e.From])
		}
		if archetypeSignalAnySubstring(target, archetypeWorkflowMarkers) {
			in.HasWorkflow = true
			in.WorkflowPaths = append(in.WorkflowPaths, filePathByID[e.From])
		}
		if archetypeSignalAnySubstring(target, archetypeExternalMarkers) {
			in.CallsExternal = true
			in.ExternalPaths = append(in.ExternalPaths, filePathByID[e.From])
		}
	}
	in.HTTPHandlerPaths = uniqueSortedPaths(in.HTTPHandlerPaths)
	in.StoragePaths = uniqueSortedPaths(in.StoragePaths)
	in.MutationPaths = uniqueSortedPaths(in.MutationPaths)
	in.WorkflowPaths = uniqueSortedPaths(in.WorkflowPaths)
	in.ExternalPaths = uniqueSortedPaths(in.ExternalPaths)
	return in
}

func uniqueSortedPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// BuildArchetypeSignalsForDomain looks up the domain's declared paths in the map
// and builds its signals. Returns false when the domain is not in the map.
func BuildArchetypeSignalsForDomain(snap GraphSnapshot, dm domains.DerivedDomainMap, domainName string) (archetype.Input, bool) {
	for _, d := range dm.Domains {
		if d.Name == domainName {
			return BuildArchetypeSignals(snap, d.Name, d.Paths), true
		}
	}
	return archetype.Input{}, false
}

func archetypeSignalPathOwned(repoPath string, paths []string) bool {
	for _, p := range paths {
		p = strings.Trim(strings.TrimSpace(p), "/")
		if p == "" {
			continue
		}
		if repoPath == p || strings.HasPrefix(repoPath, p+"/") || strings.HasPrefix(p, repoPath+"/") {
			return true
		}
	}
	return false
}

func archetypeSignalAnySubstring(s string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}
