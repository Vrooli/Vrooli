package runstate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// RootResolver resolves the durable run-state root for an operation. It keeps
// storage routing at the operation boundary: a resolver must not capture a
// process-global path before the request context is known.
type RootResolver interface {
	Resolve(context.Context) (string, error)
	RecordWrite(context.Context)
}

// StaticRoot is suitable for explicitly injected test roots.
type StaticRoot string

func (r StaticRoot) Resolve(context.Context) (string, error) {
	if r == "" {
		return "", fmt.Errorf("run state root is required")
	}
	return string(r), nil
}

func (StaticRoot) RecordWrite(context.Context) {}

// Validate checks the storage contract before an owner can admit new work.
// A successful Resolve is not enough: a missing or process-ephemeral root can
// otherwise look healthy until the first restart loses the execution record.
func (r StaticRoot) Validate(context.Context) error { return ValidateRoot(string(r)) }

// RoutedRoot resolves the state class for every operation. Test-mode contexts
// therefore select their leased state root rather than the startup path.
type RoutedRoot struct{ Roots *filerouting.RoutedRoots }

func (r RoutedRoot) Resolve(ctx context.Context) (string, error) {
	if r.Roots == nil {
		return "", fmt.Errorf("run state root resolver is not configured")
	}
	root, err := r.Roots.Pick(ctx, storage.ClassState)
	if err != nil {
		return "", fmt.Errorf("resolve run state root: %w", err)
	}
	return filepath.Join(root, "runs"), nil
}

func (r RoutedRoot) RecordWrite(ctx context.Context) {
	if r.Roots != nil {
		r.Roots.RecordWrite(ctx)
	}
}

// Validate resolves the production state root and applies the same durable
// storage gate used by static test/injected roots.
func (r RoutedRoot) Validate(ctx context.Context) error {
	root, err := r.Resolve(ctx)
	if err != nil {
		return err
	}
	return ValidateRoot(root)
}

// ValidateRoot verifies that run state is on an absolute, writable directory
// that is not a well-known process-temporary filesystem. The probe is synced
// before removal so startup exercises the same write path used for metadata.
// Host power-loss durability still depends on the operator's storage system;
// this gate prevents the more common accidental /tmp or unavailable-root
// configuration.
func ValidateRoot(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("run state root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve run state root: %w", err)
	}
	for _, prefix := range []string{"/tmp", "/var/tmp", "/run", "/dev/shm"} {
		if abs == prefix || strings.HasPrefix(abs, prefix+string(os.PathSeparator)) {
			return fmt.Errorf("run state root %q is process-ephemeral; configure durable storage outside %s", abs, prefix)
		}
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return fmt.Errorf("create run state root %q: %w", abs, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("stat run state root %q: %w", abs, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("run state root %q is not a directory", abs)
	}
	probe, err := os.CreateTemp(abs, ".runstate-write-probe-")
	if err != nil {
		return fmt.Errorf("run state root %q is not writable: %w", abs, err)
	}
	probeName := probe.Name()
	defer os.Remove(probeName)
	if err := probe.Sync(); err != nil {
		_ = probe.Close()
		return fmt.Errorf("sync run state root %q: %w", abs, err)
	}
	if err := probe.Close(); err != nil {
		return fmt.Errorf("close run state probe: %w", err)
	}
	dir, err := os.Open(abs)
	if err != nil {
		return fmt.Errorf("open run state root %q: %w", abs, err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync run state root directory %q: %w", abs, err)
	}
	return nil
}
