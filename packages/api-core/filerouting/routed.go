// Package filerouting provides runtime test-mode routing for scenario file
// storage. It mirrors database.RoutedDB: production roots remain the default,
// while requests carrying database.WithTestMode resolve to an installed,
// lease-owned throwaway root set.
package filerouting

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

const DefaultLeaseTTL = database.DefaultLeaseTTL

type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// LeaseStatsSnapshot reports the write-routing evidence for one lease.
// Callers must invoke RecordWrite after a successful filesystem mutation.
type LeaseStatsSnapshot struct {
	TestRootWrites              int64
	PrimaryWritesDuringTestMode int64
}

type leaseState struct {
	id        string
	expiresAt time.Time
}

// RoutedRoots routes storage classes independently per request context.
// Primary roots are immutable after construction. The test roots are replaced
// atomically under the same lease semantics as RoutedDB.
type RoutedRoots struct {
	mu        sync.RWMutex
	primary   storage.Paths
	test      storage.Paths
	lease     leaseState
	leaseRoot string
	clock     Clock
	stats     struct {
		testWrites    atomic.Int64
		primaryWrites atomic.Int64
	}
}

func New(primary storage.Paths) *RoutedRoots {
	return &RoutedRoots{primary: primary, clock: systemClock{}}
}

func (r *RoutedRoots) SetClock(clock Clock) {
	if clock == nil {
		clock = systemClock{}
	}
	r.mu.Lock()
	r.clock = clock
	r.mu.Unlock()
}

// InstallTestRoots installs test roots for leaseID. Reinstalling the same
// lease is idempotent; another active lease is rejected.
func (r *RoutedRoots) InstallTestRoots(roots storage.Paths, leaseID string, ttl time.Duration) error {
	return r.install(roots, leaseID, ttl, "")
}

