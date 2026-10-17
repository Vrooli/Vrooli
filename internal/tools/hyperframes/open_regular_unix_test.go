//go:build unix

package hyperframes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func assertFIFORefusedBoundedly(t *testing.T, path string, run func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- run() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("special-file source accepted")
		}
	case <-time.After(250 * time.Millisecond):
		// Release a regressed blocking open before failing, so the test never
		// leaves a blocked goroutine/descriptor in the shared test process.
		rescue, e := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0)
		if e == nil {
			defer rescue.Close()
		}
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("FIFO regression did not release after rescue")
		}
		t.Fatal("regular-file operation blocked waiting for a FIFO writer")
	}
}

func TestRegularOpenRejectsFIFOReplacementBeforeOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(path, []byte("regular"), 0600); err != nil {
		t.Fatal(err)
	}
	assertFIFORefusedBoundedly(t, path, func() error {
		f, err := openCheckedRegular(context.Background(), path, 1024, func(path string) (*os.File, error) {
			if err := os.Remove(path); err != nil {
				return nil, err
			}
			if err := syscall.Mkfifo(path, 0600); err != nil {
				return nil, err
			}
			return openRegularFile(path)
		})
		if f != nil {
			_ = f.Close()
		}
		return err
	})
}

func TestReadHashAndCopyRejectFIFOWithoutWriter(t *testing.T) {
	for _, kind := range []string{"read", "hash", "copy"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "fifo")
			if err := syscall.Mkfifo(path, 0600); err != nil {
				t.Fatal(err)
			}
			assertFIFORefusedBoundedly(t, path, func() error {
				switch kind {
				case "read":
					_, err := readFileContext(context.Background(), path, 1024)
					return err
				case "hash":
					_, err := fileHashContext(context.Background(), path)
					return err
				default:
					return copyInputContext(context.Background(), path, filepath.Join(t.TempDir(), "copy"), strings.Repeat("0", 64))
				}
			})
		})
	}
}

func TestRegularOpenRejectsSymlinkAndInodeReplacement(t *testing.T) {
	for _, kind := range []string{"symlink", "regular"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "source")
			other := filepath.Join(dir, "other")
			if err := os.WriteFile(path, []byte("before"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(other, []byte("after"), 0600); err != nil {
				t.Fatal(err)
			}
			f, err := openCheckedRegular(context.Background(), path, 1024, func(path string) (*os.File, error) {
				if kind == "symlink" {
					if e := os.Remove(path); e != nil {
						return nil, e
					}
					if e := os.Symlink(other, path); e != nil {
						return nil, e
					}
				} else if e := os.Rename(other, path); e != nil {
					return nil, e
				}
				return openRegularFile(path)
			})
			if f != nil {
				_ = f.Close()
			}
			if err == nil {
				t.Fatal("replacement source accepted")
			}
		})
	}
}
