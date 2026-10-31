package templateengine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/templatevalidation"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

// destroy is the inverse of generate.
//
// Generation writes two things: the scenario directory, and six codegen outputs
// in the SHARED packages/proto tree. Nothing links the second back to the first,
// so a plain `rm -rf scenarios/<id>` strands all six -- which is exactly how
// five throwaway surfaces left 274 files of residue in this repo. Teardown must
// therefore be a first-class command, not a documented rm.
//
// The proto footprint set is owned by internal/templatevalidation, the same
// place the deep-validation cleaner uses, so both paths reap identically and a
// newly added codegen output only has to be registered once.

func runDestroy[C any](deps HandlerDeps[C], ctx C, req templatecontracts.DestroyRequest) (templatecontracts.DestroyResult, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return templatecontracts.DestroyResult{}, fmt.Errorf("scenario id is required")
	}
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return templatecontracts.DestroyResult{}, fmt.Errorf("scenario id %q must be a bare id, not a path", name)
	}

	root := deps.Root(ctx)
	scenarioDir := filepath.Join(root, "scenarios", name)
	scenarioExists := dirExists(scenarioDir)

	// Destroying a live scenario is not something to do by accident. --proto-only
	// is exempt: it never touches the scenario directory.
	if scenarioExists && !req.ProtoOnly && !req.Force {
		return templatecontracts.DestroyResult{}, fmt.Errorf(
			"scenarios/%s still exists; pass --force to destroy it, or --proto-only to reap just the stranded codegen", name)
	}

	targets := destroyTargets(root, name, req.ProtoOnly, scenarioExists)

	result := templatecontracts.DestroyResult{
		Scenario:  name,
		DryRun:    req.DryRun,
		ProtoOnly: req.ProtoOnly,
	}
	for _, path := range targets {
		rel := relativeToRoot(root, path)
		if !pathExists(path) {
			result.PathsAbsent = append(result.PathsAbsent, rel)
			continue
		}
		result.PathsRemoved = append(result.PathsRemoved, rel)
		if req.DryRun {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			return result, fmt.Errorf("remove %s: %w", rel, err)
		}
	}

	result.NeedsProtoGenerate = len(result.PathsRemoved) > 0
	result.Message = destroyMessage(result)
	return result, nil
}

// destroyTargets is the full removal set: the scenario directory (unless
// proto-only) plus every codegen output derived from the surface.
func destroyTargets(root, name string, protoOnly, scenarioExists bool) []string {
	schemasTarget := filepath.Join(root, "packages", "proto", "schemas", name)
	targets := templatevalidation.RelocationArtifactPaths([]string{schemasTarget})

	if !protoOnly && scenarioExists {
		targets = append(targets, filepath.Join(root, "scenarios", name))
	}
	sort.Strings(targets)
	return targets
}

func destroyMessage(result templatecontracts.DestroyResult) string {
	verb := "removed"
	if result.DryRun {
		verb = "would remove"
	}
	scope := "scenario and proto footprint"
	if result.ProtoOnly {
		scope = "proto footprint"
	}
	msg := fmt.Sprintf("%s %d path(s) of %s for %q", verb, len(result.PathsRemoved), scope, result.Scenario)
	if result.NeedsProtoGenerate && !result.DryRun {
		msg += "; run (cd packages/proto && make generate) to drop it from the descriptor sets"
	}
	return msg
}

func relativeToRoot(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
