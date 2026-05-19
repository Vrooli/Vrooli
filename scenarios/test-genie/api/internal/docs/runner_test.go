package docs

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"test-genie/internal/orchestrator/workspace"
)

// docsManifestRel is the canonical scenario-relative manifest path resolved
// through the repo contract — used in place of the literal path string to
// satisfy the audit test in packages/repo-contract-go/audit_test.go.
var docsManifestRel = defaultManifestRel()

func TestRunner_Success(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf(
		"# Title\n\nSee [local](./local.md).\n\n```mermaid\ngraph TB\n  A --> B\n```\n\n[external](%s)\n",
		server.URL,
	))
	writeFile(t, filepath.Join(dir, "local.md"), "ok\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(server.Client()))

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.FilesChecked != 2 {
		t.Fatalf("expected 2 files checked, got %d", result.Summary.FilesChecked)
	}
	if result.Summary.BrokenLinks != 0 {
		t.Fatalf("expected no broken links, got %d", result.Summary.BrokenLinks)
	}
	if result.Summary.MermaidFailures != 0 {
		t.Fatalf("expected mermaid success, got %d failures", result.Summary.MermaidFailures)
	}
}

func TestRunner_UnclosedFenceFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "```\nunclosed\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure due to unclosed fence")
	}
	if result.Summary.MarkdownFailures != 1 {
		t.Fatalf("expected 1 markdown failure, got %d", result.Summary.MarkdownFailures)
	}
}

func TestRunner_BrokenLocalLink(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "broken [link](missing.md)\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure due to broken link")
	}
	if result.Summary.BrokenLinks != 1 {
		t.Fatalf("expected 1 broken link, got %d", result.Summary.BrokenLinks)
	}
}

func TestRunner_MermaidWarningWhenNotStrict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "```mermaid\nnot a diagram\n```\n")

	settings := DefaultSettings()
	strict := false
	settings.Mermaid.Strict = &strict

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatal("expected success (warning only) when mermaid strict disabled")
	}
	if result.Summary.MarkdownWarnings == 0 {
		t.Fatalf("expected mermaid warning, got summary %+v", result.Summary)
	}
}

func TestRunner_MarkdownDisabledSkipsValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "```\nunclosed\n")

	settings := DefaultSettings()
	disabled := false
	settings.Markdown.Enabled = &disabled

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success when markdown validation disabled, got failure: %+v", result)
	}
	if result.Summary.MarkdownFailures != 0 {
		t.Fatalf("expected no markdown failures when disabled, got %d", result.Summary.MarkdownFailures)
	}
}

func TestRunner_AbsolutePathAllowlist(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See /home/test/docs for details\n")

	settings := DefaultSettings()
	settings.Paths.Allow = []string{"/home/"}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with allowlisted absolute path, got failure: %+v", result)
	}
	if result.Summary.AbsolutePathHits != 1 {
		t.Fatalf("expected 1 absolute path hit, got %d", result.Summary.AbsolutePathHits)
	}
	if result.Summary.AbsoluteFailures != 0 {
		t.Fatalf("expected no absolute path failures when allowlisted, got %d", result.Summary.AbsoluteFailures)
	}
}

func TestRunner_ExcludesArchivedMarkdownByGlob(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Active Docs\n")
	writeFile(t, filepath.Join(dir, "ideas/web-console-archived/README.md"), "broken [link](missing.md)\n")

	settings := DefaultSettings()
	settings.ScanPaths.ExcludeGlobs = []string{"ideas/*-archived/**"}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with archived docs excluded, got failure: %+v", result)
	}
	if result.Summary.FilesChecked != 1 {
		t.Fatalf("expected only 1 markdown file checked, got %d", result.Summary.FilesChecked)
	}
}

func TestRunner_ExcludesArchivedCodeFromDocRefScan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Active Docs\n")
	writeFile(t, filepath.Join(dir, "ideas/web-console-archived/helper.go"), "package archived\n// DOC: docs/missing.md\n")

	settings := DefaultSettings()
	settings.ScanPaths.ExcludeGlobs = []string{"ideas/*-archived/**"}
	strict := true
	settings.References.Strict = &strict

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with excluded code refs, got failure: %+v", result)
	}
	if result.Summary.DocRefsFound != 0 {
		t.Fatalf("expected no doc refs found due to exclusion, got %d", result.Summary.DocRefsFound)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

// --- Bidirectional Reference Tests ---

func TestRunner_ValidCodeReference(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: src/main.go] for details\n")
	writeFile(t, filepath.Join(dir, "src/main.go"), "package main\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.CodeRefsFound != 1 {
		t.Fatalf("expected 1 code ref found, got %d", result.Summary.CodeRefsFound)
	}
	if result.Summary.CodeRefsBroken != 0 {
		t.Fatalf("expected 0 broken code refs, got %d", result.Summary.CodeRefsBroken)
	}
}

func TestRunner_BrokenCodeReference(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: missing/file.go] for details\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	// Default is warning mode, so should still succeed
	if !result.Success {
		t.Fatalf("expected success (warning only), got failure: %+v", result)
	}
	if result.Summary.CodeRefsFound != 1 {
		t.Fatalf("expected 1 code ref found, got %d", result.Summary.CodeRefsFound)
	}
	if result.Summary.CodeRefsBroken != 1 {
		t.Fatalf("expected 1 broken code ref, got %d", result.Summary.CodeRefsBroken)
	}
}

func TestRunner_BrokenCodeReferenceStrict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: missing/file.go] for details\n")

	settings := DefaultSettings()
	settings.References.Strict = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure in strict mode")
	}
	if result.Summary.CodeRefsBroken != 1 {
		t.Fatalf("expected 1 broken code ref, got %d", result.Summary.CodeRefsBroken)
	}
}

func TestRunner_CodeRefWithFunction(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: src/main.go#HandleRequest] for details\n")
	writeFile(t, filepath.Join(dir, "src/main.go"), "package main\nfunc HandleRequest() {}\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.CodeRefsFound != 1 {
		t.Fatalf("expected 1 code ref found, got %d", result.Summary.CodeRefsFound)
	}
	if result.Summary.CodeRefsBroken != 0 {
		t.Fatalf("expected 0 broken code refs, got %d", result.Summary.CodeRefsBroken)
	}
}

func TestRunner_CodeRefWithLineNumber(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: src/main.go:42] for details\n")
	writeFile(t, filepath.Join(dir, "src/main.go"), "package main\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.CodeRefsFound != 1 {
		t.Fatalf("expected 1 code ref found, got %d", result.Summary.CodeRefsFound)
	}
	if result.Summary.CodeRefsBroken != 0 {
		t.Fatalf("expected 0 broken code refs, got %d", result.Summary.CodeRefsBroken)
	}
}

func TestRunner_ValidDocReference(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, "src/handler.go"), "package main\n// DOC: README.md\nfunc Handle() {}\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref found, got %d", result.Summary.DocRefsFound)
	}
	if result.Summary.DocRefsBroken != 0 {
		t.Fatalf("expected 0 broken doc refs, got %d", result.Summary.DocRefsBroken)
	}
}

