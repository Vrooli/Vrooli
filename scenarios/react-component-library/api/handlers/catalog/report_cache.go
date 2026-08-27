package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"react-component-library/internal/catalogcoverage"
)

// Why this cache exists.
//
// catalogcoverage.RecomputeEvidence executes every gate runner, and one of them
// (`types`) shells out to `pnpm run catalog:check` — a full TypeScript compile
// plus ESLint pass over the library corpus. Measured on 2026-08-15 that runner
// alone costs 13.5s of an ~18s recompute, and GetCoverage was paying it on
// every request. The coverage page issues two requests (GetCoverage and
// ListNextWork, each calling report()), so a single page load spawned the
// toolchain twice and took long enough that the browser gave up — the page
// rendered "Coverage unavailable" against a perfectly healthy API.
//
// The evidence itself is not cheap to obtain and is not stored anywhere else:
// the persisted EvidenceStore holds browser/experience gate results (unit,
// interaction, visual, …) but never `types`, so dropping the live recompute
// would silently lower every asset's achieved rung. The numbers therefore have
// to stay exactly as they were; only the recomputation frequency changes.
//
// Strategy: fingerprint the inputs cheaply, serve a matching cached report
// immediately, and serve a stale one while refreshing in the background. Only
// a cold cache computes synchronously.
type reportCache struct {
	mu          sync.Mutex
	fingerprint string
	report      *catalogcoverage.Report
	computedAt  time.Time
	refreshing  bool
}

// assetFingerprint hashes the catalog declaration, active manifest, and the
// complete latest-version source/lock tree for one asset. It is intentionally
// content based: a checkout or rebase that preserves bytes must preserve the
// cached verdict too.
func assetFingerprint(repoRoot, assetID string) (string, error) {
	root := filepath.Join(repoRoot, "scenarios", "react-component-library")
	var paths []string
	catalogPaths, _ := filepath.Glob(filepath.Join(root, "catalog", "assets", "*", "*.json"))
	for _, path := range catalogPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		var doc struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		if json.Unmarshal(data, &doc) == nil && doc.Asset.ID == assetID {
			paths = append(paths, path)
		}
	}
	var manifestPaths []string
	for _, libraryRoot := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, globErr := filepath.Glob(filepath.Join(root, "library", libraryRoot, "*", "component.json"))
		if globErr != nil {
			return "", globErr
		}
		manifestPaths = append(manifestPaths, paths...)
	}
	for _, manifestPath := range manifestPaths {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return "", err
		}
		var manifest struct {
			CatalogID string `json:"catalogId"`
			Latest    string `json:"latest"`
		}
		if json.Unmarshal(data, &manifest) != nil || manifest.CatalogID != assetID {
			continue
		}
		paths = append(paths, manifestPath)
		versionRoot := filepath.Join(filepath.Dir(manifestPath), "versions", manifest.Latest)
		_ = filepath.WalkDir(versionRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				paths = append(paths, path)
			}
			return nil
		})
	}
	sort.Strings(paths)
	return hashInputs(repoRoot, paths)
}

func hashInputs(repoRoot string, paths []string) (string, error) {
	hash := sha256.New()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte(filepath.ToSlash(rel)))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// dependentAssetIDs returns the changed asset and every catalog asset whose
