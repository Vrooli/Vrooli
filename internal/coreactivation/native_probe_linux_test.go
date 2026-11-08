//go:build linux

package coreactivation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/process"
)

const coreNativeFixtureTest = "^TestCoreActivationNativeWorkerFixture$"

// This is an isolated transaction fixture, not proof of compatibility or
// activation of the serving root CLI. Its fixed test executable and marker
// artifacts exercise the real native job, lock, caller-loss and recovery path.
type coreNativeFixtureEffects struct {
	NativeWorker
	home string
}

func (f coreNativeFixtureEffects) QualifyAdmission(context.Context, Record) error { return nil }
func (f coreNativeFixtureEffects) assets(r Record) (Assets, error) {
	s, err := NewStoreAt(r.Home, os.Getenv("VROOLI_CORE_NATIVE_FIXTURE_STORE"))
	return Assets{Store: s}, err
}

func (f coreNativeFixtureEffects) PublishCandidate(ctx context.Context, r Record) error {
	a, err := f.assets(r)
	if err != nil {
		return err
	}
	return a.PublishBinary(ctx, r, false)
}

func (f coreNativeFixtureEffects) ActivateCandidate(ctx context.Context, r Record) error {
	a, err := f.assets(r)
	if err != nil {
		return err
	}
	if err := a.PublishDefinition(ctx, r, false); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	child := exec.Command(executable, "-test.run="+coreNativeFixtureTest)
	child.Env = append(coreNativeFixtureEnv(f.home), "VROOLI_CORE_NATIVE_FIXTURE_ROLE=child")
	if err := child.Start(); err != nil {
		return err
	}
	// The manager owns both descendants even if the worker root is killed.
	// Reap locally when possible; no child is allowed to leave this tree.
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	if err := config.WriteOwnedFileDurable(filepath.Join(f.home, "activated"), []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

func (f coreNativeFixtureEffects) VerifyCandidate(context.Context, Record) error {
	return errors.New("fixture must be interrupted before verification")
}

func (f coreNativeFixtureEffects) VerifyPrior(ctx context.Context, r Record) error {
	a, err := f.assets(r)
	if err != nil {
		return err
	}
	l, err := a.Layout(r)
	if err != nil {
		return err
	}
	sha, err := imageDigest(ctx, l.Executable)
	if err != nil || sha != r.PriorSHA {
		return ErrConflict
	}
	data, err := readFrozen(l.UnitPath, definitionLimit)
	if err != nil || bytesDigest(data) != r.PriorUnitSHA {
		return ErrConflict
	}
	return nil
}

func (f coreNativeFixtureEffects) RestorePrior(ctx context.Context, r Record) error {
	a, err := f.assets(r)
	if err != nil {
		return err
	}
	if err := a.PublishBinary(ctx, r, true); err != nil {
		return err
	}
	return a.PublishDefinition(ctx, r, true)
}

func (f coreNativeFixtureEffects) RetireInputs(ctx context.Context, r Record) error {
	if err := f.ReleaseWorker(ctx, r); err != nil {
		return err
	}
	a, err := f.assets(r)
	if err != nil {
		return err
	}
	return a.Retire(r)
}

func coreNativeFixtureEnv(home string) []string {
	return []string{"VROOLI_CORE_NATIVE_FIXTURE_HOME=" + home, "VROOLI_CORE_NATIVE_FIXTURE_STORE=" + os.Getenv("VROOLI_CORE_NATIVE_FIXTURE_STORE"), "VROOLI_CORE_NATIVE_FIXTURE_ROLE=worker", "HOME=" + os.Getenv("HOME"), "PATH=/usr/local/bin:/usr/bin:/bin", "XDG_RUNTIME_DIR=" + os.Getenv("XDG_RUNTIME_DIR"), "DBUS_SESSION_BUS_ADDRESS=" + os.Getenv("DBUS_SESSION_BUS_ADDRESS")}
}

func coreNativeFixtureSpec(t *testing.T, home, id string) platform.OperationSpec {
	t.Helper()
	executable := filepath.Join(os.Getenv("VROOLI_CORE_NATIVE_FIXTURE_STORE"), "inputs", id, "worker")
	return platform.OperationSpec{Path: executable, Args: []string{"-test.run=" + coreNativeFixtureTest}, Env: coreNativeFixtureEnv(home), Dir: home, Limits: platform.OperationLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 100, WallTime: 8 * time.Second}}
}

// Only invoked as a fixed subprocess by the opt-in host fixture below.
func TestCoreActivationNativeWorkerFixture(t *testing.T) {
	home := os.Getenv("VROOLI_CORE_NATIVE_FIXTURE_HOME")
	if home == "" {
		t.Skip("native transaction fixture subprocess only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if os.Getenv("VROOLI_CORE_NATIVE_FIXTURE_ROLE") == "child" {
		<-ctx.Done()
		return
	}
	s, err := NewStoreAt(home, os.Getenv("VROOLI_CORE_NATIVE_FIXTURE_STORE"))
	if err != nil {
		t.Fatal(err)
	}
	r, exists, err := s.Current()
	if err != nil || !exists {
		t.Fatalf("load admitted fixture: exists=%t err=%v", exists, err)
	}
	f := coreNativeFixtureEffects{home: home}
	e := Engine{Store: s, Effects: f}
	switch os.Getenv("VROOLI_CORE_NATIVE_FIXTURE_ROLE") {
	case "caller":
		if _, err := e.Dispatch(ctx, r.ID, coreNativeFixtureSpec(t, home, r.ID)); err != nil {
			t.Fatal(err)
		}
		<-ctx.Done()
	case "worker":
		ref, err := f.ResolveSelf(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.RunWorker(ctx, r.ID, ref); err != nil {
			t.Fatal(err)
		}
	case "recover":
		got, exists, err := e.Recover(ctx)
		if err != nil || !exists || got.Phase != FailedRestored {
			t.Fatalf("independent recovery: phase=%s exists=%t err=%v", got.Phase, exists, err)
		}
	default:
		t.Fatal("invalid fixed fixture role")
	}
}

func TestCoreActivationCurrentHostCallerLossAndDeadlineRecoveryProbe(t *testing.T) {
	if os.Getenv("VROOLI_CORE_ACTIVATION_NATIVE_PROBE") != "1" {
		t.Skip("set VROOLI_CORE_ACTIVATION_NATIVE_PROBE=1 for the isolated native transaction drill")
	}
	e, r, _, _ := fixture(t)
	t.Setenv("VROOLI_CORE_NATIVE_FIXTURE_STORE", e.Store.Directory())
	r.PriorPID = os.Getpid()
	var err error
	r.PriorIncarnation, err = process.Incarnation(r.PriorPID)
	if err != nil {
		t.Fatal(err)
	}
	originalExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r, l, _ := frozenFixtureWithWorker(t, e, r, originalExecutable)
	spec := coreNativeFixtureSpec(t, r.Home, r.ID)
	r.WorkerIntent, err = platform.NewOperationIntent("core-activation", r.ID, spec)
	if err != nil {
		t.Fatal(err)
	}
	f := coreNativeFixtureEffects{home: r.Home}
	e.Effects = f
	admit(t, e, r)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	// Register owner cleanup before dispatch, including ambiguous submission.
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 12*time.Second)
		defer done()
		ref, err := platform.LookupOwnedOperation(cleanup, r.WorkerIntent)
		if err == nil {
			_, _ = platform.StopOwnedOperation(cleanup, ref)
			_ = platform.ReleaseOwnedOperation(cleanup, ref)
		}
	})
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	caller := exec.CommandContext(ctx, executable, "-test.run="+coreNativeFixtureTest)
	caller.Env = append(coreNativeFixtureEnv(r.Home), "VROOLI_CORE_NATIVE_FIXTURE_ROLE=caller")
	var callerOutput boundedFixtureOutput
	caller.Stdout = &callerOutput
	caller.Stderr = &callerOutput
	if err := caller.Start(); err != nil {
		t.Fatal(err)
	}
	callerWaited := false
	t.Cleanup(func() {
		if !callerWaited {
			_ = caller.Process.Kill()
			_ = caller.Wait()
		}
	})
	var childPID int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, readErr := os.ReadFile(filepath.Join(r.Home, "activated"))
		if readErr == nil {
			childPID, err = strconv.Atoi(string(data))
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if childPID <= 0 {
		t.Fatalf("fixture worker did not reach activation: %s", callerOutput.String())
	}
	ref, err := platform.LookupOwnedOperation(ctx, r.WorkerIntent)
	if err != nil {
		t.Fatal(err)
	}
	childIncarnation, err := process.Incarnation(childPID)
	if err != nil {
		t.Fatal(err)
	}
	if err := caller.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = caller.Wait()
	callerWaited = true
	if err := os.RemoveAll(r.SourceRoot); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(l.Executable); err != nil || bytesDigest(data) != r.CandidateSHA {
		t.Fatal("caller loss did not leave the admitted fixture candidate visible")
	}
	if err := e.Store.GuardOrdinaryMutation(ctx); !errors.Is(err, ErrConflict) {
		t.Fatalf("ordinary mutation crossed active native worker: %v", err)
	}
	// One platform-owned blocking attachment, not a status polling loop.
	op := platform.OwnedOperation{Ref: ref}
	if err := op.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	observation, err := platform.InspectOwnedOperation(ctx, ref)
	if err != nil || observation.Populated || !observation.ExitKnown || observation.Result != "timeout" {
		t.Fatalf("native lifetime did not contain the fixture: %+v %v", observation, err)
	}
	if incarnation, err := process.Incarnation(childPID); err == nil && incarnation == childIncarnation {
		t.Fatal("fixture descendant survived native deadline")
	}
	recoverer := exec.CommandContext(ctx, executable, "-test.run="+coreNativeFixtureTest)
	recoverer.Env = append(coreNativeFixtureEnv(r.Home), "VROOLI_CORE_NATIVE_FIXTURE_ROLE=recover")
	output, err := recoverer.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh recovery process failed: %v %s", err, output)
	}
	got, exists, err := e.Store.Current()
	if err != nil || !exists || got.Phase != FailedRestored || !contains(got.FailedReleases, r.CandidateSHA) {
		t.Fatalf("durable recovery incomplete: %s %v", got.Phase, err)
	}
	if sha, err := imageDigest(ctx, l.Executable); err != nil || sha != r.PriorSHA {
		t.Fatal("wrong restored binary")
	}
	if data, err := os.ReadFile(l.UnitPath); err != nil || bytesDigest(data) != r.PriorUnitSHA {
		t.Fatal("wrong restored definition")
	}
	inputDir, _ := e.Store.inputDirectory(r.ID)
	if _, err := os.Lstat(inputDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("private executable/definition inputs were not retired")
	}
	if err := e.Store.GuardOrdinaryMutation(ctx); err != nil {
		t.Fatalf("retired terminal native worker still blocked ordinary mutation: %v", err)
	}
	if _, _, err := e.Recover(ctx); err != nil {
		t.Fatalf("repeated recovery failed: %v", err)
	}
	if _, err := e.RunWorker(ctx, r.ID, ref); !errors.Is(err, ErrClaimed) {
		t.Fatalf("late fixture worker accepted: %v", err)
	}
	t.Logf("isolated native transaction: frozen_worker_image_verified=true caller_killed=true native_result=%s descendant_excluded=true fresh_process_restored_binary_and_definition=true source_absent=true inputs_retired=true ordinary_mutation_fenced=true failed_release_suppressed=true unit=%s", observation.Result, ref.Unit)
}

// The helper emits no secrets. Keep unexpected fixture diagnostics bounded.
type boundedFixtureOutput struct {
	mu   sync.Mutex
	text []byte
}

func (b *boundedFixtureOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	if room := 4096 - len(b.text); room > 0 {
		if len(p) > room {
			p = p[:room]
		}
		b.text = append(b.text, p...)
	}
	return n, nil
}

func (b *boundedFixtureOutput) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return fmt.Sprintf("%s", b.text)
}
