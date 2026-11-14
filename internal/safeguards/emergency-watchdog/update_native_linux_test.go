//go:build linux

package emergencywatchdog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

func TestWatchdogPublicationCallerFixture(t *testing.T) {
	if os.Getenv("VROOLI_WATCHDOG_OWNER_FIXTURE") != "1" {
		return
	}
	home := os.Getenv("VROOLI_WATCHDOG_HOME_FIXTURE")
	p := resolvePaths(home)
	resolveWatchdogRootFn = func() (string, error) { return os.Getenv("VROOLI_SOURCE_ROOT"), nil }
	buildWatchdogFn = func(_, path string) error {
		sha, err := watchdogImageDigest(os.Getenv("VROOLI_WATCHDOG_CANDIDATE_FIXTURE"))
		if err != nil {
			return err
		}
		return copyWatchdogImage(os.Getenv("VROOLI_WATCHDOG_CANDIDATE_FIXTURE"), path, sha)
	}
	publishWatchdogImage = func(src, dst, sha string) error {
		if err := copyWatchdogImage(src, dst, sha); err != nil {
			return err
		}
		if err := config.WriteOwnedFileDurable(filepath.Join(home, "caller-published"), []byte("publication visible; caller has not recorded canary outcome"), 0o600); err != nil {
			return err
		}
		for {
			time.Sleep(time.Second)
		}
	}
	if err := buildAndInstallWatchdog(p); err != nil {
		t.Fatal(err)
	}
}

