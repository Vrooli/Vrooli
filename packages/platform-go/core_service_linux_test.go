//go:build linux

package platform

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCoreSupervisorStopFenceRequiresNativeAndKernelAbsence(t *testing.T) {
	oldQuery, oldParent, oldRoot := queryCoreSupervisor, queryCoreSupervisorParent, operationCgroupRoot
	t.Cleanup(func() {
		queryCoreSupervisor, queryCoreSupervisorParent, operationCgroupRoot = oldQuery, oldParent, oldRoot
	})
	operationCgroupRoot = t.TempDir()
	unit, _ := CoreUnitByID(CoreUnitRuntimeSupervisor)
	state := CoreServiceState{Unit: unit.Systemd, LoadState: "loaded", ActiveState: "inactive", SubState: "dead", InvocationID: strings.Repeat("a", 32)}
	queryCoreSupervisor = func(context.Context) (CoreServiceState, error) { return state, nil }
	queryCoreSupervisorParent = func(context.Context) (string, error) { return "/user.slice/fixture/app.slice", nil }
	fence, err := PinCoreSupervisor(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	defer fence.Close()
	if err := fence.Stopped(context.Background()); err != nil {
		t.Fatal(err)
	}
	state.PendingJob = true
	if err := fence.Stopped(context.Background()); err == nil {
		t.Fatal("pending native job was treated as stopped")
	}
	state.PendingJob = false
	state.PID = 1
	if err := fence.Stopped(context.Background()); err == nil {
		t.Fatal("live native root was treated as stopped")
	}
	state.PID = 0
	path := filepath.Join(operationCgroupRoot, "user.slice", "fixture", "app.slice", unit.Systemd)
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fence.Stopped(context.Background()); err == nil {
		t.Fatal("late/reused native group was treated as absent")
	}
}

func TestCoreSupervisorFenceRefusesChangedNativeObservation(t *testing.T) {
	old := queryCoreSupervisor
	t.Cleanup(func() { queryCoreSupervisor = old })
	unit, _ := CoreUnitByID(CoreUnitRuntimeSupervisor)
	want := CoreServiceState{Unit: unit.Systemd, LoadState: "loaded", PID: 1, InvocationID: strings.Repeat("a", 32)}
	queryCoreSupervisor = func(context.Context) (CoreServiceState, error) {
		different := want
		different.PID = 2
		return different, nil
	}
	if fence, err := PinCoreSupervisor(context.Background(), want); err == nil {
		fence.Close()
		t.Fatal("changed native owner was pinned")
	}
}

func TestCoreSupervisorCurrentHostReadOnlyFenceProbe(t *testing.T) {
	if os.Getenv("VROOLI_CORE_SUPERVISOR_OBSERVE_PROBE") != "1" {
		t.Skip("set VROOLI_CORE_SUPERVISOR_OBSERVE_PROBE=1 for actual native observation and read-only pin")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	state, err := InspectCoreSupervisor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state.ActiveState != "active" || state.PID <= 0 || state.KillMode != "control-group" {
		t.Fatalf("unexpected native root state: %+v", state)
	}
	fence, err := PinCoreSupervisor(ctx, state)
	if err != nil {
		t.Fatal(err)
	}
	defer fence.Close()
	if err := fence.Stopped(ctx); err == nil {
		t.Fatal("live root was reported quiescent")
	}
	t.Logf("native root PID=%d invocation=%s exact cgroup pinned read-only; live tree refused quiescence", state.PID, state.InvocationID)
}
