package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
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
	if r.Diff != nil {
		t.Error("expected Diff to be nil on non-dry-run")
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

// TestFixGoCli_IgnoresStringLiteralWithModuleName verifies that the fix does NOT
// add API wiring when the CLI merely uses the module name as a string constant
// (e.g. `appName = "my-scenario"`) without actually importing a subpackage.
// This was a real-world false positive affecting 11+ scenarios.
func TestFixGoCli_IgnoresStringLiteralWithModuleName(t *testing.T) {
	scenarioName := "my-scenario"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	// API module has a simple name that could also appear as a string literal.
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module my-scenario\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte("module my-scenario/cli\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Go file uses the module name as a string constant — NOT an import.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "app.go"), []byte(`package main

const appName = "my-scenario"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fixed {
		t.Error("expected fixed=false — string literal should NOT trigger API wiring")
	}

	// Verify go.mod was NOT modified.
	content, _ := os.ReadFile(filepath.Join(scenarioDir, "cli", "go.mod"))
	if strings.Contains(string(content), "replace my-scenario =>") {
		t.Error("go.mod should NOT have a replace directive for the API module")
	}
}

// TestFixGoCli_IgnoresTestStringContainingModuleName checks that test files
// using the module name as a test argument don't trigger false positives.
func TestFixGoCli_IgnoresTestStringContainingModuleName(t *testing.T) {
	scenarioName := "dev-toolchain"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module dev-toolchain\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte("module dev-toolchain/cli\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test file references the module name as a test data string.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "app_test.go"), []byte(`package main

import "testing"

func TestApp(t *testing.T) {
	err := app.Run([]string{"dev-toolchain", "--help"})
	if err != nil { t.Fatal(err) }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)

	if results[0].Fixed {
		t.Error("expected fixed=false — test string literal should NOT trigger API wiring")
	}
}

// TestFixGoCli_DetectsSubpackageImport verifies the fix correctly detects
// when CLI code imports a subpackage of the API module (the real use case).
func TestFixGoCli_DetectsSubpackageImport(t *testing.T) {
	scenarioName := "real-import"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module github.com/vrooli/real-import/api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte("module github.com/vrooli/real-import/cli\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// CLI imports a subpackage (models, not internal — still needs wiring).
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte(`package main

import _ "github.com/vrooli/real-import/api/models"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)

	if !results[0].Fixed {
		t.Fatal("expected fixed=true — subpackage import should trigger API wiring")
	}

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "cli", "go.mod"))
	text := string(content)
	if !strings.Contains(text, "replace github.com/vrooli/real-import/api => ../api") {
		t.Error("expected replace directive for API module")
	}
	if !strings.Contains(text, "github.com/vrooli/real-import/api v0.0.0") {
		t.Error("expected require directive for API module")
	}
}

// TestFixGoCli_BareModuleNameInQuotesNoSlash verifies that a quoted reference
// to the exact module name (without trailing slash) does not trigger wiring.
func TestFixGoCli_BareModuleNameInQuotesNoSlash(t *testing.T) {
	scenarioName := "bare-name"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module bare-name\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte("module bare-name/cli\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Multiple files referencing the module name as strings, but never importing subpackages.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "app.go"), []byte(`package main

const appName = "bare-name"
const prompt = "Welcome to bare-name CLI"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "version.go"), []byte(`package main

const version = "bare-name v1.0.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)
	if results[0].Fixed {
		t.Error("expected fixed=false — bare module name in strings should not trigger wiring")
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

	// Diff should be populated.
	if results[0].Diff == nil {
		t.Fatal("expected Diff to be populated in dry-run")
	}
	if results[0].Diff.Before != original {
		t.Error("expected Diff.Before to equal original go.mod content")
	}
	if !strings.Contains(results[0].Diff.After, "replace") {
		t.Error("expected Diff.After to contain 'replace'")
	}
}

// TestFixGoCli_ExistingReplaceBlock verifies the fixer correctly adds directives
// into go.mod files that already have replace/require blocks.
func TestFixGoCli_ExistingReplaceBlock(t *testing.T) {
	scenarioName := "existing-block"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"),
		[]byte("module github.com/vrooli/existing-block/api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// CLI go.mod with an existing replace block for a different module.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte(`module github.com/vrooli/existing-block/cli

go 1.23

replace (
	example.com/other => ../other
)
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte(`package main

import _ "github.com/vrooli/existing-block/api/internal/config"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "cli", "go.mod"))
	text := string(content)

	// Both the existing replace and the new one should be present.
	if !strings.Contains(text, "example.com/other => ../other") {
		t.Error("expected existing replace to be preserved")
	}
	if !strings.Contains(text, "github.com/vrooli/existing-block/api => ../api") {
		t.Error("expected new replace directive for API module")
	}

	// Verify the output is valid go.mod syntax by re-parsing.
	if _, err := modfile.Parse("go.mod", content, nil); err != nil {
		t.Errorf("fixed go.mod is not valid: %v", err)
	}
}

// TestFixGoCli_CommentedReplaceBlock verifies the fixer handles go.mod files
// with comments near replace/require blocks (a case that broke string-level patching).
func TestFixGoCli_CommentedReplaceBlock(t *testing.T) {
	scenarioName := "commented-block"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"),
		[]byte("module github.com/vrooli/commented-block/api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// go.mod with comments inside blocks.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte(`module github.com/vrooli/commented-block/cli

go 1.23

require (
	// existing dependency
	example.com/dep v1.0.0
)
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte(`package main

import _ "github.com/vrooli/commented-block/api/models"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "cli", "go.mod"))
	// Verify output is valid go.mod.
	if _, err := modfile.Parse("go.mod", content, nil); err != nil {
		t.Errorf("fixed go.mod is not valid: %v", err)
	}
	if !strings.Contains(string(content), "github.com/vrooli/commented-block/api") {
		t.Error("expected API module directives to be added")
	}
}

// TestFixGoCli_InlineCommentInModule verifies that a go.mod with an inline
// comment on the module line is parsed correctly.
func TestFixGoCli_InlineCommentInModule(t *testing.T) {
	scenarioName := "inline-comment"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	// API go.mod with inline comment.
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"),
		[]byte("module github.com/vrooli/inline-comment/api // generated\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"),
		[]byte("module github.com/vrooli/inline-comment/cli\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// CLI imports the API.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte(`package main

import _ "github.com/vrooli/inline-comment/api/internal/config"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Fixed {
		t.Fatal("expected fixed=true")
	}

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "cli", "go.mod"))
	text := string(content)
	// The replace should use the clean module path, not include the comment.
	if !strings.Contains(text, "replace github.com/vrooli/inline-comment/api => ../api") {
		t.Error("expected replace with clean module path (no comment)")
	}
	if strings.Contains(text, "// generated") {
		t.Error("module comment should not leak into replace directive")
	}
}

// TestFixGoCli_ProtoCommentDoesNotTriggerFix verifies that a go.mod with
// the proto module path appearing only in a comment does NOT trigger a fix.
// This was a false positive caused by raw string matching on the go.mod text.
func TestFixGoCli_ProtoCommentDoesNotTriggerFix(t *testing.T) {
	scenarioName := "proto-comment-fix"
	root := setupGoCliTestDir(t, scenarioName)
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	goMod := `module github.com/vrooli/proto-comment-fix/cli

go 1.23

// TODO: add github.com/vrooli/vrooli/packages/proto dependency later
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixGoCliWorkspaceIndependence(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fixed {
		t.Error("expected fixed=false — proto in comment should not trigger fix")
	}

	// Verify go.mod was NOT modified.
	content, _ := os.ReadFile(filepath.Join(scenarioDir, "cli", "go.mod"))
	if strings.Contains(string(content), "replace") {
		t.Error("go.mod should NOT have a replace directive added from a comment reference")
	}
}
