// Graph materialization — project each initiative's member item graph into a
// stable on-disk artifact (`initiatives/{name}/graph.json`).
//
// Agents and UI components read graph.json as the canonical view of an
// initiative's item graph. It is a READ-ONLY projection: the source of truth
// remains each item's `depends_on` array. Any direct write to graph.json is
// a bug — it will be overwritten on the next mutation.
//
// Invariants:
//   - graph.json contains only items currently assigned to the initiative
//   - edges reflect depends_on relationships where both endpoints live in the
//     initiative (cross-initiative edges are dropped here; surface them via
//     the full graph API instead)
//   - generated_at is a UTC RFC3339 timestamp refreshed on every write
//
// This is intentionally simpler than the full graph projection (no active
// executions, no agent activity) — the materialized artifact is for cheap
// per-initiative agent/UI consumption. For operational views use the full
// projection service.
package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/storage"
)

// graphJSONFilename is the canonical filename under each initiative folder.
const graphJSONFilename = "graph.json"

// MaterializedGraphNode describes one item in the materialized graph.
type MaterializedGraphNode struct {
	ID       string `json:"id"` // "kind/name"
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	Effort   string `json:"effort,omitempty"`
	Archived bool   `json:"archived"`
}

// MaterializedGraphEdge describes a dependency between two items in the
// initiative. Edges are directional: `From` depends on `To`.
type MaterializedGraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"` // "depends_on"
}

// MaterializedGraph is the on-disk projection of an initiative's item graph.
type MaterializedGraph struct {
	Initiative  string                  `json:"initiative"`
	GeneratedAt string                  `json:"generated_at"`
	Nodes       []MaterializedGraphNode `json:"nodes"`
	Edges       []MaterializedGraphEdge `json:"edges"`
}

// Materializer writes per-initiative graph.json projections.
//
// The materializer depends only on read interfaces — it never mutates items,
// initiatives, or any state beyond writing graph.json files.
//
// `ScheduleAll` coalesces burst invalidations (e.g., a batch-create that
// fires many invalidations in quick succession) so we don't rebuild
// graph.json N times per burst.
//
// Concurrency:
//
//	mu guards the scheduling/pending coalescing flags. Its scope is
//	intentionally narrow — only ScheduleAll and runUntilSettled touch it.
//	MaterializeInitiative and MaterializeAll are safe to call without
//	holding mu (they only read through the injected interfaces). Writes
//	to graph.json go through storage.WriteJSONAtomic, which handles
//	cross-process atomicity via the temp-file + rename pattern.
type Materializer struct {
	initLister   InitiativeLister
	backlogStore BacklogItemLoader
	initDirFn    func(name string) string

	mu         sync.Mutex
	scheduling bool // goroutine is currently materializing
	pending    bool // another burst arrived mid-run
}

// BacklogItemLoader is the minimum backlog surface the materializer needs.
// Kept narrow so the materializer can be tested in isolation and so no
// mutation verbs leak into the projection pipeline.
type BacklogItemLoader interface {
	LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
}

// BacklogStore is a deprecated alias retained for call sites that predate
// the rename to BacklogItemLoader. New code should use BacklogItemLoader.
//
// Deprecated: use BacklogItemLoader.
type BacklogStore = BacklogItemLoader

// NewMaterializer constructs a materializer. `initDirFn` returns the on-disk
// folder for a named initiative (typically `initiatives.Store.InitDir`).
// Pass `nil` for any dependency to disable materialization (all methods
// become no-ops).
func NewMaterializer(initLister InitiativeLister, backlogStore BacklogItemLoader, initDirFn func(name string) string) *Materializer {
	return &Materializer{
		initLister:   initLister,
		backlogStore: backlogStore,
		initDirFn:    initDirFn,
	}
}

// MaterializeInitiative rebuilds graph.json for a single initiative. Missing
// items (names that reference deleted backlog items) are silently dropped;
// cross-initiative dependencies are not represented.
func (m *Materializer) MaterializeInitiative(ctx context.Context, initName string) error {
	if m == nil || m.initLister == nil || m.backlogStore == nil || m.initDirFn == nil {
		return nil
	}
	if strings.TrimSpace(initName) == "" {
		return fmt.Errorf("initiative name is required")
	}

	all, err := m.initLister.List()
	if err != nil {
		return fmt.Errorf("list initiatives: %w", err)
	}
	var target *InitiativeEntry
	for i := range all {
		if all[i].Name == initName {
			target = &all[i]
			break
		}
	}
	if target == nil {
		// Initiative was deleted; remove any stale graph.json.
		path := m.graphPath(initName)
		if _, statErr := os.Stat(path); statErr == nil {
			if rmErr := os.Remove(path); rmErr != nil {
				slog.Debug("graph: remove stale graph.json failed", "err", rmErr, "path", path)
			}
		}
		return nil
	}

	graph := m.buildGraph(ctx, *target)
	return m.writeGraph(initName, graph)
}

