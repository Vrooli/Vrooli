package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

func TestRunTriggersRebuildBeforeDispatch(t *testing.T) {
	app := newTestApp("/repo")
	app.CheckStalenessFn = func() (buildinfo.StaleCheck, error) {
		return buildinfo.StaleCheck{Stale: true}, nil
	}

	var rebuiltArgs []string
	app.RebuildAndReexecFn = func(args []string) error {
		rebuiltArgs = append([]string(nil), args...)
		return nil
	}

	code := app.Run([]string{"scenario", "list"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if strings.Join(rebuiltArgs, "|") != "scenario|list" {
		t.Fatalf("rebuilt args = %v", rebuiltArgs)
	}
}

func TestRunReportsStaleCheckFailure(t *testing.T) {
	app := newTestApp("/repo")
	app.CheckStalenessFn = func() (buildinfo.StaleCheck, error) {
		return buildinfo.StaleCheck{}, errors.New("fingerprint targets drifted")
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"scenario", "list"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Runtime error: stale binary check failed: fingerprint targets drifted") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Use --no-stale-check for local experiments") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunInfoCommandUsesManifestAndListMode(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.InfoManifestFilename), `{"files":["docs/context.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"info", "--list"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if strings.TrimSpace(stdout.String()) != filepath.Join(root, "docs", "context.md") {
		t.Fatalf("info list output = %q", stdout.String())
	}
}

func TestRunInfoHelpExitsZero(t *testing.T) {
	app := newTestApp(t.TempDir())

	var stdout bytes.Buffer
	code := app.Run([]string{"info", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), topcli.InfoHelpText()) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunVersionJSONOutput(t *testing.T) {
	app := configuredApp()
	app.ResolveSourceRootFn = func() (string, error) { return "/repo", nil }
	app.CheckStalenessFn = func() (buildinfo.StaleCheck, error) {
		return buildinfo.StaleCheck{Stale: false}, nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"--json", "--version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}

	var payload map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal version json: %v", err)
	}
	if payload["root"] != "/repo" {
		t.Fatalf("root = %q", payload["root"])
	}
	if payload["cli_version"] != cliVersion {
		t.Fatalf("cli_version = %q", payload["cli_version"])
	}
}

func TestRunInfoListJSONOutput(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.InfoManifestFilename), `{"files":["docs/context.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"--json", "info", "--list"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}

	var payload struct {
		Success bool     `json:"success"`
		Root    string   `json:"root"`
		Files   []string `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal info json: %v", err)
	}
	if !payload.Success || payload.Root != root {
		t.Fatalf("root = %q", payload.Root)
	}
	if len(payload.Files) != 1 || payload.Files[0] != filepath.Join(root, "docs", "context.md") {
		t.Fatalf("files = %v", payload.Files)
	}
}

func TestRunInfoCommandSkipsMissingSourcesInJSONMode(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.InfoManifestFilename), `{"files":["docs/context.md","docs/missing.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	app := newTestApp(root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"--json", "info"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"path": "`+filepath.Join(root, "docs", "context.md")+`"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "missing.md") {
		t.Fatalf("stdout should skip missing files: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Skipping missing context file: docs/missing.md") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunUnknownCommandSuggestsNearestMatch(t *testing.T) {
	app := newTestApp("/repo")

	var stderr bytes.Buffer
	code := app.Run([]string{"statuz"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), clipolicy.UnknownCommandLabel+": statuz") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "status") {
		t.Fatalf("expected suggestion list in stderr, got %q", stderr.String())
	}
}

func TestRunVersionDoesNotRequireRootResolution(t *testing.T) {
	app := configuredApp()
	app.ResolveSourceRootFn = func() (string, error) { return "", errors.New("boom") }
	app.CheckStalenessFn = nil

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Vrooli CLI v"+cliVersion) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunMainHelpAndUnknownCommandDoNotRequireRootResolution(t *testing.T) {
	app := configuredApp()
	app.ResolveSourceRootFn = func() (string, error) {
		t.Fatal("root resolution should be skipped")
		return "", nil
	}
	app.CheckStalenessFn = nil

	var help bytes.Buffer
	if code := app.Run([]string{"--help"}, &help, &bytes.Buffer{}); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if !strings.Contains(help.String(), "Vrooli CLI - AI Platform Management Tool") {
		t.Fatalf("help output = %q", help.String())
	}

	var stderr bytes.Buffer
	if code := app.Run([]string{"statuz"}, &bytes.Buffer{}, &stderr); code != 1 {
		t.Fatalf("unknown-command exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), clipolicy.UnknownCommandLabel+": statuz") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
