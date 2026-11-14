package repohealth

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/engine"
)

// RepoLocator maps a kopia repository name to its destination id and on-disk
// repository directory. ok is false for anything that is not a filesystem
// destination, which has no directory to measure.
type RepoLocator func(ctx context.Context, repoName string) (destinationID, root string, ok bool)

// UsageEngine wraps the kopia engine so that every usage measurement (backup
// preflight, destination usage, storage-cap checks, run size accounting) goes
// through one rule: the engine's own answer wins; when the engine fails, usage
// is measured directly from the repository directory ONLY IF every unreadable
// entry in it is covered by a recorded exception and the engine's failure names
// one of those exact paths. Any other unreadable entry, or a failure that is not
// attributable to an excepted path, keeps the original error.
type UsageEngine struct {
	engine.KopiaEngine
	Locate     RepoLocator
	Exceptions ExceptionStore
	Scanner    *Scanner
	// CacheTTL bounds how long one scan is reused. A scan walks every blob in the
	// repository, so back-to-back calls (preflight, then the run's size baseline)
	// share it.
	CacheTTL time.Duration
	Now      func() time.Time

	mu    sync.Mutex
	cache map[string]cachedScan
}

type cachedScan struct {
	scan Scan
	at   time.Time
}

var _ engine.KopiaEngine = (*UsageEngine)(nil)

func (u *UsageEngine) now() time.Time {
	if u.Now != nil {
		return u.Now()
	}
	return time.Now()
}

func (u *UsageEngine) ttl() time.Duration {
	if u.CacheTTL > 0 {
		return u.CacheTTL
	}
	return 5 * time.Minute
}

// CachedScan returns a scan of root, reusing a recent one.
func (u *UsageEngine) CachedScan(ctx context.Context, root string) (Scan, error) {
	u.mu.Lock()
	if hit, ok := u.cache[root]; ok && u.now().Sub(hit.at) < u.ttl() {
		u.mu.Unlock()
		return hit.scan, nil
	}
	u.mu.Unlock()
	scan, err := u.Scanner.Walk(ctx, root)
	if err != nil {
		return Scan{}, err
	}
	u.mu.Lock()
	if u.cache == nil {
		u.cache = map[string]cachedScan{}
	}
	u.cache[root] = cachedScan{scan: scan, at: u.now()}
	u.mu.Unlock()
	return scan, nil
}

// InvalidateScan drops a cached scan, for use after the repository directory
// changed under an approved action.
func (u *UsageEngine) InvalidateScan(root string) {
	u.mu.Lock()
	delete(u.cache, root)
	u.mu.Unlock()
}

func (u *UsageEngine) RepoStats(ctx context.Context, repo string) (engine.RepoStats, error) {
	stats, engineErr := u.KopiaEngine.RepoStats(ctx, repo)
	if engineErr == nil {
		return stats, nil
	}
	if u.Locate == nil || u.Exceptions == nil {
		return engine.RepoStats{}, engineErr
	}
	destID, root, ok := u.Locate(ctx, repo)
	if !ok {
		return engine.RepoStats{}, engineErr
	}
	exceptions, err := u.Exceptions.List(ctx, destID)
	if err != nil || len(exceptions) == 0 {
		return engine.RepoStats{}, engineErr
	}
	scan, err := u.CachedScan(ctx, root)
	if err != nil {
		return engine.RepoStats{}, engineErr
	}
	verdict := Evaluate(u.Scanner.fsys(), root, scan.Unreadable, exceptions)
	if len(verdict.Blocking) > 0 {
		return engine.RepoStats{}, fmt.Errorf("%w; %d unreadable repository entr%s not covered by an approved exception (first: %s)",
			engineErr, len(verdict.Blocking), plural(len(verdict.Blocking), "y is", "ies are"), verdict.Blocking[0].RelPath)
	}
	if !failureNamesExcusedEntry(engineErr, verdict.Excused) {
		return engine.RepoStats{}, engineErr
	}
	return engine.RepoStats{SizeBytes: scan.Bytes, BlobCount: scan.Files}, nil
}

// failureNamesExcusedEntry reports whether the engine's error text names one of
// the exact excepted paths, so an exception is never used to wave through a
// failure it was not granted for.
func failureNamesExcusedEntry(err error, excused []Entry) bool {
	text := err.Error()
	for _, entry := range excused {
		if strings.Contains(text, path.Base(entry.RelPath)) {
			return true
		}
	}
	return false
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
