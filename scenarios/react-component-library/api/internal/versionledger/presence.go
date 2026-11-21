package versionledger

import (
	"context"
	"fmt"
	"strings"

	"react-component-library/internal/components"
)

// PresenceReconciler computes the materialization tier from the same
// RetireCandidates reachability graph used by the operator-facing cleanup
// report. It is shared by adoption lifecycle hooks and the index handler so
// automatic paths cannot drift into a second, weaker safety policy.
type PresenceReconciler struct {
	ledger       *Repository
	catalog      components.Service
	materializer components.Materializer
}

func NewPresenceReconciler(ledger *Repository, catalog components.Service, materializer components.Materializer) *PresenceReconciler {
	return &PresenceReconciler{ledger: ledger, catalog: catalog, materializer: materializer}
}

var _ components.PresenceReconciler = (*PresenceReconciler)(nil)

// maxPresencePasses bounds how many owner-first eviction rounds one
// reconciliation runs. Each round can only evict versions no materialized
// release pins, so a historical chain drains one link per round; dependency
// ranks run 1-6, which bounds real chains well below this.
const maxPresencePasses = 8

// ReconcilePresence applies the desired tier for one component, or the full
// catalog when componentID is empty. Dry-run reporting belongs to the
// versions transport; lifecycle hooks always call this method with apply=true.
//
// Eviction runs in rounds until nothing more moves. The reachability graph
// keeps every pin of a materialized release live, so a round never leaves a
// materialized version importing an evicted one; the next round sees the
// evicted owners' pins as cold edges and can release what they held.
func (r *PresenceReconciler) ReconcilePresence(ctx context.Context, componentID string, apply bool) error {
	if r == nil || r.ledger == nil || r.catalog == nil {
		return fmt.Errorf("presence reconciliation is not configured")
	}
	assets, err := r.catalog.List(ctx, components.SearchQuery{Limit: 2000})
	if err != nil {
		return err
	}
	// Per-version tier moves that fail are collected rather than returned, so
	// one stuck version cannot cost the caller the other 235 assets.
	var deferred []string
	seen := map[string]bool{}
	for pass := 0; pass < maxPresencePasses; pass++ {
		evicted, passDeferred, err := r.reconcilePresencePass(ctx, assets, componentID, apply)
		if err != nil {
			return err
		}
		// A move that fails in one round is retried, and fails the same way,
		// in the next; report it once.
		for _, item := range passDeferred {
			if !seen[item] {
				seen[item] = true
				deferred = append(deferred, item)
			}
		}
		if evicted == 0 || !apply {
			break
		}
	}
	if len(deferred) > 0 {
		return ErrPresenceReconciliationIncomplete{Deferred: deferred}
	}
	return nil
}

func (r *PresenceReconciler) reconcilePresencePass(ctx context.Context, assets []components.Component, componentID string, apply bool) (int, []string, error) {
	var deferred []string
	evicted := 0
	candidates, err := r.ledger.RetireCandidates(ctx, componentID)
	if err != nil {
		return 0, nil, fmt.Errorf("build presence reachability graph: %w", err)
	}
	safe := make(map[string]map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		byVersion := safe[candidate.ComponentID]
		if byVersion == nil {
			byVersion = map[string]struct{}{}
			safe[candidate.ComponentID] = byVersion
		}
		byVersion[candidate.Version] = struct{}{}
	}
	for _, asset := range assets {
		if componentID != "" && componentID != asset.ID && componentID != asset.LibraryID {
			continue
		}
		rows, err := r.catalog.ListVersions(ctx, asset.ID, 2000)
		if err != nil {
			return evicted, deferred, err
		}
		for _, row := range rows {
			if strings.EqualFold(string(row.Status), "retired") {
				if apply && row.Presence != "evicted" {
					if _, err := r.ledger.Transition(ctx, asset.ID, row.Version, "retired", true); err != nil {
						return evicted, deferred, fmt.Errorf("reclaim retired %s@%s: %w", asset.LibraryID, row.Version, err)
					}
				}
				continue
			}
			_, unreachable := safe[asset.ID][row.Version]
			if unreachable && row.Presence != "evicted" {
				if !apply {
					continue
				}
				_, planHash, err := r.ledger.PlanCleanup(ctx, CleanupScope{ComponentID: asset.ID})
				if err != nil {
					return evicted, deferred, err
				}
				if _, err := r.ledger.Transition(ctx, asset.ID, row.Version, "archived", true, planHash); err != nil {
					// Moving a version to cold storage is an optimisation, not
					// a correctness requirement, and its safety check can
					// legitimately disagree with this reachability graph. A
					// refusal here used to abort the whole reindex — and
					// because reconciliation runs after the walk, it also
					// discarded the walk's own error report, hiding every
					// manifest that had failed to index behind one unrelated
					// eviction. Record it and carry on.
					deferred = append(deferred, fmt.Sprintf("evict %s@%s: %v", asset.LibraryID, row.Version, err))
					continue
				}
				evicted++
				continue
			}
			if !unreachable && row.Presence == "evicted" {
				if !apply {
					continue
				}
				if r.materializer == nil {
					return evicted, deferred, fmt.Errorf("materializer is not configured for %s@%s", asset.LibraryID, row.Version)
				}
				if _, err := r.materializer.EnsureMaterialized(ctx, asset.ID, row.Version, ""); err != nil {
					deferred = append(deferred, fmt.Sprintf("materialize %s@%s: %v", asset.LibraryID, row.Version, err))
				}
			}
		}
	}
	return evicted, deferred, nil
}

// ErrPresenceReconciliationIncomplete reports tier moves that did not apply.
// The catalog is still correctly indexed when this is returned; only the
// materialized/evicted placement of the named versions is unchanged.
type ErrPresenceReconciliationIncomplete struct {
	Deferred []string
}

func (e ErrPresenceReconciliationIncomplete) Error() string {
	return fmt.Sprintf("%d version tier move(s) deferred: %s", len(e.Deferred), strings.Join(e.Deferred, "; "))
}
