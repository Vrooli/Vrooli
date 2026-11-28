package main

import (
	"errors"
	"fmt"
	"os"
)

// exitCoder is satisfied by errors that carry a documented process exit
// status, for example `conversations wait` returning 3 when the operator asked
// for more context.
type exitCoder interface {
	ExitCode() int
}

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		var coded exitCoder
		if errors.As(err, &coded) {
			os.Exit(coded.ExitCode())
		}
		os.Exit(1)
	}
}
