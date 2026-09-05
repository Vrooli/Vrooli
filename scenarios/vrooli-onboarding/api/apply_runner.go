package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	platform "github.com/vrooli/platform-go"
)

// The apply run is owned by a process of its own.
//
// Applying a selection starts scenarios, and a started scenario can restart
// this API -- vrooli-onboarding is a scenario like any other, it appears in its
// own selection closure, and other scenarios declare it as a startup
// dependency. While the executor ran inside the API's own process that was
// unsurvivable: the process was stopped part-way through its own apply, the run
// stayed "applying" forever with no one advancing it, every remaining item was
// silently skipped, and the operator's wizard lost the API it was polling
// moments after answering a consent prompt.
//
// Guarding the paths that lead to a restart is the wrong shape of fix. There is
// more than one, they behave differently, and each guard only patches the path
// it names. The invariant to restore is simpler: a long-running host mutation
// must not live inside a process that the mutation is allowed to restart. So
// the API accepts the run, writes it down, and hands it to a detached process;
// the API then only ever reads run state back. Restarting the API becomes an
// ordinary event that a wizard reconnects across, rather than the end of the
// run.

// applyRunnerFlag is the argument that puts this binary into runner mode
// instead of server mode.
const applyRunnerFlag = "--apply-run"

// applyRunnerRequest reports the run this process was asked to execute.
// Returns false for a normal server start.
func applyRunnerRequest(args []string) (string, bool) {
	for i, arg := range args {
		switch {
		case arg == applyRunnerFlag:
			if i+1 < len(args) {
				return strings.TrimSpace(args[i+1]), true
			}
			return "", true
		case strings.HasPrefix(arg, applyRunnerFlag+"="):
			return strings.TrimSpace(strings.TrimPrefix(arg, applyRunnerFlag+"=")), true
		}
	}
	return "", false
}

// runApplyRunner executes one persisted run and returns. It is the whole of
// runner mode: no listener, no router, no background services.
func runApplyRunner(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s requires an apply run id", applyRunnerFlag)
	}
	run, err := loadPersistedApplyRun(id)
	if err != nil {
		return fmt.Errorf("load apply run %s: %w", id, err)
	}
	if run.Status != "pending" {
		// Re-running a terminal run would repeat host mutations that already
		// happened. Claiming the run is what makes the handoff single-shot.
		return fmt.Errorf("apply run %s is %q, not pending", id, run.Status)
	}

	run.RunnerPID = os.Getpid()
	run.Heartbeat = operatorStateNow().UTC().Format(time.RFC3339)
	updateApplyRun(run)

	stop := startApplyHeartbeat(id)
	defer stop()

	executeApplyRun(ctx, run)
	return nil
}

// startApplyHeartbeat restamps the run's liveness until the returned function
// is called. It reads the run back each tick rather than closing over a
// snapshot, so it never overwrites progress the executor has recorded.
func startApplyHeartbeat(id string) func() {
	done := make(chan struct{})
	var once sync.Once
	go func() {
		ticker := time.NewTicker(applyHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				current, ok := applyRunSnapshot(id)
				if !ok || isTerminalApplyStatus(current.Status) {
					return
				}
				current.RunnerPID = os.Getpid()
				current.Heartbeat = operatorStateNow().UTC().Format(time.RFC3339)
				updateApplyRun(current)
			}
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}

func isTerminalApplyStatus(status string) bool {
	switch status {
	case "pending", "applying":
		return false
	default:
		return true
	}
}

var applyRunnerExecutablePath string

// prepareApplyRunnerExecutable keeps a stable copy of the API image outside
// the lifecycle-managed component path. Scenario setup may atomically replace
// or remove api/vrooli-onboarding-api while the old server is still serving;
// os.Executable then names a path that no longer exists, so a detached apply
// runner cannot be started. The copy is state-owned and is refreshed only when
// the API itself starts.
func prepareApplyRunnerExecutable() error {
	source, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve apply runner source: %w", err)
	}
	home, homeErr := os.UserHomeDir()
	dir := ""
	if homeErr == nil && strings.TrimSpace(home) != "" {
		dir = filepath.Join(home, ".local", "state", "vrooli-onboarding", "apply-runs")
	} else {
		statePath, err := operatorStatePath()
		if err != nil {
			return err
		}
		dir = filepath.Join(filepath.Dir(statePath), "apply-runs")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create apply runner state directory: %w", err)
	}
	destination := filepath.Join(dir, "vrooli-onboarding-api-runner")
	temporary := destination + fmt.Sprintf(".tmp-%d", os.Getpid())
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open apply runner source: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		return fmt.Errorf("create apply runner image: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		_ = os.Remove(temporary)
		return fmt.Errorf("copy apply runner image: %w", err)
	}
	if err := output.Close(); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("close apply runner image: %w", err)
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("publish apply runner image: %w", err)
	}
	applyRunnerExecutablePath = destination
	return nil
}

