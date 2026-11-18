//go:build linux

package usagemode

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestReadFileIgnoresFifo(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "fifo.json")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skip("no fifo support: ", err)
	}
	done := make(chan []Mode, 1)
	go func() { done <- ReadFile(fifo, t0) }()
	select {
	case modes := <-done:
		if modes != nil {
			t.Fatalf("fifo produced modes: %v", modes)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reading a FIFO blocked the launch")
	}
}
