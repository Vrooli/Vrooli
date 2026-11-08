//go:build linux

package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestServiceScopeManagerObservationHonorsDeadline(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "systemctl"), []byte("#!/bin/sh\nexec /bin/sleep 300\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := queryServiceScopeUnitSystemd(ctx, "vrooli-service-resource-fixture.scope"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("manager deadline cause was lost: %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("manager observation exceeded its bounded deadline")
	}
}

func TestServiceScopeNativeFixture(t *testing.T) {
	root := os.Getenv("PLATFORM_SERVICE_SCOPE_FIXTURE")
	if root == "" {
		return
	}
	signal.Ignore(syscall.SIGTERM)
	if err := os.WriteFile(filepath.Join(root, "ready"), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "spawn")); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	child := exec.Command("/bin/sleep", "300")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "child"), []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestServiceScopeCurrentHostPinnedStopProbe(t *testing.T) {
	if os.Getenv("PLATFORM_GO_SERVICE_SCOPE_PROBE") != "1" {
		t.Skip("set PLATFORM_GO_SERVICE_SCOPE_PROBE=1 for fixed native fixture capture/tree stop")
	}
	root, err := os.MkdirTemp("", "vrooli-service-scope-probe-")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestServiceScopeNativeFixture$")
	cmd.Env = append(os.Environ(), "PLATFORM_SERVICE_SCOPE_FIXTURE="+root)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var captured *ServiceScopeRef
	cleaned := false
	t.Cleanup(func() {
		if cleaned {
			os.RemoveAll(root)
			return
		}
		if captured != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if _, err := StopServiceScope(ctx, *captured); err != nil {
				t.Errorf("native fixture stop unconfirmed; exact receipt retained at %s: %v", root, err)
				return
			}
		} else {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	waitFile := func(name string) []byte {
		for {
			data, err := os.ReadFile(filepath.Join(root, name))
			if err == nil {
				return data
			}
			select {
			case <-ctx.Done():
				t.Fatalf("fixture %s did not arrive", name)
				return nil
			case <-time.After(20 * time.Millisecond):
			}
		}
	}
	_ = waitFile("ready")
	name := "vrooli-service-resource-native-scope-probe-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	scope, method, err := AdoptIntoScope(AdoptSpec{PID: cmd.Process.Pid, Scope: name, Slice: ServicesSlice, Description: "fixed native service-scope qualification fixture"})
	if err != nil || method != MethodSystemdRun {
		t.Fatalf("fixture needs native manager scope: %s %v", method, err)
	}
	stamp, alive, err := linuxProcessStartTime(cmd.Process.Pid)
	if err != nil || !alive {
		t.Fatal("fixture root not alive")
	}
	ref, err := CaptureServiceScope(ctx, scope, cmd.Process.Pid, stamp)
	if err != nil {
		t.Fatal(err)
	}
	captured = &ref
	data, _ := json.MarshalIndent(ref, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "native-ref.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"invocation", "inode", "incarnation"} {
		stale := ref
		switch field {
		case "invocation":
			stale.NativeInvocationID = strings.Repeat("b", 32)
		case "inode":
			stale.ScopeInode++
		case "incarnation":
			stale.RootStartTime += "0"
		}
		if _, err := StopServiceScope(ctx, stale); err == nil {
			t.Fatalf("stale %s stopped the fixture", field)
		}
		if _, stillAlive, err := linuxProcessStartTime(cmd.Process.Pid); err != nil || !stillAlive {
			t.Fatalf("rejected %s disturbed fixture", field)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "spawn"), []byte("spawn child after capture"), 0o600); err != nil {
		t.Fatal(err)
	}
	childText := waitFile("child")
	childPID, err := strconv.Atoi(string(childText))
	if err != nil {
		t.Fatal(err)
	}
	childStamp, childAlive, err := linuxProcessStartTime(childPID)
	if err != nil || !childAlive {
		t.Fatal("fixture child not alive")
	}
	if _, err := CaptureServiceScope(ctx, scope, cmd.Process.Pid, stamp); err == nil {
		t.Fatal("scope with an extra process was admitted as a singleton")
	}
	previous := operationTermGrace
	operationTermGrace = 200 * time.Millisecond
	defer func() { operationTermGrace = previous }()
	receipt, err := StopServiceScope(ctx, ref)
	if err != nil || !receipt.RootGone || !receipt.TreeQuiescent {
		t.Fatalf("pinned scope stop failed: %v", err)
	}
	_ = cmd.Wait()
	if stamp, alive, err := linuxProcessStartTime(childPID); err != nil || alive && stamp == childStamp {
		t.Fatalf("owned child escaped tree stop: %v", err)
	}
	if repeated, err := StopServiceScope(ctx, ref); err != nil || !repeated.TreeQuiescent {
		t.Fatalf("exact stopped receipt did not reconcile after manager GC: %v", err)
	}
	// Reuse the actual native scope name with a different process/invocation.
	// The old receipt must refuse it before signaling the replacement.
	replacementRoot := filepath.Join(root, "replacement")
	if err := os.Mkdir(replacementRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	next := exec.Command(os.Args[0], "-test.run=^TestServiceScopeNativeFixture$")
	next.Env = append(os.Environ(), "PLATFORM_SERVICE_SCOPE_FIXTURE="+replacementRoot)
	if err := next.Start(); err != nil {
		t.Fatal(err)
	}
	cmd = next
	captured = nil
	replacementScope, method, err := AdoptIntoScope(AdoptSpec{PID: next.Process.Pid, Scope: name, Slice: ServicesSlice, Description: "fixed replacement native service-scope fixture"})
	if err != nil || method != MethodSystemdRun {
		t.Fatalf("replacement native scope unavailable: %s %v", method, err)
	}
	stamp, alive, err = linuxProcessStartTime(next.Process.Pid)
	if err != nil || !alive {
		t.Fatal("replacement root not alive")
	}
	replacement, err := CaptureServiceScope(ctx, replacementScope, next.Process.Pid, stamp)
	if err != nil {
		t.Fatal(err)
	}
	captured = &replacement
	if replacement.NativeInvocationID == ref.NativeInvocationID || replacement.Scope.Path != ref.Scope.Path {
		t.Fatal("scope replacement did not exercise a new invocation at the old name")
	}
	if _, err := StopServiceScope(ctx, ref); err == nil {
		t.Fatal("old receipt stopped a recycled scope name")
	}
	if _, alive, err := linuxProcessStartTime(next.Process.Pid); err != nil || !alive {
		t.Fatal("old receipt disturbed the replacement service")
	}
	if receipt, err := StopServiceScope(ctx, replacement); err != nil || !receipt.TreeQuiescent {
		t.Fatalf("replacement scope cleanup failed: %v", err)
	}
	_ = next.Wait()
	cleaned = true
	refs, _ := json.Marshal([]ServiceScopeRef{ref, replacement})
	t.Logf("native scope receipts: %s", refs)
	t.Log(fmt.Sprintf("captured native invocation %s; stale receipts refused; held cgroup stopped root and later child; repeated stop reconciled exact empty scope; recycled name %s refused", ref.NativeInvocationID, replacement.NativeInvocationID))
}
