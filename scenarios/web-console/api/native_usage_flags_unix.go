//go:build unix

package main

import "syscall"

const nativeNonblockingOpenFlag = syscall.O_NONBLOCK
