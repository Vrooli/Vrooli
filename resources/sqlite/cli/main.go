package main

import (
	"fmt"
	"os"

	resourceapp "github.com/vrooli/resources/sqlite/cli/internal/app"
)

const (
	appName    = "resource-sqlite"
	appVersion = "0.2.0"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	app, err := resourceapp.BuildCommandApp(resourceapp.BuildInfo{
		Name:         appName,
		Version:      appVersion,
		Description:  "Serverless SQLite resource",
		Fingerprint:  buildFingerprint,
		Timestamp:    buildTimestamp,
		SourceRoot:   buildSourceRoot,
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
