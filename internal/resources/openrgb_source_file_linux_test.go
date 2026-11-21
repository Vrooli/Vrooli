//go:build linux

package resources

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestOpenRGBSourceOpenRefusesReplacedFIFOBeforeRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "acquired")
	if err := os.WriteFile(path, []byte("observed regular source"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = unix.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		f, err := openOpenRGBSourceFile(path, expected)
		if f != nil {
			f.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO replacement accepted as acquired source")
		}
	case <-time.After(time.Second):
		// Release a regressed blocking reader before failing; leave no worker.
		fd, e := unix.Open(path, unix.O_WRONLY|unix.O_NONBLOCK, 0)
		if e == nil {
			unix.Close(fd)
		}
		<-done
		t.Fatal("source open blocked on a FIFO before validating its type")
	}
}
