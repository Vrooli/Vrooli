package catalog

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

// fingerprintRoots are the trees whose contents can change a coverage verdict:
// the desired-state catalog, the implementations, and the app sources the
// `types` runner compiles.
var fingerprintRoots = []string{
	filepath.Join("scenarios", "react-component-library", "catalog"),
	filepath.Join("scenarios", "react-component-library", "library"),
	filepath.Join("scenarios", "react-component-library", "ui", "src"),
}

// fingerprint summarises the inputs by file count and newest modification time.
// It is deliberately not a content hash: hashing every file in these trees
// would reintroduce the per-request cost this cache exists to remove. The
// tradeoff is that a change which preserves both mtime and file count is not
// observed — in practice that means a deliberately backdated write, which no
// normal edit or checkout produces.
func fingerprint(repoRoot string) (string, error) {
	var (
		files  int
		newest time.Time
	)
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
				return nil
			}
			files++
			if info.ModTime().After(newest) {
				newest = info.ModTime()
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("fingerprint %s: %w", relative, err)
		}
	}
	return fmt.Sprintf("%d:%d", files, newest.UnixNano()), nil
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
			go c.refresh(repoRoot, compute)
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

func (c *reportCache) refresh(repoRoot string, compute func(context.Context) (*catalogcoverage.Report, error)) {
	defer func() {
		c.mu.Lock()
		c.refreshing = false
		c.mu.Unlock()
	}()
	// Detached from the request context on purpose: the refresh must survive
	// the response that triggered it, or a stale entry never converges.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
