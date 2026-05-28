// Package cycle is the import-cycle detector. Tarjan SCC over the
// package import graph, with each non-trivial SCC emitted as one
// conflict.
//
// Sub-classification (encoded in Conflict.Subtype) per requirements
// REQ-P0-005:
//   - "type-only"     — every edge in the cycle is types-only (best
//     break: lift the shared type into a leaf
//     package).
//   - "junk-drawer"   — at least one package in the SCC is imported by
//     many otherwise-unrelated packages (smell of a
//     "utils" or "common" drawer).
//   - "cross-domain"  — packages in the SCC belong to >1 domain in
//     the derived domain map.
//   - "within-domain" — packages in the SCC all belong to the same
//     derived domain; usually a small refactor.
package cycle

import (
	"context"
	"fmt"
	"sort"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// Detector is the production cycle detector.
type Detector struct{}

// New returns the production cycle detector.
func New() *Detector { return &Detector{} }

func (Detector) Name() string { return "cycle" }
func (Detector) Description() string {
	return "Detects strongly-connected components in the package import graph."
}

func (Detector) EmitsTypes() []string {
	return []string{"cycle"}
}

func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	sccs := tarjan(in.Snapshot)
	if len(sccs) == 0 {
		return nil, nil
	}

	out := make([]conflicts.Conflict, 0, len(sccs))
	for _, scc := range sccs {
		if len(scc) < 2 {
			// Trivial: self-loops are not cycles in v0.1's semantics.
			continue
		}
		subtype := classify(scc, in.Snapshot, in.DomainMap)
		evidence := makeEvidence(scc, in.Snapshot)
		c := conflicts.Conflict{
			Scenario:  in.Scenario,
			Detector:  d.Name(),
			Type:      "cycle",
			Subtype:   subtype,
			Severity:  conflicts.SeverityError,
			Locations: locationsForPackages(scc, in.Snapshot),
			Domains:   domainsForPackages(scc, in.Snapshot, in.DomainMap),
			Evidence:  evidence,
			SuggestedFixes: []conflicts.Fix{
				{
					ID:       "fix:break-cycle",
					Kind:     conflicts.FixKindBreakCycle,
					Resolver: "break_cycle",
					Summary:  fmt.Sprintf("break %s cycle across %d packages", subtype, len(scc)),
				},
			},
			Status: conflicts.ResolutionStatusDetected,
		}
		out = append(out, c)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Subtype != out[j].Subtype {
			return out[i].Subtype < out[j].Subtype
		}
		return joined(out[i].Locations) < joined(out[j].Locations)
	})
	return out, nil
}

