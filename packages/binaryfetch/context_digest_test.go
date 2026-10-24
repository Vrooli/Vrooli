package binaryfetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type cancelDigestReader struct {
	cancel context.CancelFunc
	reads  int
}

func (r *cancelDigestReader) Read(p []byte) (int, error) {
	r.reads++
	if r.reads > 1 {
		return 0, io.EOF
	}
	copy(p, strings.Repeat("x", len(p)))
	r.cancel()
	return len(p), nil
}

func TestDigestReaderContextStopsBetweenChunks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &cancelDigestReader{cancel: cancel}
	sum, err := digestReaderContext(ctx, r)
	if !errors.Is(err, context.Canceled) || sum != "" || r.reads != 1 {
		t.Fatalf("continued after mid-stream cancellation: %q %v reads=%d", sum, err, r.reads)
	}
}

func TestFetchCancellationAtVerificationDoesNotPublish(t *testing.T) {
	body := bigBinary()
	server := serve(t, body)
	destination := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := Fetch(ctx, Target{Name: "mytool", URL: server.URL, SHA256: sha256hex(body)}, destination, func(p Progress) {
		if p.Stage == StageVerifying {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("verification continued after cancellation: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(destination, "mytool")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled fetch published output: %v", err)
	}
}

func TestContextDigestsPreserveExactTreeAndVerification(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "z"), []byte("zeta"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a", filepath.Join(root, "link")); err != nil {
		t.Skip(err)
	}
	sha := func(body string) string { v := sha256.Sum256([]byte(body)); return hex.EncodeToString(v[:]) }
	expected := sha("a\x00" + sha("alpha") + "\nlink\x00" + sha("symlink\x00a") + "\nz\x00" + sha("zeta") + "\n")
	legacy, err := TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := TreeDigestContext(context.Background(), root)
	if err != nil || got != expected || legacy != expected {
		t.Fatalf("digest format changed: %s %s %v want %s", legacy, got, err, expected)
	}
	if err := VerifyArtifactContext(context.Background(), root, "dir", expected); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := TreeDigestContext(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("tree ignored canceled context: %v", err)
	}
	if err := VerifyArtifactContext(ctx, root, "dir", expected); !errors.Is(err, context.Canceled) {
		t.Fatalf("verification ignored canceled context: %v", err)
	}
	fileSum, err := FileDigestContext(context.Background(), filepath.Join(root, "link"))
	if err != nil || fileSum != sha("alpha") {
		t.Fatalf("legacy final file symlink semantics changed: %s %v", fileSum, err)
	}
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyArtifactContext(context.Background(), root, "dir", expected); err == nil {
		t.Fatal("true corruption accepted")
	}
}
