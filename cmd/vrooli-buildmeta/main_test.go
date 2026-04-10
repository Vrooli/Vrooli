package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRunPrintsFingerprint(t *testing.T) {
	original := computeFingerprintForPathsFn
	t.Cleanup(func() {
		computeFingerprintForPathsFn = original
	})

	var gotRoot string
	var gotPaths []string
	computeFingerprintForPathsFn = func(root string, relPaths ...string) (string, error) {
		gotRoot = root
		gotPaths = append([]string(nil), relPaths...)
		return "abc123", nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--root", "/repo", "cmd/vrooli", "internal"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr = %q", code, stderr.String())
	}
	if gotRoot != "/repo" {
		t.Fatalf("root = %q, want /repo", gotRoot)
	}
	if strings.Join(gotPaths, ",") != "cmd/vrooli,internal" {
		t.Fatalf("paths = %v", gotPaths)
	}
	if stdout.String() != "abc123\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReportsFingerprintError(t *testing.T) {
	original := computeFingerprintForPathsFn
	t.Cleanup(func() {
		computeFingerprintForPathsFn = original
	})

	computeFingerprintForPathsFn = func(root string, relPaths ...string) (string, error) {
		return "", errors.New("boom")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--root", "/repo"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "vrooli-buildmeta: boom") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReturnsFlagParseExitCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--bogus"}, &stdout, &stderr)
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
