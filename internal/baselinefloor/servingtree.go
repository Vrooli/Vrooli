package baselinefloor

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"
)

// legacyServingTreeDir is the single serving tree the 2026-10-06 layout
// materialized in place. Live instances started before the generation layout
// still run from it; it is reaped like a superseded generation once nothing
// runs from it.
const legacyServingTreeDir = "serving-tree"

// servingGenerationsDir holds one serving tree per capture generation:
// ~/.cache/vrooli/<scenario>/baseline-<slug>/serving/<generation-key>/.
const servingGenerationsDir = "serving"

// servingScenariosDir mirrors the repo's scenarios/ directory inside a serving
// tree. A scenario's relative references assume it lives at
// <repo>/scenarios/<name>.
const servingScenariosDir = "scenarios"

// servingGenerationFile records which capture generation a serving tree was
// materialized from.
const servingGenerationFile = ".serving-generation"

// ServingTreePath resolves the legacy single serving tree of an engagement:
// ~/.cache/vrooli/<scenario>/baseline-<slug>/serving-tree/.
func (s *Store) ServingTreePath(scenario, slug string) string {
	return filepath.Join(s.EngagementDir(scenario, slug), legacyServingTreeDir)
}

// ServingBasePath resolves the directory holding an engagement's serving-tree
// generations.
func (s *Store) ServingBasePath(scenario, slug string) string {
	return filepath.Join(s.EngagementDir(scenario, slug), servingGenerationsDir)
}

// ServingScenarioDir is the directory an engaged serving instance builds and
// runs from: <servingRoot>/scenarios/<scenario>.
func ServingScenarioDir(servingRoot, scenario string) string {
	return filepath.Join(servingRoot, servingScenariosDir, scenario)
}

// ServingGenerationKey maps a capture generation onto its directory name.
func ServingGenerationKey(generation string) string {
	sum := sha256.Sum256([]byte(generation))
	return "g-" + hex.EncodeToString(sum[:8])
}

// ServingSpec describes the serving tree of one capture generation.
type ServingSpec struct {
	Scenario string
	// RepoRoot is the repository the tree links to.
	RepoRoot string
	// RestorePoint is the frozen scenario copy.
	RestorePoint string
	// Generation identifies the capture (capture ID; the manifest CreatedAt
	// for an engagement captured before capture records existed).
	Generation string
	// SharedSources is the repo-shaped shared-source snapshot and Frozen the
	// repo-relative roots in it. Empty Frozen links every shared source to
	// the repository (an unfrozen capture).
	SharedSources string
	Frozen        []string
	// ConfigFallback is the working-tree scenario directory whose .vrooli
	// configuration fills a restore point that has none: restore points
	// captured before the scenario configuration was kept. The lifecycle
	// already reads the manifest from the working tree, so the build reads
	// the same configuration the start resolved.
	ConfigFallback string
	// Ignore filters the restore-point copy (a restore point captured before
	// runtime data was excluded carries databases and binaries). Nil applies
	// the floor patterns.
	Ignore Ignorer
}

