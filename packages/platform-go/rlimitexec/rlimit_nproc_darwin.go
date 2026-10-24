//go:build darwin

package rlimitexec

// rlimitNProc is the RLIMIT_NPROC constant from macOS <sys/resource.h>. The
// Go syscall package does not export it on darwin, so it is defined here;
// the value is stable kernel ABI.
const rlimitNProc = 7
