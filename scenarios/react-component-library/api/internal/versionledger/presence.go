package versionledger

import (
	"context"
	"fmt"

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
					return fmt.Errorf("evict %s@%s: %w", asset.LibraryID, row.Version, err)
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
					return fmt.Errorf("materialize %s@%s: %w", asset.LibraryID, row.Version, err)
				}
			}
		}
	}
	return nil
}
