package main

import (
	"testing"

	resourceapp "github.com/vrooli/vrooli/resources/doc-ocr/cli/internal/app"
)

func TestBuildCommandAppConfiguresCLI(t *testing.T) {
	app, err := resourceapp.BuildCommandApp(resourceapp.BuildInfo{
		Name:        appName,
		Version:     appVersion,
		Description: "Doc OCR resource CLI",
		Fingerprint: buildFingerprint,
		Timestamp:   buildTimestamp,
		SourceRoot:  buildSourceRoot,
	})
	if err != nil {
		t.Fatalf("BuildCommandApp() error = %v", err)
	}
	if app == nil {
		t.Fatal("BuildCommandApp() returned nil app")
	}
	if err := app.Run(nil); err != nil {
		t.Fatalf("app.Run(nil) error = %v", err)
	}
}
