package resources

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestComposeCandidateInputRefusesMutationDuringRead(t *testing.T) {
	for _, kind := range []string{"in-place", "replacement"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "source.py")
			if err := os.WriteFile(path, []byte("original source"), 0600); err != nil {
				t.Fatal(err)
			}
			mutated := false
			err := copyCandidateInput(context.Background(), root, "source.py", candidateWriter(func(p []byte) (int, error) {
				if mutated {
					t.Fatal("unexpected second copy")
				}
				mutated = true
				if kind == "replacement" {
					other := filepath.Join(root, "replacement")
					if err := os.WriteFile(other, []byte("replacement source"), 0600); err != nil {
						return 0, err
					}
					if err := os.Rename(other, path); err != nil {
						return 0, err
					}
				} else if err := os.WriteFile(path, []byte("changed length"), 0600); err != nil {
					return 0, err
				}
				return len(p), nil
			}))
			if err == nil || !mutated {
				t.Fatal("mutated input accepted", err)
			}
		})
	}
}

func TestComposeCandidateInputOpenCannotFollowSymlinksOrBlockOnFIFO(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "source"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "directory")); err != nil {
		t.Fatal(err)
	}
	if f, err := openCandidateInput(root, "directory/source"); err == nil {
		f.Close()
		t.Fatal("directory symlink followed")
	}
	if err := os.Symlink(filepath.Join(outside, "source"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if f, err := openCandidateInput(root, "link"); err == nil {
		f.Close()
		t.Fatal("final symlink followed")
	}
	fifo := filepath.Join(root, "fifo")
	if err := unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	// The low-level open must return without any writer, even after a regular
	// source was replaced between path inspection and open.
	f, err := openCandidateInput(root, "fifo")
	if err != nil {
		t.Fatal(err)
	}
	info, err := f.Stat()
	f.Close()
	if err != nil || info.Mode().IsRegular() {
		t.Fatal("FIFO fixture invalid", err)
	}
	if err := copyCandidateInput(context.Background(), root, "fifo", io.Discard); err == nil {
		t.Fatal("FIFO accepted")
	}
}

type candidateWriter func([]byte) (int, error)

func (f candidateWriter) Write(p []byte) (int, error) { return f(p) }

func TestComposeCandidatePublicManifestFIFOIsBounded(t *testing.T) {
	if os.Getenv("VROOLI_TEST_CANDIDATE_FIFO_CHILD") == "1" {
		base := t.TempDir()
		root := filepath.Join(base, "resources", "fifo-fixture")
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		if err := unix.Mkfifo(filepath.Join(root, "resource.json"), 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if _, err := (&Controller{Root: base}).ComposeCandidate(ctx, "fifo-fixture", filepath.Join(base, "out")); err == nil {
			t.Fatal("nonregular initial manifest accepted")
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestComposeCandidatePublicManifestFIFOIsBounded$", "-test.count=1")
	child.Env = append(os.Environ(), "VROOLI_TEST_CANDIDATE_FIFO_CHILD=1")
	output, err := child.CombinedOutput()
	if err != nil {
		t.Fatalf("initial manifest read did not refuse FIFO within bounded wait: %v %s", err, output)
	}
}
