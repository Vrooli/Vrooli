// Package importcluster is the import-cluster signal. Groups packages
// into deterministic Louvain modularity communities over the internal
// import graph, then scores domains based on the dominant declared
// domain inside the community the chunk belongs to.
//
// Default weight 1.0 per SIGNAL_LADDER.md.
package importcluster

import (
	"context"
	"fmt"
	"math"
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

const modularityEpsilon = 1e-9

// computeClusters returns a per-internal-package cluster id derived
// from deterministic Louvain modularity communities over the import graph.
// Cached on GraphContext.Caches so subsequent calls in the same
// scoring batch share the work; access is goroutine-safe.
func computeClusters(gctx signals.GraphContext) map[string]int {
	if gctx.Caches != nil {
		if cached := gctx.Caches.CommunitySnapshot(); cached != nil {
			return cached
		}
	}
	graph := newWeightedGraph()
	inScenario := make(map[string]struct{}, len(gctx.Snapshot.Packages))
	for _, p := range gctx.Snapshot.Packages {
		inScenario[p.ID] = struct{}{}
		graph.addNode(p.ID)
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
		graph.addEdge(from, e.ToPackageID, 1)
	}

	cluster := louvainCommunities(graph)
	if gctx.Caches != nil {
		gctx.Caches.SetCommunity(cluster)
	}
	return cluster
}

func clusterIDLabel(id int) string {
	return fmt.Sprintf("cluster-%d", id)
}

type weightedGraph struct {
	nodes []string
	adj   map[string]map[string]float64
}

func newWeightedGraph() weightedGraph {
	return weightedGraph{adj: make(map[string]map[string]float64)}
}

func (g *weightedGraph) addNode(node string) {
	if _, ok := g.adj[node]; ok {
		return
	}
	g.adj[node] = make(map[string]float64)
	g.nodes = append(g.nodes, node)
	sort.Strings(g.nodes)
}

func (g *weightedGraph) addEdge(a, b string, weight float64) {
	g.addNode(a)
	g.addNode(b)
	if weight <= 0 {
		return
	}
	if a == b {
		g.adj[a][a] += 2 * weight
		return
	}
	g.adj[a][b] += weight
	g.adj[b][a] += weight
}

func (g weightedGraph) edgeWeight(a, b string) float64 {
	if neighbors, ok := g.adj[a]; ok {
		return neighbors[b]
	}
	return 0
}

func (g weightedGraph) totalEdgeWeight() float64 {
	var total float64
	for _, a := range g.nodes {
		for _, weight := range g.adj[a] {
			total += weight
		}
	}
	return total / 2
}

func louvainCommunities(g weightedGraph) map[string]int {
	if len(g.nodes) == 0 {
		return map[string]int{}
	}
	if g.totalEdgeWeight() == 0 {
		return singletonCommunities(g.nodes)
	}

	originalToNode := make(map[string]string, len(g.nodes))
	for _, node := range g.nodes {
		originalToNode[node] = node
	}

	current := g
	previousModularity := math.Inf(-1)
	for {
		partition := optimizePartition(current)
		for original, node := range originalToNode {
			originalToNode[original] = partition[node]
		}

		modularity := modularity(current, partition)
		if modularity <= previousModularity+modularityEpsilon {
			break
		}
		previousModularity = modularity

		next := aggregateGraph(current, partition)
		if len(next.nodes) == len(current.nodes) {
			break
		}
		current = next
	}

	return normalizeCommunities(originalToNode)
}

func optimizePartition(g weightedGraph) map[string]string {
	partition := make(map[string]string, len(g.nodes))
	for _, node := range g.nodes {
		partition[node] = node
	}

	for improved := true; improved; {
		improved = false
		for _, node := range g.nodes {
			bestCommunity := partition[node]
			bestModularity := modularity(g, partition)
			for _, community := range candidateCommunities(g, partition, node) {
				if community == partition[node] {
					continue
				}
				trial := clonePartition(partition)
				trial[node] = community
				score := modularity(g, trial)
				if betterCommunity(score, bestModularity, community, bestCommunity) {
					bestCommunity = community
					bestModularity = score
				}
			}
			if bestCommunity != partition[node] {
				partition[node] = bestCommunity
				improved = true
			}
		}
	}
	return partition
}

func candidateCommunities(g weightedGraph, partition map[string]string, node string) []string {
	seen := map[string]struct{}{partition[node]: {}}
	for neighbor := range g.adj[node] {
		seen[partition[neighbor]] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for community := range seen {
		out = append(out, community)
	}
	sort.Strings(out)
	return out
}

func betterCommunity(score, bestScore float64, community, bestCommunity string) bool {
	if score > bestScore+modularityEpsilon {
		return true
	}
	return math.Abs(score-bestScore) <= modularityEpsilon && community < bestCommunity
}

func modularity(g weightedGraph, partition map[string]string) float64 {
	totalWeight := g.totalEdgeWeight()
	if totalWeight == 0 {
		return 0
	}
	twoM := 2 * totalWeight
	degrees := make(map[string]float64, len(g.nodes))
	for _, node := range g.nodes {
		for _, weight := range g.adj[node] {
			degrees[node] += weight
		}
	}

	var sum float64
	for _, a := range g.nodes {
		for _, b := range g.nodes {
			if partition[a] != partition[b] {
				continue
			}
			expected := degrees[a] * degrees[b] / twoM
			sum += g.edgeWeight(a, b) - expected
		}
	}
	return sum / twoM
}

func aggregateGraph(g weightedGraph, partition map[string]string) weightedGraph {
	next := newWeightedGraph()
	for _, community := range sortedUniqueValues(partition) {
		next.addNode(community)
	}
	for _, a := range g.nodes {
		for b, weight := range g.adj[a] {
			if a > b {
				continue
			}
			if a == b {
				next.addEdge(partition[a], partition[b], weight/2)
				continue
			}
			next.addEdge(partition[a], partition[b], weight)
		}
	}
	return next
}

func clonePartition(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sortedUniqueValues(in map[string]string) []string {
	seen := make(map[string]struct{}, len(in))
	for _, v := range in {
		seen[v] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func singletonCommunities(nodes []string) map[string]int {
	out := make(map[string]int, len(nodes))
	for i, node := range nodes {
		out[node] = i + 1
	}
	return out
}

func normalizeCommunities(assignments map[string]string) map[string]int {
	communityIDs := make(map[string]int)
	communities := sortedUniqueValues(assignments)
	for i, community := range communities {
		communityIDs[community] = i + 1
	}
	out := make(map[string]int, len(assignments))
	for node, community := range assignments {
		out[node] = communityIDs[community]
	}
	return out
}
