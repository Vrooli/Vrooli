package resourcedeployment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheArtifactDigestRejectsAbsoluteLinkWithInTreeLookalike(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "outside")
	lookalike := filepath.Join(root, external)
	if err := os.MkdirAll(filepath.Dir(lookalike), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{external, lookalike} {
		if err := os.WriteFile(path, []byte("same bytes"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "payload")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	if _, err := cacheArtifactDigest(t.Context(), root, link); err == nil {
		t.Fatal("accepted an absolute external artifact symlink")
	}
}

func TestCacheArtifactDigestRejectsChainedEscape(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "tree")
	if err := os.MkdirAll(filepath.Join(root, ".bin"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "outside"), filepath.Join(parent, "outside")} {
		if err := os.WriteFile(path, []byte("same bytes"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(".", filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, ".bin", "payload")
	if err := os.Symlink("../a/../outside", link); err != nil {
		t.Fatal(err)
	}
	if _, err := cacheArtifactDigest(t.Context(), root, link); err == nil {
		t.Fatal("accepted a link chain whose actual target escapes the cache")
	}
}

func TestStageCachedArtifactsCancellationDoesNotWaitForAnotherBuild(t *testing.T) {
	cacheRoot, outputRoot := t.TempDir(), t.TempDir()
	key := ArtifactCacheKey{Resource: "python", Version: "1", Platform: Platform{OS: "linux", Arch: "amd64"}, SourceIdentity: "cancel-waiter"}
	entered, release := make(chan struct{}), make(chan struct{})
	first := make(chan error, 1)
	go func() {
		_, err := StageCachedArtifacts(context.Background(), cacheRoot, outputRoot, key, func(_ context.Context, staging string) error {
			close(entered)
			<-release
			return os.WriteFile(filepath.Join(staging, "payload"), []byte("verified"), 0o644)
		})
		first <- err
	}()
	defer func() {
		close(release)
		if err := <-first; err != nil {
			t.Errorf("original owner build: %v", err)
		}
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	second := make(chan error, 1)
	go func() {
		_, err := StageCachedArtifacts(ctx, cacheRoot, t.TempDir(), key, func(context.Context, string) error {
			t.Error("canceled waiter acquired a build")
			return nil
		})
		second <- err
	}()
	cancel()
	select {
	case err := <-second:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiter result = %v, want cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("canceled waiter remained blocked behind active build")
	}
}

func TestStageCachedArtifactsReusesVerifiedResourcePlatformOutput(t *testing.T) {
	cacheRoot := t.TempDir()
	outputRoot := t.TempDir()
	key := ArtifactCacheKey{Resource: "postgres", Version: "16.4", Platform: Platform{OS: "linux", Arch: "amd64"}, SourceIdentity: "manifest-and-module-sha256"}
	builds := 0
	build := func(_ context.Context, staging string) error {
		builds++
		return os.WriteFile(filepath.Join(staging, "postgres-linux-amd64.tar.gz"), []byte("postgres bytes"), 0o644)
	}

	first, err := StageCachedArtifacts(context.Background(), cacheRoot, outputRoot, key, build)
	if err != nil {
		t.Fatal(err)
	}
	second, err := StageCachedArtifacts(context.Background(), cacheRoot, outputRoot, key, build)
	if err != nil {
		t.Fatal(err)
	}
	if builds != 1 {
		t.Fatalf("build count = %d, want one build and one cache hit", builds)
	}
	if len(first) != 1 || len(second) != 1 || first[0] != second[0] {
		t.Fatalf("cached artifact records differ: first=%+v second=%+v", first, second)
	}
	if data, err := os.ReadFile(filepath.Join(outputRoot, first[0].Name)); err != nil || string(data) != "postgres bytes" {
		t.Fatalf("staged artifact = %q, err=%v", data, err)
	}

	// A cache hit is valid only while the stored bytes still match the
	// independently recorded checksum.
	cacheEntries, err := os.ReadDir(cacheRoot)
	if err != nil || len(cacheEntries) != 1 {
		t.Fatalf("cache entries = %v, err=%v", cacheEntries, err)
	}
	cacheArtifact := filepath.Join(cacheRoot, cacheEntries[0].Name(), first[0].Name)
	if err := os.WriteFile(cacheArtifact, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := StageCachedArtifacts(context.Background(), cacheRoot, outputRoot, key, build); err != nil {
		t.Fatal(err)
	}
	if builds != 2 {
		t.Fatalf("build count after corrupt cache = %d, want rebuild", builds)
	}
}

func TestStageCachedArtifactsSeparatesSourceVersionAndPlatform(t *testing.T) {
	cacheRoot := t.TempDir()
	builds := 0
	build := func(_ context.Context, staging string) error {
		builds++
		return os.WriteFile(filepath.Join(staging, "artifact"), []byte("bytes"), 0o644)
	}
	for _, changed := range []ArtifactCacheKey{
		{Resource: "postgres", Version: "16.5", Platform: Platform{OS: "linux", Arch: "amd64"}, SourceIdentity: "source-a"},
		{Resource: "postgres", Version: "16.4", Platform: Platform{OS: "linux", Arch: "arm64"}, SourceIdentity: "source-a"},
		{Resource: "postgres", Version: "16.4", Platform: Platform{OS: "linux", Arch: "amd64"}, SourceIdentity: "source-b"},
	} {
		if _, err := StageCachedArtifacts(context.Background(), cacheRoot, t.TempDir(), changed, build); err != nil {
			t.Fatal(err)
		}
	}
	if builds != 3 {
		t.Fatalf("build count = %d, want distinct build for each version/platform/source", builds)
	}
}

func TestStageCachedArtifactsSerializesConcurrentBuildsForSameKey(t *testing.T) {
	cacheRoot := t.TempDir()
	key := ArtifactCacheKey{Resource: "postgres", Version: "16.4", Platform: Platform{OS: "linux", Arch: "amd64"}, SourceIdentity: "source-a"}
	var builds atomic.Int32
	build := func(_ context.Context, staging string) error {
		builds.Add(1)
		return os.WriteFile(filepath.Join(staging, "artifact"), []byte("bytes"), 0o644)
	}
	const callers = 8
	var group sync.WaitGroup
	errCh := make(chan error, callers)
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := StageCachedArtifacts(context.Background(), cacheRoot, t.TempDir(), key, build)
			errCh <- err
		}()
	}
	group.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := builds.Load(); got != 1 {
		t.Fatalf("concurrent build count = %d, want one", got)
	}
}

func TestStageCachedArtifactsPreservesSafeSymlinksInDirectoryArtifacts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("safe symlink behavior is required by the Linux resource artifact target")
	}
	cacheRoot, outputRoot := t.TempDir(), t.TempDir()
	key := ArtifactCacheKey{Resource: "postgres", Version: "16.15.0", Platform: Platform{OS: "linux", Arch: "amd64"}, SourceIdentity: "image-digest-and-manifest"}
	builds := 0
	build := func(_ context.Context, staging string) error {
		builds++
		dir := filepath.Join(staging, "postgres_linux_amd64")
		if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "bin", "postgres"), []byte("server"), 0o755); err != nil {
			return err
		}
		return os.Symlink("bin/postgres", filepath.Join(dir, "current"))
	}
	if _, err := StageCachedArtifacts(context.Background(), cacheRoot, outputRoot, key, build); err != nil {
		t.Fatal(err)
	}
	if builds != 1 {
		t.Fatalf("build count = %d, want one", builds)
	}
	linkPath := filepath.Join(outputRoot, "postgres_linux_amd64", "current")
	if info, err := os.Lstat(linkPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("staged current entry is not a symlink: info=%v err=%v", info, err)
	}
	if target, err := os.Readlink(linkPath); err != nil || target != "bin/postgres" {
		t.Fatalf("staged symlink target = %q, err=%v", target, err)
	}
	if _, err := StageCachedArtifacts(context.Background(), cacheRoot, outputRoot, key, build); err != nil {
		t.Fatal(err)
	}
	if builds != 1 {
		t.Fatalf("build count after verified directory cache hit = %d, want one", builds)
	}
}

