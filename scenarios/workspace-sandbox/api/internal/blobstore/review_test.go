package blobstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

func TestReviewMaterializationResumesOneUnpublishedTree(t *testing.T) {
	store, _ := newTestStore(t)
	snapshot := &types.ReviewSnapshot{SandboxID: uuid.New(), RequestID: uuid.New()}
	snapshot.ID = types.ReviewSnapshotID(snapshot.SandboxID, snapshot.RequestID)
	patch, err := store.Put(t.Context(), snapshot.ID.String(), []byte("retained patch"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot.PatchSHA256 = patch.SHA256Hex
	snapshot.InputBytes = patch.SizeUncompressed
	snapshot.SHA256 = snapshot.ContentSHA256()
	anchor, err := store.blobPath(snapshot.ID.String(), snapshot.PatchSHA256)
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(filepath.Dir(anchor), "review-tree.pending")
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "interrupted"), []byte("not source"), 0600); err != nil {
		t.Fatal(err)
	}
	release, err := store.LockReview(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	root, err := store.MaterializeReview(t.Context(), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatalf("staging survived publication: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "interrupted")); !os.IsNotExist(err) {
		t.Fatalf("partial input became published: %v", err)
	}
	if replay, err := store.MaterializeReview(t.Context(), snapshot); err != nil || replay != root {
		t.Fatalf("replay: %s, %v", replay, err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := store.MaterializeReview(cancelled, snapshot); err == nil {
		t.Fatal("cancelled replay succeeded")
	}
}

func TestReviewTreeRejectsUnsafeManifestBeforeWriting(t *testing.T) {
	file := types.ReviewFile{FileFingerprint: types.FileFingerprint{Path: "safe", SHA256: strings.Repeat("a", 64), Mode: 100644}, Size: 1}
	for _, kind := range []string{"traversal", "absolute", "unclean", "duplicate", "symlink-parent", "mode", "oversize", "count"} {
		t.Run(kind, func(t *testing.T) {
			snapshot := &types.ReviewSnapshot{Before: []types.ReviewFile{file}}
			switch kind {
			case "traversal":
				snapshot.Before[0].Path = "../outside"
			case "absolute":
				snapshot.Before[0].Path = "/outside"
			case "unclean":
				snapshot.Before[0].Path = "a/../b"
			case "duplicate":
				snapshot.Before = append(snapshot.Before, file)
			case "symlink-parent":
				snapshot.Before[0].Mode = 120000
				child := file
				child.Path = "safe/child"
				snapshot.Before = append(snapshot.Before, child)
			case "mode":
				snapshot.Before[0].Mode = 010000
			case "oversize":
				snapshot.Before[0].Size = types.MaxReviewFileBytes + 1
			case "count":
				snapshot.Before = make([]types.ReviewFile, 2*types.MaxReviewFiles+1)
			}
			if _, err := reviewTreeFiles(snapshot); err == nil {
				t.Fatal("unsafe retained manifest accepted")
			}
		})
	}
}

func TestReviewBodyReaderBoundsExpandedBytes(t *testing.T) {
	store, _ := newTestStore(t)
	id := uuid.New().String()
	stored, err := store.Put(t.Context(), id, []byte(strings.Repeat("a", 4096)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.get(t.Context(), id, stored.SHA256Hex, 64); err == nil {
		t.Fatal("expanded body exceeded read limit")
	}
	if body, err := store.Get(t.Context(), id, stored.SHA256Hex); err != nil || len(body) != 4096 {
		t.Fatalf("ordinary blob read changed: %d, %v", len(body), err)
	}
}
