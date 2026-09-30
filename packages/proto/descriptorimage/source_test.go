package descriptorimage

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestSourceCachesAndReloadsByPortableStamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.binpb")
	first := descriptorBytes(t, "first.proto")
	second := descriptorBytes(t, "second.proto")
	writeFile(t, path, first)

	source, err := New(Config{DescriptorPath: path})
	if err != nil {
		t.Fatal(err)
	}
	one, err := source.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	two, err := source.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if one != two || one.Generation != 1 {
		t.Fatalf("unchanged snapshot = %#v and %#v", one, two)
	}

	writeFile(t, path, second)
	three, err := source.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if three == one || three.Generation != 2 || three.Digest == one.Digest {
		t.Fatalf("reloaded snapshot did not advance: first=%#v second=%#v", one, three)
	}
}

func TestSourceReloadsWhenWatchedManifestChanges(t *testing.T) {
	dir := t.TempDir()
	descriptorPath := filepath.Join(dir, "image.binpb")
	manifestPath := filepath.Join(dir, "manifest.json")
	writeFile(t, descriptorPath, descriptorBytes(t, "stable.proto"))
	writeFile(t, manifestPath, []byte(`{"name":"stable"}`))

	source, err := New(Config{DescriptorPath: descriptorPath, ManifestPaths: []string{manifestPath}})
	if err != nil {
		t.Fatal(err)
	}
	first, err := source.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	writeFile(t, manifestPath, []byte(`{"name":"stable","version":2}`))
	second, err := source.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if second == first || second.Generation <= first.Generation {
		t.Fatalf("manifest change did not advance snapshot: first=%d second=%d", first.Generation, second.Generation)
	}
	if second.Digest != first.Digest {
		t.Fatalf("manifest-only reload changed descriptor digest: first=%s second=%s", first.Digest, second.Digest)
	}
}

func TestSourceKeepsLastKnownGoodOnFailedReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.binpb")
	writeFile(t, path, descriptorBytes(t, "good.proto"))
	source, err := New(Config{DescriptorPath: path})
	if err != nil {
		t.Fatal(err)
	}
	good, err := source.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, []byte("truncated"))
	got, err := source.Snapshot()
	if err == nil || got != good {
		t.Fatalf("failed reload = (%#v, %v), want previous snapshot and error", got, err)
	}
	if source.LastReloadError() == nil || source.LastReloadFailureAt().IsZero() {
		t.Fatalf("failure was not observable: err=%v at=%v", source.LastReloadError(), source.LastReloadFailureAt())
	}
	if !errors.Is(err, source.LastReloadError()) && err.Error() == "" {
		t.Fatalf("reload error was empty: %v", err)
	}
}

func TestSourceReadersNeverObservePartialRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.binpb")
	first := descriptorBytes(t, "first.proto")
	second := descriptorBytes(t, "second.proto")
	writeFile(t, path, first)
	source, err := New(Config{DescriptorPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Snapshot(); err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if _, err := source.Snapshot(); err != nil {
					t.Errorf("snapshot during atomic publish: %v", err)
					return
				}
			}
		}()
	}
	for i := 0; i < 50; i++ {
		stage := filepath.Join(dir, "stage.binpb")
		if i%2 == 0 {
			writeFile(t, stage, second)
		} else {
			writeFile(t, stage, first)
		}
		if err := os.Rename(stage, path); err != nil {
			close(stop)
			wg.Wait()
			t.Fatal(err)
		}
	}
	close(stop)
	wg.Wait()
}

func TestSourceLoadWithRetry(t *testing.T) {
	source, err := New(Config{DescriptorPath: filepath.Join(t.TempDir(), "image.binpb")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.LoadWithRetry(2, time.Millisecond); !errors.Is(err, ErrNoSnapshot) {
		t.Fatalf("LoadWithRetry error = %v, want ErrNoSnapshot", err)
	}
}

func descriptorBytes(t *testing.T, name string) []byte {
	t.Helper()
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String(name), Syntax: proto.String("proto3")}}}
	raw, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func writeFile(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkSourceCachedSnapshot(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "image.binpb")
	if err := os.WriteFile(path, descriptorBytesBenchmark("benchmark.proto"), 0o644); err != nil {
		b.Fatal(err)
	}
	source, err := New(Config{DescriptorPath: path})
	if err != nil {
		b.Fatal(err)
	}
	if _, err := source.Snapshot(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := source.Snapshot(); err != nil {
			b.Fatal(err)
		}
	}
}

func descriptorBytesBenchmark(name string) []byte {
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String(name), Syntax: proto.String("proto3")}}}
	raw, err := proto.Marshal(set)
	if err != nil {
		panic(err)
	}
	return raw
}