func TestRunner_BrokenDocReference(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, "src/handler.go"), "package main\n// DOC: docs/missing.md\nfunc Handle() {}\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	// Default is warning mode
	if !result.Success {
		t.Fatalf("expected success (warning only), got failure: %+v", result)
	}
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref found, got %d", result.Summary.DocRefsFound)
	}
	if result.Summary.DocRefsBroken != 1 {
		t.Fatalf("expected 1 broken doc ref, got %d", result.Summary.DocRefsBroken)
	}
}

func TestRunner_DocRefWithSection(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n## Section\n")
	writeFile(t, filepath.Join(dir, "src/handler.go"), "package main\n// DOC: README.md#section\nfunc Handle() {}\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref found, got %d", result.Summary.DocRefsFound)
	}
	if result.Summary.DocRefsBroken != 0 {
		t.Fatalf("expected 0 broken doc refs, got %d", result.Summary.DocRefsBroken)
	}
}

func TestRunner_DocRefBlockComment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, "src/handler.go"), "package main\n/* DOC: README.md */\nfunc Handle() {}\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref found, got %d", result.Summary.DocRefsFound)
	}
}

func TestRunner_ValidMarkedPathAndDocReferences(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See `path:src/main.go` and `doc:docs/guide.md`.\n")
	writeFile(t, filepath.Join(dir, "src/main.go"), "package main\n")
	writeFile(t, filepath.Join(dir, "docs/guide.md"), "# Guide\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.MarkedRefsFound != 2 {
		t.Fatalf("expected 2 marked refs found, got %d", result.Summary.MarkedRefsFound)
	}
	if result.Summary.MarkedRefsBroken != 0 {
		t.Fatalf("expected 0 broken marked refs, got %d", result.Summary.MarkedRefsBroken)
	}
}

func TestRunner_BrokenMarkedReferenceWarnsByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See `path:missing.go`.\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected warning-only success, got failure: %+v", result)
	}
	if result.Summary.MarkedRefsFound != 1 || result.Summary.MarkedRefsBroken != 1 {
		t.Fatalf("expected 1 found / 1 broken marked ref, got %+v", result.Summary)
	}
}

func TestRunner_BrokenMarkedReferenceFailsWhenStrict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See `doc:docs/missing.md`.\n")

	settings := DefaultSettings()
	settings.References.Strict = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected strict failure for broken marked reference")
	}
	if result.Summary.MarkedRefsBroken != 1 {
		t.Fatalf("expected 1 broken marked ref, got %d", result.Summary.MarkedRefsBroken)
	}
}

func TestRunner_QualifiedMarkedRefsAreSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "Examples may use `path[example]:missing.go`, `doc[future]:docs/future.md`, and `topic:team/foo`.\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success for skipped marked refs, got failure: %+v", result)
	}
	if result.Summary.MarkedRefsFound != 3 || result.Summary.MarkedRefsSkipped != 3 || result.Summary.MarkedRefsBroken != 0 {
		t.Fatalf("unexpected marked ref summary: %+v", result.Summary)
	}
}

func TestRunner_QualifiedMarkedAbsolutePathDoesNotTripAbsolutePathScan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "Example only: `path[example]:/Users/alice/project/file.md`.\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success for qualified absolute-path example, got failure: %+v", result)
	}
	if result.Summary.AbsolutePathHits != 0 || result.Summary.AbsoluteFailures != 0 {
		t.Fatalf("qualified marked example should not count as absolute path hit, got %+v", result.Summary)
	}
}

func TestRunner_UnqualifiedMarkedAbsolutePathStillFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "Current path: `path:/Users/alice/project/file.md`.\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected unqualified marked absolute path to fail portability scan")
	}
	if result.Summary.AbsoluteFailures != 1 {
		t.Fatalf("expected 1 absolute path failure, got %d", result.Summary.AbsoluteFailures)
	}
}

func TestRunner_UnknownMarkedRefWarnsByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See `made-up:docs/value.md`.\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected warning-only success, got failure: %+v", result)
	}
	if result.Summary.MarkedRefsUnknown != 1 {
		t.Fatalf("expected 1 unknown marked ref, got %+v", result.Summary)
	}
}

func TestRunner_CommonColonCodeSpansAreNotUnknownMarkedRefs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "Literals: `go:embed`, `http://localhost:3000`, `status: \"unhealthy\"`, `vrooli: command not found`.\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.MarkedRefsUnknown != 0 || result.Summary.MarkedRefsFound != 0 {
		t.Fatalf("expected literal colon code spans to be ignored, got %+v", result.Summary)
	}
}

func TestRunner_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Put a file with DOC: comment in node_modules - should be skipped
	writeFile(t, filepath.Join(dir, "node_modules/pkg/index.js"), "// DOC: missing.md\n")
	// Put a file outside node_modules - should be scanned
	writeFile(t, filepath.Join(dir, "src/main.js"), "// DOC: README.md\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	// Only the file outside node_modules should be found
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref found (node_modules skipped), got %d", result.Summary.DocRefsFound)
	}
}

func TestRunner_CustomSkipDirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Put a file with DOC: comment in custom skip dir
	writeFile(t, filepath.Join(dir, "generated/client.ts"), "// DOC: missing.md\n")
	// Put a file outside the skip dir
	writeFile(t, filepath.Join(dir, "src/main.ts"), "// DOC: README.md\n")

	settings := DefaultSettings()
	settings.References.SkipDirs = []string{"generated"}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	// Only the file outside generated should be found
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref found (generated skipped), got %d", result.Summary.DocRefsFound)
	}
}

func TestRunner_ReferencesDisabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: missing.go] for details\n")
	writeFile(t, filepath.Join(dir, "src/main.go"), "// DOC: missing.md\n")

	settings := DefaultSettings()
	settings.References.Enabled = boolPtr(false)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success when references disabled, got failure: %+v", result)
	}
	// No refs should be tracked when disabled
	if result.Summary.CodeRefsFound != 0 || result.Summary.DocRefsFound != 0 {
		t.Fatalf("expected no refs when disabled, got code=%d doc=%d",
			result.Summary.CodeRefsFound, result.Summary.DocRefsFound)
	}
}

func TestRunner_CodeRefsOnlyDisabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: src/main.go] for details\n")
	writeFile(t, filepath.Join(dir, "src/main.go"), "package main\n// DOC: README.md\n")

	settings := DefaultSettings()
	settings.References.ValidateCodeRefs = boolPtr(false)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	// Code refs should not be found, but doc refs should be
	if result.Summary.CodeRefsFound != 0 {
		t.Fatalf("expected no code refs when disabled, got %d", result.Summary.CodeRefsFound)
	}
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref found, got %d", result.Summary.DocRefsFound)
	}
}

func TestRunner_ManifestCoverage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, "docs/guide.md"), "# Guide\n")
	writeFile(t, filepath.Join(dir, docsManifestRel), `["README.md", "docs/guide.md"]`)

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.DocsInManifest != 2 {
		t.Fatalf("expected 2 docs in manifest, got %d", result.Summary.DocsInManifest)
	}
}

