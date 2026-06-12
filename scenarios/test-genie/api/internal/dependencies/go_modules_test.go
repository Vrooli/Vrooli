package dependencies

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoModuleCheckerFailsMissingLocalReplace(t *testing.T) {
	scenarioDir := t.TempDir()
	apiDir := filepath.Join(scenarioDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(apiDir, "go.mod"), `module demo

go 1.25.0

replace github.com/vrooli/missing => ../../../packages/missing
`)

	settings := DefaultSettings().GoModules
	settings.TidyDiff = false
	result := NewGoModuleChecker(scenarioDir, settings).Check(context.Background())
	if result.Success {
		t.Fatal("expected missing local replace failure")
	}
	if !strings.Contains(result.Error.Error(), "local replace targets are missing") {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}

func TestGoModuleCheckerPassesWithoutGoModule(t *testing.T) {
	result := NewGoModuleChecker(t.TempDir(), DefaultSettings().GoModules).Check(context.Background())
	if !result.Success {
		t.Fatalf("expected success for non-Go scenario, got %v", result.Error)
	}
}
