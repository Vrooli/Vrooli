//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package binaryfetch

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestDigestOpenDoesNotBlockOnReplacedFIFO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skip(err)
	}
	type result struct {
		file *os.File
		err  error
	}
	done := make(chan result, 1)
	go func() { file, err := openDigestFile(path); done <- result{file, err} }()
	select {
	case got := <-done:
		if got.err != nil {
			return
		}
		defer got.file.Close()
		info, err := got.file.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().IsRegular() {
			t.Fatal("FIFO reported as regular")
		}
	case <-time.After(time.Second):
		// Release an accidentally blocking open before failing, rather than leave
		// a test goroutine stranded. This never starts a process or reads the FIFO.
		rescue, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0)
		if err == nil {
			defer rescue.Close()
		}
		select {
		case got := <-done:
			if got.file != nil {
				got.file.Close()
			}
		case <-time.After(time.Second):
		}
		t.Fatal("digest open blocked on a special-file replacement")
	}
}
