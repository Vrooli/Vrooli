// Package filerouting provides runtime test-mode routing for scenario file
// storage. It mirrors database.RoutedDB: production roots remain the default,
// while requests carrying database.WithTestMode resolve to an installed,
// lease-owned throwaway root set.
package filerouting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
)

const DefaultLeaseTTL = database.DefaultLeaseTTL

type Clock = schedule.Clock

// LeaseStatsSnapshot reports the write-routing evidence for one lease.
// Callers must invoke RecordWrite after a successful filesystem mutation.
type LeaseStatsSnapshot struct {
	TestRootWrites              int64
	PrimaryWritesDuringTestMode int64
}

// TestRootsRestartState is a private owner snapshot for devrouting's durable
// graceful-restart checkpoint. Paths are internal owner data and must never be
// accepted directly from an RPC caller or written to public evidence.
type TestRootsRestartState struct {
	LeaseID            string
	ExpiresAt          time.Time
	LeaseRoot          string
	Roots              storage.Paths
	PrimaryFingerprint string
	Stats              LeaseStatsSnapshot
}

type leaseState struct {
	id        string
	expiresAt time.Time
}

type testRootsPinKey struct{}

type testRootsPin struct {
	owner      *RoutedRoots
	leaseID    string
	generation uint64
}

// RoutedRoots routes storage classes independently per request context.
// Primary roots are immutable after construction. The test roots are replaced
// atomically under the same lease semantics as RoutedDB.
type RoutedRoots struct {
	mu         sync.RWMutex
	primary    storage.Paths
	test       storage.Paths
	lease      leaseState
	leaseRoot  string
	generation uint64
	clock      Clock
	stats      struct {
		testWrites    atomic.Int64
		primaryWrites atomic.Int64
	}
}

func New(primary storage.Paths) *RoutedRoots {
	return &RoutedRoots{primary: primary, clock: schedule.System()}
}