// MaterializeServing prepares the repo-shaped tree that an engaged serving
// (live) instance builds and runs from, and returns the scenario directory in
// it.
//
// A scenario's sources address shared repo content by relative paths: Go
// replace directives (../../../packages/api-core, ../../..), pnpm file:
// dependencies, and bundler aliases. The serving tree restores the repo
// layout around the frozen sources:
//
//   - <tree>/scenarios/<scenario> is a real copy of the restore point. It
//     must be a real directory: Node and bundlers resolve module paths
//     through symlinks, so a linked directory would resolve its relative
//     references from the cache again.
//   - Every frozen shared root (packages/api-core/storage, go.mod, ...) is a
//     real copy from the capture's shared-source snapshot, and each of its
//     ancestors is a real directory. A frozen build therefore compiles
//     against the shared packages it was captured with, whatever the working
//     tree holds now.
//   - Every other repository entry is a symlink into repoRoot.
//
// Each generation gets its own directory under base, and a materialized tree
// is never moved or rewritten: a running process keeps resolving the
// absolute paths it computed at startup (node serves ui/dist through
// __dirname), so a restart can build the next generation while the current
// instance still serves the previous one. The tree is built in a staging
// sibling and renamed into place, so an interrupted materialization never
// leaves a partial tree. A repeat call for the same generation only adds
// links for repository entries created since.
func MaterializeServing(base string, spec ServingSpec) (string, error) {
	scenario := strings.TrimSpace(spec.Scenario)
	if scenario == "" || scenario != filepath.Base(scenario) || scenario == "." || scenario == ".." {
		return "", fmt.Errorf("baselinefloor: serving tree: invalid scenario %q", spec.Scenario)
	}
	if strings.TrimSpace(spec.Generation) == "" {
		return "", fmt.Errorf("baselinefloor: serving tree: generation is required")
	}
	if err := ensureDir(spec.RestorePoint); err != nil {
		return "", fmt.Errorf("baselinefloor: serving tree restore point: %w", err)
	}
	if err := ensureDir(filepath.Join(spec.RepoRoot, servingScenariosDir)); err != nil {
		return "", fmt.Errorf("baselinefloor: serving tree repo root: %w", err)
	}
	plan, err := newServingPlan(scenario, spec.Frozen)
	if err != nil {
		return "", err
	}
	if len(plan.frozen) > 0 {
		if err := ensureDir(spec.SharedSources); err != nil {
			return "", fmt.Errorf("baselinefloor: serving tree shared sources: %w", err)
		}
	}
	if err := os.MkdirAll(base, tuning.PermGroupDir); err != nil {
		return "", fmt.Errorf("baselinefloor: serving tree mkdir %q: %w", base, err)
	}
	tree := filepath.Join(base, ServingGenerationKey(spec.Generation))
	if info, err := os.Lstat(tree); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("baselinefloor: serving tree: %q exists and is not a directory", tree)
		}
		if err := plan.overlay(tree, spec, "", true); err != nil {
			return "", err
		}
		return ServingScenarioDir(tree, scenario), nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("baselinefloor: serving tree: stat %q: %w", tree, err)
	}

	staging, err := os.MkdirTemp(base, "."+filepath.Base(tree)+".materializing-")
	if err != nil {
		return "", fmt.Errorf("baselinefloor: serving tree staging: %w", err)
	}
	if err := plan.overlay(staging, spec, "", false); err != nil {
		_ = removeTree(staging)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(staging, servingGenerationFile), []byte(spec.Generation+"\n"), tuning.PermFile); err != nil {
		_ = removeTree(staging)
		return "", fmt.Errorf("baselinefloor: serving tree generation: %w", err)
	}
	if err := os.Chmod(staging, tuning.PermGroupDir); err != nil {
		_ = removeTree(staging)
		return "", fmt.Errorf("baselinefloor: serving tree mode: %w", err)
	}
	if err := os.Rename(staging, tree); err != nil {
		_ = removeTree(staging)
		if info, statErr := os.Stat(tree); statErr == nil && info.IsDir() {
			// A concurrent materialization won the rename; its tree is complete.
			return ServingScenarioDir(tree, scenario), nil
		}
		return "", fmt.Errorf("baselinefloor: serving tree publish %q: %w", tree, err)
	}
	return ServingScenarioDir(tree, scenario), nil
}

// servingPlan classifies repo-relative paths: the scenario copy, frozen
// roots, and real directories (their ancestors). Anything else is a link.
type servingPlan struct {
	scenarioRel string
	frozen      map[string]struct{}
	realDirs    map[string]struct{}
	children    map[string][]string
}

