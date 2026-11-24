package artifactfile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishStreamsHashAndNoClobber(t *testing.T) {
	workspace := t.TempDir()
	body := []byte("synthetic test-genie artifact")
	wantHash := sha256.Sum256(body)
	receipt, err := Publish(workspace, "receipts/run-1/result.bin", bytes.NewReader(body), int64(len(body)), 1024, false)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Path != "receipts/run-1/result.bin" || receipt.Bytes != int64(len(body)) || receipt.SHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("receipt = %+v", receipt)
	}
	path := filepath.Join(workspace, filepath.FromSlash(receipt.Path))
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("published body = %q, err=%v", got, err)
	}
	if _, err := Publish(workspace, receipt.Path, bytes.NewReader([]byte("replacement")), int64(len("replacement")), 1024, false); err == nil {
		t.Fatal("existing destination was overwritten without explicit permission")
	}
	got, _ = os.ReadFile(path)
	if !bytes.Equal(got, body) {
		t.Fatalf("failed no-clobber changed destination to %q", got)
	}
	if _, err := Publish(workspace, receipt.Path, bytes.NewReader([]byte("replacement")), int64(len("replacement")), 1024, true); err != nil {
		t.Fatalf("explicit overwrite: %v", err)
	}
	got, _ = os.ReadFile(path)
	if string(got) != "replacement" {
		t.Fatalf("explicit overwrite body = %q", got)
	}
}

func TestPublishRejectsTraversalSizeMismatchAndCleansTemp(t *testing.T) {
	workspace := t.TempDir()
	for _, path := range []string{"", "/tmp/out", "../escape", "nested/../../escape"} {
		if _, err := Publish(workspace, path, strings.NewReader("x"), 1, 8, false); err == nil {
			t.Errorf("accepted destination %q", path)
		}
	}
	if _, err := Publish(workspace, "bad-size", strings.NewReader("x"), 2, 8, false); err == nil {
		t.Fatal("accepted stream with wrong declared size")
	}
	if _, err := Publish(workspace, "over-cap", strings.NewReader("12345"), 5, 4, false); err == nil {
		t.Fatal("accepted stream above byte cap")
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed stream left workspace entries: %v", entries)
	}
}

func TestPublishRejectsSymlinkEscape(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(workspace, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := Publish(workspace, "escape/file.bin", strings.NewReader("x"), 1, 8, false); err == nil {
		t.Fatal("followed symlink outside the admitted workspace")
	}
	if _, err := os.Stat(filepath.Join(outside, "file.bin")); !os.IsNotExist(err) {
		t.Fatalf("outside file exists or could not be checked: %v", err)
	}
	rootLink := filepath.Join(t.TempDir(), "workspace-link")
	if err := os.Symlink(workspace, rootLink); err != nil {
		t.Fatal(err)
	}
	if _, err := Publish(rootLink, "root-escape.bin", strings.NewReader("x"), 1, 8, false); err == nil {
		t.Fatal("accepted an admitted workspace root that is itself a symlink")
	}
}
