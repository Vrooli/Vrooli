//go:build linux

package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/recovery"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestNativeCodingRecoveryActualRootAdapterHostProbe(t *testing.T) {
	image := os.Getenv("VROOLI_NATIVE_CODING_ROOT_IMAGE")
	if image == "" {
		t.Skip("set VROOLI_NATIVE_CODING_ROOT_IMAGE to a governed root image for the isolated native adapter probe")
	}
	home, err := os.MkdirTemp("", "vrooli-native-recovery-home-")
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp("", "vrooli-native-recovery-source-")
	if err != nil {
		_ = os.RemoveAll(home)
		t.Fatal(err)
	}
	retirementConfirmed := true
	t.Cleanup(func() {
		if retirementConfirmed {
			_ = os.RemoveAll(home)
			_ = os.RemoveAll(root)
		} else {
			t.Logf("unresolved native fixture retained for exact owner recovery: home=%s root=%s", home, root)
		}
	})
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("AGENT_MANAGER_API_BASE", "http://127.0.0.1:1")
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "recovery-fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	const private = "fixture-child-diagnostic-must-not-enter-receipt"
	killWorker := os.Getenv("VROOLI_NATIVE_CODING_ROOT_FAULT") == "kill_worker"
	delay := "2"
	if killWorker {
		delay = "60"
	}
	body := "#!/bin/sh\nprintf '%s\\n' '" + private + "'\nprintf '%s' \"$$\" > fixture-started\n/bin/sleep " + delay + "\nprintf healthy > fixture-state\n"
	if err := os.WriteFile(filepath.Join(bin, "opencode"), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	shimOnly := os.Getenv("VROOLI_NATIVE_CODING_ROOT_FAULT") == "first_missing"
	if shimOnly {
		managedBin := filepath.Join(home, ".vrooli", "bin")
		if err := os.MkdirAll(managedBin, 0o700); err != nil {
			t.Fatal(err)
		}
		launcher := filepath.Join(managedBin, "vrooli-agent-launcher")
		if err := os.WriteFile(launcher, []byte("#!/bin/sh\nprintf unexpected > shim-executed\nexit 9\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(bin, "opencode")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(launcher, filepath.Join(bin, "opencode")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bin, "codex"), []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := cliutil.ResolveAgentBinaryExcludingInEnvironment("opencode", launcher, []string{"PATH=" + platform.DefaultPath("linux", home)}); err == nil {
			t.Skip("host has another real OpenCode installation; cannot isolate pre-launch fallback without a model")
		}
	}
	if err := os.WriteFile(filepath.Join(root, "fixture-state"), []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := recovery.DefaultCodingAgentRecordPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, home+string(filepath.Separator)) {
		t.Fatal("fixture ledger escaped private home")
	}
	b, err := recovery.NewNativeCodingAgentBroker(path, image)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		retirementConfirmed = false
		// Cleanup uses the same owner; deadline expiry is not fabricated exit
		// evidence. The exact native tree must be excluded before retirement.
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		executions, err := b.NativeExecutions()
		if err != nil {
			t.Error(err)
			return
		}
		for _, e := range executions {
			if !e.ExitProven {
				if err := b.RevokeNativeExecution(ctx, e.ID); err != nil {
					t.Error(err)
				}
			}
		}
		if _, err := b.ReconcileNative(ctx, 32); err != nil {
			t.Error(err)
		}
		executions, err = b.NativeExecutions()
		if err != nil {
			t.Error(err)
			return
		}
		for _, e := range executions {
			if !e.ExitProven {
				t.Errorf("retained native fixture requires owner recovery: %s", e.ID)
			}
		}
		retirementConfirmed = true
		for _, e := range executions {
			if !e.ExitProven || !e.Retired {
				retirementConfirmed = false
			}
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r, err := b.SubmitContext(ctx, recovery.CodingAgentRequest{Scenario: "recovery-fixture", Reason: "isolated deterministic worker adapter qualification", Requester: "native-fixture", WorkingDir: root, Claim: filepath.Join("scenarios", "recovery-fixture")})
	if err != nil {
		t.Fatal(err)
	}
	if killWorker {
		deadline := time.Now().Add(12 * time.Second)
		for {
			if _, err := os.Stat(filepath.Join(root, "fixture-started")); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("private runner did not start")
			}
			time.Sleep(40 * time.Millisecond)
		}
		executions, err := b.NativeExecutions()
		if err != nil || len(executions) != 1 || executions[0].Ref == nil {
			t.Fatalf("worker native identity unavailable: %+v %v", executions, err)
		}
		e := executions[0]
		ref := *e.Ref
		handle, err := process.OpenHandle(ref.RootPID)
		if err != nil {
			t.Fatal(err)
		}
		defer handle.Close()
		stamp, err := process.Incarnation(ref.RootPID)
		if err != nil || stamp != e.BootID+":"+ref.RootStartTime {
			t.Fatal("worker incarnation changed")
		}
		sha, err := coreactivation.RunningImageDigest(ctx, ref.RootPID)
		if err != nil || sha != e.WorkerSHA {
			t.Fatal("worker running image changed")
		}
		observation, err := platform.InspectOwnedOperation(ctx, ref)
		if err != nil || !observation.RootAlive || !observation.Populated {
			t.Fatal("worker is not the live native root")
		}
		childBytes, err := os.ReadFile(filepath.Join(root, "fixture-started"))
		if err != nil {
			t.Fatal(err)
		}
		childPID, err := strconv.Atoi(string(childBytes))
		if err != nil {
			t.Fatal(err)
		}
		members, err := platform.ScopeProcessesContext(ctx, ref.Scope)
		if err != nil {
			t.Fatal(err)
		}
		childBound := false
		for _, pid := range members {
			if pid == childPID {
				childBound = true
			}
		}
		if !childBound {
			t.Fatal("runner escaped the bounded native tree")
		}
		if alive, err := handle.Alive(); err != nil || !alive {
			t.Fatal("pinned worker is not live")
		}
		if err := handle.Signal(true); err != nil {
			t.Fatal(err)
		}
		if _, err := b.ReconcileNative(ctx, 32); err != nil {
			t.Fatal(err)
		}
		closed, err := b.NativeExecutions()
		if err != nil || len(closed) != 1 || !closed[0].ExitProven || !closed[0].Retired {
			t.Fatalf("native descendant exclusion was not proved: %+v %v", closed, err)
		}
		records, err := b.Snapshot()
		if err != nil || records[0].Outcome != "failed" || records[0].FinishedAt.IsZero() {
			t.Fatalf("worker loss became a false completion: %+v %v", records, err)
		}
		state, err := os.ReadFile(filepath.Join(root, "fixture-state"))
		if err != nil || string(state) != "broken" {
			t.Fatal("interrupted fixture was marked repaired")
		}
		store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		lease, err := store.GetEditorLease(ctx, e.LeaseID)
		if err != nil || lease.Scope != ref.Scope.String() || lease.PID != ref.RootPID {
			t.Fatalf("lease did not retain native custody: %+v %v", lease, err)
		}
		// Advance only the fixture's expiry cutoff. Death is still proved by
		// the real native stop and the actual host's process reader.
		expired, err := store.ExpireStaleEditorLeases(ctx, time.Now().Add(2*time.Minute), scenarioruntime.StartingLeaseGuard{CurrentBootID: e.BootID, PIDRunning: platform.IsPIDRunning})
		if err != nil || len(expired) != 1 || expired[0].SessionID != e.LeaseID {
			t.Fatalf("existing lease owner could not retire the killed worker: %+v %v", expired, err)
		}
		t.Logf("exact native worker SIGKILL excluded its runner tree and closed request %s as failed; existing registry owner expired its bound lease using real death proof and an accelerated private cutoff", r.ID)
		return
	}
	deadline := time.Now().Add(12 * time.Second)
	for {
		records, err := b.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if len(records) == 1 && records[0].Outcome == "succeeded" {
			if shimOnly && records[0].Runner != "codex" {
				t.Fatal("missing real runner did not fall back to Codex")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("actual root worker did not finish: %+v", records)
		}
		time.Sleep(40 * time.Millisecond)
	}
	executions, err := b.NativeExecutions()
	if err != nil || len(executions) != 1 {
		t.Fatalf("native execution unavailable: %+v %v", executions, err)
	}
	e := executions[0]
	if e.Ref == nil || e.LeaseID == "" || !e.LeaseRegistered {
		t.Fatalf("actual root worker did not link its editor lease: %+v", e)
	}
	op := platform.OwnedOperation{Ref: *e.Ref}
	if err := op.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReconcileNative(ctx, 32); err != nil {
		t.Fatal(err)
	}
	state, err := os.ReadFile(filepath.Join(root, "fixture-state"))
	if err != nil || string(state) != "healthy" {
		t.Fatal("fixture repair was not written")
	}
	if shimOnly {
		if _, err := os.Stat(filepath.Join(root, "shim-executed")); !os.IsNotExist(err) {
			t.Fatal("managed shim ran before pre-launch unavailability was established")
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), private) {
		t.Fatal("raw child diagnostic escaped into recovery receipt")
	}
	executions, err = b.NativeExecutions()
	if err != nil || !executions[0].ExitProven || !executions[0].Retired {
		t.Fatalf("actual root job retirement unconfirmed: %+v %v", executions, err)
	}
	t.Logf("actual frozen root adapter request %s bound lease %s, ran a private headless fixture, preserved diagnostics and retired exact native inputs; no model or scenario API was called", r.ID, e.LeaseID)
}