func newServingPlan(scenario string, frozen []string) (servingPlan, error) {
	plan := servingPlan{
		scenarioRel: servingScenariosDir + "/" + scenario,
		frozen:      map[string]struct{}{},
		realDirs:    map[string]struct{}{},
		children:    map[string][]string{},
	}
	addChild := func(rel string) {
		parent := path.Dir(rel)
		if parent == "." {
			parent = ""
		}
		for _, existing := range plan.children[parent] {
			if existing == path.Base(rel) {
				return
			}
		}
		plan.children[parent] = append(plan.children[parent], path.Base(rel))
	}
	addAncestors := func(rel string) {
		for dir := path.Dir(rel); dir != "." && dir != ""; dir = path.Dir(dir) {
			plan.realDirs[dir] = struct{}{}
			addChild(dir)
		}
		addChild(rel)
	}
	addAncestors(plan.scenarioRel)
	for _, raw := range frozen {
		rel, err := cleanRepoRel(raw)
		if err != nil {
			return servingPlan{}, err
		}
		if rel == plan.scenarioRel || strings.HasPrefix(rel, plan.scenarioRel+"/") || strings.HasPrefix(plan.scenarioRel, rel+"/") || rel == ".git" || strings.HasPrefix(rel, ".git/") {
			return servingPlan{}, fmt.Errorf("baselinefloor: serving tree: frozen root %q overlaps the scenario copy or VCS metadata", rel)
		}
		plan.frozen[rel] = struct{}{}
		addAncestors(rel)
	}
	for rel := range plan.frozen {
		if _, isAncestor := plan.realDirs[rel]; isAncestor {
			return servingPlan{}, fmt.Errorf("baselinefloor: serving tree: frozen root %q contains another frozen root", rel)
		}
	}
	return plan, nil
}

// overlay fills dstRoot/rel. In refresh mode the tree already exists: frozen
// and copied entries are left alone and only missing links are added.
func (p servingPlan) overlay(dstRoot string, spec ServingSpec, rel string, refresh bool) error {
	dstDir := filepath.Join(dstRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(dstDir, tuning.PermGroupDir); err != nil {
		return fmt.Errorf("baselinefloor: serving tree mkdir %q: %w", dstDir, err)
	}
	names := map[string]struct{}{}
	repoDir := filepath.Join(spec.RepoRoot, filepath.FromSlash(rel))
	if entries, err := os.ReadDir(repoDir); err == nil {
		for _, entry := range entries {
			names[entry.Name()] = struct{}{}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("baselinefloor: serving tree read %q: %w", repoDir, err)
	}
	for _, child := range p.children[rel] {
		names[child] = struct{}{}
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		childRel := joinSlash(rel, name)
		if rel == "" && (name == ".git" || name == servingGenerationFile) {
			// Build tools that probe for version control must never see a
			// working tree that does not match the index.
			continue
		}
		dst := filepath.Join(dstRoot, filepath.FromSlash(childRel))
		_, frozen := p.frozen[childRel]
		_, real := p.realDirs[childRel]
		switch {
		case childRel == p.scenarioRel:
			if refresh {
				continue
			}
			if err := materializeScenarioCopy(spec, dst); err != nil {
				return err
			}
		case frozen:
			if refresh {
				continue
			}
			if err := copyFrozenRoot(spec.SharedSources, childRel, dst); err != nil {
				return err
			}
		case real:
			if err := p.overlay(dstRoot, spec, childRel, refresh); err != nil {
				return err
			}
		default:
			if _, inRepo := names[name]; !inRepo {
				continue
			}
			if err := ensureSymlink(filepath.Join(repoDir, name), dst); err != nil {
				return err
			}
		}
	}
	return nil
}

func materializeScenarioCopy(spec ServingSpec, dst string) error {
	opts := CopyOptions{Exclude: defaultExcludes, Ignore: spec.Ignore, Reflink: true}
	if opts.Ignore == nil {
		opts.Ignore = floorIgnorer
	}
	if _, err := CopyTree(spec.RestorePoint, dst, opts); err != nil {
		return fmt.Errorf("baselinefloor: serving tree copy %q: %w", spec.RestorePoint, err)
	}
	// Carry the restore point's mode onto the copy's root.
	if info, err := os.Stat(spec.RestorePoint); err == nil {
		if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
			return fmt.Errorf("baselinefloor: serving tree mode: %w", err)
		}
	}
	configDir := filepath.Join(dst, repocontractmeta.ProjectConfigDir)
	if _, err := os.Stat(configDir); errors.Is(err, fs.ErrNotExist) && strings.TrimSpace(spec.ConfigFallback) != "" {
		fallback := filepath.Join(spec.ConfigFallback, repocontractmeta.ProjectConfigDir)
		if info, statErr := os.Stat(fallback); statErr == nil && info.IsDir() {
			if _, err := CopyTree(fallback, configDir, CopyOptions{Exclude: defaultExcludes, Ignore: configFallbackIgnorer, Reflink: true}); err != nil {
				return fmt.Errorf("baselinefloor: serving tree config fallback: %w", err)
			}
		}
	}
	return nil
}

// configFallbackIgnorer filters a working-tree .vrooli directory copied into
// a legacy serving copy. Paths are relative to .vrooli itself.
var configFallbackIgnorer = NewIgnoreMatcher(IgnoreSpec{Patterns: append(FloorIgnorePatterns(),
	"secrets.json", "*.lock", "*.local.json", "state/", "runtime/", "generated-service.json", "operator-state.json")})

func copyFrozenRoot(snapshot, rel, dst string) error {
	src := filepath.Join(snapshot, filepath.FromSlash(rel))
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("baselinefloor: serving tree frozen root %s: %w", rel, err)
	}
	if info.IsDir() {
		if _, err := CopyTree(src, dst, CopyOptions{Exclude: map[string]struct{}{}, Ignore: noIgnore{}, Reflink: true}); err != nil {
			return fmt.Errorf("baselinefloor: serving tree frozen root %s: %w", rel, err)
		}
		return nil
	}
	if _, _, err := copyRegularFile(src, dst, info.Mode().Perm(), true); err != nil {
		return fmt.Errorf("baselinefloor: serving tree frozen root %s: %w", rel, err)
	}
	return nil
}