// latest dependency lock names it. Dependents are derived from generated
// locks, not from import text or a hand-maintained allowlist.
func dependentAssetIDs(repoRoot, assetID string) ([]string, error) {
	root := filepath.Join(repoRoot, "scenarios", "react-component-library")
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(root, "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := catalogcoverage.LoadImplementations(filepath.Join(root, "library"))
	if err != nil {
		return nil, err
	}
	targetLibrary := ""
	idsByLibrary := map[string]string{}
	for _, impl := range impls {
		if impl.CatalogID != "" {
			idsByLibrary[impl.LibraryID] = impl.CatalogID
			if impl.CatalogID == assetID {
				targetLibrary = impl.LibraryID
			}
		}
	}
	if targetLibrary == "" {
		for _, asset := range assets {
			if asset.ID == assetID {
				for _, impl := range impls {
					if impl.Name == asset.Name {
						targetLibrary = impl.LibraryID
					}
				}
			}
		}
	}
	changed := map[string]bool{assetID: true}
	for _, impl := range impls {
		if impl.CatalogID == "" || impl.CatalogID == assetID {
			continue
		}
		for _, dep := range impl.Dependencies {
			if dep.LibraryID == targetLibrary {
				changed[impl.CatalogID] = true
			}
		}
	}
	result := make([]string, 0, len(changed))
	for id := range changed {
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

func (c *reportCache) invalidate() {
	c.mu.Lock()
	c.report = nil
	c.fingerprint = ""
	c.mu.Unlock()
}

// peek returns the most recently computed report without triggering a cold
// recomputation. Read-only summaries such as readiness must remain available
// while a fresh full gate report is still being built.
func (c *reportCache) peek() *catalogcoverage.Report {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.report
}

// fingerprintRoots are the trees whose contents can change a coverage verdict:
// the desired-state catalog, the implementations, and the app sources the
// `types` runner compiles.
var fingerprintRoots = []string{
	filepath.Join("scenarios", "react-component-library", "catalog"),
	filepath.Join("scenarios", "react-component-library", "library"),
	filepath.Join("scenarios", "react-component-library", "ui", "src"),
}

// fingerprint hashes the content and relative path of every verdict input.
// Modification times are intentionally excluded: rebases, copies, and fresh
// checkouts must not invalidate a report when the authored inputs are equal.
// Paths are sorted before hashing so filesystem traversal order cannot change
// the cache key.
func fingerprint(repoRoot string) (string, error) {
	type input struct {
		path string
		data []byte
	}
	var inputs []input
	for _, relative := range fingerprintRoots {
		root := filepath.Join(repoRoot, relative)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				// A missing tree is a legitimate state (a fresh checkout may not
				// have built ui/src yet); it simply contributes nothing.
				if os.IsNotExist(err) {
					return fs.SkipDir
				}
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == "dist" {
					return fs.SkipDir
				}
				return nil
			}
			info, statErr := entry.Info()
			if statErr != nil {
				return statErr
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			inputs = append(inputs, input{path: filepath.ToSlash(rel), data: data})
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("fingerprint %s: %w", relative, err)
		}
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].path < inputs[j].path })
	hash := sha256.New()
	for _, item := range inputs {
		_, _ = hash.Write([]byte(item.path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(item.data)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// get returns a report for the current inputs. A cache hit returns
// immediately. A stale entry is returned immediately and refreshed in the
// background, because a coverage number a few seconds old is far more useful
// than a page that times out. Only a cold cache blocks.
func (c *reportCache) get(ctx context.Context, repoRoot string, compute func(context.Context) (*catalogcoverage.Report, error)) (*catalogcoverage.Report, error) {
	current, err := fingerprint(repoRoot)
	if err != nil {
		// Fingerprinting failed, so freshness cannot be judged. Prefer a
		// possibly-stale report over an error page; fall through to a
		// synchronous compute only when nothing is cached.
		current = ""
	}

	c.mu.Lock()
	cached, cachedFingerprint := c.report, c.fingerprint
	if cached != nil && current != "" && cachedFingerprint == current {
		c.mu.Unlock()
		return cached, nil
	}
	if cached != nil {
		// Stale but usable. Kick a single background refresh; concurrent
		// requests share it rather than each spawning their own toolchain.
		if !c.refreshing {
			c.refreshing = true
			go c.refresh(ctx, repoRoot, compute)
		}
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()

	report, err := compute(ctx)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.report, c.fingerprint, c.computedAt = report, current, time.Now()
	c.mu.Unlock()
	return report, nil
}

func (c *reportCache) refresh(requestCtx context.Context, repoRoot string, compute func(context.Context) (*catalogcoverage.Report, error)) {
	defer func() {
		c.mu.Lock()
		c.refreshing = false
		c.mu.Unlock()
	}()
	// Detached from the request context on purpose: the refresh must survive
	// the response that triggered it, or a stale entry never converges.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), 5*time.Minute)
	defer cancel()
	report, err := compute(ctx)
	if err != nil {
		return
	}
	current, err := fingerprint(repoRoot)
	if err != nil {
		return
	}
	c.mu.Lock()
	c.report, c.fingerprint, c.computedAt = report, current, time.Now()
	c.mu.Unlock()
}
