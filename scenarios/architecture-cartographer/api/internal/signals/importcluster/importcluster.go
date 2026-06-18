// Package importcluster is the import-cluster signal. Groups packages
// into clusters via a simple connected-components walk over the
// internal import graph, then scores domains based on the dominant
// declared domain inside the cluster the chunk belongs to.
//
// v0.1 ships a deterministic connected-components implementation —
// adequate for the day-one signal. Louvain modularity (the production
// algorithm called out in the plan) is a later upgrade once a stable
// gonum-free implementation is vetted. Cache lookup is on GraphContext.
//
// Default weight 1.0 per SIGNAL_LADDER.md.
package importcluster

import (
	"context"
	"fmt"
	"sort"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/graphindex"
)

const name = "import-cluster"

// Signal is the production import-cluster signal.
type Signal struct{}

// New returns the production signal.
func New() *Signal { return &Signal{} }

func (Signal) Name() string                               { return name }
func (Signal) DefaultWeight() float64                     { return 1.0 }
func (Signal) IsAvailable(context.Context) (bool, string) { return true, "" }

func (Signal) Score(_ context.Context, gctx signals.GraphContext, chunk graph.Chunk) signals.ScoreResult {
	if chunk.FileID == "" {
		return signals.Abstain(name, "chunk has no file id", chunk.Path)
	}
	pkgID := graphindex.PackageForFile(chunk.FileID, gctx.Snapshot)
	if pkgID == "" {
		return signals.Abstain(name, "file has no package in snapshot", chunk.Path)
	}
	clusters := computeClusters(gctx)
	clusterID, ok := clusters[pkgID]
	if !ok {
		return signals.Abstain(name, "package is not in any import cluster", chunk.Path)
	}

	// Find every package in the same cluster and tally their domains.
	domainFor := graphindex.DomainPackages(gctx)
	tally := make(map[string]int)
	total := 0
	for pkg, cid := range clusters {
		if cid != clusterID {
			continue
		}
		dom := domainFor[pkg]
		if dom == "" {
			continue
		}
		tally[dom]++
		total++
	}
	if total == 0 {
		return signals.Abstain(name, "no packages in this cluster are mapped to a derived domain", chunk.Path)
	}

	// Stable iteration order for evidence determinism.
	domains := make([]string, 0, len(tally))
	for d := range tally {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	var out []signals.Score
	for _, dom := range domains {
		count := tally[dom]
		value := float64(count) / float64(total)
		out = append(out, signals.Score{
			Signal: name,
			Domain: dom,
			Value:  value,
			Reason: fmt.Sprintf("%d/%d cluster packages belong to %q", count, total, dom),
			Evidence: []signals.Evidence{{
				Kind:    "import_cluster",
				Summary: fmt.Sprintf("cluster %s: %s majority", clusterIDLabel(clusterID), dom),
				Locator: chunk.Path,
				Weight:  value,
			}},
		})
	}
	return signals.ScoreResult{Scores: out}
}

// computeClusters returns a per-internal-package cluster id derived
// from undirected connected components over the import edges.
// Cached on GraphContext.Caches so subsequent calls in the same
// scoring batch share the work; access is goroutine-safe.
func computeClusters(gctx signals.GraphContext) map[string]int {
	if gctx.Caches != nil {
		if cached := gctx.Caches.CommunitySnapshot(); cached != nil {
			return cached
		}
	}
	inScenario := make(map[string]struct{})
	for _, p := range gctx.Snapshot.Packages {
		inScenario[p.ID] = struct{}{}
	}
	adj := make(map[string]map[string]struct{}, len(inScenario))
	for k := range inScenario {
		adj[k] = make(map[string]struct{})
	}
	for _, e := range gctx.Snapshot.Imports {
		from := graphindex.PackageFor(e.From, gctx.Snapshot)
		if from == "" {
			continue
		}
		if _, ok := inScenario[from]; !ok {
			continue
		}
		if _, ok := inScenario[e.ToPackageID]; !ok {
			continue
		}
		adj[from][e.ToPackageID] = struct{}{}
		adj[e.ToPackageID][from] = struct{}{}
	}

	cluster := make(map[string]int, len(inScenario))
	visited := make(map[string]bool, len(inScenario))
	order := make([]string, 0, len(inScenario))
	for id := range inScenario {
		order = append(order, id)
	}
	sort.Strings(order)
	next := 0
	for _, root := range order {
		if visited[root] {
			continue
		}
		next++
		stack := []string{root}
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if visited[n] {
				continue
			}
			visited[n] = true
			cluster[n] = next
			neighbors := make([]string, 0, len(adj[n]))
			for nb := range adj[n] {
				neighbors = append(neighbors, nb)
			}
			sort.Strings(neighbors)
			for _, nb := range neighbors {
				if !visited[nb] {
					stack = append(stack, nb)
				}
			}
		}
	}
	if gctx.Caches != nil {
		gctx.Caches.SetCommunity(cluster)
	}
	return cluster
}

func clusterIDLabel(id int) string {
	return fmt.Sprintf("cluster-%d", id)
}