// This fixture runs the actual installed CLI's supervisor in an isolated
// home/state database and a separately owned native service. The receiving
// connector, host watchdog, fleet database and production units are untouched.
func TestWatchdogCurrentHostIndependentNativeRecoveryProbe(t *testing.T) {
	if os.Getenv("VROOLI_WATCHDOG_NATIVE_UPDATE_PROBE") != "1" {
		t.Skip("set VROOLI_WATCHDOG_NATIVE_UPDATE_PROBE=1 for the isolated native recovery proof")
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := expectedWatchdogVersion(root)
	if err != nil {
		t.Fatal(err)
	}
	actualHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	work, err := os.MkdirTemp("", "vrooli-watchdog-native-recovery-")
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(work, "home")
	p := resolvePaths(home)
	if err := os.MkdirAll(filepath.Dir(p.Binary), 0o700); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	copyFixture := func(destination string, marker byte) {
		t.Helper()
		source, err := os.Open(executable)
		if err != nil {
			t.Fatal(err)
		}
		defer source.Close()
		if err := config.WriteOwnedFileDurableStream(destination, 0o700, func(w io.Writer) error {
			if _, err := io.Copy(w, source); err != nil {
				return err
			}
			_, err := w.Write([]byte{marker})
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	copyFixture(p.Binary, 0xa1)
	candidate := filepath.Join(work, "candidate-watchdog")
	copyFixture(candidate, 0xa2)
	installedCLI := filepath.Join(actualHome, ".vrooli", "bin", "vrooli")
	fixtureCLI := filepath.Join(home, ".vrooli", "bin", "vrooli")
	cliSHA, err := watchdogImageDigest(installedCLI)
	if err != nil {
		t.Fatal(err)
	}
	if err := copyWatchdogImage(installedCLI, fixtureCLI, cliSHA); err != nil {
		t.Fatal(err)
	}
	commonEnv := []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "XDG_STATE_HOME=" + filepath.Join(home, "xdg-state"), "XDG_RUNTIME_DIR=" + os.Getenv("XDG_RUNTIME_DIR"), "DBUS_SESSION_BUS_ADDRESS=" + os.Getenv("DBUS_SESSION_BUS_ADDRESS"), "VROOLI_SOURCE_ROOT=" + root, "VROOLI_ROOT=" + root, "VROOLI_WATCHDOG_VERSION_FIXTURE=publication", "VROOLI_WATCHDOG_WANT_STAMP=" + want, "VROOLI_WATCHDOG_CANONICAL_FIXTURE_PATH=" + p.Binary}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	caller := exec.CommandContext(ctx, executable, "-test.run=^TestWatchdogPublicationCallerFixture$", "-test.v")
	caller.Env = append(append([]string(nil), commonEnv...), "VROOLI_WATCHDOG_OWNER_FIXTURE=1", "VROOLI_WATCHDOG_HOME_FIXTURE="+home, "VROOLI_WATCHDOG_CANDIDATE_FIXTURE="+candidate)
	caller.WaitDelay = time.Second
	callerLog, err := os.Create(filepath.Join(work, "caller.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer callerLog.Close()
	caller.Stdout = callerLog
	caller.Stderr = callerLog
	if err := caller.Start(); err != nil {
		t.Fatal(err)
	}
	callerReaped := false
	t.Cleanup(func() {
		if !callerReaped {
			_ = caller.Process.Kill()
			_ = caller.Wait()
		}
	})
	await := func(description string, ready func() bool) {
		t.Helper()
		timer := time.NewTimer(20 * time.Second)
		defer timer.Stop()
		tick := time.NewTicker(50 * time.Millisecond)
		defer tick.Stop()
		for {
			if ready() {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatalf("%s: %v; retained workspace %s", description, ctx.Err(), work)
			case <-timer.C:
				t.Fatalf("%s timed out; retained workspace %s", description, work)
			case <-tick.C:
			}
		}
	}
	await("caller publication checkpoint", func() bool { _, err := os.Stat(filepath.Join(home, "caller-published")); return err == nil })
	if err := caller.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := caller.Wait(); err == nil {
		t.Fatal("caller did not die at the publication checkpoint")
	}
	callerReaped = true
	requireWatchdogPhase(t, p, "publishing")
	// Only a new native supervisor can now finish the admitted publication.
	spec := platform.OwnedServiceSpec{Path: fixtureCLI, Args: []string{"runtime", "supervisor", "run"}, Dir: root, Env: append(commonEnv, "VROOLI_RUNTIME_SUPERVISOR=on", "VROOLI_RUNTIME_SUPERVISOR_RENEW_INTERVAL=1s"), Limits: platform.ServiceLimits{MemoryMaxBytes: 512 << 20, TasksMax: 128, CPUWeight: 100}}
	intent, err := platform.NewServiceIntent("watchdog-update-test", fmt.Sprintf("probe-%d", time.Now().UnixNano()), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 20*time.Second)
		defer stop()
		ref, err := platform.LookupOwnedService(cleanup, intent)
		if err != nil {
			if absent, e := platform.OwnedServiceIntentAbsent(cleanup, intent); e != nil || !absent {
				t.Errorf("fixture cleanup ownership unresolved: %v %v; retained %s", err, e, work)
				return
			}
		} else {
			receipt, e := platform.StopOwnedService(cleanup, ref)
			if e != nil || !receipt.NativeQuiescent {
				t.Errorf("fixture native tree remains unresolved: %v; retained %s", e, work)
				return
			}
			if e := platform.ReleaseOwnedService(cleanup, ref); e != nil {
				t.Errorf("fixture native release failed: %v; retained %s", e, work)
				return
			}
		}
		if !t.Failed() {
			if err := os.RemoveAll(work); err != nil {
				t.Error(err)
			}
		}
	})
	if _, err := platform.StartOwnedService(ctx, intent, spec); err != nil {
		t.Fatalf("start isolated native supervisor: %v; retained %s", err, work)
	}
	await("independent native restoration receipt", func() bool {
		r, err := readWatchdogUpdateReceipt(p)
		return err == nil && r != nil && strings.HasPrefix(r.Phase, "failed_")
	})
	requireWatchdogPhase(t, p, "failed_restored")
	actual, err := watchdogImageDigest(p.Binary)
	if err != nil {
		t.Fatal(err)
	}
	prior, err := watchdogImageDigest(p.Binary + ".previous")
	if err != nil || actual != prior {
		t.Fatalf("native owner did not restore exact retained image: %v", err)
	}
	ref, err := platform.LookupOwnedService(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	native, err := platform.InspectOwnedService(ctx, ref)
	if err != nil || !native.RootAlive || !native.Populated {
		t.Fatalf("independent native owner unavailable: %+v %v", native, err)
	}
	if _, err := os.Stat(filepath.Join(actualHome, ".vrooli", "libexec", "vrooli-watchdog")); errors.Is(err, os.ErrNotExist) {
		t.Fatal("production watchdog disappeared during isolated fixture")
	}
	t.Logf("caller SIGKILL at durable publication checkpoint; fresh native supervisor restored exact prior image and recorded failed_restored; fixture native identity %s/%s; workspace %s", ref.Unit, ref.NativeInvocationID, work)
}
