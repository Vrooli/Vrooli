// Package mocks holds test doubles for the sources domain seams. Lives in a
// mocks/ directory (no _test.go suffix) so sibling _test.go files in the runs
// and restores packages can import it; never linked into production.
package mocks

import (
	"context"
	"sync"

	"data-backup-manager/internal/sources"
)

// FakeCapturer satisfies sources.Capturer for tests. SourceKind selects which
// kind it claims to handle; CaptureFn / RestoreFn let a test program exact
// behavior (deterministic bytes, an injected failure). When a func field is
// nil the fake records the call and returns a trivial artifact / nil error.
//
// Capture/Restore are concurrency-safe: a run fans out target×destination
// units in parallel, so the recording tolerates concurrent calls the same way
// the production (stateless) capturers do.
type FakeCapturer struct {
	SourceKind sources.SourceKind

	CaptureFn func(ctx context.Context, spec sources.CaptureSpec) (sources.Artifact, error)
	RestoreFn func(ctx context.Context, spec sources.RestoreSpec) error

	mu       sync.Mutex
	Captures []sources.CaptureSpec
	Restores []sources.RestoreSpec
}

// Compile-time guarantee.
var _ sources.Capturer = (*FakeCapturer)(nil)

func (f *FakeCapturer) Kind() sources.SourceKind { return f.SourceKind }

func (f *FakeCapturer) Capture(ctx context.Context, spec sources.CaptureSpec) (sources.Artifact, error) {
	f.mu.Lock()
	f.Captures = append(f.Captures, spec)
	f.mu.Unlock()
	if f.CaptureFn != nil {
		return f.CaptureFn(ctx, spec)
	}
	return sources.Artifact{Path: spec.StageDir, Bytes: 0}, nil
}

func (f *FakeCapturer) Restore(ctx context.Context, spec sources.RestoreSpec) error {
	f.mu.Lock()
	f.Restores = append(f.Restores, spec)
	f.mu.Unlock()
	if f.RestoreFn != nil {
		return f.RestoreFn(ctx, spec)
	}
	return nil
}
