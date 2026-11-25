//go:build linux

package main

import (
	"os"
	"os/exec"
	"slices"
	"testing"
)

// The memory reader walks real /proc: this test process has resident memory,
// and a child it starts is found under it.
func TestProcMemoryAndChildrenReadTheLiveProcessTable(t *testing.T) {
	if got := procMemoryBytes(os.Getpid()); got <= 0 {
		t.Fatalf("procMemoryBytes(self) = %d, want resident bytes", got)
	}
	if got := procMemoryBytes(-1); got != 0 {
		t.Fatalf("procMemoryBytes(missing) = %d, want 0", got)
	}
	child := exec.Command("sleep", "30")
	if err := child.Start(); err != nil {
		t.Skipf("cannot start a child process: %v", err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	if kids := procChildren()[os.Getpid()]; !slices.Contains(kids, child.Process.Pid) {
		t.Fatalf("children of %d = %v, want %d", os.Getpid(), kids, child.Process.Pid)
	}
}

func TestSessionAgentMemoryWithoutAgentProcessIsUnknown(t *testing.T) {
	if bytes, ok := sessionAgentMemory("no-such-session-00000000"); ok || bytes != 0 {
		t.Fatalf("sessionAgentMemory(unknown) = %d, %v; want 0, false", bytes, ok)
	}
}
