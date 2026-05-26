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
	"strings"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

// Signal is the production import-cluster signal.
type Signal struct{}

// New returns the production signal.
func New() *Signal { return &Signal{} }

func (Signal) Name() string                               { return "import-cluster" }
func (Signal) DefaultWeight() float64                     { return 1.0 }
func (Signal) IsAvailable(context.Context) (bool, string) { return true, "" }

func (Signal) Score(_ context.Context, gctx signals.GraphContext, chunk graph.Chunk) []signals.Score {
	if chunk.FileID == "" {
		return nil
	}
	pkgID := packageForFile(chunk.FileID, gctx.Snapshot)
	if pkgID == "" {
		return nil
	}
	clusters := computeClusters(gctx)
	clusterID, ok := clusters[pkgID]
	if !ok {
		return nil
	}

	// Find every package in the same cluster and tally their domains.
	domainFor := indexDomainPackages(gctx)
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
		return nil
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
			Signal: "import-cluster",
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
	return out
}

func packageForFile(fileID string, snap graph.GraphSnapshot) string {
	for _, f := range snap.Files {
		if f.ID == fileID {
			return f.PackageID
		}
	}
	return ""
}

// computeClusters returns a per-internal-package cluster id derived
// from undirected connected components over the import edges.
// Cached on GraphContext.Caches.Community so subsequent calls in the
// same scoring batch share the work.
func computeClusters(gctx signals.GraphContext) map[string]int {
	if gctx.Caches != nil && len(gctx.Caches.Community) > 0 {
		return gctx.Caches.Community
	}
	internal := make(map[string]struct{})
	for _, p := range gctx.Snapshot.Packages {
		if p.Internal {
			internal[p.ID] = struct{}{}
		}
	}
	adj := make(map[string]map[string]struct{}, len(internal))
	for k := range internal {
		adj[k] = make(map[string]struct{})
	}
	for _, e := range gctx.Snapshot.Imports {
		from := resolvePkg(e.From, gctx.Snapshot)
		if from == "" {
			continue
		}
		if _, ok := internal[from]; !ok {
			continue
		}
		if _, ok := internal[e.ToPackageID]; !ok {
			continue
		}
		adj[from][e.ToPackageID] = struct{}{}
		adj[e.ToPackageID][from] = struct{}{}
	}

	cluster := make(map[string]int, len(internal))
	visited := make(map[string]bool, len(internal))
	order := make([]string, 0, len(internal))
	for id := range internal {
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
		for k, v := range cluster {
			gctx.Caches.Community[k] = v
		}
	}
	return cluster
}

func resolvePkg(id string, snap graph.GraphSnapshot) string {
	for _, f := range snap.Files {
		if f.ID == id {
			return f.PackageID
		}
	}
	for _, p := range snap.Packages {
		if p.ID == id {
			return p.ID
		}
	}
	return ""
}

func indexDomainPackages(gctx signals.GraphContext) map[string]string {
	out := make(map[string]string, len(gctx.Snapshot.Packages))
	for _, p := range gctx.Snapshot.Packages {
		if p.Directory == "" {
			continue
		}
		for _, d := range gctx.Manifest.Domains {
			for _, g := range d.Paths {
				if matches(p.Directory, g) {
					out[p.ID] = d.Name
					break
				}
			}
		}
	}
	return out
}

func matches(path, glob string) bool {
	switch {
	case glob == "**":
		return true
	case strings.HasSuffix(glob, "/**"):
		prefix := glob[:len(glob)-3]
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	default:
		return path == glob
	}
}

func clusterIDLabel(id int) string {
	return fmt.Sprintf("cluster-%d", id)
}
