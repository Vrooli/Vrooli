//go:build linux

package recovery

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestCodingAgentLedgerSpecialFileCannotHangInitialization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coding-agent-records.json")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := NewCodingAgentBroker(path, nil); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO became recovery authority")
		}
	case <-time.After(time.Second):
		// Release a regressed blocking Open before reporting the failure.
		if fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = syscall.Close(fd)
		}
		t.Fatal("special ledger file blocked recovery initialization")
	}
}
