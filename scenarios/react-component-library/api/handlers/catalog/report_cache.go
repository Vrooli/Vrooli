package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"react-component-library/internal/librarywalk"

	"react-component-library/internal/catalogcoverage"

	repocontract "github.com/vrooli/repo-contract-go"
)

// ErrReportWarming means the first full gate pass is still running. It is a
// state, not a failure: the coverage page must say "preparing" instead of
// "unavailable", and it must never render zero as if the library were empty.
var ErrReportWarming = errors.New("coverage report is being prepared")

// Why this cache exists.
//
// catalogcoverage.RecomputeEvidence executes every gate runner. A cold pass
// walks the gate corpus, shells out to the TypeScript toolchain, and fetches
// Experience Manager evidence; measured on this machine it takes minutes. The
// coverage page issues two requests (GetCoverage and ListNextWork, each calling
// report()), so a cold request used to hold both open until the browser gave
// up, and every restart paid the cost again.
//
// Strategy: fingerprint the inputs cheaply, persist the last report beside the
// runtime data, serve a matching cached report immediately, serve a stale one
// while refreshing in the background, and coalesce concurrent cold callers onto
// one compute. A caller that arrives during a cold pass gets ErrReportWarming
// instead of a minutes-long request.
type reportCache struct {
	mu          sync.Mutex
	fingerprint string
	report      *catalogcoverage.Report
	computedAt  time.Time
	refreshing  bool
	ready       chan struct{}
	diskPath    string
	diskLoaded  bool
	fpAt        time.Time
	fpValue     string
}

// fingerprintTTL keeps a burst of page requests from re-hashing the library.
// Thirty seconds of staleness can delay noticing an edit, never mislabel it.
const fingerprintTTL = 30 * time.Second

func (c *reportCache) cachedFingerprint(repoRoot string, force bool) (string, error) {
	c.mu.Lock()
	if !force && !c.fpAt.IsZero() && time.Since(c.fpAt) < fingerprintTTL {
		value := c.fpValue
		c.mu.Unlock()
		return value, nil
	}
	c.mu.Unlock()
	value, err := fingerprint(repoRoot)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.fpAt, c.fpValue = time.Now(), value
	c.mu.Unlock()
	return value, nil
}

// reportWarmGrace is the short wait before a cold request is told the report
// is still being prepared. Tests shorten it.
var reportWarmGrace = 2 * time.Second

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

type persistedReport struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Fingerprint   string                  `json:"fingerprint"`
	ComputedAt    time.Time               `json:"computedAt"`
	Report        *catalogcoverage.Report `json:"report"`
}

func loadPersistedReport(path string) *persistedReport {
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var stored persistedReport
	if err := json.Unmarshal(raw, &stored); err != nil || stored.Report == nil {
		return nil
	}
	if stored.SchemaVersion != 1 || stored.Fingerprint == "" {
		return nil
	}
	return &stored
}

func savePersistedReport(path, fingerprint string, report *catalogcoverage.Report) {
	if path == "" || report == nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	payload, err := json.Marshal(persistedReport{
		SchemaVersion: 1,
		Fingerprint:   fingerprint,
		ComputedAt:    time.Now().UTC(),
		Report:        report,
	})
	if err != nil {
		return
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, payload, 0o644); err != nil {
		return
	}
	_ = os.Rename(temp, path)
}

// coverageCachePath is runtime data, never source. A missing HOME simply
// disables persistence.
func coverageCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return ""
	}
	return filepath.Join(root, "data", "vrooli", "react-component-library", "coverage-report-v1.json")
}

func (c *reportCache) get(ctx context.Context, repoRoot string, compute func(context.Context) (*catalogcoverage.Report, error)) (*catalogcoverage.Report, error) {
	current, err := c.cachedFingerprint(repoRoot, false)
	if err != nil {
		// Fingerprinting failed, so freshness cannot be judged. Prefer a
		// possibly-stale report over an error page; fall through to a
		// synchronous compute only when nothing is cached.
		current = ""
	}

	c.mu.Lock()
	if !c.diskLoaded {
		c.diskLoaded = true
		if c.report == nil {
			if stored := loadPersistedReport(c.diskPath); stored != nil {
				c.report = stored.Report
				c.fingerprint = stored.Fingerprint
				c.computedAt = stored.ComputedAt
			}
		}
	}
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
			c.ready = make(chan struct{})
			go c.refresh(ctx, repoRoot, compute, c.ready)
		}
		c.mu.Unlock()
		return cached, nil
	}
	if !c.refreshing {
		c.refreshing = true
		c.ready = make(chan struct{})
		go c.refresh(ctx, repoRoot, compute, c.ready)
	}
	ready := c.ready
	c.mu.Unlock()

	// Cold start: wait briefly so a fast fixture still answers synchronously,
	// then report "warming" rather than holding the request open for minutes.
	select {
	case <-ready:
	case <-time.After(reportWarmGrace):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	c.mu.Lock()
	report := c.report
	c.mu.Unlock()
	if report != nil {
		return report, nil
	}
	return nil, ErrReportWarming
}

func (c *reportCache) refresh(requestCtx context.Context, repoRoot string, compute func(context.Context) (*catalogcoverage.Report, error), ready chan struct{}) {
	defer func() {
		c.mu.Lock()
		c.refreshing = false
		if c.ready == ready {
			close(ready)
		}
		c.mu.Unlock()
	}()
	// Detached from the request context on purpose: the refresh must survive
	// the response that triggered it, or a stale entry never converges. The
	// full corpus can take several minutes on a cold machine.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), 15*time.Minute)
	defer cancel()
	report, err := compute(ctx)
	if err != nil {
		return
	}
	current, err := c.cachedFingerprint(repoRoot, true)
	if err != nil {
		return
	}
	c.mu.Lock()
	c.report, c.fingerprint, c.computedAt = report, current, time.Now()
	c.mu.Unlock()
	savePersistedReport(c.diskPath, current, report)
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
	digests, err := catalogcoverage.RuleSetDigests(repoRoot)
	if err != nil {
		return nil, false
	}
	fingerprints := make(map[string]string, len(assets))
	for _, asset := range assets {
		revision, ok := revisions[asset.ID]
		if !ok {
			continue
		}
		digest, ok := digests[asset.ID]
		if !ok {
			continue
		}
		fingerprints[asset.ID] = revision + "\x00" + digest
	}
	return fingerprints, true
}
