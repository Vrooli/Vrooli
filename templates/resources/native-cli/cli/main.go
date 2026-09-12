package main

import (
	"fmt"
	"os"

	resourceapp "{{RESOURCE_CLI_COMMAND}}/cli/internal/app"
)

const (
	appName    = "{{RESOURCE_CLI_COMMAND}}"
	appVersion = "0.1.0"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	app, err := resourceapp.BuildCommandApp(resourceapp.BuildInfo{
		Name:        appName,
		Version:     appVersion,
		Description: "{{RESOURCE_DISPLAY_NAME}} resource CLI",
		Fingerprint: buildFingerprint,
		Timestamp:   buildTimestamp,
		SourceRoot:  buildSourceRoot,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
