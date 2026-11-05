//go:build linux

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func operationTestSpec() OperationSpec {
	return OperationSpec{
		Path: "/usr/bin/true", Args: []string{"--fixed"}, Env: []string{"LANG=C", "MODE=render"}, Dir: "/tmp",
		Limits: OperationLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 100, WallTime: 2 * time.Minute},
	}
}

func TestOperationIntentBindsResolvedInvocationAndHasUniqueAttemptIdentity(t *testing.T) {
	spec := operationTestSpec()
	first, err := NewOperationIntent("device-studio", "job-a-attempt-1", spec)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewOperationIntent("device-studio", "job-a-attempt-1", spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.InvocationID == second.InvocationID {
		t.Fatal("new intents reused invocation identity")
	}
	if err := validateOperationIntent(first, spec); err != nil {
		t.Fatalf("valid intent rejected: %v", err)
	}
	changed := operationTestSpec()
	changed.Args = []string{"--changed"}
	if err := validateOperationIntent(first, changed); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("changed invocation was not rejected as an identity mismatch: %v", err)
	}
}

func TestOperationSpecRejectsUnboundedInputsAndRedactsEnvironmentValues(t *testing.T) {
	t.Run("environment value is redacted", func(t *testing.T) {
		spec := operationTestSpec()
		spec.Env = []string{"TOKEN=private-secret"}
		// A malformed key still carries a secret-looking value; diagnostics must
		// identify the key without serializing the supplied value.
		spec.Env = []string{"=private-secret"}
		err := validateOperationSpec(spec)
		if err == nil {
			t.Fatal("malformed environment accepted")
		}
		if strings.Contains(err.Error(), "private-secret") {
			t.Fatalf("environment secret leaked in error: %v", err)
		}
	})
	t.Run("wall limit is mandatory", func(t *testing.T) {
		spec := operationTestSpec()
		spec.Limits.WallTime = 0
		if err := validateOperationSpec(spec); err == nil {
			t.Fatal("missing wall bound accepted")
		}
	})
	t.Run("wall limit is capped", func(t *testing.T) {
		spec := operationTestSpec()
		spec.Limits.WallTime = maxOperationWallTime + time.Nanosecond
		if err := validateOperationSpec(spec); err == nil {
			t.Fatal("excessive wall bound accepted")
		}
	})
	t.Run("environment is explicit", func(t *testing.T) {
		spec := operationTestSpec()
		spec.Env = nil
		if err := validateOperationSpec(spec); err == nil {
			t.Fatal("nil environment accepted")
		}
	})
}

func TestOperationIntentAndRefRejectReusedOrChangedIdentity(t *testing.T) {
	spec := operationTestSpec()
	intent, err := NewOperationIntent("device-studio", "attempt-1", spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateOperationIntentShape(intent); err != nil {
		t.Fatal(err)
	}
	ref := OperationRef{Intent: intent, Unit: operationUnitName(intent), Scope: ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/user-1000.slice/user@1000.service/vrooli-operations.slice/" + operationUnitName(intent)}, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	if err := validateOperationRefShape(ref); err != nil {
		t.Fatalf("valid native identity rejected: %v", err)
	}
	wrong := ref
	wrong.Unit = "vrooli-op-device-studio-later.service"
	if err := validateOperationRefShape(wrong); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("wrong unit accepted: %v", err)
	}
	wrong = ref
	wrong.Intent.InvocationID = "fedcba9876543210fedcba9876543210"
	if err := validateOperationRefShape(wrong); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("reused invocation accepted: %v", err)
	}
	wrong = ref
	wrong.Scope.Path = strings.Replace(ref.Scope.Path, OperationsSlice, "vrooli-services.slice", 1)
	if err := validateOperationRefShape(wrong); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("foreign namespace accepted: %v", err)
	}
}

