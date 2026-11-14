//go:build linux

package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	runtimeProbeModeEnv  = "PLATFORM_GO_RUNTIME_MAX_MODE"
	runtimeProbeReadyEnv = "PLATFORM_GO_RUNTIME_MAX_READY"
)

// This fixed test-binary fixture is invoked only by the opt-in host probe. It
// avoids a shell or an imported executable/script while still creating a real
// owned descendant tree for systemd to manage.
func TestOwnedOperationRuntimeMaxFixture(t *testing.T) {
	mode := os.Getenv(runtimeProbeModeEnv)
	if mode == "" {
		return
	}
	if mode != "parent-running" && mode != "parent-exits" {
		t.Fatalf("unsupported fixed runtime probe mode %q", mode)
	}
	readyPath := os.Getenv(runtimeProbeReadyEnv)
	if readyPath == "" || !filepath.IsAbs(readyPath) {
		t.Fatal("runtime probe requires an absolute readiness path")
	}

	childReady := readyPath + ".child"
	cmd := exec.Command(os.Args[0], "-test.run=^TestOwnedOperationRuntimeMaxChildFixture$")
	cmd.Env = []string{runtimeProbeModeEnv + "=child", runtimeProbeReadyEnv + "=" + childReady}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fixed child fixture: %v", err)
	}
	childRecord := struct {
		PID int `json:"pid"`
	}{PID: cmd.Process.Pid}
	data, err := json.Marshal(childRecord)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readyPath, data, 0o600); err != nil {
		t.Fatalf("write child identity: %v", err)
	}
	if err := waitForFile(childReady, 3*time.Second); err != nil {
		t.Fatalf("child did not become ready: %v", err)
	}
	if mode == "parent-running" {
		for {
			time.Sleep(time.Second)
		}
	}
	// In parent-exits mode, leave the child in the same manager-owned service
	// and let this main process exit successfully.
}

func TestOwnedOperationRuntimeMaxChildFixture(t *testing.T) {
	if os.Getenv(runtimeProbeModeEnv) != "child" {
		return
	}
	readyPath := os.Getenv(runtimeProbeReadyEnv)
	if readyPath == "" || !filepath.IsAbs(readyPath) {
		t.Fatal("runtime probe child requires an absolute readiness path")
	}
	if err := os.WriteFile(readyPath, []byte("ready\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for {
		time.Sleep(time.Second)
	}
}

// TestOwnedOperationCurrentHostRuntimeMaxProbe is intentionally opt-in. It
// tests the manager's configured deadline on the exact retained transient
// service and only releases units created from its persisted test intents.
func TestOwnedOperationCurrentHostRuntimeMaxProbe(t *testing.T) {
	if os.Getenv("PLATFORM_GO_RUNTIME_MAX_PROBE") != "1" {
		t.Skip("set PLATFORM_GO_RUNTIME_MAX_PROBE=1 for the bounded current-host runtime deadline probe")
	}
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Fatalf("systemd-run unavailable: %v", err)
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		t.Fatalf("systemctl unavailable: %v", err)
	}

	for _, mode := range []string{"parent-running", "parent-exits"} {
		t.Run(mode, func(t *testing.T) {
			probeOperationRuntimeMax(t, mode)
		})
	}
}

type runtimeProbeIntentRecord struct {
	Intent OperationIntent `json:"intent"`
	Spec   struct {
		Path    string          `json:"path"`
		Args    []string        `json:"args"`
		EnvKeys []string        `json:"env_keys"`
		Dir     string          `json:"dir"`
		Limits  OperationLimits `json:"limits"`
	} `json:"spec"`
}

