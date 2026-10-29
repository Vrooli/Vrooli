package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	recoveryapp "github.com/vrooli/vrooli/internal/app/recovery"
	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// floorEngagementResolver is the production lifecycle resolver. Keeping the
// concrete floor dependency at the process-entry wiring edge prevents the
// lifecycle package from acquiring floor filesystem policy.
type floorEngagementResolver struct {
	store *baselinefloor.Store
	// servingDirs lists the working directories of registered instances that
	// may still be running. Superseded serving trees are reaped only when no
	// listed directory lies inside them. Nil disables reaping.
	servingDirs func(scenario string) ([]string, error)
}

var (
	_ lifecycle.EngagementResolver        = (*floorEngagementResolver)(nil)
	_ lifecycle.ServingSourceMaterializer = (*floorEngagementResolver)(nil)
	_ lifecycle.ServingSourceLocator      = (*floorEngagementResolver)(nil)
)

func newFloorEngagementResolver() (*floorEngagementResolver, error) {
	store, err := baselinefloor.DefaultStore()
	if err != nil {
		return nil, err
	}
	return &floorEngagementResolver{store: store, servingDirs: registryWorkingDirs}, nil
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
	info := lifecycle.EngagementInfo{
		RestorePointDir: r.store.RestorePointPath(match.Scenario, match.Slug),
		Mode:            string(match.Mode),
		Slug:            match.Slug,
		Generation:      match.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	record, ok, err := r.store.ReadCaptureRecord(match.Scenario, match.Slug)
	if err != nil {
		return lifecycle.EngagementInfo{}, false, fmt.Errorf("floor engagement: %w", err)
	}
	if ok && strings.TrimSpace(record.CaptureID) != "" {
		info.Generation = record.CaptureID
		info.CaptureID = record.CaptureID
		info.SharedSourcesFrozen = record.SharedSources.Frozen()
		info.ProtoArtifactID = record.SharedSources.ProtoArtifactID
	}
	return info, true, nil
}

// MaterializeServingSource builds the engagement's repo-shaped serving tree
// for the current capture generation: the frozen restore point plus the
// frozen shared sources, with every other repository entry linked (see
// baselinefloor.MaterializeServing). Superseded generations that no
// registered instance runs from are reaped afterwards.
func (r *floorEngagementResolver) MaterializeServingSource(scenario string, info lifecycle.EngagementInfo, repoRoot string) (string, error) {
	spec := baselinefloor.ServingSpec{
		Scenario:       scenario,
		RepoRoot:       repoRoot,
		RestorePoint:   info.RestorePointDir,
		Generation:     info.Generation,
		ConfigFallback: filepath.Join(repoRoot, "scenarios", scenario),
		Ignore:         recoveryapp.ScenarioCopyIgnorer(repoRoot, scenario, info.RestorePointDir, filepath.Join(repoRoot, "scenarios", scenario)),
	}
	if info.SharedSourcesFrozen {
		record, ok, err := r.store.ReadCaptureRecord(scenario, info.Slug)
		if err != nil {
			return "", err
		}
		if !ok || !record.SharedSources.Frozen() {
			return "", fmt.Errorf("floor engagement: %s/%s capture record no longer reports frozen shared sources", scenario, info.Slug)
		}
		spec.SharedSources = r.store.SharedSourcesPath(scenario, info.Slug)
		for _, root := range record.SharedSources.Roots {
			spec.Frozen = append(spec.Frozen, root.Path)
		}
	}
	dir, err := baselinefloor.MaterializeServing(r.store.ServingBasePath(scenario, info.Slug), spec)
	if err != nil {
		return "", err
	}
	if r.servingDirs != nil {
		keep := baselinefloor.ServingGenerationKey(info.Generation)
		dirs, listErr := r.servingDirs(scenario)
		inUse := func(treeRoot string) bool {
			if listErr != nil {
				return true
			}
			for _, dir := range dirs {
				if within(treeRoot, dir) {
					return true
				}
			}
			return false
		}
		// Best effort: a superseded tree that cannot be removed now is
		// removed with the engagement.
		_, _ = r.store.ReapServingTrees(scenario, info.Slug, keep, inUse)
	}
	return dir, nil
}

// ServingSourceLocation resolves the expected serving generation without
// materializing it. Deploy reports use this to inspect the recorded serving
// baseline while remaining read-only.
func (r *floorEngagementResolver) ServingSourceLocation(scenario string, info lifecycle.EngagementInfo, _ string) (lifecycle.ServingSourceLocation, error) {
	if strings.TrimSpace(info.Slug) == "" || strings.TrimSpace(info.Generation) == "" {
		return lifecycle.ServingSourceLocation{}, fmt.Errorf("floor engagement: serving source for %q has no engagement slug or generation", scenario)
	}
	root := filepath.Join(r.store.ServingBasePath(scenario, info.Slug), baselinefloor.ServingGenerationKey(info.Generation))
	return lifecycle.ServingSourceLocation{RepoRoot: root, ScenarioDir: baselinefloor.ServingScenarioDir(root, scenario)}, nil
}

// registryWorkingDirs lists the working directories of every registered
// instance of a scenario that has not reached a terminal state.
func registryWorkingDirs(scenario string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbPath, err := scenarioruntime.DefaultDBPath("")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err // no registry: cannot prove a tree is unused
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer store.Close()
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: scenario,
		Statuses: append(scenarioruntime.ActiveInstanceStatuses(), scenarioruntime.StatusStopping),
	})
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(instances))
	for _, instance := range instances {
		if strings.TrimSpace(instance.WorkingDir) != "" {
			dirs = append(dirs, instance.WorkingDir)
		}
	}
	return dirs, nil
}

func within(root, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