func TestRunner_NoManifestSkipsCheck(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// No manifest file

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success when manifest missing, got failure: %+v", result)
	}
	// Should gracefully handle missing manifest
	if result.Summary.DocsInManifest != 0 {
		t.Fatalf("expected 0 docs in manifest when file missing, got %d", result.Summary.DocsInManifest)
	}
}

func TestRunner_ManifestRequireAll(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, "untracked.md"), "# Untracked\n")
	writeFile(t, filepath.Join(dir, docsManifestRel), `["README.md"]`)

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)
	settings.Manifest.RequireAllDocsRegistered = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	// Should still succeed but have warning observation
	if !result.Success {
		t.Fatalf("expected success with warnings, got failure: %+v", result)
	}
	if result.Summary.DocsNotInManifest != 1 {
		t.Fatalf("expected 1 doc not in manifest, got %d", result.Summary.DocsNotInManifest)
	}

	// Check for warning observation about orphaned doc
	hasWarning := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning && contains(obs.Message, "untracked.md") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Fatal("expected warning observation for orphaned doc")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Edge Case and Error Path Tests ---

func TestRunner_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := runner.Run(ctx)
	// Should return early with system failure, not panic
	if result.Success {
		t.Fatal("expected failure on cancelled context")
	}
	if result.FailureClass != FailureClassSystem {
		t.Fatalf("expected system failure class, got %v", result.FailureClass)
	}
}

func TestRunner_ExternalLinkDeduplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	// Same URL referenced 3 times in the same file
	content := fmt.Sprintf(
		"# Links\n\n[link1](%s/page1)\n[link2](%s/page1)\n[link3](%s/page1)\n",
		server.URL, server.URL, server.URL,
	)
	writeFile(t, filepath.Join(dir, "README.md"), content)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(server.Client()))

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}

	// Deduplication is verified by ExternalLinks count - same URL 3 times
	// should count as 1 external link due to deduplication
	// Note: localhost URLs are auto-ignored (return "ok" immediately) but still counted
	if result.Summary.ExternalLinks != 1 {
		t.Fatalf("expected 1 external link (deduplication), got %d", result.Summary.ExternalLinks)
	}
}

func TestRunner_MalformedManifest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Invalid JSON in manifest
	writeFile(t, filepath.Join(dir, docsManifestRel), `{invalid json}`)

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	// Should still succeed (manifest errors are logged, not fatal)
	if !result.Success {
		t.Fatalf("expected success with malformed manifest warning, got failure: %+v", result)
	}
}

func TestRunner_EmptyMarkdownFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "empty.md"), "")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with empty file, got failure: %+v", result)
	}
	if result.Summary.FilesChecked != 1 {
		t.Fatalf("expected 1 file checked, got %d", result.Summary.FilesChecked)
	}
}

func TestRunner_ExternalLink404(t *testing.T) {
	// Server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf("[broken](%s/missing)\n", server.URL))

	settings := DefaultSettings()
	// Note: localhost URLs are auto-ignored (return "ok"), so this test
	// verifies that the link is processed even though the actual HTTP
	// request is skipped due to localhost detection

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	}, WithHTTPClient(server.Client()))

	result := runner.Run(context.Background())
	// localhost URLs are auto-ignored, so this should succeed
	if !result.Success {
		t.Fatalf("expected success (localhost auto-ignored), got failure: %+v", result)
	}
	// External link should be counted even though localhost is ignored
	if result.Summary.ExternalLinks != 1 {
		t.Fatalf("expected 1 external link counted, got %d", result.Summary.ExternalLinks)
	}
}

func TestRunner_ResolvePath(t *testing.T) {
	dir := t.TempDir()
	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	tests := []struct {
		name     string
		target   string
		base     string
		expected string
	}{
		{
			name:     "relative to scenario root",
			target:   "docs/guide.md",
			base:     "",
			expected: filepath.Join(dir, "docs/guide.md"),
		},
		{
			name:     "relative to base file",
			target:   "sibling.md",
			base:     filepath.Join(dir, "docs/README.md"),
			expected: filepath.Join(dir, "docs/sibling.md"),
		},
		{
			name:     "absolute path unchanged",
			target:   "/absolute/path.md",
			base:     "",
			expected: "/absolute/path.md",
		},
		{
			name:     "parent directory reference",
			target:   "../other.md",
			base:     filepath.Join(dir, "docs/sub/file.md"),
			expected: filepath.Join(dir, "docs/other.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.resolvePath(tt.target, tt.base)
			if result != tt.expected {
				t.Errorf("resolvePath(%q, %q) = %q, want %q", tt.target, tt.base, result, tt.expected)
			}
		})
	}
}

