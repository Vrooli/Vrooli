//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"golang.org/x/sys/windows"
)

const snapshotLeaseWindowsHelperEnv = "VROOLI_SNAPSHOT_WINDOWS_HELPER"

func TestWindowsSnapshotLeaseManagerHelper(t *testing.T) {
	mode := os.Getenv(snapshotLeaseWindowsHelperEnv)
	if mode != "manager" && mode != "root-manager" {
		return
	}
	root := os.Getenv("VROOLI_SNAPSHOT_WINDOWS_ROOT")
	snapshot := os.Getenv("VROOLI_SNAPSHOT_WINDOWS_SNAPSHOT")
	// Make the command child run the worker mode. Keeping "manager" in the
	// inherited environment would recursively launch another helper.
	if err := os.Setenv(snapshotLeaseWindowsHelperEnv, "worker"); err != nil {
		t.Fatal(err)
	}
	args := []string{"vrooli-atomic-install"}
	if mode == "root-manager" {
		args = append(args, "--snapshot-lock-exec", root, "--")
	} else {
		args = append(args, "--snapshot-lease-exec", root, snapshot, "--")
	}
	args = append(args, os.Args[0], "-test.run=^TestWindowsSnapshotLeaseCommandHelper$")
	os.Exit(run(args, os.Stderr))
}

func TestWindowsSnapshotLeaseCommandHelper(t *testing.T) {
	if os.Getenv(snapshotLeaseWindowsHelperEnv) != "worker" {
		return
	}
	ready := os.Getenv("VROOLI_SNAPSHOT_WINDOWS_READY")
	child := exec.Command(os.Args[0], "-test.run=^TestWindowsSnapshotLeaseDescendantHelper$")
	child.Env = envWithout(os.Environ(), snapshotLeaseWindowsHelperEnv)
	child.Env = append(child.Env, snapshotLeaseWindowsHelperEnv+"=descendant", "VROOLI_SNAPSHOT_WINDOWS_READY="+ready)
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(ready); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = child.Process.Kill()
	t.Fatal("descendant did not become active before timeout")
}

