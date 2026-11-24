package catalogcoverage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// BuildRevisionIndex hashes every catalog asset's own inputs — its catalog
// declaration, its manifest, and the source and lock of its latest released
// version — and then folds in the revisions of the assets it depends on.
//
// Both halves are load-bearing, and neither used to work. The manifest glob
// was one directory short of where manifests actually live, so the loop that
// hashes the manifest and the version source never executed and an asset's
// revision was a digest of its catalog declaration alone: editing a
// component's source left every cached per-asset gate verdict looking current.
// Folding dependencies in is what makes a foundation change invalidate the
// assets built on it, which the generated per-version locks describe exactly.
//
// The fold is memoized and cycle-guarded. The dependency-rank gate already
// proves the graph is acyclic, so the guard is a safety net rather than an
// expected path — an asset caught in one falls back to its own digest, which
// is stale-safe in the only direction that matters: it can force an extra
// recomputation, never suppress a needed one.
func BuildRevisionIndex(root string) (map[string]string, error) {
	scenarioRoot := resolveScenarioRoot(root)

	own := map[string]hash.Hash{}
	declarationPaths, err := filepath.Glob(filepath.Join(scenarioRoot, "catalog", "assets", "*", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(declarationPaths)
	for _, path := range declarationPaths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var doc struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		if json.Unmarshal(data, &doc) != nil || doc.Asset.ID == "" {
			continue
		}
		h := sha256.New()
		h.Write(data)
		h.Write([]byte{0})
		own[doc.Asset.ID] = h
	}

	dependencies := map[string][]string{}
	catalogByLibrary := map[string]string{}
	type manifestRecord struct {
		path      string
		data      []byte
		libraryID string
		requires  []string
	}
	var manifests []manifestRecord
	paths, globErr := filepath.Glob(filepath.Join(scenarioRoot, "library", "*", "*", "component.json"))
	if globErr != nil {
		return nil, globErr
	}
	sort.Strings(paths)
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var manifest struct {
			CatalogID string `json:"catalogId"`
			LibraryID string `json:"libraryId"`
			Latest    string `json:"latest"`
		}
		if json.Unmarshal(data, &manifest) != nil {
			continue
		}
		record := manifestRecord{path: path, data: data, libraryID: manifest.LibraryID}
		// Edges come from the version's generated lock, never from the
		// manifest's dependencies array. That array is per-component while
		// immutability is per-version, and it is empty on all 234 manifests
		// in the live tree — folding on it would silently produce a graph
		// with no edges at all. The lock is generated from the version's
		// real imports, so it is the only per-version dependency record
		// that exists.
		if lock, lockErr := os.ReadFile(filepath.Join(filepath.Dir(path), "versions", manifest.Latest, "dependencies.json")); lockErr == nil {
			var doc struct {
				Dependencies []struct {
					LibraryID string `json:"libraryId"`
				} `json:"dependencies"`
			}
			if json.Unmarshal(lock, &doc) == nil {
				for _, dependency := range doc.Dependencies {
					record.requires = append(record.requires, dependency.LibraryID)
				}
			}
		}
		if manifest.CatalogID != "" && manifest.LibraryID != "" {
			catalogByLibrary[manifest.LibraryID] = manifest.CatalogID
		}
		manifests = append(manifests, record)
		if manifest.CatalogID == "" {
			continue
		}
		h, ok := own[manifest.CatalogID]
		if !ok {
			continue
		}
		h.Write(data)
		h.Write([]byte{0})
		// Fold every live version, not only latest. A historical but live
		// major line can be imported by an adopter, so changing it must
		// invalidate evidence just as changing latest does.
		versionDirs, versionGlobErr := filepath.Glob(filepath.Join(filepath.Dir(path), "versions", "*"))
		if versionGlobErr != nil {
			return nil, versionGlobErr
		}
		sort.Strings(versionDirs)
		for _, versionDir := range versionDirs {
			if info, statErr := os.Stat(versionDir); statErr != nil || !info.IsDir() {
				continue
			}
			h.Write([]byte(filepath.Base(versionDir)))
			h.Write([]byte{0})
			for _, versionPath := range versionSourcePaths(versionDir) {
				versionData, versionErr := revisionFileData(versionPath)
				if versionErr != nil {
					return nil, versionErr
				}
				h.Write(versionData)
				h.Write([]byte{0})
			}
		}
	}
	// The manifest's own requires list is the declared edge. It is resolved to
	// catalog ids in a second pass because an asset may be declared after the
	// assets it depends on have already been read.
	for _, record := range manifests {
		catalogID := catalogByLibrary[record.libraryID]
		if catalogID == "" {
			continue
		}
		for _, libraryID := range record.requires {
			if dependency := catalogByLibrary[libraryID]; dependency != "" && dependency != catalogID {
				dependencies[catalogID] = append(dependencies[catalogID], dependency)
			}
		}
	}

	digests := make(map[string]string, len(own))
	for id, h := range own {
		digests[id] = hex.EncodeToString(h.Sum(nil))
	}

	index := make(map[string]string, len(digests))
	visiting := map[string]bool{}
	var resolve func(string) string
	resolve = func(id string) string {
		if revision, ok := index[id]; ok {
			return revision
		}
		if visiting[id] {
			return digests[id]
		}
		visiting[id] = true
		requires := append([]string(nil), dependencies[id]...)
		sort.Strings(requires)
		h := sha256.New()
		h.Write([]byte(digests[id]))
		h.Write([]byte{0})
		seen := map[string]bool{}
		for _, dependency := range requires {
			if seen[dependency] || digests[dependency] == "" {
				continue
			}
			seen[dependency] = true
			h.Write([]byte(resolve(dependency)))
			h.Write([]byte{0})
		}
		delete(visiting, id)
		revision := hex.EncodeToString(h.Sum(nil))
		index[id] = revision
		return revision
	}
	for id := range digests {
		resolve(id)
	}
	return index, nil
}

