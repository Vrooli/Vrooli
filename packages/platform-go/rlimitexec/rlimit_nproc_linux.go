//go:build linux

package rlimitexec

// rlimitNProc is the RLIMIT_NPROC constant from Linux <bits/resource.h>. The
// Go syscall package does not export it (only golang.org/x/sys/unix does, and
// this module takes no new dependencies), so it is defined here; the value is
// stable kernel ABI.
const rlimitNProc = 6
