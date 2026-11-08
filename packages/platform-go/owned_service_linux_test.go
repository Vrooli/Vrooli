//go:build linux

package platform

import (
	"bytes"
	"context"
	"encoding/json"
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
	serviceDispatchInputEnv = "PLATFORM_GO_SERVICE_DISPATCH_INPUT"
	serviceDispatchWaitEnv  = "PLATFORM_GO_SERVICE_DISPATCH_WAIT"
)

type serviceDispatchFixtureInput struct {
	Intent ServiceIntent
	Spec   OwnedServiceSpec
}

// A short-lived dispatcher exits while its native service and child remain.
// This is a fixed test fixture, not a scenario lifecycle alternative.
func TestOwnedServiceDispatchExitFixture(t *testing.T) {
	path := os.Getenv(serviceDispatchInputEnv)
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var input serviceDispatchFixtureInput
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, err := StartOwnedService(ctx, input.Intent, input.Spec); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(serviceDispatchWaitEnv) == "1" {
		if err := writeSyncedFile(path+".started", []byte("native dispatch accepted; domain acknowledgement not recorded")); err != nil {
			t.Fatal(err)
		}
		for {
			time.Sleep(time.Second)
		}
	}
}

func TestOwnedServiceCurrentHostCallerExitRecoveryProbe(t *testing.T) {
	if os.Getenv("PLATFORM_GO_SERVICE_PROBE") != "1" {
		t.Skip("set PLATFORM_GO_SERVICE_PROBE=1 for the exact-owned native service recovery probe")
	}
	for _, killed := range []bool{false, true} {
		t.Run(fmt.Sprintf("dispatcher_killed_%t", killed), func(t *testing.T) { probeOwnedServiceCallerRecovery(t, killed) })
	}
}