func (r *RoutedRoots) SetClock(clock Clock) {
	if clock == nil {
		clock = schedule.System()
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
	r.generation++
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
	if leaseRoot != "" {
		if err := os.RemoveAll(leaseRoot); err != nil {
			return fmt.Errorf("remove leased test roots: %w", err)
		}
	}
	r.test = storage.Paths{}
	r.leaseRoot = ""
	r.generation++
	r.lease = leaseState{}
	return nil
}

// ClearRetainedTestRoots removes an owner-created root tree from a detached
// restart checkpoint. The caller must verify the checkpoint marker first.
// This owner independently proves the exact primary identity, generated temp
// root shape, and absence of any active lease before deleting the tree.
func (r *RoutedRoots) ClearRetainedTestRoots(state TestRootsRestartState, expectedPrimaryFingerprint string) error {
	if r == nil {
		return fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.id != "" || r.leaseRoot != "" {
		return fmt.Errorf("filerouting.RoutedRoots: an active test-root lease prevents retained cleanup")
	}
	if state.LeaseID == "" || state.LeaseRoot == "" || expectedPrimaryFingerprint == "" ||
		state.PrimaryFingerprint != expectedPrimaryFingerprint || pathsFingerprint(r.primary) != expectedPrimaryFingerprint {
		return fmt.Errorf("filerouting.RoutedRoots: retained root checkpoint identity mismatch")
	}
	root, err := filepath.Abs(state.LeaseRoot)
	if err != nil || filepath.Clean(state.LeaseRoot) != root || filepath.Base(root) == "." {
		return fmt.Errorf("filerouting.RoutedRoots: retained root path is invalid")
	}
	temp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return fmt.Errorf("filerouting.RoutedRoots: system temp root is unavailable")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(root))
	if err != nil || filepath.Clean(parent) != filepath.Clean(temp) || !strings.HasPrefix(filepath.Base(root), "vrooli-test-roots-") {
		return fmt.Errorf("filerouting.RoutedRoots: retained root is outside the owner temp namespace")
	}
	wantRoots := storage.Paths{
		ConfigDir: filepath.Join(root, string(storage.ClassConfig)), DataDir: filepath.Join(root, string(storage.ClassData)),
		CacheDir: filepath.Join(root, string(storage.ClassCache)), LogsDir: filepath.Join(root, string(storage.ClassLogs)),
		StateDir: filepath.Join(root, string(storage.ClassState)),
	}
	if state.Roots != wantRoots {
		return fmt.Errorf("filerouting.RoutedRoots: retained root classes differ from the owner layout")
	}
	for _, class := range []storage.Class{storage.ClassConfig, storage.ClassData, storage.ClassCache, storage.ClassLogs, storage.ClassState} {
		primaryRoot, _ := r.primary.ForClass(class)
		primaryAbs, absErr := filepath.Abs(primaryRoot)
		if absErr != nil || pathContains(root, primaryAbs) || pathContains(primaryAbs, root) {
			return fmt.Errorf("filerouting.RoutedRoots: retained root overlaps primary storage")
		}
	}
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("filerouting.RoutedRoots: retained root is not a real directory")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(root) {
		return fmt.Errorf("filerouting.RoutedRoots: retained root resolved outside its owner path")
	}
	return os.RemoveAll(root)
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}

// TestRootsRestartState returns the exact live owner-created tree binding.
func (r *RoutedRoots) TestRootsRestartState(leaseID string) (TestRootsRestartState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.lease.id != leaseID || r.leaseRoot == "" {
		return TestRootsRestartState{}, fmt.Errorf("filerouting.RoutedRoots: no live owner-created root lease")
	}
	return TestRootsRestartState{
		LeaseID: leaseID, ExpiresAt: r.lease.expiresAt, LeaseRoot: r.leaseRoot,
		Roots: r.test, PrimaryFingerprint: pathsFingerprint(r.primary), Stats: r.LeaseStats(),
	}, nil
}

// ResumeTestRoots reattaches an existing owner-created tree under a fresh
// generation. Its state must come from devrouting's private checkpoint, never
// from a request field. The directory tree is validated before attachment.
func (r *RoutedRoots) ResumeTestRoots(state TestRootsRestartState, expectedPrimaryFingerprint string) error {
	if r == nil {
		return fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	if state.LeaseID == "" || state.LeaseRoot == "" || !state.ExpiresAt.After(r.clock.Now()) {
		return fmt.Errorf("filerouting.RoutedRoots: restart binding is invalid or expired")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.id != "" && (r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt)) {
		return fmt.Errorf("filerouting.RoutedRoots: another test root binding is active")
	}
	if r.lease.id == "" && r.leaseRoot != "" {
		return fmt.Errorf("filerouting.RoutedRoots: another test root binding is active")
	}
	if pathsFingerprint(r.primary) != expectedPrimaryFingerprint || state.PrimaryFingerprint != expectedPrimaryFingerprint {
		return fmt.Errorf("filerouting.RoutedRoots: restart target identity mismatch")
	}
	rootInfo, err := os.Lstat(state.LeaseRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("filerouting.RoutedRoots: retained test root is unavailable")
	}
	for _, class := range []storage.Class{storage.ClassConfig, storage.ClassData, storage.ClassCache, storage.ClassLogs, storage.ClassState} {
		path, pathErr := state.Roots.ForClass(class)
		if pathErr != nil || filepath.Dir(path) != state.LeaseRoot {
			return fmt.Errorf("filerouting.RoutedRoots: retained test root layout is invalid")
		}
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("filerouting.RoutedRoots: retained test root is incomplete")
		}
	}
	if !state.ExpiresAt.After(r.clock.Now()) {
		return fmt.Errorf("filerouting.RoutedRoots: restart binding is invalid or expired")
	}
	if r.leaseRoot != "" && r.leaseRoot != state.LeaseRoot {
		if err := os.RemoveAll(r.leaseRoot); err != nil {
			return fmt.Errorf("filerouting.RoutedRoots: expired test root could not be removed")
		}
	}
	if !state.ExpiresAt.After(r.clock.Now()) {
		return fmt.Errorf("filerouting.RoutedRoots: restart binding is invalid or expired")
	}
	r.test = state.Roots
	r.leaseRoot = state.LeaseRoot
	r.lease = leaseState{id: state.LeaseID, expiresAt: state.ExpiresAt}
	r.generation++
	r.stats.testWrites.Store(0)
	r.stats.primaryWrites.Store(0)
	return nil
}

// PrimaryFingerprint identifies the immutable primary roots without exposing
// their paths to the public routing contract.
func (r *RoutedRoots) PrimaryFingerprint() string {
	if r == nil {
		return ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return pathsFingerprint(r.primary)
}

func pathsFingerprint(paths storage.Paths) string {
	values := []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LogsDir, paths.StateDir}
	canonical := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			canonical = append(canonical, "")
			continue
		}
		absolute, err := filepath.Abs(value)
		if err != nil {
			absolute = filepath.Clean(value)
		}
		canonical = append(canonical, filepath.Clean(absolute))
	}
	body, _ := json.Marshal(canonical)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
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
	if ctx != nil {
		if pin, ok := ctx.Value(testRootsPinKey{}).(testRootsPin); ok {
			return r.pickPinned(pin, class)
		}
	}
	r.mu.RLock()
	primary, test, lease, clock := r.primary, r.test, r.lease, r.clock
	r.mu.RUnlock()
	if database.IsTestMode(ctx) && lease.id != "" && (lease.expiresAt.IsZero() || !clock.Now().After(lease.expiresAt)) {
		return test.ForClass(class)
	}
	return primary.ForClass(class)
}

