package capacity

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	platform "github.com/vrooli/platform-go"
)

// ErrAssetInUse is a retryable refusal: an artifact is being consumed or
// changed in this or another process. It is not a capacity-sensing failure.
var ErrAssetInUse = errors.New("capacity: asset is in use or being changed")

// UseGate coordinates artifact readers with installation/removal writers.
// Root is the shared artifact collection's storage directory, never an
// individual model directory that a writer can replace. All consumers of a
// collection must use the same Root and artifact IDs. Native locks, rather
// than process-local counters or expiring readers, prevent a slow inference
// from losing protection and release ownership automatically on process exit.
type UseGate struct {
	Root string
}

// Acquire retains an artifact for reading. Readers across processes coexist.
func (g *UseGate) Acquire(id string) (func(), error) { return g.acquire(id, false) }

// AcquireExclusive refuses mutation while any process is reading or changing
// the artifact. It never blocks an install request behind long inference.
func (g *UseGate) AcquireExclusive(id string) (func(), error) { return g.acquire(id, true) }

func (g *UseGate) acquire(id string, exclusive bool) (func(), error) {
	if g == nil || strings.TrimSpace(g.Root) == "" || strings.TrimSpace(id) == "" {
		return nil, errors.New("capacity: artifact collection root and ID are required")
	}
	root, err := filepath.Abs(g.Root)
	if err != nil {
		return nil, fmt.Errorf("capacity: resolve artifact collection: %w", err)
	}
	// Locks are siblings of model directories, so removing/replacing an
	// exclusively held artifact never removes its coordinating inode.
	dir := filepath.Join(root, ".capacity-use-leases")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("capacity: create artifact lease directory: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("%x.lock", sha256.Sum256([]byte(strings.TrimSpace(id)))))
	file, err := platform.OpenLockFile(path, 0o600)
	if err != nil {
		return nil, fmt.Errorf("capacity: open artifact lease: %w", err)
	}
	lock := platform.ReadLockFile
	if exclusive {
		lock = platform.LockFile
	}
	release, err := lock(file, true)
	if err != nil {
		_ = file.Close()
		if errors.Is(err, platform.ErrLockUnavailable) {
			return nil, fmt.Errorf("%w: %s", ErrAssetInUse, id)
		}
		return nil, fmt.Errorf("capacity: acquire artifact lease: %w", err)
	}
	if err := platform.VerifyLockFile(file, path); err != nil {
		release()
		_ = file.Close()
		return nil, err
	}
	var once sync.Once
	return func() { once.Do(func() { release(); _ = file.Close() }) }, nil
}
