package proposals

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"
)

func (a *Applier) applySplit(ctx context.Context, ref string, into []ItemSpec, source Source) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	_, err = a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}

	created := make([]string, 0, len(into))
	rollback := func(reason error) {
		for _, r := range created {
			if rbErr := a.applyArchive(ctx, r); rbErr != nil {
				slog.Warn("proposals: split rollback failed",
					"source", ref,
					"child", r,
					"reason", reason,
					"err", rbErr,
				)
			}
		}
	}
	for _, spec := range into {
		if err := a.applyAddItem(ctx, spec, source); err != nil {
			rollback(err)
			return fmt.Errorf("create split child %s: %w", spec.Ref(), err)
		}
		created = append(created, spec.Ref())
	}

	// True atomicity: if archiving the source fails after all children
	// land, roll back the children too so the user sees pre-split state
	// instead of a half-split orphan graph. Dependents of the source
	// stay pointed at the original ref and must be retargeted by
	// subsequent OpAddEdge / OpRemoveEdge mutations — split does not
	// retarget dependents implicitly (see types.go OpSplitItem).
	if err := a.applyArchive(ctx, ref); err != nil {
		rollback(err)
		return fmt.Errorf("archive source item: %w", err)
	}
	return nil
}

// applyMergeItems collapses sourceRefs into a single new merged item described
// by spec. Edges to/from sources are auto-retargeted to the merged item;
// edges between sources are dropped. The order is:
//
//  1. Capture pre-merge state of every external item that depends on a
//     source (so we can roll back).
//  2. Compute the merged item's final depends_on:
//     spec.DependsOn (with any source refs filtered out)
//     ∪ outbound non-source deps from each source
//     deduplicated.
//  3. Create the merged item via the standard add-item path.
//  4. Re-target every external dependent: replace each source ref in its
//     depends_on with the merged ref (deduplicated).
//  5. Archive each source.
//
// On any failure the steps reverse: un-archive sources, restore external
// dependents' depends_on from the snapshot, archive the merged item.
func (a *Applier) applyMergeItems(ctx context.Context, sourceRefs []string, spec ItemSpec, current CurrentState, source Source) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(sourceRefs) < 2 {
		return fmt.Errorf("merge_items: need at least 2 sources, got %d", len(sourceRefs))
	}
	mergedRef := spec.Ref()

	sourceSet := make(map[string]struct{}, len(sourceRefs))
	for _, s := range sourceRefs {
		sourceSet[s] = struct{}{}
	}

	// Step 1: classify edges into outbound deps and inbound dependents.
	outboundDeps, inboundDependents := classifyMergeEdges(current.Edges, sourceSet)

	// Step 2: build merged spec.depends_on, filter sources, dedup, union
	// with outboundDeps. Stable ordering for deterministic test output.
	merged := spec
	merged.DependsOn = buildMergedDeps(spec.DependsOn, outboundDeps, sourceSet)

	// Step 1b: capture original depends_on for every inbound dependent so
	// rollback can restore them exactly. Done before any write so a
	// failure midway through Step 4 has full original state.
	snapshots, err := a.snapshotDependents(inboundDependents)
	if err != nil {
		return err
	}

	// Step 3: create the merged item.
	if err := a.applyAddItem(ctx, merged, source); err != nil {
		return fmt.Errorf("create merged item %s: %w", mergedRef, err)
	}

	// Rollback closure: archive merged + restore snapshots + un-archive
	// any sources that have already been archived.
	mergedCreated := true
	archivedSources := make([]string, 0, len(sourceRefs))
	rollback := func(reason error) {
		// Restore each source we archived in step 5.
		for _, sref := range archivedSources {
			if rbErr := a.unarchiveItem(ctx, sref); rbErr != nil {
				slog.Warn("proposals: merge rollback unarchive failed",
					"source", sref,
					"reason", reason,
					"err", rbErr,
				)
			}
		}
		// Restore each external dependent's depends_on from snapshot.
		for _, snap := range snapshots {
			if rbErr := a.restoreDependsOn(ctx, snap.ref, snap.original); rbErr != nil {
				slog.Warn("proposals: merge rollback restore depends_on failed",
					"dependent", snap.ref,
					"reason", reason,
					"err", rbErr,
				)
			}
		}
		// Archive the merged item.
		if mergedCreated {
			if rbErr := a.applyArchive(ctx, mergedRef); rbErr != nil {
				slog.Warn("proposals: merge rollback archive merged failed",
					"merged", mergedRef,
					"reason", reason,
					"err", rbErr,
				)
			}
		}
	}

	// Step 4: retarget inbound dependents.
	for _, snap := range snapshots {
		kind, name, err := splitRef(snap.ref)
		if err != nil {
			rollback(err)
			return fmt.Errorf("merge_items: retarget %s: %w", snap.ref, err)
		}
		item, err := a.store.LoadItem(kind, name)
		if err != nil {
			rollback(err)
			return fmt.Errorf("merge_items: reload dependent %s: %w", snap.ref, err)
		}
		newDeps := retargetDependsOn(item.DependsOn, sourceSet, mergedRef)
		if stringSlicesEqual(item.DependsOn, newDeps) {
			continue
		}
		if err := a.store.ValidateDependencies(newDeps); err != nil {
			rollback(err)
			return fmt.Errorf("merge_items: validate retargeted deps for %s: %w", snap.ref, err)
		}
		item.DependsOn = newDeps
		item.Updated = a.clock().UTC().Format(time.RFC3339)
		if err := a.store.SaveItem(item); err != nil {
			rollback(err)
			return fmt.Errorf("merge_items: save retargeted %s: %w", snap.ref, err)
		}
	}

	// Step 5: archive each source. Order is deterministic (sourceRefs
	// in agent-supplied order) so rollback can mirror it.
	for _, sref := range sourceRefs {
		if err := a.applyArchive(ctx, sref); err != nil {
			rollback(err)
			return fmt.Errorf("merge_items: archive source %s: %w", sref, err)
		}
		archivedSources = append(archivedSources, sref)
	}
	return nil
}

