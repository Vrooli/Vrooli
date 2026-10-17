package main

import (
	"fmt"
	"os"

	"scenario-to-cloud/cli/internal/apierr"
)

// main runs the installed scenario CLI. The binary is an operator command
// (installed by the lifecycle as `scenario-to-cloud`), not a lifecycle-managed
// service, so it must run directly; the API's lifecycle guard does not apply
// here.
func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(apierr.ExitFailed)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		// Typed API outcomes carry their own exit code: 2 refused/conflict,
		// 3 pending/needs-input, 124 observer timeout; everything else is 1.
		fmt.Fprintln(os.Stderr, apierr.Format(err))
		os.Exit(apierr.ExitCode(err))
	}
}