func TestRunner_MappedRepoRelativeLink(t *testing.T) {
	repoRoot := t.TempDir()
	physicalScenario := filepath.Join(t.TempDir(), "scenarios", "demo")
	writeFile(t, filepath.Join(physicalScenario, "docs", "reference", "configuration.md"), "[ports](../../../../docs/reference/port-allocation.md)\n")
	writeFile(t, filepath.Join(repoRoot, "docs", "reference", "port-allocation.md"), "# Ports\n")

	mapping, err := workspace.NewMapping(physicalScenario, workspace.AppRootFromScenario(physicalScenario), repoRoot, "scenarios/demo", "demo")
	if err != nil {
		t.Fatalf("NewMapping() error = %v", err)
	}
	runner := New(Config{
		ScenarioDir:  physicalScenario,
		ScenarioName: "demo",
		Mapping:      mapping,
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected mapped repo-relative link to pass, got %+v", result.Observations)
	}
}

func TestRunner_UnmappedRepoRelativeLinkFails(t *testing.T) {
	physicalScenario := filepath.Join(t.TempDir(), "scenarios", "demo")
	writeFile(t, filepath.Join(physicalScenario, "docs", "reference", "configuration.md"), "[ports](../../../../docs/reference/port-allocation.md)\n")
	runner := New(Config{
		ScenarioDir:  physicalScenario,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatalf("expected unmapped repo-relative link to fail")
	}
}

func TestRunner_MultipleCodeRefsInOneLine(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: src/a.go] and [CODE: src/b.go] for details\n")
	writeFile(t, filepath.Join(dir, "src/a.go"), "package main\n")
	writeFile(t, filepath.Join(dir, "src/b.go"), "package main\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.CodeRefsFound != 2 {
		t.Fatalf("expected 2 code refs found, got %d", result.Summary.CodeRefsFound)
	}
}

func TestRunner_HashOnlyLink(t *testing.T) {
	dir := t.TempDir()
	// Link to anchor in same document
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n\nSee [section](#section)\n\n## Section\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success for hash-only link, got failure: %+v", result)
	}
	if result.Summary.BrokenLinks != 0 {
		t.Fatalf("expected no broken links for hash-only anchor, got %d", result.Summary.BrokenLinks)
	}
}

// --- LoadSettings Tests ---

func TestLoadSettings_FromFile(t *testing.T) {
	dir := t.TempDir()

	// Create testing.json with docs section
	testingJSON := `{
		"docs": {
			"mermaid": {
				"enabled": false
			},
			"links": {
				"max_concurrency": 10,
				"timeout_ms": 10000
			},
			"paths": {
				"exclude_dirs": ["ideas"],
				"exclude_globs": ["ideas/*-archived/**"]
			}
		}
	}`
	writeFile(t, filepath.Join(dir, ".vrooli", "testing.json"), testingJSON)

	settings, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	// Verify overridden values
	if settings.mermaidEnabled() {
		t.Error("expected mermaid disabled from config")
	}
	if settings.Links.MaxConcurrency != 10 {
		t.Errorf("expected max_concurrency=10, got %d", settings.Links.MaxConcurrency)
	}
	if settings.Links.TimeoutMs != 10000 {
		t.Errorf("expected timeout_ms=10000, got %d", settings.Links.TimeoutMs)
	}
	if len(settings.scanExcludeDirs()) != 1 || settings.scanExcludeDirs()[0] != "ideas" {
		t.Errorf("expected paths.exclude_dirs to load, got %#v", settings.scanExcludeDirs())
	}
	if len(settings.scanExcludeGlobs()) != 1 || settings.scanExcludeGlobs()[0] != "ideas/*-archived/**" {
		t.Errorf("expected paths.exclude_globs to load, got %#v", settings.scanExcludeGlobs())
	}
}

func TestLoadSettings_MissingFile(t *testing.T) {
	dir := t.TempDir()
	// No testing.json file

	settings, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	// Should return defaults
	if !settings.mermaidEnabled() {
		t.Error("expected default mermaid enabled")
	}
	if settings.Links.MaxConcurrency != 6 {
		t.Errorf("expected default max_concurrency=6, got %d", settings.Links.MaxConcurrency)
	}
}

func TestLoadSettings_PartialOverride(t *testing.T) {
	dir := t.TempDir()

	// Only override some fields - note: json tag is "absolute_paths"
	testingJSON := `{
		"docs": {
			"absolute_paths": {
				"enabled": false
			}
		}
	}`
	writeFile(t, filepath.Join(dir, ".vrooli", "testing.json"), testingJSON)

	settings, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	// Paths should be overridden
	if settings.pathsEnabled() {
		t.Error("expected paths disabled from config")
	}
	// Other settings should be defaults
	if !settings.mermaidEnabled() {
		t.Error("expected default mermaid enabled")
	}
	if !settings.linksEnabled() {
		t.Error("expected default links enabled")
	}
}

func TestLoadSettings_InvalidJSON(t *testing.T) {
	dir := t.TempDir()

	// Invalid JSON
	writeFile(t, filepath.Join(dir, ".vrooli", "testing.json"), `{invalid}`)

	_, err := LoadSettings(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- checkExternalLink Tests ---
// These tests use WithLinkIgnoreChecker to bypass localhost detection and test actual HTTP logic.

// noIgnore is a LinkIgnoreChecker that never ignores any URL.
func noIgnore(url string) bool { return false }

func TestCheckExternalLink_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf("[link](%s/page)\n", server.URL))

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(server.Client()), WithLinkIgnoreChecker(noIgnore))

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.ExternalLinks != 1 {
		t.Fatalf("expected 1 external link, got %d", result.Summary.ExternalLinks)
	}
	if result.Summary.ExternalFailures != 0 {
		t.Fatalf("expected 0 external failures, got %d", result.Summary.ExternalFailures)
	}
}

func TestCheckExternalLink_HeadBlockedFallbackToGet(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf("[link](%s/page)\n", server.URL))

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(server.Client()), WithLinkIgnoreChecker(noIgnore))

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success via GET fallback, got failure: %+v", result)
	}

	// Verify HEAD was tried first, then GET fallback
	if len(methods) < 2 {
		t.Fatalf("expected at least 2 requests (HEAD then GET), got %d: %v", len(methods), methods)
	}
	if methods[0] != http.MethodHead {
		t.Errorf("expected first request to be HEAD, got %s", methods[0])
	}
	if methods[1] != http.MethodGet {
		t.Errorf("expected second request to be GET, got %s", methods[1])
	}
}

func TestCheckExternalLink_404Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf("[broken](%s/missing)\n", server.URL))

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(server.Client()), WithLinkIgnoreChecker(noIgnore))

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure for 404 link")
	}
	if result.Summary.ExternalFailures != 1 {
		t.Fatalf("expected 1 external failure, got %d", result.Summary.ExternalFailures)
	}
}

func TestCheckExternalLink_500Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf("[broken](%s/error)\n", server.URL))

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(server.Client()), WithLinkIgnoreChecker(noIgnore))

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure for 500 error")
	}
	if result.Summary.ExternalFailures != 1 {
		t.Fatalf("expected 1 external failure, got %d", result.Summary.ExternalFailures)
	}
}

func TestCheckExternalLink_StrictModeNetworkError(t *testing.T) {
	// Server that closes connection immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf("[link](%s/page)\n", server.URL))

	settings := DefaultSettings()
	settings.Links.StrictExternal = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	}, WithHTTPClient(server.Client()), WithLinkIgnoreChecker(noIgnore))

	result := runner.Run(context.Background())
	// In strict mode, network errors are failures
	if result.Success {
		t.Fatal("expected failure in strict mode for network error")
	}
	if result.Summary.ExternalFailures != 1 {
		t.Fatalf("expected 1 external failure, got %d", result.Summary.ExternalFailures)
	}
}

func TestCheckExternalLink_NonStrictModeNetworkErrorWarning(t *testing.T) {
	// Server that closes connection immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf("[link](%s/page)\n", server.URL))

	settings := DefaultSettings()
	settings.Links.StrictExternal = boolPtr(false)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	}, WithHTTPClient(server.Client()), WithLinkIgnoreChecker(noIgnore))

	result := runner.Run(context.Background())
	// In non-strict mode, network errors are warnings (not failures)
	if !result.Success {
		t.Fatalf("expected success with warning, got failure: %+v", result)
	}
	if result.Summary.ExternalWarnings != 1 {
		t.Fatalf("expected 1 external warning, got %d", result.Summary.ExternalWarnings)
	}
}

func TestCheckExternalLink_IgnoredByChecker(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf("[link](%s/page)\n", server.URL))

	// Ignore checker that ignores everything
	allIgnore := func(url string) bool { return true }

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(server.Client()), WithLinkIgnoreChecker(allIgnore))

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}

	// No HTTP requests should be made when links are ignored
	if requestCount != 0 {
		t.Fatalf("expected 0 HTTP requests when ignored, got %d", requestCount)
	}
}

// --- matchPattern and shouldIgnoreLink Tests ---

func TestMatchPattern_Wildcard(t *testing.T) {
	tests := []struct {
		pattern  string
		value    string
		expected bool
	}{
		{"*.example.com", "test.example.com", true},
		{"*.example.com", "example.com", false},
		{"http://*/api", "http://test.com/api", true},
		{"prefix*", "prefixsuffix", true},
		{"prefix*", "other", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.pattern, tt.value), func(t *testing.T) {
			result := matchPattern(tt.pattern, tt.value)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.value, result, tt.expected)
			}
		})
	}
}

