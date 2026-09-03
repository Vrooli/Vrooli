//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/vrooli/platform-go/rlimitexec"
)

// macOS has no cgroups. The ceiling is per process, applied by setrlimit and
// inherited across fork and exec: RLIMIT_NPROC from TasksMax and RLIMIT_AS
// from MemoryMax (per process, honestly narrower than a tree ceiling; the
// requirement module says so). The tree identity is the process group the
// command is started in. CPUWeight has no per-process lever here beyond
// nice, which launchd applies to units; sessions run at the launcher's nice.
//
// The shim is this binary re-executed as `<self> rlimit-exec ... -- <cmd>`:
// the host binary must route os.Args[1] == rlimitexec.Subcommand into
// rlimitexec.MaybeRun at the top of main (cmd/vrooli-agent-launcher does).
// Fixture- and compile-verified only; no macOS host is reachable.

func containedCommand(spec ContainedSpec) (*Contained, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("platform: rlimit shim needs the executable path: %w", err)
	}
	limits, err := rlimitSpec(spec.Containment)
	if err != nil {
		return nil, err
	}
	argv := []string{rlimitexec.Subcommand}
	if limits.MaxProcesses > 0 {
		argv = append(argv, fmt.Sprintf("--%s=%d", rlimitexec.FlagMaxProcesses, limits.MaxProcesses))
	}
	if limits.AddressSpaceBytes > 0 {
		argv = append(argv, fmt.Sprintf("--%s=%d", rlimitexec.FlagAddressSpace, limits.AddressSpaceBytes))
	}
	argv = append(argv, "--", spec.Path)
	argv = append(argv, spec.Args...)
	cmd := exec.Command(self, argv...)
	cmd.Env = spec.Env
	cmd.Dir = spec.Dir
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &Contained{Cmd: cmd, Method: MethodRlimitShim, after: func(c *Contained) error {
		c.Scope = ScopeRef{Name: spec.Scope, Kind: ScopeKindProcessGroup, PID: c.Cmd.Process.Pid}
		return nil
	}}, nil
}

func rlimitSpec(c Containment) (rlimitexec.Spec, error) {
	physical := physicalMemoryBytes()
	bytes, err := memoryCeilingBytes(c.MemoryMax, physical)
	if err != nil {
		return rlimitexec.Spec{}, err
	}
	return rlimitexec.Spec{AddressSpaceBytes: bytes, MaxProcesses: int64(c.TasksMax)}, nil
}

// physicalMemoryBytes reads hw.memsize; 0 when sysctl is unavailable, which
// makes a percentage ceiling an error rather than a silent no-limit.
func physicalMemoryBytes() int64 {
	output, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	return n
}

// containSelf applies the limits to the calling process; they survive the
// exec that follows. The process group of the caller is the scope.
func containSelf(scope string, c Containment) (ScopeRef, string, error) {
	limits, err := rlimitSpec(c)
	if err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
	}
	if err := rlimitexec.Apply(limits); err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
	}
	return ScopeRef{Name: scope, Kind: ScopeKindProcessGroup, PID: syscall.Getpgrp()}, MethodRlimitShim, nil
}

func freezeScope(ref ScopeRef) error { return signalGroup(ref, syscall.SIGSTOP) }

func thawScope(ref ScopeRef) error { return signalGroup(ref, syscall.SIGCONT) }

func signalGroup(ref ScopeRef, signal syscall.Signal) error {
	if ref.Kind != ScopeKindProcessGroup || ref.PID <= 0 {
		return fmt.Errorf("platform: %s is not a process group scope", ref.String())
	}
	return syscall.Kill(-ref.PID, signal)
}

// scopeFrozen reads the group's task states; a leading T is stopped.
func scopeFrozen(ref ScopeRef) (bool, error) {
	if ref.Kind != ScopeKindProcessGroup || ref.PID <= 0 {
		return false, fmt.Errorf("platform: %s is not a process group scope", ref.String())
	}
	output, err := exec.Command("ps", "-o", "stat=", "-g", strconv.Itoa(ref.PID)).Output()
	if err != nil {
		return false, err
	}
	states := strings.Fields(string(output))
	if len(states) == 0 {
		return false, nil
	}
	for _, state := range states {
		if !strings.HasPrefix(state, "T") {
			return false, nil
		}
	}
	return true, nil
}
