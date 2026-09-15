package main

import (
	"fmt"
	"os"

	resourceapp "github.com/vrooli/vrooli/resources/doc-ocr/cli/internal/app"
)

const (
	appName    = "resource-doc-ocr"
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
		Description: "Doc OCR resource CLI",
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
