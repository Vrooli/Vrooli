// Package main is the entry point for the Test Genie CLI.
package main

import (
	"errors"
	"fmt"
	"os"
)

// exitCoder is satisfied by errors carrying a documented exit code (e.g. the
// `eligibility check` subcommand which distinguishes routed/not-routed/
// unreachable via 0/1/2).
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
		var ec exitCoder
		if errors.As(err, &ec) {
			os.Exit(ec.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
