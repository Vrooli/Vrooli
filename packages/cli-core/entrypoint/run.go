// Package entrypoint owns the common process boundary for Vrooli CLI apps.
package entrypoint

import (
	"fmt"
	"io"
)

// Runner is implemented by a CLI application that accepts its command-line
// arguments and returns an error for the executable to report.
type Runner interface {
	Run([]string) error
}

// RunAndExit creates an app, runs it with args, and reports failures using the
// shared executable contract. exit is injected so the lifecycle can be tested
// without terminating the test process.
func RunAndExit[T Runner](newApp func() (T, error), args []string, stderr io.Writer, exit func(int)) {
	app, err := newApp()
	if err == nil {
		err = app.Run(args)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		exit(1)
	}
}