func TestOwnedServiceCommandUsesFixedManagerArgvAndBoundedProperties(t *testing.T) {
	spec := operationTestSpec()
	spec.Args = []string{"--literal; $(touch /tmp/should-not-run)", "space stays"}
	intent, err := NewOperationIntent("device-studio", "argv-test", spec)
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := ownedServiceCommand(intent, spec)
	if err != nil {
		t.Fatal(err)
	}
	args := cmd.Args[1:]
	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 {
		t.Fatal("command has no explicit exec separator")
	}
	for _, want := range []string{
		"--no-block", "--remain-after-exit", "--service-type=exec", "--expand-environment=no", "--unit=" + operationUnitName(intent), "--slice=" + OperationsSlice,
		"--property=RuntimeMaxSec=2m0s", "--property=ExitType=cgroup", "--property=KillMode=mixed", "--property=TimeoutStopSec=10s", "--property=MemoryMax=268435456", "--property=TasksMax=32", "--property=CPUWeight=100", "--property=LogRateLimitIntervalSec=30s", "--property=LogRateLimitBurst=1000", "--property=StandardOutput=journal", "--property=StandardError=journal", "--working-directory=/tmp",
	} {
		found := false
		for _, arg := range args[:separator] {
			if arg == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("manager invocation omitted option %q", want)
		}
	}
	if got := args[separator+1:]; len(got) != 7 || got[0] != "/usr/bin/env" || got[1] != "-i" || got[2] != spec.Env[0] || got[3] != spec.Env[1] || got[4] != spec.Path || got[5] != spec.Args[0] || got[6] != spec.Args[1] {
		t.Fatalf("executable/argv were changed or shell interpreted (got %d command arguments)", len(got))
	}
}

func TestPublicOperationAPIRejectsUnpersistedOrInvalidCallsBeforeNativeWork(t *testing.T) {
	if _, err := StartOwnedOperation(context.Background(), OperationIntent{}, operationTestSpec()); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("malformed intent accepted: %v", err)
	}
	if _, err := LookupOwnedOperation(context.Background(), OperationIntent{}); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("malformed recovery intent accepted: %v", err)
	}
	if _, err := StartOwnedOperation(context.Background(), OperationIntent{}, operationTestSpec()); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("invalid start did not fail before dispatch: %v", err)
	}
}

func TestParseCgroupPopulationRequiresRecursiveKernelEvidence(t *testing.T) {
	for _, test := range []struct {
		data string
		want bool
		ok   bool
	}{
		{"populated 0\nfrozen 0\n", false, true},
		{"populated 1\nfrozen 0\n", true, true},
		{"populated 2\n", false, false},
		{"frozen 0\n", false, false},
	} {
		got, err := parseCgroupPopulated(test.data)
		if (err == nil) != test.ok || (err == nil && got != test.want) {
			t.Errorf("parseCgroupPopulated(%q) = %t,%v; want %t, ok=%t", test.data, got, err, test.want, test.ok)
		}
	}
}

