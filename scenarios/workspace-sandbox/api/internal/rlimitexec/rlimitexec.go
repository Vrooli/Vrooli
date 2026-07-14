// Package rlimitexec implements the api binary's `rlimit-exec` self-exec
// shim. On Linux resource limits are applied by prepending the system
// `prlimit` binary (see driver/exec/args.go); prlimit is Linux-only, so on
// macOS the Seatbelt backend prepends this shim instead. The shim applies
// setrlimit for the requested limits and then execs the target command,
// replacing its own process image so the limits carry into the workload.
//
// Layering: the flag parsing and the Spec -> limit mapping live here and are
// OS-neutral so they unit-test on Linux. Only the raw setrlimit/exec syscalls
// are build-tagged (apply_unix.go for linux/darwin, apply_other.go elsewhere).
package rlimitexec

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// Subcommand is the argv[1] token that routes the api binary into the shim.
// The Seatbelt backend prepends "<api-binary> rlimit-exec …" ahead of the
// target command, mirroring how BuildExecCommand prepends prlimit on Linux.
const Subcommand = "rlimit-exec"

// Flag names for the shim's resource-limit flags. Defined here so the
// Seatbelt backend that produces them (driver/exec) and the parser that
// consumes them stay in lockstep.
const (
	FlagAddressSpace = "as"     // RLIMIT_AS, in bytes
	FlagCPUTime      = "cpu"    // RLIMIT_CPU, in seconds
	FlagMaxProcesses = "nproc"  // RLIMIT_NPROC, count
	FlagMaxOpenFiles = "nofile" // RLIMIT_NOFILE, count
)

// Spec holds the resource limits the shim applies. A zero field means "no
// limit for this resource" (the field is skipped, mirroring prlimit and
// ResourceLimits.HasLimits on the Linux path).
type Spec struct {
	// AddressSpaceBytes maps to RLIMIT_AS.
	AddressSpaceBytes int64
	// CPUTimeSec maps to RLIMIT_CPU.
	CPUTimeSec int64
	// MaxProcesses maps to RLIMIT_NPROC.
	MaxProcesses int64
	// MaxOpenFiles maps to RLIMIT_NOFILE.
	MaxOpenFiles int64
}

// limitKind is the OS-neutral identity of one applied resource limit. The
// build-tagged apply layer maps each kind to the platform's RLIMIT_* value.
type limitKind string

const (
	limitAddressSpace limitKind = "address-space"
	limitCPUTime      limitKind = "cpu-time"
	limitProcesses    limitKind = "processes"
	limitOpenFiles    limitKind = "open-files"
)

// limitValue pairs a resource kind with the (soft==hard) value to set.
type limitValue struct {
	Kind  limitKind
	Value uint64
}

// Limits returns the resource limits to apply, in a stable order, for the
// Spec fields that are set (> 0). Pure and OS-neutral so the mapping is
// unit-tested on Linux; the apply layer only translates each kind to a
// syscall constant.
func (s Spec) Limits() []limitValue {
	var out []limitValue
	if s.AddressSpaceBytes > 0 {
		out = append(out, limitValue{limitAddressSpace, uint64(s.AddressSpaceBytes)})
	}
	if s.CPUTimeSec > 0 {
		out = append(out, limitValue{limitCPUTime, uint64(s.CPUTimeSec)})
	}
	if s.MaxProcesses > 0 {
		out = append(out, limitValue{limitProcesses, uint64(s.MaxProcesses)})
	}
	if s.MaxOpenFiles > 0 {
		out = append(out, limitValue{limitOpenFiles, uint64(s.MaxOpenFiles)})
	}
	return out
}

// ParseArgs parses the shim flags that precede the "--" separator and
// returns the parsed Spec plus the target command (the argv after "--").
// It is OS-neutral and the single source of truth for the shim's flag
// grammar; the apply layer never re-parses.
func ParseArgs(args []string) (Spec, []string, error) {
	fs := flag.NewFlagSet(Subcommand, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	as := fs.Int64(FlagAddressSpace, 0, "max address space in bytes (RLIMIT_AS)")
	cpu := fs.Int64(FlagCPUTime, 0, "max CPU time in seconds (RLIMIT_CPU)")
	nproc := fs.Int64(FlagMaxProcesses, 0, "max processes (RLIMIT_NPROC)")
	nofile := fs.Int64(FlagMaxOpenFiles, 0, "max open files (RLIMIT_NOFILE)")
	if err := fs.Parse(args); err != nil {
		return Spec{}, nil, fmt.Errorf("parse rlimit flags: %w", err)
	}
	target := fs.Args()
	if len(target) == 0 {
		return Spec{}, nil, errors.New("rlimit-exec: no target command after --; usage: rlimit-exec [--as N] [--cpu N] [--nproc N] [--nofile N] -- <cmd> [args...]")
	}
	spec := Spec{
		AddressSpaceBytes: *as,
		CPUTimeSec:        *cpu,
		MaxProcesses:      *nproc,
		MaxOpenFiles:      *nofile,
	}
	return spec, target, nil
}

// Run parses args, applies the resource limits, and execs the target. On
// success it does not return: applyAndExec replaces the process image. Any
// return value is therefore always an error suitable for the caller to
// print and exit non-zero on.
func Run(args []string) error {
	spec, target, err := ParseArgs(args)
	if err != nil {
		return err
	}
	return applyAndExec(spec, target)
}
