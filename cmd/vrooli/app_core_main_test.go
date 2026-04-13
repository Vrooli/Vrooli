package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

func TestRunTriggersRebuildBeforeDispatch(t *testing.T) {
	app := newTestApp("/repo")
	app.isStale = func() bool { return true }

	var rebuiltArgs []string
	app.rebuildAndReexec = func(args []string) error {
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
	app.checkStaleness = func() (buildinfo.StaleCheck, error) {
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

func TestRunInfoCommandRejectsUnknownOption(t *testing.T) {
	err := runInfoCommand("/repo", globalOptions{}, []string{"--bogus"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown option for info") {
		t.Fatalf("runInfoCommand error = %v", err)
	}
}

func TestRunInfoCommandErrorsWhenNoSourcesConfigured(t *testing.T) {
	originalDefaults := topcli.DefaultInfoFiles
	topcli.DefaultInfoFiles = nil
	t.Cleanup(func() {
		topcli.DefaultInfoFiles = originalDefaults
	})

	err := runInfoCommand(t.TempDir(), globalOptions{}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no context sources defined") {
		t.Fatalf("runInfoCommand error = %v", err)
	}
}

func TestRunVersionJSONOutput(t *testing.T) {
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return "/repo", nil }
	app.isStale = func() bool { return false }
	app.checkStaleness = nil

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

func TestRunInfoCommandHelpAndJSONMissingFiles(t *testing.T) {
	root := t.TempDir()
	absoluteFile := filepath.Join(t.TempDir(), "extra.md")
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	writeTestFile(t, filepath.Dir(absoluteFile), filepath.Base(absoluteFile), "extra context\n")

	var help bytes.Buffer
	if err := runInfoCommand(root, globalOptions{}, []string{"--help"}, &help, &bytes.Buffer{}); err != nil {
		t.Fatalf("runInfoCommand help: %v", err)
	}
	if !strings.Contains(help.String(), "Usage: vrooli info [--list]") {
		t.Fatalf("missing help output: %s", help.String())
	}

	t.Setenv("VROOLI_INFO_FILES", "docs/context.md:"+absoluteFile+":docs/missing.md")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runInfoCommand(root, globalOptions{json: true}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("runInfoCommand json: %v", err)
	}

	var payload topcli.InfoOutput
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

func TestCollectInfoSourcesPrefersEnvAndFallsBackOnInvalidManifest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_INFO_FILES", "docs/context.md:/tmp/extra.md")

	files, warnings, err := collectInfoSourcesDetailed(root)
	if err != nil {
		t.Fatalf("collectInfoSources env: %v", err)
	}
	if got, want := strings.Join(files, ","), "docs/context.md,/tmp/extra.md"; got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}

	t.Setenv("VROOLI_INFO_FILES", "")
	writeTestFile(t, root, filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.InfoManifestFilename), `{"files":`)

	files, warnings, err = collectInfoSourcesDetailed(root)
	if err != nil {
		t.Fatalf("collectInfoSources fallback: %v", err)
	}
	if got, want := strings.Join(files, ","), strings.Join(topcli.DefaultInfoFiles, ","); got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Invalid info manifest") {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestRunUnknownCommandSuggestsNearestMatch(t *testing.T) {
	app := newTestApp("/repo")

	var stderr bytes.Buffer
	code := app.Run([]string{"statuz"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Unknown command: statuz") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "status") {
		t.Fatalf("expected suggestion list in stderr, got %q", stderr.String())
	}
}

func TestShowVersionAndHelpOutput(t *testing.T) {
	var version bytes.Buffer
	if err := showVersion(&version, "/repo", globalOptions{}); err != nil {
		t.Fatalf("showVersion: %v", err)
	}
	if !strings.Contains(version.String(), "Vrooli CLI v"+cliVersion) {
		t.Fatalf("version output = %q", version.String())
	}

	var help bytes.Buffer
	showMainHelp(&help)
	if !strings.Contains(help.String(), "scenario") || !strings.Contains(help.String(), "Manage scenarios from their source locations") {
		t.Fatalf("help output = %q", help.String())
	}
}

func TestRunVersionDoesNotRequireRootResolution(t *testing.T) {
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return "", errors.New("boom") }
	app.checkStaleness = nil

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
	app.resolveSourceRoot = func() (string, error) {
		t.Fatal("root resolution should be skipped")
		return "", nil
	}
	app.checkStaleness = nil

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
	if !strings.Contains(stderr.String(), "Unknown command: statuz") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCommandEnvPreservesExistingSourceRootAndNoColor(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("LANG", "C")
	t.Setenv("VROOLI_SOURCE_ROOT", "/custom/source")

	env := configuredApp().commandEnv("/repo", globalOptions{noColor: true})
	got := strings.Join(env, "\n")
	if !strings.Contains(got, "VROOLI_ROOT=/repo") {
		t.Fatalf("env missing VROOLI_ROOT: %v", env)
	}
	if !strings.Contains(got, "VROOLI_SOURCE_ROOT=/custom/source") {
		t.Fatalf("env missing preserved source root: %v", env)
	}
	if !strings.Contains(got, "NO_COLOR=1") {
		t.Fatalf("env missing NO_COLOR: %v", env)
	}
}

func TestResolveInfoPathAndPassthroughFlags(t *testing.T) {
	absolute := resolveInfoPath("/repo", "/tmp/context.md")
	if absolute != "/tmp/context.md" {
		t.Fatalf("resolveInfoPath absolute = %q", absolute)
	}

	flags := passthroughFlags(globalOptions{json: true, verbose: true, noColor: true}, []string{"--json", "scenario"})
	if got, want := strings.Join(flags, ","), "--verbose,--no-color"; got != want {
		t.Fatalf("flags = %q, want %q", got, want)
	}
	if containsArg([]string{"alpha", "beta"}, "--json") {
		t.Fatalf("containsArg should not match absent flag")
	}
}

func TestExitCodeError(t *testing.T) {
	if got := (exitCodeError{code: 7, message: "boom"}).Error(); got != "boom" {
		t.Fatalf("exitCodeError message = %q", got)
	}
	if got := (exitCodeError{code: 7}).Error(); got != "exit code 7" {
		t.Fatalf("exitCodeError default = %q", got)
	}
}