func TestParseOwnedUnitStatePreservesTerminalEvidenceAndRejectsAmbiguity(t *testing.T) {
	state, err := parseOwnedUnitState("LoadState=loaded\nActiveState=active\nSubState=exited\nControlGroup=/user.slice/user-1000.slice/user@1000.service/vrooli-operations.slice/vrooli-op-device-studio-a.service\nInvocationID=0123456789abcdef0123456789abcdef\nResult=success\nMainPID=0\nExecMainCode=1\nExecMainStatus=0\nJob=\nJobType=\nJobPath=\n")
	if err != nil {
		t.Fatal(err)
	}
	if !state.ExitKnown || state.Result != "success" || state.ExecMainStatus != 0 || state.MainPID != 0 {
		t.Fatalf("terminal manager result was lost: %+v", state)
	}
	starting, err := parseOwnedUnitState("LoadState=loaded\nActiveState=active\nSubState=running\nControlGroup=/user.slice/user-1000.slice/user@1000.service/vrooli-operations.slice/vrooli-op-device-studio-a.service\nInvocationID=0123456789abcdef0123456789abcdef\nResult=success\nMainPID=345\nExecMainCode=0\nExecMainStatus=0\nJob=\nJobType=\nJobPath=\n")
	if err != nil {
		t.Fatal(err)
	}
	if starting.ExitKnown {
		t.Fatalf("default zero status was mistaken for an exited operation: %+v", starting)
	}
	valid := "LoadState=loaded\nActiveState=active\nSubState=running\nControlGroup=/user.slice/user-1000.slice/user@1000.service/vrooli-operations.slice/vrooli-op-device-studio-a.service\nInvocationID=0123456789abcdef0123456789abcdef\nResult=success\nMainPID=345\nExecMainCode=0\nExecMainStatus=0\nJob=\nJobType=\nJobPath=\n"
	if _, err := parseOwnedUnitState("LoadState=loaded\nActiveState=active\n"); err == nil {
		t.Fatal("manager response missing required lifecycle evidence was accepted")
	}
	if _, err := parseOwnedUnitState(strings.Replace(valid, "MainPID=345\n", "", 1)); err == nil {
		t.Fatal("manager response missing MainPID was accepted")
	}
	if _, err := parseOwnedUnitState(strings.Replace(valid, "SubState=running\n", "", 1)); err == nil {
		t.Fatal("manager response missing SubState was accepted")
	}
	if _, err := parseOwnedUnitState(strings.Replace(valid, "Job=\n", "", 1)); err == nil {
		t.Fatal("manager response missing Job state was accepted")
	}
	withoutOptionalJobFields := strings.Replace(strings.Replace(valid, "JobType=\n", "", 1), "JobPath=\n", "", 1)
	if _, err := parseOwnedUnitState(withoutOptionalJobFields); err != nil {
		t.Fatalf("idle manager response without optional JobType/JobPath was rejected: %v", err)
	}
	if _, err := parseOwnedUnitState(strings.Replace(valid, "MainPID=345", "MainPID=not-a-pid", 1)); err == nil {
		t.Fatal("malformed manager PID was accepted")
	}
	pendingData := strings.Replace(valid, "ActiveState=active", "ActiveState=inactive", 1)
	pendingData = strings.Replace(pendingData, "Job=\n", "Job=42\n", 1)
	pending, err := parseOwnedUnitState(pendingData)
	if err != nil {
		t.Fatal(err)
	}
	if terminalOperationState(pending) || pending.ExitKnown {
		t.Fatalf("pending manager job was accepted as terminal: %+v", pending)
	}
	inconsistent := starting
	inconsistent.ActiveState, inconsistent.SubState = "inactive", "running"
	if terminalOperationState(inconsistent) {
		t.Fatalf("inconsistent manager state was accepted as terminal: %+v", inconsistent)
	}
	notFound, err := parseOwnedUnitState("MainPID=0\nResult=success\nExecMainCode=0\nExecMainStatus=0\nControlGroup=\nLoadState=not-found\nActiveState=inactive\nSubState=dead\nJob=\nInvocationID=\n")
	if err != nil || notFound.LoadState != "not-found" {
		t.Fatalf("actual unloaded-manager response shape was rejected: state=%+v err=%v", notFound, err)
	}
}