// depSnapshot captures an inbound dependent's pre-merge depends_on so the
// merge rollback path can restore it exactly.
type depSnapshot struct {
	ref      string
	original []string
}

// classifyMergeEdges enumerates edges. Edge (a, b) means a depends on b. It
// returns:
//   - outboundDeps: { b : (a,b) ∈ E, a ∈ sources, b ∉ sources }
//   - inboundDependents: external items a with at least one (a, b) where b ∈ sources
//
// Intra-source edges (both endpoints in sources) are dropped.
func classifyMergeEdges(edges []GraphEdge, sourceSet map[string]struct{}) (outboundDeps, inboundDependents map[string]struct{}) {
	outboundDeps = make(map[string]struct{})
	inboundDependents = make(map[string]struct{})
	for _, e := range edges {
		_, fromIsSource := sourceSet[e.From]
		_, toIsSource := sourceSet[e.To]
		switch {
		case fromIsSource && toIsSource:
			// intra-source edge: drop
		case fromIsSource && !toIsSource:
			outboundDeps[e.To] = struct{}{}
		case !fromIsSource && toIsSource:
			inboundDependents[e.From] = struct{}{}
		}
	}
	return outboundDeps, inboundDependents
}

// buildMergedDeps computes the merged item's final depends_on: specDeps with
// source refs filtered out, unioned with outboundDeps, deduplicated and sorted
// for deterministic output.
func buildMergedDeps(specDeps []string, outboundDeps, sourceSet map[string]struct{}) []string {
	depsSet := make(map[string]struct{}, len(specDeps)+len(outboundDeps))
	for _, dep := range specDeps {
		if _, isSource := sourceSet[dep]; isSource {
			continue
		}
		depsSet[dep] = struct{}{}
	}
	for dep := range outboundDeps {
		depsSet[dep] = struct{}{}
	}
	mergedDeps := make([]string, 0, len(depsSet))
	for dep := range depsSet {
		mergedDeps = append(mergedDeps, dep)
	}
	sort.Strings(mergedDeps)
	return mergedDeps
}

// snapshotDependents loads every inbound dependent and captures its original
// depends_on. Done before any write so a failure midway through the retarget
// step still has full original state to roll back to.
func (a *Applier) snapshotDependents(inboundDependents map[string]struct{}) ([]depSnapshot, error) {
	snapshots := make([]depSnapshot, 0, len(inboundDependents))
	for ref := range inboundDependents {
		kind, name, err := splitRef(ref)
		if err != nil {
			return nil, fmt.Errorf("merge_items: invalid dependent ref %s: %w", ref, err)
		}
		item, err := a.store.LoadItem(kind, name)
		if err != nil {
			return nil, fmt.Errorf("merge_items: load dependent %s: %w", ref, err)
		}
		snapshots = append(snapshots, depSnapshot{
			ref:      ref,
			original: append([]string(nil), item.DependsOn...),
		})
	}
	return snapshots, nil
}

// retargetDependsOn replaces every reference to a source with mergedRef,
// dedupes, and preserves order of non-source deps.
func retargetDependsOn(deps []string, sourceSet map[string]struct{}, mergedRef string) []string {
	out := make([]string, 0, len(deps))
	seen := make(map[string]struct{}, len(deps))
	hadSource := false
	for _, dep := range deps {
		if _, isSource := sourceSet[dep]; isSource {
			hadSource = true
			continue
		}
		if _, dup := seen[dep]; dup {
			continue
		}
		seen[dep] = struct{}{}
		out = append(out, dep)
	}
	if hadSource {
		if _, dup := seen[mergedRef]; !dup {
			out = append(out, mergedRef)
		}
	}
	return out
}

// unarchiveItem clears ArchivedAt on the given ref. Used by merge rollback.
// Errors are returned unwrapped — caller logs context.
func (a *Applier) unarchiveItem(ctx context.Context, ref string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	if item.ArchivedAt == nil || *item.ArchivedAt == "" {
		return nil
	}
	item.ArchivedAt = nil
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

// restoreDependsOn writes original onto the item's depends_on without any
// validation — rollback path is best-effort restoration of pre-merge state.
func (a *Applier) restoreDependsOn(ctx context.Context, ref string, original []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	item.DependsOn = append([]string(nil), original...)
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}
