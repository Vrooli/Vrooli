package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(runExitCode(os.Args[1:], os.Stderr))
}

func runMain(args []string) error {
	app, err := NewApp()
	if err != nil {
		return err
	}
	return app.Run(args)
}

// runExitCode reports the process exit status and, crucially, says why.
//
// This used to extract the exit code from the error and then drop the error on
// the floor, so every failure -- an unreachable API, a rejected selection, a
// connection lost mid-apply -- surfaced to the operator as a bare exit 1 with
// no output whatsoever. During the onboarding wizard that is worse than
// unhelpful: the operator answers "apply this selection now?", the command
// vanishes, and nothing on screen distinguishes "applied successfully" from
// "could not reach the API at all".
//
// Every other scenario CLI in the repository writes the error to stderr; this
// one was the sole exception.
func runExitCode(args []string, stderr io.Writer) int {
	err := runMain(args)
	if err == nil {
		return 0
	}
	if stderr != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
	}
	if coded, ok := err.(interface{ ExitCode() int }); ok {
		return coded.ExitCode()
	}
	return 1
}