func TestMatchPattern_Prefix(t *testing.T) {
	tests := []struct {
		pattern  string
		value    string
		expected bool
	}{
		{"http://example.com", "http://example.com/page", true},
		{"http://example.com", "http://other.com/page", false},
		{"/docs/", "/docs/guide.md", true},
		{"/docs/", "/src/main.go", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.pattern, tt.value), func(t *testing.T) {
			result := matchPattern(tt.pattern, tt.value)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.value, result, tt.expected)
			}
		})
	}
}

func TestShouldIgnoreLink_Patterns(t *testing.T) {
	dir := t.TempDir()
	settings := DefaultSettings()
	settings.Links.Ignore = []string{
		"http://internal.company.com",
		"http://*.local/*", // glob pattern for .local URLs
	}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	tests := []struct {
		url      string
		expected bool
	}{
		{"http://internal.company.com/page", true},
		{"http://external.com/page", false},
		{"http://server.local/api", true}, // matches http://*.local/* glob
		{"http://localhost:8080", true},   // Built-in localhost check
		{"http://127.0.0.1:3000", true},   // Built-in 127.0.0.1 check
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := runner.shouldIgnoreLink(tt.url)
			if result != tt.expected {
				t.Errorf("shouldIgnoreLink(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestShouldIgnoreLink_EmptyPattern(t *testing.T) {
	dir := t.TempDir()
	settings := DefaultSettings()
	settings.Links.Ignore = []string{"", "http://valid.com"}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	// Empty pattern should be skipped
	if !runner.shouldIgnoreLink("http://valid.com/page") {
		t.Error("expected valid.com to be ignored")
	}
	if runner.shouldIgnoreLink("http://other.com/page") {
		t.Error("expected other.com not to be ignored (besides localhost check)")
	}
}

// --- balancedBrackets Tests ---

func TestBalancedBrackets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty", "", true},
		{"simple parentheses", "()", true},
		{"simple brackets", "[]", true},
		{"simple braces", "{}", true},
		{"nested", "[({})]", true},
		{"mixed", "([{}])", true},
		{"complex", "a(b[c{d}e]f)g", true},
		{"unclosed paren", "(", false},
		{"unclosed bracket", "[", false},
		{"unclosed brace", "{", false},
		{"mismatched", "[}", false},
		{"wrong order", "([)]", false},
		{"extra closing", "())", false},
		{"extra opening", "(()", false},
		{"only closing", "}", false},
		{"text with balanced", "function() { return [1, 2, 3]; }", true},
		{"text with unbalanced", "function() { return [1, 2, 3;", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := balancedBrackets(tt.input)
			if result != tt.expected {
				t.Errorf("balancedBrackets(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// --- WithLogger Test ---

func TestWithLogger(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")

	var buf bytes.Buffer
	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithLogger(&buf))

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}

	// Verify that output was written to our custom logger
	// The docs runner writes section headers and other info to the logger
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected output written to custom logger")
	}
}

// --- Additional Edge Case Tests ---

func TestRunner_LinksDisabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "[broken](missing.md)\n")

	settings := DefaultSettings()
	settings.Links.Enabled = boolPtr(false)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success when links disabled, got failure: %+v", result)
	}
	if result.Summary.LocalLinks != 0 {
		t.Errorf("expected 0 local links when disabled, got %d", result.Summary.LocalLinks)
	}
}

func TestRunner_MermaidDisabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "```mermaid\ninvalid diagram\n```\n")

	settings := DefaultSettings()
	settings.Mermaid.Enabled = boolPtr(false)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success when mermaid disabled, got failure: %+v", result)
	}
	if result.Summary.MermaidValidated != 0 {
		t.Errorf("expected 0 mermaid blocks validated when disabled, got %d", result.Summary.MermaidValidated)
	}
}

func TestRunner_PathsDisabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See /absolute/path here\n")

	settings := DefaultSettings()
	settings.Paths.Enabled = boolPtr(false)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success when paths disabled, got failure: %+v", result)
	}
	if result.Summary.AbsolutePathHits != 0 {
		t.Errorf("expected 0 absolute path hits when disabled, got %d", result.Summary.AbsolutePathHits)
	}
}

func TestRunner_NestedDirectoryLinks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs/guide.md"), "See [api](../api/README.md)\n")
	writeFile(t, filepath.Join(dir, "api/README.md"), "# API\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success for nested directory link, got failure: %+v", result)
	}
	if result.Summary.BrokenLinks != 0 {
		t.Fatalf("expected no broken links, got %d", result.Summary.BrokenLinks)
	}
}

func TestRunner_NilSettings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")

	// Config with nil settings - should use defaults
	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     nil,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with nil settings, got failure: %+v", result)
	}
}

// =============================================================================
// Coverage Improvement Tests
// =============================================================================

// --- Manifest Object Format Tests (parseJSONDocsField) ---

func TestRunner_ManifestObjectFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, "docs/guide.md"), "# Guide\n")
	// Object format with "docs" field
	writeFile(t, filepath.Join(dir, docsManifestRel), `{"docs": ["README.md", "docs/guide.md"]}`)

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.DocsInManifest != 2 {
		t.Fatalf("expected 2 docs in manifest (object format), got %d", result.Summary.DocsInManifest)
	}
}

func TestRunner_ManifestEmptyDocsField(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Object format with empty docs array
	writeFile(t, filepath.Join(dir, docsManifestRel), `{"docs": []}`)

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)
	settings.Manifest.RequireAllDocsRegistered = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	// README.md should be orphaned (not in manifest)
	if result.Summary.DocsNotInManifest != 1 {
		t.Fatalf("expected 1 doc not in manifest, got %d", result.Summary.DocsNotInManifest)
	}
}

// --- Nil Settings Receiver Tests ---

func TestSettings_NilReceiver(t *testing.T) {
	var s *Settings = nil

	// Test all accessor methods with nil receiver - should return defaults
	if !s.mermaidEnabled() {
		t.Error("nil Settings.mermaidEnabled() should return true")
	}
	if !s.mermaidStrict() {
		t.Error("nil Settings.mermaidStrict() should return true")
	}
	if !s.linksEnabled() {
		t.Error("nil Settings.linksEnabled() should return true")
	}
	if s.linksStrictExternal() {
		t.Error("nil Settings.linksStrictExternal() should return false")
	}
	if !s.pathsEnabled() {
		t.Error("nil Settings.pathsEnabled() should return true")
	}
	if !s.markdownEnabled() {
		t.Error("nil Settings.markdownEnabled() should return true")
	}
	if !s.referencesEnabled() {
		t.Error("nil Settings.referencesEnabled() should return true")
	}
	if !s.codeRefsEnabled() {
		t.Error("nil Settings.codeRefsEnabled() should return true")
	}
	if !s.docRefsEnabled() {
		t.Error("nil Settings.docRefsEnabled() should return true")
	}
	if !s.markedRefsEnabled() {
		t.Error("nil Settings.markedRefsEnabled() should return true")
	}
	if s.referencesStrict() {
		t.Error("nil Settings.referencesStrict() should return false")
	}
	if s.manifestEnabled() {
		t.Error("nil Settings.manifestEnabled() should return false")
	}
	if s.manifestRequireAll() {
		t.Error("nil Settings.manifestRequireAll() should return false")
	}

	// Check defaults for slice/string returns
	exts := s.codeExtensions()
	if len(exts) == 0 {
		t.Error("nil Settings.codeExtensions() should return default extensions")
	}
	skipDirs := s.referencesSkipDirs()
	if skipDirs != nil {
		t.Errorf("nil Settings.referencesSkipDirs() should return nil, got %v", skipDirs)
	}
	path := s.manifestPath()
	if path != docsManifestRel {
		t.Errorf("nil Settings.manifestPath() should return default, got %s", path)
	}
}

