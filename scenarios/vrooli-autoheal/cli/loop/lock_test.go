package main

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAcquireLoopLockPreventsSecondSupervisor(t *testing.T) {
	statusPath := filepath.Join(t.TempDir(), "state", "loop-status.json")
	first, err := acquireLoopLock(statusPath)
	if err != nil {
		t.Fatalf("first acquireLoopLock() error = %v", err)
	}
	defer first.close()

	second, err := acquireLoopLock(statusPath)
	if second != nil {
		second.close()
		t.Fatal("second loop acquired the supervisor lock")
	}
	if !errors.Is(err, errLoopAlreadyRunning) {
		t.Fatalf("second acquireLoopLock() error = %v, want errLoopAlreadyRunning", err)
	}

	first.close()
	third, err := acquireLoopLock(statusPath)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	third.close()
}

func TestParseFlagsRejectsLoopSubcommands(t *testing.T) {
	if _, err := parseFlags([]string{"status", "--json"}); err == nil {
		t.Fatal("parseFlags accepted a loop subcommand and would start a second supervisor")
	}
}