// CurrentRevision is a content hash for the catalog declaration, its linked
// implementation, and everything that implementation depends on. Persisted
// evidence with another revision is stale.
func CurrentRevision(root, assetID string) (string, error) {
	index, err := BuildRevisionIndex(root)
	if err != nil {
		return "", err
	}
	revision, ok := index[assetID]
	if !ok {
		return "", fmt.Errorf("catalog asset %q not found", assetID)
	}
	return revision, nil
}

// CurrentRevisionForVersion returns the revision of one materialized version
// and the dependency revisions selected by that version's lock. It is the
// identity used by version-pinned component-test reports; the aggregate index
// above remains the cheap latest-version projection for catalog coverage.
func CurrentRevisionForVersion(root, libraryID, version string) (string, error) {
	scenarioRoot := resolveScenarioRoot(root)
	resolver, err := NewVersionRevisionResolver(scenarioRoot)
	if err != nil {
		return "", err
	}
	return resolver.Resolve(libraryID, version)
}

type versionManifest struct {
	path      string
	data      []byte
	catalogID string
}

// VersionRevisionResolver indexes manifests once and resolves materialized
// version revisions through that index. Callers that resolve many versions
// (for example the freshness gate) must share one resolver; constructing one
// per version needlessly rereads the entire manifest corpus.
type VersionRevisionResolver struct {
	root      string
	manifests map[string]versionManifest
	cache     map[string]string
}

func NewVersionRevisionResolver(root string) (*VersionRevisionResolver, error) {
	root = resolveScenarioRoot(root)
	manifestPaths, err := filepath.Glob(filepath.Join(root, "library", "*", "*", "component.json"))
	if err != nil {
		return nil, err
	}
	manifests := make(map[string]versionManifest, len(manifestPaths))
	for _, path := range manifestPaths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var manifest struct {
			LibraryID string `json:"libraryId"`
			CatalogID string `json:"catalogId"`
		}
		if json.Unmarshal(data, &manifest) != nil || manifest.LibraryID == "" {
			continue
		}
		manifests[manifest.LibraryID] = versionManifest{path: path, data: data, catalogID: manifest.CatalogID}
	}
	return &VersionRevisionResolver{root: root, manifests: manifests, cache: map[string]string{}}, nil
}

func (r *VersionRevisionResolver) Resolve(libraryID, version string) (string, error) {
	key := libraryID + "@" + version
	if revision, ok := r.cache[key]; ok {
		return revision, nil
	}
	revision, err := r.resolve(libraryID, version, map[string]bool{})
	if err != nil {
		return "", err
	}
	r.cache[key] = revision
	return revision, nil
}

