package main

import (
	"fmt"
	"os"

	resourceapp "github.com/vrooli/vrooli/resources/openrouter/cli/internal/app"
	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/health"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	app, err := resourceapp.New(buildFingerprint, buildTimestamp, buildSourceRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		// A typed provider failure crosses the resource-command boundary as a
		// single machine-readable stderr line so a subprocess caller (the AI
		// Gateway) preserves the observed status and Retry-After instead of
		// collapsing it into a generic exit error.
		if providerErr, ok := health.AsProviderError(err); ok {
			if line, markerErr := providerErr.MarkerLine(); markerErr == nil {
				fmt.Fprintln(os.Stderr, string(line))
			}
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
