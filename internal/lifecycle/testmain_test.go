package lifecycle

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestMain makes process cleanup an explicit package invariant. Lifecycle
// tests intentionally start detached scenario processes, so a green test can
// otherwise leave a listener behind and poison the next invocation.
func TestMain(m *testing.M) {
	before, beforeErr := lifecycleTestDescendants()
	status := m.Run()
	after, afterErr := waitForLifecycleTestDescendants()

	if beforeErr == nil && afterErr == nil {
		var leaked []int
		for pid := range after {
			if _, existed := before[pid]; !existed {
				leaked = append(leaked, pid)
			}
		}
		if len(leaked) > 0 {
			fmt.Fprintf(os.Stderr, "lifecycle test leaked child processes: %v\n", leaked)
			args := []string{"-o", "pid=,ppid=,stat=,args=", "-p"}
			pidArgs := make([]string, 0, len(leaked))
			for _, pid := range leaked {
				pidArgs = append(pidArgs, strconv.Itoa(pid))
			}
			args = append(args, strings.Join(pidArgs, ","))
			if details, err := exec.Command("ps", args...).CombinedOutput(); err == nil {
				fmt.Fprintln(os.Stderr, strings.TrimSpace(string(details)))
			}
			if status == 0 {
				status = 1
			}
		}
	}
	if beforeErr != nil || afterErr != nil {
		fmt.Fprintf(os.Stderr, "lifecycle child-process assertion unavailable: before=%v after=%v\n", beforeErr, afterErr)
	}
	os.Exit(status)
}

func waitForLifecycleTestDescendants() (map[int]struct{}, error) {
	current, err := lifecycleTestDescendants()
	if err != nil || len(current) == 0 {
		return current, err
	}
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			return current, nil
		case <-ticker.C:
			current, err = lifecycleTestDescendants()
			if err != nil || len(current) == 0 {
				return current, err
			}
		}
	}
}

// lifecycleTestDescendants returns all descendants of the test binary. ps is
// used instead of a Linux-only /proc walk so the assertion remains buildable
// on every supported Unix host; Windows has no portable parent-PID query and
// is covered by the native containment tests in packages/platform-go.
func lifecycleTestDescendants() (map[int]struct{}, error) {
	if runtime.GOOS == "windows" {
		return map[int]struct{}{}, nil
	}
	output, err := exec.Command("ps", "-eo", "pid=,ppid=,stat=,args=").Output()
	if err != nil {
		return nil, err
	}
	children := make(map[int][]int)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		pid, pidErr := strconv.Atoi(fields[0])
		parent, parentErr := strconv.Atoi(fields[1])
		if pidErr != nil || parentErr != nil {
			continue
		}
		// Detached lifecycle commands can briefly remain as zombies until the
		// test process exits. They no longer own resources and are not live
		// children, so do not turn that kernel bookkeeping window into a leak
		// failure.
		if strings.HasPrefix(fields[2], "Z") {
			continue
		}
		if strings.HasPrefix(strings.Join(fields[3:], " "), "ps ") {
			continue
		}
		children[parent] = append(children[parent], pid)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	root := os.Getpid()
	descendants := make(map[int]struct{})
	queue := append([]int(nil), children[root]...)
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if _, seen := descendants[pid]; seen {
			continue
		}
		descendants[pid] = struct{}{}
		queue = append(queue, children[pid]...)
	}
	return descendants, nil
}
