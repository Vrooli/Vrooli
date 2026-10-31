package binaryfetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type cancelDigestReader struct {
	cancel context.CancelFunc
	reads  int
}

func TestTreeDigestRejectsAbsoluteLinkWithInTreeLookalike(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "external")
	if err := os.WriteFile(external, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	// filepath.Join does not implement absolute symlink resolution. Populate
	// its misleading in-tree result so a dangling-target check cannot mask it.
	lookalike := filepath.Join(root, external)
	if err := os.MkdirAll(filepath.Dir(lookalike), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lookalike, []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "tool")); err != nil {
		t.Fatal(err)
	}
	if _, err := TreeDigestContext(t.Context(), root); err == nil {
		t.Fatal("accepted an absolute external symlink because its in-tree lookalike exists")
	}
}

func TestTreeDigestRejectsRootAliasAndChainedEscape(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "tree")
	if err := os.MkdirAll(filepath.Join(root, ".bin"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "outside"), filepath.Join(parent, "outside")} {
		if err := os.WriteFile(path, []byte("same version"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(".", filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, ".bin", "tool")
	if err := os.Symlink("../a/../outside", link); err != nil {
		t.Fatal(err)
	}
	actual, err := filepath.EvalSymlinks(link)
	if err != nil || actual != filepath.Join(parent, "outside") {
		t.Fatalf("fixture did not escape: %q, %v", actual, err)
	}
	if _, err := TreeDigestContext(t.Context(), root); err == nil {
		t.Error("accepted a link chain whose actual target escapes the tree")
	}
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := TreeDigestContext(t.Context(), alias); err == nil {
		t.Error("accepted root symlink as an empty artifact tree")
	}
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

func TestTreeDigestParallelismIsBoundedAndDeterministic(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("file-%02d", i)), []byte(strings.Repeat("x", i+1)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var active, maximum int32
	digest := func(ctx context.Context, path string) (string, error) {
		current := atomic.AddInt32(&active, 1)
		defer atomic.AddInt32(&active, -1)
		for old := atomic.LoadInt32(&maximum); current > old && !atomic.CompareAndSwapInt32(&maximum, old, current); old = atomic.LoadInt32(&maximum) {
		}
		time.Sleep(5 * time.Millisecond)
		return FileDigestContext(ctx, path)
	}
	got, err := treeDigestContext(context.Background(), root, 32, digest)
	if err != nil {
		t.Fatal(err)
	}
	want, err := TreeDigest(root)
	if err != nil || got != want {
		t.Fatalf("parallel digest changed identity: got=%s want=%s err=%v", got, want, err)
	}
	if maximum < 2 || maximum > treeDigestWorkers {
		t.Fatalf("parallel workers = %d, want 2..%d", maximum, treeDigestWorkers)
	}
}

func TestTreeDigestParallelErrorsKeepWalkOrderAndDrain(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	marker := errors.New("first file digest failure")
	if err := os.Symlink("../../escape", filepath.Join(root, "z")); err != nil {
		t.Skip(err)
	}
	var started, completed int32
	digest := func(ctx context.Context, path string) (string, error) {
		atomic.AddInt32(&started, 1)
		defer atomic.AddInt32(&completed, 1)
		if filepath.Base(path) == "a" {
			return "", marker
		}
		return FileDigestContext(ctx, path)
	}
	_, err := treeDigestContext(context.Background(), root, treeDigestWorkers, digest)
	if !errors.Is(err, marker) {
		t.Fatalf("walk did not report earliest digest failure: %v", err)
	}
	if started != completed {
		t.Fatalf("returned before in-flight digests drained: started=%d completed=%d", started, completed)
	}
}

func TestTreeDigestCancellationDrainsInflightFiles(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("file-%02d", i)), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var started, completed int32
	digest := func(ctx context.Context, _ string) (string, error) {
		if atomic.AddInt32(&started, 1) == treeDigestWorkers {
			cancel()
		}
		defer atomic.AddInt32(&completed, 1)
		<-ctx.Done()
		return "", ctx.Err()
	}
	_, err := treeDigestContext(ctx, root, treeDigestWorkers, digest)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was not preserved: %v", err)
	}
	if started != completed {
		t.Fatalf("returned before canceled work drained: started=%d completed=%d", started, completed)
	}
}

// BenchmarkTreeDigestManagedTree is opt-in: set BINARYFETCH_BENCH_TREE to an
// explicitly selected managed installation tree before asking Go to run it.
func BenchmarkTreeDigestManagedTree(b *testing.B) {
	root := strings.TrimSpace(os.Getenv("BINARYFETCH_BENCH_TREE"))
	if root == "" {
		b.Skip("set BINARYFETCH_BENCH_TREE to the managed tree to measure")
	}
	if !filepath.IsAbs(root) {
		b.Fatalf("BINARYFETCH_BENCH_TREE must be absolute: %q", root)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		b.Fatalf("BINARYFETCH_BENCH_TREE must name a readable directory: %q (%v)", root, err)
	}
	single, err := treeDigestContext(context.Background(), root, 1, FileDigestContext)
	if err != nil {
		b.Fatalf("single-worker identity check: %v", err)
	}
	parallel, err := treeDigestContext(context.Background(), root, treeDigestWorkers, FileDigestContext)
	if err != nil {
		b.Fatalf("parallel identity check: %v", err)
	}
	if single != parallel {
		b.Fatalf("worker modes produced different tree digests: single=%s parallel=%s", single, parallel)
	}
	b.Log("measures local TreeDigestContext worker modes; it does not establish native-preview performance")
	b.Run("workers=1", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := treeDigestContext(context.Background(), root, 1, FileDigestContext); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("workers=4", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := treeDigestContext(context.Background(), root, treeDigestWorkers, FileDigestContext); err != nil {
				b.Fatal(err)
			}
		}
	})
}