func probeOperationRuntimeMax(t *testing.T, mode string) {
	t.Helper()
	const wallLimit = 2 * time.Second
	work := t.TempDir()
	readyPath := filepath.Join(work, "child.json")
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve fixed test executable: %v", err)
	}
	spec := OperationSpec{
		Path: executable,
		Args: []string{"-test.run=^TestOwnedOperationRuntimeMaxFixture$"},
		Env:  []string{runtimeProbeModeEnv + "=" + mode, runtimeProbeReadyEnv + "=" + readyPath},
		Dir:  work,
		Limits: OperationLimits{
			MemoryMaxBytes: 256 << 20,
			TasksMax:       32,
			CPUWeight:      100,
			WallTime:       wallLimit,
		},
	}
	intent, err := NewOperationIntent("platform-go", "runtime-max-"+mode+"-"+strconv.FormatInt(time.Now().UnixNano(), 10), spec)
	if err != nil {
		t.Fatal(err)
	}
	record := runtimeProbeIntentRecord{Intent: intent}
	record.Spec.Path = spec.Path
	record.Spec.Args = append([]string(nil), spec.Args...)
	record.Spec.EnvKeys = []string{runtimeProbeModeEnv, runtimeProbeReadyEnv}
	record.Spec.Dir = spec.Dir
	record.Spec.Limits = spec.Limits
	intentBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	intentPath := filepath.Join(work, "persisted-intent.json")
	if err := writeSyncedFile(intentPath, append(intentBytes, '\n')); err != nil {
		t.Fatalf("persist intent before dispatch: %v", err)
	}
	var persisted runtimeProbeIntentRecord
	if data, err := os.ReadFile(intentPath); err != nil {
		t.Fatal(err)
	} else if err := json.Unmarshal(data, &persisted); err != nil || persisted.Intent != intent {
		t.Fatalf("pre-dispatch intent did not round-trip: err=%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()
	op, err := StartOwnedOperation(ctx, intent, spec)
	if err != nil {
		// A failed return can still mean manager submission won a race. Reconcile
		// and clean up only this never-reused persisted intent.
		if ref, lookupErr := LookupOwnedOperation(ctx, intent); lookupErr == nil {
			cleanupOwnedRuntimeProbe(ref)
		}
		t.Fatalf("start fixed runtime probe operation: %v", err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			cleanupOwnedRuntimeProbe(op.Ref)
		}
	})

	childPID, err := waitForChildRecord(ctx, readyPath)
	if err != nil {
		t.Fatalf("wait for fixed child identity: %v", err)
	}
	if err := waitForFile(readyPath+".child", 2*time.Second); err != nil {
		t.Fatalf("child fixture did not publish readiness: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var before OperationObservation
	conditionMet := false
	for time.Now().Before(deadline) {
		before, err = InspectOwnedOperation(ctx, op.Ref)
		conditionMet = err == nil && before.Populated
		if conditionMet && mode == "parent-running" {
			conditionMet = before.RootAlive && op.Ref.RootPID > 0
		}
		if conditionMet && mode == "parent-exits" {
			conditionMet = !before.RootAlive
		}
		if conditionMet {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil || !conditionMet {
		t.Fatalf("required pre-deadline tree state for mode %s was not reached (observation=%+v err=%v)", mode, before, err)
	}
	childStamp, childAlive, err := linuxProcessStartTime(childPID)
	if err != nil || !childAlive {
		t.Fatalf("expected recorded child %d alive before deadline (alive=%t err=%v)", childPID, childAlive, err)
	}
	_ = childStamp

	managerLimit, err := readUnitRuntimeMax(ctx, op.Ref.Unit)
	if err != nil {
		t.Fatalf("read exact manager RuntimeMaxUSec: %v", err)
	}
	if managerLimit != wallLimit {
		t.Fatalf("manager retained RuntimeMaxUSec=%s, want %s", managerLimit, wallLimit)
	}
	managerExitType, err := readUnitExitType(ctx, op.Ref.Unit)
	if err != nil {
		t.Fatalf("read exact manager ExitType: %v", err)
	}
	if managerExitType != "cgroup" {
		t.Fatalf("manager retained ExitType=%q, want cgroup", managerExitType)
	}

	started := time.Now()
	waitCtx, waitCancel := context.WithTimeout(ctx, wallLimit+5*time.Second)
	waitErr := op.Wait(waitCtx)
	waitCancel()
	if waitErr != nil {
		observation, inspectErr := InspectOwnedOperation(ctx, op.Ref)
		managerState, managerErr := queryOwnedUnit(ctx, op.Ref.Unit)
		_, childStillAlive, childInspectErr := linuxProcessStartTime(childPID)
		fmt.Printf("runtime-max probe incomplete mode=%s unit=%s invocation=%s child_pid=%d runtime_max=%s exit_type=%s elapsed=%s wait_error=%v observation=%+v inspect_error=%v manager_state=%+v manager_error=%v child_alive=%t child_error=%v\n", mode, op.Ref.Unit, op.Ref.NativeInvocationID, childPID, managerLimit, managerExitType, time.Since(started).Round(time.Millisecond), waitErr, observation, inspectErr, managerState, managerErr, childStillAlive, childInspectErr)
		t.Fatalf("wait for manager-enforced deadline: %v", waitErr)
	}
	elapsed := time.Since(started)
	after, err := InspectOwnedOperation(ctx, op.Ref)
	if err != nil {
		t.Fatalf("inspect terminal deadline result: %v", err)
	}
	_, childAliveAfter, childErr := linuxProcessStartTime(childPID)
	if childErr != nil || childAliveAfter {
		t.Fatalf("deadline left descendant %d alive (alive=%t err=%v)", childPID, childAliveAfter, childErr)
	}
	if !after.ExitKnown || after.Populated || after.RootAlive {
		t.Fatalf("manager deadline did not establish terminal empty tree: %+v", after)
	}
	if after.Result != "timeout" {
		t.Fatalf("manager result=%q, want timeout after RuntimeMaxSec", after.Result)
	}
	if elapsed < wallLimit-1500*time.Millisecond || elapsed > wallLimit+8*time.Second {
		t.Fatalf("deadline wait took %s outside bounded expected window for %s", elapsed, wallLimit)
	}
	if err := ReleaseOwnedOperation(ctx, op.Ref); err != nil {
		t.Fatalf("release exact terminal runtime probe service: %v", err)
	}
	released = true
	fmt.Printf("runtime-max probe mode=%s unit=%s invocation=%s child_pid=%d runtime_max=%s exit_type=%s elapsed=%s result=%s exit_known=%t populated=%t root_alive=%t child_alive_after=%t\n", mode, op.Ref.Unit, op.Ref.NativeInvocationID, childPID, managerLimit, managerExitType, elapsed.Round(time.Millisecond), after.Result, after.ExitKnown, after.Populated, after.RootAlive, childAliveAfter)
}

func readUnitRuntimeMax(ctx context.Context, unit string) (time.Duration, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=RuntimeMaxUSec", "--value", "--no-pager", "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	data, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("systemctl show exact runtime limit: %w: %s", err, strings.TrimSpace(string(data)))
	}
	value := strings.TrimSpace(string(data))
	if value == "" || value == "infinity" {
		return 0, errors.New("manager returned no finite RuntimeMaxUSec")
	}
	if micros, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil {
		return time.Duration(micros) * time.Microsecond, nil
	}
	duration, parseErr := time.ParseDuration(value)
	if parseErr != nil {
		return 0, fmt.Errorf("unrecognized RuntimeMaxUSec %q", value)
	}
	return duration, nil
}

func readUnitExitType(ctx context.Context, unit string) (string, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=ExitType", "--value", "--no-pager", "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("systemctl show exact exit type: %w: %s", err, strings.TrimSpace(string(data)))
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", errors.New("manager returned no ExitType")
	}
	return value, nil
}

func waitForChildRecord(ctx context.Context, path string) (int, error) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			var record struct {
				PID int `json:"pid"`
			}
			if err := json.Unmarshal(data, &record); err != nil || record.PID <= 0 {
				return 0, fmt.Errorf("invalid child identity record %q: %v", string(data), err)
			}
			return record.PID, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return 0, err
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForFile(path string, maximum time.Duration) error {
	deadline := time.NewTimer(maximum)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		select {
		case <-deadline.C:
			return errors.New("readiness file deadline exceeded")
		case <-ticker.C:
		}
	}
}

func writeSyncedFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func cleanupOwnedRuntimeProbe(ref OperationRef) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, err := StopOwnedOperation(ctx, ref); err != nil {
		return
	}
	_ = ReleaseOwnedOperation(ctx, ref)
}
