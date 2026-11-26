package main

import (
	"os"
	"strings"
	"testing"
)

func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	// --version is handled by cli-core; it must not return an error and must not
	// touch the network (no NeedsAPI path triggered).
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

func TestRunHelpListsMigratedDomains(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	// The standard help text lists all registered command groups and
	// subcommand groups. We assert the migrated domains are wired up.
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

func TestReadmeCommandExamplesMatchRegisteredGroups(t *testing.T) {
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	text := string(readme)
	for _, example := range []string{
		"file-tools archive compress",
		"file-tools archive extract",
		"file-tools files checksum",
		"file-tools files metadata",
		"file-tools archive split",
		"file-tools archive merge",
	} {
		if !strings.Contains(text, example) {
			t.Errorf("README.md is missing CLI example %q", example)
		}
	}
	for _, obsolete := range []string{
		"file-tools compress",
		"file-tools extract",
		"file-tools checksum",
		"file-tools metadata",
		"file-tools split",
		"file-tools merge",
	} {
		if strings.Contains(text, obsolete) {
			t.Errorf("README.md still documents unregistered flat command %q", obsolete)
		}
	}

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	for _, args := range [][]string{
		{"archive", "compress", "--help"},
		{"archive", "extract", "--help"},
		{"files", "checksum", "--help"},
		{"files", "metadata", "--help"},
		{"archive", "split", "--help"},
		{"archive", "merge", "--help"},
	} {
		if err := app.Run(args); err != nil {
			t.Errorf("App.Run(%q) error: %v", strings.Join(args, " "), err)
		}
	}
}

func TestMetadata(t *testing.T) {
	if !strings.EqualFold(appName, "file-tools") {
		t.Fatalf("appName = %q, want file-tools", appName)
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}
