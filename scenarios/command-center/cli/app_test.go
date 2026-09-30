package main

import (
	"strings"
	"testing"
)

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}
	if app == nil {
		t.Fatal("NewApp returned nil app")
	}
	if app.core == nil {
		t.Fatal("NewApp did not wire core")
	}
}

func TestAppRunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}
	// Running with --help must succeed and not panic.
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("Run --help returned error: %v", err)
	}
}

func TestAppMetadata(t *testing.T) {
	if appName != "command-center" {
		t.Errorf("appName = %q, want command-center", appName)
	}
	if !strings.HasPrefix(appVersion, "0.") && !strings.HasPrefix(appVersion, "1.") {
		t.Errorf("appVersion = %q, want semver-like value", appVersion)
	}
}