func TestSettings_NilSubStructs(t *testing.T) {
	// Settings with nil References and Manifest
	s := &Settings{
		References: nil,
		Manifest:   nil,
	}

	// These should return defaults when sub-struct is nil
	if !s.referencesEnabled() {
		t.Error("Settings with nil References should return default true")
	}
	if !s.codeRefsEnabled() {
		t.Error("Settings with nil References should return default true for codeRefsEnabled")
	}
	if !s.docRefsEnabled() {
		t.Error("Settings with nil References should return default true for docRefsEnabled")
	}
	if !s.markedRefsEnabled() {
		t.Error("Settings with nil References should return default true for markedRefsEnabled")
	}
	if s.referencesStrict() {
		t.Error("Settings with nil References should return default false for strict")
	}
	exts := s.codeExtensions()
	if len(exts) == 0 {
		t.Error("Settings with nil References should return default extensions")
	}
	if s.manifestEnabled() {
		t.Error("Settings with nil Manifest should return false")
	}
	if s.manifestRequireAll() {
		t.Error("Settings with nil Manifest should return false")
	}
	if s.manifestPath() != docsManifestRel {
		t.Error("Settings with nil Manifest should return default path")
	}
}

// --- allowedPrefix Edge Cases ---

func TestAllowedPrefix_EmptyList(t *testing.T) {
	if allowedPrefix("/home/user", nil) {
		t.Error("allowedPrefix with nil list should return false")
	}
	if allowedPrefix("/home/user", []string{}) {
		t.Error("allowedPrefix with empty list should return false")
	}
}

func TestAllowedPrefix_NoMatch(t *testing.T) {
	allow := []string{"/opt/", "/var/"}
	if allowedPrefix("/home/user", allow) {
		t.Error("allowedPrefix should return false when no prefix matches")
	}
}

func TestAllowedPrefix_Match(t *testing.T) {
	allow := []string{"/opt/", "/var/", "/home/"}
	if !allowedPrefix("/home/user/docs", allow) {
		t.Error("allowedPrefix should return true when prefix matches")
	}
}

// --- Windows Path Detection ---

func TestRunner_WindowsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	// Windows path regex requires pattern at start of line (^[A-Za-z]:\\)
	writeFile(t, filepath.Join(dir, "README.md"), "C:\\Users\\test\\docs contains files\n")

	settings := DefaultSettings()
	// Ensure paths check is enabled
	settings.Paths.Enabled = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	// Should fail due to Windows absolute path
	if result.Success {
		t.Fatal("expected failure for Windows absolute path")
	}
	if result.Summary.AbsolutePathHits != 1 {
		t.Fatalf("expected 1 absolute path hit, got %d", result.Summary.AbsolutePathHits)
	}
	if result.Summary.AbsoluteFailures != 1 {
		t.Fatalf("expected 1 absolute path failure, got %d", result.Summary.AbsoluteFailures)
	}
}

func TestRunner_WindowsPathAllowlisted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See C:\\Program Files\\app for details\n")

	settings := DefaultSettings()
	settings.Paths.Enabled = boolPtr(true)
	settings.Paths.Allow = []string{"C:\\Program Files"}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with allowlisted Windows path, got failure: %+v", result)
	}
}

// --- Custom Manifest Path ---

func TestRunner_CustomManifestPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Custom manifest location
	writeFile(t, filepath.Join(dir, ".meta/docs-index.json"), `["README.md"]`)

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)
	settings.Manifest.ManifestPath = ".meta/docs-index.json"

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with custom manifest path, got failure: %+v", result)
	}
	if result.Summary.DocsInManifest != 1 {
		t.Fatalf("expected 1 doc in manifest with custom path, got %d", result.Summary.DocsInManifest)
	}
}

// --- Reference to Directory Error ---

func TestRunner_CodeRefToDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [CODE: src/] for details\n")
	// Create src as a directory, not a file
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	settings := DefaultSettings()
	settings.References.Strict = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure when code ref points to directory")
	}
	if result.Summary.CodeRefsBroken != 1 {
		t.Fatalf("expected 1 broken code ref, got %d", result.Summary.CodeRefsBroken)
	}
}

func TestRunner_DocRefToDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Create docs directory and code file referencing the directory
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	writeFile(t, filepath.Join(dir, "src/main.go"), "package main\n// DOC: docs\nfunc main() {}\n")

	settings := DefaultSettings()
	settings.References.Strict = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure when doc ref points to directory")
	}
	if result.Summary.DocRefsBroken != 1 {
		t.Fatalf("expected 1 broken doc ref, got %d", result.Summary.DocRefsBroken)
	}
}

// --- isDigits Edge Cases ---

func TestIsDigits(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true}, // empty string - all chars are digits (vacuously true)
		{"0", true},
		{"123", true},
		{"0123456789", true},
		{"1a2", false},
		{"abc", false},
		{"-1", false},  // negative sign
		{"1.5", false}, // decimal point
		{" 1", false},  // leading space
		{"1 ", false},  // trailing space
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("isDigits(%q)", tt.input), func(t *testing.T) {
			result := isDigits(tt.input)
			if result != tt.expected {
				t.Errorf("isDigits(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// --- Python # DOC: Comment Style ---

func TestRunner_PythonDocComment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, "src/main.py"), "# DOC: README.md\ndef main():\n    pass\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref found (Python style), got %d", result.Summary.DocRefsFound)
	}
}

func TestRunner_PythonDocCommentBroken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, "src/main.py"), "# DOC: missing.md\ndef main():\n    pass\n")

	settings := DefaultSettings()
	settings.References.Strict = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure for broken Python DOC comment")
	}
	if result.Summary.DocRefsBroken != 1 {
		t.Fatalf("expected 1 broken doc ref, got %d", result.Summary.DocRefsBroken)
	}
}

// --- .mdx File Extension Handling ---

func TestRunner_MdxFileExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "component.mdx"), "# MDX Component\n\nexport const Component = () => <div>Hello</div>\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success for .mdx file, got failure: %+v", result)
	}
	if result.Summary.FilesChecked != 1 {
		t.Fatalf("expected 1 file checked (.mdx), got %d", result.Summary.FilesChecked)
	}
}

func TestRunner_MdxWithBrokenLink(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "component.mdx"), "# MDX\n\nSee [link](missing.md)\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure for broken link in .mdx file")
	}
	if result.Summary.BrokenLinks != 1 {
		t.Fatalf("expected 1 broken link in .mdx, got %d", result.Summary.BrokenLinks)
	}
}

// --- Multiple Mermaid Blocks Per File ---

