// Package sessioncontain is the platform-go implementation of cli-core's
// SessionContainer: the coding-agent launcher binaries register it so every
// session is born inside a scope under vrooli-agents.slice (Linux), an rlimit
// shim (macOS) or a Job Object (Windows). cli-core cannot import platform-go
// itself (244 modules replace cli-core), so the seam lives there and the
// primitive here.
package sessioncontain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
)

// Container adapts platform-go's containment to cli-core's seam.
type Container struct{}

// Register installs the container as cli-core's default.
func Register() {
	cliutil.RegisterSessionContainer(Container{})
	cliutil.RegisterCeilingDiagnostic(CeilingDiagnostic)
}

func containment(c cliutil.SessionContainment) platform.Containment {
	return platform.Containment{Slice: c.Slice, CPUWeight: c.CPUWeight, MemoryHigh: c.MemoryHigh, MemoryMax: c.MemoryMax, TasksMax: c.TasksMax}
}

// Run starts the process inside the scope and waits for it. A failure before
// the process starts is an UncontainedError so the launcher can fall back; a
// failure after spawn is cleaned up at the platform boundary. Cancellation
// stops the acquired scope and reaps the exact root before returning.
func (Container) Run(ctx context.Context, scope string, c cliutil.SessionContainment, p cliutil.SessionProcess) (cliutil.ContainedSession, error) {
	contained, err := platform.ContainedCommand(platform.ContainedSpec{
		Path: p.Path, Args: p.Args, Env: p.Env, Dir: p.Dir, Stdin: p.Stdin, Stdout: p.Stdout, Stderr: p.Stderr,
		Scope: scope, Containment: containment(c),
	})
	if err != nil {
		return cliutil.ContainedSession{Method: platform.MethodNone}, &cliutil.UncontainedError{Err: err}
	}
	defer contained.Release()
	contained.Cmd.WaitDelay = 2 * time.Second
	if err := contained.StartContext(ctx); err != nil {
		if ctx.Err() != nil {
			return cliutil.ContainedSession{Scope: contained.Scope.String(), Method: contained.Method}, err
		}
		if contained.Cmd.Process == nil {
			return cliutil.ContainedSession{Method: platform.MethodNone}, &cliutil.UncontainedError{Err: err}
		}
		// StartContext already stopped/reaped the exact process on a
		// post-spawn placement failure. Never start a fallback agent here.
		return cliutil.ContainedSession{Scope: contained.Scope.String(), Method: contained.Method}, err
	}
	session := cliutil.ContainedSession{Scope: contained.Scope.String(), Method: contained.Method}
	if ctx.Done() == nil {
		return session, contained.Cmd.Wait()
	}
	done := make(chan error, 1)
	go func() { done <- contained.Cmd.Wait() }()
	select {
	case err := <-done:
		return session, err
	case <-ctx.Done():
	}
	cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stopErr := platform.StopScopeContext(cleanup, contained.Scope)
	// systemd-run may be outside the acquired agent scope. Kill only this
	// invocation's root handle, then wait for its reaping with bounded pipes.
	killErr := contained.Cmd.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case waitErr := <-done:
		if stopErr == nil {
			stopErr = platform.WaitScopeEmpty(cleanup, contained.Scope)
		}
		if stopErr != nil {
			stopErr = errors.Join(platform.ErrContainmentCleanupUnverified, stopErr)
		}
		return session, errors.Join(ctx.Err(), stopErr, killErr, waitErr)
	case <-timer.C:
		return session, errors.Join(ctx.Err(), stopErr, killErr, platform.ErrContainmentCleanupUnverified, errors.New("contained launcher reaping timed out"))
	}
}

// ContainSelf moves the calling process into the scope before an exec.
func (Container) ContainSelf(scope string, c cliutil.SessionContainment) (cliutil.ContainedSession, error) {
	ref, method, err := platform.ContainSelf(scope, containment(c))
	return cliutil.ContainedSession{Scope: ref.String(), Method: method}, err
}

// SupportsContainment satisfies cliutil.ContainmentCapability so the launcher
// can tell a platform that has no ceiling primitive from one whose ceiling
// broke. Only the second is worth an operator's attention; without the
// distinction a host that can never comply warns on every single launch.
func (Container) SupportsContainment() bool { return platform.SupportsContainment() }

// ceilingSaturationFloor is how full a ceiling has to be before a launch is
// warned about it. Below this a session starts normally and silently; at or
// above it the next fork may be the one the kernel refuses.
const ceilingSaturationFloor = 0.95

// CeilingDiagnostic reports why the agent slice is about to refuse work, or
// "" when it has room. It is registered with cli-core, which cannot read a
// cgroup itself: 244 modules replace cli-core, so it carries no platform-go
// import (the 2026-08 api-core incident).
//
// The reading is deliberately cheap and deliberately silent about anything it
// cannot answer. A launcher diagnostic must never delay or block a launch,
// and a ceiling it cannot read is not a reason to alarm an operator.
func CeilingDiagnostic() string {
	path, err := platform.SliceCgroup(cliutil.AgentSlice)
	if err != nil {
		return ""
	}
	occupancy, err := platform.ScopeOccupancy(platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: path})
	if err != nil {
		return ""
	}
	if tasks := occupancy.TaskSaturation(); tasks >= ceilingSaturationFloor {
		return fmt.Sprintf(
			"%s holds %d of its %d tasks. At the ceiling the kernel refuses this session's first fork, and the agent will report that failure in its own words — most often as a corrupt or unreadable database. The database is fine. Free the ceiling or raise TasksMax in the agent_session_containment safeguard.",
			cliutil.AgentSlice, occupancy.Tasks, occupancy.TasksMax)
	}
	if memory := occupancy.MemorySaturation(); memory >= ceilingSaturationFloor {
		return fmt.Sprintf(
			"%s holds %d of its %d memory bytes. At the ceiling systemd-oomd kills inside the slice, which may take this session or another one.",
			cliutil.AgentSlice, occupancy.MemoryBytes, occupancy.MemoryMaxBytes)
	}
	return ""
}