func TestWindowsSnapshotLeaseDescendantHelper(t *testing.T) {
	if os.Getenv(snapshotLeaseWindowsHelperEnv) != "descendant" {
		return
	}
	if err := os.WriteFile(os.Getenv("VROOLI_SNAPSHOT_WINDOWS_READY"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	for {
		time.Sleep(time.Hour)
	}
}

func TestWindowsAbruptLeaseHelperExitBlocksUntilNamedJobDrains(t *testing.T) {
	root := t.TempDir()
	snapshot := filepath.Join(root, "rebaseline-hub.abrupt01")
	if err := os.Mkdir(snapshot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapshot, ".lease"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(root, "descendant.ready")
	manager := exec.Command(os.Args[0], "-test.run=^TestWindowsSnapshotLeaseManagerHelper$")
	manager.Env = envWithout(os.Environ(), snapshotLeaseWindowsHelperEnv)
	manager.Env = append(manager.Env,
		snapshotLeaseWindowsHelperEnv+"=manager",
		"VROOLI_SNAPSHOT_WINDOWS_ROOT="+root,
		"VROOLI_SNAPSHOT_WINDOWS_SNAPSHOT="+snapshot,
		"VROOLI_SNAPSHOT_WINDOWS_READY="+ready,
	)
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	managerDone := false
	defer func() {
		if !managerDone {
			_ = manager.Process.Kill()
			_ = manager.Wait()
		}
	}()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(ready); err != nil {
		t.Fatalf("descendant readiness marker missing: %v", err)
	}
	name, err := cliinstall.SnapshotLeaseJobName(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	jobName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	job, err := openTestJobByName(jobName)
	if err != nil {
		t.Fatal(err)
	}
	var active snapshotJobAccountingForTest
	if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&active)), uint32(unsafe.Sizeof(active)), nil); err != nil {
		_ = windows.CloseHandle(job)
		t.Fatal(err)
	}
	if active.ActiveProcesses < 1 {
		_ = windows.CloseHandle(job)
		t.Fatal("named job has no active descendants before manager kill")
	}
	if err := manager.Process.Kill(); err != nil {
		_ = windows.CloseHandle(job)
		t.Fatal(err)
	}
	_ = manager.Wait()
	managerDone = true
	// The test's probe handle keeps kill-on-last-close pending so this assertion
	// deterministically exercises the admission witness while a member remains.
	acquired, err := cliinstall.TryWithSnapshotLease(root, snapshot, func() error { return nil })
	if err != nil {
		_ = windows.CloseHandle(job)
		t.Fatal(err)
	}
	if acquired {
		_ = windows.CloseHandle(job)
		t.Fatal("lease admitted while a member of its named job was alive")
	}
	if err := cliinstall.PruneHubSnapshots(root, "", time.Now().Add(48*time.Hour), nil); err != nil {
		_ = windows.CloseHandle(job)
		t.Fatalf("prune while named lease job was active: %v", err)
	}
	if _, err := os.Stat(snapshot); err != nil {
		_ = windows.CloseHandle(job)
		t.Fatalf("active snapshot was pruned before named job drained: %v", err)
	}
	if err := windows.CloseHandle(job); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		acquired, err = cliinstall.TryWithSnapshotLease(root, snapshot, func() error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		if acquired {
			if err := cliinstall.PruneHubSnapshots(root, "", time.Now().Add(48*time.Hour), nil); err != nil {
				t.Fatalf("prune after named lease job drained: %v", err)
			}
			if _, err := os.Stat(snapshot); !os.IsNotExist(err) {
				t.Fatalf("drained incomplete snapshot still exists after pruning: %v", err)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("lease remained blocked after the final named-job handle closed")
}

func TestWindowsAbruptRootHelperExitBlocksRootLockUntilNamedJobDrains(t *testing.T) {
	root := t.TempDir()
	ready := filepath.Join(root, "root-descendant.ready")
	manager := exec.Command(os.Args[0], "-test.run=^TestWindowsSnapshotLeaseManagerHelper$")
	manager.Env = envWithout(os.Environ(), snapshotLeaseWindowsHelperEnv)
	manager.Env = append(manager.Env,
		snapshotLeaseWindowsHelperEnv+"=root-manager",
		"VROOLI_SNAPSHOT_WINDOWS_ROOT="+root,
		"VROOLI_SNAPSHOT_WINDOWS_READY="+ready,
	)
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	managerDone := false
	defer func() {
		if !managerDone {
			_ = manager.Process.Kill()
			_ = manager.Wait()
		}
	}()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(ready); err != nil {
		t.Fatalf("root helper descendant readiness marker missing: %v", err)
	}
	name, err := cliinstall.SnapshotRootJobName(root)
	if err != nil {
		t.Fatal(err)
	}
	jobName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	job, err := openTestJobByName(jobName)
	if err != nil {
		t.Fatal(err)
	}
	var active snapshotJobAccountingForTest
	if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&active)), uint32(unsafe.Sizeof(active)), nil); err != nil {
		_ = windows.CloseHandle(job)
		t.Fatal(err)
	}
	if active.ActiveProcesses < 1 {
		_ = windows.CloseHandle(job)
		t.Fatal("named root job has no active descendants before manager kill")
	}
	if err := manager.Process.Kill(); err != nil {
		_ = windows.CloseHandle(job)
		t.Fatal(err)
	}
	_ = manager.Wait()
	managerDone = true

	entered := make(chan struct{})
	lockDone := make(chan error, 1)
	go func() {
		lockDone <- cliinstall.WithSnapshotLock(root, func() error {
			close(entered)
			return nil
		})
	}()
	select {
	case <-entered:
		_ = windows.CloseHandle(job)
		t.Fatal("root lock admitted while a member of its named job was alive")
	case <-time.After(150 * time.Millisecond):
	}
	if err := windows.CloseHandle(job); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-lockDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("root lock remained blocked after final named-job handle closed")
	}
}

func openTestJobByName(name *uint16) (windows.Handle, error) {
	const jobObjectQuery = 0x0004
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenJobObjectW")
	handle, _, callErr := proc.Call(jobObjectQuery, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		if callErr == windows.ERROR_SUCCESS {
			callErr = windows.ERROR_INVALID_HANDLE
		}
		return 0, callErr
	}
	return windows.Handle(handle), nil
}

func envWithout(env []string, key string) []string {
	prefix := strings.ToUpper(key) + "="
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(strings.ToUpper(entry), prefix) {
			out = append(out, entry)
		}
	}
	return out
}

type snapshotJobAccountingForTest struct {
	TotalUserTime, TotalKernelTime                     int64
	ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaults, TotalProcesses, ActiveProcesses   uint32
	TotalTerminatedProcesses                           uint32
}
