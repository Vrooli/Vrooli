package main

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/lifecycle"
)

// floorEngagementResolver is the production lifecycle resolver. Keeping the
// concrete floor dependency at the process-entry wiring edge prevents the
// lifecycle package from acquiring floor filesystem policy.
type floorEngagementResolver struct {
	store *baselinefloor.Store
}

var _ lifecycle.EngagementResolver = (*floorEngagementResolver)(nil)

func newFloorEngagementResolver() (*floorEngagementResolver, error) {
	store, err := baselinefloor.DefaultStore()
	if err != nil {
		return nil, err
	}
	return &floorEngagementResolver{store: store}, nil
}

func newFloorEngagementResolverWithStore(store *baselinefloor.Store) *floorEngagementResolver {
	return &floorEngagementResolver{store: store}
}

func (r *floorEngagementResolver) Engagement(scenario string) (lifecycle.EngagementInfo, bool, error) {
	manifests, err := r.store.ListManifests()
	if err != nil {
		return lifecycle.EngagementInfo{}, false, fmt.Errorf("floor engagement: list manifests: %w", err)
	}

	var match *baselinefloor.Manifest
	for i := range manifests {
		manifest := manifests[i]
		if manifest.Scenario != scenario || manifest.Mode != baselinefloor.ModeShadow {
			continue
		}
		if match != nil {
			return lifecycle.EngagementInfo{}, false, fmt.Errorf("floor engagement: scenario %q has multiple open shadow engagements (%q, %q)", scenario, match.Slug, manifest.Slug)
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
