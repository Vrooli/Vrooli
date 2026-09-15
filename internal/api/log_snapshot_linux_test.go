package api

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestLogSnapshotRejectsFIFOWithoutWriter(t *testing.T) {
	for _, linked := range []bool{false, true} {
		name := "direct"
		if linked {
			name = "symlink"
		}
		t.Run(name, func(t *testing.T) {
			fifo := filepath.Join(t.TempDir(), "pipe")
			if err := syscall.Mkfifo(fifo, 0600); err != nil {
				t.Fatal(err)
			}
			path := fifo
			if linked {
				path = fifo + ".log"
				if err := os.Symlink(fifo, path); err != nil {
					t.Fatal(err)
				}
			}
			result := make(chan error, 1)
			go func() {
				_, err := (&App{}).readTail(path, "1")
				result <- err
			}()
			select {
			case err := <-result:
				if err == nil {
					t.Fatal("snapshot accepted a named pipe")
				}
			case <-time.After(2 * time.Second):
				// Unblock the regressed reader before failing, without leaving a
				// writer or goroutine behind in the test process.
				rescue, err := os.OpenFile(fifo, os.O_RDWR|syscall.O_NONBLOCK, 0)
				if err != nil {
					t.Fatal(err)
				}
				defer rescue.Close()
				select {
				case <-result:
				case <-time.After(2 * time.Second):
					t.Fatal("reader did not recover after writer connected")
				}
				t.Fatal("log snapshot blocked waiting for a pipe writer")
			}
		})
	}
}
