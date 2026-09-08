//go:build unix

package api

import "syscall"

// O_NONBLOCK does not change regular-file reads, but prevents a raced FIFO
// replacement from blocking open before the descriptor's type can be checked.
const logSnapshotOpenFlags = syscall.O_NONBLOCK
