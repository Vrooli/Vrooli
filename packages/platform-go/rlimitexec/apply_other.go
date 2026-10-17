//go:build !linux && !darwin

package rlimitexec

import "fmt"

// applyAndExec is unsupported off linux/darwin: setrlimit + exec-replace has
// no portable equivalent on Windows, and the shim is only ever prepended by
// the macOS Seatbelt backend. Returning an error keeps the cross-compile
// gate green without pretending to enforce limits.
func applyAndExec(_ Spec, _ []string) error {
	return fmt.Errorf("rlimit-exec is not supported on this platform")
}

func apply(Spec) error { return fmt.Errorf("rlimit-exec is not supported on this platform") }
