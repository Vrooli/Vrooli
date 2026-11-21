//go:build darwin || linux

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSnapshotLockSignalHelper(t *testing.T) {
	mode := os.Getenv("VROOLI_SNAPSHOT_SIGNAL_HELPER")
	if mode == "" {
		return
	}
	root := os.Getenv("VROOLI_SNAPSHOT_SIGNAL_ROOT")
	var args []string
	switch mode {
	case "hold":
		args = []string{"installer", "--snapshot-lock-exec", root, "--", "sh", "-c", `set -eu; trap '' TERM; touch "$1"; (sleep 3; touch "$2") & wait`, "sh", os.Getenv("VROOLI_SNAPSHOT_SIGNAL_STARTED"), os.Getenv("VROOLI_SNAPSHOT_SIGNAL_DONE")}
	case "contend":
		args = []string{"installer", "--snapshot-lock-exec", root, "--", "sh", "-c", `touch "$1"`, "sh", os.Getenv("VROOLI_SNAPSHOT_SIGNAL_ACQUIRED")}
	case "lease-contend":
		args = []string{"installer", "--snapshot-lease-exec", root, os.Getenv("VROOLI_SNAPSHOT_SIGNAL_SNAPSHOT"), "--", "sh", "-c", `touch "$1"`, "sh", os.Getenv("VROOLI_SNAPSHOT_SIGNAL_ACQUIRED")}
	default:
		t.Fatalf("unknown helper mode %q", mode)
	}
	os.Exit(run(args, os.Stderr))
}

func TestSnapshotEscapedManagerHelper(t *testing.T) {
	if os.Getenv("VROOLI_SNAPSHOT_ESCAPED_MANAGER") == "" {
		return
	}
	root := os.Getenv("VROOLI_SNAPSHOT_SIGNAL_ROOT")
	args := []string{"installer", "--snapshot-lease-exec", root, os.Getenv("VROOLI_SNAPSHOT_SIGNAL_SNAPSHOT"), "--", os.Args[0], "-test.run=^TestSnapshotEscapedDescendantHelper$"}
	os.Exit(run(args, os.Stderr))
}

