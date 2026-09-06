// Package sessioncontain is the platform-go implementation of cli-core's
// SessionContainer: the coding-agent launcher binaries register it so every
// session is born inside a scope under vrooli-agents.slice (Linux), an rlimit
// shim (macOS) or a Job Object (Windows). cli-core cannot import platform-go
// itself (244 modules replace cli-core), so the seam lives there and the
// primitive here.
package sessioncontain

import (
	"context"
	"fmt"

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
// failure of the post-start step leaves the process running and is reported
// through the session's method.
func (Container) Run(ctx context.Context, scope string, c cliutil.SessionContainment, p cliutil.SessionProcess) (cliutil.ContainedSession, error) {
	contained, err := platform.ContainedCommand(platform.ContainedSpec{
		Path: p.Path, Args: p.Args, Env: p.Env, Dir: p.Dir, Stdin: p.Stdin, Stdout: p.Stdout, Stderr: p.Stderr,
		Scope: scope, Containment: containment(c),
	})
	if err != nil {
		return cliutil.ContainedSession{Method: platform.MethodNone}, &cliutil.UncontainedError{Err: err}
	}
	defer contained.Release()
	if err := contained.Start(); err != nil {
		if contained.Cmd.Process == nil {
			return cliutil.ContainedSession{Method: platform.MethodNone}, &cliutil.UncontainedError{Err: err}
		}
		// Started but not placed: wait for it and report the gap.
		waitErr := contained.Cmd.Wait()
		return cliutil.ContainedSession{Scope: contained.Scope.String(), Method: contained.Method + " (unplaced: " + err.Error() + ")"}, waitErr
	}
	session := cliutil.ContainedSession{Scope: contained.Scope.String(), Method: contained.Method}
	waitErr := contained.Cmd.Wait()
	if ctx.Err() != nil && waitErr != nil {
		return session, fmt.Errorf("%w: %v", ctx.Err(), waitErr)
	}
	return session, waitErr
}

// ContainSelf moves the calling process into the scope before an exec.
func (Container) ContainSelf(scope string, c cliutil.SessionContainment) (cliutil.ContainedSession, error) {
	ref, method, err := platform.ContainSelf(scope, containment(c))
	return cliutil.ContainedSession{Scope: ref.String(), Method: method}, err
}

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
