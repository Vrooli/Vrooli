//go:build linux

package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This fixture is intentionally opt-in. It exercises the current host's real
// user manager once, never runs during ordinary package tests, and owns only
// the uniquely named service created from its persisted intent.
func TestOwnedOperationCurrentHostTreeProbe(t *testing.T) {
	if os.Getenv("PLATFORM_GO_OPERATION_PROBE") != "1" {
		t.Skip("set PLATFORM_GO_OPERATION_PROBE=1 for the bounded current-host owner probe")
	}
	work := t.TempDir()
	pidFile := filepath.Join(work, "children.txt")
	spec := OperationSpec{
		Path: "/bin/sh",
		Args: []string{"-c", "/bin/sleep 60 & p1=$!; /bin/sh -c '/bin/sleep 60' & p2=$!; printf '%s %s\\n' \"$p1\" \"$p2\" > \"$1\"; exit 0", "device-studio-owner-probe", pidFile},
		Env:  []string{}, Dir: work,
		Limits: OperationLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 100, WallTime: 30 * time.Second},
	}
	intent, err := NewOperationIntent("device-studio", "host-probe-"+strconv.FormatInt(time.Now().UnixNano(), 10), spec)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	op, err := StartOwnedOperation(ctx, intent, spec)
	if err != nil {
		t.Fatalf("start owned probe: %v", err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cleanupCancel()
			_, _ = StopOwnedOperation(cleanupCtx, op.Ref)
		}
	})

	var pids []int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, readErr := os.ReadFile(pidFile)
		if readErr == nil {
			for _, field := range strings.Fields(string(data)) {
				pid, parseErr := strconv.Atoi(field)
				if parseErr != nil {
					t.Fatalf("parse fixture pid: %v", parseErr)
				}
				pids = append(pids, pid)
			}
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if len(pids) != 2 {
		t.Fatalf("fixture did not persist its child identities: %v", pids)
	}

	// The service main shell exits while its direct and nested descendants live.
	var observation OperationObservation
	for time.Now().Before(deadline.Add(3 * time.Second)) {
		observation, err = InspectOwnedOperation(ctx, op.Ref)
		if err == nil && observation.Populated {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("inspect surviving descendants: %v", err)
	}
	if !observation.Populated {
		t.Fatalf("manager lost recursively populated service after main exit: %+v", observation)
	}
	if _, err := LookupOwnedOperation(ctx, intent); err != nil {
		t.Fatalf("restart lookup did not recover the exact active attempt: %v", err)
	}
	if _, err := StartOwnedOperation(ctx, intent, spec); err == nil {
		t.Fatal("duplicate persisted attempt started a second service")
	}

	receipt, err := StopOwnedOperation(ctx, op.Ref)
	if err != nil {
		t.Fatalf("stop owned parent+descendant tree: %v", err)
	}
	stopped = true
	if !receipt.NativeQuiescent || receipt.LauncherReap != LauncherReapUnknown {
		t.Fatalf("stop receipt omitted verified lifecycle facts: %+v", receipt)
	}
	localReceipt, err := op.Stop(ctx)
	if err != nil {
		t.Fatalf("local owner did not observe terminal state: %v", err)
	}
	if localReceipt.LauncherReap != LauncherReaped {
		t.Fatalf("local launcher reap was not reported separately: %+v", localReceipt)
	}
	for _, pid := range pids {
		if _, alive, inspectErr := linuxProcessStartTime(pid); inspectErr != nil || alive {
			t.Fatalf("owned descendant %d remains alive after verified stop (alive=%t err=%v)", pid, alive, inspectErr)
		}
	}
	post, err := InspectOwnedOperation(ctx, op.Ref)
	if err != nil {
		t.Fatalf("inspect retained terminal manager result: %v", err)
	}
	if post.Populated {
		t.Fatalf("service still has processes after stop: %+v", post)
	}
	fmt.Printf("operation probe unit=%s native_invocation=%s children=%v exit_known=%t result=%s quiescent=%t\n", op.Ref.Unit, op.Ref.NativeInvocationID, pids, post.ExitKnown, post.Result, receipt.NativeQuiescent)
	if err := ReleaseOwnedOperation(ctx, op.Ref); err != nil {
		t.Fatalf("release caller-settled terminal service: %v", err)
	}
}

// This opt-in current-host fixture covers both short success and exec failure,
// which can complete before the API receives its first manager query.
func TestOwnedOperationCurrentHostTerminalProbe(t *testing.T) {
	if os.Getenv("PLATFORM_GO_OPERATION_PROBE") != "1" {
		t.Skip("set PLATFORM_GO_OPERATION_PROBE=1 for bounded current-host terminal probes")
	}
	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{name: "fast-success", path: "/usr/bin/true", want: "success"},
		{name: "failed-exec", path: "/nonexistent/vrooli-operation-probe", want: "failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			work := t.TempDir()
			spec := OperationSpec{
				Path: tc.path, Args: []string{}, Env: []string{}, Dir: work,
				Limits: OperationLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 100, WallTime: 10 * time.Second},
			}
			intent, err := NewOperationIntent("device-studio", "terminal-probe-"+tc.name+"-"+strconv.FormatInt(time.Now().UnixNano(), 10), spec)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			op, err := StartOwnedOperation(ctx, intent, spec)
			if err != nil {
				// Dispatch may have won the race; recover only by this persisted
				// deterministic intent, then clean up only that exact reference.
				if ref, lookupErr := LookupOwnedOperation(ctx, intent); lookupErr == nil {
					_, _ = StopOwnedOperation(ctx, ref)
					_ = ReleaseOwnedOperation(ctx, ref)
				}
				t.Fatalf("start terminal probe: %v", err)
			}
			released := false
			t.Cleanup(func() {
				if !released {
					_, _ = StopOwnedOperation(context.Background(), op.Ref)
					_ = ReleaseOwnedOperation(context.Background(), op.Ref)
				}
			})
			observation, err := InspectOwnedOperation(ctx, op.Ref)
			if err != nil {
				t.Fatalf("inspect terminal result: %v", err)
			}
			if !observation.ExitKnown || observation.Populated {
				t.Fatalf("short operation lost terminal/quiescence evidence: %+v", observation)
			}
			if tc.want == "success" && (observation.Result != "success" || observation.ExecMainStatus != 0) {
				t.Fatalf("fast success was not retained: %+v", observation)
			}
			if tc.want == "failure" && (observation.Result == "success" || observation.ExecMainStatus == 0) {
				t.Fatalf("failed exec was reported as success: %+v", observation)
			}
			if err := ReleaseOwnedOperation(ctx, op.Ref); err != nil {
				t.Fatalf("release exact terminal probe service: %v", err)
			}
			released = true
			fmt.Printf("operation terminal probe case=%s unit=%s invocation=%s result=%s exec_status=%d quiescent=%t\n", tc.name, op.Ref.Unit, op.Ref.NativeInvocationID, observation.Result, observation.ExecMainStatus, !observation.Populated)
		})
	}
}
