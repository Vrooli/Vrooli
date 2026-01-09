package infra

import (
	"os"
	"syscall"
)

// Signal constants for cross-platform compatibility.

// Interrupt is os.Interrupt for use in Process.Signal calls.
var Interrupt = os.Interrupt

// Kill signal for forceful termination.
var Kill = syscall.SIGKILL
