package repocontract

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

var targetKindSet = map[TargetKind]struct{}{
	TargetKindScenario: {}, TargetKindResource: {}, TargetKindTool: {}, TargetKindSafeguard: {},
	TargetKindTeam: {}, TargetKindPackage: {}, TargetKindControlPlane: {}, TargetKindDocs: {}, TargetKindProject: {},
}

// DefaultTargetIndexTTL bounds target discovery staleness after filesystem changes.
const DefaultTargetIndexTTL = 10 * time.Second

var (
	targetIndexNow   = time.Now
	targetIndexStat  = os.Stat
	targetIndexTTL   = DefaultTargetIndexTTL
	targetIndexMu    sync.Mutex
	targetIndexCache = map[string]targetIndexCacheEntry{}
)

type targetIndexCacheEntry struct {
	index       *TargetIndex
	contractMod time.Time
	expiresAt   time.Time
}

// TargetIndex resolves repository-relative paths to concrete targets.
type TargetIndex struct {
	targets []Target
}

// NewTargetIndex creates an immutable path resolver from concrete targets.
func NewTargetIndex(targets []Target) *TargetIndex {
	copyTargets := append([]Target(nil), targets...)
	sort.SliceStable(copyTargets, func(i, j int) bool {
		leftRoot := normalizedTargetRoot(copyTargets[i].Root)
		rightRoot := normalizedTargetRoot(copyTargets[j].Root)
		if len(leftRoot) != len(rightRoot) {
			return len(leftRoot) > len(rightRoot)
		}
		if leftRoot != rightRoot {
			return leftRoot < rightRoot
		}
		if copyTargets[i].Kind != copyTargets[j].Kind {
			return copyTargets[i].Kind < copyTargets[j].Kind
		}
		return copyTargets[i].ID < copyTargets[j].ID
	})
	return &TargetIndex{targets: copyTargets}
}

// Lookup returns the longest-root target containing path.
func (i *TargetIndex) Lookup(path string) (Target, bool) {
	if i == nil {
		return Target{}, false
	}
	path = normalizedTargetPath(path)
	for _, target := range i.targets {
		root := normalizedTargetRoot(target.Root)
		if root == "" || path == root || strings.HasPrefix(path, root+"/") {
			return target, true
		}
	}
	return Target{}, false
}

// NewTargetIndex returns a cached path resolver for the repository. The cache
// refreshes when the contract mtime changes or the TTL expires.
func (c *Contract) NewTargetIndex(repoRoot string) (*TargetIndex, error) {
	if c == nil {
		return nil, &Error{Kind: ErrInvalidInput, Message: "contract is required"}
	}
	root, err := filepath.Abs(strings.TrimSpace(repoRoot))
	if err != nil || strings.TrimSpace(repoRoot) == "" {
		return nil, &Error{Kind: ErrInvalidInput, Message: "repo root is required", Err: err}
	}
	now := targetIndexNow()
	contractInfo, statErr := targetIndexStat(filepath.Join(root, filepath.FromSlash(defaultContractRelPath)))
	var contractMod time.Time
	if statErr == nil {
		contractMod = contractInfo.ModTime()
	}

	targetIndexMu.Lock()
	defer targetIndexMu.Unlock()
	if cached, ok := targetIndexCache[root]; ok && cached.contractMod.Equal(contractMod) && now.Before(cached.expiresAt) {
		return cached.index, nil
	}
	targets, err := c.EnumerateTargets(root)
	if err != nil {
		return nil, err
	}
	index := NewTargetIndex(targets)
	targetIndexCache[root] = targetIndexCacheEntry{
		index:       index,
		contractMod: contractMod,
		expiresAt:   now.Add(targetIndexTTL),
	}
	return index, nil
}

// Lookup resolves a repository-relative path using the cached target index.
func (c *Contract) Lookup(repoRoot, path string) (Target, bool, error) {
	index, err := c.NewTargetIndex(repoRoot)
	if err != nil {
		return Target{}, false, err
	}
	target, ok := index.Lookup(path)
	return target, ok, nil
}

func normalizedTargetPath(path string) string {
	path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if path == "." {
		return ""
	}
	return strings.TrimPrefix(path, "./")
}

func normalizedTargetRoot(root string) string {
	return normalizedTargetPath(root)
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
				if kind == TargetKindProject {
					id = "repo"
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
