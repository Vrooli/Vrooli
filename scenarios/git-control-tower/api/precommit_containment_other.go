//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package main

import "os/exec"

// applyProcessGroupContainment is a no-op where POSIX process groups do not
// exist. exec.CommandContext still kills the direct child on cancel, and
// shellCommandWaitDelay still forces inherited pipes closed so Wait returns —
// but descendants are not guaranteed to be reaped.
//
// This makes the package compile for these platforms; it does not make
// git-control-tower work on them. The precommit runner shells to `bash -lc`,
// which is a separate and larger portability question than process groups.
func applyProcessGroupContainment(_ *exec.Cmd) {}