func TestStageCachedArtifactsReplacesDirectoryOutputWithoutStaleEntries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("safe symlink behavior is required by the Linux resource artifact target")
	}
	cacheRoot, outputRoot := t.TempDir(), t.TempDir()
	key := ArtifactCacheKey{Resource: "postgres", Version: "16.15.0", Platform: Platform{OS: "linux", Arch: "amd64"}, SourceIdentity: "before-normalization"}
	firstBuild := func(_ context.Context, staging string) error {
		dir := filepath.Join(staging, "postgres_linux_amd64")
		if err := os.MkdirAll(filepath.Join(dir, "usr", "local", "share", "man"), 0o755); err != nil {
			return err
		}
		return os.Symlink("share/man", filepath.Join(dir, "usr", "local", "man"))
	}
	if _, err := StageCachedArtifacts(context.Background(), cacheRoot, outputRoot, key, firstBuild); err != nil {
		t.Fatal(err)
	}
	staleLink := filepath.Join(outputRoot, "postgres_linux_amd64", "usr", "local", "man")
	if _, err := os.Lstat(staleLink); err != nil {
		t.Fatalf("first tree link missing: %v", err)
	}

	key.SourceIdentity = "after-normalization"
	secondBuild := func(_ context.Context, staging string) error {
		return os.MkdirAll(filepath.Join(staging, "postgres_linux_amd64", "usr", "local"), 0o755)
	}
	if _, err := StageCachedArtifacts(context.Background(), cacheRoot, outputRoot, key, secondBuild); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(staleLink); !os.IsNotExist(err) {
		t.Fatalf("stale tree entry remains after replacement (err %v)", err)
	}
}
