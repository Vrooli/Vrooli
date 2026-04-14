package topcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

func TestRunInfoListJSONOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, repocontractmeta.ProjectConfigDir), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(repocontractmeta.InfoManifestPath(root), []byte(`{"files":["docs/context.md"]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "context.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write context: %v", err)
	}

	var stdout bytes.Buffer
	if err := RunInfo(root, cliout.FormatJSON, InfoRequest{ListOnly: true}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunInfo: %v", err)
	}

	var payload struct {
		Success bool     `json:"success"`
		Root    string   `json:"root"`
		Files   []string `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !payload.Success || payload.Root != root {
		t.Fatalf("root = %q", payload.Root)
	}
	if len(payload.Files) != 1 || payload.Files[0] != filepath.Join(root, "docs", "context.md") {
		t.Fatalf("files = %v", payload.Files)
	}
}

func TestCollectInfoSourcesDetailedFallsBackOnInvalidManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, repocontractmeta.ProjectConfigDir), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(repocontractmeta.InfoManifestPath(root), []byte(`{"files":`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	files, warnings, err := CollectInfoSourcesDetailed(root)
	if err != nil {
		t.Fatalf("CollectInfoSourcesDetailed: %v", err)
	}
	if got, want := strings.Join(files, ","), strings.Join(DefaultInfoFiles, ","); got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Invalid info manifest") {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestParseInfoRequestRejectsUnknownOption(t *testing.T) {
	_, err := ParseInfoRequest([]string{"--bogus"})
	if err == nil || !strings.Contains(err.Error(), "unknown option for info") {
		t.Fatalf("ParseInfoRequest() error = %v", err)
	}
}

func TestRunInfoErrorsWhenNoSourcesConfigured(t *testing.T) {
	originalDefaults := DefaultInfoFiles
	DefaultInfoFiles = nil
	t.Cleanup(func() {
		DefaultInfoFiles = originalDefaults
	})

	err := RunInfo(t.TempDir(), cliout.FormatHuman, InfoRequest{}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no context sources defined") {
		t.Fatalf("RunInfo() error = %v", err)
	}
}

func TestRunInfoJSONSkipsMissingSourcesAndWarns(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, repocontractmeta.ProjectConfigDir), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(repocontractmeta.InfoManifestPath(root), []byte(`{"files":["docs/context.md","docs/missing.md"]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "context.md"), []byte("hello world\n"), 0o644); err != nil {
		t.Fatalf("write context: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := RunInfo(root, cliout.FormatJSON, InfoRequest{}, &stdout, &stderr); err != nil {
		t.Fatalf("RunInfo() error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"path":"`+filepath.Join(root, "docs", "context.md")+`"`) &&
		!strings.Contains(stdout.String(), `"path": "`+filepath.Join(root, "docs", "context.md")+`"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "missing.md") {
		t.Fatalf("stdout should skip missing files: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Skipping missing context file: docs/missing.md") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCollectInfoSourcesDetailedPrefersEnv(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_INFO_FILES", "docs/context.md:/tmp/extra.md")

	files, warnings, err := CollectInfoSourcesDetailed(root)
	if err != nil {
		t.Fatalf("CollectInfoSourcesDetailed() error = %v", err)
	}
	if got, want := strings.Join(files, ","), "docs/context.md,/tmp/extra.md"; got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
}

func TestResolveInfoPathPreservesAbsolutePaths(t *testing.T) {
	if got := ResolveInfoPath("/repo", "/tmp/context.md"); got != "/tmp/context.md" {
		t.Fatalf("ResolveInfoPath() = %q", got)
	}
}

func TestRunInfoPreservesAbsolutePathsFromEnv(t *testing.T) {
	root := t.TempDir()
	absoluteFile := filepath.Join(t.TempDir(), "extra.md")
	if err := os.MkdirAll(filepath.Dir(absoluteFile), 0o755); err != nil {
		t.Fatalf("mkdir absolute dir: %v", err)
	}
	if err := os.WriteFile(absoluteFile, []byte("extra context\n"), 0o644); err != nil {
		t.Fatalf("write absolute file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "context.md"), []byte("hello world\n"), 0o644); err != nil {
		t.Fatalf("write context: %v", err)
	}

	t.Setenv("VROOLI_INFO_FILES", "docs/context.md:"+absoluteFile+":docs/missing.md")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := RunInfo(root, cliout.FormatJSON, InfoRequest{}, &stdout, &stderr); err != nil {
		t.Fatalf("RunInfo() error = %v", err)
	}

	var payload InfoOutput
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal info payload: %v", err)
	}
	if len(payload.Files) != 2 {
		t.Fatalf("file count = %d, want 2", len(payload.Files))
	}
	if payload.Files[1].Path != absoluteFile {
		t.Fatalf("expected absolute info path to be preserved, got %q", payload.Files[1].Path)
	}
	if !strings.Contains(stderr.String(), "Skipping missing context file: docs/missing.md") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestInfoHelpTextMentionsListFlag(t *testing.T) {
	if help := InfoHelpText(); !strings.Contains(help, "--list") {
		t.Fatalf("InfoHelpText() = %q", help)
	}
}

func TestRenderMainHelpUsesPlainLabelsAndIncludesContract(t *testing.T) {
	var stdout bytes.Buffer
	RenderMainHelp(&stdout, CommandSpecs())

	output := stdout.String()
	if strings.Contains(output, "🚀") || strings.Contains(output, "📋") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Vrooli CLI - AI Platform Management Tool") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "contract") {
		t.Fatalf("output = %q", output)
	}
	for _, want := range []string{"--no-stale-check", "--verbose", "Documentation: docs/"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output = %q", want, output)
		}
	}
}
