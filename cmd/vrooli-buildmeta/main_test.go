package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=6 | LAST: 2026-04-12

func TestRunPrintsFingerprint(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			if root != "/repo" {
				t.Fatalf("root = %q, want /repo", root)
			}
			if !options.RequireExistingTargets || !options.RequireGoFiles {
				t.Fatalf("options = %+v, want strict target validation", options)
			}
			if got, want := strings.Join(relPaths, ","), "cmd/vrooli,internal"; got != want {
				t.Fatalf("paths = %q, want %q", got, want)
			}
			return buildinfo.FingerprintReport{Fingerprint: "abc123"}, nil
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", "/repo", "cmd/vrooli", "internal"}, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("run exit code = %d, want %d, stderr = %q", code, exitCodeSuccess, stderr.String())
	}
	if stdout.String() != "abc123\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRequiresTargets(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			t.Fatalf("computeFingerprint should not be called without targets")
			return buildinfo.FingerprintReport{}, nil
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", "/repo"}, &stdout, &stderr)
	if code != exitCodeUsage {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "at least one relative path is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage: vrooli-buildmeta") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReportsFingerprintError(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{}, errors.New("boom")
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", "/repo", "cmd/vrooli"}, &stdout, &stderr)
	if code != exitCodeInternal {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeInternal)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `vrooli-buildmeta: boom (root="/repo" targets=cmd/vrooli)`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunPrintsCategorizedMissingTargetError(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{}, buildinfo.MissingTargetsError{Targets: []string{"internal"}}
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", "/repo", "cmd/vrooli", "internal"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeValidation)
	}
	if !strings.Contains(stderr.String(), `requested targets do not exist: internal`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsFlagParseExitCode(t *testing.T) {
	cmd := command{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--bogus"}, &stdout, &stderr)
	if code != exitCodeUsage {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsHelpExitCode(t *testing.T) {
	cmd := command{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"-h"}, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeSuccess)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: vrooli-buildmeta") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsInternalExitCodeWhenUsageWriteFails(t *testing.T) {
	cmd := command{}

	var stdout bytes.Buffer
	code := cmd.run([]string{"-h"}, &stdout, failingWriter{})
	if code != exitCodeInternal {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeInternal)
	}
}

func TestRunFingerprintsRealFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/logx/logx.go", "package logx\n")

	cmd := command{computeFingerprint: buildinfo.ComputeSourceFingerprintReport}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", root, "cmd/vrooli", "internal"}, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("run exit code = %d, want %d, stderr = %q", code, exitCodeSuccess, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); len(got) != 64 {
		t.Fatalf("stdout fingerprint length = %d, want 64 (%q)", len(got), got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunPrintsJSONReport(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{
				Root:         "/repo",
				Targets:      []string{"cmd/vrooli", "internal"},
				MatchedFiles: 2,
				Fingerprint:  "abc123",
			}, nil
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--json", "--root", "/repo", "cmd/vrooli", "internal"}, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("run exit code = %d, want %d, stderr = %q", code, exitCodeSuccess, stderr.String())
	}

	var report buildinfo.FingerprintReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json output: %v (%q)", err, stdout.String())
	}
	if report.Fingerprint != "abc123" || report.MatchedFiles != 2 {
		t.Fatalf("report = %+v", report)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunVerbosePrintsSummaryToStderr(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{
				Root:         "/repo",
				Targets:      []string{"cmd/vrooli", "internal"},
				MatchedFiles: 2,
				Fingerprint:  "abc123",
			}, nil
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--verbose", "--root", "/repo", "cmd/vrooli", "internal"}, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("run exit code = %d, want %d, stderr = %q", code, exitCodeSuccess, stderr.String())
	}
	if stdout.String() != "abc123\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `fingerprinted 2 Go files under "/repo"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsInternalExitCodeWhenVerboseWriteFails(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{
				Root:         "/repo",
				Targets:      []string{"cmd/vrooli"},
				MatchedFiles: 1,
				Fingerprint:  "abc123",
			}, nil
		},
	}

	var stdout bytes.Buffer
	code := cmd.run([]string{"--verbose", "--root", "/repo", "cmd/vrooli"}, &stdout, failingWriter{})
	if code != exitCodeInternal {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeInternal)
	}
	if stdout.String() != "abc123\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunRejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	cmd := command{computeFingerprint: buildinfo.ComputeSourceFingerprintReport}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", root, "../outside"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeValidation)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "escapes repository root") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsMissingTarget(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")

	cmd := command{computeFingerprint: buildinfo.ComputeSourceFingerprintReport}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", root, "cmd/vrooli", "internal"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeValidation)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "requested targets do not exist: internal") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsTargetsWithoutGoFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/README.md", "hello\n")

	cmd := command{computeFingerprint: buildinfo.ComputeSourceFingerprintReport}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", root, "docs"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeValidation)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "requested targets do not contain any Go files: docs") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunNilFingerprintFuncFallsBackToBuildinfo(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")

	cmd := command{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", root, "cmd/vrooli"}, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("run exit code = %d, want %d, stderr = %q", code, exitCodeSuccess, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); len(got) != 64 {
		t.Fatalf("stdout fingerprint length = %d, want 64 (%q)", len(got), got)
	}
}

func TestRunJSONErrorOutput(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{}, buildinfo.MissingTargetsError{Targets: []string{"internal"}}
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--json", "--root", "/repo", "cmd/vrooli", "internal"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeValidation)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}

	var response errorResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("json output: %v (%q)", err, stdout.String())
	}
	if response.Kind != errorKindMissingTargets {
		t.Fatalf("kind = %q, want %q", response.Kind, errorKindMissingTargets)
	}
	if response.Root != "/repo" {
		t.Fatalf("root = %q, want /repo", response.Root)
	}
	if got, want := strings.Join(response.Targets, ","), "cmd/vrooli,internal"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
}

func TestRunReturnsInternalExitCodeWhenFingerprintWriteFails(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{Fingerprint: "abc123"}, nil
		},
	}

	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", "/repo", "cmd/vrooli"}, failingWriter{}, &stderr)
	if code != exitCodeInternal {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeInternal)
	}
	if !strings.Contains(stderr.String(), "write fingerprint") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsInternalExitCodeWhenJSONEncodeFails(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{Fingerprint: "abc123"}, nil
		},
	}

	var stderr bytes.Buffer
	code := cmd.run([]string{"--json", "--root", "/repo", "cmd/vrooli"}, failingWriter{}, &stderr)
	if code != exitCodeInternal {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeInternal)
	}
	if !strings.Contains(stderr.String(), "encode JSON output") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsInternalExitCodeWhenJSONErrorEncodeFails(t *testing.T) {
	cmd := command{
		computeFingerprint: func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error) {
			return buildinfo.FingerprintReport{}, buildinfo.MissingTargetsError{Targets: []string{"internal"}}
		},
	}

	var stderr bytes.Buffer
	code := cmd.run([]string{"--json", "--root", "/repo", "cmd/vrooli", "internal"}, failingWriter{}, &stderr)
	if code != exitCodeInternal {
		t.Fatalf("run exit code = %d, want %d", code, exitCodeInternal)
	}
	if !strings.Contains(stderr.String(), "encode JSON error output") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func writeTestFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}
