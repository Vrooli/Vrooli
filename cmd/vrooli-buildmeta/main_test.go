package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=5 | LAST: 2026-04-11

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
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr = %q", code, stderr.String())
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
	if code != 2 {
		t.Fatalf("run exit code = %d, want 2", code)
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
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `vrooli-buildmeta: boom (root="/repo" targets=cmd/vrooli)`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsFlagParseExitCode(t *testing.T) {
	cmd := command{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--bogus"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("run exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr = %q", stderr.String())
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
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr = %q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); len(got) != 64 {
		t.Fatalf("stdout fingerprint length = %d, want 64 (%q)", len(got), got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	cmd := command{computeFingerprint: buildinfo.ComputeSourceFingerprintReport}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cmd.run([]string{"--root", root, "../outside"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
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
	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing fingerprint targets: internal") {
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
	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "no Go files matched") {
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
