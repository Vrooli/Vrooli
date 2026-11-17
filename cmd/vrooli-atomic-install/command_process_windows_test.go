//go:build windows

package main

import (
	"errors"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWaitForWindowsJobDrainWaitsForAllMembers(t *testing.T) {
	now := time.Unix(10, 0)
	counts := []uint32{2, 1, 0}
	queries, sleeps, terminations := 0, 0, 0
	forced, termErr, queryErr := waitForWindowsJobDrain(func() (uint32, error) {
		count := counts[queries]
		queries++
		return count, nil
	}, func() (time.Time, bool) { return time.Time{}, false }, func() error {
		terminations++
		return nil
	}, func(d time.Duration) {
		if d != 10*time.Millisecond {
			t.Fatalf("unexpected poll delay %s", d)
		}
		sleeps++
		now = now.Add(d)
	}, func() time.Time { return now })
	if queryErr != nil || termErr != nil || forced {
		t.Fatalf("forced=%v terminate=%v query=%v", forced, termErr, queryErr)
	}
	if queries != 3 || sleeps != 2 || terminations != 0 {
		t.Fatalf("queries=%d sleeps=%d terminations=%d", queries, sleeps, terminations)
	}
}

func TestWaitForWindowsJobDrainTerminatesAfterInterruptGrace(t *testing.T) {
	now := time.Unix(10, 0)
	deadline := now.Add(20 * time.Millisecond)
	terminations := 0
	active := true
	forced, termErr, queryErr := waitForWindowsJobDrain(func() (uint32, error) {
		if active {
			return 1, nil
		}
		return 0, nil
	}, func() (time.Time, bool) { return deadline, true }, func() error {
		terminations++
		active = false
		return nil
	}, func(d time.Duration) { now = now.Add(d) }, func() time.Time { return now })
	if queryErr != nil || termErr != nil || !forced {
		t.Fatalf("forced=%v terminate=%v query=%v", forced, termErr, queryErr)
	}
	if terminations != 1 {
		t.Fatalf("termination count=%d; want 1", terminations)
	}
}

func TestWaitForWindowsJobDrainTerminatesImmediatelyWithoutGrace(t *testing.T) {
	now := time.Unix(10, 0)
	active := true
	terminations := 0
	forced, _, queryErr := waitForWindowsJobDrain(func() (uint32, error) {
		if active {
			return 1, nil
		}
		return 0, nil
	}, func() (time.Time, bool) { return now, true }, func() error {
		terminations++
		active = false
		return nil
	}, func(d time.Duration) { now = now.Add(d) }, func() time.Time { return now })
	if queryErr != nil || !forced || terminations != 1 {
		t.Fatalf("forced=%v terminations=%d query=%v", forced, terminations, queryErr)
	}
}

func TestWaitForWindowsJobDrainDoesNotReleaseOnTerminationError(t *testing.T) {
	now := time.Unix(10, 0)
	termFailure := errors.New("terminate failed")
	queries := 0
	forced, termErr, queryErr := waitForWindowsJobDrain(func() (uint32, error) {
		queries++
		if queries < 4 {
			return 1, nil
		}
		return 0, nil
	}, func() (time.Time, bool) { return now, true }, func() error { return termFailure }, func(time.Duration) {}, func() time.Time { return now })
	if queryErr != nil || !forced || !errors.Is(termErr, termFailure) || queries != 4 {
		t.Fatalf("forced=%v terminate=%v query=%v queries=%d", forced, termErr, queryErr, queries)
	}
}

func TestDuplicateWindowsHandleDoesNotChangeSourceInheritance(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	source := windows.Handle(f.Fd())
	before, err := getWindowsHandleInformation(source)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := duplicateWindowsHandleForInheritance(source)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(duplicate)
	after, err := getWindowsHandleInformation(source)
	if err != nil {
		t.Fatal(err)
	}
	dupFlags, err := getWindowsHandleInformation(duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("source handle flags changed: before=%#x after=%#x", before, after)
	}
	if dupFlags&windows.HANDLE_FLAG_INHERIT == 0 {
		t.Fatalf("duplicated handle is not inheritable: flags=%#x", dupFlags)
	}
}

func TestManagedCommandExitErrorCarriesExitCode(t *testing.T) {
	var exit interface{ ExitCode() int } = managedCommandExitError(23)
	if exit.ExitCode() != 23 {
		t.Fatalf("exit code = %d; want 23", exit.ExitCode())
	}
}
