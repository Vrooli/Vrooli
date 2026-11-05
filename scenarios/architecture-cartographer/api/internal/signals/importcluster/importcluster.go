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
	"log"
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

func (Signal) Score(ctx context.Context, gctx signals.GraphContext, chunk graph.Chunk) signals.ScoreResult {
	if err := ctx.Err(); err != nil {
		return signals.Abstain(name, err.Error(), chunk.Path)
	}
	if chunk.FileID == "" {
		return signals.Abstain(name, "chunk has no file id", chunk.Path)
	}
	pkgID := graphindex.PackageForFileIn(chunk.FileID, gctx)
	if pkgID == "" {
		return signals.Abstain(name, "file has no package in snapshot", chunk.Path)
	}
	clusters, err := computeClusters(ctx, gctx)
	if err != nil {
		return signals.Abstain(name, err.Error(), chunk.Path)
	}
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

// maxClusterNodes bounds the number of connected packages fed into the
// Louvain pass. Real internal import graphs are small (tens to low
// hundreds of packages). A count far above that means an upstream
// producer mis-classified symbol-level nodes as packages (the
// typescript-code-graph NODE_KIND_PACKAGE call/reference nodes are the
// canonical offender). Rather than spend minutes pegging a core on an
// O(N²) modularity pass over garbage, we degrade to singleton clusters
// and log the drop so the regression is visible instead of silent.
const maxClusterNodes = 2000

// computeClusters returns a per-internal-package cluster id derived
// from deterministic Louvain modularity communities over the import graph.
// Cached on GraphContext.Caches so subsequent calls in the same
// scoring batch share the work; access is goroutine-safe.
//
// Only packages that participate in at least one internal import edge are
// fed to the Louvain pass; isolated packages carry no co-import evidence,
// so they are assigned deterministic singleton clusters without inflating
// the O(N²) modularity computation.
func computeClusters(ctx context.Context, gctx signals.GraphContext) (map[string]int, error) {
	if gctx.Caches != nil {
		if cached := gctx.Caches.CommunitySnapshot(); cached != nil {
			return cached, nil
		}
	}
	inScenario := make(map[string]struct{}, len(gctx.Snapshot.Packages))
	allPackages := make([]string, 0, len(gctx.Snapshot.Packages))
	for _, p := range gctx.Snapshot.Packages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, dup := inScenario[p.ID]; dup {
			continue
		}
		inScenario[p.ID] = struct{}{}
		allPackages = append(allPackages, p.ID)
	}

	// Build the connected subgraph: addEdge introduces only the endpoints
	// of kept edges, so isolated packages never enter the weighted graph.
	graph := newWeightedGraph()
	for _, e := range gctx.Snapshot.Imports {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		from := graphindex.PackageForIn(e.From, gctx)
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

	var cluster map[string]int
	if len(graph.nodes) > maxClusterNodes {
		log.Printf("import-cluster: %d connected packages exceeds cap %d (likely an upstream producer emitting symbol-level nodes as packages); degrading to singleton clusters", len(graph.nodes), maxClusterNodes)
		cluster = map[string]int{}
	} else {
		var err error
		cluster, err = louvainCommunities(ctx, graph)
		if err != nil {
			return nil, err
		}
	}

	// Assign each isolated (or, on degrade, every) package its own
	// deterministic singleton cluster id above the Louvain-assigned range.
	cluster = appendSingletonClusters(cluster, allPackages)

	if gctx.Caches != nil {
		gctx.Caches.SetCommunity(cluster)
	}
	return cluster, nil
}

// appendSingletonClusters gives every package in all that the Louvain
// pass did not place its own cluster id, numbered deterministically above
// the existing maximum so ids never collide.
func appendSingletonClusters(cluster map[string]int, all []string) map[string]int {
	if cluster == nil {
		cluster = make(map[string]int, len(all))
	}
	maxID := 0
	for _, id := range cluster {
		if id > maxID {
			maxID = id
		}
	}
	missing := make([]string, 0, len(all))
	for _, pkg := range all {
		if _, ok := cluster[pkg]; !ok {
			missing = append(missing, pkg)
		}
	}
	sort.Strings(missing)
	for i, pkg := range missing {
		cluster[pkg] = maxID + 1 + i
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

func (g weightedGraph) totalEdgeWeight() float64 {
	var total float64
	for _, a := range g.nodes {
		for _, weight := range g.adj[a] {
			total += weight
		}
	}
	return total / 2
}

// maxLocalMovingSweeps bounds the local-moving phase of a single Louvain
// level. Local moving converges in a handful of sweeps in practice; the
// cap is a non-termination backstop (defensive against pathological
// oscillation), not an expected limit.
const maxLocalMovingSweeps = 100

// louvainCommunities runs deterministic multi-level Louvain over g and
// returns a node→community-id assignment. Modularity gains are computed
// incrementally in O(degree) per candidate move (see optimizePartition);
// the global modularity is never recomputed from scratch.
func louvainCommunities(ctx context.Context, g weightedGraph) (map[string]int, error) {
	if len(g.nodes) == 0 {
		return map[string]int{}, nil
	}
	if g.totalEdgeWeight() == 0 {
		return singletonCommunities(g.nodes), nil
	}

	originalToNode := make(map[string]string, len(g.nodes))
	for _, node := range g.nodes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		originalToNode[node] = node
	}

	current := g
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		partition, changed, err := optimizePartition(ctx, current)
		if err != nil {
			return nil, err
		}
		for original, node := range originalToNode {
			originalToNode[original] = partition[node]
		}
		if !changed {
			break
		}

		next := aggregateGraph(current, partition)
		if len(next.nodes) == len(current.nodes) {
			break
		}
		current = next
	}

	return normalizeCommunities(originalToNode), nil
}

// optimizePartition runs the Louvain local-moving phase on g using
// incremental modularity gain: each candidate move is evaluated in
// O(degree) from the per-community degree sums (sigmaTot) and the node's
// edge weight into each neighbouring community, instead of recomputing
// the global O(N²) modularity. It returns the node→community-label
// partition and whether any node changed community. Determinism comes
// from the sorted node order (g.nodes is kept sorted) and a sorted,
// lexicographic tie-break across candidate communities.
func optimizePartition(ctx context.Context, g weightedGraph) (map[string]string, bool, error) {
	twoM := 2 * g.totalEdgeWeight()
	if twoM == 0 {
		partition := make(map[string]string, len(g.nodes))
		for _, node := range g.nodes {
			partition[node] = node
		}
		return partition, false, nil
	}

	degree := make(map[string]float64, len(g.nodes))
	comm := make(map[string]string, len(g.nodes))
	sigmaTot := make(map[string]float64, len(g.nodes))
	for _, node := range g.nodes {
		var d float64
		for _, w := range g.adj[node] {
			d += w
		}
		degree[node] = d
		comm[node] = node
		sigmaTot[node] = d
	}

	changedAny := false
	for sweep := 0; sweep < maxLocalMovingSweeps; sweep++ {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		moved := false
		for _, node := range g.nodes {
			if err := ctx.Err(); err != nil {
				return nil, false, err
			}
			cur := comm[node]
			// Detach node from its current community before scoring moves.
			sigmaTot[cur] -= degree[node]

			// Edge weight from node into each neighbouring community
			// (self-loops are internal to the node and irrelevant here).
			toCommunity := make(map[string]float64, len(g.adj[node]))
			for nb, w := range g.adj[node] {
				if nb == node {
					continue
				}
				toCommunity[comm[nb]] += w
			}

			candidates := make([]string, 0, len(toCommunity)+1)
			seen := map[string]struct{}{cur: {}}
			candidates = append(candidates, cur)
			for c := range toCommunity {
				if _, ok := seen[c]; ok {
					continue
				}
				seen[c] = struct{}{}
				candidates = append(candidates, c)
			}
			sort.Strings(candidates)

			best := cur
			bestGain := toCommunity[cur] - sigmaTot[cur]*degree[node]/twoM
			for _, c := range candidates {
				if c == cur {
					continue
				}
				gain := toCommunity[c] - sigmaTot[c]*degree[node]/twoM
				if gain > bestGain+modularityEpsilon ||
					(math.Abs(gain-bestGain) <= modularityEpsilon && c < best) {
					best = c
					bestGain = gain
				}
			}

			// Reattach into the chosen community.
			sigmaTot[best] += degree[node]
			if best != cur {
				comm[node] = best
				moved = true
				changedAny = true
			}
		}
		if !moved {
			break
		}
	}
	return comm, changedAny, nil
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
