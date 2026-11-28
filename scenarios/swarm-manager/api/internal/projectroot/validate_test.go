package projectroot

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiteralPrefix(t *testing.T) {
	tests := []struct {
		glob string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"scenarios/foo/cli/**", "scenarios/foo/cli"},
		{"scenarios/foo/cli/**/*.go", "scenarios/foo/cli"},
		{"scenarios/foo/api/main.go", "scenarios/foo/api/main.go"},
		{"scenarios/foo/api/", "scenarios/foo/api"},
		{"scenarios/foo/api/foo*.go", "scenarios/foo/api"},
		{"scenarios/foo/api/{a,b}/**", "scenarios/foo/api"},
		{"scenarios/foo/api/?test.go", "scenarios/foo/api"},
		{"scenarios/foo/api/[abc]/**", "scenarios/foo/api"},
		{"**/*.go", ""},
		{"*.md", ""},
		{"foo*.go", ""},
		{"scenarios/foo", "scenarios/foo"},
	}
	for _, tt := range tests {
		t.Run(tt.glob, func(t *testing.T) {
			got := literalPrefix(tt.glob)
			if got != tt.want {
				t.Errorf("literalPrefix(%q) = %q, want %q", tt.glob, got, tt.want)
			}
		})
	}
}

func TestValidateAcceptance_AllExist(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "dtv", "cli"))
	mustMkdirAll(t, filepath.Join(root, "scenarios", "dtv", "api"))

	report, err := ValidateAcceptance(root, []string{
		"scenarios/dtv/cli/**",
		"scenarios/dtv/api/**",
	}, nil)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if !report.Clean() {
		t.Errorf("expected clean report, got %+v", report.Problems)
	}
}

func TestValidateAcceptance_MissingPath(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "dtv", "cli"))
	// scenarios/dtv/api intentionally not created

	report, err := ValidateAcceptance(root, []string{
		"scenarios/dtv/cli/**",
		"scenarios/dtv/api/**",
	}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch, got %v", err)
	}
	if len(report.Problems) != 1 {
		t.Fatalf("expected 1 problem, got %d: %+v", len(report.Problems), report.Problems)
	}
	if report.Problems[0].Glob != "scenarios/dtv/api/**" {
		t.Errorf("unexpected glob in problem: %q", report.Problems[0].Glob)
	}
}

func TestValidateAcceptance_CreatesAllowsMissingPath(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "dtv", "cli"))
	// scenarios/dtv/api intentionally not created — but creates declares it.

	report, err := ValidateAcceptance(root, []string{
		"scenarios/dtv/cli/**",
		"scenarios/dtv/api/**",
	}, []string{"scenarios/dtv/api/**"})
	if err != nil {
		t.Fatalf("expected nil with creates coverage, got %v", err)
	}
	if !report.Clean() {
		t.Errorf("expected clean report, got %+v", report.Problems)
	}
}

func TestValidateAcceptance_CreatesAncestorAllowsMissingPath(t *testing.T) {
	root := t.TempDir()
	// Acceptance prefix points at "docs"; creates points at a descendant.
	report, err := ValidateAcceptance(root, []string{
		"docs/**",
	}, []string{"docs/internal/SANDBOX-CONTRACT.md"})
	if err != nil {
		t.Fatalf("expected nil when creates is more specific descendant, got %v", err)
	}
	if !report.Clean() {
		t.Errorf("expected clean, got %+v", report.Problems)
	}
}

func TestValidateAcceptance_CreatesPathTraversalRejected(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "dtv", "cli"))

	_, err := ValidateAcceptance(root, []string{
		"scenarios/dtv/cli/**",
	}, []string{"../etc/passwd"})
	if err == nil {
		t.Fatal("expected error for traversal in creates, got nil")
	}
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch, got %v", err)
	}
}

func TestValidateAcceptance_PathTraversalRejected(t *testing.T) {
	root := t.TempDir()

	_, err := ValidateAcceptance(root, []string{
		"../etc/passwd/**",
	}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch, got %v", err)
	}
}

func TestValidateAcceptance_WildcardOnlyGlobsSkipped(t *testing.T) {
	root := t.TempDir()
	report, err := ValidateAcceptance(root, []string{"**/*.go", "*.md"}, nil)
	if err != nil {
		t.Errorf("expected nil for wildcard-only globs, got %v", err)
	}
	if !report.Clean() {
		t.Errorf("expected clean, got %+v", report.Problems)
	}
}

func TestValidateAcceptance_LiteralFilePath(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "foo"))
	mustWriteFile(t, filepath.Join(root, "scenarios", "foo", "README.md"))

	_, err := ValidateAcceptance(root, []string{"scenarios/foo/README.md"}, nil)
	if err != nil {
		t.Errorf("expected nil for existing literal file, got %v", err)
	}

	_, err = ValidateAcceptance(root, []string{"scenarios/foo/MISSING.md"}, nil)
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch for missing literal file, got %v", err)
	}
}

func TestValidateAcceptance_RejectsRelativeProjectRoot(t *testing.T) {
	_, err := ValidateAcceptance("relative/path", []string{"foo/**"}, nil)
	if err == nil {
		t.Fatal("expected error for relative projectRoot, got nil")
	}
}

func TestValidateAcceptance_AggregatesProblems(t *testing.T) {
	root := t.TempDir()

	report, err := ValidateAcceptance(root, []string{
		"scenarios/missing-a/**",
		"scenarios/missing-b/**",
	}, nil)
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Fatalf("expected ErrAcceptanceMismatch, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "missing-a") || !strings.Contains(msg, "missing-b") {
		t.Errorf("expected error to mention both missing paths, got %q", msg)
	}
	if len(report.Problems) != 2 {
		t.Errorf("expected 2 problems, got %d: %+v", len(report.Problems), report.Problems)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}
