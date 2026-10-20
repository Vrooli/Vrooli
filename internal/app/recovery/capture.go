package recovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenario"
)

// ErrSharedSourcesUnresolved is returned when a capture cannot freeze the
// shared sources the scenario build needs and the caller did not allow an
// unfrozen capture.
var ErrSharedSourcesUnresolved = errors.New("recovery: shared sources could not be frozen")

// captureScenario loads the scenario descriptor from the capture source. A
// source without a manifest declares no builds, so it has no shared sources
// and no declared outputs.
func captureScenario(name, dir string) (scenario.Scenario, bool, error) {
	item, err := lifecycle.ScenarioAt(name, dir)
	if err == nil {
		return item, true, nil
	}
	if _, statErr := os.Stat(filepath.Join(dir, repocontractmeta.ProjectConfigDir, "service.json")); errors.Is(statErr, fs.ErrNotExist) {
		return scenario.Scenario{Slug: name, Path: dir}, false, nil
	}
	return scenario.Scenario{}, false, fmt.Errorf("recovery: read scenario manifest in %s: %w", dir, err)
}

// ScenarioCopyIgnorer builds the exclusion rules for copying a scenario tree
// that sits at repoRoot/scenarios/<name>: the repository's .gitignore files,
// the floor patterns, and the scenario's declared build outputs and runtime
// data directories. nestedRoot is where nested .gitignore files are read
// (the copy source); manifestDir is where the manifest is read.
func ScenarioCopyIgnorer(repoRoot, name, nestedRoot, manifestDir string) baselinefloor.Ignorer {
	patterns := baselinefloor.FloorIgnorePatterns()
	if item, ok, err := captureScenario(name, manifestDir); err == nil && ok {
		patterns = append(patterns, lifecycle.CaptureExcludePatterns(item)...)
	}
	return baselinefloor.NewIgnoreMatcher(baselinefloor.IgnoreSpec{
		RepoRoot:   repoRoot,
		TreeRoot:   filepath.Join(repoRoot, repocontractmeta.ScenarioDir, name),
		NestedRoot: nestedRoot,
		Patterns:   patterns,
	})
}

// treeIgnorer builds the rules for a tree that sits at treeRoot in the
// repository (a capture source or restore destination).
func (s Service) treeIgnorer(item scenario.Scenario, declared bool, treeRoot string) baselinefloor.Ignorer {
	patterns := baselinefloor.FloorIgnorePatterns()
	if declared {
		patterns = append(patterns, lifecycle.CaptureExcludePatterns(item)...)
	}
	return baselinefloor.NewIgnoreMatcher(baselinefloor.IgnoreSpec{
		RepoRoot: s.Root,
		TreeRoot: treeRoot,
		Patterns: patterns,
	})
}

// freezeSharedSources resolves and copies the shared sources of a capture.
// It runs before the restore point is touched, so a failed freeze leaves the
// previous capture intact.
func (s Service) freezeSharedSources(req CaptureRequest, item scenario.Scenario) (baselinefloor.SharedSources, error) {
	unfrozen := func(cause error) (baselinefloor.SharedSources, error) {
		if !req.AllowUnfrozen {
			return baselinefloor.SharedSources{}, fmt.Errorf("%w for %s: %v (pass --allow-unfrozen to capture without a freeze; the serving build then reads the repository's current shared packages)", ErrSharedSourcesUnresolved, req.Scenario, cause)
		}
		return baselinefloor.SharedSources{Status: baselinefloor.SharedSourcesUnfrozen, Reason: cause.Error()}, nil
	}
	resolve := s.SharedSourceRoots
	if resolve == nil {
		resolve = lifecycle.SharedSourceRoots
	}
	roots, err := resolve(context.Background(), s.Root, item)
	if err != nil {
		return unfrozen(err)
	}
	floorRoots := make([]baselinefloor.SharedRoot, 0, len(roots))
	for _, root := range roots {
		floorRoots = append(floorRoots, baselinefloor.SharedRoot{Path: root.Path, Reason: root.Reason, KeepOutputs: root.KeepOutputs})
	}
	dest := s.Store.SharedSourcesPath(req.Scenario, req.Slug)
	copied, total, err := baselinefloor.CaptureSharedSources(s.Root, dest, floorRoots, !req.NoReflink)
	if err != nil {
		return unfrozen(err)
	}
	return baselinefloor.SharedSources{
		Status:          baselinefloor.SharedSourcesFrozen,
		Roots:           copied,
		Bytes:           total,
		ProtoArtifactID: protoArtifactID(filepath.Join(dest, filepath.FromSlash(lifecycle.ProtoArtifactStampPath))),
	}, nil
}

func protoArtifactID(stamp string) string {
	data, err := os.ReadFile(stamp)
	if err != nil {
		return ""
	}
	var record struct {
		ArtifactID string `json:"artifact_id"`
	}
	if json.Unmarshal(data, &record) != nil {
		return ""
	}
	return strings.TrimSpace(record.ArtifactID)
}
