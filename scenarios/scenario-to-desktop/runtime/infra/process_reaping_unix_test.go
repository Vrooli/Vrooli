//go:build unix || !windows

package infra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestRealProcessStopReapsProcessGroupChildren(t *testing.T) {
	childFile := filepath.Join(t.TempDir(), "child.pid")
	runner := RealProcessRunner{}
	command := fmt.Sprintf("sleep 30 & child=$!; trap 'kill \"$child\" 2>/dev/null; wait \"$child\" 2>/dev/null; exit 143' TERM; printf '%%s' \"$child\" > %q; wait \"$child\"", childFile)
	process, err := runner.Start(context.Background(), "sh", []string{"-c", command}, nil, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	var childPID int
	for childPID == 0 && time.Now().Before(deadline) {
		data, readErr := os.ReadFile(childFile)
		if readErr == nil {
			childPID, _ = strconv.Atoi(string(data))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if childPID == 0 {
		_ = process.Kill()
		_ = process.Wait()
		t.Fatal("child process did not publish its pid")
	}
	if err := StopProcess(process); err != nil {
		t.Fatalf("stop process group: %v", err)
	}
	if err := process.Wait(); err != nil {
		t.Logf("parent exited after group stop: %v", err)
	}
	if err := syscall.Kill(childPID, 0); err == nil || err != syscall.ESRCH {
		t.Fatalf("child process %d still exists after parent stop: %v", childPID, err)
	}
}
