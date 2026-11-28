package privilegebroker

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

type hostMaintenanceExecutor struct {
	calls  [][]string
	failAt int
	output []byte
}

func (e *hostMaintenanceExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	e.calls = append(e.calls, append([]string{name}, args...))
	if len(e.calls) == e.failAt {
		return nil, errors.New("service failure")
	}
	return e.output, nil
}

func TestHostRecoveryUsesOnlyFixedServicesAndVerifiesActivity(t *testing.T) {
	exec := &hostMaintenanceExecutor{}
	b, err := New(Config{SocketPath: "/unused.sock", AllowedUID: 1000, Executor: exec})
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Version: ProtocolVersion, RequestID: "service-1", Action: ActionServiceRecover, Service: &ServiceSubject{Unit: "rsyslog", Operation: "restart"}}
	result := b.Execute(context.Background(), req, 1000)
	want := [][]string{{"/usr/bin/systemctl", "restart", "rsyslog.service"}, {"/usr/bin/systemctl", "is-active", "--quiet", "rsyslog.service"}}
	if result.Status != "verified" || !reflect.DeepEqual(exec.calls, want) {
		t.Fatalf("result=%+v calls=%v", result, exec.calls)
	}
	for _, subject := range []ServiceSubject{{Unit: "ssh", Operation: "restart"}, {Unit: "rsyslog;sh", Operation: "restart"}, {Unit: "rsyslog", Operation: "stop"}, {Unit: "rsyslog", Operation: "enable"}} {
		req.Service = &subject
		if Validate(req) == nil {
			t.Fatalf("accepted %+v", subject)
		}
	}
	req.Service = &ServiceSubject{Unit: "rsyslog", Operation: "restart"}
	req.Journal = &JournalSubject{MaxUseBytes: 1}
	if Validate(req) == nil {
		t.Fatal("accepted mixed action subjects")
	}
	if got := b.Execute(context.Background(), req, 999); got.Code != "caller_not_authorized" {
		t.Fatalf("wrong caller: %+v", got)
	}
}

func TestHostRecoveryCannotClaimSuccessWithoutActiveService(t *testing.T) {
	exec := &hostMaintenanceExecutor{failAt: 2}
	b, _ := New(Config{SocketPath: "/unused.sock", AllowedUID: 1000, Executor: exec})
	result := b.Execute(context.Background(), Request{Version: ProtocolVersion, RequestID: "service-2", Action: ActionServiceRecover, Service: &ServiceSubject{Unit: "rsyslog.service", Operation: "restart"}}, 1000)
	if result.Status != "failed" || result.Code != "service_not_active" {
		t.Fatalf("false success: %+v", result)
	}
}

func TestKernelMemoryIsReadOnlySubjectlessAndPermissionErrorsStayUnavailable(t *testing.T) {
	called := false
	b, _ := New(Config{SocketPath: "/unused.sock", AllowedUID: 1000, KernelMemory: func() (hostinventory.KernelMemoryReport, error) {
		called = true
		return hostinventory.KernelMemoryReport{}, errors.New("permission denied")
	}})
	req := Request{Version: ProtocolVersion, RequestID: "memory-1", Action: ActionKernelMemoryInspect}
	result := b.Execute(context.Background(), req, 1000)
	if !called || result.Status != "failed" || result.Code != "kernel_memory_unavailable" || result.KernelMemory != nil {
		t.Fatalf("masked diagnostic failure: %+v", result)
	}
	req.Service = &ServiceSubject{Unit: "rsyslog", Operation: "restart"}
	if Validate(req) == nil {
		t.Fatal("read action accepted a recovery subject")
	}
}

func TestKernelMemoryStackSampleUsesBoundedFixedArguments(t *testing.T) {
	exec := &hostMaintenanceExecutor{output: []byte("stack summary\n")}
	report, err := runKernelMemoryStackSampler(context.Background(), exec, "/test/memleak-bpfcc", KernelMemorySortSize)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"/test/memleak-bpfcc", "--combined-only", "--sample-rate", "100", "--sort", "size", "--top", "10", "10", "1"}}
	if !reflect.DeepEqual(exec.calls, want) {
		t.Fatalf("sampler arguments were not fixed and bounded: %v", exec.calls)
	}
	if report.WindowSeconds != 10 || report.SampleEvery != 100 || report.StackLimit != 10 || report.Output != "stack summary" {
		t.Fatalf("unexpected bounded sample metadata: %+v", report)
	}
}

func TestKernelMemoryStackSampleIsSubjectlessAndPropagatesFailure(t *testing.T) {
	called := false
	b, err := New(Config{SocketPath: "/unused.sock", AllowedUID: 1000, KernelMemoryStacks: func(context.Context, string) (hostinventory.KernelMemoryStackReport, error) {
		called = true
		return hostinventory.KernelMemoryStackReport{}, errors.New("sampler unavailable")
	}})
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Version: ProtocolVersion, RequestID: "stacks-1", Action: ActionKernelMemoryStackSample}
	result := b.Execute(context.Background(), req, 1000)
	if !called || result.Status != "failed" || result.Code != "kernel_memory_stack_sample_unavailable" || result.KernelMemoryStacks != nil {
		t.Fatalf("sample failure was masked: %+v", result)
	}
	req.Service = &ServiceSubject{Unit: "rsyslog", Operation: "restart"}
	if Validate(req) == nil {
		t.Fatal("read-only sampling accepted a recovery subject")
	}
}

func TestBrokerUpgradeDetectsMissingHostCapabilities(t *testing.T) {
	missing := missingCapabilities([]string{ActionCapabilitiesInspect})
	if !strings.Contains(strings.Join(missing, ","), ActionKernelMemoryInspect) || !strings.Contains(strings.Join(missing, ","), ActionKernelMemoryStackSample) || !strings.Contains(strings.Join(missing, ","), ActionServiceRecover) {
		t.Fatalf("missing registry drift: %v", missing)
	}
	if len(missingCapabilities(capabilities())) != 0 {
		t.Fatal("current registry needs an upgrade")
	}
}
