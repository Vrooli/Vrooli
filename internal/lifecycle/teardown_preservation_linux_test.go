package lifecycle

import (
	"context"
	"fmt"
	"github.com/vrooli/vrooli/internal/process"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestAgentManagerRestartPreservesDetachedExecutorTree(t *testing.T) {
	for _, tc := range []struct {
		name, scenario, step          string
		immediate, detached, preserve bool
	}{
		{"legacy-api", "agent-manager", "start-api", true, true, true},
		{"registry-api", "agent-manager", "start-api", true, true, true},
		{"same-group-helper", "agent-manager", "start-api", true, false, false},
		{"normal-stop", "agent-manager", "start-api", false, true, false},
		{"ui-browser", "agent-manager", "start-ui", true, true, false},
		{"other-api", "browser-automation-studio", "start-api", true, true, false},
		{"missing-scenario", "", "start-api", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := "sh -c 'sleep 30 & wait' & wait"
			if tc.detached {
				command = "setsid sh -c 'sleep 30 & wait' & wait"
			}
			owner := exec.Command("sh", "-c", command)
			owner.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := owner.Start(); err != nil {
				t.Fatal(err)
			}
			var childPID, grandchild int
			defer func() {
				if grandchild > 0 {
					_ = syscall.Kill(grandchild, syscall.SIGTERM)
				}
				if childPID > 0 {
					_ = syscall.Kill(childPID, syscall.SIGTERM)
				}
				_ = owner.Process.Kill()
				_ = owner.Wait()
			}()
			oneChild := func(pid int) int {
				raw, _ := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", pid, pid))
				fields := strings.Fields(string(raw))
				if len(fields) != 1 {
					return 0
				}
				result, _ := strconv.Atoi(fields[0])
				return result
			}
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				childPID = oneChild(owner.Process.Pid)
				if childPID > 0 {
					grandchild = oneChild(childPID)
				}
				if grandchild > 0 {
					break
				}
				time.Sleep(time.Millisecond)
			}
			if grandchild == 0 {
				t.Fatal("fixture descendant tree did not start")
			}
			pgid, err := syscall.Getpgid(owner.Process.Pid)
			if err != nil {
				t.Fatal(err)
			}
			childGroup, err := syscall.Getpgid(childPID)
			if err != nil {
				t.Fatal(err)
			}
			record := process.Record{PID: owner.Process.Pid, PGID: pgid, Scenario: tc.scenario, Step: tc.step, ProcessID: "vrooli.develop.agent-manager.start-api"}
			if tc.name == "legacy-api" {
				record.PortKey = "API_PORT"
			}
			signaled := map[int]bool{}
			observed := map[int]bool{}
			deps := defaultLifecycleDeps()
			deps.signalProcessGroup = func(group int, force bool) error { signaled[group] = true; return nil }
			deps.isPIDRunning = func(pid int) bool { observed[pid] = true; return false }
			r := &Runner{deps: deps}
			if err := r.terminateAndAwait(context.Background(), []process.Record{record}, tc.immediate); err != nil {
				t.Fatal(err)
			}
			if !tc.immediate {
				// Non-immediate already-exited fixture roots need no group signal, but
				// descendant observation still establishes its cleanup target scope.
				if !observed[childPID] {
					t.Fatal("ordinary stop lost descendant scope")
				}
				return
			}
			if !signaled[pgid] {
				t.Fatal("tracked owner group was not terminated")
			}
			if tc.preserve {
				if signaled[childGroup] || observed[childPID] || observed[grandchild] {
					t.Fatal("detached admitted executor was signaled or awaited")
				}
				if err := syscall.Kill(childPID, syscall.Signal(0)); err != nil {
					t.Fatal("fixture executor did not survive", err)
				}
			} else if !signaled[childGroup] || !observed[childPID] {
				t.Fatal("ordinary descendant cleanup lost")
			}
		})
	}
}

func TestAgentManagerRestartRetainsTrackedOwnerEscalation(t *testing.T) {
	now := time.Unix(100, 0)
	forced := false
	deps := defaultLifecycleDeps()
	deps.now = func() time.Time { return now }
	deps.sleep = func(d time.Duration) { now = now.Add(d) }
	deps.isPIDRunning = func(int) bool { return !forced }
	deps.signalProcessGroup = func(group int, force bool) error {
		if group != 900000001 {
			t.Fatal("unexpected group", group)
		}
		if force {
			forced = true
		}
		return nil
	}
	r := &Runner{deps: deps}
	if err := r.terminateAndAwait(context.Background(), []process.Record{{PID: 900000001, PGID: 900000001, Scenario: "agent-manager", Step: "start-api"}}, true); err != nil {
		t.Fatal(err)
	}
	if !forced {
		t.Fatal("tracked owner escalation lost")
	}
}
