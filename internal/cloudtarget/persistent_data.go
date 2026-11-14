package cloudtarget

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Persistent data lives outside every release tree so a code update never
// carries it by copying and a retired release never takes it along:
//
//	<deployment-dir>/persistent-data/<binding-id>/           declared bindings
//	<deployment-dir>/persistent-data/legacy/<scenario>/<rel>  legacy carries
//
// Activation binds each declared path inside the candidate release tree to
// its persistent directory with a symlink. The first activation that finds
// data at the same relative path in the predecessor release (or, for a
// deployment converted from the in-place layout, beneath the legacy root)
// adopts it by rename: nothing is copied and nothing is deleted. A mutable
// directory the heuristic would have preserved but no binding or carry names
// is reported as unmapped and left exactly where it is.
const (
	persistentDataDirName = "persistent-data"
	legacyCarryDirName    = "legacy"
)

// DataBinding maps one declared persistent-data binding onto a path inside a
// scenario directory of the release tree.
type DataBinding struct {
	ID       string `json:"id"`
	Scenario string `json:"scenario"`
	Path     string `json:"path"`
}

// ParseDataBindingFlags decodes repeated `--data-binding <id>=<scenario>/<path>`
// values.
func ParseDataBindingFlags(values []string) ([]DataBinding, error) {
	bindings := make([]DataBinding, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		id, location, ok := strings.Cut(strings.TrimSpace(value), "=")
		if !ok || strings.TrimSpace(id) == "" {
			return nil, refuse(CodeInvalidArgument, "data binding %q must be <id>=<scenario>/<path>", value)
		}
		if err := validIdentifier("data binding id", id); err != nil {
			return nil, err
		}
		scenario, rel, err := splitScenarioPath(location)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			return nil, refuse(CodeInvalidArgument, "data binding %q is declared twice", id)
		}
		seen[id] = true
		bindings = append(bindings, DataBinding{ID: id, Scenario: scenario, Path: rel})
	}
	return bindings, nil
}

// ParseLegacyCarryFlags decodes repeated `--legacy-carry <scenario>/<path>`
// values: heuristic-preserved directories the operator chose to carry
// forward without declaring a binding yet.
func ParseLegacyCarryFlags(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		scenario, rel, err := splitScenarioPath(value)
		if err != nil {
			return nil, err
		}
		out = append(out, scenario+"/"+rel)
	}
	sort.Strings(out)
	return out, nil
}

func splitScenarioPath(location string) (string, string, error) {
	location = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(location), "scenarios/"))
	scenario, rel, ok := strings.Cut(location, "/")
	if !ok {
		return "", "", refuse(CodeInvalidArgument, "location %q must be <scenario>/<relative path>", location)
	}
	if err := validIdentifier("scenario id", scenario); err != nil {
		return "", "", err
	}
	rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(rel))))
	if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return "", "", refuse(CodeInvalidArgument, "path %q must stay inside the scenario directory", rel)
	}
	return scenario, rel, nil
}

// bindingPlan is what one activation did for persistent data.
type bindingPlan struct {
	Bound    []map[string]any `json:"bound"`
	Unmapped []string         `json:"unmapped"`
}

// PersistentDataDir returns the persistent root of a deployment.
func (s *Store) PersistentDataDir(deploymentID string) (string, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, persistentDataDirName), nil
}