// MaterializeAll rebuilds graph.json for every initiative. Called on startup
// and after bulk mutations. Per-initiative failures are logged individually
// (so operators can tell which initiatives went stale) and joined into the
// returned error — never aborts the loop, so a single bad initiative can't
// block the rest from re-materializing.
func (m *Materializer) MaterializeAll(ctx context.Context) error {
	if m == nil || m.initLister == nil || m.backlogStore == nil || m.initDirFn == nil {
		return nil
	}
	all, err := m.initLister.List()
	if err != nil {
		return fmt.Errorf("list initiatives: %w", err)
	}
	var errs []error
	for _, init := range all {
		if err := m.MaterializeInitiative(ctx, init.Name); err != nil {
			slog.Warn("materialize initiative failed", "initiative", init.Name, "err", err)
			errs = append(errs, fmt.Errorf("initiative %s: %w", init.Name, err))
		}
	}
	return errors.Join(errs...)
}

// ReadGraph loads the materialized graph.json for a single
// initiative. Returns (nil, nil) if no graph has been materialized yet.
func (m *Materializer) ReadGraph(initName string) (*MaterializedGraph, error) {
	path := m.graphPath(initName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var g MaterializedGraph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &g, nil
}

// buildGraph projects the initiative's member items and their dependency
// edges into a MaterializedGraph. Pure — no I/O beyond reading items.
func (m *Materializer) buildGraph(ctx context.Context, init InitiativeEntry) MaterializedGraph {
	nodes := make([]MaterializedGraphNode, 0, len(init.Items))
	memberSet := make(map[string]struct{}, len(init.Items))

	for _, ref := range init.Items {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			slog.Warn("initiative references unparsable item during materialization",
				"initiative", init.Name, "item_ref", ref)
			continue
		}
		kind := backlog.BacklogKind(parts[0])
		name := parts[1]
		item, err := m.backlogStore.LoadItem(kind, name)
		if err != nil {
			// Dropped items are not errors — they may be mid-deletion or
			// the initiative has a stale reference. Log so the gap is
			// visible without failing the whole projection.
			slog.Warn("initiative item not found during materialization",
				"initiative", init.Name, "item_ref", ref, "err", err)
			continue
		}
		archived := item.ArchivedAt != nil && *item.ArchivedAt != ""
		memberSet[ref] = struct{}{}
		nodes = append(nodes, MaterializedGraphNode{
			ID:       ref,
			Kind:     string(item.Kind),
			Name:     item.Name,
			Title:    item.Title,
			Status:   string(item.Status),
			Priority: item.Priority,
			Effort:   item.Effort,
			Archived: archived,
		})
	}

	edges := make([]MaterializedGraphEdge, 0)
	for _, ref := range init.Items {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			continue
		}
		item, err := m.backlogStore.LoadItem(backlog.BacklogKind(parts[0]), parts[1])
		if err != nil {
			continue
		}
		for _, dep := range item.DependsOn {
			// Only represent intra-initiative edges in graph.json; the full
			// graph projection covers cross-initiative relationships.
			if _, ok := memberSet[dep]; !ok {
				continue
			}
			edges = append(edges, MaterializedGraphEdge{
				From: ref,
				To:   dep,
				Kind: "depends_on",
			})
		}
	}

	return MaterializedGraph{
		Initiative:  init.Name,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Nodes:       nodes,
		Edges:       edges,
	}
}

// writeGraph persists a MaterializedGraph to disk atomically.
//
// Skips the write when the existing graph.json has identical Initiative,
// Nodes, and Edges — GeneratedAt alone is not worth a file-system write or
// the git-status noise it produces across ~60 initiatives on every burst.
// First-time writes, corrupt existing files, and any real content change
// fall through to WriteJSONAtomic.
func (m *Materializer) writeGraph(initName string, g MaterializedGraph) error {
	path := m.graphPath(initName)
	if existing, err := m.ReadGraph(initName); err == nil && existing != nil {
		if existing.Initiative == g.Initiative &&
			reflect.DeepEqual(existing.Nodes, g.Nodes) &&
			reflect.DeepEqual(existing.Edges, g.Edges) {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return storage.WriteJSONAtomic(path, g)
}

// graphPath returns the absolute path to an initiative's graph.json.
func (m *Materializer) graphPath(initName string) string {
	if m.initDirFn == nil {
		return ""
	}
	return filepath.Join(m.initDirFn(initName), graphJSONFilename)
}

// ScheduleAll triggers MaterializeAll on a background goroutine. If a
// materialization is already in flight, the call marks "pending" and the
// running goroutine will re-run once after it completes. This coalesces
// burst invalidations (e.g. batch-create) into at most two runs: one for
// the work already happening, one to catch up.
//
// Never blocks the caller. Safe to call from any goroutine, including
// dispatch invalidation hooks running in the request goroutine.
//
// The mu critical section holds only the scheduling/pending flag swap —
// the materialization itself runs unlocked so a slow projection never
// stalls producers. See the Materializer doc comment for the full
// concurrency model.
func (m *Materializer) ScheduleAll() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.scheduling {
		m.pending = true
		m.mu.Unlock()
		return
	}
	m.scheduling = true
	m.mu.Unlock()

	go m.runUntilSettled()
}

func (m *Materializer) runUntilSettled() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := m.MaterializeAll(ctx); err != nil {
			slog.Warn("materialize graph.json failed", "err", err)
		}
		cancel()

		m.mu.Lock()
		if !m.pending {
			m.scheduling = false
			m.mu.Unlock()
			return
		}
		m.pending = false
		m.mu.Unlock()
	}
}