// PickRequired selects a root atomically and refuses a production fallback for
// a test-mode caller. Writers that require isolation must use this instead of
// a separate HasTestRoots/Pick pair, which can race lease expiry or removal.
func (r *RoutedRoots) PickRequired(ctx context.Context, class storage.Class) (string, error) {
	if r == nil {
		return "", fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	if ctx != nil {
		if pin, ok := ctx.Value(testRootsPinKey{}).(testRootsPin); ok {
			return r.pickPinned(pin, class)
		}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if database.IsTestMode(ctx) {
		if r.lease.id == "" || (!r.lease.expiresAt.IsZero() && r.clock.Now().After(r.lease.expiresAt)) {
			return "", fmt.Errorf("test-mode file access requires an active leased root")
		}
		return r.test.ForClass(class)
	}
	return r.primary.ForClass(class)
}

func (r *RoutedRoots) pickPinned(pin testRootsPin, class storage.Class) (string, error) {
	if pin.owner != r {
		return "", fmt.Errorf("test file route pin belongs to another RoutedRoots owner")
	}
	r.mu.RLock()
	valid := r.lease.id == pin.leaseID && r.generation == pin.generation && (r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt))
	roots := r.test
	r.mu.RUnlock()
	if !valid {
		return "", fmt.Errorf("test file route lease %q is no longer active", pin.leaseID)
	}
	return roots.ForClass(class)
}

// PinTestRoots binds later file-root selections made with ctx to the exact
// active test-root installation. Reinstalling the same lease ID creates a new
// generation, so a stale context fails closed instead of selecting it.
func (r *RoutedRoots) PinTestRoots(ctx context.Context, expectedLeaseID string) (context.Context, error) {
	if r == nil {
		return nil, fmt.Errorf("filerouting.RoutedRoots is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if expectedLeaseID == "" {
		return nil, fmt.Errorf("test file route lease ID is required")
	}
	if ctx != nil {
		if existing, ok := ctx.Value(testRootsPinKey{}).(testRootsPin); ok {
			if existing.owner != r || existing.leaseID != expectedLeaseID {
				return nil, fmt.Errorf("test file route lease pin cannot be rebound")
			}
			r.mu.RLock()
			valid := r.lease.id == existing.leaseID && r.generation == existing.generation && (r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt))
			r.mu.RUnlock()
			if !valid {
				return nil, fmt.Errorf("test file route lease pin is no longer active")
			}
			return ctx, nil
		}
	}
	r.mu.RLock()
	valid := r.lease.id == expectedLeaseID && (r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt))
	pin := testRootsPin{owner: r, leaseID: r.lease.id, generation: r.generation}
	r.mu.RUnlock()
	if !valid {
		return nil, fmt.Errorf("test file route lease mismatch: expected %q", expectedLeaseID)
	}
	ctx = database.WithTestMode(ctx)
	return context.WithValue(ctx, testRootsPinKey{}, pin), nil
}

// RecordWrite records the destination selected by Pick after a successful
// write. Keeping it explicit avoids counting failed operations as leaks.
func (r *RoutedRoots) RecordWrite(ctx context.Context) {
	if r == nil || !database.IsTestMode(ctx) {
		return
	}
	if ctx != nil {
		if pin, ok := ctx.Value(testRootsPinKey{}).(testRootsPin); ok {
			if pin.owner != r {
				return
			}
			r.mu.RLock()
			valid := r.lease.id == pin.leaseID && r.generation == pin.generation && (r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt))
			r.mu.RUnlock()
			if valid {
				r.stats.testWrites.Add(1)
			}
			return
		}
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

// ActiveLeaseID reports the owner-bound lease even after expiry, for exact
// cleanup decisions. Routing still refuses expired roots.
func (r *RoutedRoots) ActiveLeaseID() string {
	if r == nil {
		return ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lease.id
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
