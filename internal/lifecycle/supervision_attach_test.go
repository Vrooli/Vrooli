package lifecycle

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// noSupervisorLaunch keeps these tests hermetic. The real hook would ask the
// host's service manager to start a supervisor, which a unit test has no
// business doing; the handover itself is what is under test here, and the
// session it attaches to is registered directly in the test's own registry.
func noSupervisorLaunch(deps *lifecycleDeps) {
	deps.ensureRuntimeSupervisor = func(context.Context, string, io.Writer, io.Writer) error { return nil }
}

// newSupervisorSessionForTest registers a live supervisor session in the home's
// registry so a start has something to hand ownership to.
func newSupervisorSessionForTest(t *testing.T, home, supervisorID string) {
	t.Helper()
	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()
	pid := 4242
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  supervisorID,
		HostBootID:    "boot-test",
		HostSessionID: "session-test",
		PID:           &pid,
	}, time.Hour); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
}

func runningInstanceForTest(t *testing.T, home, slug string) scenarioruntime.Instance {
	t.Helper()
	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: slug,
		Statuses: []string{scenarioruntime.StatusRunning},
	})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("running instances = %d, want exactly one", len(instances))
	}
	return instances[0]
}

// A start must not finish with the instance leased to the process that is about
// to exit. Before this, `vrooli scenario start` returned with owner_kind
// 'lifecycle' and owner_pid set to the CLI's own PID; the CLI then exited, so
// nothing renewed the lease and the scenario read as stopped roughly 30 seconds
// later while its API was still serving.
func TestStartHandsOwnershipToTheLiveSupervisor(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	newSupervisorSessionForTest(t, home, "sup-live")

	runner := newLifecycleRunnerForTest(t, root, home, noSupervisorLaunch)
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(alpha): %v", err)
	}
	defer func() {
		if err := runner.Stop("alpha", StopOptions{}); err != nil {
			t.Fatalf("Stop(alpha): %v", err)
		}
	}()

	instance := runningInstanceForTest(t, home, "alpha")
	if instance.OwnerKind != scenarioruntime.OwnerKindSupervisor {
		t.Fatalf("owner kind = %q, want %q — the start returned still owning the lease",
			instance.OwnerKind, scenarioruntime.OwnerKindSupervisor)
	}
	if instance.OwnerPID != nil {
		t.Fatalf("owner pid = %d, want nil — a supervisor-owned lease must not name a process", *instance.OwnerPID)
	}
	if instance.SupervisorID != "sup-live" {
		t.Fatalf("supervisor id = %q, want sup-live", instance.SupervisorID)
	}
	if instance.SupervisedAt == nil {
		t.Fatal("supervised_at is unset, so the handover was not recorded")
	}
}

// With no supervisor to hand off to, the start still succeeds and keeps
// lifecycle ownership. Failing the start would be far worse than the degraded
// ownership it is reporting.
func TestStartKeepsLifecycleOwnershipWithoutALiveSupervisor(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner := newLifecycleRunnerForTest(t, root, home, noSupervisorLaunch)
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(alpha): %v", err)
	}
	defer func() {
		if err := runner.Stop("alpha", StopOptions{}); err != nil {
			t.Fatalf("Stop(alpha): %v", err)
		}
	}()

	instance := runningInstanceForTest(t, home, "alpha")
	if instance.OwnerKind != scenarioruntime.OwnerKindLifecycle || instance.OwnerPID == nil {
		t.Fatalf("instance = %+v, want lifecycle ownership retained", instance)
	}
}

// The provenance has to reach the durable record, not just exist in memory:
// the whole point is that it outlives the process that started the scenario.
func TestStartRecordsInitiatorProvenance(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		noSupervisorLaunch(deps)
		deps.captureInitiator = func() process.InitiatorInfo {
			return process.InitiatorInfo{
				PID:        4242,
				Argv:       "vrooli scenario start alpha",
				ParentPID:  4241,
				ParentArgv: "claude --dangerously-skip-permissions",
				Scope:      "/user.slice/tmux-spawn-abc.scope",
			}
		}
	})
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(alpha): %v", err)
	}
	defer func() {
		if err := runner.Stop("alpha", StopOptions{}); err != nil {
			t.Fatalf("Stop(alpha): %v", err)
		}
	}()

	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()
	op, err := store.GetLatestStartOperation(ctx, "alpha", "live")
	if err != nil {
		t.Fatalf("GetLatestStartOperation: %v", err)
	}
	if op.InitiatorArgv != "vrooli scenario start alpha" {
		t.Fatalf("initiator argv = %q", op.InitiatorArgv)
	}
	if op.InitiatorParentPID == nil || *op.InitiatorParentPID != 4241 {
		t.Fatalf("initiator parent pid = %v, want 4241", op.InitiatorParentPID)
	}
	if op.InitiatorParentArgv != "claude --dangerously-skip-permissions" {
		t.Fatalf("initiator parent argv = %q — the caller's identity is the answer to 'who started this?'", op.InitiatorParentArgv)
	}
	if op.InitiatorScope != "/user.slice/tmux-spawn-abc.scope" {
		t.Fatalf("initiator scope = %q — the scope outlives the pid and names the pane", op.InitiatorScope)
	}
}