func TestRunner_MultipleMermaidBlocks(t *testing.T) {
	dir := t.TempDir()
	content := `# Title

` + "```mermaid" + `
graph TD
  A --> B
` + "```" + `

Some text between.

` + "```mermaid" + `
sequenceDiagram
  Alice->>Bob: Hello
` + "```" + `

More content.

` + "```mermaid" + `
pie title Pets
  "Dogs" : 386
  "Cats" : 85
` + "```"
	writeFile(t, filepath.Join(dir, "README.md"), content)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.MermaidValidated != 3 {
		t.Fatalf("expected 3 mermaid blocks validated, got %d", result.Summary.MermaidValidated)
	}
}

func TestRunner_MultipleMermaidBlocksMixedValidity(t *testing.T) {
	dir := t.TempDir()
	content := `# Title

` + "```mermaid" + `
graph TD
  A --> B
` + "```" + `

` + "```mermaid" + `
not a valid diagram
` + "```" + `

` + "```mermaid" + `
sequenceDiagram
  Alice->>Bob: Hello
` + "```"
	writeFile(t, filepath.Join(dir, "README.md"), content)

	settings := DefaultSettings()
	settings.Mermaid.Strict = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure for invalid mermaid block")
	}
	if result.Summary.MermaidValidated != 3 {
		t.Fatalf("expected 3 mermaid blocks validated, got %d", result.Summary.MermaidValidated)
	}
	if result.Summary.MermaidFailures != 1 {
		t.Fatalf("expected 1 mermaid failure, got %d", result.Summary.MermaidFailures)
	}
}

// --- Custom Code Extensions ---

func TestRunner_CustomCodeExtensions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// .rb file with DOC comment - should only be scanned with custom extensions
	writeFile(t, filepath.Join(dir, "src/main.rb"), "# DOC: README.md\nclass Main\nend\n")
	// .go file should NOT be scanned when custom extensions don't include .go
	writeFile(t, filepath.Join(dir, "src/main.go"), "package main\n// DOC: README.md\n")

	settings := DefaultSettings()
	settings.References.CodeExtensions = []string{".rb"} // Only scan Ruby files

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	// Only the .rb file should be scanned for DOC: comments
	if result.Summary.DocRefsFound != 1 {
		t.Fatalf("expected 1 doc ref (from .rb only), got %d", result.Summary.DocRefsFound)
	}
}

// --- Link With Title Syntax ---
// Note: The current markdown link regex does not properly handle title syntax.
// Links like [text](url "title") include the title in the destination.
// This test documents the current limitation.

func TestRunner_LinkWithTitle_CurrentBehavior(t *testing.T) {
	dir := t.TempDir()
	// Current regex extracts `./guide.md "The Guide"` as dest (includes title)
	// This test documents the limitation - title syntax causes broken link detection
	writeFile(t, filepath.Join(dir, "README.md"), `See [guide](./guide.md "The Guide") for help.`)
	writeFile(t, filepath.Join(dir, "guide.md"), "# Guide\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	// Currently fails because title is included in dest path
	// If this test starts passing, the regex was improved to handle titles
	if result.Success {
		// Great! The limitation was fixed
		return
	}
	// Document current behavior: title syntax is not supported
	if result.Summary.BrokenLinks != 1 {
		t.Fatalf("expected 1 broken link (title included in dest), got %d", result.Summary.BrokenLinks)
	}
}

// --- Angle Bracket Links ---

func TestRunner_AngleBracketLink(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [link](<path with spaces.md>) for details\n")
	writeFile(t, filepath.Join(dir, "path with spaces.md"), "# Spaced\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success for angle bracket link, got failure: %+v", result)
	}
	if result.Summary.BrokenLinks != 0 {
		t.Fatalf("expected no broken links, got %d", result.Summary.BrokenLinks)
	}
}

// --- linksConcurrency and linksTimeout Edge Cases ---

func TestSettings_LinksConcurrencyZero(t *testing.T) {
	s := &Settings{
		Links: LinkSettings{
			MaxConcurrency: 0,
		},
	}
	if s.linksConcurrency() != 6 {
		t.Errorf("linksConcurrency() with 0 should return default 6, got %d", s.linksConcurrency())
	}
}

func TestSettings_LinksConcurrencyNegative(t *testing.T) {
	s := &Settings{
		Links: LinkSettings{
			MaxConcurrency: -1,
		},
	}
	if s.linksConcurrency() != 6 {
		t.Errorf("linksConcurrency() with -1 should return default 6, got %d", s.linksConcurrency())
	}
}

func TestSettings_LinksTimeoutZero(t *testing.T) {
	s := &Settings{
		Links: LinkSettings{
			TimeoutMs: 0,
		},
	}
	expected := 5 * time.Second
	if s.linksTimeout() != expected {
		t.Errorf("linksTimeout() with 0 should return default %v, got %v", expected, s.linksTimeout())
	}
}

func TestSettings_LinksTimeoutNegative(t *testing.T) {
	s := &Settings{
		Links: LinkSettings{
			TimeoutMs: -100,
		},
	}
	expected := 5 * time.Second
	if s.linksTimeout() != expected {
		t.Errorf("linksTimeout() with -100 should return default %v, got %v", expected, s.linksTimeout())
	}
}

// --- extractFilePath Edge Cases ---

func TestExtractFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"path/to/file.go", "path/to/file.go"},
		{"path/to/file.go#FunctionName", "path/to/file.go"},
		{"path/to/file.go:42", "path/to/file.go"},
		{"path/to/file.go#Method:42", "path/to/file.go"},               // anchor takes precedence
		{"path/to/file:with:colons.go", "path/to/file:with:colons.go"}, // colons in filename
		{"file.go:", "file.go:"},                                       // trailing colon, no digits
		{"  file.go  ", "file.go"},                                     // whitespace trimmed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("extractFilePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// --- Manifest with Missing Referenced Docs ---

func TestRunner_ManifestWithMissingDocs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Manifest references a doc that doesn't exist
	writeFile(t, filepath.Join(dir, docsManifestRel), `["README.md", "docs/missing.md"]`)

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	// Should still succeed but have warning
	if !result.Success {
		t.Fatalf("expected success with warning, got failure: %+v", result)
	}

	// Check for warning observation about missing doc
	hasWarning := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning && contains(obs.Message, "missing.md") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Fatal("expected warning observation for missing doc in manifest")
	}
}

// --- Tilde (~) Mermaid Fence ---

func TestRunner_TildeMermaidFence(t *testing.T) {
	dir := t.TempDir()
	content := "# Title\n\n~~~mermaid\ngraph TD\n  A --> B\n~~~\n"
	writeFile(t, filepath.Join(dir, "README.md"), content)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success for tilde fence, got failure: %+v", result)
	}
	if result.Summary.MermaidValidated != 1 {
		t.Fatalf("expected 1 mermaid block validated with ~~~ fence, got %d", result.Summary.MermaidValidated)
	}
}

// --- All Validations Disabled ---

