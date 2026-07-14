package main

import (
	"fmt"
	"os"

	"workspace-sandbox/internal/rlimitexec"
)

// init intercepts the rlimit-exec self-exec shim invocation before main runs.
// When the api binary is invoked as `workspace-sandbox-api rlimit-exec … --
// <cmd> …` — which the macOS Seatbelt backend prepends ahead of a workload as
// the portable replacement for Linux-only prlimit — the process applies
// setrlimit and execs the target instead of starting the server.
//
// This lives in init() rather than main() on purpose: the shim is an
// orthogonal entry mode, not part of server startup, so main() keeps
// preflight.Run as its first statement. Running before main also means the
// shim never hits preflight's staleness rebuild (which would fail inside a
// write-contained sandbox) or its lifecycle guard (which would os.Exit a
// process that is not lifecycle-managed). For a normal server start or
// `go test`, os.Args[1] is never the shim subcommand, so this is a no-op.
func init() {
	if len(os.Args) <= 1 || os.Args[1] != rlimitexec.Subcommand {
		return
	}
	if err := rlimitexec.Run(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "workspace-sandbox-api %s: %v\n", rlimitexec.Subcommand, err)
		os.Exit(1)
	}
	// Unreachable on success: rlimitexec.Run's exec replaces the process image.
	os.Exit(0)
}
