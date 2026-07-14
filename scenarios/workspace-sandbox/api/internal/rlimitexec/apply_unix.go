//go:build linux || darwin

package rlimitexec

import (
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"syscall"
)

// applyAndExec sets each requested resource limit via setrlimit, then
// replaces the current process image with the target command. This file is
// an OS-split seam: the raw setrlimit/exec syscalls are confined here (see
// the launcher's exec_unix.go for the same pattern), keeping the shim's
// parsing and mapping logic OS-neutral and Linux-testable.
func applyAndExec(spec Spec, target []string) error {
	for _, lim := range spec.Limits() {
		resource, ok := resourceFor(lim.Kind)
		if !ok {
			continue
		}
		rl := syscall.Rlimit{Cur: lim.Value, Max: lim.Value}
		if err := syscall.Setrlimit(resource, &rl); err != nil {
			return fmt.Errorf("setrlimit %s=%d: %w", lim.Kind, lim.Value, err)
		}
	}

	argv0 := target[0]
	if !strings.ContainsRune(argv0, '/') {
		resolved, err := osexec.LookPath(argv0)
		if err != nil {
			return fmt.Errorf("resolve target command %q: %w", argv0, err)
		}
		argv0 = resolved
	}
	if err := syscall.Exec(argv0, target, os.Environ()); err != nil {
		return fmt.Errorf("exec target command %q: %w", argv0, err)
	}
	return nil
}

// resourceFor maps an OS-neutral limit kind to the platform's RLIMIT_*
// constant. RLIMIT_AS/CPU/NOFILE are exported by the Go syscall package on
// both linux and darwin; RLIMIT_NPROC differs (rlimitNProc is provided by a
// per-OS constant file).
func resourceFor(kind limitKind) (int, bool) {
	switch kind {
	case limitAddressSpace:
		return syscall.RLIMIT_AS, true
	case limitCPUTime:
		return syscall.RLIMIT_CPU, true
	case limitProcesses:
		return rlimitNProc, true
	case limitOpenFiles:
		return syscall.RLIMIT_NOFILE, true
	}
	return 0, false
}
