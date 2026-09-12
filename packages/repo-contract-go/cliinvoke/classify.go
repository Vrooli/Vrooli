package cliinvoke

import (
	"context"
	"errors"
	"io/fs"
	"os/exec"
	"strings"
)

// Class is what happened to an invocation, reduced to the categories a
// supervisor must treat differently. A supervisor retries Lifecycle and
// Timeout; it must not retry Usage or BinaryMissing identically, because
// nothing about the host changes between attempts.
type Class int

const (
	// OK: exit status zero.
	OK Class = iota
	// Usage: the CLI rejected the argv (unknown command, unknown option,
	// missing value). The invoker is out of contract with the installed CLI.
	Usage
	// BinaryMissing: the binary could not be started at all.
	BinaryMissing
	// Timeout: the caller's deadline expired before the command exited.
	Timeout
	// Refusal: the CLI ran but refused because a store it depends on is
	// locked, uninitialized, or unresponsive; a recovery command is named.
	Refusal
	// Lifecycle: any other non-zero exit, i.e. the command ran and reported
	// a real failure that a retry or a heal can address.
	Lifecycle
)

func (c Class) String() string {
	switch c {
	case OK:
		return "ok"
	case Usage:
		return "usage"
	case BinaryMissing:
		return "binary-missing"
	case Timeout:
		return "timeout"
	case Refusal:
		return "refusal"
	case Lifecycle:
		return "lifecycle"
	}
	return "unknown"
}

// Retryable reports whether an identical retry could reasonably succeed.
func (c Class) Retryable() bool {
	return c == Timeout || c == Lifecycle
}

// usageMarkers are the labels the vrooli root parser prints for argv errors.
// They are matched case-insensitively against stderr. An exit code alone
// cannot identify a usage error because the CLI exits 1 for every category.
var usageMarkers = []string{
	"unknown command",
	"unknown scenario command",
	"unknown option",
	"usage error",
	"flag provided but not defined",
	"missing value for --",
}

// refusalMarkers mirror the credential-store refusal states the vrooli-bridge
// agent classifies; the agent keeps the recovery text, this package keeps
// only the class.
var refusalMarkers = []string{
	"uninitialized",
	"not initialized",
	"credential store is locked",
	"keyring is locked",
	"unresponsive",
}

// Classify turns an exec result into a Class. stderr is the captured error
// stream (or combined output when the caller merged them).
func Classify(ctx context.Context, err error, stderr []byte) Class {
	if err == nil {
		return OK
	}
	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Timeout
	}
	var execErr *exec.Error
	var pathErr *fs.PathError
	if errors.As(err, &execErr) || errors.As(err, &pathErr) || errors.Is(err, exec.ErrNotFound) || errors.Is(err, ErrBinaryMissing) {
		// exec.Error is LookPath's miss; fs.PathError is fork/exec's ENOENT
		// or EACCES on the resolved path. Both mean the binary never ran.
		return BinaryMissing
	}
	text := strings.ToLower(string(stderr))
	for _, marker := range usageMarkers {
		if strings.Contains(text, marker) {
			return Usage
		}
	}
	for _, marker := range refusalMarkers {
		if strings.Contains(text, marker) {
			return Refusal
		}
	}
	return Lifecycle
}