// spawnApplyRunner starts the detached executor for an accepted run.
//
// Setsid puts the runner in its own session so it is not in the API's process
// group: stopping the scenario signals the group, and a runner that shared it
// would die exactly when it must not. The binary is resolved through
// os.Executable so a runner keeps executing the image it started with even
// after a scenario restart swaps the file underneath it.
//
// The context is not passed to the child: a detached runner establishes its own
// storage routing from the environment, exactly as a freshly started API does.
// It is in the signature so that an in-process substitute -- the one behaviour
// tests use -- can carry the request's routed-storage selection, which is the
// only thing distinguishing a test root from the default one.
var spawnApplyRunner = func(_ context.Context, run applyRun) error {
	id := run.ID
	executable := strings.TrimSpace(applyRunnerExecutablePath)
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return fmt.Errorf("resolve runner executable: %w", err)
		}
	}
	logFile, err := applyRunnerLog(id)
	if err != nil {
		return err
	}

	command := exec.Command(executable, applyRunnerFlag, id)
	command.Env = os.Environ()
	// The runner inherits no working directory from the request. It resolves
	// everything from the roots it is configured with.
	if roots, rootErr := resolveRoots(); rootErr == nil && strings.TrimSpace(roots.RepoRoot) != "" {
		command.Dir = roots.RepoRoot
	}
	command.Stdin = nil
	command.Stdout = logFile
	command.Stderr = logFile
	command.SysProcAttr = platform.DetachedProcessAttrs()

	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start apply runner: %w", err)
	}
	// The parent keeps no claim on the child. Reaping it here would block the
	// request; the run's own persisted state is the completion signal.
	go func() {
		_ = command.Wait()
		_ = logFile.Close()
	}()
	return nil
}

func applyRunnerLog(id string) (*os.File, error) {
	statePath, err := operatorStatePath()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(filepath.Dir(statePath), "apply-runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, id+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}

// observedApplyRun reports a run as a reader should see it, replacing a
// non-terminal status whose executor is gone with an explicit "interrupted".
//
// This is computed on read and never written back. A reader that persisted its
// own verdict would race the runner: a slow heartbeat during a long item would
// be recorded as death, and the still-healthy runner would then be reporting
// progress on a run the store had already buried.
func observedApplyRun(run applyRun) applyRun {
	if isTerminalApplyStatus(run.Status) {
		return run
	}
	if applyRunnerAlive(run) {
		return run
	}
	run.Status = "interrupted"
	if run.Error == "" {
		run.Error = "the process executing this apply is no longer running, so the remaining items were not attempted"
	}
	run.Blockers = append(run.Blockers, completionBlocker{
		Kind:        "apply",
		Name:        "apply-runner",
		Reason:      "the apply run was interrupted before it finished",
		Remediation: "Review the items already applied, then apply the selection again to resume the remainder.",
	})
	return run
}

// applyRunnerAlive reports whether the run still has a working executor.
//
// A run that has not yet recorded a heartbeat is treated as alive: it was just
// accepted and its runner is still starting. Only a run whose heartbeat has
// gone quiet past the stale window, and whose recorded process is gone, is
// declared interrupted -- both signals are required, because a heartbeat can
// lag on a slow filesystem and a pid can be reused.
func applyRunnerAlive(run applyRun) bool {
	if strings.TrimSpace(run.Heartbeat) == "" {
		return true
	}
	beat, err := time.Parse(time.RFC3339, run.Heartbeat)
	if err != nil {
		return true
	}
	if operatorStateNow().UTC().Sub(beat.UTC()) < staleApplyHeartbeat {
		return true
	}
	return run.RunnerPID > 0 && processAlive(run.RunnerPID)
}

var processAlive = func(pid int) bool {
	return platform.IsPIDRunning(pid)
}
