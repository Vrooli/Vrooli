package components

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	platform "github.com/vrooli/platform-go"
)

type libraryMutationKey struct{}

// AcquireLibraryMutation serializes cooperating owner processes across a complete
// source/index mutation. Pass the returned context to nested owner operations;
// never share it with an independent concurrent mutation.
func AcquireLibraryMutation(ctx context.Context, root string) (context.Context, func(), error) {
	return acquireLibraryMutation(ctx, root, false)
}

// AcquireLibraryRetirement permits the retirement owner to recover its pending
// journal while ordinary mutations remain fenced out.
func AcquireLibraryRetirement(ctx context.Context, root string) (context.Context, func(), error) {
	return acquireLibraryMutation(ctx, root, true)
}

const RetirementPendingFile = ".retirement-pending.json"

func acquireLibraryMutation(ctx context.Context, root string, recovery bool) (context.Context, func(), error) {
	noop := func() {}
	if err := ctx.Err(); err != nil {
		return ctx, noop, err
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return ctx, noop, fmt.Errorf("resolve library mutation root: %w", err)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return ctx, noop, err
	}
	if held, _ := ctx.Value(libraryMutationKey{}).(string); held == canonical {
		return ctx, noop, nil
	}
	release, err := platform.AcquireFileLockContext(ctx, filepath.Join(canonical, ".mutation.lock"))
	if err != nil {
		return ctx, noop, err
	}
	if !recovery {
		if _, err := os.Lstat(filepath.Join(canonical, RetirementPendingFile)); err == nil {
			release()
			return ctx, noop, fmt.Errorf("library has a pending retirement; recover it through the retirement owner before indexing or authoring")
		} else if !os.IsNotExist(err) {
			release()
			return ctx, noop, err
		}
	}
	return context.WithValue(ctx, libraryMutationKey{}, canonical), release, nil
}

func (s *service) acquireMutation(ctx context.Context) (context.Context, func(), error) {
	if store, ok := s.content.(*FSContentStore); ok {
		return AcquireLibraryMutation(ctx, store.Root())
	}
	return ctx, func() {}, ctx.Err()
}
