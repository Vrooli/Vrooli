package binaryfetch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalLeafDigestSerializationMatchesActualTree(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b"), []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	actual, err := TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	a, err := FileDigestContext(t.Context(), filepath.Join(root, "a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := FileDigestContext(t.Context(), filepath.Join(root, "b"))
	if err != nil {
		t.Fatal(err)
	}
	leaves := []treeLeafDigest{{Path: "b", SHA256: b}, {Path: "a", SHA256: a}}
	got, err := hashTreeLeafDigestsContext(context.Background(), leaves)
	if err != nil || got != actual {
		t.Fatalf("canonical records=%s actual=%s err=%v", got, actual, err)
	}
	if _, err := hashTreeLeafDigestsContext(t.Context(), append(leaves, leaves[0])); err == nil {
		t.Fatal("duplicate filesystem leaf identity accepted")
	}
	missing, err := hashTreeLeafDigestsContext(t.Context(), leaves[:1])
	if err != nil || missing == actual {
		t.Fatal("omitted leaf concealed by canonical serialization")
	}
}
