//go:build linux

package platform

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func ownedServiceTestSpec() OwnedServiceSpec {
	return OwnedServiceSpec{Path: "/usr/bin/true", Dir: "/tmp", Args: []string{"--literal"}, Env: []string{"LANG=C"}, Limits: ServiceLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 100}}
}

func TestOwnedServiceHasNoJobDeadlineAndUsesSeparateNamespace(t *testing.T) {
	spec := ownedServiceTestSpec()
	intent, err := NewServiceIntent("resource-update", "candidate-a", spec)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	var recovered ServiceIntent
	if err := json.Unmarshal(encoded, &recovered); err != nil || recovered != intent {
		t.Fatalf("intent did not survive caller loss: %v", err)
	}
	cmd, err := ownedServiceCommand(OperationIntent(recovered), spec.processSpec())
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"--slice=" + ServicesSlice, "--property=RuntimeMaxSec=infinity", "--property=Restart=no", "--property=ExitType=cgroup", "--property=MemoryMax=268435456", "--property=TasksMax=32", "--property=CPUWeight=100"} {
		if !slices.Contains(cmd.Args, expected) {
			t.Fatalf("service launch omitted %s", expected)
		}
	}
	unit := operationUnitName(OperationIntent(intent))
	if !strings.HasPrefix(unit, "vrooli-svc-resource-update-") || slices.Contains(cmd.Args, "--slice="+OperationsSlice) {
		t.Fatal("service shares a bounded job namespace")
	}
	if !validOperationServicePath("/user.slice/"+ServicesSlice+"/"+unit, unit) || validOperationServicePath("/user.slice/"+OperationsSlice+"/"+unit, unit) {
		t.Fatal("service scope validation accepted another workload namespace")
	}
}

func TestOwnedServiceAndBoundedOperationAdmissionAreNotInterchangeable(t *testing.T) {
	spec := ownedServiceTestSpec()
	service, err := NewServiceIntent("resource-update", "candidate-a", spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := StartOwnedOperation(context.Background(), OperationIntent(service), operationTestSpec()); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("job API accepted a service intent: %v", err)
	}
	job, err := NewOperationIntent("resource-update", "job-a", operationTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := StartOwnedService(context.Background(), ServiceIntent(job), spec); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("service API removed a job's deadline: %v", err)
	}
	if _, err := NewOperationIntent("resource-update", "job-a", spec.processSpec()); err == nil {
		t.Fatal("unbounded process spec entered the job API")
	}
	service.Kind = "unknown"
	if _, err := LookupOwnedService(context.Background(), service); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("unknown workload mode reached native lookup: %v", err)
	}
}

func TestOwnedServiceRetainsResourceBoundsAndRejectsChangedInvocation(t *testing.T) {
	spec := ownedServiceTestSpec()
	intent, err := NewServiceIntent("resource-update", "candidate-a", spec)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*OwnedServiceSpec){
		"executable":  func(s *OwnedServiceSpec) { s.Path = "/usr/bin/false" },
		"arguments":   func(s *OwnedServiceSpec) { s.Args = []string{"changed"} },
		"environment": func(s *OwnedServiceSpec) { s.Env = []string{"LANG=other"} },
		"directory":   func(s *OwnedServiceSpec) { s.Dir = "/" },
		"memory":      func(s *OwnedServiceSpec) { s.Limits.MemoryMaxBytes++ },
		"tasks":       func(s *OwnedServiceSpec) { s.Limits.TasksMax++ },
		"cpu":         func(s *OwnedServiceSpec) { s.Limits.CPUWeight++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := ownedServiceTestSpec()
			change(&changed)
			if _, err := StartOwnedService(context.Background(), intent, changed); !errors.Is(err, ErrOperationIdentity) {
				t.Fatalf("changed service invocation was dispatched: %v", err)
			}
		})
	}
	for name, change := range map[string]func(*OwnedServiceSpec){
		"memory":                func(s *OwnedServiceSpec) { s.Limits.MemoryMaxBytes = 0 },
		"tasks":                 func(s *OwnedServiceSpec) { s.Limits.TasksMax = 0 },
		"cpu":                   func(s *OwnedServiceSpec) { s.Limits.CPUWeight = 0 },
		"implicit environment":  func(s *OwnedServiceSpec) { s.Env = nil },
		"duplicate environment": func(s *OwnedServiceSpec) { s.Env = []string{"X=one", "X=two"} },
		"relative executable":   func(s *OwnedServiceSpec) { s.Path = "service" },
	} {
		t.Run("invalid "+name, func(t *testing.T) {
			invalid := ownedServiceTestSpec()
			change(&invalid)
			if _, err := NewServiceIntent("resource-update", "candidate-a", invalid); err == nil {
				t.Fatal("unbounded or implicit service input accepted")
			}
		})
	}
}