type noIgnore struct{}

func (noIgnore) Ignored(string, bool) bool { return false }

// ReapServingTrees removes serving-tree generations of an engagement other
// than keep (a generation key) for which inUse reports false. inUse receives
// the absolute tree root; it must report true whenever any process might run
// from inside it. The legacy single serving tree is treated the same way.
// Errors are collected; reaping never fails a materialization.
func (s *Store) ReapServingTrees(scenario, slug, keep string, inUse func(treeRoot string) bool) ([]string, error) {
	var candidates []string
	base := s.ServingBasePath(scenario, slug)
	if entries, err := os.ReadDir(base); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() || name == keep || strings.HasPrefix(name, ".") {
				continue
			}
			candidates = append(candidates, filepath.Join(base, name))
		}
	}
	if info, err := os.Lstat(s.ServingTreePath(scenario, slug)); err == nil && info.IsDir() {
		candidates = append(candidates, s.ServingTreePath(scenario, slug))
	}
	var reaped []string
	var errs []error
	for _, tree := range candidates {
		if inUse == nil || inUse(tree) {
			continue
		}
		if err := removeTree(tree); err != nil {
			errs = append(errs, err)
			continue
		}
		reaped = append(reaped, tree)
	}
	return reaped, errors.Join(errs...)
}

func ensureSymlink(target, link string) error {
	info, err := os.Lstat(link)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return fmt.Errorf("baselinefloor: serving tree stat %q: %w", link, err)
	case info.Mode()&fs.ModeSymlink == 0:
		return nil
	default:
		current, readErr := os.Readlink(link)
		if readErr == nil && current == target {
			return nil
		}
		if removeErr := os.Remove(link); removeErr != nil {
			return fmt.Errorf("baselinefloor: serving tree replace link %q: %w", link, removeErr)
		}
	}
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("baselinefloor: serving tree link %q: %w", link, err)
	}
	return nil
}
