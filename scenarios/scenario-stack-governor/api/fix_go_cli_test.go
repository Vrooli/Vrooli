package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupGoCliTestDir(t *testing.T, scenarioName string) (repoRoot string) {
	t.Helper()
	root := setupFixTestDir(t)

	// Create scenario with cli and api dirs.
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(filepath.Join(scenarioDir, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "packages", "proto"), 0o755); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestFixGoCli_AddsMissingAPIReplace(t *testing.T) {
	scenarioName := "test-go-cli"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	// Write api/go.mod with module name.
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module github.com/vrooli/test-go-cli/api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write cli/go.mod without replace.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte("module github.com/vrooli/test-go-cli/cli\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a Go file that imports the API module.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte(`package main

import _ "github.com/vrooli/test-go-cli/api/internal/config"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.Fixed {
		t.Fatalf("expected fixed=true, got false; error=%s", r.Error)
	}

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "cli", "go.mod"))
	text := string(content)
	if !strings.Contains(text, "replace github.com/vrooli/test-go-cli/api => ../api") {
		t.Error("expected replace directive for API module")
	}
	if !strings.Contains(text, "github.com/vrooli/test-go-cli/api v0.0.0") {
		t.Error("expected require directive for API module")
	}
}

func TestFixGoCli_AddsMissingProtoReplace(t *testing.T) {
	scenarioName := "test-proto"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	// Write cli/go.mod that references proto but no replace.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte(`module github.com/vrooli/test-proto/cli

go 1.23

require (
	github.com/vrooli/vrooli/packages/proto v0.0.0
)
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "cli", "go.mod"))
	text := string(content)
	if !strings.Contains(text, "replace github.com/vrooli/vrooli/packages/proto =>") {
		t.Error("expected replace directive for proto module")
	}
}

func TestFixGoCli_Idempotent(t *testing.T) {
	scenarioName := "test-idempotent"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module github.com/vrooli/test-idempotent/api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write cli/go.mod that already has the correct directives.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte(`module github.com/vrooli/test-idempotent/cli

go 1.23

require (
	github.com/vrooli/test-idempotent/api v0.0.0
)

replace github.com/vrooli/test-idempotent/api => ../api
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte(`package main

import _ "github.com/vrooli/test-idempotent/api/internal/config"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fixed {
		t.Error("expected fixed=false when already correct")
	}
}

func TestFixGoCli_DryRunDoesNotWrite(t *testing.T) {
	scenarioName := "test-dryrun"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module github.com/vrooli/test-dryrun/api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	original := "module github.com/vrooli/test-dryrun/cli\n\ngo 1.23\n"
	goModPath := filepath.Join(scenarioDir, "cli", "go.mod")
	if err := os.WriteFile(goModPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte(`package main

import _ "github.com/vrooli/test-dryrun/api/internal/config"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, true)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Fixed {
		t.Error("expected fixed=true in dry-run mode")
	}

	// File should be unchanged.
	content, _ := os.ReadFile(goModPath)
	if string(content) != original {
		t.Error("expected go.mod to be unchanged in dry-run mode")
	}
}
