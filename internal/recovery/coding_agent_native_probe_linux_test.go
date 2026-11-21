//go:build linux

package recovery

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/process"
	"strings"
)

// The isolated native fixture exercises the production domain/native fence,
// not a provider model or the root command adapter. It writes only private
// fixture files and never contacts Agent Manager or another scenario API.
func TestMain(m *testing.M) {
	if len(os.Args) == 4 && os.Args[1] == "--coding-agent-recovery-execute" {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		b, err := NewCodingAgentBroker(os.Args[3], nil)
		if err != nil {
			os.Exit(2)
		}
		e, err := b.ClaimNativeExecution(ctx, os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
		err = b.RunNativeExecution(ctx, e, func(ctx context.Context, runner string, req CodingAgentRequest) error {
			if err := b.BindNativeLease(ctx, e.ID, "native-fixture-session", false); err != nil {
				return err
			}
			if err := b.GuardNativeChild(ctx, e.ID); err != nil {
				return err
			}
			child := exec.CommandContext(ctx, "/bin/sleep", "2")
			if err := child.Start(); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(req.WorkingDir, "child-started"), []byte(fmt.Sprint(child.Process.Pid)), 0o600); err != nil {
				_ = child.Process.Kill()
				_ = child.Wait()
				return err
			}
			if err := child.Wait(); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(req.WorkingDir, "fixture-repaired"), []byte("healthy"), 0o600)
		})
		if err != nil {
			os.Exit(4)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestNativeCodingRecoveryCallerLossHostProbe(t *testing.T) {
	const dispatcherKey = "VROOLI_NATIVE_CODING_DISPATCH_FIXTURE"
	const pointKey = "VROOLI_NATIVE_CODING_DISPATCH_POINT"
	if ledger := os.Getenv(dispatcherKey); ledger != "" {
		b, err := NewCodingAgentBroker(ledger, nil)
		if err != nil {
			t.Fatal(err)
		}
		b.native = nativeCodingOwner("")
		records, err := b.Snapshot()
		if err != nil || len(records) != 1 {
			t.Fatal("fixture receipt missing")
		}
		e, err := b.readExecution(records[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		unlock, err := b.lockLedger(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if os.Getenv(pointKey) == "before_dispatch" {
			if err := os.WriteFile(filepath.Join(e.Request.WorkingDir, "caller-before-dispatch"), []byte("ready"), 0o600); err != nil {
				t.Fatal(err)
			}
			select {} // Parent kills this exact lock-holding dispatcher.
		}
		if os.Getenv(pointKey) == "after_commit" {
			b.native.start = func(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error) {
				if err := os.WriteFile(filepath.Join(e.Request.WorkingDir, "caller-before-dispatch"), []byte("committed"), 0o600); err != nil {
					t.Fatal(err)
				}
				select {}
			}
		}
		err = b.dispatchNative(ctx, &e)
		unlock()
		if err != nil {
			t.Fatal(err)
		}
		select {}
	}
	if os.Getenv("VROOLI_NATIVE_CODING_RECOVERY_PROBE") != "1" {
		t.Skip("set VROOLI_NATIVE_CODING_RECOVERY_PROBE=1 for the isolated native caller-loss probe")
	}
	t.Setenv("HOME", t.TempDir())
	b, e := nativeRecoveryFixture(t)
	incarnation, err := process.Incarnation(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	e.BootID, _, _ = strings.Cut(incarnation, ":")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	image, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	worker, _ := b.workerPath(e.ID)
	if err := config.WriteOwnedFileDurableStream(worker, 0o700, func(w io.Writer) error { _, err := io.Copy(w, image); return err }); err != nil {
		t.Fatal(err)
	}
	e.WorkerSHA, err = coreactivation.ImageDigest(context.Background(), worker)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv(pointKey) == "after_commit" {
		e.Deadline = time.Now().Add(3 * time.Second)
		b.records[0].DeadlineAt = e.Deadline
		if err := b.persistLocked(); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		unlock, err := b.lockLedger(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		current, readErr := b.readExecution(e.ID)
		if readErr == nil {
			current.Fence = "revoked"
			readErr = b.writeExecution(current)
		}
		unlock()
		if readErr != nil {
			t.Error(readErr)
			return
		}
		ref, err := platform.LookupOwnedOperation(ctx, e.Intent)
		if err == nil {
			if _, err := platform.StopOwnedOperation(ctx, ref); err != nil {
				t.Error(err)
				return
			}
			if err := platform.ReleaseOwnedOperation(ctx, ref); err != nil {
				t.Error(err)
			}
		}
	})
	caller := exec.Command(self, "-test.run=^TestNativeCodingRecoveryCallerLossHostProbe$")
	caller.Env = append(os.Environ(), dispatcherKey+"="+b.path)
	caller.Stdout = io.Discard
	caller.Stderr = io.Discard
	if err := caller.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caller.Process.Kill(); _ = caller.Wait() }()
	point := os.Getenv(pointKey)
	killedBeforeStart := point == "before_dispatch" || point == "after_commit"
	if killedBeforeStart {
		deadline := time.Now().Add(5 * time.Second)
		for {
			if _, err := os.Stat(filepath.Join(e.Request.WorkingDir, "caller-before-dispatch")); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("caller did not reach pre-dispatch crash edge")
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err := caller.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		_ = caller.Wait()
		prepared, err := b.readExecution(e.ID)
		if err != nil || prepared.DispatchCommitted != (point == "after_commit") || prepared.Fence != "ready" {
			t.Fatal("pre-dispatch crash point was not established")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err = b.ReconcileNative(ctx, 32)
		cancel()
		if err != nil {
			t.Fatalf("fresh observer did not make the original first submission: %v", err)
		}
		if point == "after_commit" {
			if _, err := os.Stat(filepath.Join(e.Request.WorkingDir, "child-started")); !os.IsNotExist(err) {
				t.Fatal("uncertain dispatch was repeated")
			}
			if wait := time.Until(e.Deadline); wait > 0 {
				time.Sleep(wait + 20*time.Millisecond)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := b.ReconcileNative(ctx, 32); err != nil {
				t.Fatalf("lost dispatch never retired after its actual deadline: %v", err)
			}
			retired, err := b.readExecution(e.ID)
			if err != nil || !retired.ExitProven || !retired.Retired {
				t.Fatal("lost dispatch retirement lacks complete native absence")
			}
			rows, err := b.Snapshot()
			if err != nil || rows[0].Outcome != "failed" {
				t.Fatal("lost dispatch did not retain a truthful failed attempt")
			}
			t.Logf("caller SIGKILL after dispatch commit and before manager submission: no replay, actual deadline elapsed, native absence proven, inputs retired; request=%s", e.ID)
			return
		}
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(e.Request.WorkingDir, "child-started")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("native child did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
	claimed, err := b.readExecution(e.ID)
	if err != nil || claimed.Fence != "claimed" || claimed.Ref == nil || claimed.LeaseID != "native-fixture-session" {
		t.Fatalf("native lease was not bound before child start: %+v %v", claimed, err)
	}
	if !killedBeforeStart {
		if err := caller.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		_ = caller.Wait()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	operation := platform.OwnedOperation{Ref: *claimed.Ref}
	if err := operation.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.Request.WorkingDir, "fixture-repaired")); err != nil {
		t.Fatal("repair did not survive caller loss")
	}
	if _, err := b.ReconcileNative(ctx, 32); err != nil {
		t.Fatal(err)
	}
	records, err := b.Snapshot()
	if err != nil || records[0].Outcome != "succeeded" {
		t.Fatalf("native completion lost: %+v %v", records, err)
	}
	retired, err := b.readExecution(e.ID)
	if err != nil || !retired.ExitProven || !retired.Retired {
		t.Fatalf("native retirement unconfirmed: %+v %v", retired, err)
	}
	t.Logf("caller SIGKILL point=%s preserved native request %s, claimed lease, child completion, exact tree exclusion and retained-image retirement", os.Getenv(pointKey), e.ID)
}

func TestNativeCodingPreparationOrphanCurrentHostProbe(t *testing.T) {
	image := os.Getenv("VROOLI_NATIVE_CODING_PREPARATION_ROOT_IMAGE")
	if image == "" {
		t.Skip("requires explicit governed root image for isolated failed-admission preparation probe")
	}
	dir := t.TempDir()
	b, err := NewNativeCodingAgentBroker(filepath.Join(dir, "receipts.json"), image)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < codingAgentMaxRecords; i++ {
		b.records = append(b.records, CodingAgentRecord{ID: uuid.NewString(), Scenario: "historical-fixture", Requester: "test", Outcome: "failed", CreatedAt: time.Now().Add(-24 * time.Hour)})
	}
	if err := b.persistLocked(); err != nil {
		t.Fatal(err)
	}
	b.native.start = func(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error) {
		t.Fatal("failed admission dispatched a worker")
		return nil, nil
	}
	_, err = b.SubmitContext(context.Background(), CodingAgentRequest{Scenario: "web-console", Requester: "test", WorkingDir: dir, Claim: filepath.Join("scenarios", "web-console"), Reason: "private failed-admission fixture"})
	if err == nil {
		t.Fatal("full ledger unexpectedly admitted another request")
	}
	parent := filepath.Join(dir, "coding-agent-executions")
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 1 {
		t.Fatalf("prepared orphan was not established: %v", err)
	}
	e, err := b.readExecution(entries[0].Name())
	if err != nil || e.DispatchCommitted {
		t.Fatal("preparation lacks a durable undispatched identity")
	}
	if count, err := b.ReconcileNative(context.Background(), 32); err != nil || count != 1 {
		t.Fatalf("unadmitted actual worker image was not retired: %d %v", count, err)
	}
	e, err = b.readExecution(e.ID)
	if err != nil || !e.ExitProven || !e.Retired {
		t.Fatal("preparation retirement lacks native absence proof")
	}
	rows, err := b.Snapshot()
	if err != nil || len(rows) != codingAgentMaxRecords {
		t.Fatal("preparation cleanup changed admission history")
	}
	t.Logf("actual governed worker preparation followed by failed ledger commit: native never dispatched, original ledger unchanged, descriptor retained and image retired; request=%s", e.ID)
}

func TestNativeCodingRetainedLedgerReadCompatibilityHostProbe(t *testing.T) {
	ledger, worker := os.Getenv("VROOLI_NATIVE_CODING_READONLY_LEDGER"), os.Getenv("VROOLI_NATIVE_CODING_READONLY_WORKER")
	if ledger == "" || worker == "" {
		t.Skip("requires explicit retained ledger and governed worker for read-only compatibility proof")
	}
	b, err := NewNativeCodingAgentBroker(ledger, worker)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := b.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	executions, err := b.NativeExecutions()
	if err != nil {
		t.Fatal(err)
	}
	unknown := 0
	for _, r := range rows {
		if r.Outcome == "unconfirmed" {
			unknown++
		}
	}
	t.Logf("existing ledger/sidecars readable with new owner: receipts=%d native_sidecars=%d unresolved_legacy_receipts=%d; no submission, reconciliation, budget reset or process effects", len(rows), len(executions), unknown)
}
