package artifactlock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

func TestAcquireContextCancelsQueuedGateAndReleasesIdempotently(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.lock")
	key := path
	pathFn := func(string) (string, error) { return path, nil }
	first, err := AcquireWithPath(key, pathFn)
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := AcquireWithPathContext(ctx, key, pathFn); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for i := 0; i < 3; i++ {
		group.Add(1)
		go func() { defer group.Done(); first() }()
	}
	group.Wait()
	next, err := AcquireWithPathContext(context.Background(), key, pathFn)
	if err != nil {
		t.Fatal(err)
	}
	next()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("artifact ownership mode lost: %v %v", info, err)
	}
}

func TestAcquireContextNativeContentionAndPrecancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native.lock")
	first, err := platform.AcquireFileLockContext(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	pathFn := func(string) (string, error) { return path, nil }
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := AcquireWithPathContext(ctx, "different-in-process-key", pathFn); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	first()
	next, err := AcquireWithPathContext(context.Background(), "different-in-process-key", pathFn)
	if err != nil {
		t.Fatal(err)
	}
	next()
	canceled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := AcquireWithPathContext(canceled, "pre-cancel", func(string) (string, error) { t.Fatal("canceled caller resolved a path"); return "", nil }); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestArtifactLockAliasCannotChangeForeignFile(t *testing.T) {
	for _, hard := range []bool{false, true} {
		t.Run(fmt.Sprint(hard), func(t *testing.T) {
			dir := t.TempDir()
			foreign := filepath.Join(dir, "foreign")
			path := filepath.Join(dir, "artifact.lock")
			if err := os.WriteFile(foreign, []byte("foreign contents"), 0o644); err != nil {
				t.Fatal(err)
			}
			var err error
			if hard {
				err = os.Link(foreign, path)
			} else {
				err = os.Symlink(foreign, path)
			}
			if err != nil {
				t.Fatal(err)
			}
			release, err := AcquireWithPathContext(context.Background(), path, func(string) (string, error) { return path, nil })
			if err == nil {
				release()
				t.Error("aliased lock file accepted")
			}
			info, err := os.Stat(foreign)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(foreign)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o644 || string(data) != "foreign contents" {
				t.Fatalf("foreign file changed through lock alias: mode=%v data=%q", info.Mode().Perm(), data)
			}
		})
	}
}