func (r *RoutedRoots) install(roots storage.Paths, leaseID string, ttl time.Duration, leaseRoot string) error {
	if r == nil {
		return fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	if leaseID == "" {
		return fmt.Errorf("filerouting.RoutedRoots.InstallTestRoots: lease id is empty")
	}
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.id != "" && r.lease.id != leaseID && (r.lease.expiresAt.IsZero() || r.clock.Now().Before(r.lease.expiresAt)) {
		return fmt.Errorf("filerouting.RoutedRoots: test roots already installed under lease %q", r.lease.id)
	}
	if r.leaseRoot != "" && r.leaseRoot != leaseRoot {
		_ = os.RemoveAll(r.leaseRoot)
	}
	r.test = roots
	r.leaseRoot = leaseRoot
	r.lease = leaseState{id: leaseID, expiresAt: r.clock.Now().Add(ttl)}
	r.stats.testWrites.Store(0)
	r.stats.primaryWrites.Store(0)
	return nil
}

// InstallLeasedTestRoots creates and installs disposable class roots. Config
// is copied from primary unless emptyConfig is requested; all mutable classes
// start empty. ClearTestRoots removes the resulting temporary tree.
func (r *RoutedRoots) InstallLeasedTestRoots(leaseID string, ttl time.Duration, emptyConfig bool) (storage.Paths, error) {
	if r == nil {
		return storage.Paths{}, fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	r.mu.RLock()
	primary := r.primary
	r.mu.RUnlock()
	base, err := os.MkdirTemp("", "vrooli-test-roots-")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create leased test root: %w", err)
	}
	cleanup := func(cause error) (storage.Paths, error) {
		_ = os.RemoveAll(base)
		return storage.Paths{}, cause
	}
	roots := storage.Paths{
		ConfigDir: filepath.Join(base, string(storage.ClassConfig)),
		DataDir:   filepath.Join(base, string(storage.ClassData)),
		CacheDir:  filepath.Join(base, string(storage.ClassCache)),
		LogsDir:   filepath.Join(base, string(storage.ClassLogs)),
		StateDir:  filepath.Join(base, string(storage.ClassState)),
	}
	for _, class := range []storage.Class{storage.ClassConfig, storage.ClassData, storage.ClassCache, storage.ClassLogs, storage.ClassState} {
		root, _ := roots.ForClass(class)
		if err := os.MkdirAll(root, 0o755); err != nil {
			return cleanup(fmt.Errorf("create test %s root: %w", class, err))
		}
	}
	if !emptyConfig {
		if err := copyTree(primary.ConfigDir, roots.ConfigDir); err != nil {
			return cleanup(fmt.Errorf("seed test config root: %w", err))
		}
	}
	if err := r.install(roots, leaseID, ttl, base); err != nil {
		return cleanup(err)
	}
	return roots, nil
}

func (r *RoutedRoots) ClearTestRoots(leaseID string) error {
	if r == nil {
		return fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.id == "" {
		return nil
	}
	if r.lease.id != leaseID {
		return fmt.Errorf("filerouting.RoutedRoots: lease mismatch (active lease %q)", r.lease.id)
	}
	leaseRoot := r.leaseRoot
	r.test = storage.Paths{}
	r.leaseRoot = ""
	r.lease = leaseState{}
	if leaseRoot != "" {
		if err := os.RemoveAll(leaseRoot); err != nil {
			return fmt.Errorf("remove leased test roots: %w", err)
		}
	}
	return nil
}

// HeartbeatTestRoots extends the active file-root lease.
func (r *RoutedRoots) HeartbeatTestRoots(leaseID string, ttl time.Duration) (time.Time, error) {
	if r == nil {
		return time.Time{}, fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.id == "" || r.lease.id != leaseID {
		return time.Time{}, fmt.Errorf("filerouting.RoutedRoots: lease mismatch (active lease %q)", r.lease.id)
	}
	r.lease.expiresAt = r.clock.Now().Add(ttl)
	return r.lease.expiresAt, nil
}

// Pick resolves a class for this request. An expired lease deliberately falls
// back to primary; RecordWrite will account for a test-mode write leak.
func (r *RoutedRoots) Pick(ctx context.Context, class storage.Class) (string, error) {
	if r == nil {
		return "", fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	r.mu.RLock()
	primary, test, lease, clock := r.primary, r.test, r.lease, r.clock
	r.mu.RUnlock()
	if database.IsTestMode(ctx) && lease.id != "" && (lease.expiresAt.IsZero() || !clock.Now().After(lease.expiresAt)) {
		return test.ForClass(class)
	}
	return primary.ForClass(class)
}

// RecordWrite records the destination selected by Pick after a successful
// write. Keeping it explicit avoids counting failed operations as leaks.
func (r *RoutedRoots) RecordWrite(ctx context.Context) {
	if r == nil || !database.IsTestMode(ctx) {
		return
	}
	r.mu.RLock()
	lease, clock := r.lease, r.clock
	r.mu.RUnlock()
	if lease.id != "" && (lease.expiresAt.IsZero() || !clock.Now().After(lease.expiresAt)) {
		r.stats.testWrites.Add(1)
		return
	}
	r.stats.primaryWrites.Add(1)
}

func (r *RoutedRoots) LeaseStats() LeaseStatsSnapshot {
	if r == nil {
		return LeaseStatsSnapshot{}
	}
	return LeaseStatsSnapshot{TestRootWrites: r.stats.testWrites.Load(), PrimaryWritesDuringTestMode: r.stats.primaryWrites.Load()}
}

// HasTestRoots reports whether a non-expired leased root set is installed.
// Callers that need to authorize a test-only product seam should require this
// alongside database.RoutedDB.HasTestPool so the complete isolation contract
// is present before accepting the request.
func (r *RoutedRoots) HasTestRoots() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.lease.id == "" {
		return false
	}
	return r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt)
}

func copyTree(source, destination string) error {
	info, err := os.Stat(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source %q is not a directory", source)
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to seed symlink %q", path)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