func TestRunner_AllValidationsDisabled(t *testing.T) {
	dir := t.TempDir()
	// File with all kinds of issues that should be ignored
	content := "```\nunclosed fence\n[broken](missing.md)\n/absolute/path\n[CODE: missing.go]\n"
	writeFile(t, filepath.Join(dir, "README.md"), content)

	settings := DefaultSettings()
	settings.Markdown.Enabled = boolPtr(false)
	settings.Mermaid.Enabled = boolPtr(false)
	settings.Links.Enabled = boolPtr(false)
	settings.Paths.Enabled = boolPtr(false)
	settings.References.Enabled = boolPtr(false)
	settings.Manifest.Enabled = boolPtr(false)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	// Should succeed because all validation is disabled
	if !result.Success {
		t.Fatalf("expected success with all validations disabled, got failure: %+v", result)
	}
}

// --- validateLocalLink Edge Cases ---

func TestRunner_EmptyLinkDest(t *testing.T) {
	dir := t.TempDir()
	// Link with empty destination [text]()
	writeFile(t, filepath.Join(dir, "README.md"), "See [empty link]() for details\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	// Empty dest should be treated as valid (returns true early)
	if !result.Success {
		t.Fatalf("expected success for empty link dest, got failure: %+v", result)
	}
}

func TestRunner_RootRelativeLink(t *testing.T) {
	dir := t.TempDir()
	// Root-relative site paths like /docs/guide are treated as portable
	writeFile(t, filepath.Join(dir, "README.md"), "See [api docs](/api/guide) for details\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	// Root-relative paths (not OS-rooted like /home/...) should be valid
	if !result.Success {
		t.Fatalf("expected success for root-relative link, got failure: %+v", result)
	}
}

func TestRunner_AbsoluteUnixPathInLink(t *testing.T) {
	dir := t.TempDir()
	// Link with absolute Unix path
	writeFile(t, filepath.Join(dir, "README.md"), "See [config](/etc/config) for details\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	// OS-rooted paths like /etc/... should fail (not allowlisted)
	if result.Success {
		t.Fatal("expected failure for absolute Unix path in link")
	}
	if result.Summary.BrokenLinks != 1 {
		t.Fatalf("expected 1 broken link, got %d", result.Summary.BrokenLinks)
	}
}

func TestRunner_AbsoluteUnixPathInLinkAllowlisted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See [config](/etc/app/config) for details\n")

	settings := DefaultSettings()
	settings.Paths.Allow = []string{"/etc/"}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	// Should succeed because /etc/ is allowlisted
	if !result.Success {
		t.Fatalf("expected success for allowlisted absolute path, got failure: %+v", result)
	}
}

func TestRunner_LinkToDirectory(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	// Link pointing to directory, not file
	writeFile(t, filepath.Join(dir, "README.md"), "See [docs](./docs) for details\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	// Links to directories should fail
	if result.Success {
		t.Fatal("expected failure for link to directory")
	}
	if result.Summary.BrokenLinks != 1 {
		t.Fatalf("expected 1 broken link, got %d", result.Summary.BrokenLinks)
	}
}

// --- extractCodeRefs Edge Cases ---

func TestExtractCodeRefs_EmptyContent(t *testing.T) {
	refs := extractCodeRefs("test.md", "")
	if len(refs) != 0 {
		t.Errorf("expected 0 refs from empty content, got %d", len(refs))
	}
}

func TestExtractCodeRefs_NoMatch(t *testing.T) {
	refs := extractCodeRefs("test.md", "# Title\n\nNo code references here\n")
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestExtractCodeRefs_IgnoresInlineAndFencedExamples(t *testing.T) {
	content := strings.Join([]string{
		"Inline syntax example: `[CODE: path/to/file.ext]`",
		"```markdown",
		"- [CODE: src/example.ts#DoThing]",
		"```",
		"Real reference: [CODE: src/main.go#Run]",
	}, "\n")

	refs := extractCodeRefs("test.md", content)
	if len(refs) != 1 {
		t.Fatalf("expected exactly one real ref, got %d", len(refs))
	}
	if refs[0].Ref != "src/main.go#Run" {
		t.Fatalf("expected real ref to survive extraction, got %+v", refs[0])
	}
}

func TestExtractDocRefsFromFile_IgnoresSyntaxMentionsAndStringLiterals(t *testing.T) {
	dir := t.TempDir()
	codePath := filepath.Join(dir, "main.go")
	writeFile(t, codePath, strings.Join([]string{
		"package main",
		"// ValidateDocRefs checks // DOC: comments in code point to valid docs.",
		`var _ = "// DOC: docs/ignored.md"`,
		"// DOC: README.md",
		"/* DOC: docs/guide.md */",
	}, "\n"))

	refs, err := extractDocRefsFromFile(codePath)
	if err != nil {
		t.Fatalf("extractDocRefsFromFile returned error: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 standalone doc refs, got %d (%+v)", len(refs), refs)
	}
	if refs[0].DocPath != "README.md" || refs[1].DocPath != "docs/guide.md" {
		t.Fatalf("unexpected extracted refs: %+v", refs)
	}
}

// --- extractDocRefsFromFile Edge Cases ---

func TestRunner_UnreadableCodeFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Create a code file then make it unreadable
	codePath := filepath.Join(dir, "src/main.go")
	writeFile(t, codePath, "// DOC: README.md\n")

	// Make file unreadable (only works on Unix-like systems)
	if err := os.Chmod(codePath, 0o000); err != nil {
		t.Skip("cannot change file permissions on this system")
	}
	t.Cleanup(func() { _ = os.Chmod(codePath, 0o644) })

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	// Should still succeed - unreadable files are logged but not fatal
	if !result.Success {
		t.Fatalf("expected success (unreadable file logged), got failure: %+v", result)
	}
}

// --- scanCodeFilesForDocRefs Edge Cases ---

func TestRunner_ContextCancellationDuringCodeScan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	// Create multiple code files to make scan take longer
	for i := 0; i < 10; i++ {
		writeFile(t, filepath.Join(dir, fmt.Sprintf("src/file%d.go", i)), "package main\n// DOC: README.md\n")
	}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := runner.Run(ctx)
	// Should return early with system failure
	if result.Success {
		t.Fatal("expected failure on cancelled context")
	}
}

// --- checkManifestCoverage Edge Cases ---

func TestRunner_ManifestEmptyArray(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Title\n")
	writeFile(t, filepath.Join(dir, docsManifestRel), `[]`)

	settings := DefaultSettings()
	settings.Manifest.Enabled = boolPtr(true)

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with empty manifest, got failure: %+v", result)
	}
	if result.Summary.DocsInManifest != 0 {
		t.Fatalf("expected 0 docs in manifest, got %d", result.Summary.DocsInManifest)
	}
}

// --- New() Edge Cases ---

func TestNew_WithCustomHTTPClient(t *testing.T) {
	dir := t.TempDir()
	customClient := &http.Client{Timeout: 10 * time.Second}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(customClient))

	// Verify custom client was set (by checking it's not nil)
	if runner.client != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestNew_NilHTTPClientOption(t *testing.T) {
	dir := t.TempDir()

	// Pass nil as HTTP client - should fall back to default
	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(nil))

	// Should have a non-nil client (the fallback)
	if runner.client == nil {
		t.Error("expected fallback HTTP client when nil passed")
	}
}
