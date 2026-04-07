package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockFS simulates a filesystem with a set of files that exist.
type mockFS struct {
	files []string // full paths of files that "exist"
}

func (m *mockFS) Stat(name string) (os.FileInfo, error) {
	// Directory exists if any file is under it
	for _, f := range m.files {
		if strings.HasPrefix(f, name) {
			return nil, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *mockFS) Glob(pattern string) ([]string, error) {
	var matches []string
	for _, f := range m.files {
		matched, err := filepath.Match(pattern, f)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, f)
		}
	}
	return matches, nil
}

func TestFindBuiltPackageWith_UnknownPlatform(t *testing.T) {
	fs := &mockFS{files: []string{"dist/something"}}
	_, err := FindBuiltPackageWith(fs, "dist", "bsd")
	if err == nil {
		t.Fatal("expected error for unknown platform")
	}
	if !strings.Contains(err.Error(), "unknown platform") {
		t.Errorf("error = %q, want 'unknown platform'", err.Error())
	}
}

func TestFindBuiltPackageWith_MissingDirectory(t *testing.T) {
	fs := &mockFS{files: nil}
	_, err := FindBuiltPackageWith(fs, "nonexistent", "win")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err.Error())
	}
}

func TestFindBuiltPackageWith_NoMatchingFiles(t *testing.T) {
	fs := &mockFS{files: []string{"dist/readme.txt"}}
	_, err := FindBuiltPackageWith(fs, "dist", "win")
	if err == nil {
		t.Fatal("expected error when no matching files")
	}
	if !strings.Contains(err.Error(), "no built package found") {
		t.Errorf("error = %q, want 'no built package found'", err.Error())
	}
}

func TestFindBuiltPackageWith_Windows(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			"prefers MSI over EXE",
			[]string{"dist/app.exe", "dist/app.msi"},
			"dist/app.msi",
		},
		{
			"prefers Setup.exe over plain exe",
			[]string{"dist/app.exe", "dist/appSetup.exe"},
			"dist/appSetup.exe",
		},
		{
			"single MSI",
			[]string{"dist/installer.msi"},
			"dist/installer.msi",
		},
		{
			"single EXE",
			[]string{"dist/app.exe"},
			"dist/app.exe",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := &mockFS{files: tc.files}
			got, err := FindBuiltPackageWith(fs, "dist", "win")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestFindBuiltPackageWith_Mac(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			"prefers PKG over DMG",
			[]string{"dist/app.dmg", "dist/app.pkg"},
			"dist/app.pkg",
		},
		{
			"single DMG",
			[]string{"dist/app.dmg"},
			"dist/app.dmg",
		},
		{
			"filters blockmap files",
			[]string{"dist/app.dmg.blockmap", "dist/app.dmg"},
			"dist/app.dmg",
		},
		{
			"prefers non-arm64 when multiple",
			[]string{"dist/app-arm64.dmg", "dist/app.dmg"},
			"dist/app.dmg",
		},
		{
			"single zip",
			[]string{"dist/app.zip"},
			"dist/app.zip",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := &mockFS{files: tc.files}
			got, err := FindBuiltPackageWith(fs, "dist", "mac")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestFindBuiltPackageWith_Linux(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			"AppImage found",
			[]string{"dist/app.AppImage"},
			"dist/app.AppImage",
		},
		{
			"deb found",
			[]string{"dist/app.deb"},
			"dist/app.deb",
		},
		{
			"prefers AppImage over deb (searched first)",
			[]string{"dist/app.deb", "dist/app.AppImage"},
			"dist/app.AppImage",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := &mockFS{files: tc.files}
			got, err := FindBuiltPackageWith(fs, "dist", "linux")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}
