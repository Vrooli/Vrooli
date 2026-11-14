//go:build linux

package storageactivity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
)

func TestProcessExitDoesNotBecomeAFalsePermissionBlocker(t *testing.T) {
	for _, err := range []error{syscall.ESRCH, os.ErrNotExist, &os.PathError{Op: "open", Path: "/proc/123/maps", Err: syscall.ESRCH}} {
		if !processExited(err) {
			t.Fatalf("did not recognize exited process: %v", err)
		}
	}
	for _, err := range []error{nil, syscall.EACCES, syscall.EPERM, errors.New("unreadable")} {
		if processExited(err) {
			t.Fatalf("permission/unknown error incorrectly ignored: %v", err)
		}
	}
}

func TestObservationScopesReferencesToTemporaryRoots(t *testing.T) {
	root := t.TempDir()
	proc := filepath.Join(root, "proc", "123")
	if err := os.MkdirAll(filepath.Join(proc, "fd"), 0700); err != nil {
		t.Fatal(err)
	}
	temporary := filepath.Join(root, "temporary")
	if err := os.MkdirAll(temporary, 0700); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(temporary, "payload")
	if err := os.WriteFile(payload, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "private")
	cache := filepath.Join(temporary, "private-build-cache")
	if err := os.Mkdir(cache, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proc, "environ"), []byte("GOCACHE="+cache+"\x00API_KEY="+filepath.Join(temporary, "do-not-publish")+"\x00TMPDIR="+temporary+"\x00"), 0600); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"fd/4": payload, "cwd": temporary, "exe": outside} {
		if err := os.Symlink(target, filepath.Join(proc, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(proc, "maps"), []byte("0001-0002 r-xp 0000 00:00 1 "+payload+"\n0002-0003 r-xp 0000 00:00 1 "+outside+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := observe(context.Background(), filepath.Dir(proc), uint32(os.Getuid()), []string{temporary})
	if err != nil || !got.Complete {
		t.Fatalf("observation=%#v err=%v", got, err)
	}
	if len(got.Paths) != 3 || got.Paths[0] != temporary || got.Paths[1] != payload || got.Paths[2] != cache {
		t.Fatalf("out-of-scope references escaped: %v", got.Paths)
	}
}
func TestObservationCancellationCannotProduceCompleteProof(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, strconv.Itoa(os.Getpid())), 0700); err != nil {
		t.Fatal(err)
	}
	got, err := observe(ctx, root, uint32(os.Getuid()), []string{"/tmp"})
	if err == nil || got.Complete {
		t.Fatalf("cancellation produced complete proof: %#v %v", got, err)
	}
}
func TestMappedTemporaryPathPreservesSpaces(t *testing.T) {
	path := "/tmp/build with  two spaces/payload"
	if got := mappedPath("0001-0002 r-xp 0000 00:00 1     " + path); got != path {
		t.Fatalf("mapped reference changed: %q", got)
	}
}
