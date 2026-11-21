//go:build linux

package platform

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type gatedProbe struct {
	Intent OperationIntent
	Spec   OperationSpec
}

func gateProbeSpec(root string) OperationSpec {
	return OperationSpec{Path: "/usr/bin/touch", Args: []string{filepath.Join(root, "effect")}, Env: []string{"PATH=/usr/bin:/bin"}, Dir: root, Limits: OperationLimits{MemoryMaxBytes: 64 << 20, TasksMax: 32, CPUWeight: 50, WallTime: 10 * time.Second}}
}
func requireGateProbe(t *testing.T) {
	t.Helper()
	if os.Getenv("PLATFORM_GO_GATE_PROBE") != "1" {
		t.Skip("set PLATFORM_GO_GATE_PROBE=1 for exact native gate drills")
	}
}
func TestOperationGateOwnerChild(t *testing.T) {
	if os.Getenv("PLATFORM_GO_GATE_CHILD") != "1" {
		return
	}
	root := os.Getenv("PLATFORM_GO_GATE_ROOT")
	spec := gateProbeSpec(root)
	gate, err := PrepareOperationGate(root, "late-admission")
	if err != nil {
		t.Fatal(err)
	}
	spec.Gate = &gate
	intent, err := NewOperationIntent("gate-probe", "late-owner", spec)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(gatedProbe{intent, spec})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "prepared.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	select {}
}
func TestOperationGateCurrentHostLateAdmissionAfterOwnerDeath(t *testing.T) {
	requireGateProbe(t)
	root := privateGateDir(t)
	child := exec.Command(os.Args[0], "-test.run=^TestOperationGateOwnerChild$", "-test.timeout=30s")
	child.Env = append(os.Environ(), "PLATFORM_GO_GATE_CHILD=1", "PLATFORM_GO_GATE_ROOT="+root)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	var probe gatedProbe
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(filepath.Join(root, "prepared.json"))
		if err == nil && json.Unmarshal(raw, &probe) == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if probe.Spec.Gate == nil {
		t.Fatal("owner did not durably prepare dispatch")
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	if err := probe.Spec.Gate.Revoke(); err != nil {
		t.Fatal(err)
	}
	if revoked, err := probe.Spec.Gate.Revoked(); err != nil || !revoked {
		t.Fatalf("gate revocation not durable: %t %v", revoked, err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	if absent, err := OwnedOperationIntentAbsent(ctx, probe.Intent); err != nil || !absent {
		t.Fatalf("pre-dispatch complete absence not proven: %t %v", absent, err)
	}
	// Deliver the already-frozen native submission after its owner is dead and
	// cleanup has observed absence. The manager's pre-exec gate must deny it.
	operation, err := StartOwnedOperation(ctx, probe.Intent, probe.Spec)
	if operation != nil {
		defer func() {
			_, _ = operation.Stop(context.Background())
			_ = ReleaseOwnedOperation(context.Background(), operation.Ref)
		}()
		if waitErr := operation.Wait(ctx); waitErr != nil {
			t.Fatal(waitErr)
		}
		observation, inspectErr := InspectOwnedOperation(ctx, operation.Ref)
		if inspectErr != nil {
			t.Fatal(inspectErr)
		}
		if observation.ExecMainCode == 1 && observation.ExecMainStatus == 0 {
			t.Fatalf("denied condition became successful executable evidence: %+v", observation)
		}
	} else if err == nil {
		t.Fatal("native dispatch returned neither handle nor refusal")
	}
	if _, err := os.Stat(filepath.Join(root, "effect")); !os.IsNotExist(err) {
		t.Fatalf("late admission executed after durable revocation: %v", err)
	}
}
func TestOperationGateCurrentHostStopAfterConditionBeforeExec(t *testing.T) {
	requireGateProbe(t)
	root := privateGateDir(t)
	spec := gateProbeSpec(root)
	gate, err := PrepareOperationGate(root, "condition-passed")
	if err != nil {
		t.Fatal(err)
	}
	spec.Gate = &gate
	intent, err := NewOperationIntent("gate-probe", "condition-passed", spec)
	if err != nil {
		t.Fatal(err)
	}
	command, err := ownedServiceCommand(intent, spec)
	if err != nil {
		t.Fatal(err)
	}
	// Test-only native staging holds the actual manager between its successful
	// condition and ExecStart. Production offers no arbitrary pre-exec command.
	for i, arg := range command.Args {
		if arg == "--" {
			command.Args = append(command.Args[:i], append([]string{"--property=ExecStartPre=/usr/bin/sleep 30"}, command.Args[i:]...)...)
			break
		}
	}
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	deadline := time.Now().Add(5 * time.Second)
	var ref OperationRef
	for time.Now().Before(deadline) {
		state, err := queryOwnedUnit(ctx, operationUnitName(intent))
		if err == nil && state.SubState == "start-pre" {
			ref, err = LookupOwnedOperation(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ref.Unit == "" {
		t.Fatal("native condition-passed staging was not observed")
	}
	defer func() {
		_, _ = StopOwnedOperation(context.Background(), ref)
		_ = ReleaseOwnedOperation(context.Background(), ref)
	}()
	if err := gate.Revoke(); err != nil {
		t.Fatal(err)
	}
	if absent, err := OwnedOperationIntentAbsent(ctx, intent); err != nil || absent {
		t.Fatalf("pending accepted condition became absence: %t %v", absent, err)
	}
	receipt, err := StopOwnedOperation(ctx, ref)
	if err != nil || !receipt.NativeQuiescent {
		t.Fatalf("pending native tree was not stopped: %+v %v", receipt, err)
	}
	if _, err := os.Stat(filepath.Join(root, "effect")); !os.IsNotExist(err) {
		t.Fatalf("ExecStart raced cleanup: %v", err)
	}
}
func TestOperationGateNativeEscapingUsesLiteralSpecifierAndSpacePath(t *testing.T) {
	root := filepath.Join(privateGateDir(t), "space %h \"quote\"")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	gate, err := PrepareOperationGate(root, "escaping")
	if err != nil {
		t.Fatal(err)
	}
	spec := gateProbeSpec(root)
	spec.Gate = &gate
	intent, err := NewOperationIntent("gate-probe", "escaping", spec)
	if err != nil {
		t.Fatal(err)
	}
	command, err := ownedServiceCommand(intent, spec)
	if err != nil {
		t.Fatal(err)
	}
	want := "--property=ExecCondition=/usr/bin/test -f " + systemdExecArgument(gate.ReadyPath)
	found := false
	for _, arg := range command.Args {
		if arg == want {
			found = true
		}
	}
	if !found {
		t.Fatal("native literal ExecCondition argument was not retained")
	}
	if os.Getenv("PLATFORM_GO_GATE_PROBE") == "1" {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		operation, err := StartOwnedOperation(ctx, intent, spec)
		if err != nil {
			state, _ := queryOwnedUnit(ctx, operationUnitName(intent))
			t.Logf("native literal-path state: %+v", state)
			logs, _ := exec.Command("journalctl", "--user", "--unit="+operationUnitName(intent), "--no-pager", "-n", "10").CombinedOutput()
			t.Logf("native literal-path journal: %s", logs)
			if ref, lookupErr := LookupOwnedOperation(ctx, intent); lookupErr == nil {
				_, _ = StopOwnedOperation(context.Background(), ref)
				_ = ReleaseOwnedOperation(context.Background(), ref)
			}
			t.Fatal(err)
		}
		defer func() {
			_ = gate.Revoke()
			_, _ = operation.Stop(context.Background())
			_ = ReleaseOwnedOperation(context.Background(), operation.Ref)
		}()
		if err := operation.Wait(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(root, "effect")); err != nil {
			t.Fatalf("native escaping changed the literal gate/command path: %v", err)
		}
	}
}

func TestOperationGateConcurrentPreparingChild(t *testing.T) {
	if os.Getenv("PLATFORM_GO_GATE_RACE_CHILD") != "1" {
		return
	}
	root := os.Getenv("PLATFORM_GO_GATE_ROOT")
	_, err := prepareOperationGate(root, "cross-process", func() {
		if err := os.WriteFile(filepath.Join(root, "checked"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(filepath.Join(root, "release")); err == nil {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("preparation barrier was not released")
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestOperationGateRevokeSerializesAcrossPreparingProcess(t *testing.T) {
	root := privateGateDir(t)
	child := exec.Command(os.Args[0], "-test.run=^TestOperationGateConcurrentPreparingChild$", "-test.timeout=15s")
	child.Env = append(os.Environ(), "PLATFORM_GO_GATE_RACE_CHILD=1", "PLATFORM_GO_GATE_ROOT="+root)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	deadline := time.Now().Add(5 * time.Second)
	checked := false
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(root, "checked")); err == nil {
			checked = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !checked {
		t.Fatal("preparing process did not reach post-check barrier")
	}
	gate := OperationGate{ReadyPath: filepath.Join(root, "cross-process.ready")}
	revoked := make(chan error, 1)
	go func() { revoked <- gate.Revoke() }()
	early := false
	var revokeErr error
	select {
	case revokeErr = <-revoked:
		early = true
	case <-time.After(50 * time.Millisecond):
	}
	if err := os.WriteFile(filepath.Join(root, "release"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err != nil {
		t.Fatal(err)
	}
	if !early {
		revokeErr = <-revoked
	}
	if early {
		t.Fatal("revocation returned before in-flight cross-process preparation was fenced")
	}
	if revokeErr != nil {
		t.Fatal(revokeErr)
	}
	if closed, err := gate.Revoked(); err != nil || !closed {
		t.Fatalf("stale preparing process revived revoked gate: %t %v", closed, err)
	}
}
