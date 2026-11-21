// Package executionartifactlock coordinates artifact mutations across BAS API
// instances that share a recordings root.
package executionartifactlock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	platform "github.com/vrooli/platform-go"
)

type lockContextKey struct{}

type lockContextValue struct {
	root string
	id   uuid.UUID
}

// Acquire holds one of 256 cross-process exclusive execution-artifact locks.
// Sharding bounds persistent lock files while unrelated executions proceed in
// parallel except for an occasional hash collision. The inode lives outside
// UUID artifact directories so deletion cannot split it.
func Acquire(ctx context.Context, recordingsRoot string, executionID uuid.UUID) (func(), error) {
	if executionID == uuid.Nil || strings.TrimSpace(recordingsRoot) == "" {
		return nil, fmt.Errorf("execution artifact lock requires a UUID and recordings root")
	}
	root, err := filepath.Abs(filepath.Clean(recordingsRoot))
	if err != nil {
		return nil, fmt.Errorf("resolve recordings root: %w", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(root); resolveErr == nil {
		root = resolved
	}
	root = filepath.Clean(root)
	lockDir := filepath.Join(root, ".execution-locks")
	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		// Invalid-root fixtures still exercise the writer's own failure path
		// without replacing that error with a lock-directory error.
		rootDigest := sha256.Sum256([]byte(root))
		lockDir = filepath.Join(os.TempDir(), "vrooli-browser-automation-studio", hex.EncodeToString(rootDigest[:]))
	}
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return nil, fmt.Errorf("create execution lock directory: %w", err)
	}
	executionDigest := sha256.Sum256([]byte(executionID.String()))
	path := filepath.Join(lockDir, hex.EncodeToString(executionDigest[:1])+".lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open execution lock: %w", err)
	}
	releaseLock, err := platform.LockFileContext(ctx, file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("acquire execution lock: %w", err)
	}
	return func() {
		releaseLock()
		_ = file.Close()
	}, nil
}

// WithLock runs work under the shared execution lock and marks the context so
// repository adapters called by the writer can safely reuse the held lock.
func WithLock(ctx context.Context, recordingsRoot string, executionID uuid.UUID, work func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	root := canonicalRootKey(recordingsRoot)
	if held, ok := ctx.Value(lockContextKey{}).(lockContextValue); ok && held.root == root && held.id == executionID {
		return work(ctx)
	}
	release, err := Acquire(ctx, recordingsRoot, executionID)
	if err != nil {
		return err
	}
	defer release()
	return work(context.WithValue(ctx, lockContextKey{}, lockContextValue{root: root, id: executionID}))
}

func canonicalRootKey(recordingsRoot string) string {
	root, err := filepath.Abs(filepath.Clean(recordingsRoot))
	if err != nil {
		return filepath.Clean(recordingsRoot)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(root); resolveErr == nil {
		root = resolved
	}
	return filepath.Clean(root)
}
