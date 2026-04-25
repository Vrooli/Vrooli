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

func TestValidateAcceptanceUnderRoot_AllExist(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "dtv", "cli"))
	mustMkdirAll(t, filepath.Join(root, "scenarios", "dtv", "api"))

	err := ValidateAcceptanceUnderRoot(root, []string{
		"scenarios/dtv/cli/**",
		"scenarios/dtv/api/**",
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateAcceptanceUnderRoot_MissingPath(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "dtv", "cli"))
	// scenarios/dtv/api intentionally not created

	err := ValidateAcceptanceUnderRoot(root, []string{
		"scenarios/dtv/cli/**",
		"scenarios/dtv/api/**",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch, got %v", err)
	}
}

func TestValidateAcceptanceUnderRoot_PathTraversalRejected(t *testing.T) {
	root := t.TempDir()

	err := ValidateAcceptanceUnderRoot(root, []string{
		"../etc/passwd/**",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch, got %v", err)
	}
}

func TestValidateAcceptanceUnderRoot_WildcardOnlyGlobsSkipped(t *testing.T) {
	root := t.TempDir()
	// No paths created. Pure wildcard globs have no literal prefix to check.
	err := ValidateAcceptanceUnderRoot(root, []string{"**/*.go", "*.md"})
	if err != nil {
		t.Errorf("expected nil for wildcard-only globs, got %v", err)
	}
}

func TestValidateAcceptanceUnderRoot_LiteralFilePath(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "foo"))
	mustWriteFile(t, filepath.Join(root, "scenarios", "foo", "README.md"))

	err := ValidateAcceptanceUnderRoot(root, []string{"scenarios/foo/README.md"})
	if err != nil {
		t.Errorf("expected nil for existing literal file, got %v", err)
	}

	err = ValidateAcceptanceUnderRoot(root, []string{"scenarios/foo/MISSING.md"})
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch for missing literal file, got %v", err)
	}
}

func TestValidateAcceptanceUnderRoot_RejectsRelativeProjectRoot(t *testing.T) {
	err := ValidateAcceptanceUnderRoot("relative/path", []string{"foo/**"})
	if err == nil {
		t.Fatal("expected error for relative projectRoot, got nil")
	}
}

func TestValidateAcceptanceUnderRoot_AggregatesProblems(t *testing.T) {
	root := t.TempDir()

	err := ValidateAcceptanceUnderRoot(root, []string{
		"scenarios/missing-a/**",
		"scenarios/missing-b/**",
	})
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Fatalf("expected ErrAcceptanceMismatch, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "missing-a") || !strings.Contains(msg, "missing-b") {
		t.Errorf("expected error to mention both missing paths, got %q", msg)
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

