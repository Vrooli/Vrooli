package systemmetrics

import "scenario-to-cloud/domain"

// CommandSpec describes a remote command needed by a system metrics collector.
type CommandSpec struct {
	ID      string
	Command string
}

// CommandResult is the normalized command output used by collectors.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Collector is an OS-specific parser for system-level metrics.
//
// Implementations should only parse command output; they should not perform I/O.
type Collector interface {
	Name() string
	SystemCommands() []CommandSpec
	ParseSystemState(results map[string]CommandResult) domain.SystemState
}

// CollectorForOS returns an OS-specific collector.
//
// Unknown OS IDs currently fall back to Linux parsing because current
// deployments are Linux-based. Adding support for a new OS is a matter of
// adding another collector and branching here.
func CollectorForOS(osID string) Collector {
	switch osID {
	case "linux", "ubuntu", "debian", "":
		fallthrough
	default:
		return linuxCollector{}
	}
}