// tarjan returns the non-trivial strongly-connected components of the
// package-level import graph. Each component is a slice of package ids.
func tarjan(snap graph.GraphSnapshot) [][]string {
	// Build adjacency over internal packages only.
	internalIDs := make(map[string]struct{}, len(snap.Packages))
	for _, p := range snap.Packages {
		if p.Internal {
			internalIDs[p.ID] = struct{}{}
		}
	}
	adj := make(map[string][]string, len(internalIDs))
	for _, e := range snap.Imports {
		from := edgePackageID(e.From, snap)
		if from == "" {
			continue
		}
		if _, ok := internalIDs[from]; !ok {
			continue
		}
		if _, ok := internalIDs[e.ToPackageID]; !ok {
			continue
		}
		adj[from] = append(adj[from], e.ToPackageID)
	}
	// Stable order for determinism.
	for k := range adj {
		sort.Strings(adj[k])
	}

	ids := make([]string, 0, len(internalIDs))
	for id := range internalIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var (
		index    int
		stack    []string
		onStack  = make(map[string]bool, len(ids))
		indices  = make(map[string]int, len(ids))
		lowlinks = make(map[string]int, len(ids))
		sccs     [][]string
	)
	var strongconnect func(v string)
	strongconnect = func(v string) {
		indices[v] = index
		lowlinks[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if _, seen := indices[w]; !seen {
				strongconnect(w)
				if lowlinks[w] < lowlinks[v] {
					lowlinks[v] = lowlinks[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlinks[v] {
					lowlinks[v] = indices[w]
				}
			}
		}
		if lowlinks[v] == indices[v] {
			var comp []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			sort.Strings(comp)
			sccs = append(sccs, comp)
		}
	}
	for _, v := range ids {
		if _, seen := indices[v]; !seen {
			strongconnect(v)
		}
	}

	sort.SliceStable(sccs, func(i, j int) bool {
		return sccs[i][0] < sccs[j][0]
	})
	return sccs
}

func edgePackageID(from string, snap graph.GraphSnapshot) string {
	// Edges from packages point directly to packages. Edges from files
	// resolve to the file's package id.
	for _, f := range snap.Files {
		if f.ID == from {
			return f.PackageID
		}
	}
	for _, p := range snap.Packages {
		if p.ID == from {
			return p.ID
		}
	}
	return ""
}

func classify(scc []string, snap graph.GraphSnapshot, m domains.DerivedDomainMap) string {
	doms := domainSet(scc, snap, m)
	if len(doms) > 1 {
		return "cross-domain"
	}
	if onlyTypeOnlyEdges(scc, snap) {
		return "type-only"
	}
	if isJunkDrawer(scc, snap) {
		return "junk-drawer"
	}
	return "within-domain"
}

func domainSet(scc []string, snap graph.GraphSnapshot, m domains.DerivedDomainMap) map[string]struct{} {
	out := make(map[string]struct{})
	for _, pid := range scc {
		dir := packageDir(pid, snap)
		if dir == "" {
			continue
		}
		dom := m.DomainFor(dir)
		if dom == "" {
			continue
		}
		out[dom] = struct{}{}
	}
	return out
}

func packageDir(pid string, snap graph.GraphSnapshot) string {
	for _, p := range snap.Packages {
		if p.ID == pid {
			return p.Directory
		}
	}
	return ""
}

func onlyTypeOnlyEdges(scc []string, snap graph.GraphSnapshot) bool {
	memberOf := make(map[string]struct{}, len(scc))
	for _, p := range scc {
		memberOf[p] = struct{}{}
	}
	internal := false
	for _, e := range snap.Imports {
		from := edgePackageID(e.From, snap)
		if _, ok := memberOf[from]; !ok {
			continue
		}
		if _, ok := memberOf[e.ToPackageID]; !ok {
			continue
		}
		internal = true
		// If any cycle-internal edge carries non-type symbol ids, it's
		// not type-only. v0.1 treats any symbol_ids presence as "non
		// type-only" because the snapshot doesn't yet distinguish
		// symbol kinds per edge. Phase 5 refines this.
		if len(e.SymbolIDs) > 0 {
			return false
		}
	}
	return internal
}

// isJunkDrawer flags a cycle that includes a "junk drawer" — a package
// imported by many otherwise-unrelated packages. The heuristic counts
// inbound edges per package in the SCC; if any single package has
// >= junkDrawerThreshold importers (across the whole snapshot), it's a
// drawer.
const junkDrawerThreshold = 5

func isJunkDrawer(scc []string, snap graph.GraphSnapshot) bool {
	memberOf := make(map[string]struct{}, len(scc))
	for _, p := range scc {
		memberOf[p] = struct{}{}
	}
	inDegree := make(map[string]int, len(scc))
	for _, e := range snap.Imports {
		if _, ok := memberOf[e.ToPackageID]; !ok {
			continue
		}
		inDegree[e.ToPackageID]++
	}
	for _, d := range inDegree {
		if d >= junkDrawerThreshold {
			return true
		}
	}
	return false
}

func locationsForPackages(scc []string, snap graph.GraphSnapshot) []string {
	out := make([]string, 0, len(scc))
	for _, pid := range scc {
		dir := packageDir(pid, snap)
		if dir == "" {
			out = append(out, pid)
			continue
		}
		out = append(out, dir)
	}
	sort.Strings(out)
	return out
}

func domainsForPackages(scc []string, snap graph.GraphSnapshot, m domains.DerivedDomainMap) []string {
	set := domainSet(scc, snap, m)
	out := make([]string, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

func makeEvidence(scc []string, snap graph.GraphSnapshot) []conflicts.Evidence {
	memberOf := make(map[string]struct{}, len(scc))
	for _, p := range scc {
		memberOf[p] = struct{}{}
	}
	var edges []string
	for _, e := range snap.Imports {
		from := edgePackageID(e.From, snap)
		if _, ok := memberOf[from]; !ok {
			continue
		}
		if _, ok := memberOf[e.ToPackageID]; !ok {
			continue
		}
		edges = append(edges, fmt.Sprintf("%s -> %s", from, e.ToPackageID))
	}
	sort.Strings(edges)
	out := []conflicts.Evidence{{
		Kind:    "scc_member",
		Summary: fmt.Sprintf("%d packages form a strongly-connected component", len(scc)),
		Locator: joined(scc),
	}}
	for _, e := range edges {
		out = append(out, conflicts.Evidence{
			Kind:    "import_edge",
			Summary: e,
			Locator: e,
		})
	}
	return out
}

func joined(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}