func probeOwnedServiceCallerRecovery(t *testing.T, killed bool) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// Failed native cleanup must retain the recovery intent. t.TempDir would
	// delete it even when the service manager could not establish quiescence.
	work, err := os.MkdirTemp("", "vrooli-owned-service-recovery-")
	if err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(work, "child.json")
	spec := OwnedServiceSpec{
		Path: executable, Dir: work, Args: []string{"-test.run=^TestOwnedOperationRuntimeMaxFixture$"},
		Env:    []string{runtimeProbeModeEnv + "=parent-running", runtimeProbeReadyEnv + "=" + ready},
		Limits: ServiceLimits{MemoryMaxBytes: 512 << 20, TasksMax: 64, CPUWeight: 100},
	}
	intent, err := NewServiceIntent("platform-go", "service-recovery-"+strconv.FormatInt(time.Now().UnixNano(), 10), spec)
	if err != nil {
		t.Fatal(err)
	}
	inputPath := filepath.Join(work, "persisted-service-intent.json")
	data, err := json.Marshal(serviceDispatchFixtureInput{Intent: intent, Spec: spec})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSyncedFile(inputPath, data); err != nil {
		t.Fatal(err)
	}
	// Register cleanup before dispatch: even an unsuccessful acknowledgement
	// can have created the exact persisted native service.
	cleaned := false
	t.Cleanup(func() {
		if cleaned || cleanupOwnedServiceProbe(t, intent) {
			if err := os.RemoveAll(work); err != nil {
				t.Errorf("remove verified-quiescent fixture workspace: %v", err)
			}
		} else {
			t.Logf("Retained exact fixture intent for recovery: %s", inputPath)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	dispatch := exec.CommandContext(ctx, executable, "-test.run=^TestOwnedServiceDispatchExitFixture$")
	dispatch.Env = []string{"PATH=" + os.Getenv("PATH"), serviceDispatchInputEnv + "=" + inputPath}
	dispatch.WaitDelay = time.Second
	if killed {
		dispatch.Env = append(dispatch.Env, serviceDispatchWaitEnv+"=1")
		var output bytes.Buffer
		dispatch.Stdout, dispatch.Stderr = &output, &output
		if err := dispatch.Start(); err != nil {
			t.Fatal(err)
		}
		readyErr := waitForFile(inputPath+".started", 6*time.Second)
		killErr := dispatch.Process.Kill()
		waitErr := dispatch.Wait() // Reap even when the handshake failed.
		if readyErr != nil || killErr != nil || waitErr == nil {
			t.Fatalf("caller kill fixture failed: ready=%v kill=%v wait=%v output=%s", readyErr, killErr, waitErr, output.String())
		}
	} else if output, err := dispatch.CombinedOutput(); err != nil {
		t.Fatalf("fixed service dispatcher failed: %v: %s", err, output)
	}
	// The dispatcher is now reaped. Recover using only its persisted intent.
	ref, err := LookupOwnedService(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	childPID, err := waitForChildRecord(ctx, ready)
	if err != nil {
		t.Fatal(err)
	}
	if err := waitForFile(ready+".child", 2*time.Second); err != nil {
		t.Fatal(err)
	}
	childStart, alive, err := linuxProcessStartTime(childPID)
	if err != nil || !alive {
		t.Fatalf("owned service child did not survive dispatcher exit: %v", err)
	}
	observation, err := InspectOwnedService(ctx, ref)
	if err != nil || !observation.RootAlive || !observation.Populated || observation.State != OperationRunning || ref.RootPID <= 0 {
		t.Fatalf("native identity did not survive caller exit: %+v %v", observation, err)
	}
	property := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=RuntimeMaxUSec,Restart", "--no-pager", "--", ref.Unit)
	property.Env = userManagerEnv(nil)
	property.WaitDelay = time.Second
	value, err := property.CombinedOutput()
	properties := map[string]string{}
	for _, line := range strings.Split(string(value), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			properties[key] = value
		}
	}
	if err != nil || properties["RuntimeMaxUSec"] != "infinity" || properties["Restart"] != "no" {
		t.Fatalf("serving process acquired a hidden deadline or native restart: %q %v", value, err)
	}
	if exitType, err := readUnitExitType(ctx, ref.Unit); err != nil || exitType != "cgroup" {
		t.Fatalf("descendants did not define service lifetime: %q %v", exitType, err)
	}
	if _, err := StartOwnedService(ctx, intent, spec); err != ErrOperationExists {
		t.Fatalf("same persisted service intent was redispatched: %v", err)
	}
	stop, err := StopOwnedService(ctx, ref)
	if err != nil || !stop.NativeQuiescent {
		t.Fatalf("owned service tree stop was not proven: %+v %v", stop, err)
	}
	if after, stillAlive, err := linuxProcessStartTime(childPID); err != nil || stillAlive && after == childStart {
		t.Fatalf("owned descendant survived verified service stop: %v", err)
	}
	if err := ReleaseOwnedService(ctx, ref); err != nil {
		t.Fatal(err)
	}
	state, err := queryOwnedUnit(ctx, ref.Unit)
	if err != nil || state.LoadState != "not-found" || state.HasPendingJob {
		t.Fatalf("released fixture unit remained dispatchable: %+v %v", state, err)
	}
	cleaned = true
	t.Logf("exact native service survived dispatcher exit, rejected redispatch, retained infinite runtime, and stopped/released its owned descendant tree: %s", ref.Unit)
}

func cleanupOwnedServiceProbe(t *testing.T, intent ServiceIntent) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 16*time.Second)
	defer cancel()
	ref, err := LookupOwnedService(ctx, intent)
	if err != nil {
		state, queryErr := queryOwnedUnit(ctx, operationUnitName(OperationIntent(intent)))
		if queryErr == nil && state.LoadState == "not-found" && !state.HasPendingJob {
			return true
		}
		t.Errorf("fixture cleanup cannot establish exact service ownership: %v; native query: %v", err, queryErr)
		return false
	}
	if receipt, err := StopOwnedService(ctx, ref); err != nil || !receipt.NativeQuiescent {
		t.Errorf("fixture service cleanup did not prove quiescence: %v", err)
		return false
	}
	if err := ReleaseOwnedService(ctx, ref); err != nil {
		t.Errorf("fixture service release: %v", err)
		return false
	}
	return true
}
