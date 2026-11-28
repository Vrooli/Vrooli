//go:build linux

package privilegebroker

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// scriptedExecutor answers each call by index so a test can express the exact
// success/failure shape of a multi-step adapter without a shell.
type scriptedExecutor struct {
	calls   [][]string
	results []error
}

func (e *scriptedExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	e.calls = append(e.calls, append([]string{name}, args...))
	if len(e.results) == 0 {
		return nil, nil
	}
	err := e.results[0]
	e.results = e.results[1:]
	return nil, err
}

func TestSystemRebootImmediateUsesFixedArgv(t *testing.T) {
	exec := &scriptedExecutor{}
	result := executeSystemReboot(context.Background(), exec, Request{
		Version: ProtocolVersion, RequestID: "r1", Action: ActionSystemReboot,
		Reboot: &RebootSubject{DecisionID: "decision-42"},
	})
	want := [][]string{{"/usr/bin/systemctl", "reboot", "--no-block"}}
	if !reflect.DeepEqual(exec.calls, want) {
		t.Fatalf("argv = %v, want %v", exec.calls, want)
	}
	if result.Status != "changed" || !result.Changed {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(result.Evidence.Detail, "decision-42") {
		t.Fatalf("decision reference missing from evidence: %q", result.Evidence.Detail)
	}
}

func TestSystemRebootSchedulesOneNamedTimer(t *testing.T) {
	exec := &scriptedExecutor{}
	result := executeSystemReboot(context.Background(), exec, Request{
		Version: ProtocolVersion, RequestID: "r2", Action: ActionSystemReboot,
		Reboot: &RebootSubject{DelayMinutes: 360, DecisionID: "decision-7"},
	})
	if len(exec.calls) != 2 {
		t.Fatalf("calls = %v", exec.calls)
	}
	schedule := exec.calls[0]
	if schedule[0] != systemdRunBinary {
		t.Fatalf("scheduler = %q", schedule[0])
	}
	joined := strings.Join(schedule, " ")
	for _, want := range []string{"--on-active=360m", "--unit " + scheduledRebootUnit, "/usr/bin/systemctl reboot --no-block"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("scheduled argv missing %q: %v", want, schedule)
		}
	}
	if result.Status != "changed" || !strings.Contains(result.Evidence.Detail, "360m") {
		t.Fatalf("result = %+v", result)
	}
}

func TestSystemRebootExistingTimerIsAlreadyPresent(t *testing.T) {
	exec := &scriptedExecutor{results: []error{errors.New("unit exists"), nil}}
	result := executeSystemReboot(context.Background(), exec, Request{
		Version: ProtocolVersion, RequestID: "r3", Action: ActionSystemReboot,
		Reboot: &RebootSubject{DelayMinutes: 60},
	})
	if result.Status != "already_present" {
		t.Fatalf("result = %+v", result)
	}
}

func TestSystemRebootReportsScheduleThatDidNotTake(t *testing.T) {
	exec := &scriptedExecutor{results: []error{errors.New("schedule failed"), errors.New("not active")}}
	result := executeSystemReboot(context.Background(), exec, Request{
		Version: ProtocolVersion, RequestID: "r4", Action: ActionSystemReboot,
		Reboot: &RebootSubject{DelayMinutes: 60},
	})
	if result.Status != "failed" || result.Code != "reboot_schedule_failed" {
		t.Fatalf("result = %+v", result)
	}
}

func TestSystemRebootCancelIsANoopWhenNothingIsScheduled(t *testing.T) {
	exec := &scriptedExecutor{results: []error{errors.New("inactive")}}
	result := executeSystemRebootCancel(context.Background(), exec, Request{
		Version: ProtocolVersion, RequestID: "c1", Action: ActionSystemRebootCancel,
	})
	if result.Status != "unchanged" || result.Changed {
		t.Fatalf("result = %+v", result)
	}
}

func TestSystemRebootCancelStopsAndVerifies(t *testing.T) {
	exec := &scriptedExecutor{results: []error{nil, nil, errors.New("inactive")}}
	result := executeSystemRebootCancel(context.Background(), exec, Request{
		Version: ProtocolVersion, RequestID: "c2", Action: ActionSystemRebootCancel,
	})
	wantLast := []string{"/usr/bin/systemctl", "stop", scheduledRebootUnit + ".timer"}
	if !reflect.DeepEqual(exec.calls[1], wantLast) {
		t.Fatalf("stop call = %v", exec.calls[1])
	}
	if result.Status != "changed" || !result.Changed {
		t.Fatalf("result = %+v", result)
	}
}

func TestSystemRebootCancelFailsWhenTheTimerSurvives(t *testing.T) {
	exec := &scriptedExecutor{results: []error{nil, nil, nil}}
	result := executeSystemRebootCancel(context.Background(), exec, Request{
		Version: ProtocolVersion, RequestID: "c3", Action: ActionSystemRebootCancel,
	})
	if result.Status != "failed" || result.Code != "reboot_cancel_unverified" {
		t.Fatalf("result = %+v", result)
	}
}

func TestKernelMemorySortCountReachesTheSampler(t *testing.T) {
	exec := &scriptedExecutor{}
	report, err := runKernelMemoryStackSampler(context.Background(), exec, "/test/memleak-bpfcc", KernelMemorySortCount)
	if err != nil {
		t.Fatal(err)
	}
	if len(exec.calls) != 1 || !strings.Contains(strings.Join(exec.calls[0], " "), "--sort count") {
		t.Fatalf("sort selection did not reach argv: %v", exec.calls)
	}
	if report.SortBy != KernelMemorySortCount {
		t.Fatalf("report sort = %q", report.SortBy)
	}
}

func TestKernelMemorySortDefaultsToSize(t *testing.T) {
	exec := &scriptedExecutor{}
	report, err := runKernelMemoryStackSampler(context.Background(), exec, "/test/memleak-bpfcc", "")
	if err != nil {
		t.Fatal(err)
	}
	if report.SortBy != KernelMemorySortSize {
		t.Fatalf("report sort = %q", report.SortBy)
	}
	if !strings.Contains(strings.Join(exec.calls[0], " "), "--sort size") {
		t.Fatalf("argv = %v", exec.calls)
	}
}
