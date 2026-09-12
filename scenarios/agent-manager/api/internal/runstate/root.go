package runstate

import (
	"context"
	"fmt"
	"path/filepath"

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