func TestOperationTreeRequiresTerminalManagerStateEvenWhenCgroupIsEmpty(t *testing.T) {
	root := t.TempDir()
	oldRoot := operationCgroupRoot
	operationCgroupRoot = root
	t.Cleanup(func() { operationCgroupRoot = oldRoot })
	path := "/user.slice/vrooli-operations.slice/vrooli-op-test.service"
	dir := filepath.Join(root, strings.TrimPrefix(path, "/"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cgroup.events"), []byte("populated 0\nfrozen 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pending := ownedUnitState{LoadState: "loaded", ActiveState: "activating", SubState: "start", MainPID: 0}
	if empty, err := operationTreeEmpty(pending, path); err != nil || empty {
		t.Fatalf("empty cgroup during pending start claimed quiescence: empty=%t err=%v", empty, err)
	}
	running := ownedUnitState{LoadState: "loaded", ActiveState: "active", SubState: "running", MainPID: 0}
	if empty, err := operationTreeEmpty(running, path); err != nil || empty {
		t.Fatalf("empty cgroup during start/running claimed quiescence: empty=%t err=%v", empty, err)
	}
	terminal := ownedUnitState{LoadState: "loaded", ActiveState: "active", SubState: "exited", MainPID: 0, ExitKnown: true, Result: "success"}
	if empty, err := operationTreeEmpty(terminal, path); err != nil || !empty {
		t.Fatalf("terminal successful service with empty cgroup was not quiescent: empty=%t err=%v", empty, err)
	}
	failed := ownedUnitState{LoadState: "loaded", ActiveState: "failed", SubState: "failed", MainPID: 0, ExitKnown: true, Result: "exit-code"}
	if empty, err := operationTreeEmpty(failed, path); err != nil || !empty {
		t.Fatalf("terminal failed service with empty cgroup was not quiescent: empty=%t err=%v", empty, err)
	}
}

func TestOperationCgroupAbsenceDistinguishesDirectoryFromEvents(t *testing.T) {
	root := t.TempDir()
	oldRoot := operationCgroupRoot
	operationCgroupRoot = root
	t.Cleanup(func() { operationCgroupRoot = oldRoot })
	path := "/user.slice/vrooli-operations.slice/vrooli-op-test.service"
	dir := filepath.Join(root, strings.TrimPrefix(path, "/"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := linuxScopePopulated(path); err == nil || errors.Is(err, errOperationCgroupAbsent) {
		t.Fatalf("existing cgroup with missing cgroup.events was treated as absent: %v", err)
	}
	terminal := ownedUnitState{LoadState: "loaded", ActiveState: "inactive", SubState: "dead", MainPID: 0, ExitKnown: true, Result: "success"}
	if empty, err := operationTreeEmpty(terminal, path); err == nil {
		t.Fatalf("missing events in existing cgroup was accepted as empty: empty=%t", empty)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if populated, err := linuxScopePopulated(path); !errors.Is(err, errOperationCgroupAbsent) || populated {
		t.Fatalf("missing cgroup directory not distinguished: populated=%t err=%v", populated, err)
	}
	if empty, err := operationTreeEmpty(terminal, path); err != nil || !empty {
		t.Fatalf("terminal service with removed cgroup was not accepted as quiescent: empty=%t err=%v", empty, err)
	}
	pending := ownedUnitState{LoadState: "loaded", ActiveState: "activating", SubState: "start", MainPID: 0}
	if empty, err := operationTreeEmpty(pending, path); err != nil || empty {
		t.Fatalf("missing cgroup plus pending manager state claimed quiescence: empty=%t err=%v", empty, err)
	}
}

func TestAwaitOperationStartRequiresManagerAcknowledgement(t *testing.T) {
	oldQuery := queryOwnedUnit
	t.Cleanup(func() { queryOwnedUnit = oldQuery })
	unit := "vrooli-op-device-studio-test.service"
	cgroup := "/user.slice/user-1000.slice/user@1000.service/vrooli-operations.slice/" + unit
	t.Run("pending empty state remains ambiguous until running", func(t *testing.T) {
		calls := 0
		queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
			calls++
			if calls < 2 {
				return ownedUnitState{LoadState: "loaded", ActiveState: "activating", SubState: "start", InvocationID: "0123456789abcdef0123456789abcdef"}, nil
			}
			return ownedUnitState{LoadState: "loaded", ActiveState: "active", SubState: "running", ControlGroup: cgroup, InvocationID: "0123456789abcdef0123456789abcdef", MainPID: 34}, nil
		}
		state, err := awaitOperationStart(context.Background(), unit)
		if err != nil || state.SubState != "running" || calls < 2 {
			t.Fatalf("start returned before manager acknowledgment: state=%+v calls=%d err=%v", state, calls, err)
		}
	})
	t.Run("fast successful exit is retained", func(t *testing.T) {
		queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
			return ownedUnitState{LoadState: "loaded", ActiveState: "active", SubState: "exited", InvocationID: "0123456789abcdef0123456789abcdef", MainPID: 0, ExitKnown: true, Result: "success"}, nil
		}
		state, err := awaitOperationStart(context.Background(), unit)
		if err != nil || !state.ExitKnown || state.Result != "success" {
			t.Fatalf("fast terminal success was not preserved: state=%+v err=%v", state, err)
		}
	})
	t.Run("terminal failed start is retained", func(t *testing.T) {
		queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
			return ownedUnitState{LoadState: "loaded", ActiveState: "failed", SubState: "failed", InvocationID: "0123456789abcdef0123456789abcdef", MainPID: 0, ExitKnown: true, Result: "exit-code"}, nil
		}
		state, err := awaitOperationStart(context.Background(), unit)
		if err != nil || !state.ExitKnown || state.Result != "exit-code" {
			t.Fatalf("terminal failed start status was discarded: state=%+v err=%v", state, err)
		}
	})
	t.Run("ambiguous empty start times out", func(t *testing.T) {
		oldWait := operationStartWait
		operationStartWait = time.Millisecond
		defer func() { operationStartWait = oldWait }()
		queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
			return ownedUnitState{LoadState: "loaded", ActiveState: "activating", SubState: "start", InvocationID: "0123456789abcdef0123456789abcdef", MainPID: 0}, nil
		}
		if _, err := awaitOperationStart(context.Background(), unit); !errors.Is(err, ErrOperationOutcomeLost) {
			t.Fatalf("ambiguous start was not left for reconciliation: %v", err)
		}
	})
}

func TestStopCancelsExactPendingStartBeforeClaimingQuiescence(t *testing.T) {
	oldQuery, oldCancel := queryOwnedUnit, cancelOwnedUnit
	t.Cleanup(func() { queryOwnedUnit, cancelOwnedUnit = oldQuery, oldCancel })
	spec := operationTestSpec()
	intent, err := NewOperationIntent("device-studio", "pending-stop", spec)
	if err != nil {
		t.Fatal(err)
	}
	unit := operationUnitName(intent)
	ref := OperationRef{Intent: intent, Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/user-1000.slice/user@1000.service/" + OperationsSlice + "/" + unit}, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	state := ownedUnitState{LoadState: "loaded", ActiveState: "activating", SubState: "start", ControlGroup: ref.Scope.Path, InvocationID: ref.NativeInvocationID}
	queries := 0
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
		queries++
		if queries == 1 {
			return state, nil
		}
		return ownedUnitState{LoadState: "loaded", ActiveState: "inactive", SubState: "dead", ControlGroup: ref.Scope.Path, InvocationID: ref.NativeInvocationID}, nil
	}
	canceled := false
	cancelOwnedUnit = func(_ context.Context, got string) error {
		if got != unit {
			t.Fatalf("canceled wrong manager unit: %s", got)
		}
		canceled = true
		return nil
	}
	oldRoot := operationCgroupRoot
	operationCgroupRoot = t.TempDir() // absent exact cgroup is accepted only after terminal manager state
	t.Cleanup(func() { operationCgroupRoot = oldRoot })
	receipt, err := StopOwnedOperation(context.Background(), ref)
	if err != nil {
		t.Fatalf("cancel exact pending unit: %v", err)
	}
	if !canceled || !receipt.NativeQuiescent || queries < 2 {
		t.Fatalf("pending start was not canceled and verified: canceled=%t queries=%d receipt=%+v", canceled, queries, receipt)
	}
}

func TestStopSendsGracefulSignalToMainBeforeKillingOperationTree(t *testing.T) {
	oldQuery, oldSignal, oldRoot, oldGrace := queryOwnedUnit, signalOwnedUnit, operationCgroupRoot, operationTermGrace
	t.Cleanup(func() {
		queryOwnedUnit, signalOwnedUnit, operationCgroupRoot, operationTermGrace = oldQuery, oldSignal, oldRoot, oldGrace
	})
	operationTermGrace = 100 * time.Millisecond
	operationCgroupRoot = t.TempDir()
	spec := operationTestSpec()
	intent, err := NewOperationIntent("device-studio", "graceful-stop-order", spec)
	if err != nil {
		t.Fatal(err)
	}
	unit := operationUnitName(intent)
	path := "/user.slice/user-1000.slice/user@1000.service/" + OperationsSlice + "/" + unit
	ref := OperationRef{Intent: intent, Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: path}, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	cgroup := filepath.Join(operationCgroupRoot, strings.TrimPrefix(path, "/"))
	if err := os.MkdirAll(cgroup, 0o700); err != nil {
		t.Fatal(err)
	}
	events := filepath.Join(cgroup, "cgroup.events")
	if err := os.WriteFile(events, []byte("populated 1\nfrozen 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	terminal := false
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
		state := ownedUnitState{LoadState: "loaded", ActiveState: "active", SubState: "running", MainPID: 1234, ControlGroup: path, InvocationID: ref.NativeInvocationID}
		if terminal {
			state.ActiveState, state.SubState, state.MainPID, state.ExitKnown, state.Result = "inactive", "dead", 0, true, "success"
		}
		return state, nil
	}
	var signals []string
	signalOwnedUnit = func(_ context.Context, gotUnit, signal string) error {
		if gotUnit != unit {
			t.Fatalf("signaled unexpected unit %q", gotUnit)
		}
		signals = append(signals, signal)
		if signal == "SIGTERM" {
			// Model the operation worker finishing its cleanup child and exiting.
			terminal = true
			if err := os.WriteFile(events, []byte("populated 0\nfrozen 0\n"), 0o600); err != nil {
				return err
			}
		}
		return nil
	}
	if _, err := StopOwnedOperation(context.Background(), ref); err != nil {
		t.Fatalf("graceful stop failed: %v", err)
	}
	if len(signals) != 1 || signals[0] != "SIGTERM" {
		t.Fatalf("cleanup did not finish during the main-only grace period: signals=%v", signals)
	}
}

func TestOwnedUnitSignalsTargetMainThenWholeTree(t *testing.T) {
	for _, test := range []struct {
		signal, want string
	}{{"SIGTERM", "main"}, {"SIGKILL", "all"}} {
		got, err := killWhomForSignal(test.signal)
		if err != nil || got != test.want {
			t.Errorf("killWhomForSignal(%q) = %q, %v; want %q", test.signal, got, err, test.want)
		}
	}
	if _, err := killWhomForSignal("SIGINT"); err == nil {
		t.Fatal("unsupported signal was accepted")
	}
}

func TestStopEscalatesToWholeTreeOnlyAfterGraceExpires(t *testing.T) {
	oldQuery, oldSignal, oldRoot, oldGrace := queryOwnedUnit, signalOwnedUnit, operationCgroupRoot, operationTermGrace
	t.Cleanup(func() {
		queryOwnedUnit, signalOwnedUnit, operationCgroupRoot, operationTermGrace = oldQuery, oldSignal, oldRoot, oldGrace
	})
	operationTermGrace = 75 * time.Millisecond
	operationCgroupRoot = t.TempDir()
	spec := operationTestSpec()
	intent, err := NewOperationIntent("device-studio", "graceful-stop-escalation", spec)
	if err != nil {
		t.Fatal(err)
	}
	unit := operationUnitName(intent)
	path := "/user.slice/user-1000.slice/user@1000.service/" + OperationsSlice + "/" + unit
	ref := OperationRef{Intent: intent, Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: path}, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	cgroup := filepath.Join(operationCgroupRoot, strings.TrimPrefix(path, "/"))
	if err := os.MkdirAll(cgroup, 0o700); err != nil {
		t.Fatal(err)
	}
	events := filepath.Join(cgroup, "cgroup.events")
	if err := os.WriteFile(events, []byte("populated 1\nfrozen 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	terminal := false
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
		state := ownedUnitState{LoadState: "loaded", ActiveState: "active", SubState: "running", MainPID: 1234, ControlGroup: path, InvocationID: ref.NativeInvocationID}
		if terminal {
			state.ActiveState, state.SubState, state.MainPID, state.ExitKnown, state.Result = "inactive", "dead", 0, true, "signal"
		}
		return state, nil
	}
	var signals []string
	signalOwnedUnit = func(_ context.Context, gotUnit, signal string) error {
		if gotUnit != unit {
			t.Fatalf("signaled unexpected unit %q", gotUnit)
		}
		signals = append(signals, signal)
		if signal == "SIGKILL" {
			terminal = true
			if err := os.WriteFile(events, []byte("populated 0\nfrozen 0\n"), 0o600); err != nil {
				return err
			}
		}
		return nil
	}
	if _, err := StopOwnedOperation(context.Background(), ref); err != nil {
		t.Fatalf("bounded graceful stop did not escalate and verify quiescence: %v", err)
	}
	if len(signals) != 2 || signals[0] != "SIGTERM" || signals[1] != "SIGKILL" {
		t.Fatalf("whole-tree kill was not used only after the main-process grace expired: signals=%v", signals)
	}
}

func TestTerminalRetainedUnitCanRecoverReferenceWithoutControlGroup(t *testing.T) {
	oldSlice := queryOperationsSlice
	t.Cleanup(func() { queryOperationsSlice = oldSlice })
	base := "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/" + OperationsSlice
	queryOperationsSlice = func(context.Context) (string, error) { return base, nil }
	spec := operationTestSpec()
	intent, err := NewOperationIntent("device-studio", "fast-terminal", spec)
	if err != nil {
		t.Fatal(err)
	}
	unit := operationUnitName(intent)
	state := ownedUnitState{
		LoadState: "loaded", ActiveState: "active", SubState: "exited", InvocationID: "0123456789abcdef0123456789abcdef",
		MainPID: 0, Result: "success", ExitKnown: true,
	}
	ref, err := makeOperationRef(context.Background(), intent, unit, state)
	if err != nil {
		t.Fatalf("terminal unit without ControlGroup could not be recovered: %v", err)
	}
	want := filepath.Join(base, unit)
	if ref.Scope.Path != want {
		t.Fatalf("derived scope path = %q, want %q", ref.Scope.Path, want)
	}
	if err := validateOwnedUnitRef(ref, state); err != nil {
		t.Fatalf("retained terminal manager record rejected: %v", err)
	}
	pending := ownedUnitState{
		LoadState: "loaded", ActiveState: "activating", SubState: "start", InvocationID: state.InvocationID,
		MainPID: 0, Job: "42", HasPendingJob: true,
	}
	oldQuery := queryOwnedUnit
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) { return pending, nil }
	t.Cleanup(func() { queryOwnedUnit = oldQuery })
	pendingRef, err := LookupOwnedOperation(context.Background(), intent)
	if err != nil {
		t.Fatalf("pending start could not be recovered by persisted intent: %v", err)
	}
	if pendingRef.Scope.Path != want {
		t.Fatalf("pending ref scope = %q, want deterministic slice path %q", pendingRef.Scope.Path, want)
	}
	if empty, err := operationTreeEmpty(pending, pendingRef.Scope.Path); err != nil || empty {
		t.Fatalf("pending ref was mistaken for an empty terminal operation: empty=%t err=%v", empty, err)
	}
}

func TestInspectAndStopRejectMismatchedInvocation(t *testing.T) {
	previousQuery, previousSignal := queryOwnedUnit, signalOwnedUnit
	t.Cleanup(func() { queryOwnedUnit, signalOwnedUnit = previousQuery, previousSignal })
	spec := operationTestSpec()
	intent, err := NewOperationIntent("device-studio", "attempt-mismatch", spec)
	if err != nil {
		t.Fatal(err)
	}
	unit := operationUnitName(intent)
	ref := OperationRef{Intent: intent, Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/user-1000.slice/user@1000.service/vrooli-operations.slice/" + unit}, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) {
		return ownedUnitState{LoadState: "loaded", ActiveState: "active", ControlGroup: ref.Scope.Path, InvocationID: "fedcba9876543210fedcba9876543210"}, nil
	}
	stopCalled := false
	signalOwnedUnit = func(context.Context, string, string) error { stopCalled = true; return nil }
	if _, err := InspectOwnedOperation(context.Background(), ref); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("inspection accepted later invocation: %v", err)
	}
	if _, err := StopOwnedOperation(context.Background(), ref); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("stop accepted later invocation: %v", err)
	}
	if stopCalled {
		t.Fatal("mismatched unit was stopped")
	}
}