func (r *VersionRevisionResolver) resolve(libraryID, version string, visiting map[string]bool) (string, error) {
	key := libraryID + "@" + version
	if visiting[key] {
		return "", fmt.Errorf("dependency revision cycle at %s", key)
	}
	visiting[key] = true
	defer delete(visiting, key)

	manifest, ok := r.manifests[libraryID]
	if !ok {
		return "", fmt.Errorf("library asset %q not found", libraryID)
	}
	versionDir := filepath.Join(filepath.Dir(manifest.path), "versions", version)
	if info, statErr := os.Stat(versionDir); statErr != nil || !info.IsDir() {
		return "", fmt.Errorf("version %s@%s is not materialized", libraryID, version)
	}
	h := sha256.New()
	if manifest.catalogID != "" {
		parts := strings.SplitN(manifest.catalogID, ".", 2)
		if len(parts) == 2 {
			declaration := filepath.Join(r.root, "catalog", "assets", parts[0], parts[1]+".json")
			if data, readErr := os.ReadFile(declaration); readErr == nil {
				h.Write(data)
				h.Write([]byte{0})
			}
		}
	}
	h.Write(manifest.data)
	h.Write([]byte{0})
	for _, sourcePath := range versionSourcePaths(versionDir) {
		data, readErr := revisionFileData(sourcePath)
		if readErr != nil {
			return "", readErr
		}
		h.Write([]byte(filepath.Base(sourcePath)))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	lockData, lockErr := os.ReadFile(filepath.Join(versionDir, "dependencies.json"))
	if lockErr == nil {
		var lock struct {
			Dependencies []struct {
				LibraryID string `json:"libraryId"`
				Version   string `json:"version"`
				Major     int    `json:"major"`
				Observed  string `json:"observed"`
			} `json:"dependencies"`
		}
		if json.Unmarshal(lockData, &lock) == nil {
			for _, dependency := range lock.Dependencies {
				observed := dependency.Observed
				if observed == "" {
					observed = dependency.Version
				}
				if dependency.Major > 0 {
					observed = resolveMajorVersion(r.root, dependency.LibraryID, dependency.Major, observed)
				}
				child, childErr := r.resolve(dependency.LibraryID, observed, visiting)
				if childErr != nil {
					return "", childErr
				}
				h.Write([]byte(dependency.LibraryID))
				h.Write([]byte{0})
				h.Write([]byte(child))
				h.Write([]byte{0})
			}
		}
	}
	revision := hex.EncodeToString(h.Sum(nil))
	r.cache[key] = revision
	return revision, nil
}

func resolveMajorVersion(root, libraryID string, major int, fallback string) string {
	name := strings.TrimPrefix(libraryID, "react-component-library:")
	paths, _ := filepath.Glob(filepath.Join(root, "library", "*", name, "versions", fmt.Sprintf("%d.*", major)))
	versions := make([]string, 0, len(paths))
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			versions = append(versions, filepath.Base(path))
		}
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		return fallback
	}
	return versions[len(versions)-1]
}

// resolveScenarioRoot accepts the repository root used by API handlers, the
// scenario root used by local tools, and the library root used by the
// component service. Keeping this normalization at the revision boundary
// prevents evidence from silently disappearing when those callers use their
// own legitimate source-root abstraction.
func resolveScenarioRoot(root string) string {
	clean := filepath.Clean(root)
	if filepath.Base(clean) == "library" {
		return filepath.Dir(clean)
	}
	if _, err := os.Stat(filepath.Join(clean, "catalog")); err == nil {
		return clean
	}
	candidate := filepath.Join(clean, "scenarios", "react-component-library")
	if _, err := os.Stat(filepath.Join(candidate, "catalog")); err == nil {
		return candidate
	}
	return clean
}

func versionSourcePaths(versionDir string) []string {
	var paths []string
	for _, extension := range []string{"*.tsx", "*.ts"} {
		matches, _ := filepath.Glob(filepath.Join(versionDir, extension))
		paths = append(paths, matches...)
	}
	for _, generated := range []string{"dependencies.json", "story.json"} {
		path := filepath.Join(versionDir, generated)
		if fileExists(path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

// revisionFileData removes generator bookkeeping that is intentionally
// refreshed on every catalog build. The lock's dependency choices remain in
// the digest; only resolvedAt is non-semantic and must not invalidate test
// evidence or cache entries by itself. Story contracts remain fully semantic:
// changing a story must invalidate its browser evidence.
func revisionFileData(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil || filepath.Base(path) != "dependencies.json" {
		return data, err
	}
	var lock map[string]any
	if json.Unmarshal(data, &lock) != nil {
		return data, nil
	}
	delete(lock, "resolvedAt")
	canonical, marshalErr := json.Marshal(lock)
	if marshalErr != nil {
		return data, nil
	}
	return canonical, nil
}
