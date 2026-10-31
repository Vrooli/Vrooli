//go:build linux

package target

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestKillElectronCommandKillsDescendantsInSeparateProcessGroups(t *testing.T) {
	childFile := t.TempDir() + "/child.pid"
	command := fmt.Sprintf("setsid sh -c 'sleep 30' & child=$!; printf '%%d' \"$child\" > %q; wait", childFile)
	cmd := exec.Command("sh", "-c", command)
	configureElectronCommand(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = killElectronCommand(cmd)
		_ = cmd.Wait()
	}()

	var childPID int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(childFile)
		if err == nil {
			childPID, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if childPID == 0 {
		t.Fatal("detached child PID was not recorded")
	}

	if err := killElectronCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "signal: killed") {
		t.Fatalf("launcher wait: %v", err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", childPID)); os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("detached child %d survived launcher cleanup", childPID)
}

func TestDescendantsOfOrdersDeepestFirst(t *testing.T) {
	parents := map[int]int{
		10: 1,
		11: 10,
		12: 11,
		13: 10,
		14: 99,
	}
	got := descendantsOf(10, parents)
	if len(got) != 3 || got[0] != 12 {
		t.Fatalf("descendantsOf() = %v, want deepest descendant first", got)
	}
}
