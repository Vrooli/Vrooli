package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"react-component-library/internal/librarywalk"
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
	if fingerprints, ok := assetCacheFingerprints(repoRoot); ok {
		hash := sha256.New()
		ids := make([]string, 0, len(fingerprints))
		for assetID := range fingerprints {
			ids = append(ids, assetID)
		}
		sort.Strings(ids)
		for _, assetID := range ids {
			_, _ = hash.Write([]byte(assetID))
			_, _ = hash.Write([]byte{0})
			_, _ = hash.Write([]byte(fingerprints[assetID]))
			_, _ = hash.Write([]byte{0})
		}
		return hex.EncodeToString(hash.Sum(nil)), nil
	}
	type input struct {
		path string
		data []byte
	}
	var inputs []input
	for _, relative := range fingerprintRoots {
		root := filepath.Join(repoRoot, relative)
		err := librarywalk.WalkContext(context.Background(), root, func(path string, entry fs.DirEntry, err error) error {
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

// assetCacheFingerprints is the freshness authority for real catalog
// evidence. Each asset gets its own revision/rule-set key, and dependency
// changes are folded into BuildRevisionIndex. The report-level cache hashes
// these keys only to decide whether a refresh is needed; evidence rows remain
// independently reusable by asset and gate.
func assetCacheFingerprints(repoRoot string) (map[string]string, bool) {
	revisions, err := catalogcoverage.BuildRevisionIndex(repoRoot)
	if err != nil {
		return nil, false
	}
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog"))
	if err != nil || len(assets) == 0 {
		return nil, false
	}
	fingerprints := make(map[string]string, len(assets))
	for _, asset := range assets {
		revision, ok := revisions[asset.ID]
		if !ok {
			continue
		}
		digest, digestErr := catalogcoverage.RuleSetDigest(repoRoot, asset.ID)
		if digestErr != nil {
			return nil, false
		}
		fingerprints[asset.ID] = revision + "\x00" + digest
	}
	return fingerprints, true
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