// bindPersistentData prepares the candidate release tree: every declared
// binding and legacy carry becomes a symlink into the persistent root, data
// found in the predecessor or legacy tree is adopted by rename, and every
// heuristic-preserved directory nobody named is reported and left in place.
func (s *Store) bindPersistentData(deploymentID, releaseDir, previousReleaseDir, legacyRoot string, scenarios []string, bindings []DataBinding, legacyCarry []string) (bindingPlan, error) {
	plan := bindingPlan{Bound: []map[string]any{}, Unmapped: []string{}}
	if len(bindings) == 0 && len(legacyCarry) == 0 && legacyRoot == "" && previousReleaseDir == "" {
		return plan, nil
	}
	root, err := s.PersistentDataDir(deploymentID)
	if err != nil {
		return plan, err
	}
	type target struct {
		id, scenario, rel, persistent string
	}
	targets := make([]target, 0, len(bindings)+len(legacyCarry))
	for _, b := range bindings {
		targets = append(targets, target{id: b.ID, scenario: b.Scenario, rel: b.Path, persistent: filepath.Join(root, b.ID)})
	}
	for _, carry := range legacyCarry {
		scenario, rel, _ := strings.Cut(carry, "/")
		targets = append(targets, target{id: "legacy:" + carry, scenario: scenario, rel: rel, persistent: filepath.Join(root, legacyCarryDirName, scenario, filepath.FromSlash(rel))})
	}
	covered := map[string][]string{}
	for _, t := range targets {
		linkPath := filepath.Join(releaseDir, "scenarios", t.scenario, filepath.FromSlash(t.rel))
		source, shippedAside := "", ""
		if _, err := os.Lstat(t.persistent); errors.Is(err, os.ErrNotExist) {
			for _, candidateRoot := range []string{previousReleaseDir, legacyRoot} {
				if candidateRoot == "" {
					continue
				}
				candidate := filepath.Join(candidateRoot, "scenarios", t.scenario, filepath.FromSlash(t.rel))
				if info, err := os.Lstat(candidate); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
					source = candidate
					break
				}
			}
			if err := os.MkdirAll(filepath.Dir(t.persistent), 0o755); err != nil {
				return plan, fail(CodeStoreIO, "prepare persistent data root: %v", err)
			}
			if source != "" {
				if err := os.Rename(source, t.persistent); err != nil {
					return plan, fail(CodeStoreIO, "adopt %s into %s: %v", source, t.persistent, err)
				}
			} else if info, err := os.Lstat(linkPath); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				// The release ships initial content for the binding: it
				// becomes the first persistent generation.
				if err := os.Rename(linkPath, t.persistent); err != nil {
					return plan, fail(CodeStoreIO, "adopt release seed %s: %v", linkPath, err)
				}
				source = linkPath
			} else if err := os.MkdirAll(t.persistent, 0o755); err != nil {
				return plan, fail(CodeStoreIO, "create persistent directory %s: %v", t.persistent, err)
			}
		}
		if info, err := os.Lstat(linkPath); err == nil {
			switch {
			case info.Mode()&os.ModeSymlink != 0:
				if err := os.Remove(linkPath); err != nil {
					return plan, fail(CodeStoreIO, "replace stale link %s: %v", linkPath, err)
				}
			case info.IsDir():
				// The release ships content (placeholders, seeds) at a path
				// the persistent store already owns: the shipped tree is
				// moved aside inside the release, never deleted, and the
				// persistent data wins.
				aside := linkPath + ".shipped"
				if err := os.RemoveAll(aside); err != nil {
					return plan, fail(CodeStoreIO, "clear previous shipped copy %s: %v", aside, err)
				}
				if err := os.Rename(linkPath, aside); err != nil {
					return plan, fail(CodeStoreIO, "move shipped content aside %s: %v", linkPath, err)
				}
				shippedAside = aside
			default:
				return plan, refuse(CodeDataBindingConflict, "release ships a file at %s where binding %s expects a directory", filepath.ToSlash(filepath.Join("scenarios", t.scenario, t.rel)), t.id)
			}
		} else if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			return plan, fail(CodeStoreIO, "prepare %s: %v", filepath.Dir(linkPath), err)
		}
		if err := os.Symlink(t.persistent, linkPath); err != nil {
			return plan, fail(CodeStoreIO, "bind %s: %v", linkPath, err)
		}
		covered[t.scenario] = append(covered[t.scenario], t.rel)
		entry := map[string]any{"id": t.id, "scenario": t.scenario, "path": t.rel, "persistent": t.persistent}
		if source != "" {
			entry["adopted_from"] = source
		}
		if shippedAside != "" {
			entry["shipped_aside"] = shippedAside
		}
		plan.Bound = append(plan.Bound, entry)
	}
	// Report, never touch, what the heuristic would have preserved but
	// nothing named. The predecessor and legacy trees stay as they are.
	for _, scenario := range scenarios {
		var known []Binding
		for _, rel := range covered[scenario] {
			known = append(known, Binding{ID: rel, Path: rel})
		}
		for _, candidateRoot := range []string{previousReleaseDir, legacyRoot} {
			if candidateRoot == "" {
				continue
			}
			report, err := Inventory(InventoryRequest{DeploymentID: deploymentID, Workdir: candidateRoot, Scenario: scenario, Bindings: known})
			if err != nil {
				continue
			}
			for _, entry := range report.Entries {
				if !entry.Covered {
					plan.Unmapped = appendUnique(plan.Unmapped, fmt.Sprintf("%s:%s/%s", filepath.Base(candidateRoot), scenario, entry.Path))
				}
			}
		}
	}
	sort.Strings(plan.Unmapped)
	return plan, nil
}

func appendUnique(list []string, value string) []string {
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}
