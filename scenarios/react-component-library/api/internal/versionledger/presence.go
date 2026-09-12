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

// ReconcilePresence applies the desired tier for one component, or the full
// catalog when componentID is empty. Dry-run reporting belongs to the
// versions transport; lifecycle hooks always call this method with apply=true.
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
	candidates, err := r.ledger.RetireCandidates(ctx, componentID)
	if err != nil {
		return fmt.Errorf("build presence reachability graph: %w", err)
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
			return err
		}
		for _, row := range rows {
			if strings.EqualFold(string(row.Status), "retired") {
				if apply && row.Presence != "evicted" {
					if _, err := r.ledger.Transition(ctx, asset.ID, row.Version, "retired", true); err != nil {
						return fmt.Errorf("reclaim retired %s@%s: %w", asset.LibraryID, row.Version, err)
					}
				}
				continue
			}
			_, unreachable := safe[asset.ID][row.Version]
			if unreachable && row.Presence != "evicted" {
				if !apply {
					continue
				}
				items, planHash, err := r.ledger.PlanCleanup(ctx, CleanupScope{ComponentID: asset.ID})
				if err != nil {
					return err
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
				}
				_ = items
				continue
			}
			if !unreachable && row.Presence == "evicted" {
				if !apply {
					continue
				}
				if r.materializer == nil {
					return fmt.Errorf("materializer is not configured for %s@%s", asset.LibraryID, row.Version)
				}
				if _, err := r.materializer.EnsureMaterialized(ctx, asset.ID, row.Version, ""); err != nil {
					deferred = append(deferred, fmt.Sprintf("materialize %s@%s: %v", asset.LibraryID, row.Version, err))
				}
			}
		}
	}
	if len(deferred) > 0 {
		return ErrPresenceReconciliationIncomplete{Deferred: deferred}
	}
	return nil
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
