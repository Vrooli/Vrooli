package repocontract

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var targetKindSet = map[TargetKind]struct{}{
	TargetKindScenario: {}, TargetKindResource: {}, TargetKindTool: {}, TargetKindSafeguard: {},
	TargetKindTeam: {}, TargetKindPackage: {}, TargetKindControlPlane: {}, TargetKindDocs: {},
}

// EnumerateTargets discovers all concrete targets from the loaded contract.
// It intentionally does not contain repository-specific names: adding a
// matching directory changes the result without changing this code.
func (c *Contract) EnumerateTargets(repoRoot string) ([]Target, error) {
	if c == nil {
		return nil, &Error{Kind: ErrInvalidInput, Message: "contract is required"}
	}
	root, err := filepath.Abs(strings.TrimSpace(repoRoot))
	if err != nil || strings.TrimSpace(repoRoot) == "" {
		return nil, &Error{Kind: ErrInvalidInput, Message: "repo root is required", Err: err}
	}
	fsys := os.DirFS(root)
	var targets []Target
	seen := map[string]struct{}{}
	for rawKind, spec := range c.doc.Targets.Kinds {
		kind := TargetKind(rawKind)
		for _, pattern := range spec.Roots {
			matches, err := doublestar.Glob(fsys, pattern)
			if err != nil {
				return nil, &Error{Kind: ErrInvalidContract, Message: "enumerate target glob", Details: pattern, Err: err}
			}
			for _, rel := range matches {
				rel = filepath.ToSlash(filepath.Clean(rel))
				info, err := fs.Stat(fsys, rel)
				if err != nil || !info.IsDir() || targetExcluded(rel, spec.Exclude) {
					continue
				}
				if spec.Marker != "" {
					marker := filepath.Join(root, filepath.FromSlash(rel), filepath.FromSlash(spec.Marker))
					if _, err := os.Stat(marker); err != nil {
						continue
					}
				}
				id := filepath.Base(rel)
				if kind == TargetKindTeam {
					id = teamOwnerID(filepath.Join(root, filepath.FromSlash(rel), filepath.FromSlash(spec.Marker)))
				}
				if kind == TargetKindControlPlane || kind == TargetKindDocs {
					id = rel
				}
				key := string(kind) + "\x1e" + id
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				targets = append(targets, Target{Kind: kind, ID: id, Root: rel})
			}
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		return targets[i].ID < targets[j].ID
	})
	return targets, nil
}

func teamOwnerID(markerPath string) string {
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return ""
	}
	var manifest struct {
		Contract struct {
			Team string `json:"team"`
		} `json:"contract"`
	}
	if json.Unmarshal(data, &manifest) != nil || strings.TrimSpace(manifest.Contract.Team) == "" {
		return ""
	}
	return strings.TrimSpace(manifest.Contract.Team)
}

func targetExcluded(rel string, excludes []string) bool {
	for _, pattern := range excludes {
		matched, err := MatchRepoGlob(pattern, rel)
		if err == nil && matched {
			return true
		}
		// Exclusions may describe children of a positional root. The root target
		// remains, but its identity is still unambiguous and appears once.
	}
	return false
}

// Target returns a concrete target by its kind and ID.
func (c *Contract) Target(repoRoot string, kind TargetKind, id string) (Target, error) {
	targets, err := c.EnumerateTargets(repoRoot)
	if err != nil {
		return Target{}, err
	}
	for _, target := range targets {
		if target.Kind == kind && target.ID == id {
			return target, nil
		}
	}
	// Manifest-backed targets use the owner slug as their canonical ID, while
	// operators commonly address them by the repo-relative marker directory
	// (for example team:docs/marketing). Accept that ergonomic alias without
	// changing the stable identity stored in a run or finding.
	for _, target := range targets {
		if target.Kind == kind && target.Root == filepath.ToSlash(filepath.Clean(id)) {
			return target, nil
		}
	}
	return Target{}, &Error{Kind: ErrNotFound, Message: "target not found", Details: fmt.Sprintf("%s:%s", kind, id)}
}
