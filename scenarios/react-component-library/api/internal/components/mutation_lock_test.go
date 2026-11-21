package components

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLibraryMutationLockSerializesAndAllowsNestedOwnerCalls(t *testing.T) {
	root := t.TempDir()
	owned, release, err := AcquireLibraryMutation(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	nested, releaseNested, err := AcquireLibraryMutation(owned, root)
	if err != nil || nested != owned {
		t.Fatalf("nested acquisition failed: %v", err)
	}
	releaseNested()
	waiting, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	_, _, err = AcquireLibraryMutation(waiting, root)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("independent writer entered held lock: %v", err)
	}
	release()
	_, releaseNext, err := AcquireLibraryMutation(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	releaseNext()
}

func TestLibraryMutationBlocksIndexAndManifestOwner(t *testing.T) {
	root := t.TempDir()
	_, release, err := AcquireLibraryMutation(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	for _, operation := range []string{"index", "manifest"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
			defer cancel()
			var err error
			if operation == "index" {
				_, err = NewIndexer(nil, root, nil).Run(ctx)
			} else {
				s := NewServiceWithContent(nil, NewFSContentStore(root))
				_, err = s.UpdateComponentManifest(ctx, UpdateComponentManifestInput{ComponentID: "old"})
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("owner bypassed mutation lock: %v", err)
			}
		})
	}
}
