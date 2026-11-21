//go:build linux

package desktophelper

import (
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
	"path/filepath"
	"testing"
	"time"
)

func TestPrivateReadRejectsFIFOWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-config")
	require.NoError(t, unix.Mkfifo(path, 0600))
	result := make(chan error, 1)
	go func() { _, err := readPrivate(path, 1024); result <- err }()
	select {
	case err := <-result:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("private config read blocked on a FIFO")
	}
}