func TestSnapshotEscapedDescendantHelper(t *testing.T) {
	if os.Getenv("VROOLI_SNAPSHOT_ESCAPED_DESCENDANT") == "" {
		return
	}
	lock := os.NewFile(3, "inherited-snapshot-lock")
	if lock == nil {
		t.Fatal("lock descriptor 3 was not inherited")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSnapshotDetachedSleeperHelper$")
	cmd.Env = append(os.Environ(), "VROOLI_SNAPSHOT_DETACHED_DONE="+os.Getenv("VROOLI_SNAPSHOT_SIGNAL_DONE"))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.ExtraFiles = []*os.File{lock}
	output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		_ = output.Close()
		t.Fatal(err)
	}
	_ = output.Close()
	if err := os.WriteFile(os.Getenv("VROOLI_SNAPSHOT_SIGNAL_STARTED"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	defer signal.Stop(signals)
	<-signals
}

func TestSnapshotDetachedSleeperHelper(t *testing.T) {
	path := os.Getenv("VROOLI_SNAPSHOT_DETACHED_DONE")
	if path == "" {
		return
	}
	time.Sleep(3 * time.Second)
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotLockRemainsHeldUntilSignaledChildGroupExits(t *testing.T) {
	root := t.TempDir()
	started := filepath.Join(root, "started")
	done := filepath.Join(root, "done")
	acquired := filepath.Join(root, "acquired")
	startHelper := func(mode string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSnapshotLockSignalHelper$")
		cmd.Env = append(os.Environ(), "VROOLI_SNAPSHOT_SIGNAL_HELPER="+mode, "VROOLI_SNAPSHOT_SIGNAL_ROOT="+root, "VROOLI_SNAPSHOT_SIGNAL_STARTED="+started, "VROOLI_SNAPSHOT_SIGNAL_DONE="+done, "VROOLI_SNAPSHOT_SIGNAL_ACQUIRED="+acquired)
		output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Stdout, cmd.Stderr = output, output
		if err := cmd.Start(); err != nil {
			_ = output.Close()
			t.Fatal(err)
		}
		_ = output.Close()
		return cmd
	}
	holder := startHelper("hold")
	defer func() {
		if holder.ProcessState == nil {
			_ = holder.Process.Kill()
			_ = holder.Wait()
		}
	}()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(started); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(started); err != nil {
		t.Fatalf("child did not start: %v", err)
	}
	if err := holder.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	contender := startHelper("contend")
	time.Sleep(150 * time.Millisecond)
	if _, err := os.Stat(acquired); err == nil {
		t.Fatalf("lock was released before signaled child finished; done=%t", fileExists(done))
	}
	if err := holder.Wait(); err == nil {
		t.Fatal("holder returned success after receiving SIGTERM")
	}
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("child process group did not finish: %v", err)
	}
	if err := contender.Wait(); err != nil {
		t.Fatalf("contender: %v", err)
	}
	if _, err := os.Stat(acquired); err != nil {
		t.Fatalf("contender did not acquire after child group exited: %v", err)
	}
}

func TestSnapshotLockSurvivesManagerSIGKILLWhileChildOwnsDescriptor(t *testing.T) {
	root := t.TempDir()
	started := filepath.Join(root, "started")
	done := filepath.Join(root, "done")
	acquired := filepath.Join(root, "acquired")
	startHelper := func(mode string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSnapshotLockSignalHelper$")
		cmd.Env = append(os.Environ(), "VROOLI_SNAPSHOT_SIGNAL_HELPER="+mode, "VROOLI_SNAPSHOT_SIGNAL_ROOT="+root, "VROOLI_SNAPSHOT_SIGNAL_STARTED="+started, "VROOLI_SNAPSHOT_SIGNAL_DONE="+done, "VROOLI_SNAPSHOT_SIGNAL_ACQUIRED="+acquired)
		output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Stdout, cmd.Stderr = output, output
		if err := cmd.Start(); err != nil {
			_ = output.Close()
			t.Fatal(err)
		}
		_ = output.Close()
		return cmd
	}
	holder := startHelper("hold")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(started); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(started); err != nil {
		t.Fatal("child did not start")
	}
	startedAt := time.Now()
	if err := holder.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := holder.Wait(); err == nil {
		t.Fatal("manager unexpectedly survived SIGKILL")
	}
	contender := startHelper("contend")
	defer func() {
		if contender.ProcessState == nil {
			_ = contender.Process.Kill()
			_ = contender.Wait()
		}
	}()
	time.Sleep(150 * time.Millisecond)
	if _, err := os.Stat(acquired); err == nil {
		t.Fatalf("child lock descriptor did not survive manager SIGKILL; elapsed=%s done=%t", time.Since(startedAt), fileExists(done))
	}
	deadline = time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) && !fileExists(done) {
		time.Sleep(10 * time.Millisecond)
	}
	if !fileExists(done) {
		t.Fatal("child did not finish after manager was killed")
	}
	if err := contender.Wait(); err != nil {
		t.Fatalf("contender: %v", err)
	}
	if _, err := os.Stat(acquired); err != nil {
		t.Fatalf("contender did not acquire after inherited lock closed: %v", err)
	}
}

func TestSnapshotLeaseSurvivesEscapedDescendantAfterManagerTerm(t *testing.T) {
	root := t.TempDir()
	snapshot := filepath.Join(root, "rebaseline-hub.snapshot01")
	if err := os.Mkdir(snapshot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapshot, ".lease"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	started := filepath.Join(root, "escaped.started")
	done := filepath.Join(root, "escaped.done")
	acquired := filepath.Join(root, "escaped.acquired")
	manager := exec.Command(os.Args[0], "-test.run=^TestSnapshotEscapedManagerHelper$")
	manager.Env = append(os.Environ(), "VROOLI_SNAPSHOT_ESCAPED_MANAGER=1", "VROOLI_SNAPSHOT_ESCAPED_DESCENDANT=1", "VROOLI_SNAPSHOT_SIGNAL_ROOT="+root, "VROOLI_SNAPSHOT_SIGNAL_SNAPSHOT="+snapshot, "VROOLI_SNAPSHOT_SIGNAL_STARTED="+started, "VROOLI_SNAPSHOT_SIGNAL_DONE="+done)
	manager.Stdout, manager.Stderr = os.Stdout, os.Stderr
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !fileExists(started) {
		time.Sleep(10 * time.Millisecond)
	}
	if !fileExists(started) {
		_ = manager.Process.Kill()
		_ = manager.Wait()
		t.Fatal("escaped child did not start")
	}
	if err := manager.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- manager.Wait() }()
	select {
	case err := <-waited:
		if err == nil {
			t.Fatal("manager returned success after SIGTERM")
		}
	case <-time.After(time.Second):
		t.Fatal("manager waited for a descendant outside its process group")
	}
	contender := exec.Command(os.Args[0], "-test.run=^TestSnapshotLockSignalHelper$")
	contender.Env = append(os.Environ(), "VROOLI_SNAPSHOT_SIGNAL_HELPER=lease-contend", "VROOLI_SNAPSHOT_SIGNAL_ROOT="+root, "VROOLI_SNAPSHOT_SIGNAL_SNAPSHOT="+snapshot, "VROOLI_SNAPSHOT_SIGNAL_ACQUIRED="+acquired)
	output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	contender.Stdout, contender.Stderr = output, output
	if err := contender.Start(); err != nil {
		_ = output.Close()
		t.Fatal(err)
	}
	_ = output.Close()
	defer func() {
		if contender.ProcessState == nil {
			_ = contender.Process.Kill()
			_ = contender.Wait()
		}
	}()
	time.Sleep(150 * time.Millisecond)
	if fileExists(acquired) {
		t.Fatal("snapshot lease was explicitly unlocked while an escaped descendant still held its inherited descriptor")
	}
	deadline = time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) && !fileExists(done) {
		time.Sleep(10 * time.Millisecond)
	}
	if !fileExists(done) {
		t.Fatal("escaped descendant did not finish")
	}
	if err := contender.Wait(); err != nil {
		t.Fatal(err)
	}
	if !fileExists(acquired) {
		t.Fatal("contender did not acquire after escaped descendant closed its descriptor")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