func TestOwnedServiceNeverStopsAReplacedNativeInvocation(t *testing.T) {
	intent, err := NewServiceIntent("resource-update", "candidate-a", ownedServiceTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	unit := operationUnitName(OperationIntent(intent))
	ref := ServiceRef{Intent: OperationIntent(intent), Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/" + ServicesSlice + "/" + unit}, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	oldQuery, oldSignal := queryOwnedUnit, signalOwnedUnit
	t.Cleanup(func() { queryOwnedUnit, signalOwnedUnit = oldQuery, oldSignal })
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
		return ownedUnitState{LoadState: "loaded", ActiveState: "active", SubState: "running", ControlGroup: ref.Scope.Path, InvocationID: "fedcba9876543210fedcba9876543210"}, nil
	}
	signalOwnedUnit = func(context.Context, string, string) error {
		t.Fatal("foreign native invocation was signalled")
		return nil
	}
	if _, err := InspectOwnedService(context.Background(), ref); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("inspection accepted replacement invocation: %v", err)
	}
	if _, err := StopOwnedService(context.Background(), ref); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("stop accepted replacement invocation: %v", err)
	}
	if err := ReleaseOwnedService(context.Background(), ref); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("release accepted replacement invocation: %v", err)
	}
}

func TestOwnedNativeManagerCommandFailurePreservesCause(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := queryOwnedUnitSystemd(context.Background(), "vrooli-svc-platform-go-missing.service")
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("manager command failure became a misleading response-shape error: %v", err)
	}
}

func TestOwnedServiceManagerInterruptionRetainsUnknownOutcomeAndSameRecoveryIdentity(t *testing.T) {
	intent, err := NewServiceIntent("resource-update", "manager-interruption", ownedServiceTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	unit := operationUnitName(OperationIntent(intent))
	ref := ServiceRef{Intent: OperationIntent(intent), Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/" + ServicesSlice + "/" + unit}, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	oldQuery, oldRoot := queryOwnedUnit, operationCgroupRoot
	t.Cleanup(func() { queryOwnedUnit, operationCgroupRoot = oldQuery, oldRoot })
	operationCgroupRoot = t.TempDir()
	interrupted := errors.New("injected native manager transport unavailable")
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) { return ownedUnitState{}, interrupted }
	if _, err := LookupOwnedService(context.Background(), intent); !errors.Is(err, interrupted) {
		t.Fatalf("manager failure was interpreted as service absence: %v", err)
	}
	if _, err := InspectOwnedService(context.Background(), ref); !errors.Is(err, interrupted) {
		t.Fatalf("manager failure was interpreted as service health: %v", err)
	}
	if receipt, err := StopOwnedService(context.Background(), ref); !errors.Is(err, interrupted) || receipt.NativeQuiescent {
		t.Fatalf("unavailable manager acknowledged service cleanup: %+v %v", receipt, err)
	}
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
		return ownedUnitState{LoadState: "loaded", ActiveState: "inactive", SubState: "dead", InvocationID: ref.NativeInvocationID, ControlGroup: ref.Scope.Path, ExitKnown: true}, nil
	}
	if receipt, err := StopOwnedService(context.Background(), ref); err != nil || !receipt.NativeQuiescent || receipt.Ref.Intent != OperationIntent(intent) || receipt.Ref.NativeInvocationID != ref.NativeInvocationID {
		t.Fatalf("recovery changed identity or failed to verify exact exclusion: %+v %v", receipt, err)
	}
}
