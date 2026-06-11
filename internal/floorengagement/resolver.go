// Package floorengagement provides the production EngagementResolver: it mirrors
// Baseline Modes engagement state from the on-disk floor (internal/baselinefloor)
// into the fact shape the lifecycle's source-dir decision consumes.
//
// It lives outside internal/lifecycle on purpose. The lifecycle owns the
// EngagementResolver interface and depends only on it; this package supplies the
// concrete implementation and is the single place that imports the floor, so the
// lifecycle never pulls the floor's filesystem machinery into its dependency
// graph. The production binary wires it once via lifecycle.SetDefaultEngagementResolver.
package floorengagement

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/lifecycle"
)

// Resolver answers lifecycle engagement queries from the floor store.
type Resolver struct {
	store *baselinefloor.Store
}

var _ lifecycle.EngagementResolver = (*Resolver)(nil)

// New constructs a Resolver over the default floor cache root. It is the form
// the production binary wires at startup.
func New() (*Resolver, error) {
	store, err := baselinefloor.DefaultStore()
	if err != nil {
		return nil, err
	}
	return &Resolver{store: store}, nil
}

// NewWithStore constructs a Resolver over an explicit store (tests inject a temp
// root).
func NewWithStore(store *baselinefloor.Store) *Resolver {
	return &Resolver{store: store}
}

// Engagement reports whether scenario is in a source-dir split and, if so, the
// facts needed to resolve its locations. Only a SHADOW-mode engagement creates a
// split (the serving instance must run from the frozen copy while the working
// tree holds the candidate); a live-mode engagement edits the working tree in
// place and is therefore reported engaged=false. The one-engagement-per-scenario
// invariant means at most one shadow engagement matches; finding more than one
// (a corrupt floor) is surfaced as an error so the caller fails closed rather
// than silently picking one.
func (r *Resolver) Engagement(scenario string) (lifecycle.EngagementInfo, bool, error) {
	manifests, err := r.store.ListManifests()
	if err != nil {
		return lifecycle.EngagementInfo{}, false, fmt.Errorf("floorengagement: list manifests: %w", err)
	}

	var match *baselinefloor.Manifest
	for i := range manifests {
		m := manifests[i]
		if m.Scenario != scenario || m.Mode != baselinefloor.ModeShadow {
			continue
		}
		if match != nil {
			return lifecycle.EngagementInfo{}, false, fmt.Errorf(
				"floorengagement: scenario %q has multiple open shadow engagements (%q, %q) — the one-engagement-per-scenario invariant is broken",
				scenario, match.Slug, m.Slug)
		}
		match = &manifests[i]
	}
	if match == nil {
		return lifecycle.EngagementInfo{}, false, nil
	}

	return lifecycle.EngagementInfo{
		RestorePointDir: r.store.RestorePointPath(match.Scenario, match.Slug),
		Mode:            string(match.Mode),
		Slug:            match.Slug,
	}, true, nil
}
