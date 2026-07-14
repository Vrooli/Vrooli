package exec

import (
	"fmt"
	"runtime"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

// containmentBackend is the platform-neutral seam for one OS-specific
// process-containment mechanism. Linux ships the bwrap backend
// (backend_linux.go); a macOS Seatbelt backend plugs in here without
// touching the dispatch below. Each backend reports whether it can run on
// the host (available) and, when it can, assembles the process.StartOpts
// that launch the command under its containment (buildStartOpts).
type containmentBackend interface {
	// id returns the backend's stable identifier (e.g. "bwrap"). It is the
	// backend id surfaced in the /driver/containment report.
	id() string

	// available reports whether the backend can run on this host, returning
	// nil when it can and a diagnostic error (suitable for surfacing to an
	// operator) when it cannot.
	available(starter process.Starter) error

	// buildStartOpts assembles the StartOpts that run cmd under this
	// backend's containment. Only called after available returned nil.
	buildStartOpts(starter process.Starter, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (process.StartOpts, error)
}

// noneBackend is the direct-execution path: no containment, command runs
// in s.MergedDir. It backs ContainmentNone and the ContainmentPreferred
// fallback when no platform backend is available.
type noneBackend struct{}

func (noneBackend) id() string { return "none" }

func (noneBackend) available(process.Starter) error { return nil }

func (noneBackend) buildStartOpts(_ process.Starter, s *types.Sandbox, _ BwrapConfig, cmd string, args ...string) (process.StartOpts, error) {
	return process.StartOpts{
		Path: cmd,
		Args: append([]string(nil), args...),
		Dir:  s.MergedDir,
	}, nil
}

// buildStartOpts assembles a process.StartOpts for the requested
// containment level by dispatching to the platform backend, and reports
// the id of the backend that actually ran so callers can stamp per-launch
// provenance with the truth rather than re-inferring it:
//
//	ContainmentNone:      always direct exec in s.MergedDir ("none").
//	ContainmentPreferred: platform backend when available, else direct.
//	ContainmentRequired:  platform backend, or a hard error when none is
//	                      available on this host.
func buildStartOpts(starter process.Starter, s *types.Sandbox, level driver.ContainmentLevel, cfg BwrapConfig, cmd string, args ...string) (process.StartOpts, string, error) {
	direct := noneBackend{}
	switch level {
	case driver.ContainmentNone:
		opts, err := direct.buildStartOpts(starter, s, cfg, cmd, args...)
		return opts, direct.id(), err
	case driver.ContainmentPreferred:
		if backend := platformContainmentBackend(); backend != nil && backend.available(starter) == nil {
			opts, err := backend.buildStartOpts(starter, s, cfg, cmd, args...)
			return opts, backend.id(), err
		}
		opts, err := direct.buildStartOpts(starter, s, cfg, cmd, args...)
		return opts, direct.id(), err
	case driver.ContainmentRequired:
		backend := platformContainmentBackend()
		if backend == nil {
			return process.StartOpts{}, "", fmt.Errorf("containment level Required has no available backend on %s", runtime.GOOS)
		}
		if err := backend.available(starter); err != nil {
			return process.StartOpts{}, "", err
		}
		opts, err := backend.buildStartOpts(starter, s, cfg, cmd, args...)
		return opts, backend.id(), err
	}
	return process.StartOpts{}, "", fmt.Errorf("unknown containment level: %d", level)
}
